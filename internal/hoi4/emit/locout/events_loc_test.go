package locout

import (
	"strings"
	"testing"
)

func locObj(pairs ...any) *Obj {
	o := &Obj{M: map[string]Value{}}
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i].(string)
		o.Keys = append(o.Keys, key)
		switch v := pairs[i+1].(type) {
		case string:
			o.M[key] = Str(v)
		case Value:
			o.M[key] = v
		}
	}
	return o
}

func TestEventOptionWithoutTextWritesNoLine(t *testing.T) {
	options := Value{Kind: KArr, Arr: []Value{
		{Kind: KObj, Obj: locObj("name", "germany.107.a")},
		{Kind: KObj, Obj: locObj("name", "germany.107.b", "name_en", "Very well")},
		{Kind: KObj, Obj: locObj()},
	}}
	event := locObj("event_id", "germany.107", "options", options)
	out := GenerateEventsLocalisation(nil, []*Obj{event}, "en")
	if strings.Contains(out, "germany.107.a:") {
		t.Fatalf("an option without text wrote a line:\n%s", out)
	}
	if !strings.Contains(out, `germany.107.b: "Very well"`) {
		t.Fatalf("an option with text lost its line:\n%s", out)
	}
	if strings.Contains(out, "germany.107.c:") {
		t.Fatalf("a nameless option without text wrote a line:\n%s", out)
	}
}
