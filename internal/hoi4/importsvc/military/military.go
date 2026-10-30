package military

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

type ParseResult struct {
	Rows     []map[string]any `json:"rows"`
	Warnings []string         `json:"warnings"`
}

type DivisionParseResult struct {
	Rows     []map[string]any `json:"rows"`
	Tail     string           `json:"tail"`
	Warnings []string         `json:"warnings"`
}

func parse(text string) (*paradox.Block, error) {
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, err
	}
	expandAtConstants(ast.Entries)
	return ast, nil
}

func expandAtConstants(entries []*paradox.Entry) {
	registry := map[string]paradox.Scalar{}
	for _, entry := range entries {
		if entry.Block == nil && entry.Key != nil && strings.HasPrefix(*entry.Key, "@") {
			registry[*entry.Key] = entry.Scalar
		}
	}
	var walk func([]*paradox.Entry)
	walk = func(items []*paradox.Entry) {
		for _, entry := range items {
			if entry.Block != nil {
				walk(entry.Block.Entries)
				continue
			}
			if entry.Scalar.Kind == paradox.ScalarStr && strings.HasPrefix(entry.Scalar.Str, "@") {
				if value, ok := registry[entry.Scalar.Str]; ok {
					entry.Scalar = value
				}
			}
		}
	}
	walk(entries)
}

func findOne(entries []*paradox.Entry, key string) *paradox.Entry {
	for _, entry := range entries {
		if entry.Key != nil && *entry.Key == key {
			return entry
		}
	}
	return nil
}

func findAll(entries []*paradox.Entry, key string) []*paradox.Entry {
	out := []*paradox.Entry{}
	for _, entry := range entries {
		if entry.Key != nil && *entry.Key == key {
			out = append(out, entry)
		}
	}
	return out
}

func childBlock(entries []*paradox.Entry, key string) *paradox.Block {
	entry := findOne(entries, key)
	if entry == nil {
		return nil
	}
	return entry.Block
}

func nativeScalar(value paradox.Scalar) any {
	switch value.Kind {
	case paradox.ScalarInt:

		return new(big.Int).Set(value.Int)
	case paradox.ScalarFloat:
		return value.Float
	default:
		return value.Str
	}
}

func scalarText(value paradox.Scalar) string { return importsvc.PyScalarStr(value) }

func scalar(entries []*paradox.Entry, key string, fallback paradox.Scalar) paradox.Scalar {
	entry := findOne(entries, key)
	if entry == nil || entry.Block != nil {
		return fallback
	}
	return entry.Scalar
}

func blockRaw(block *paradox.Block) string {
	return strings.TrimRight(importsvc.ToScript(block, 0), "\n") + "\n"
}

func accumulate(target map[string]any, key string, value any) {
	old, ok := target[key]
	if !ok {
		target[key] = value
		return
	}
	if values, ok := old.([]any); ok {
		target[key] = append(values, value)
		return
	}
	target[key] = []any{old, value}
}

func scalarFloat(value paradox.Scalar) (*float64, bool) {
	raw := strings.TrimSpace(scalarText(value))
	switch strings.ToLower(raw) {
	case "nan", "+nan", "-nan":
		n := math.NaN()
		return &n, true
	case "inf", "+inf", "infinity", "+infinity":
		n := math.Inf(1)
		return &n, true
	case "-inf", "-infinity":
		n := math.Inf(-1)
		return &n, true
	}
	n, err := strconv.ParseFloat(raw, 64)
	return &n, err == nil
}

func finiteFloat(value paradox.Scalar) (*float64, bool) {
	n, ok := scalarFloat(value)
	return n, ok && !math.IsNaN(*n) && !math.IsInf(*n, 0)
}

func wholeInt(value paradox.Scalar) (*int64, bool) {
	n, ok := finiteFloat(value)
	if !ok || math.Trunc(*n) != *n || *n < math.MinInt64 || *n > math.MaxInt64 {
		return nil, false
	}
	v := int64(*n)
	return &v, true
}

func truncatedInt(value paradox.Scalar) (*int64, bool) {
	n, ok := finiteFloat(value)
	if !ok || *n < math.MinInt64 || *n > math.MaxInt64 {
		return nil, false
	}
	v := int64(*n)
	return &v, true
}

func scalarBool(value paradox.Scalar) (*bool, bool) {
	switch strings.ToLower(strings.TrimSpace(scalarText(value))) {
	case "yes", "true", "1":
		v := true
		return &v, true
	case "no", "false", "0":
		v := false
		return &v, true
	default:
		return nil, false
	}
}

