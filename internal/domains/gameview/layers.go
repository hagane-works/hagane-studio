package gameview

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const snapshotLimit = int64(512 << 20)

func projectKey(p *project.Project) string {
	return filepath.Clean(p.Dir())
}

type snapshotRef struct {
	Key      string
	Revision string
}

func guiSnapshot(p *project.Project) (snapshotRef, bool, error) {
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return snapshotRef{}, false, err
	}
	for _, row := range rows {
		if str(row["asset_type"]) == "gui_source" && str(row["gfx_name"]) == "snapshot.zip" {
			key := strings.TrimSpace(str(row["obs_key"]))
			if key == "" {
				return snapshotRef{}, false, nil
			}
			return snapshotRef{Key: key, Revision: str(row["created_at"])}, true, nil
		}
	}
	return snapshotRef{}, false, nil
}

func openSnapshot(p *project.Project, ref snapshotRef) (*zip.ReadCloser, error) {
	path, err := p.Blobs().Path(ref.Key)
	if err != nil {
		return nil, err
	}
	return zip.OpenReader(path)
}

func snapshotFile(p *project.Project, ref snapshotRef, name string, limit int64) ([]byte, bool, error) {
	archive, err := openSnapshot(p, ref)
	if err != nil {
		return nil, false, err
	}
	defer archive.Close()
	for _, member := range archive.File {
		if member.FileInfo().IsDir() || strings.ReplaceAll(member.Name, "\\", "/") != name {
			continue
		}
		body, err := readMember(member, limit)
		return body, err == nil, err
	}
	return nil, false, nil
}

func readMember(member *zip.File, limit int64) ([]byte, error) {
	if member.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%s is over %d bytes", member.Name, limit)
	}
	rc, err := member.Open()
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(rc, limit+1))
	closeErr := rc.Close()
	if readErr != nil {
		return nil, readErr
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s is over %d bytes", member.Name, limit)
	}
	return body, closeErr
}

func snapshotLacksGameData(p *project.Project, ref snapshotRef) (bool, error) {

	memoKey := projectKey(p) + "|" + ref.Key + "|" + ref.Revision
	if lacks, ok := lacksGameData.Load(memoKey); ok {
		return lacks.(bool), nil
	}
	archive, err := openSnapshot(p, ref)
	if err != nil {
		return false, err
	}
	defer archive.Close()
	lacks := true
	for _, member := range archive.File {
		if strings.ReplaceAll(member.Name, "\\", "/") == importassets.GUISourceGameDataMarker {
			lacks = false
			break
		}
	}
	lacksGameData.Store(memoKey, lacks)
	return lacks, nil
}

var lacksGameData sync.Map

func readSnapshotFiles(p *project.Project, ref snapshotRef) (map[string][]byte, error) {
	raw, err := p.Blobs().Get(ref.Key, snapshotLimit)
	if err != nil {
		return nil, err
	}
	return exportengine.ReadGUIPreviewSnapshot(raw)
}

var dataStamps = &stampMemo{ttl: 500 * time.Millisecond, entries: map[string]stampEntry{}}

type stampEntry struct {
	stamp string
	at    time.Time
}

type stampMemo struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]stampEntry
}

func (m *stampMemo) get(p *project.Project) (string, error) {
	key := projectKey(p)
	m.mu.Lock()
	if entry, ok := m.entries[key]; ok && time.Since(entry.at) < m.ttl {
		m.mu.Unlock()
		return entry.stamp, nil
	}
	m.mu.Unlock()
	stamp, err := projectStamp(p)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.entries[key] = stampEntry{stamp: stamp, at: time.Now()}
	m.mu.Unlock()
	return stamp, nil
}

func (m *stampMemo) forget(key string) {
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
}

func projectStamp(p *project.Project) (string, error) {
	h := sha256.New()
	stat := func(path, name string) error {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		fmt.Fprintf(h, "%s|%d|%d\n", name, info.Size(), info.ModTime().UnixNano())
		return nil
	}
	if err := stat(filepath.Join(p.Dir(), "project.json"), "project.json"); err != nil {
		return "", err
	}
	dir := filepath.Join(p.Dir(), "data")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := stat(filepath.Join(dir, name), name); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)[:16]), nil
}

