package emitmisc

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/emit/internal/fastjson"
)

func tokenDecode(data []byte) (any, error) {
	return decodeValue(newDecoder(data))
}

func TestFastDecodeMatchesTokenDecoder(t *testing.T) {
	for _, input := range fastjson.ParityInputs {
		requireSameDecode(t, []byte(input))
	}
	deep := strings.Repeat("[", 10001) + strings.Repeat("]", 10001)
	if _, ok := fastDecodeValue([]byte(deep)); ok {
		t.Fatal("fast path accepted nesting encoding/json refuses")
	}
}

func requireSameDecode(t *testing.T, input []byte) {
	t.Helper()
	fast, ok := fastDecodeValue(input)
	want, err := tokenDecode(input)
	if !ok {
		return
	}
	if err != nil {
		t.Fatalf("fast path accepted %q, token decoder rejects it: %v", input, err)
	}
	if !reflect.DeepEqual(fast, want) {
		t.Fatalf("decode of %q differs:\nfast  %#v\ntoken %#v", input, fast, want)
	}
}

func FuzzFastDecodeMatchesTokenDecoder(f *testing.F) {
	for _, input := range fastjson.ParityInputs {
		f.Add([]byte(input))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		fast, ok := fastDecodeValue(input)
		if !ok {
			return
		}
		want, err := tokenDecode(input)
		if err != nil {

			t.Fatalf("fast path accepted %q, token decoder rejects it: %v", input, err)
		}
		if !reflect.DeepEqual(fast, want) {
			t.Fatalf("decode of %q differs", input)
		}
	})
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
			if _, ok := fastDecodeValue(row); !ok {
				t.Fatalf("fast path declined a stored row: %.200s", row)
			}
			requireSameDecode(t, row)
			rows++
		}
	}
	t.Logf("%d rows decode identically", rows)
}