var equipmentFloatFields = []string{
	"soft_attack", "hard_attack", "ap_attack", "air_attack", "breakthrough",
	"defense", "armor_value", "hardness", "max_strength", "build_cost_ic",
	"reliability", "maximum_speed", "naval_speed", "fire_range",
	"lg_armor_piercing", "lg_attack", "hg_armor_piercing", "hg_attack",
	"torpedo_attack", "anti_air_attack", "shore_bombardment", "evasion",
	"surface_detection", "sub_attack", "sub_detection", "surface_visibility",
	"sub_visibility", "naval_range", "port_capacity_usage",
	"search_and_destroy_coordination", "convoy_raiding_coordination",
	"air_defence", "air_range", "air_agility", "air_ground_attack",
	"air_bombing", "air_superiority", "naval_strike_attack",
	"naval_strike_targetting", "carrier_size", "default_carrier_composition_weight",
}

var equipmentBoolFields = []string{"is_archetype", "is_buildable", "can_license", "carrier_capable"}

var equipmentKnown = func() map[string]bool {
	out := map[string]bool{
		"year": true, "type": true, "picture": true, "archetype": true,
		"parent": true, "manpower": true, "resources": true,
	}
	for _, key := range equipmentFloatFields {
		out[key] = true
	}
	for _, key := range equipmentBoolFields {
		out[key] = true
	}
	return out
}()

func mapEquipment(id string, body []*paradox.Entry, warnings *[]string) map[string]any {
	out := map[string]any{"equipment_id": id}
	raw := map[string]any{}
	preserve := func(key string, entry *paradox.Entry) {
		value := nativeScalar(entry.Scalar)
		if entry.Block != nil {
			value = blockRaw(entry.Block)
		}
		accumulate(raw, key, value)
		*warnings = append(*warnings, fmt.Sprintf("%s.%s: preserved unparseable scalar %v in raw_extras", id, key, pythonRepr(value)))
	}
	for _, pair := range [][2]string{{"type", "equipment_type"}, {"picture", "picture"}, {"archetype", "archetype"}, {"parent", "parent"}} {
		if entry := findOne(body, pair[0]); entry != nil {
			if entry.Block != nil {
				accumulate(raw, pair[0], blockRaw(entry.Block))
			} else {
				out[pair[1]] = scalarText(entry.Scalar)
			}
		}
	}
	for _, key := range []string{"year", "manpower"} {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry)
			} else if value, ok := wholeInt(entry.Scalar); ok {
				out[key] = *value
			} else {
				preserve(key, entry)
			}
		}
	}
	for _, key := range equipmentBoolFields {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry)
			} else if value, ok := scalarBool(entry.Scalar); ok {
				out[key] = *value
				if key == "is_archetype" && !*value {
					accumulate(raw, key, "no")
				}
			} else {
				preserve(key, entry)
			}
		}
	}
	for _, key := range equipmentFloatFields {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry)
			} else if value, ok := finiteFloat(entry.Scalar); ok {
				out[key] = *value
			} else {
				preserve(key, entry)
			}
		}
	}
	if entry := findOne(body, "resources"); entry != nil {
		if entry.Block == nil {
			preserve("resources", entry)
		} else {
			resources := map[string]any{}
			valid := true
			for _, child := range entry.Block.Entries {
				if child.Key == nil || child.Block != nil {
					valid = false
					continue
				}
				resources[*child.Key] = nativeScalar(child.Scalar)
			}
			if valid {
				out["resources"] = resources
			} else {
				accumulate(raw, "resources", blockRaw(entry.Block))
			}
		}
	}
	for _, entry := range body {
		if entry.Key == nil || equipmentKnown[*entry.Key] {
			continue
		}
		value := nativeScalar(entry.Scalar)
		if entry.Block != nil {
			value = blockRaw(entry.Block)
		}
		accumulate(raw, *entry.Key, value)
	}
	if len(raw) > 0 {
		out["raw_extras"] = raw
	}
	return out
}

