package importjob

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
)

const (
	guiSourceSnapshotInputLimit   int64 = 512 << 20
	guiSourceSnapshotArchiveLimit int64 = 512 << 20
	guiSourceSnapshotMemberLimit        = 100_000
)

type GUISourceStore interface {
	EnvPrefix() string
	Put(key, contentType string, size int64, body io.Reader) error
}

func buildGUISourceArchive(root string) (string, int, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return "", 0, err
	}
	if !info.IsDir() {
		return "", 0, errors.New("GUI source root is not a directory")
	}
	archive, err := os.CreateTemp(filepath.Dir(root), "gui_source_snapshot_*.zip")
	if err != nil {
		return "", 0, err
	}
	archivePath := archive.Name()
	failed := func(err error) (string, int, error) {
		_ = archive.Close()
		_ = os.Remove(archivePath)
		return "", 0, err
	}
	writer := zip.NewWriter(archive)
	var files int
	var total int64
	err = filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, filename)
		if err != nil || !isGUISourcePath(rel) {
			return err
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if fileInfo.Size() < 0 || total > guiSourceSnapshotInputLimit-fileInfo.Size() {
			return fmt.Errorf("GUI source input exceeds %d byte limit", guiSourceSnapshotInputLimit)
		}

		if files >= guiSourceSnapshotMemberLimit-1 {
			return fmt.Errorf("GUI source input exceeds %d member limit", guiSourceSnapshotMemberLimit)
		}
		total += fileInfo.Size()
		input, err := os.Open(filename)
		if err != nil {
			return err
		}
		member, err := writer.Create(filepath.ToSlash(rel))
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(member, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files++
		return nil
	})
	if err != nil {
		return failed(err)
	}
	if files > 0 {
		marker, err := writer.Create(importassets.GUISourceGameDataMarker)
		if err == nil {
			_, err = io.WriteString(marker, "1\n")
		}
		if err != nil {
			return failed(err)
		}
	}
	if err := writer.Close(); err != nil {
		return failed(err)
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(archivePath)
		return "", 0, err
	}
	info, err = os.Stat(archivePath)
	if err != nil {
		_ = os.Remove(archivePath)
		return "", 0, err
	}
	if info.Size() > guiSourceSnapshotArchiveLimit {
		_ = os.Remove(archivePath)
		return "", 0, fmt.Errorf("GUI source archive exceeds %d byte limit", guiSourceSnapshotArchiveLimit)
	}
	if files == 0 {
		_ = os.Remove(archivePath)
		return "", 0, nil
	}
	return archivePath, files, nil
}

func isGUISourcePath(raw string) bool {
	path := strings.ToLower(filepath.ToSlash(filepath.Clean(raw)))
	if path == "." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") {
		return false
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch {
	case strings.HasPrefix(path, "interface/"):
		return ext == "gui" || ext == "gfx"
	case strings.HasPrefix(path, "gfx/"):
		return ext == "dds" || ext == "png" || ext == "tga" || ext == "fnt" || ext == "ttf" || ext == "otf" ||
			ext == "asset" || ext == "gfx"
	case strings.HasPrefix(path, "localisation/"):
		return ext == "yml"
	case strings.HasPrefix(path, "common/"), strings.HasPrefix(path, "history/"), strings.HasPrefix(path, "events/"):
		return ext == "txt"
	case path == "map/continent.txt", path == "map/definition.csv":
		return true
	default:
		return false
	}
}

func fingerprintTechnologySurfaceSource(root string) (techsurfaceprofiles.SourceFingerprint, error) {
	result := techsurfaceprofiles.SourceFingerprint{Files: []techsurfaceprofiles.SourceFile{}}
	for _, rel := range []string{
		techsurfaceprofiles.CountryTechTreeGUIPath,
		techsurfaceprofiles.TechnologyTagsPath,
	} {
		filename := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return techsurfaceprofiles.SourceFingerprint{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			continue
		}
		input, err := os.Open(filename)
		if err != nil {
			return techsurfaceprofiles.SourceFingerprint{}, err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, input)
		closeErr := input.Close()
		if copyErr != nil {
			return techsurfaceprofiles.SourceFingerprint{}, copyErr
		}
		if closeErr != nil {
			return techsurfaceprofiles.SourceFingerprint{}, closeErr
		}
		result.Files = append(result.Files, techsurfaceprofiles.SourceFile{
			Path: rel, SHA256: fmt.Sprintf("%x", hash.Sum(nil)),
		})
	}
	return result, nil
}
