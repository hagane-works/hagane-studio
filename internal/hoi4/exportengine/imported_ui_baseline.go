package exportengine

import (
	"path"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

const importedUIBaselineSource = "gui_source_baseline"

func IsImportedUIBaselinePath(name string) bool {
	if _, err := SafePath(name); err != nil {
		return false
	}
	lower := strings.ToLower(name)
	ext := strings.ToLower(path.Ext(lower))
	switch {
	case lower == "common/alerts.txt":
		return true
	case strings.HasPrefix(lower, "common/scripted_guis/"):
		return ext == ".txt"
	case strings.HasPrefix(lower, "interface/"):
		return ext == ".gui" || ext == ".gfx"
	case strings.HasPrefix(lower, "gfx/"):
		switch ext {
		case ".dds", ".png", ".tga", ".fnt", ".ttf", ".otf", ".asset", ".gfx":
			return true
		}
	}
	return false
}

func nativeImportedUIBaseline(
	snapshot Snapshot,
	modDir string,
	includeImported bool,
	resolve BlobResolver,
	sink *imageSink,
) []Entry {
	if !includeImported {
		return nil
	}
	assets, err := projectOBSAssets(snapshot)
	if err != nil {
		sink.say("ui_baseline.skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "ui_baseline.reason.metadata"}})
		return nil
	}
	snapshotKey := assets["gui_source\x00snapshot.zip"]
	if snapshotKey == "" {
		return nil
	}
	if resolve == nil {
		sink.say("ui_baseline.skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.no_access"}})
		return nil
	}
	rawZip, err := resolve(snapshotKey, guiSnapshotLimit)
	if err != nil {
		sink.say("ui_baseline.skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.snapshot_unread"}})
		return nil
	}

	if isManifestOnlyBlob(rawZip) {
		return nil
	}
	files, err := readSafeZip(rawZip, guiSnapshotLimit)
	if err != nil {
		sink.say("ui_baseline.skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.snapshot_broken"}})
		return nil
	}
	names := make([]string, 0, len(files))
	spelling := map[string]string{}
	for name := range files {
		if IsImportedUIBaselinePath(name) {
			folded := strings.ToLower(name)
			if previous, exists := spelling[folded]; exists && previous != name {
				sink.say("ui_baseline.skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "ui_baseline.reason.case_collision"}})
				return nil
			}
			spelling[folded] = name
			names = append(names, name)
		}
	}
	sort.Strings(names)
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		out = append(out, Entry{
			Path:         modDir + "/" + name,
			Bytes:        files[name],
			Origin:       "imported",
			SourceEntity: importedUIBaselineSource,
		})
	}
	return out
}
