package importjob

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"haganeworks/internal/project"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const mapFilesNote = "mod map kept on import"

const maxMapTextBytes = 32 << 20

const maxRawObjectKeyLen = 255

type InputObject struct {
	ObjectKey string
	SHA256    string
}

func IsModMapFile(rel string) bool {
	lower := strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	if !strings.HasPrefix(lower, "map/") {
		return false
	}
	if lower == "map/terrain/colormap_rgb_cityemissivemask_a.dds" {
		return true
	}
	switch path.Ext(lower) {
	case ".bmp", ".txt", ".csv", ".map":
		return true
	}
	return false
}

type BlobHeader interface {
	Head(key string) (bool, error)
}

func mapFileRow(file, rel string, objects map[string]InputObject, store GUISourceStore) ([]any, string, error) {
	info, err := os.Stat(file)
	if err != nil {
		return nil, "", err
	}
	if ext := path.Ext(strings.ToLower(rel)); ext != ".bmp" && ext != ".dds" && info.Size() <= maxMapTextBytes {
		body, err := os.ReadFile(file)
		if err != nil {
			return nil, "", err
		}
		if utf8.Valid(body) && !bytes.ContainsRune(body, 0) {
			return []any{rel, string(body), false, nil, nil}, "", nil
		}
	}
	sha, err := fileSHA256(file)
	if err != nil {
		return nil, "", err
	}
	key := ""
	if object, ok := objects[rel]; ok && strings.EqualFold(object.SHA256, sha) {
		key = strings.TrimSpace(object.ObjectKey)
	} else if store != nil {
		key = strings.Trim(store.EnvPrefix(), "/") + "/rawblobs/" + sha[:2] + "/" + sha
		if err := putMapBlob(store, key, file, info.Size()); err != nil {
			return nil, fmt.Sprintf("its bytes could not be stored (%v)", err), nil
		}
	}
	switch {
	case key == "":
		return nil, "there is no object storage for its bytes", nil
	case len(key) > maxRawObjectKeyLen:
		return nil, "its storage key is too long", nil
	}
	return []any{rel, "", true, sha, key}, "", nil
}

func putMapBlob(store GUISourceStore, key, file string, size int64) error {
	if header, ok := store.(BlobHeader); ok {
		if exists, err := header.Head(key); err == nil && exists {
			return nil
		}
	}
	body, err := os.Open(file)
	if err != nil {
		return err
	}
	defer body.Close()
	return store.Put(key, "application/octet-stream", size, body)
}

func fileSHA256(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

type mapFileStaged struct {
	destPath, content string
	isBinary          bool
	sha256, objectKey any
}

type mapFileStage struct{ rows []mapFileStaged }

type mapFileExister interface{ Exists(key string) bool }

type mapFileHeadStore struct {
	GUISourceStore
	exister mapFileExister
}

func (s mapFileHeadStore) Head(key string) (bool, error) { return s.exister.Exists(key), nil }

func (s *session) stageMapFiles(files GUISourceStore) (int, error) {
	store := files
	if exister, ok := files.(mapFileExister); ok {
		if _, hasHead := files.(BlobHeader); !hasHead {
			store = mapFileHeadStore{GUISourceStore: files, exister: exister}
		}
	}
	kept := 0
	err := filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(s.root, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !IsModMapFile(rel) {
			return nil
		}
		row, problem, err := mapFileRow(p, rel, nil, store)
		if err != nil {
			return err
		}
		if problem != "" {
			s.warn(fmt.Sprintf("Map file %s was not kept: %s", rel, problem))
			return nil
		}
		kept++
		content, _ := row[1].(string)
		isBinary, _ := row[2].(bool)
		s.mapFiles.rows = append(s.mapFiles.rows, mapFileStaged{
			destPath: rel, content: content, isBinary: isBinary, sha256: row[3], objectKey: row[4],
		})
		return nil
	})
	if err != nil {
		return 0, err
	}
	return kept, nil
}

func (s *session) mergeMapFiles(tx *project.Tx, options MiscMergeOptions) error {

	if len(s.mapFiles.rows) == 0 {
		return nil
	}
	skip := strings.EqualFold(options.ConflictPolicy, "skip")
	if !skip {

		shipped := make(map[string]bool, len(s.mapFiles.rows))
		for _, f := range s.mapFiles.rows {
			shipped[f.destPath] = true
		}
		tx.Delete("project_raw_files", func(row project.Row) bool {
			destPath, _ := row["dest_path"].(string)
			note, _ := row["note"].(string)
			return miscImportedRawFile(row) && note == mapFilesNote && !shipped[destPath]
		})
	}

	when := miscImportedRawFile
	if skip {
		when = nil
	}
	raw := newKeyedTable(tx, "project_raw_files", "dest_path")
	for _, f := range s.mapFiles.rows {
		miscUpsertRawFile(raw, project.Row{
			"dest_path": f.destPath, "content": f.content, "generate_replace_path": false,
			"raw_kind": "verbatim_override", "note": mapFilesNote, "is_binary": f.isBinary,
			"content_binary": nil, "content_sha256": f.sha256, "content_obs_key": f.objectKey,
			"source": "imported",
		}, []string{"content", "is_binary", "content_binary", "content_sha256", "content_obs_key", "raw_kind", "note"}, when)
	}
	return nil
}
