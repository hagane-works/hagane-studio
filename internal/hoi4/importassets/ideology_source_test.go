package importassets

import "testing"

func TestIdeologySourceSnapshotKeyIsDeterministicAndScoped(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	got, err := IdeologySourceSnapshotKey("p0-local/", "project-uuid", "dependency-key", digest)
	if err != nil {
		t.Fatal(err)
	}
	if want := "p0-local/p/project-uuid/ideology-source/dependency-key/" + digest + ".zip"; got != want {
		t.Fatalf("key=%q want=%q", got, want)
	}
	vanilla, err := IdeologySourceSnapshotKey("p0-local", "project-uuid", "vanilla", digest)
	if err != nil || vanilla != "p0-local/p/project-uuid/ideology-source/vanilla/"+digest+".zip" {
		t.Fatalf("vanilla=%q err=%v", vanilla, err)
	}
}

func TestIdeologySourceSnapshotKeyRejectsPathEscape(t *testing.T) {
	for _, tc := range []struct{ env, project, source string }{
		{"", "project", "vanilla"}, {"dev", "", "vanilla"},
		{"dev", "../project", "vanilla"}, {"dev", "project", "../dependency"},
		{"dev", "project", "dependency/key"}, {"dev", "project", "dependency\\key"},
	} {
		if _, err := IdeologySourceSnapshotKey(tc.env, tc.project, tc.source, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil {
			t.Fatalf("accepted unsafe scope %#v", tc)
		}
	}
	if _, err := IdeologySourceSnapshotKey("dev", "project", "vanilla", "not-a-sha"); err == nil {
		t.Fatal("accepted unsafe archive digest")
	}
}
