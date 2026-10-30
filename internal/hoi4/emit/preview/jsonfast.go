package preview

import (
	"encoding/json"
)

type requestBuilder struct{}

func (requestBuilder) Null() any           { return nil }
func (requestBuilder) Bool(v bool) any     { return v }
func (requestBuilder) String(s string) any { return s }
func (requestBuilder) Number(literal string) (any, bool) {
	return jsonNumber(json.Number(literal)), true
}
func (requestBuilder) Array(items []any) any         { return items }
func (requestBuilder) NewObject() *OD                { return NewOD() }
func (requestBuilder) Set(od *OD, key string, v any) { odOrdered(od, key, v) }
func (requestBuilder) Object(od *OD) any             { return od }