func ImportEquipment(text, sourceFile string) ParseResult {
	result := ParseResult{Rows: []map[string]any{}, Warnings: []string{}}
	ast, err := parse(text)
	if err != nil {
		result.Warnings = append(result.Warnings, "parse failed: "+err.Error())
		return result
	}
	wrapper := childBlock(ast.Entries, "equipments")
	if wrapper == nil {
		result.Warnings = append(result.Warnings, "no `equipments = { ... }` wrapper found")
		return result
	}
	path := strings.ReplaceAll(sourceFile, "\\", "/")
	for _, entry := range wrapper.Entries {
		if entry.Key == nil || strings.HasPrefix(*entry.Key, "@") || entry.Block == nil {
			continue
		}
		row := mapEquipment(*entry.Key, entry.Block.Entries, &result.Warnings)
		if path != "" {
			row["source_file"] = path
		}
		result.Rows = append(result.Rows, row)
	}
	return result
}

var subStringFields = []string{"abbreviation", "sprite", "map_icon_category", "transport", "group"}
var subIntFields = []string{"priority", "ai_priority"}
var subBoolFields = []string{
	"active", "cavalry", "special_forces", "marines", "can_exfiltrate_from_coast",
	"mountaineers", "can_be_parachuted", "affects_speed",
}
var subTokenFields = map[string]string{"type": "unit_types", "categories": "categories", "essential": "essential"}
var terrainKeys = map[string]bool{
	"forest": true, "hills": true, "mountain": true, "jungle": true,
	"marsh": true, "plains": true, "urban": true, "desert": true,
	"river": true, "amphibious": true, "fort": true, "shore": true,
	"snow": true, "ice": true,
}
var subKnown = func() map[string]bool {
	out := map[string]bool{"need": true}
	for _, key := range subStringFields {
		out[key] = true
	}
	for _, key := range subIntFields {
		out[key] = true
	}
	for _, key := range subBoolFields {
		out[key] = true
	}
	for key := range subTokenFields {
		out[key] = true
	}
	return out
}()

func tokenList(entry *paradox.Entry) ([]string, bool) {
	if entry.Block == nil {
		return []string{scalarText(entry.Scalar)}, true
	}
	values := []string{}
	for _, child := range entry.Block.Entries {
		if child.Key != nil || child.Block != nil {
			return nil, false
		}
		values = append(values, scalarText(child.Scalar))
	}
	return values, true
}

func scalarMapping(block *paradox.Block) (map[string]any, bool) {
	out := map[string]any{}
	for _, entry := range block.Entries {
		if entry.Key == nil || entry.Block != nil {
			return nil, false
		}
		out[*entry.Key] = nativeScalar(entry.Scalar)
	}
	return out, true
}

func mapSubUnit(id string, body []*paradox.Entry, sourceFile string, ordering int, warnings *[]string) map[string]any {
	out := map[string]any{"sub_unit_id": id, "ordering": ordering}
	if sourceFile != "" {
		out["source_file"] = strings.ReplaceAll(sourceFile, "\\", "/")
	}
	raw := map[string]any{}
	preserve := func(key string, entry *paradox.Entry, reason string, warn bool) {
		value := nativeScalar(entry.Scalar)
		if entry.Block != nil {
			value = blockRaw(entry.Block)
		}
		accumulate(raw, key, value)
		if warn {
			*warnings = append(*warnings, fmt.Sprintf("%s.%s: %s; preserved in raw_extras", id, key, reason))
		}
	}
	for _, key := range subStringFields {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry, "expected scalar", true)
			} else {
				out[key] = scalarText(entry.Scalar)
			}
		}
	}
	for _, key := range subIntFields {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry, "expected integer", true)
			} else if value, ok := wholeInt(entry.Scalar); ok {
				out[key] = *value
			} else {
				preserve(key, entry, "expected integer", true)
			}
		}
	}
	for _, key := range subBoolFields {
		if entry := findOne(body, key); entry != nil {
			if entry.Block != nil {
				preserve(key, entry, "expected yes/no", true)
			} else if value, ok := scalarBool(entry.Scalar); ok {
				out[key] = *value
			} else {
				preserve(key, entry, "expected yes/no", true)
			}
		}
	}
	for scriptKey, column := range subTokenFields {
		if entry := findOne(body, scriptKey); entry != nil {
			if values, ok := tokenList(entry); ok {
				out[column] = values
			} else {
				preserve(scriptKey, entry, "expected scalar or token block", true)
			}
		}
	}
	if entry := findOne(body, "need"); entry != nil {
		if entry.Block != nil {
			if values, ok := scalarMapping(entry.Block); ok {
				out["need"] = values
			} else {
				preserve("need", entry, "expected equipment-to-amount block", true)
			}
		} else {
			preserve("need", entry, "expected equipment-to-amount block", true)
		}
	}
	stats := map[string]any{}
	terrain := map[string]any{}
	for _, entry := range body {
		if entry.Key == nil || subKnown[*entry.Key] {
			continue
		}
		if terrainKeys[*entry.Key] && entry.Block != nil {
			if values, ok := scalarMapping(entry.Block); ok {
				terrain[*entry.Key] = values
				continue
			}
		}
		if entry.Block == nil && (entry.Scalar.Kind == paradox.ScalarInt || entry.Scalar.Kind == paradox.ScalarFloat) {
			stats[*entry.Key] = nativeScalar(entry.Scalar)
			continue
		}
		preserve(*entry.Key, entry, "unsupported structured value", false)
	}
	if len(stats) > 0 {
		out["stats"] = stats
	}
	if len(terrain) > 0 {
		out["terrain_modifiers"] = terrain
	}
	if len(raw) > 0 {
		out["raw_extras"] = raw
	}
	return out
}

