package states

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"haganeworks/internal/hoi4/importsvc/world"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
)

const statesDir = "history/states"

type vanillaStateFile struct {
	StateID  int64
	Filename string

	Row map[string]any

	RawExtrasKeys []string

	Content VanillaStateContentCounts
}

type vanillaStates struct {
	DirMissing bool
	Files      int
	States     int

	Catalog []*vanillaStateFile

	ByID    map[int64]*vanillaStateFile
	Version gameVersion
}

type gameVersion struct {
	Raw     string
	Display string
	Known   bool
}

type vanillaGame struct {
	mu       sync.Mutex
	game     fs.FS
	names    map[string]map[string]string
	states   *vanillaStates
	geometry *baseGeometry
}

var vanilla = &vanillaGame{}

func (v *vanillaGame) use(game fs.FS) {
	if !sameFS(game, v.game) {
		v.game, v.names, v.states, v.geometry = game, nil, nil, nil
	}
}

func sameFS(a, b fs.FS) bool {
	if a == nil || b == nil || reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.String, reflect.Pointer:
		return a == b
	}
	return false
}

func (v *vanillaGame) stateNames(game fs.FS) map[string]map[string]string {
	if game == nil {
		return map[string]map[string]string{}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.use(game)
	if v.names == nil {
		v.names = readStateNames(game)
	}
	return v.names
}

func (v *vanillaGame) stateFiles(game fs.FS) *vanillaStates {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.use(game)
	if v.states == nil {
		v.states = readStateFiles(game)
	}
	return v.states
}

var stateLocRE = regexp.MustCompile(`^STATE_(\d+):\d*\s*"`)

func readStateNames(game fs.FS) map[string]map[string]string {
	out := map[string]map[string]string{}
	files, _ := fs.Glob(game, "localisation/*/state_names_l_*.yml")
	sort.Strings(files)
	for _, name := range files {
		base := path.Base(name)
		lang, ok := loc.LanguageToSuffix(strings.TrimSuffix(strings.TrimPrefix(base, "state_names_"), ".yml"))
		if !ok {
			continue
		}
		raw, err := fs.ReadFile(game, name)
		if err != nil {
			continue
		}
		text := strings.TrimPrefix(strings.ToValidUTF8(string(raw), "\ufffd"), "\ufeff")
		for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
			line = strings.TrimSpace(line)
			m := stateLocRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			i, j := strings.Index(line, `"`), strings.LastIndex(line, `"`)
			if j <= i {
				continue
			}
			id, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				continue
			}
			key := strconv.FormatInt(id, 10)
			if out[key] == nil {
				out[key] = map[string]string{}
			}
			out[key][lang] = line[i+1 : j]
		}
	}
	return out
}

func decodeGameScript(raw []byte) string {
	var text string
	if utf8.Valid(raw) {
		text = strings.TrimPrefix(string(raw), "\ufeff")
	} else if decoded, _, err := transform.Bytes(charmap.Windows1252.NewDecoder(), raw); err == nil {
		text = string(decoded)
	} else {
		text = string(raw)
	}
	return strings.ReplaceAll(text, "\x00", "")
}

func stateFileNames(game fs.FS) ([]string, bool) {
	entries, err := fs.ReadDir(game, statesDir)
	if err != nil {
		return nil, false
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, true
}

func readStateFiles(game fs.FS) *vanillaStates {
	out := &vanillaStates{ByID: map[int64]*vanillaStateFile{}, Version: readGameVersion(game)}
	names, ok := stateFileNames(game)
	if !ok {
		out.DirMissing = true
		return out
	}
	first := map[int64]*vanillaStateFile{}
	for _, name := range names {
		raw, err := fs.ReadFile(game, statesDir+"/"+name)
		if err != nil {
			continue
		}
		out.Files++
		text := decodeGameScript(raw)
		rows, _ := world.ImportStateFiles(text, name)
		parsed := map[int64]*vanillaStateFile{}
		for _, row := range rows {
			sid, ok := bigIntOf(row.MustGet("state_id"))
			if !ok {
				continue
			}
			out.States++
			entry := &vanillaStateFile{StateID: sid, Filename: name, Content: emptyContentCounts()}
			entry.Row, _ = pyPlain(row).(map[string]any)
			if extras, ok := row.MustGet("raw_extras").(*world.PyDict); ok {
				entry.RawExtrasKeys = append([]string{}, extras.Keys()...)
			}
			out.ByID[sid] = entry
			parsed[sid] = entry
			if first[sid] == nil {
				first[sid] = entry
			}
		}

		block, _, err := paradox.Parse(text)
		if err != nil || block == nil {
			continue
		}
		for _, ent := range block.Entries {
			if ent.Key == nil || ent.Block == nil || strings.TrimSpace(*ent.Key) != "state" {
				continue
			}
			if sid, ok := stateIDOfBlock(ent.Block); ok && parsed[sid] != nil {
				parsed[sid].Content = scanVanillaStateContent(ent.Block)
			}
		}
	}
	ids := make([]int64, 0, len(first))
	for id := range first {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		out.Catalog = append(out.Catalog, first[id])
	}
	return out
}

func readGameVersion(game fs.FS) gameVersion {
	raw, err := fs.ReadFile(game, "launcher-settings.json")
	if err != nil {
		return gameVersion{}
	}
	var payload struct {
		Version    string `json:"version"`
		RawVersion string `json:"rawVersion"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &payload); err != nil {
		return gameVersion{}
	}
	v := gameVersion{Raw: strings.TrimSpace(payload.RawVersion), Display: strings.TrimSpace(payload.Version)}
	v.Known = v.Raw != ""
	return v
}

func bigIntOf(v any) (int64, bool) {
	switch t := v.(type) {
	case *big.Int:
		if !t.IsInt64() {
			return 0, false
		}
		return t.Int64(), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	}
	return 0, false
}

func pyPlain(v any) any {
	switch t := v.(type) {
	case *big.Int:
		return json.Number(t.String())
	case *world.PyDict:
		out := make(map[string]any, t.Len())
		for _, k := range t.Keys() {
			value, _ := t.Get(k)
			out[k] = pyPlain(value)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = pyPlain(item)
		}
		return out
	default:
		return t
	}
}

func stateIDOfBlock(block *paradox.Block) (int64, bool) {
	for _, ent := range block.Entries {
		if ent.Key == nil || ent.Block != nil || strings.TrimSpace(*ent.Key) != "id" {
			continue
		}
		var n int64
		if _, err := fmt.Sscanf(strings.TrimSpace(ent.Scalar.StrValue()), "%d", &n); err == nil {
			return n, true
		}
		return 0, false
	}
	return 0, false
}
