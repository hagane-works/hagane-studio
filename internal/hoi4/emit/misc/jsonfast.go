package emitmisc

import (
	"encoding/json"

	"haganeworks/internal/hoi4/emit/internal/fastjson"
)

func fastDecodeValue(data []byte) (any, bool) {
	return fastjson.Parse[any, *OD](data, entityBuilder{})
}

type entityBuilder struct{}

func (entityBuilder) Null() any           { return nil }
func (entityBuilder) Bool(v bool) any     { return v }
func (entityBuilder) String(s string) any { return s }
func (entityBuilder) Number(literal string) (any, bool) {
	return jsonNumber(json.Number(literal)), true
}
func (entityBuilder) Array(items []any) any         { return items }
func (entityBuilder) NewObject() *OD                { return NewOD() }
func (entityBuilder) Set(od *OD, key string, v any) { od.Set(key, v) }
func (entityBuilder) Object(od *OD) any             { return od }
