package timeline

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	goBlockCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	goLineCommentRE  = regexp.MustCompile(`(?m)//.*$`)
)

func TestExportEngineConsumesTimelinePlan(t *testing.T) {
	nativePath, nativeSource := centralSource(t, "exportengine", "native.go")
	const requiredBuilder = "BuildNativeShadowWithTimelinePlan"
	if !productionSourceContains(nativeSource, requiredBuilder) {
		t.Fatalf("central integration pending: go/exportengine/native.go must declare %s and consume timeline.Plan", requiredBuilder)
	}
	if !productionSourceContains(nativeSource, "ArtifactMetadata()") {
		t.Fatalf("central integration pending: %q must attach plan.ArtifactMetadata()", nativePath)
	}
}

func centralSource(t *testing.T, elements ...string) (string, string) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate seam contract test")
	}
	parts := append([]string{filepath.Dir(thisFile), "..", ".."}, elements...)
	path := filepath.Join(parts...)
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read central seam %q: %v", path, err)
	}
	return path, string(bytes)
}

func productionSourceContains(source, marker string) bool {
	withoutComments := goBlockCommentRE.ReplaceAllString(source, "")
	withoutComments = goLineCommentRE.ReplaceAllString(withoutComments, "")
	return strings.Contains(withoutComments, marker)
}

func TestProductionSourceContainsIgnoresComments(t *testing.T) {
	if productionSourceContains("// FingerprintFields()\n/* ExportIssuesProjection() */", "FingerprintFields()") {
		t.Fatal("comment-only marker must not satisfy a central seam assertion")
	}
	if !productionSourceContains("func use() { _ = plan.FingerprintFields() }", "FingerprintFields()") {
		t.Fatal("real source marker must satisfy a central seam assertion")
	}
}
