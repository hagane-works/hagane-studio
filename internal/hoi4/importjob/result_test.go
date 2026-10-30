package importjob

import "testing"

func TestLegacyResultRequiredKeys(t *testing.T) {
	r := legacyResult("s", "overwrite", []string{"gfx/x.dds"}, 3)
	for _, key := range []string{"characters", "focus_trees", "technologies", "ideas", "events", "decisions", "warnings", "needed_images", "session_id", "conflict_policy"} {
		if _, ok := r[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
}
