package lint

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestRawBookmarkHashMismatch(t *testing.T) {
	content := "bookmark = {\n name = \"x\"\n date = 1936.1.1.12\n}\n"
	sum := sha256.Sum256([]byte(content))
	actual := hex.EncodeToString(sum[:])

	codes := func(declared string) []string {
		plan := BuildTimelineExportPlan("", nil, []PreservedRawBookmarkInput{{
			Path: "common/bookmarks/a.txt", Content: content,
			Source: "project_preserved_raw", SourceHash: declared, Emitted: true,
		}})
		out := []string{}
		for _, f := range plan.Findings {
			out = append(out, f.Code)
		}
		return out
	}

	wrong := codes("0000000000000000000000000000000000000000000000000000000000000000")
	if len(wrong) != 2 || wrong[0] != "timeline.raw_bookmark_hash_mismatch" ||
		wrong[1] != "timeline.raw_bookmark_date_unknown" {
		t.Fatalf("声明指纹错误时应先报 hash_mismatch 再报 date_unknown,得到 %v", wrong)
	}
	for _, tc := range []struct {
		name, declared string
	}{
		{"声明正确", actual},
		{"未声明", ""},
		{"大小写不敏感", "AE8771C7368264DBC7FB32057B37AFF88ACFE99C1E3C5CFC6F8B9E413954D739"},
	} {
		got := codes(tc.declared)
		for _, code := range got {
			if code == "timeline.raw_bookmark_hash_mismatch" {
				t.Errorf("%s 不该报 hash_mismatch,得到 %v", tc.name, got)
			}
		}
	}

	plan := BuildTimelineExportPlan("", nil, []PreservedRawBookmarkInput{{
		Path: "common/bookmarks/a.txt", Content: content,
		Source: "project_preserved_raw", SourceHash: "deadbeef", Emitted: true,
	}})
	for _, f := range plan.Findings {
		if f.Code != "timeline.raw_bookmark_hash_mismatch" {
			continue
		}
		if f.Evidence["actual_hash"] != actual {
			t.Errorf("actual_hash = %v want %s", f.Evidence["actual_hash"], actual)
		}
		if f.Evidence["declared_hash"] != "deadbeef" {
			t.Errorf("declared_hash = %v want deadbeef", f.Evidence["declared_hash"])
		}
	}
}
