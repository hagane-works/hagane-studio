package exportcheck

import (
	"context"
	"fmt"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/techsurface"
	"strings"
)

type ExportIssuesOptions struct {
	IncludeImported bool
	ReplacePaths    []string
}

type exportIssuesBundle struct {
	project      Project
	characters   []Character
	trees        []FocusTree
	techs        []Technology
	ideas        []Idea
	events       []Event
	decisions    []Decision
	states       []StateRow
	historyFiles []HistoryFile
	bookmarks    []Bookmark
	countries    []Country
	rawFiles     []gateRawFile
	subUnits     []SubUnit
	divisions    []DivisionTemplate
	doctrines    []Doctrine
	airWings     []AirWingGroup
	airWingWings []AirWing
	policies     []focusPolicyRow
}

func projectTechSurfaceRows(rows []Technology) []techsurface.Technology {
	out := make([]techsurface.Technology, 0, len(rows))
	for _, technology := range rows {
		placements := make([]techsurface.FolderPlacement, 0, len(technology.FolderPlacements))
		for _, placement := range technology.FolderPlacements {
			folder, _ := placement["folder"].(string)
			placements = append(placements, techsurface.FolderPlacement{
				Folder: folder, X: folderPlacementFloat(placement["x"]), Y: folderPlacementFloat(placement["y"]),
			})
		}
		targets := []string{}
		for _, edge := range technology.Path {
			if target, ok := edge["leads_to_tech"].(string); ok && strings.TrimSpace(target) != "" {
				targets = append(targets, strings.TrimSpace(target))
			}
		}
		out = append(out, techsurface.Technology{
			ID: technology.TechID, Source: technology.Source, Folder: technology.Folder, StartYear: technology.StartYear,
			FolderPlacements: placements, PathTargets: targets,
		})
	}
	return out
}

func techSurfaceExportIssues(project Project, techs []Technology) []map[string]any {
	manifest, err := techsurface.Parse(project.TechSurfaceManifest)
	if err != nil {
		return []map[string]any{techSurfaceIssue("tech_surface_manifest_invalid", err.Error(), "", "")}
	}
	findings := techsurface.Validate(manifest, projectTechSurfaceRows(techs), len(project.Dependencies) > 0)
	issues := make([]map[string]any, 0, len(findings))
	for _, item := range findings {
		issues = append(issues, techSurfaceIssue(item.Code, item.Message, item.Branch, item.TechID))
	}
	return issues
}

func techSurfaceIssue(code, message, branch, techID string) map[string]any {
	ref := map[string]any{"type": "project", "id": "tech_surface"}
	if techID != "" {
		ref = map[string]any{"type": "technology", "id": techID}
	}
	detail := map[string]any{}
	if branch != "" {
		detail["branch_id"] = branch
	}
	if techID != "" {
		detail["tech_id"] = techID
	}
	return map[string]any{
		"level": "error", "code": code, "entity_ref": ref, "detail": detail,
		"severity": "blocker", "fix_hint": "Open Technology surface, choose a certified profile slot, and repair the root/path mapping.",
		"message": fmt.Sprintf("Technology surface: %s", message),
	}
}

func airWingLocationIssues(groups []AirWingGroup, wings []AirWing) []map[string]any {
	byGroup := map[int64][]AirWing{}
	for _, w := range wings {
		byGroup[w.GroupID] = append(byGroup[w.GroupID], w)
	}
	out := []map[string]any{}
	for _, group := range groups {
		location := strings.TrimSpace(group.Location)
		if _, err := importsvc.ValidateHOI4ScalarToken(location, false); err != nil {
			missing := location == ""
			code, hint, adjective := "oob_air_location_invalid",
				"Replace the spawn state/location with one valid HOI4 scalar token.",
				"an invalid"
			if missing {
				code = "oob_air_location_missing"
				hint = "Set the state/location where this air wing group spawns."
				adjective = "no"
			}
			out = append(out, map[string]any{
				"level": "error", "code": code,
				"entity_ref": map[string]any{"type": "air_wing_group", "id": group.ID},
				"severity":   "blocker", "fix_hint": hint,
				"message": sprintfInt("Air wing group #%d has ", group.ID) + adjective +
					" spawn state/location. HOI4 requires every air_wings group to be keyed by its spawn state, " +
					"so this OOB cannot be exported until the location is set.",
			})
		}
		for _, wing := range byGroup[group.ID] {
			equipment := strings.TrimSpace(wing.EquipmentType)
			if _, err := importsvc.ValidateHOI4ScalarToken(equipment, false); err == nil {
				continue
			}
			missing := equipment == ""
			code, hint, adjective := "oob_air_equipment_type_invalid",
				"Replace the aircraft equipment type with one valid HOI4 scalar token.",
				"an invalid"
			if missing {
				code = "oob_air_equipment_type_missing"
				hint = "Set the aircraft equipment type for this air wing."
				adjective = "no"
			}
			out = append(out, map[string]any{
				"level": "error", "code": code,
				"entity_ref": map[string]any{"type": "air_wing", "id": wing.ID},
				"severity":   "blocker", "fix_hint": hint,
				"message": sprintfInt("Air wing #%d has ", wing.ID) + adjective +
					" equipment type. HOI4 requires each air wing to name the aircraft equipment it uses, " +
					"so this OOB cannot be exported until the type is set.",
			})
		}
	}
	return out
}

func descriptorImportedPathIssue(projectID int64, paths []string) map[string]any {
	return map[string]any{
		"level":      "error",
		"code":       "descriptor.imported_replace_paths_require_full_export",
		"entity_ref": map[string]any{"type": "project", "id": projectID},
		"severity":   "blocker",
		"source": map[string]any{
			"kind": "descriptor", "identifier": "descriptor.mod",
		},
		"evidence": map[string]any{
			"replace_paths": paths, "include_imported": false,
		},
		"repair_route": "export.source_mode.full",
		"message": "This loaded mod's descriptor contains directory-level " +
			"replace_path rules, but the requested export excludes its " +
			"imported files. Emitting those rules in a delta ZIP could " +
			"unload the base directory while shipping none of its content.",
		"fix_hint": "Choose Full loaded-mod export so the imported files travel " +
			"with their replace_path rules, or explicitly clear the " +
			"imported descriptor metadata before building a delta.",
	}
}

func (s *Store) CollectExportIssues(
	ctx context.Context, projectID int64, opts ExportIssuesOptions,
) ([]ExportIssue, bool, error) {
	bundle, found, err := s.loadExportIssuesBundle(ctx, projectID)
	if err != nil || !found {
		return nil, found, err
	}
	raw, err := s.assembleExportIssues(ctx, bundle, opts)
	if err != nil {
		return nil, true, err
	}
	return NormalizeExportIssues(raw), true, nil
}
