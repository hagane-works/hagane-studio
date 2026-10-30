package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

const (
	coverFilename  = "thumbnail.png"
	coverAssetType = "project_cover"
)

type coverPlan struct {
	Filename string
	Bytes    []byte
	Warning  string
}

func resolveProjectCover(snapshot Snapshot, requested string, resolve BlobResolver, msg exportmsg.Printer) (coverPlan, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return coverPlan{}, nil
	}
	if !strings.EqualFold(requested, coverFilename) {
		return coverPlan{Warning: msg.Text("cover.only_thumbnail", exportmsg.Args{"file": coverFilename})}, nil
	}
	assets, err := projectOBSAssets(snapshot)
	if err != nil {
		return coverPlan{}, err
	}
	key := strings.TrimSpace(assets[coverAssetType+"\x00"+coverFilename])
	if key == "" || resolve == nil {
		return coverPlan{Warning: msg.Text("cover.unreadable", exportmsg.Args{"file": coverFilename})}, nil
	}

	data, ok, err := ResolveBlobTolerant(resolve, key, 32<<20, "project cover "+coverFilename)
	if err != nil {
		return coverPlan{}, err
	}
	if !ok || len(data) == 0 {
		return coverPlan{Warning: msg.Text("cover.unreadable", exportmsg.Args{"file": coverFilename})}, nil
	}
	return coverPlan{Filename: coverFilename, Bytes: data}, nil
}

func coverWarningEntry(modDir, warning string) Entry {
	if modDir == "" || warning == "" {
		return Entry{}
	}
	return Entry{
		Path:         modDir + "_PROJECT_COVER_WARNING.log",
		Bytes:        []byte(warning + "\n"),
		Origin:       "generated",
		SourceEntity: "project_cover",
	}
}
