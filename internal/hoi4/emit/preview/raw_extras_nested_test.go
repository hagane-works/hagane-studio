package preview

import (
	"strings"
	"testing"
)

func TestIdeaModifierWithNestedObjectIsABlock(t *testing.T) {
	body := `{"ideas": [{"id": 1, "idea_id": "tsw_shiyan_supporter", "idea_type": "country",
		"modifier": {"stability_factor": 0.05, "trade_cost_factor": {"SYZ": -0.1}, "empty_block": {}}}],
		"picture_refs": null}`
	out, err := EmitIdeasJSON([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\n\t\t\t\tstability_factor = 0.05\n",
		"\n\t\t\t\ttrade_cost_factor = {\n\t\t\t\t\tSYZ = -0.1\n\t\t\t\t}\n",
		"\n\t\t\t\tempty_block = { }\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("ideas script is missing %q:\n%s", want, out)
		}
	}
}
