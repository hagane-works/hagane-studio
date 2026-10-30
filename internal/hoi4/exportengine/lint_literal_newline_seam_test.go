package exportengine

import (
	"os"
	"strings"
	"testing"
)

func TestLiteralNewlineLintReachesPublishedNativeArtifact(t *testing.T) {
	source, err := os.ReadFile("native.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{"ScanLiteralNewlines(artifact.Entries, opts.msg)", "LiteralNewlineLogEntry(artifactRoot(artifact)", "literal_newline_lint"} {
		if !strings.Contains(text, required) {
			t.Fatalf("native artifact seam missing %q", required)
		}
	}
}
