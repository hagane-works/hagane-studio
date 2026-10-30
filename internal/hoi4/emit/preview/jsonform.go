package preview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/emit/internal/fastjson"
)

func odOrdered(d *OD, k string, v any) {
	if _, exists := d.m[k]; !exists {
		d.keys = append(d.keys, k)
	}
	d.m[k] = v
}

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
				odOrdered(od, key, val)
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
		if z, ok := new(big.Int).SetString(s, 10); ok {
			return z
		}
	}
	f, ok := pyFloatParse(s)
	if !ok {
		return float64(0)
	}
	return f
}

func decodeRequest(body []byte) (*OD, error) {
	v, ok := fastjson.Parse[any, *OD](body, requestBuilder{})
	if !ok {
		var err error
		if v, err = decodeValue(newDecoder(body)); err != nil {
			return nil, err
		}
	}
	od, ok := v.(*OD)
	if !ok {
		return nil, errors.New("request body is not a JSON object")
	}
	return od, nil
}

func reqOD(req *OD, key string) *OD {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return NewOD()
	}
	if od, isOD := v.(*OD); isOD {
		return od
	}
	return NewOD()
}

func reqStr(req *OD, key string) string {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return ""
	}
	if s, isStr := v.(string); isStr {
		return s
	}
	return pyStr(v)
}

func reqBool(req *OD, key string, def bool) bool {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return def
	}
	b, isBool := v.(bool)
	if !isBool {
		return def
	}
	return b
}

func reqList(req *OD, key string) []any {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return nil
	}
	l, isList := v.([]any)
	if !isList {
		return nil
	}
	return l
}

func reqStrSet(req *OD, key string) (map[string]bool, bool) {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return nil, false
	}
	l, isList := v.([]any)
	if !isList {
		return nil, false
	}
	set := make(map[string]bool, len(l))
	for _, item := range l {
		set[pyStr(item)] = true
	}
	return set, true
}

func reqIntStrMap(req *OD, key string) map[int]string {
	v, ok := req.Get(key)
	if !ok || v == nil {
		return nil
	}
	inner, isOD := v.(*OD)
	if !isOD {
		return nil
	}
	out := make(map[int]string, inner.Len())
	for _, k := range inner.Keys() {
		id, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		val, _ := inner.Get(k)
		out[id] = pyStr(val)
	}
	return out
}

func entID(od *OD) int {
	v, ok := od.Get("id")
	if !ok {
		return 0
	}
	if z, isBig := v.(*big.Int); isBig {
		return int(z.Int64())
	}
	return 0
}

func ideaFrom(v any) *Idea {
	od, ok := v.(*OD)
	if !ok {
		od = NewOD()
	}
	return &Idea{Ent: buildEnt(ideaFields, odSrc(od)), ID: entID(od)}
}

func EmitCharacterJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	ent := buildEnt(characterFields, odSrc(reqOD(req, "char")))
	return GenerateCharacterScript(
		ent, reqStr(req, "country_tag"),
		reqBool(req, "has_portrait_image", true),
		reqBool(req, "has_advisor_image", true),
		reqStr(req, "portrait_path_override"),
		reqStr(req, "advisor_path_override"),
	), nil
}

func EmitIdeasJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	var ideas []*Idea
	for _, item := range reqList(req, "ideas") {
		ideas = append(ideas, ideaFrom(item))
	}
	return GenerateIdeasScript(ideas, reqIntStrMap(req, "picture_refs")), nil
}

func EmitSingleIdeaJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	v, _ := req.Get("idea")
	return GenerateSingleIdeaScript(ideaFrom(v)), nil
}

func EmitEventsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	var ents []*OD
	for _, item := range reqList(req, "events") {
		od, ok := item.(*OD)
		if !ok {
			od = NewOD()
		}
		ents = append(ents, buildEnt(eventFields, odSrc(od)))
	}
	valid, gate := reqStrSet(req, "valid_pictures")
	return GenerateEventsPreview(reqStr(req, "namespace"), ents, valid, gate), nil
}