type lazyIndex struct {
	once  sync.Once
	load  func(context.Context) (*guirender.Index, bool, error)
	index *guirender.Index
	found bool
	err   error
}

func (l *lazyIndex) get(ctx context.Context) (*guirender.Index, bool, error) {
	l.once.Do(func() {
		if l.load != nil {
			l.index, l.found, l.err = l.load(ctx)
		}
	})
	return l.index, l.found, l.err
}

func layerID(index *guirender.Index) string {
	if index == nil {
		return "none"
	}
	return fmt.Sprintf("%p", index)
}

func (s *Service) loadIndex(ctx context.Context, p *project.Project) (*guirender.Index, bool, error) {
	ref, attached, err := guiSnapshot(p)
	if err != nil {
		return nil, false, err
	}

	full := &lazyIndex{load: func(buildCtx context.Context) (*guirender.Index, bool, error) {
		return s.loadImportedIndex(buildCtx, p)
	}}
	vanilla := s.BaseIndex()
	baseKey := "vanilla|" + layerID(vanilla)
	legacy := false
	if attached {

		if legacy, err = snapshotLacksGameData(p, ref); err != nil {
			return nil, false, err
		}
		baseKey = "snapshot|" + ref.Key + "|" + ref.Revision + "|" + layerID(vanilla)
	}
	if index := s.withProjectData(ctx, p, full, baseKey, legacy); index != nil {
		return index, true, nil
	}
	base, found, err := full.get(ctx)
	if err != nil {
		return nil, false, err
	}
	return base, found && base != nil, nil
}

func (s *Service) withProjectData(ctx context.Context, p *project.Project, full *lazyIndex, baseKey string, legacyImport bool) *guirender.Index {
	baseOf := func(buildCtx context.Context) *guirender.Index {
		base, _, _ := full.get(buildCtx)
		return base
	}
	stamp, err := dataStamps.get(p)
	if err != nil {
		log.Printf("[gameview] project %s data digest: %v", p.Dir(), err)
		return baseOf(ctx)
	}
	key := projectKey(p) + "|projectdata|" + stamp + "|" + baseKey + "|" + fmt.Sprint(legacyImport)
	index, err := projectIndexes.load(ctx, key, func(buildCtx context.Context) (*guirender.Index, error) {
		return s.buildProjectLayer(buildCtx, p, full, legacyImport)
	})
	if err != nil || index == nil {

		if err != nil {
			log.Printf("[gameview] project %s layer: %v", p.Dir(), err)
		}
		return baseOf(ctx)
	}
	return index
}

func (s *Service) buildProjectLayer(ctx context.Context, p *project.Project, full *lazyIndex, legacyImport bool) (*guirender.Index, error) {
	build := buildProjectGameFiles(p, func() *guirender.Index {
		base, _, _ := full.get(ctx)
		return base
	}, legacyImport)

	overrides := spriteOverrides(p)
	projectSprites, projectSpriteVersions := projectSpriteSizes(p)

	if len(build.Files) == 0 && len(build.Flags) == 0 && len(overrides) == 0 && len(projectSprites) == 0 &&
		len(build.ShadowPaths) == 0 && len(build.ReplacePaths) == 0 {
		base, _, err := full.get(ctx)
		return base, err
	}
	var below *guirender.Index
	masked := false
	if build.ProjectionOK {
		switch build.Spec.Base {
		case baseVanilla:
			below = s.BaseIndex()
		case baseImportedUI:
			loaded, found, err := s.loadedModBase(ctx, p, build)
			if err != nil {
				log.Printf("[gameview] project %s imported UI baseline: %v", p.Dir(), err)
				below = s.BaseIndex()
			} else if found && loaded != nil {
				below, masked = loaded, true
			}
		}
	}
	if below == nil {
		base, _, err := full.get(ctx)
		if err != nil {
			return nil, err
		}
		below = base
	}

	if build.ProjectionOK && !masked && below != nil {
		unmasked := below
		var err error
		if build.Spec.ReplaceScope == replaceVanillaOnly {
			below, err = below.WithoutVanillaFolders(build.ReplacePaths)
		} else {
			below, err = below.WithoutFolders(build.ReplacePaths)
		}
		if err != nil {
			log.Printf("[gameview] project %s lower layers without %v: %v", p.Dir(), build.ReplacePaths, err)
			below = unmasked
		}
		if without, err := below.WithoutFiles(build.ShadowPaths); err != nil {
			log.Printf("[gameview] project %s lower layers without shadowed files: %v", p.Dir(), err)
		} else {
			below = without
		}
	}
	index, err := guirender.NewIndexWithFallback(build.Files, below)
	if err != nil || index == nil {
		return index, err
	}
	index.SetProjectFlags(build.Flags)
	if build.OwnedDLCs != nil {
		index.SetOwnedDLCs(*build.OwnedDLCs, false)
	}
	index.SetProjectSprites(projectSprites)
	index.SetProjectSpriteVersions(projectSpriteVersions)
	index.SetSpriteOverrides(overrides)
	index.SetProjectRawTextures(measureRawTextures(p, build.RawTextures))
	sources := make(map[string]string, len(build.Winners))
	for name, winner := range build.Winners {
		sources[name] = winner.SourceEntity
	}
	index.SetProjectFileSources(sources)
	return index, nil
}

