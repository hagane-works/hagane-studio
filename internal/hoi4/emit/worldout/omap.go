package worldout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type Value = any

type OMap struct {
	keys []string
	m    map[string]Value
}

func NewOMap() *OMap { return &OMap{m: map[string]Value{}} }

func (o *OMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return err
	}
	om, ok := v.(*OMap)
	if !ok {
		return fmt.Errorf("expected JSON object")
	}
	*o = *om
	return nil
}

func decodeValue(dec *json.Decoder) (Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			om := NewOMap()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string)
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				om.Set(key, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return om, nil
		case '[':
			arr := []Value{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("unexpected delim %v", t)
	default:
		return tok, nil
	}
}

func (o *OMap) Set(k string, v Value) {
	if o.m == nil {
		o.m = map[string]Value{}
	}
	if _, exists := o.m[k]; !exists {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

func (o *OMap) Get(k string) (Value, bool) {
	if o == nil || o.m == nil {
		return nil, false
	}
	v, ok := o.m[k]
	return v, ok
}

func (o *OMap) Has(k string) bool {
	_, ok := o.Get(k)
	return ok
}

func (o *OMap) Keys() []string {
	if o == nil {
		return nil
	}
	return o.keys
}

func (o *OMap) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

func (o *OMap) Clone() *OMap {
	c := NewOMap()
	for _, k := range o.Keys() {
		v, _ := o.Get(k)
		c.Set(k, v)
	}
	return c
}

func pyStr(v Value) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		panic(fmt.Sprintf("pyStr: unsupported %T", v))
	}
}

func pyTruthy(v Value) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return true
		}
		return f != 0
	case []Value:
		return len(t) > 0
	case *OMap:
		return t.Len() > 0
	default:
		return true
	}
}

func pyStrOrEmpty(v Value) string {
	if !pyTruthy(v) {
		return ""
	}
	return pyStr(v)
}

func numFloat(v Value) (float64, bool) {
	if n, ok := v.(json.Number); ok {
		f, err := n.Float64()
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func numIsFloatLiteral(n json.Number) bool {
	s := n.String()
	for i := 0; i < len(s); i++ {
		if s[i] == '.' || s[i] == 'e' || s[i] == 'E' {
			return true
		}
	}
	return false
}

func intVal(v Value) (int64, bool) {
	if n, ok := v.(json.Number); ok {
		i, err := n.Int64()
		if err == nil {
			return i, true
		}
		f, err := n.Float64()
		if err == nil {
			return int64(f), true
		}
	}
	return 0, false
}

func eGet(e *OMap, k string) Value {
	v, _ := e.Get(k)
	return v
}

func eStr(e *OMap, k string) string {
	v, ok := e.Get(k)
	if !ok || v == nil {
		return ""
	}
	return pyStr(v)
}

func eStrOrEmpty(e *OMap, k string) string {
	return pyStrOrEmpty(eGet(e, k))
}

func eList(e *OMap, k string) []Value {
	v, ok := e.Get(k)
	if !ok {
		return nil
	}
	l, _ := v.([]Value)
	return l
}

func eMap(e *OMap, k string) *OMap {
	v, ok := e.Get(k)
	if !ok {
		return nil
	}
	m, _ := v.(*OMap)
	return m
}

func eInt(e *OMap, k string) int64 {
	v, _ := e.Get(k)
	i, _ := intVal(v)
	return i
}

func eBool(e *OMap, k string) bool {
	v, _ := e.Get(k)
	b, _ := v.(bool)
	return b
}

func fmtInt(i int64) string { return strconv.FormatInt(i, 10) }
