package exportengine

import (
	"encoding/json"
	"fmt"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/techsurface"
)

const techSurfaceSnapshotKey = "gui_source\x00snapshot.zip"

func nativeTechnologySurface(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, fmt.Errorf("decode technology surface project context: %w", err)
	}
	manifestRaw, ok := odToMap(project.A("tech_surface_manifest"))
	if !ok || len(manifestRaw) == 0 {
		return nil, nil
	}
	manifest, err := techsurface.Parse(manifestRaw)
	if err != nil {
		return nil, err
	}
	techs, err := snapshotTechnologySurfaceRows(snapshot, includeImported)
	if err != nil {
		return nil, err
	}
	if findings := techsurface.Validate(manifest, techs, len(project.List("dependencies")) > 0); len(findings) > 0 {
		return nil, techsurface.FindingsError{Findings: findings}
	}
	assets, err := projectOBSAssets(snapshot)
	if err != nil {
		return nil, err
	}
	snapshotKey := assets[techSurfaceSnapshotKey]
	if snapshotKey == "" {
		return nil, fmt.Errorf("technology surface requires an imported %s baseline snapshot", techsurface.BaselinePath)
	}
	if resolve == nil {
		return nil, fmt.Errorf("technology surface requires private GUI snapshot access")
	}
	rawZip, err := resolve(snapshotKey, guiSnapshotLimit)
	if err != nil {
		return nil, fmt.Errorf("read technology surface GUI snapshot: %w", err)
	}
	if isManifestOnlyBlob(rawZip) {
		return nil, fmt.Errorf("technology surface export needs the GUI source bytes, not a manifest-only snapshot")
	}
	sources, err := readSafeZip(rawZip, guiSnapshotLimit)
	if err != nil {
		return nil, fmt.Errorf("read technology surface GUI source archive: %w", err)
	}
	base, exists := sources[techsurface.BaselinePath]
	if !exists {
		return nil, fmt.Errorf("technology surface snapshot does not contain %s", techsurface.BaselinePath)
	}
	if err := techSurfacePathIsUnclaimed(snapshot, includeImported); err != nil {
		return nil, err
	}
	patched, err := techsurface.Compile(base, manifest, techs, len(project.List("dependencies")) > 0)
	if err != nil {
		return nil, err
	}
	return []Entry{{
		Path: modDir + "/" + techsurface.BaselinePath, Bytes: patched,
		Origin: "generated", SourceEntity: "technology_surface",
	}}, nil
}

func snapshotTechnologySurfaceRows(snapshot Snapshot, includeImported bool) ([]techsurface.Technology, error) {
	rows := make([]techsurface.Technology, 0, len(snapshot.Tables["technologies"]))
	for _, raw := range snapshot.Tables["technologies"] {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("decode technology surface row: %w", err)
		}
		source := entity.Str("source")
		if !includeImported && source != "" && source != "created" {
			continue
		}
		placements := []techsurface.FolderPlacement{}
		for _, rawPlacement := range entity.List("folder_placements") {
			placement, ok := odToMap(rawPlacement)
			if !ok {
				continue
			}
			placements = append(placements, techsurface.FolderPlacement{
				Folder: stringFrom(placement["folder"]), X: toFloat64(placement["x"]), Y: toFloat64(placement["y"]),
			})
		}
		targets := []string{}
		for _, rawPath := range entity.List("path") {
			path, ok := odToMap(rawPath)
			if !ok {
				continue
			}
			if target := strings.TrimSpace(stringFrom(path["leads_to_tech"])); target != "" {
				targets = append(targets, target)
			}
		}
		rows = append(rows, techsurface.Technology{
			ID: entity.Str("tech_id"), Source: source, Folder: entity.Str("folder"), StartYear: int(entity.Int("start_year", 1936)),
			FolderPlacements: placements, PathTargets: targets,
		})
	}
	return rows, nil
}

func toFloat64(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	}
	return 0
}

func stringFrom(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func odToMap(value any) (map[string]any, bool) {
	plain := plainJSONValue(value)
	if result, ok := plain.(map[string]any); ok && len(result) > 0 {
		return result, true
	}
	if plain == nil {
		return nil, false
	}

	b, err := json.Marshal(plain)
	if err != nil {
		return nil, false
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil || len(result) == 0 {
		return nil, false
	}
	return result, true
}

func plainJSONValue(value any) any {
	switch value := value.(type) {
	case *emitmisc.OD:
		result := make(map[string]any)
		for key, child := range value.MapCopy() {
			result[key] = plainJSONValue(child)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, child := range value {
			result[key] = plainJSONValue(child)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for index, child := range value {
			result[index] = plainJSONValue(child)
		}
		return result
	default:
		return value
	}
}

func techSurfacePathIsUnclaimed(snapshot Snapshot, includeImported bool) error {
	const target = techsurface.BaselinePath
	for _, raw := range snapshot.Tables["project_raw_files"] {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return err
		}
		if !includeImported && entity.Str("source") == "imported" {
			continue
		}
		if strings.EqualFold(strings.Trim(strings.ReplaceAll(entity.Str("dest_path"), "\\", "/"), "/"), target) {
			return fmt.Errorf("technology surface cannot overwrite authored raw file %s", target)
		}
	}
	for _, raw := range snapshot.Tables["gui_overlays"] {
		var row struct {
			File string `json:"file"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return err
		}
		clean, err := safeGUIPath(row.File)
		if err != nil {
			return err
		}
		if strings.EqualFold(clean, target) {
			return fmt.Errorf("technology surface cannot compose with a generic GUI overlay for %s yet", target)
		}
	}
	return nil
}
