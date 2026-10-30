package importjob

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type CosmeticColors struct {
	Tag            string
	Color, ColorUI *cosmetic.RGB
}

func ParseCosmeticColors(text string) ([]CosmeticColors, error) {
	root, recoveries, err := paradox.Parse(strings.TrimPrefix(text, "\ufeff"))
	if err != nil {
		return nil, err
	}
	if len(recoveries) > 0 {
		return nil, fmt.Errorf("cosmetic color script needs repair: %s at line %d", recoveries[0].Code, recoveries[0].Line)
	}
	out := []CosmeticColors{}
	seen := map[string]CosmeticColors{}
	for _, entry := range root.Entries {
		if entry.Key == nil || entry.Block == nil {
			continue
		}
		var row CosmeticColors
		row.Tag = *entry.Key
		fields := map[string]bool{}
		for _, field := range entry.Block.Entries {
			if field.Key == nil || (*field.Key != "color" && *field.Key != "color_ui") {
				continue
			}
			value, err := parseCosmeticColor(field)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", row.Tag, *field.Key, err)
			}
			if fields[*field.Key] {
				old := row.Color
				if *field.Key == "color_ui" {
					old = row.ColorUI
				}
				if !reflect.DeepEqual(old, value) {
					return nil, fmt.Errorf("%s has conflicting duplicate %s", row.Tag, *field.Key)
				}
				continue
			}
			fields[*field.Key] = true
			if *field.Key == "color" {
				row.Color = value
			} else {
				row.ColorUI = value
			}
		}
		if len(fields) == 0 {
			continue
		}
		if err := cosmetic.ValidateTag(row.Tag); err != nil {
			return nil, err
		}
		if old, ok := seen[row.Tag]; ok {
			if !reflect.DeepEqual(old, row) {
				return nil, fmt.Errorf("conflicting cosmetic color definitions for %s", row.Tag)
			}
			continue
		}
		seen[row.Tag] = row
		out = append(out, row)
	}
	return out, nil
}
func parseCosmeticColor(entry *paradox.Entry) (*cosmetic.RGB, error) {
	if entry.Block == nil || len(entry.Block.Entries) != 3 {
		return nil, fmt.Errorf("expected three numeric channels")
	}
	var values [3]float64
	fractional := false
	for i, v := range entry.Block.Entries {
		if v.Key != nil || v.Block != nil || (v.Scalar.Kind != paradox.ScalarInt && v.Scalar.Kind != paradox.ScalarFloat) {
			return nil, fmt.Errorf("expected three numeric channels")
		}
		value, err := strconv.ParseFloat(v.Scalar.StrValue(), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("invalid numeric channel")
		}
		values[i] = value
		fractional = fractional || v.Scalar.Kind == paradox.ScalarFloat
	}
	model := "rgb"
	if entry.Qualifier != nil {
		model = strings.ToLower(*entry.Qualifier)
	}
	if model == "hsv" {
		for _, v := range values {
			if v < 0 || v > 1 {
				return nil, fmt.Errorf("HSV channels must be between 0 and 1")
			}
		}
		h, s, v := values[0]*6, values[1], values[2]
		c := v * s
		x := c * (1 - math.Abs(math.Mod(h, 2)-1))
		m := v - c
		switch int(h) % 6 {
		case 0:
			values = [3]float64{c, x, 0}
		case 1:
			values = [3]float64{x, c, 0}
		case 2:
			values = [3]float64{0, c, x}
		case 3:
			values = [3]float64{0, x, c}
		case 4:
			values = [3]float64{x, 0, c}
		case 5:
			values = [3]float64{c, 0, x}
		}
		for i := range values {
			values[i] = (values[i] + m) * 255
		}
	} else if model == "rgb" {
		normalized := fractional
		for _, v := range values {
			if v < 0 || v > 255 {
				return nil, fmt.Errorf("RGB channels must be between 0 and 255")
			}
			normalized = normalized && v <= 1
		}
		if normalized {
			for i := range values {
				values[i] *= 255
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported color model %q", model)
	}
	var rgb cosmetic.RGB
	for i, v := range values {
		rgb[i] = int(math.Round(v))
	}
	return &rgb, nil
}

func cosmeticColorJSON(color *cosmetic.RGB) any {
	if color == nil {
		return nil
	}
	b, _ := json.Marshal(color)
	return string(b)
}

type cosmeticStage struct {
	colorTags []string
	colors    map[string]cosmeticStagedColor
	rawPaths  []string
	raw       map[string]string
}

type cosmeticStagedColor struct{ color, colorUI any }

const cosmeticRawNote = "Cosmetic colors retained as raw: unresolved parsing or duplicate definitions; raw can override structured color edits"

func (s *session) stageCosmeticTags() error {
	s.cosmetic = cosmeticStage{colors: map[string]cosmeticStagedColor{}, raw: map[string]string{}}
	root := s.root
	files := []string{}
	err := filepath.WalkDir(filepath.Join(root, "common", "countries"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".txt") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	sort.Strings(files)
	seen := map[string]CosmeticColors{}
	sources := map[string]string{}
	ambiguous := map[string]bool{}

	preserve := func(file string) error {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		dest := filepath.ToSlash(rel)
		if _, ok := s.cosmetic.raw[dest]; !ok {
			s.cosmetic.raw[dest] = decodeScriptBytes(body)
			s.cosmetic.rawPaths = append(s.cosmetic.rawPaths, dest)
		}
		return nil
	}
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		rows, err := ParseCosmeticColors(decodeScriptBytes(body))
		if err != nil {
			if err := preserve(file); err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, file)
			s.warn(fmt.Sprintf("Cosmetic colors preserved as raw in %s: %v; structured colors were not imported and raw overrides may take precedence", filepath.ToSlash(rel), err))
			continue
		}
		for _, row := range rows {
			if ambiguous[row.Tag] {
				if err := preserve(file); err != nil {
					return err
				}
				continue
			}
			if old, ok := seen[row.Tag]; ok {
				if !reflect.DeepEqual(old, row) {
					if err := preserve(file); err != nil {
						return err
					}
					if err := preserve(sources[row.Tag]); err != nil {
						return err
					}

					delete(s.cosmetic.colors, row.Tag)
					ambiguous[row.Tag] = true
					s.warn(fmt.Sprintf("Conflicting cosmetic color definitions for %s preserved as raw; engine precedence needs author review and no structured winner was selected", row.Tag))
				}
				continue
			}
			seen[row.Tag] = row
			sources[row.Tag] = file

			s.cosmetic.colors[row.Tag] = cosmeticStagedColor{
				color: cosmeticJSONValue(cosmeticColorJSON(row.Color)), colorUI: cosmeticJSONValue(cosmeticColorJSON(row.ColorUI)),
			}
			s.cosmetic.colorTags = append(s.cosmetic.colorTags, row.Tag)
		}
	}
	return nil
}

func cosmeticJSONValue(text any) any {
	raw, ok := text.(string)
	if !ok {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func cosmeticJSONDistinct(a, b any) bool {
	if a == nil || b == nil {
		return (a == nil) != (b == nil)
	}
	normalize := func(v any) any {
		raw, err := json.Marshal(v)
		if err != nil {
			return v
		}
		var out any
		if json.Unmarshal(raw, &out) != nil {
			return v
		}
		return out
	}
	return !reflect.DeepEqual(normalize(a), normalize(b))
}

func cosmeticInt(v any) int64 {
	switch n := v.(type) {
	case int32:
		return int64(n)
	case int16:
		return int64(n)
	}
	return project.Int64(v)
}

func (s *session) mergeCosmeticTags(tx *project.Tx) error {

	tx.Delete("project_raw_files", func(row project.Row) bool {
		note, _ := row["note"].(string)
		dest, _ := row["dest_path"].(string)
		_, staged := s.cosmetic.raw[dest]
		return row["source"] == "imported" && strings.HasPrefix(note, "Cosmetic colors retained as raw:") && !staged
	})

	raw := newKeyedTable(tx, "project_raw_files", "dest_path")
	for _, dest := range s.cosmetic.rawPaths {
		content := s.cosmetic.raw[dest]
		if id, ok := raw.find(dest); ok {
			if tx.Find("project_raw_files", id)["source"] != "imported" {
				continue
			}
			if _, err := tx.Update("project_raw_files", id, func(row project.Row) {
				row["content"], row["note"] = content, cosmeticRawNote
			}); err != nil {
				return err
			}
			continue
		}
		raw.insertNew(project.Row{
			"dest_path": dest, "content": content, "generate_replace_path": false,
			"note": cosmeticRawNote, "is_binary": false, "source": "imported",
		})
	}

	countryTags := map[any]bool{}
	for _, row := range tx.Rows("countries") {
		countryTags[row["tag"]] = true
	}
	tags := newKeyedTable(tx, "project_cosmetic_tags", "tag")
	for _, tag := range s.cosmetic.colorTags {
		if _, staged := s.cosmetic.colors[tag]; !staged || countryTags[tag] {
			continue
		}
		tags.insertNew(project.Row{"tag": tag, "source": "imported", "revision": 1})
	}
	if err := cosmeticRegisterHistoryTags(tx, tags); err != nil {
		return err
	}

	for _, row := range tx.Rows("project_cosmetic_tags") {
		tag, _ := row["tag"].(string)
		staged := s.cosmetic.colors[tag]
		if !cosmeticJSONDistinct(row["imported_color"], staged.color) && !cosmeticJSONDistinct(row["imported_color_ui"], staged.colorUI) {
			continue
		}
		if _, err := tx.Update("project_cosmetic_tags", project.Int64(row["id"]), func(next project.Row) {
			next["imported_color"], next["imported_color_ui"] = staged.color, staged.colorUI
			next["revision"] = cosmeticInt(next["revision"]) + 1
		}); err != nil {
			return err
		}
	}
	cosmeticRefreshImportedNames(tx)
	return nil
}

func cosmeticRegisterHistoryTags(tx *project.Tx, tags *keyedTable) error {
	seen := map[string]bool{}
	list := []string{}
	for _, row := range tx.Rows("history_files") {
		tag, _ := row["set_cosmetic_tag"].(string)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		if err := cosmetic.ValidateTag(tag); err != nil {
			return err
		}
		list = append(list, tag)
	}
	sort.Strings(list)
	for _, tag := range list {
		tags.insertNew(project.Row{"tag": tag, "source": "imported", "revision": 1})
	}
	return nil
}

type cosmeticNameKey struct{ locale, suffix string }

type cosmeticLocCandidate struct {
	value                                        string
	replacePriority, sourceOrder, entryOrder, id int64
}

func (a cosmeticLocCandidate) ranksAbove(b cosmeticLocCandidate) bool {
	switch {
	case a.replacePriority != b.replacePriority:
		return a.replacePriority > b.replacePriority
	case a.sourceOrder != b.sourceOrder:
		return a.sourceOrder > b.sourceOrder
	case a.entryOrder != b.entryOrder:
		return a.entryOrder > b.entryOrder
	}
	return a.id > b.id
}

func cosmeticRefreshImportedNames(tx *project.Tx) {

	locales := map[string]bool{}
	for _, locale := range cosmetic.Locales() {
		locales[locale] = true
	}
	best := map[string]map[string]cosmeticLocCandidate{}
	for _, row := range tx.Rows("project_loc_entries") {
		locale, _ := row["locale"].(string)
		if !locales[locale] {
			continue
		}
		source, _ := row["source_file"].(string)
		if _, ok := exportengine.SafePreservedLocSourceFile(source); !ok {
			continue
		}
		key, _ := row["key"].(string)
		value, _ := row["value"].(string)
		candidate := cosmeticLocCandidate{
			value: value, replacePriority: cosmeticInt(row["replace_priority"]), sourceOrder: cosmeticInt(row["source_order"]),
			entryOrder: cosmeticInt(row["entry_order"]), id: cosmeticInt(row["id"]),
		}
		if best[key] == nil {
			best[key] = map[string]cosmeticLocCandidate{}
		}
		if current, ok := best[key][locale]; !ok || candidate.ranksAbove(current) {
			best[key][locale] = candidate
		}
	}
	tagRows := append([]project.Row(nil), tx.Rows("project_cosmetic_tags")...)
	sort.SliceStable(tagRows, func(i, j int) bool { return project.Int64(tagRows[i]["id"]) < project.Int64(tagRows[j]["id"]) })
	fresh := map[int64]map[cosmeticNameKey]string{}
	for _, row := range tagRows {
		id := project.Int64(row["id"])
		tag, _ := row["tag"].(string)
		names := map[cosmeticNameKey]string{}
		for _, suffix := range cosmetic.Suffixes() {
			key := tag
			if suffix != "" {
				key += "_" + suffix
			}
			for locale, candidate := range best[key] {
				names[cosmeticNameKey{locale, suffix}] = candidate.value
			}
		}
		fresh[id] = names
	}
	old := map[int64]map[cosmeticNameKey]string{}
	for _, row := range tx.Rows("project_cosmetic_tag_names") {
		if row["source"] != "imported" {
			continue
		}
		id := project.Int64(row["cosmetic_tag_id"])
		if old[id] == nil {
			old[id] = map[cosmeticNameKey]string{}
		}
		locale, _ := row["locale"].(string)
		suffix, _ := row["suffix"].(string)
		value, _ := row["value"].(string)
		old[id][cosmeticNameKey{locale, suffix}] = value
	}

	for _, row := range tagRows {
		id := project.Int64(row["id"])
		if cosmeticNamesEqual(fresh[id], old[id]) {
			continue
		}
		_, _ = tx.Update("project_cosmetic_tags", id, func(next project.Row) {
			next["revision"] = cosmeticInt(next["revision"]) + 1
		})
	}

	tx.Delete("project_cosmetic_tag_names", func(row project.Row) bool {
		if row["source"] != "imported" {
			return false
		}
		names, ok := fresh[project.Int64(row["cosmetic_tag_id"])]
		if !ok {
			return false
		}
		locale, _ := row["locale"].(string)
		suffix, _ := row["suffix"].(string)
		_, projected := names[cosmeticNameKey{locale, suffix}]
		return !projected
	})

	stored := newKeyedTable(tx, "project_cosmetic_tag_names", "cosmetic_tag_id", "locale", "suffix", "source")
	for _, row := range tagRows {
		id := project.Int64(row["id"])
		keys := make([]cosmeticNameKey, 0, len(fresh[id]))
		for key := range fresh[id] {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].locale != keys[j].locale {
				return keys[i].locale < keys[j].locale
			}
			return keys[i].suffix < keys[j].suffix
		})
		for _, key := range keys {
			value := fresh[id][key]
			if existing, ok := stored.find(id, key.locale, key.suffix, "imported"); ok {
				if tx.Find("project_cosmetic_tag_names", existing)["value"] != value {
					_, _ = tx.Update("project_cosmetic_tag_names", existing, func(next project.Row) { next["value"] = value })
				}
				continue
			}
			stored.insertNew(project.Row{"cosmetic_tag_id": id, "locale": key.locale, "suffix": key.suffix, "value": value, "source": "imported"})
		}
	}
}

func cosmeticNamesEqual(a, b map[cosmeticNameKey]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if other, ok := b[key]; !ok || other != value {
			return false
		}
	}
	return true
}
