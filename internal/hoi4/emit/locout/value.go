package locout

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/emit/internal/fastjson"
	"haganeworks/internal/hoi4/paradox"
)

func pyFloatRepr(f float64) string {
	return paradox.Scalar{Kind: paradox.ScalarFloat, Float: f}.StrValue()
}

type Kind int

const (
	KMissing Kind = iota
	KNull
	KBool
	KInt
	KFloat
	KStr
	KArr
	KObj
)

type Value struct {
	Kind  Kind
	Bool  bool
	Int   *big.Int
	Float float64
	Str   string
	Arr   []Value
	Obj   *Obj
}

type Obj struct {
	Keys []string
	M    map[string]Value
}

func (o *Obj) Get(key string) (Value, bool) {
	if o == nil {
		return Value{Kind: KMissing}, false
	}
	v, ok := o.M[key]
	if !ok {
		return Value{Kind: KMissing}, false
	}
	return v, true
}

func (o *Obj) Attr(name string) Value {
	v, _ := o.Get(name)
	return v
}

func Str(s string) Value { return Value{Kind: KStr, Str: s} }
func Null() Value        { return Value{Kind: KNull} }
func IntV(i int64) Value { return Value{Kind: KInt, Int: big.NewInt(i)} }

func PyTruthy(v Value) bool {
	switch v.Kind {
	case KMissing, KNull:
		return false
	case KBool:
		return v.Bool
	case KInt:
		return v.Int.Sign() != 0
	case KFloat:
		return v.Float != 0
	case KStr:
		return v.Str != ""
	case KArr:
		return len(v.Arr) > 0
	case KObj:
		return len(v.Obj.Keys) > 0
	}
	return false
}

func PyStr(v Value) string {
	switch v.Kind {
	case KMissing, KNull:
		return "None"
	case KBool:
		if v.Bool {
			return "True"
		}
		return "False"
	case KInt:
		return v.Int.String()
	case KFloat:
		return pyFloatRepr(v.Float)
	case KStr:
		return v.Str
	}
	return fmt.Sprintf("<unstr kind=%d>", v.Kind)
}

func strOrEmpty(v Value) string {
	if !PyTruthy(v) {
		return ""
	}
	if v.Kind == KStr {
		return v.Str
	}
	return PyStr(v)
}

func ParseJSON(data []byte) (Value, error) {
	if v, ok := fastjson.Parse[Value, *Obj](data, valueBuilder{}); ok {
		return v, nil
	}
	return parseJSONTokens(data)
}

func parseJSONTokens(data []byte) (Value, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return Value{}, err
	}
	if dec.More() {
		return Value{}, fmt.Errorf("trailing JSON content")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return Value{}, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (Value, error) {
	switch t := tok.(type) {
	case nil:
		return Value{Kind: KNull}, nil
	case bool:
		return Value{Kind: KBool, Bool: t}, nil
	case json.Number:
		s := t.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := t.Float64()
			if err != nil {
				return Value{}, err
			}
			return Value{Kind: KFloat, Float: f}, nil
		}
		z, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return Value{}, fmt.Errorf("bad int %q", s)
		}
		return Value{Kind: KInt, Int: z}, nil
	case string:
		return Value{Kind: KStr, Str: t}, nil
	case json.Delim:
		switch t {
		case '[':
			arr := []Value{}
			for dec.More() {
				v, err := decodeValue(dec)
				if err != nil {
					return Value{}, err
				}
				arr = append(arr, v)
			}
			if _, err := dec.Token(); err != nil {
				return Value{}, err
			}
			return Value{Kind: KArr, Arr: arr}, nil
		case '{':
			obj := &Obj{M: map[string]Value{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return Value{}, err
				}
				key, ok := kt.(string)
				if !ok {
					return Value{}, fmt.Errorf("non-string object key %v", kt)
				}
				v, err := decodeValue(dec)
				if err != nil {
					return Value{}, err
				}
				if _, dup := obj.M[key]; !dup {
					obj.Keys = append(obj.Keys, key)
				}
				obj.M[key] = v
			}
			if _, err := dec.Token(); err != nil {
				return Value{}, err
			}
			return Value{Kind: KObj, Obj: obj}, nil
		}
	}
	return Value{}, fmt.Errorf("unexpected JSON token %v", tok)
}
