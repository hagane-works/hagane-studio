package importjob

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/hoi4/ideologysource"
)

const ideologySourceSnapshotInputLimit int64 = 512 << 20

type IdeologySourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type IdeologySourceProvenance struct {
	SchemaVersion          int                  `json:"schema_version"`
	SourceKey              string               `json:"source_key"`
	DependencyKey          string               `json:"dependency_key,omitempty"`
	DescriptorReplacePaths []string             `json:"descriptor_replace_paths"`
	Files                  []IdeologySourceFile `json:"files"`
}

func BuildIdeologySourceArchive(root, sourceKey, dependencyKey string, replacePaths []string) (string, IdeologySourceProvenance, error) {
	manifest := IdeologySourceProvenance{
		SchemaVersion: 1, SourceKey: strings.TrimSpace(sourceKey),
		DependencyKey:          strings.TrimSpace(dependencyKey),
		DescriptorReplacePaths: normalizeReplacePaths(replacePaths),
		Files:                  []IdeologySourceFile{},
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", manifest, err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return "", manifest, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return "", manifest, errors.New("ideology source root must be a real directory")
	}
	base := filepath.Join(root, "common", "ideologies")
	baseInfo, err := os.Lstat(base)
	if errors.Is(err, os.ErrNotExist) {
		return "", manifest, nil
	}
	if err != nil {
		return "", manifest, err
	}
	if baseInfo.Mode()&os.ModeSymlink != 0 || !baseInfo.IsDir() {
		return "", manifest, errors.New("common/ideologies must be a real directory")
	}

	var names []string
	var total int64
	err = filepath.WalkDir(base, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("ideology source rejects symlink %q", filename)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("ideology source rejects non-regular file %q", filename)
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			return nil
		}
		rel, err := filepath.Rel(root, filename)
		if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("ideology source path escaped staged root")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() < 0 || total > ideologySourceSnapshotInputLimit-info.Size() {
			return fmt.Errorf("ideology source input exceeds %d byte limit", ideologySourceSnapshotInputLimit)
		}
		total += info.Size()
		names = append(names, filename)
		return nil
	})
	if err != nil {
		return "", manifest, err
	}
	if len(names) == 0 {
		return "", manifest, nil
	}
	sort.Slice(names, func(i, j int) bool {
		left, _ := filepath.Rel(root, names[i])
		right, _ := filepath.Rel(root, names[j])
		return filepath.ToSlash(left) < filepath.ToSlash(right)
	})

	archive, err := os.CreateTemp(filepath.Dir(root), "ideology_source_snapshot_*.zip")
	if err != nil {
		return "", manifest, err
	}
	archivePath := archive.Name()
	fail := func(err error) (string, IdeologySourceProvenance, error) {
		_ = archive.Close()
		_ = os.Remove(archivePath)
		return "", manifest, err
	}
	writer := zip.NewWriter(archive)
	var bytesRead int64
	for _, filename := range names {
		input, err := os.Open(filename)
		if err != nil {
			return fail(err)
		}
		remaining := ideologySourceSnapshotInputLimit - bytesRead
		data, readErr := io.ReadAll(io.LimitReader(input, remaining+1))
		closeErr := input.Close()
		if readErr != nil {
			return fail(readErr)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
		if int64(len(data)) > remaining {
			return fail(fmt.Errorf("ideology source input exceeds %d byte limit", ideologySourceSnapshotInputLimit))
		}
		bytesRead += int64(len(data))
		rel, err := filepath.Rel(root, filename)
		if err != nil {
			return fail(err)
		}
		rel = filepath.ToSlash(rel)
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		manifest.Files = append(manifest.Files, IdeologySourceFile{Path: rel, SHA256: digest})
		header := &zip.FileHeader{Name: rel, Method: zip.Deflate}
		header.SetMode(0o600)
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		member, err := writer.CreateHeader(header)
		if err != nil {
			return fail(err)
		}
		if _, err := member.Write(data); err != nil {
			return fail(err)
		}
	}
	if err := writer.Close(); err != nil {
		return fail(err)
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(archivePath)
		return "", manifest, err
	}
	return archivePath, manifest, nil
}

func discoverImportedIdeologySource(root string, replacePaths []string) (ideologysource.Discovery, error) {
	base := filepath.Join(root, "common", "ideologies")
	files := []ideologysource.SourceFile{}
	err := filepath.WalkDir(base, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("ideology source rejects symlink %q", filename)
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			return nil
		}
		rel, err := filepath.Rel(root, filename)
		if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("ideology source path escaped staged root")
		}
		body, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		files = append(files, ideologysource.SourceFile{Path: filepath.ToSlash(rel), Bytes: body})
		return nil
	})
	if err != nil {
		return ideologysource.Discovery{}, err
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
	return ideologysource.Discover([]ideologysource.Layer{
		{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: []ideologysource.SourceFile{}},
		{SourceKind: ideologysource.SourceDependency, DependencyKey: "imported", LoadOrder: 1, ReplacePaths: replacePaths, Files: files},
	})
}

func normalizeReplacePaths(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")), "/")
		if value == "" || value == "." || value == ".." || strings.HasPrefix(value, "../") || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