func ImportSubUnits(text, sourceFile string) ParseResult {
	result := ParseResult{Rows: []map[string]any{}, Warnings: []string{}}
	ast, err := parse(text)
	if err != nil {
		result.Warnings = append(result.Warnings, "parse failed: "+err.Error())
		return result
	}
	wrappers := findAll(ast.Entries, "sub_units")
	if len(wrappers) == 0 {
		result.Warnings = append(result.Warnings, "no `sub_units = { ... }` wrapper found")
		return result
	}
	ordering := 0
	for _, wrapper := range wrappers {
		if wrapper.Block == nil {
			result.Warnings = append(result.Warnings, "ignored scalar `sub_units` wrapper")
			continue
		}
		for _, entry := range wrapper.Block.Entries {
			if entry.Key == nil || entry.Block == nil {
				continue
			}
			result.Rows = append(result.Rows, mapSubUnit(*entry.Key, entry.Block.Entries, sourceFile, ordering, &result.Warnings))
			ordering++
		}
	}
	return result
}

var divisionKnown = map[string]bool{
	"name": true, "division_names_group": true, "regiments": true,
	"support": true, "is_locked": true, "priority": true,
}

type textSpan struct{ start, end int }

func identifierRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }

func divisionSpans(text string) []textSpan {
	const keyword = "division_template"
	spans := []textSpan{}
	n, i, depth := len(text), 0, 0
	for i < n {
		switch text[i] {
		case '#':
			for i < n && text[i] != '\n' {
				i++
			}
			continue
		case '"':
			i++
			for i < n && text[i] != '"' {
				if text[i] == '\\' {
					i++
				}
				i++
			}
			if i < n {
				i++
			}
			continue
		case '{':
			depth++
			i++
			continue
		case '}':
			depth--
			i++
			continue
		}
		if depth != 0 || !strings.HasPrefix(text[i:], keyword) {
			i++
			continue
		}
		beforeOK := i == 0
		if !beforeOK {
			r, _ := utf8LastRune(text[:i])
			beforeOK = !identifierRune(r)
		}
		j := i + len(keyword)
		afterOK := j >= n
		if !afterOK {
			r, _ := utf8FirstRune(text[j:])
			afterOK = !identifierRune(r)
		}
		if !beforeOK || !afterOK {
			i = j
			continue
		}
		k := skipSpace(text, j)
		if k >= n || text[k] != '=' {
			i = j
			continue
		}
		k = skipSpace(text, k+1)
		if k >= n || text[k] != '{' {
			i = j
			continue
		}
		d, m := 1, k+1
		for m < n && d > 0 {
			switch text[m] {
			case '#':
				for m < n && text[m] != '\n' {
					m++
				}
				continue
			case '"':
				m++
				for m < n && text[m] != '"' {
					if text[m] == '\\' {
						m++
					}
					m++
				}
				if m < n {
					m++
				}
				continue
			case '{':
				d++
			case '}':
				d--
			}
			m++
		}
		spans = append(spans, textSpan{start: i, end: m})
		i = m
	}
	return spans
}

func utf8FirstRune(text string) (rune, int) {
	for _, r := range text {
		return r, len(string(r))
	}
	return 0, 0
}

func utf8LastRune(text string) (rune, int) {
	var last rune
	var size int
	for _, r := range text {
		last, size = r, len(string(r))
	}
	return last, size
}

