package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type BlobStore struct {
	root string
	uuid string
}

const BlobEnvPrefix = "local"

func (p *Project) Blobs() *BlobStore {
	return &BlobStore{root: p.FilesDir(), uuid: p.uuid}
}

func (b *BlobStore) EnvPrefix() string { return BlobEnvPrefix }

func (b *BlobStore) ProjectUUID() string { return b.uuid }

func (b *BlobStore) Path(key string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(key, "\\", "/"))
	if clean == "/" || strings.Contains(key, "\x00") {
		return "", fmt.Errorf("invalid blob key %q", key)
	}
	rel := strings.TrimPrefix(clean, "/")
	if projectScope := BlobEnvPrefix + "/p/" + b.uuid + "/"; b.uuid != "" && strings.HasPrefix(rel, projectScope) {
		rel = strings.TrimPrefix(rel, projectScope)
	} else {
		rel = strings.TrimPrefix(rel, BlobEnvPrefix+"/")
	}
	return filepath.Join(b.root, filepath.FromSlash(rel)), nil
}

func (b *BlobStore) Put(key, contentType string, size int64, body io.Reader) error {
	target, err := b.Path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*.tmp")
	if err != nil {
		return err
	}
	written, err := io.Copy(tmp, body)
	if err == nil && size >= 0 && written != size {
		err = fmt.Errorf("blob %s: wrote %d bytes, expected %d", key, written, size)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), target)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

var ErrBlobTooLarge = errors.New("blob is too large")

func (b *BlobStore) Get(key string, maxBytes int64) ([]byte, error) {
	target, err := b.Path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: %s", ErrBlobTooLarge, key)
	}
	return data, nil
}

func (b *BlobStore) Exists(key string) bool {
	target, err := b.Path(key)
	if err != nil {
		return false
	}
	_, err = os.Stat(target)
	return err == nil
}
