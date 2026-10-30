package exportengine

import (
	"bytes"
	"path"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportmsg"
)

var (
	english = exportmsg.For("en")
	chinese = exportmsg.For("zh")
)

func TestPlayersLanguageChangesOnlyTheReports(t *testing.T) {
	build := func(msg exportmsg.Printer) NativeArtifact {
		artifact, err := buildNativeShadowWithBlobs(defaultLintSnapshot(), nil, false, nil, nativeBuildOptions{msg: msg})
		if err != nil {
			t.Fatal(err)
		}
		return artifact
	}
	en, zh := build(english), build(chinese)
	report := func(name string) bool {
		return strings.HasSuffix(name, ".log") || path.Base(name) == ExportFindingsBasename
	}
	byPath := map[string][]byte{}
	for _, entry := range en.Entries {
		byPath[entry.Path] = entry.Bytes
	}
	if len(en.Entries) != len(zh.Entries) {
		t.Fatalf("%d entries in English, %d in Chinese", len(en.Entries), len(zh.Entries))
	}
	reportsDiffer := false
	for _, entry := range zh.Entries {
		enBytes, ok := byPath[entry.Path]
		switch {
		case !ok:
			t.Fatalf("%s only exists in the Chinese export", entry.Path)
		case report(entry.Path):
			reportsDiffer = reportsDiffer || !bytes.Equal(enBytes, entry.Bytes)
		case !bytes.Equal(enBytes, entry.Bytes):
			t.Errorf("%s depends on the interface language", entry.Path)
		}
	}
	if !reportsDiffer {
		t.Fatal("the reports did not follow the language")
	}
}
