package emitmisc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

func newDecoder(body []byte) *json.Decoder {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	return dec
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case string:
		return t, nil
	case json.Number:
		return jsonNumber(t), nil
	case json.Delim:
		switch t {
		case '{':
			od := NewOD()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, errors.New("object key is not a string")
				}
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				od.Set(key, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return od, nil
		case '[':
			out := []any{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				out = append(out, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

func jsonNumber(n json.Number) any {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if z, ok := new(big.Int).SetString(s, 10); ok {
			return z
		}
	}
	f, err := n.Float64()
	if err != nil {
		return float64(0)
	}
	return f
}

func decodeRequest(body []byte) (*OD, error) {
	v, err := decodeValue(newDecoder(body))
	if err != nil {
		return nil, err
	}
	od, ok := v.(*OD)
	if !ok {
		return nil, errors.New("request body is not a JSON object")
	}
	return od, nil
}

func nsFrom(v any) *NS {
	n := NewNS()
	od, ok := v.(*OD)
	if !ok {
		return n
	}
	for _, k := range od.Keys() {
		val, _ := od.Get(k)
		n.Set(k, val)
	}
	return n
}

func nsListFrom(v any) []*NS {
	lst, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]*NS, 0, len(lst))
	for _, item := range lst {
		out = append(out, nsFrom(item))
	}
	return out
}

func DecodeEntityJSON(body []byte) (*NS, error) {
	v, ok := fastDecodeValue(body)
	if !ok {
		var err error
		if v, err = decodeValue(newDecoder(body)); err != nil {
			return nil, err
		}
	}
	if _, ok := v.(*OD); !ok {
		return nil, errors.New("entity body is not a JSON object")
	}
	return nsFrom(v), nil
}

func nsListAny(v any) []any {
	entities := nsListFrom(v)
	out := make([]any, 0, len(entities))
	for _, e := range entities {
		out = append(out, e)
	}
	return out
}

func strSetFrom(od *OD, key string) map[string]bool {
	v, ok := od.Get(key)
	if !ok || v == nil {
		return nil
	}
	lst, ok := v.([]any)
	if !ok {
		return nil
	}
	set := make(map[string]bool, len(lst))
	for _, item := range lst {
		set[PyStr(item)] = true
	}
	return set
}

func intBoolMapFrom(od *OD, key string) map[int64]bool {
	v, ok := od.Get(key)
	if !ok || v == nil {
		return nil
	}
	inner, ok := v.(*OD)
	if !ok {
		return nil
	}
	out := make(map[int64]bool, inner.Len())
	for _, k := range inner.Keys() {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			continue
		}
		val, _ := inner.Get(k)
		b, _ := val.(bool)
		out[id] = b
	}
	return out
}

func optNS(od *OD, key string) *NS {
	v, ok := od.Get(key)
	if !ok || v == nil {
		return nil
	}
	return nsFrom(v)
}

func optNSList(od *OD, key string) []*NS {
	v, ok := od.Get(key)
	if !ok || v == nil {
		return nil
	}
	return nsListFrom(v)
}

func odStr(od *OD, key string) string {
	v, ok := od.Get(key)
	if !ok || v == nil {
		return ""
	}
	if s, isStr := v.(string); isStr {
		return s
	}
	return PyStr(v)
}

func GenerateFocusNodeScript(node *NS) string {
	lines := generateFocusNodeScript(node, "focus", nil)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(out, "\n")
}

func GenerateSingleTechScript(tech *NS, allTechs []*NS, project *NS) string {
	lines := generateTechScript(tech, allTechs, resolveYearRows(project))
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(out, "\n")
}

func EmitFocusTreeJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	treeV, _ := req.Get("tree")
	tree := nsFrom(treeV)
	nodesV, _ := req.Get("nodes")
	tree.Set("nodes", nsListAny(nodesV))
	return GenerateFocusTreeScript(tree, strSetFrom(req, "valid_icons")), nil
}

func EmitFocusNodeJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	nodeV, _ := req.Get("node")
	return GenerateFocusNodeScript(nsFrom(nodeV)), nil
}

func EmitTechLineJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	techsV, _ := req.Get("techs")
	return GenerateTechLineScript(
		nsListFrom(techsV), odStr(req, "tech_line"),
		optNS(req, "project"), optNSList(req, "siblings"),
	), nil
}

func EmitSingleTechJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	techV, _ := req.Get("tech")
	allV, _ := req.Get("all_techs")
	return GenerateSingleTechScript(
		nsFrom(techV), nsListFrom(allV), optNS(req, "project")), nil
}

func EmitDecisionCategoriesJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	catsV, _ := req.Get("categories")
	return GenerateDecisionCategoriesScript(
		nsListFrom(catsV), intBoolMapFrom(req, "icon_gates")), nil
}

func EmitDecisionsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	catsV, _ := req.Get("categories")
	decsV, _ := req.Get("decisions")
	return GenerateDecisionsPreview(
		nsListFrom(catsV), nsListFrom(decsV),
		intBoolMapFrom(req, "icon_gates")), nil
}
