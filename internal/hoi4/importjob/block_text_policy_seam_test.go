package importjob

import (
	"os"
	"strings"
	"testing"
)

func TestImportJSONBoundaryNormalizesBlockTextBeforeStaging(t *testing.T) {
	source, err := os.ReadFile("decode.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	anchor := "func marshalPGJSON(value any)"
	start := strings.Index(text, anchor)
	if start < 0 {
		t.Fatalf("missing named anchor %q", anchor)
	}
	end := strings.Index(text[start+len(anchor):], "\n}")
	if end < 0 {
		end = len(text) - start
	} else {
		end += len(anchor)
	}
	if !strings.Contains(text[start:start+end], "NormalizeImportedBlockPayload(") {
		t.Fatal("marshalPGJSON still stages parser payloads without the block-text storage policy")
	}
}
