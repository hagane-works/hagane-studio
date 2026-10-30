package importsvc

import (
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type ImportSummary struct {
	ObjectType      string
	Total           int
	PreservedFields *OrderedCounts
	Warnings        []string
}

func NewImportSummary(objectType string) *ImportSummary {
	return &ImportSummary{
		ObjectType:      objectType,
		PreservedFields: NewOrderedCounts(),
		Warnings:        []string{},
	}
}

func (s *ImportSummary) AddPreserved(fieldName string) {
	s.PreservedFields.Incr(fieldName)
}

type OrderedCounts struct {
	Keys   []string
	counts map[string]int
}

func NewOrderedCounts() *OrderedCounts {
	return &OrderedCounts{counts: map[string]int{}}
}

func (m *OrderedCounts) Incr(k string) {
	if _, ok := m.counts[k]; !ok {
		m.Keys = append(m.Keys, k)
	}
	m.counts[k]++
}

func (m *OrderedCounts) Get(k string) int { return m.counts[k] }

type TreeBundle struct {
	TreeData        map[string]any
	NodesData       []map[string]any
	TreeKind        string
	SharedFocusRefs []map[string]any
}

func NewTreeBundle(treeData map[string]any, nodesData []map[string]any) *TreeBundle {
	return &TreeBundle{TreeData: treeData, NodesData: nodesData, TreeKind: "primary"}
}

const WarnSentinel = "§§"

var contentNoun = map[string]string{
	"character":    "a characters file",
	"history_file": "a country history file",
	"state":        "a state file",
	"super_event":  "a super-event file",
	"bookmark":     "a bookmark file",
	"focus_node":   "a focus tree file",
	"technology":   "a technology file",
	"idea":         "an ideas file",
	"event":        "an events file",
	"decision":     "a decisions file",
}

type WarnParam struct {
	Key   string
	Value any
}

func pyJSONValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return pyJSONString(t)
	case int:
		return big.NewInt(int64(t)).String()
	case int64:
		return big.NewInt(t).String()
	case *big.Int:
		return t.String()
	case float64:
		return pyFloatRepr(t)
	default:
		return pyJSONString("")
	}
}

func CodedWarning(code string, params []WarnParam, fallback string) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, p := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(pyJSONString(p.Key))
		b.WriteString(": ")
		b.WriteString(pyJSONValue(p.Value))
	}
	b.WriteByte('}')
	return WarnSentinel + code + WarnSentinel + b.String() + WarnSentinel + fallback
}

func ParseSkipWarning(summary *ImportSummary, filename string) string {
	noun, ok := contentNoun[summary.ObjectType]
	if !ok {
		noun = "a file"
	}
	where := filename
	if where == "" {
		where = noun
	}
	return CodedWarning(
		"parse_skipped",
		[]WarnParam{{Key: "file", Value: where}},
		"Couldn't read "+where+" — it was skipped, and the rest imported fine.",
	)
}

func BlockToText(b *paradox.Block, indent int) string {
	result := ToScript(b, indent)
	if result == "" {
		return "\n"
	}
	if !strings.Contains(result, "\n") {
		result += "\n"
	}
	return result
}

func CoerceTraitInt(value paradox.Scalar, summary *ImportSummary, traitID, fieldName string) (intVal *big.Int, strVal string, isInt bool) {
	raw := pyStrip(PyScalarStr(value))
	if f, ok := pyFloat(raw); ok {
		if z, ok2 := pyIntFromFloat(f); ok2 {
			return z, "", true
		}
	}
	if summary != nil {
		summary.Warnings = append(summary.Warnings, CodedWarning(
			"trait_script_scalar_preserved",
			[]WarnParam{
				{Key: "trait", Value: traitID},
				{Key: "field", Value: fieldName},
				{Key: "value", Value: raw},
			},
			"Trait "+traitID+" uses script value "+raw+" for "+fieldName+". "+
				"It was kept intact for export; the numeric editor may not edit it.",
		))
	}
	return nil, raw, false
}

func CoerceTraitNumber(value paradox.Scalar) paradox.Scalar {
	raw := pyStrip(PyScalarStr(value))
	f, ok := pyFloat(raw)
	if !ok {
		return paradox.Scalar{Kind: paradox.ScalarStr, Str: raw}
	}
	if pyFloatIsInteger(f) && !strings.Contains(raw, ".") && !strings.ContainsAny(raw, "eE") {
		z, _ := pyIntFromFloat(f)
		return paradox.Scalar{Kind: paradox.ScalarInt, Int: z}
	}
	return paradox.Scalar{Kind: paradox.ScalarFloat, Float: f}
}

type FlatModifiers struct {
	Keys   []string
	values map[string]paradox.Scalar
}

func (m *FlatModifiers) Get(k string) (paradox.Scalar, bool) {
	v, ok := m.values[k]
	return v, ok
}