func skipSpace(text string, start int) int {
	for start < len(text) && strings.ContainsRune(" \t\r\n", rune(text[start])) {
		start++
	}
	return start
}

var divisionGap = regexp.MustCompile(`\n[ \t]*\n[ \t]*\n+`)

func divisionTail(text string, spans []textSpan) string {
	if len(spans) == 0 {
		return strings.TrimSpace(text)
	}
	var out strings.Builder
	previous := 0
	for _, span := range spans {
		out.WriteString(text[previous:span.start])
		previous = span.end
	}
	out.WriteString(text[previous:])
	return strings.TrimSpace(divisionGap.ReplaceAllString(out.String(), "\n\n"))
}

func parseGrid(block *paradox.Block) []map[string]any {
	out := []map[string]any{}
	if block == nil {
		return out
	}
	zero := paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(0)}
	for _, entry := range block.Entries {
		if entry.Key == nil || entry.Block == nil {
			continue
		}
		x, _ := truncatedInt(scalar(entry.Block.Entries, "x", zero))
		y, _ := truncatedInt(scalar(entry.Block.Entries, "y", zero))
		var xv, yv int64
		if x != nil {
			xv = *x
		}
		if y != nil {
			yv = *y
		}
		out = append(out, map[string]any{"token": *entry.Key, "x": xv, "y": yv})
	}
	return out
}

func ImportDivisionTemplates(text string) DivisionParseResult {
	spans := divisionSpans(text)
	result := DivisionParseResult{Rows: []map[string]any{}, Tail: divisionTail(text, spans), Warnings: []string{}}
	if len(spans) == 0 {
		result.Warnings = append(result.Warnings, "no `division_template = { ... }` block found")
	}
	for ordering, span := range spans {
		ast, _, err := paradox.Parse(text[span.start:span.end])
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("template #%d parse failed: %v", ordering, err))
			continue
		}
		body := childBlock(ast.Entries, "division_template")
		if body == nil {
			continue
		}
		fallback := paradox.Scalar{Kind: paradox.ScalarStr, Str: fmt.Sprintf("Division %d", ordering)}
		row := map[string]any{
			"template_name": scalarText(scalar(body.Entries, "name", fallback)),
			"ordering":      ordering,
			"regiments":     parseGrid(childBlock(body.Entries, "regiments")),
			"support":       parseGrid(childBlock(body.Entries, "support")),
		}
		if entry := findOne(body.Entries, "division_names_group"); entry != nil {
			fallback := paradox.Scalar{Kind: paradox.ScalarStr, Str: ""}
			row["division_names_group"] = scalarText(scalar(body.Entries, "division_names_group", fallback))
		}
		if entry := findOne(body.Entries, "is_locked"); entry != nil {
			if value, ok := scalarBool(scalar(body.Entries, "is_locked", paradox.Scalar{})); ok {
				row["is_locked"] = *value
			} else {
				row["is_locked"] = nil
			}
		}
		if entry := findOne(body.Entries, "priority"); entry != nil {
			if value, ok := truncatedInt(scalar(body.Entries, "priority", paradox.Scalar{})); ok {
				row["priority"] = *value
			} else {
				row["priority"] = nil
			}
		}
		raw := map[string]any{}
		for _, entry := range body.Entries {
			if entry.Key == nil || divisionKnown[*entry.Key] {
				continue
			}
			if entry.Block != nil {
				raw[*entry.Key] = importsvc.ToScript(entry.Block, 0)
			} else {
				raw[*entry.Key] = nativeScalar(entry.Scalar)
			}
		}
		if len(raw) > 0 {
			row["raw_extras"] = raw
		}
		result.Rows = append(result.Rows, row)
	}
	return result
}

func pythonRepr(value any) string {
	switch v := value.(type) {
	case string:
		quote := byte('\'')
		if strings.ContainsRune(v, '\'') && !strings.ContainsRune(v, '"') {
			quote = '"'
		}
		var out strings.Builder
		out.WriteByte(quote)
		for _, r := range v {
			switch r {
			case '\\':
				out.WriteString(`\\`)
			case '\n':
				out.WriteString(`\n`)
			case '\r':
				out.WriteString(`\r`)
			case '\t':
				out.WriteString(`\t`)
			default:
				if byte(r) == quote && r < 128 {
					out.WriteByte('\\')
				}
				out.WriteRune(r)
			}
		}
		out.WriteByte(quote)
		return out.String()
	default:
		return fmt.Sprint(v)
	}
}
