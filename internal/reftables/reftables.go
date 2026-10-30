package reftables

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

const (
	IdeologiesFile = "ideologies.json"
	AlertsFile     = "alerts.json"
	TextColorsFile = "text_colors.json"
)

var files = []string{IdeologiesFile, AlertsFile, TextColorsFile}

//go:embed defaults/*.json
var defaults embed.FS

type IdeologyTable struct {
	Groups        []string      `json:"groups"`
	SubIdeologies []SubIdeology `json:"sub_ideologies"`

	groups   map[string]bool
	subGroup map[string]string
}

type SubIdeology struct {
	ID             string            `json:"id"`
	Group          string            `json:"group"`
	Name           map[string]string `json:"name,omitempty"`
	RandomDisabled bool              `json:"random_disabled,omitempty"`
}

func (t *IdeologyTable) IsGroup(id string) bool { return t.groups[id] }

func (t *IdeologyTable) IsSub(id string) bool {
	_, ok := t.subGroup[id]
	return ok
}

func (t *IdeologyTable) GroupOf(sub string) string { return t.subGroup[sub] }

func (t *IdeologyTable) Members(group string) []string {
	var out []string
	if t.groups[group] {
		out = append(out, group)
	}
	for _, sub := range t.SubIdeologies {
		if sub.Group == group {
			out = append(out, sub.ID)
		}
	}
	return out
}

type AlertTable struct {
	Engine   []EngineAlert        `json:"engine"`
	TextKeys map[string][2]string `json:"text_keys"`
}

type EngineAlert struct {
	Type  int    `json:"type"`
	Name  string `json:"name"`
	Frame int    `json:"frame"`
}

type textColors struct {
	Global map[string]string            `json:"global"`
	Fonts  map[string]map[string]string `json:"fonts"`
}

type tables struct {
	raw        map[string][]byte
	ideologies IdeologyTable
	alerts     AlertTable
}

var (
	current      atomic.Pointer[tables]
	defaultsOnce sync.Once
	builtIn      *tables
)

func Load(dir string) {
	if dir == "" {
		current.Store(build(nil))
		return
	}
	current.Store(build(func(name string) ([]byte, error) {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err == nil {
			log.Printf("reference table %s: reading %s", name, path)
		}
		return raw, err
	}))
}

func Use(files map[string][]byte) (restore func()) {
	previous := current.Swap(build(func(name string) ([]byte, error) {
		if raw, ok := files[name]; ok {
			return raw, nil
		}
		return nil, fs.ErrNotExist
	}))
	return func() { current.Store(previous) }
}

func Ideologies() *IdeologyTable { return &get().ideologies }

func Alerts() *AlertTable { return &get().alerts }

func Raw(name string) json.RawMessage { return get().raw[name] }

func get() *tables {
	if t := current.Load(); t != nil {
		return t
	}
	defaultsOnce.Do(func() { builtIn = build(nil) })
	return builtIn
}

func build(override func(name string) ([]byte, error)) *tables {
	t := &tables{raw: map[string][]byte{}}
	for _, name := range files {
		if override != nil {
			raw, err := override(name)
			if err == nil {
				err = t.set(name, raw)
			}
			if err == nil {
				continue
			}
			if !errors.Is(err, fs.ErrNotExist) {
				log.Printf("reference table %s: %v; using the built-in table", name, err)
			}
		}
		raw, err := defaults.ReadFile("defaults/" + name)
		if err == nil {
			err = t.set(name, raw)
		}
		if err != nil {
			panic(fmt.Sprintf("built-in reference table %s: %v", name, err))
		}
	}
	return t
}

func (t *tables) set(name string, raw []byte) error {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("not a JSON object")
	}
	switch name {
	case IdeologiesFile:
		var table IdeologyTable
		if err := json.Unmarshal(raw, &table); err != nil {
			return err
		}
		table.groups = map[string]bool{}
		for _, group := range table.Groups {
			table.groups[group] = true
		}
		table.subGroup = map[string]string{}
		for _, sub := range table.SubIdeologies {
			table.subGroup[sub.ID] = sub.Group
		}
		t.ideologies = table
	case AlertsFile:
		var table AlertTable
		if err := json.Unmarshal(raw, &table); err != nil {
			return err
		}
		t.alerts = table
	case TextColorsFile:
		if err := json.Unmarshal(raw, &textColors{}); err != nil {
			return err
		}
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return err
	}
	t.raw[name] = compact.Bytes()
	return nil
}
