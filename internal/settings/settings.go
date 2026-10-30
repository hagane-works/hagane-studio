package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Settings struct {
	ProjectsDir string `json:"projects_dir"`

	GameDir string `json:"game_dir"`

	ModDir string `json:"mod_dir,omitempty"`

	CWToolsDir string `json:"cwtools_dir,omitempty"`

	Port int `json:"port,omitempty"`

	CPU string `json:"cpu,omitempty"`
}

const (
	CPUOne  = "1"
	CPUHalf = "50%"
	CPUAll  = "100%"
)

func ValidCPU(value string) bool {
	return value == CPUOne || value == CPUHalf || value == CPUAll
}

func (s Settings) CPUSetting() string {
	if ValidCPU(s.CPU) {
		return s.CPU
	}
	return CPUAll
}

func (s Settings) CPUCores() int {
	switch s.CPUSetting() {
	case CPUOne:
		return 1
	case CPUHalf:
		return max(1, runtime.NumCPU()/2)
	}
	return runtime.NumCPU()
}

const DefaultPort = 37330

func (s Settings) ListenPort() int {
	if s.Port == 0 {
		return DefaultPort
	}
	return s.Port
}

type Store struct {
	path string
	mu   sync.Mutex
	data Settings
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Hagane Works", "settings.json"), nil
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) TablesDir() string { return filepath.Join(filepath.Dir(s.path), "tables") }

func ToolsTokenPath(settingsPath string) string {
	return filepath.Join(filepath.Dir(settingsPath), "tools-token")
}

func ReadToolsToken(settingsPath string) (string, error) {
	raw, err := os.ReadFile(ToolsTokenPath(settingsPath))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", errors.New("the tools-token file is empty")
	}
	return token, nil
}

func (s *Store) ToolsToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token, err := ReadToolsToken(s.path); err == nil {
		return token, nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	path := ToolsTokenPath(s.path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

func (s *Store) Update(change func(*Settings)) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	change(&next)
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return s.data, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return s.data, err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return s.data, err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return s.data, err
	}
	s.data = next
	return next, nil
}
