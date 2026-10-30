package history

import (
	"os"
	"strings"
	"testing"
)

func TestCentralSeamNativeBuilderConsumesScenarioHistoryPlan(t *testing.T) {
	content, err := os.ReadFile("../../exportengine/native.go")
	if err != nil {
		t.Fatalf("read native builder seam: %v", err)
	}
	if !strings.Contains(string(content), "ScenarioHistoryPlan") || !strings.Contains(string(content), "BuildNativeShadowWithPlan") {
		t.Fatal("central seam missing: native.go must expose BuildNativeShadowWithPlan and consume ScenarioHistoryPlan")
	}
}