func (s *Service) loadedModBase(ctx context.Context, p *project.Project, build projectBuild) (*guirender.Index, bool, error) {
	own, found, err := s.importedUIOwn(ctx, p)
	if err != nil || !found || own == nil {
		return nil, false, err
	}
	var lower *guirender.Index
	if vanilla := s.BaseIndex(); vanilla != nil {
		replaced := append(own.OwnFiles(), build.ShadowPaths...)
		if lower, err = vanilla.WithoutFilesAndFolders(replaced, build.ReplacePaths); err != nil {
			return nil, false, err
		}
	}

	var folders []string
	if build.Spec.ReplaceScope != replaceVanillaOnly {
		folders = build.ReplacePaths
	}
	top, err := own.WithoutFilesAndFolders(build.ShadowPaths, folders)
	if err != nil {
		return nil, false, err
	}
	return top.Over(lower), true, nil
}

func (s *Service) importedUIOwn(ctx context.Context, p *project.Project) (*guirender.Index, bool, error) {
	ref, found, err := guiSnapshot(p)
	if err != nil || !found {
		return nil, false, err
	}
	key := projectKey(p) + "|" + ref.Key + "|" + ref.Revision + "|ui-own"
	index, err := importedIndexes.load(ctx, key, func(context.Context) (*guirender.Index, error) {
		files, err := readSnapshotFiles(p, ref)
		if err != nil {
			return nil, err
		}
		for name := range files {
			if !exportengine.IsImportedUIBaselinePath(name) {
				delete(files, name)
			}
		}
		index, err := guirender.NewIndexWithFallback(files, nil)
		if err != nil {
			return nil, err
		}
		index.SetSourceArchiveRevision(ref.Key, ref.Revision)
		return index, nil
	})
	return index, err == nil && index != nil, err
}

func (s *Service) loadImportedIndex(ctx context.Context, p *project.Project) (*guirender.Index, bool, error) {
	vanilla := s.BaseIndex()
	ref, found, err := guiSnapshot(p)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return vanilla, vanilla != nil, nil
	}
	key := projectKey(p) + "|" + ref.Key + "|" + ref.Revision + "|full|" + layerID(vanilla)
	index, err := importedIndexes.load(ctx, key, func(context.Context) (*guirender.Index, error) {
		files, err := readSnapshotFiles(p, ref)
		if err != nil {
			return nil, err
		}

		shadowed := make([]string, 0, len(files))
		for name := range files {
			shadowed = append(shadowed, name)
		}
		below, err := vanilla.WithoutFiles(shadowed)
		if err != nil {
			return nil, err
		}
		index, err := guirender.NewIndexWithFallback(files, below)
		if err != nil {
			return nil, err
		}
		index.SetSourceArchiveRevision(ref.Key, ref.Revision)
		return index, nil
	})
	return index, err == nil, err
}
