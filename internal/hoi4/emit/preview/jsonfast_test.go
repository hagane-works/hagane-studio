package preview

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/emit/internal/fastjson"
)

func requireSameDecode(t *testing.T, input []byte) {
	t.Helper()
	fast, ok := fastjson.Parse[any, *OD](input, requestBuilder{})
	if !ok {
		return
	}
	want, err := decodeValue(newDecoder(input))
	if err != nil {
		t.Fatalf("fast path accepted %q, token decoder rejects it: %v", input, err)
	}
	if !reflect.DeepEqual(fast, want) {
		t.Fatalf("decode of %q differs:\nfast  %#v\ntoken %#v", input, fast, want)
	}
}

func TestFastDecodeMatchesTokenDecoder(t *testing.T) {
	for _, input := range fastjson.ParityInputs {
		requireSameDecode(t, []byte(input))
	}
}

func FuzzFastDecodeMatchesTokenDecoder(f *testing.F) {
	for _, input := range fastjson.ParityInputs {
		f.Add([]byte(input))
	}
	f.Fuzz(requireSameDecode)
}

func TestFastDecodeOnARealProject(t *testing.T) {
	path := os.Getenv("HOI4_DECODE_PARITY_SNAPSHOT")
	if path == "" {
		t.Skip("HOI4_DECODE_PARITY_SNAPSHOT unset (skip is not a pass)")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Project json.RawMessage              `json:"project"`
		Tables  map[string][]json.RawMessage `json:"tables"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	rows := 1
	requireSameDecode(t, snapshot.Project)
	for _, table := range snapshot.Tables {
		for _, row := range table {
			input := []byte(row)
			if _, ok := fastjson.Parse[any, *OD](input, requestBuilder{}); !ok {
				t.Fatalf("fast path declined a stored row: %.200s", row)
			}
			requireSameDecode(t, input)
			rows++
		}
	}
	t.Logf("%d rows decode identically", rows)
}
