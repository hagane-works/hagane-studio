package guirender

import (
	"testing"
)

func TestBookmarksMergeByFileAcrossLayers(t *testing.T) {
	bookmark := func(name, date string) []byte {
		return []byte(`bookmarks = { bookmark = { name = "` + name + `" date = ` + date + ` default_country = "GER" } }`)
	}
	base, err := NewIndex(map[string][]byte{
		"common/bookmarks/the_gathering_storm.txt": bookmark("GATHERING_STORM_NAME", "1936.1.1.12"),
		"common/bookmarks/blitzkrieg.txt":          bookmark("BLITZKRIEG_NAME", "1939.8.14.12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"common/bookmarks/blitzkrieg.txt": bookmark("BLITZKRIEG_TEST_NAME", "1939.8.14.12"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	marks := project.bookmarkList()
	if len(marks) != 2 || marks[0].Name != "GATHERING_STORM_NAME" || marks[1].Name != "BLITZKRIEG_TEST_NAME" {
		t.Fatalf("bookmarks = %+v", marks)
	}

	extra, err := NewIndexWithFallback(map[string][]byte{
		"common/bookmarks/my_start.txt": bookmark("MY_START_NAME", "1937.1.1.12"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(extra.bookmarkList()); got != 3 {
		t.Errorf("with a new file: %d scenarios, want 3", got)
	}
}
