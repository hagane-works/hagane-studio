package locout

import (
	"encoding/json"
	"math/big"
	"strings"
)

type valueBuilder struct{}

func (valueBuilder) Null() Value           { return Value{Kind: KNull} }
func (valueBuilder) Bool(v bool) Value     { return Value{Kind: KBool, Bool: v} }
func (valueBuilder) String(s string) Value { return Value{Kind: KStr, Str: s} }
func (valueBuilder) Number(literal string) (Value, bool) {
	if strings.ContainsAny(literal, ".eE") {
		f, err := json.Number(literal).Float64()
		if err != nil {
			return Value{}, false
		}
		return Value{Kind: KFloat, Float: f}, true
	}
	z, ok := new(big.Int).SetString(literal, 10)
	if !ok {
		return Value{}, false
	}
	return Value{Kind: KInt, Int: z}, true
}
func (valueBuilder) Array(items []Value) Value { return Value{Kind: KArr, Arr: items} }
func (valueBuilder) NewObject() *Obj           { return &Obj{M: map[string]Value{}} }
func (valueBuilder) Set(obj *Obj, key string, v Value) {
	if _, dup := obj.M[key]; !dup {
		obj.Keys = append(obj.Keys, key)
	}
	obj.M[key] = v
}
func (valueBuilder) Object(obj *Obj) Value { return Value{Kind: KObj, Obj: obj} }
