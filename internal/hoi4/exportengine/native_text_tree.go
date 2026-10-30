package exportengine

import (
	"fmt"
	"path"
	"strings"
)

var nativeTextTreeSuffixes = map[string]bool{
	".txt": true, ".yml": true, ".gui": true, ".gfx": true, ".asset": true, ".csv": true,
	".lua": true, ".fnt": true,
}

type NativePreviewProjection struct {
	TextFiles map[string][]byte

	ShadowPaths []string

	Winners               map[string]NativePreviewWinner
	EffectiveReplacePaths []string

	StageReuse StageReuse
}

type NativePreviewWinner struct {
	Origin       string
	SourceEntity string
}

func BuildNativePreviewProjection(snapshot Snapshot, requestedReplacePaths []string, includeImported bool) (projection NativePreviewProjection, err error) {
	defer func() {
		if value := recover(); value != nil {
			projection = NativePreviewProjection{}
			err = fmt.Errorf("native preview projection rejected project data: %v", value)
		}
	}()
	return buildNativePreviewProjection(snapshot, requestedReplacePaths, includeImported)
}

func BuildNativeTextTree(snapshot Snapshot, includeImported bool) (files map[string][]byte, err error) {
	defer func() {
		if value := recover(); value != nil {
			files = nil
			err = fmt.Errorf("native text tree rejected project data: %v", value)
		}
	}()
	projection, err := buildNativePreviewProjection(snapshot, nil, includeImported)
	if err != nil {
		return nil, err
	}
	return projection.TextFiles, nil
}

func buildNativePreviewProjection(snapshot Snapshot, requestedReplacePaths []string, includeImported bool) (NativePreviewProjection, error) {

	artifact, err := buildNativeShadowWithBlobs(snapshot, requestedReplacePaths, includeImported, manifestOnlyResolver, nativeBuildOptions{skipAdvisory: true, gameFilesOnly: true})
	if err != nil {
		return NativePreviewProjection{}, err
	}
	if artifact.Timeline == nil {
		return NativePreviewProjection{}, fmt.Errorf("native preview artifact is missing timeline metadata")
	}
	files := make(map[string][]byte, len(artifact.Entries))
	winners := make(map[string]NativePreviewWinner, len(artifact.Entries))
	shadowPaths := make([]string, 0, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		name := strings.ReplaceAll(entry.Path, "\\", "/")
		slash := strings.Index(name, "/")
		if slash < 0 || isArtifactManifestPath(name) {
			continue
		}
		rel := name[slash+1:]
		if strings.EqualFold(strings.Trim(rel, "/"), "descriptor.mod") {
			continue
		}
		if isManifestOnlyGUIPath(entry) {
			continue
		}
		key := strings.ToLower(strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/"))
		winners[key] = NativePreviewWinner{Origin: entry.Origin, SourceEntity: entry.SourceEntity}
		shadowPaths = append(shadowPaths, rel)
		if !nativeTextTreeSuffixes[strings.ToLower(path.Ext(rel))] || isManifestOnlyBlob(entry.Bytes) {
			continue
		}
		files[rel] = entry.Bytes
	}
	return NativePreviewProjection{
		TextFiles:             files,
		ShadowPaths:           shadowPaths,
		Winners:               winners,
		EffectiveReplacePaths: append([]string(nil), artifact.Timeline.Fingerprint.EffectiveReplacePaths...),
		StageReuse:            artifact.stageReuse,
	}, nil
}

func isManifestOnlyGUIPath(entry Entry) bool {
	if entry.Bytes != nil {
		return false
	}
	switch entry.SourceEntity {
	case "gui_overlays", "gui_overlay_sprites", "gui_overlay_localisation":
		return true
	default:
		return false
	}
}
