package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyBuildIdentityTrueFalseArtifactDifference(t *testing.T) {
	base := NativeArtifact{Entries: []Entry{{Path: "Demo.mod"}, {Path: "Demo/descriptor.mod"}}}
	without := base
	if err := ApplyBuildIdentity(&without, ""); err != nil {
		t.Fatal(err)
	}
	for _, entry := range without.Entries {
		if strings.HasSuffix(entry.Path, BuildIdentityFilename) {
			t.Fatal("disabled build identity emitted evidence")
		}
	}
	with := base
	const wantID = "hagane-build-00000000-0000-4000-8000-000000000000"
	if err := ApplyBuildIdentity(&with, wantID); err != nil {
		t.Fatal(err)
	}
	for _, entry := range with.Entries {
		if entry.Path != "Demo/"+BuildIdentityFilename {
			continue
		}
		var got map[string]string
		if err := json.Unmarshal(entry.Bytes, &got); err != nil {
			t.Fatal(err)
		}
		if got["build_id"] != wantID || got["schema_version"] != BuildIdentitySchemaVersion {
			t.Fatalf("identity payload = %#v", got)
		}
		return
	}
	t.Fatal("enabled build identity did not emit evidence")
}
