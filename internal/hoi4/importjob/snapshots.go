package importjob

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

func (s *session) reportUnrouted() error {
	byDir, err := UnroutedByDir(s.root)
	if err != nil {
		return err
	}
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if byDir[dirs[i]] != byDir[dirs[j]] {
			return byDir[dirs[i]] > byDir[dirs[j]]
		}
		return dirs[i] < dirs[j]
	})
	for _, d := range dirs {
		s.warn(fmt.Sprintf(
			"unrouted_dir: %s — %d file(s) in this directory were not imported. "+
				"Hagane Works has no workbench for them and they are not in the "+
				"verbatim-preserve list, so they are not in the project and will "+
				"not be in the export. To ship them, add them as raw files.",
			d, byDir[d]))
	}
	return nil
}

func (s *session) persistGUISourceSnapshot(p *project.Project, store *project.BlobStore) (int, error) {
	archivePath, files, err := buildGUISourceArchive(s.root)
	if err != nil || files == 0 {
		return files, err
	}
	defer os.Remove(archivePath)
	source, err := fingerprintTechnologySurfaceSource(s.root)
	if err != nil {
		return files, fmt.Errorf("fingerprint technology surface source: %w", err)
	}
	key, size, err := putArchive(store, archivePath, func(sha string) (string, error) {
		return importassets.CertifiedGUISourceSnapshotKey(store.EnvPrefix(), store.ProjectUUID(), sha)
	})
	if err != nil {
		return files, fmt.Errorf("write GUI source snapshot: %w", err)
	}
	_ = size
	provenance := map[string]any{
		"technology_surface": map[string]any{"schema_version": 1, "source": jsonValueOf(source), "dlc_confirmed": false},
	}
	err = p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		assets := newKeyedTable(tx, "project_obs_assets", "asset_type", "gfx_name")
		assets.upsert(project.Row{
			"asset_type": "gui_source", "gfx_name": "snapshot.zip", "obs_key": key, "provenance": provenance,
		}, []string{"obs_key", "provenance"})
		return nil
	})
	return files, err
}

func (s *session) persistIdeologySourceSnapshot(p *project.Project, store *project.BlobStore) (int, error) {
	replacePaths := []string{}
	if list, ok := p.Record()["imported_descriptor_replace_paths"].([]any); ok {
		for _, item := range list {
			if path, ok := item.(string); ok {
				replacePaths = append(replacePaths, path)
			}
		}
	}
	archivePath, provenance, err := BuildIdeologySourceArchive(s.root, "imported", "", replacePaths)
	if err != nil || len(provenance.Files) == 0 {
		return len(provenance.Files), err
	}
	defer os.Remove(archivePath)
	key, _, err := putArchive(store, archivePath, func(sha string) (string, error) {
		return importassets.IdeologySourceSnapshotKey(store.EnvPrefix(), store.ProjectUUID(), provenance.SourceKey, sha)
	})
	if err != nil {
		return len(provenance.Files), fmt.Errorf("write ideology source snapshot: %w", err)
	}
	if provenance.DescriptorReplacePaths == nil {
		provenance.DescriptorReplacePaths = []string{}
	}
	err = p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		assets := newKeyedTable(tx, "project_obs_assets", "asset_type", "gfx_name")
		assets.upsert(project.Row{
			"asset_type": "ideology_source", "gfx_name": provenance.SourceKey, "obs_key": key,
			"provenance": jsonValueOf(provenance),
		}, []string{"obs_key", "provenance"})

		tx.SetRecord(func(record project.Row) {
			manifest, _ := record["ideology_source_manifest"].(map[string]any)
			entries, _ := manifest["entries"].([]any)
			if len(entries) == 0 {
				return
			}
			next := map[string]any{}
			for k, v := range manifest {
				if k != "ready" && k != "ready_at" && k != "compiled_at" {
					next[k] = v
				}
			}
			next["state"] = "stale"
			next["source_key"] = provenance.SourceKey
			next["snapshot_updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
			record["ideology_source_manifest"] = next
		})
		return nil
	})
	if err != nil {
		return len(provenance.Files), err
	}

	return len(provenance.Files), nil
}

func putArchive(store GUISourceStore, archivePath string, keyFor func(sha string) (string, error)) (string, int64, error) {
	archive, err := os.Open(archivePath)
	if err != nil {
		return "", 0, err
	}
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return "", 0, err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	key, err := keyFor(fmt.Sprintf("%x", hash.Sum(nil)))
	if err != nil {
		return "", 0, err
	}
	if err := store.Put(key, "application/zip", info.Size(), archive); err != nil {
		return "", 0, err
	}
	return key, info.Size(), nil
}

func jsonValueOf(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func countryFlagVariantName(tag, ideology string) string {
	return strings.ToUpper(tag) + "_" + strings.ToLower(ideology)
}
