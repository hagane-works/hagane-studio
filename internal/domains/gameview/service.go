package gameview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

type Service struct {
	deps domains.Deps
}

func New(deps domains.Deps) *Service { return &Service{deps: deps} }

func (s *Service) Deps() domains.Deps { return s.deps }

func (s *Service) gameLayer() (*gameLayer, error) {
	return gameLayerFor(s.deps.Game)
}

func (s *Service) gameLayerQuiet() (*gameLayer, string) {
	layer, err := s.gameLayer()
	if err != nil || layer == nil {
		return nil, ""
	}
	gameCache.Lock()
	identity := gameCache.key
	gameCache.Unlock()
	return layer, identity
}

func (s *Service) BaseIndex() *guirender.Index {
	layer, _ := s.gameLayer()
	if layer == nil {
		return nil
	}
	return layer.index
}

func (s *Service) Index(ctx context.Context, p *project.Project) (*guirender.Index, bool, error) {
	return s.loadIndex(ctx, p)
}

func (s *Service) ImportedIndex(ctx context.Context, p *project.Project) (*guirender.Index, bool, error) {
	return s.loadImportedIndex(ctx, p)
}

func (s *Service) InLanguage(r *http.Request, index *guirender.Index) *guirender.Index {
	return s.inLanguage(r, index)
}

var ErrNoGame = errNoGame

func (s *Service) ReadGameFile(rel string, maxBytes int64) ([]byte, error) {
	layer, err := s.gameLayer()
	if err != nil {
		return nil, err
	}
	body, _, err := layer.readFile(rel, maxBytes)
	return body, err
}

func (s *Service) GameRoots() ([]fs.FS, error) {
	layer, err := s.gameLayer()
	if err != nil {
		return nil, err
	}
	out := make([]fs.FS, len(layer.roots))
	for i, root := range layer.roots {
		out[i] = root.fsys
	}
	return out, nil
}

func (s *Service) Forget(p *project.Project) {
	dataStamps.forget(projectKey(p))
}

func (s *Service) Refresh(p *project.Project) {
	key := projectKey(p)
	projectIndexes.forget(key)
	importedIndexes.forget(key)
	dataStamps.forget(key)
}

func (s *Service) StoreRawBlob(p *project.Project, data []byte) (sha, key string, err error) {
	return storeRawBlob(p, data)
}

func storeRawBlob(p *project.Project, data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", errors.New("empty raw file")
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	blobs := p.Blobs()
	key := strings.Trim(blobs.EnvPrefix(), "/") + "/rawblobs/" + sha[:2] + "/" + sha
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "application/octet-stream", int64(len(data)), bytes.NewReader(data)); err != nil {
			return "", "", err
		}
	}
	return sha, key, nil
}