func (m *FlatModifiers) set(k string, v paradox.Scalar) {
	if _, ok := m.values[k]; !ok {
		m.Keys = append(m.Keys, k)
	}
	m.values[k] = v
}

func ParseFlatModifierBlock(entries []*paradox.Entry) *FlatModifiers {
	out := &FlatModifiers{values: map[string]paradox.Scalar{}}
	for _, child := range entries {
		if child.Key == nil {
			continue
		}
		ck := *child.Key
		if child.Block != nil {
			out.set(ck, paradox.Scalar{
				Kind: paradox.ScalarStr, Str: BlockToText(child.Block, 0),
			})
		} else {
			out.set(ck, CoerceTraitNumber(child.Scalar))
		}
	}
	return out
}

func CoerceInt(raw *paradox.Scalar, def int64) *big.Int {
	if raw == nil {
		return big.NewInt(def)
	}
	switch raw.Kind {
	case paradox.ScalarInt:
		return raw.Int
	case paradox.ScalarFloat:
		if z, ok := pyIntFromFloat(raw.Float); ok {
			return z
		}
		return big.NewInt(def)
	default:
		if z, ok := pyIntFromStr(raw.Str); ok {
			return z
		}
		return big.NewInt(def)
	}
}

func CoerceFloat(raw *paradox.Scalar, def float64) (f float64, overflow bool) {
	if raw == nil {
		return def, false
	}
	switch raw.Kind {
	case paradox.ScalarFloat:
		return raw.Float, false
	case paradox.ScalarInt:
		if f, ok := pyFloatFromBigInt(raw.Int); ok {
			return f, false
		}
		return 0, true
	default:
		if f, ok := pyFloat(raw.Str); ok {
			return f, false
		}
		return def, false
	}
}

func TopLevelLineBounds(lines []string, ast []*paradox.Entry, idx int) (start, end int, ok bool) {
	n := len(lines)
	ent := ast[idx]
	startLine := ent.Line
	if startLine < 1 || startLine > n {
		return 0, 0, false
	}
	start = startLine - 1

	prevFloor := 0
	if idx > 0 {
		prevStart := ast[idx-1].Line
		if prevStart >= 1 && prevStart <= n {
			prevFloor = prevStart
		}
	}

	s := start
	for s-1 >= prevFloor {
		prior := pyStrip(lines[s-1])
		if prior == "" || strings.HasPrefix(prior, "#") {
			s--
		} else {
			break
		}
	}
	start = s

	if idx+1 < len(ast) {
		nextStart := ast[idx+1].Line
		if nextStart >= 1 && nextStart <= n {
			end = nextStart - 1
		} else {
			end = n
		}
	} else {
		end = n
	}
	if end <= start {
		end = start + 1
		if end > n {
			end = n
		}
	}
	return start, end, true
}

func EmitRawScriptFromBounds(lines []string, bounds [][2]int) string {
	if len(bounds) == 0 {
		return ""
	}
	var chunks []string
	runStart, runEnd := bounds[0][0], bounds[0][1]
	for _, b := range bounds[1:] {
		bs, be := b[0], b[1]
		if bs <= runEnd {

			if be > runEnd {
				runEnd = be
			}
		} else {
			chunks = append(chunks, strings.Join(lines[runStart:runEnd], "\n"))
			runStart, runEnd = bs, be
		}
	}
	chunks = append(chunks, strings.Join(lines[runStart:runEnd], "\n"))
	var cleaned []string
	for _, c := range chunks {
		c = pyRStrip(pyStripChar(c, '\n'))
		if pyStrip(c) != "" {
			cleaned = append(cleaned, c)
		}
	}
	return pyRStrip(strings.Join(cleaned, "\n\n"))
}

type RawExtras struct {
	Keys   []string
	values map[string]any
}

func NewRawExtras() *RawExtras { return &RawExtras{values: map[string]any{}} }

func (r *RawExtras) Get(k string) (any, bool) {
	v, ok := r.values[k]
	return v, ok
}

func (r *RawExtras) set(k string, v any) {
	if _, ok := r.values[k]; !ok {
		r.Keys = append(r.Keys, k)
	}
	r.values[k] = v
}

func isEmptyStrValue(v any) bool {
	switch t := v.(type) {
	case string:
		return t == ""
	case paradox.Scalar:
		return t.Kind == paradox.ScalarStr && t.Str == ""
	}
	return false
}

func AccumulateRawExtra(rawExtras *RawExtras, key string, piece any) {
	existing, present := rawExtras.Get(key)
	if !present || existing == nil || isEmptyStrValue(existing) {
		rawExtras.set(key, piece)
	} else if lst, ok := existing.([]any); ok {
		rawExtras.set(key, append(lst, piece))
	} else {
		rawExtras.set(key, []any{existing, piece})
	}
}

func ScalarToText(ent *paradox.Entry) string {
	return scalarText(ent.Scalar, ent.Quoted)
}
