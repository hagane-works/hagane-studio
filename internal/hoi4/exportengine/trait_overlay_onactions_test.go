package exportengine

import (
	"strings"
	"testing"
)

func TestNativeOnActionsUsesEmittedTraitOverlayNamespace(t *testing.T) {
	entries, err := NativeOnActionsWithTraitOverlays(
		onActionsSnapshot(), "Demo", false,
		map[string]string{"GER": "ger_trait_overlay_2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var body string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Path, "/common/on_actions/GER_on_actions.txt") {
			body = string(entry.Bytes)
			break
		}
	}
	if body == "" {
		t.Fatal("GER on_actions file was not emitted")
	}
	if !strings.Contains(body, "country_event = ger_trait_overlay_2.1") {
		t.Fatalf("on_actions did not consume the emitted overlay namespace:\n%s", body)
	}
}
