package exportmerge

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportengine"
)

func TestPreserveBookmarkNavigationPlaceholderOnPlatformWinsAppend(t *testing.T) {
	base := []byte("bookmarks = {\n\tbookmark = {\n\t\tname = OLD\n\t\tdate = 1964.5.1.12\n\t}\n\tbookmark = {\n\t\tdate = 1964.5.1.12\n\t}\n}\n")
	platform := []byte("bookmarks = {\n\tbookmark = {\n\t\tname = CURRENT\n\t\tdate = 1964.5.1.12\n\t}\n}\n")

	merged, preserved := preserveBookmarkNavigationPlaceholder(base, platform)
	if !preserved {
		t.Fatal("date-only navigation placeholder was not preserved")
	}
	text := string(merged)
	if got := len(bookmarkBlockStartRE.FindAll(merged, -1)); got != 2 {
		t.Fatalf("bookmark blocks=%d, want 2:\n%s", got, text)
	}
	if !strings.Contains(text, "name = CURRENT") || strings.Contains(text, "name = OLD") {
		t.Fatalf("generated content did not remain authoritative:\n%s", text)
	}
	if got := strings.Count(text, "date = 1964.5.1.12"); got != 2 {
		t.Fatalf("date count=%d, want 2:\n%s", got, text)
	}
}

func TestPreserveBookmarkNavigationPlaceholderDoesNotMergeAuthoredBookmark(t *testing.T) {
	base := []byte("bookmarks = {\n\tbookmark = { name = FIRST date = 1936.1.1.12 }\n\tbookmark = { name = SECOND date = 1939.8.14.12 }\n}\n")
	platform := []byte("bookmarks = {\n\tbookmark = { name = CURRENT date = 1964.5.1.12 }\n}\n")

	merged, preserved := preserveBookmarkNavigationPlaceholder(base, platform)
	if preserved || string(merged) != string(platform) {
		t.Fatalf("authored Bookmark must not be merged:\n%s", merged)
	}
}

func TestPreserveBookmarkNavigationPlaceholderLeavesMultiBookmarkPlatformAlone(t *testing.T) {
	base := []byte("bookmarks = {\n\tbookmark = { date = 1964.5.1.12 }\n}\n")
	platform := []byte("bookmarks = {\n\tbookmark = {\n\t\tdate = 1964.5.1.12\n\t}\n\tbookmark = {\n\t\tdate = 1972.1.1.12\n\t}\n}\n")

	merged, preserved := preserveBookmarkNavigationPlaceholder(base, platform)
	if preserved || string(merged) != string(platform) {
		t.Fatalf("multi-Bookmark generated output must remain byte-identical:\n%s", merged)
	}
}

func TestMergeAppendPreservesHiddenNavigationPlaceholderAtRealCollisionSeam(t *testing.T) {
	baseFile := writeZIP(t, map[string][]byte{
		"Base/descriptor.mod":          []byte("name=Base\n"),
		"Base/common/bookmarks/hj.txt": []byte("bookmarks = {\n\tbookmark = {\n\t\tname = OLD\n\t\tdate = 1964.5.1.12\n\t}\n\tbookmark = { date = 1964.5.1.12 }\n}\n"),
	})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "appended.zip")
	manifest, conflicts, err := MergeAppend(context.Background(), base, []exportengine.Entry{
		{Path: "Generated/descriptor.mod", Bytes: []byte("name=Generated\n")},
		{Path: "Generated/common/bookmarks/hj.txt", Bytes: []byte("bookmarks = {\n\tbookmark = {\n\t\tname = CURRENT\n\t\tdate = 1964.5.1.12\n\t}\n}\n")},
	}, destination, "bookmark-navigation")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0] != "Base/common/bookmarks/hj.txt" {
		t.Fatalf("conflicts=%v", conflicts)
	}
	archive, err := ReadArchive(context.Background(), destination, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	text := string(archive.ByPath["Base/common/bookmarks/hj.txt"])
	if got := len(bookmarkBlockStartRE.FindAllString(text, -1)); got != 2 {
		t.Fatalf("final bookmark blocks=%d, want 2:\n%s", got, text)
	}
	for _, entry := range manifest.Entries {
		if entry.Path == "Base/common/bookmarks/hj.txt" && entry.Origin != "platform_bookmark_navigation_compat" {
			t.Fatalf("bookmark manifest origin=%q", entry.Origin)
		}
	}
}
