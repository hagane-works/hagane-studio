package imagebatch

import (
	"strings"
	"testing"
)

func TestGeneratePlaceholdersStopsAtEncodedOutputBudget(t *testing.T) {
	_, err := GeneratePlaceholdersLimited(
		[]PlaceholderSpec{{EntityID: "a"}}, 32, 32, 1,
	)
	if err == nil || !strings.Contains(err.Error(), "output exceeds") {
		t.Fatalf("err=%v", err)
	}
}
