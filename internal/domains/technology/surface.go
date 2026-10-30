package technology

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const assetsTable = "project_obs_assets"

const (
	TechSurfaceEnvironmentBaselineMissing = "baseline_missing"
	TechSurfaceEnvironmentDLCUnconfirmed  = "dlc_unconfirmed"
	TechSurfaceEnvironmentReady           = "ready"
	TechSurfaceCompatibilityAllPlayers    = "all_players"
	TechSurfaceCompatibilityRequiresDLC   = "requires_dlc"
)

type TechSurfaceBranchCreateInput struct {
	ID       string      `json:"id"`
	EntryID  string      `json:"entry_id"`
	TargetID string      `json:"target_id"`
	Root     CreateInput `json:"root"`
}

type TechSurfaceBranchResult struct {
	Manifest ManifestResult     `json:"manifest"`
	Branch   techsurface.Branch `json:"branch"`
	Root     Technology         `json:"root"`
}

type TechSurfaceBranchTechnologyCreateInput struct {
	ParentTechnologyID string      `json:"parent_technology_id"`
	Technology         CreateInput `json:"technology"`
}

type TechSurfaceBranchTechnologyResult struct {
	Manifest   ManifestResult     `json:"manifest"`
	Branch     techsurface.Branch `json:"branch"`
	Technology Technology         `json:"technology"`
}

type ManifestResult struct {
	SchemaVersion     int                  `json:"schema_version"`
	Profile           string               `json:"profile"`
	BaselineSHA256    string               `json:"baseline_sha256"`
	CompatibilityMode string               `json:"compatibility_mode,omitempty"`
	Branches          []techsurface.Branch `json:"branches"`
}

type TechSurfaceStatus struct {
	Enabled                    bool                         `json:"enabled"`
	BaselineAttached           bool                         `json:"baseline_attached"`
	LegacyTimelineRepairNeeded bool                         `json:"legacy_timeline_repair_needed"`
	Manifest                   ManifestResult               `json:"manifest"`
	Environment                TechSurfaceEnvironmentStatus `json:"environment"`
	ActiveProfile              *TechSurfaceProfileResult    `json:"active_profile,omitempty"`
}

type TechSurfaceRepairResult struct {
	Manifest              ManifestResult `json:"manifest"`
	RepairedTechnologyIDs []string       `json:"repaired_technology_ids"`
}

type TechSurfaceEnvironmentInput struct {
	DLC               techsurfaceprofiles.DLCState `json:"dlc"`
	CompatibilityMode string                       `json:"compatibility_mode"`
}

type TechSurfaceEnvironmentStatus struct {
	State             string                                `json:"state"`
	Code              techsurfaceprofiles.ErrorCode         `json:"code,omitempty"`
	Detail            string                                `json:"detail"`
	Source            techsurfaceprofiles.SourceFingerprint `json:"source"`
	SourceConnection  TechnologySourceConnection            `json:"source_connection"`
	DLCConfirmed      bool                                  `json:"dlc_confirmed"`
	DLC               techsurfaceprofiles.DLCState          `json:"dlc"`
	CompatibilityMode string                                `json:"compatibility_mode"`
}

type TechSurfaceProfileResult struct {
	ID      string                          `json:"id"`
	Entries []TechSurfaceProfileEntryResult `json:"entries"`
}

type TechSurfaceProfileEntryResult struct {
	ID      string                           `json:"id"`
	Ledger  string                           `json:"ledger"`
	Targets []TechSurfaceProfileTargetResult `json:"targets"`
}

type TechSurfaceProfileTargetResult struct {
	ID                  string                            `json:"id"`
	FolderID            string                            `json:"folder_id"`
	AuthoringEnabled    bool                              `json:"authoring_enabled"`
	Certification       techsurfaceprofiles.Certification `json:"certification"`
	TimelineYears       []int                             `json:"timeline_years"`
	AllPlayersSupported bool                              `json:"all_players_supported"`
}

type technologySurfaceProvenance struct {
	SchemaVersion     int                                   `json:"schema_version"`
	Source            techsurfaceprofiles.SourceFingerprint `json:"source"`
	TechnologySource  *technologySourceMetadata             `json:"technology_source,omitempty"`
	DLCConfirmed      bool                                  `json:"dlc_confirmed"`
	Environment       techsurfaceprofiles.Environment       `json:"environment,omitempty"`
	CompatibilityMode string                                `json:"compatibility_mode,omitempty"`
}

type technologySurfaceSelection struct {
	Environment       techsurfaceprofiles.Environment
	CompatibilityMode string
}

func manifestResult(manifest techsurface.Manifest) ManifestResult {
	branches := manifest.Branches
	if branches == nil {
		branches = []techsurface.Branch{}
	}
	return ManifestResult{
		SchemaVersion: manifest.SchemaVersion, Profile: manifest.Profile,
		BaselineSHA256: manifest.BaselineSHA256, CompatibilityMode: manifest.CompatibilityMode, Branches: branches,
	}
}

func manifestValue(manifest techsurface.Manifest) any {
	return jsonValue(manifest)
}

func normalizedTechSurfaceCompatibilityMode(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return TechSurfaceCompatibilityAllPlayers, nil
	}
	switch value {
	case TechSurfaceCompatibilityAllPlayers, TechSurfaceCompatibilityRequiresDLC:
		return value, nil
	default:
		return "", surfaceError{Detail: "Choose either compatibility for all players or a Mod that requires the selected DLC."}
	}
}

func profileResult(profile techsurfaceprofiles.Profile) TechSurfaceProfileResult {
	result := TechSurfaceProfileResult{ID: profile.ID, Entries: []TechSurfaceProfileEntryResult{}}
	for _, entry := range profile.Entries {
		entryResult := TechSurfaceProfileEntryResult{ID: entry.ID, Ledger: entry.Ledger, Targets: []TechSurfaceProfileTargetResult{}}
		for _, target := range entry.Targets {
			_, allPlayersErr := techsurfaceprofiles.AllPlayerTargets(entry.ID, target.ID)
			targetResult := TechSurfaceProfileTargetResult{
				ID: target.ID, FolderID: target.FolderID,
				AuthoringEnabled: target.AuthoringEnabled, Certification: target.Certification,
				TimelineYears: []int{}, AllPlayersSupported: allPlayersErr == nil,
			}
			for _, cell := range target.Timeline {
				targetResult.TimelineYears = append(targetResult.TimelineYears, cell.StartYear)
			}
			entryResult.Targets = append(entryResult.Targets, targetResult)
		}
		result.Entries = append(result.Entries, entryResult)
	}
	return result
}

func decodeTechnologySurfaceProvenance(value any) (map[string]any, technologySurfaceProvenance, error) {
	root := map[string]any{}
	if value != nil {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, technologySurfaceProvenance{}, err
		}
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, technologySurfaceProvenance{}, err
		}
		if root == nil {
			root = map[string]any{}
		}
	}
	part, ok := root["technology_surface"]
	if !ok {
		return root, technologySurfaceProvenance{}, nil
	}
	encoded, err := json.Marshal(part)
	if err != nil {
		return nil, technologySurfaceProvenance{}, err
	}
	var provenance technologySurfaceProvenance
	if err := json.Unmarshal(encoded, &provenance); err != nil {
		return nil, technologySurfaceProvenance{}, err
	}
	return root, provenance, nil
}

func technologySurfaceEnvironmentStatus(provenance technologySurfaceProvenance, loadedExistingMod bool) (TechSurfaceEnvironmentStatus, *techsurfaceprofiles.Profile) {
	compatibilityMode, compatibilityErr := normalizedTechSurfaceCompatibilityMode(provenance.CompatibilityMode)
	status := TechSurfaceEnvironmentStatus{
		State:  TechSurfaceEnvironmentDLCUnconfirmed,
		Detail: "Choose player compatibility and confirm the relevant DLCs you own before creating a technology branch.",
		Source: provenance.Source, SourceConnection: technologySourceConnection(provenance, loadedExistingMod),
		DLCConfirmed: provenance.DLCConfirmed, CompatibilityMode: compatibilityMode,
	}
	if compatibilityErr != nil {
		status.State = TechSurfaceEnvironmentDLCUnconfirmed
		status.Detail = compatibilityErr.Error()
		return status, nil
	}
	if _, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(provenance.Source, techsurfaceprofiles.DLCState{})); err != nil {
		status.State = string(techsurfaceprofiles.ErrorCodeOf(err))
		status.Code = techsurfaceprofiles.ErrorCodeOf(err)
		status.Detail = err.Error()
		return status, nil
	}
	if !provenance.DLCConfirmed {
		return status, nil
	}
	status.DLC = provenance.Environment.DLC
	profile, err := techsurfaceprofiles.Resolve(provenance.Environment)
	if err != nil {
		status.State = string(techsurfaceprofiles.ErrorCodeOf(err))
		status.Code = techsurfaceprofiles.ErrorCodeOf(err)
		status.Detail = err.Error()
		return status, nil
	}
	status.State = TechSurfaceEnvironmentReady
	status.Detail = "The imported source, owned DLC preview and player compatibility policy resolve to certified export placements."
	return status, &profile
}

func guiSourceRow(rows []project.Row) project.Row {
	for _, row := range rows {
		if text(row, "asset_type", "") == "gui_source" && text(row, "gfx_name", "") == "snapshot.zip" {
			return row
		}
	}
	return nil
}

func attachedGUISource(rows []project.Row) project.Row {
	if row := guiSourceRow(rows); row != nil && text(row, "obs_key", "") != "" {
		return row
	}
	return nil
}

func technologySurfaceEnvironment(rows []project.Row) (technologySurfaceSelection, error) {
	row := attachedGUISource(rows)
	if row == nil {
		return technologySurfaceSelection{}, surfaceError{Detail: "Import and lock the original technology GUI baseline before creating a visible extension branch."}
	}
	_, provenance, err := decodeTechnologySurfaceProvenance(row["provenance"])
	if err != nil {
		return technologySurfaceSelection{}, err
	}
	if !provenance.DLCConfirmed {
		return technologySurfaceSelection{}, surfaceError{Detail: "Confirm the DLC ownership and player compatibility policy before creating a technology branch."}
	}
	mode, err := normalizedTechSurfaceCompatibilityMode(provenance.CompatibilityMode)
	if err != nil {
		return technologySurfaceSelection{}, err
	}
	return technologySurfaceSelection{Environment: provenance.Environment, CompatibilityMode: mode}, nil
}

func technologySurfaceStatus(p *project.Project) (TechSurfaceStatus, error) {
	record := p.Record()
	manifest, err := recordManifest(record)
	if err != nil {
		return TechSurfaceStatus{}, err
	}
	assets, err := p.Rows(assetsTable)
	if err != nil {
		return TechSurfaceStatus{}, err
	}
	row := attachedGUISource(assets)
	status := TechSurfaceStatus{Enabled: techsurface.IsEnabled(manifest), BaselineAttached: row != nil, Manifest: manifestResult(manifest)}
	if row == nil {
		status.Environment = TechSurfaceEnvironmentStatus{
			State:            TechSurfaceEnvironmentBaselineMissing,
			Detail:           "Connect a supported local HOI4 installation before creating a branch.",
			Source:           techsurfaceprofiles.SourceFingerprint{Files: []techsurfaceprofiles.SourceFile{}},
			SourceConnection: TechnologySourceConnection{State: TechnologySourceNotConnected, Kind: TechnologySourceKindNone},
		}
	} else {
		_, provenance, err := decodeTechnologySurfaceProvenance(row["provenance"])
		if err != nil {
			return TechSurfaceStatus{}, err
		}
		var resolved *techsurfaceprofiles.Profile
		status.Environment, resolved = technologySurfaceEnvironmentStatus(provenance, boolean(record, "is_loaded_existing_mod"))
		if resolved != nil {
			profile := profileResult(*resolved)
			status.ActiveProfile = &profile
		}
	}
	if status.Enabled {
		technologies, err := p.Rows(table)
		if err != nil {
			return TechSurfaceStatus{}, err
		}
		repairNeeded, err := technologySurfaceTimelineRepairNeeded(technologies, manifest)
		if err != nil {
			return TechSurfaceStatus{}, err
		}
		status.LegacyTimelineRepairNeeded = repairNeeded
	}
	return status, nil
}

func (h *handlers) surfaceStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	status, err := technologySurfaceStatus(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology surface unavailable")
		return
	}
	apiutil.JSON(w, http.StatusOK, status)
}

func writeSurfaceError(w http.ResponseWriter, err error) {
	var branchError surfaceError
	var selection techsurfaceprofiles.SelectionError
	var findings techsurface.FindingsError
	switch {
	case errors.As(err, &branchError):
		detail(w, http.StatusUnprocessableEntity, branchError.Detail)
	case errors.As(err, &findings):
		apiutil.JSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": "Technology surface branch is not exportable", "findings": findings.Findings})
	case errors.As(err, &selection):
		apiutil.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": selection.Code, "detail": selection.Detail})
	default:
		detail(w, http.StatusServiceUnavailable, "technology surface unavailable")
	}
}

func (h *handlers) surfaceEnvironment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	var input TechSurfaceEnvironmentInput
	if err := decodeBody(w, r, 1<<20, true, &input); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology environment payload")
		return
	}
	if err := confirmTechnologySurfaceEnvironment(p, input); err != nil {
		writeSurfaceError(w, err)
		return
	}
	status, err := technologySurfaceStatus(p)
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology surface unavailable")
		return
	}
	apiutil.JSON(w, http.StatusOK, status)
}

func confirmTechnologySurfaceEnvironment(p *project.Project, input TechSurfaceEnvironmentInput) error {
	return p.Write([]string{assetsTable}, func(tx *project.Tx) error {
		manifest, err := recordManifest(tx.Record())
		if err != nil {
			return err
		}
		row := attachedGUISource(tx.Rows(assetsTable))
		if row == nil {
			return surfaceError{Detail: "Import both original technology source files before confirming a DLC profile."}
		}
		root, provenance, err := decodeTechnologySurfaceProvenance(row["provenance"])
		if err != nil {
			return err
		}
		compatibilityMode, err := normalizedTechSurfaceCompatibilityMode(input.CompatibilityMode)
		if err != nil {
			return err
		}
		if compatibilityMode == TechSurfaceCompatibilityRequiresDLC && !input.DLC.NoStepBack && !input.DLC.ByBloodAlone && !input.DLC.ManTheGuns {
			return surfaceError{Detail: "Select at least one required DLC, or use compatibility for all players."}
		}
		previousCompatibilityMode, err := normalizedTechSurfaceCompatibilityMode(provenance.CompatibilityMode)
		if err != nil {
			return err
		}
		if techsurface.IsEnabled(manifest) && provenance.DLCConfirmed && previousCompatibilityMode != compatibilityMode {
			return surfaceError{Detail: "This project already has technology branches for a different player compatibility policy; remove or rebase those branches before changing it."}
		}
		environment := techsurfaceprofiles.NewEnvironment(provenance.Source, input.DLC)
		profile, err := techsurfaceprofiles.Resolve(environment)
		if err != nil {
			return err
		}
		if techsurface.IsEnabled(manifest) && manifest.Profile != profile.ID {
			return surfaceError{Detail: "This project already has branches for a different DLC profile; rebase or remove those branches before changing the environment."}
		}
		provenance.SchemaVersion = 1
		provenance.DLCConfirmed = true
		provenance.Environment = environment
		provenance.CompatibilityMode = compatibilityMode
		root["technology_surface"] = provenance
		_, err = tx.Update(assetsTable, project.Int64(row["id"]), func(asset project.Row) {
			asset["provenance"] = jsonValue(root)
			asset["created_at"] = workbench.NowTimestamp()
		})
		return err
	})
}

type techSurfaceTimelineAssignment struct {
	TechnologyID string
	Folder       string
	Cell         techsurface.TimelineCell
	Placements   []techSurfaceTimelinePlacement
}

type techSurfaceTimelinePlacement struct {
	Folder string
	Cell   techsurface.TimelineCell
}

func technologySurfaceTimelineAssignments(manifest techsurface.Manifest) ([]techSurfaceTimelineAssignment, error) {
	if !techsurface.IsEnabled(manifest) {
		return nil, surfaceError{Detail: "This project has no certified technology surface branch to repair."}
	}
	profile, exists := techsurface.ProfileByID(manifest.Profile)
	if !exists {
		return nil, surfaceError{Detail: "The project's certified technology surface profile is unavailable."}
	}
	assignments := []techSurfaceTimelineAssignment{}
	for _, branch := range manifest.Branches {
		for index, technologyID := range branch.TechnologyIDs {
			assignment := techSurfaceTimelineAssignment{TechnologyID: strings.TrimSpace(technologyID), Folder: branch.Folder}
			for placementIndex, placement := range surfaceBranchPlacements(branch) {
				folder, exists := profile.Folders[placement.Folder]
				if !exists || folder.Surface != branch.Surface {
					return nil, surfaceError{Detail: "A branch no longer belongs to a certified research area and cannot be repaired automatically."}
				}
				cell, ok := folder.TimelineCellAt(index)
				if !ok {
					return nil, surfaceError{Detail: "This branch is longer than a certified player-compatible timeline and cannot be repaired automatically."}
				}
				assignment.Placements = append(assignment.Placements, techSurfaceTimelinePlacement{Folder: placement.Folder, Cell: cell})
				if placementIndex == 0 {
					assignment.Cell = cell
				}
			}
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

func timelineAssignmentNeedsRepair(technology techsurface.Technology, assignment techSurfaceTimelineAssignment) bool {
	if technology.StartYear != assignment.Cell.StartYear {
		return true
	}
	for _, placement := range assignment.Placements {
		if !technologyHasTimelinePlacement(technology, placement.Folder, placement.Cell) {
			return true
		}
	}
	return false
}

func technologyHasTimelinePlacement(technology techsurface.Technology, folder string, cell techsurface.TimelineCell) bool {
	for _, placement := range technology.FolderPlacements {
		if strings.TrimSpace(placement.Folder) == folder && placement.X == float64(cell.X) && placement.Y == float64(cell.Y) {
			return true
		}
	}
	return false
}

func technologySurfaceTimelineRepairNeeded(rows []project.Row, manifest techsurface.Manifest) (bool, error) {
	assignments, err := technologySurfaceTimelineAssignments(manifest)
	if err != nil {
		return false, err
	}
	byID := map[string]techsurface.Technology{}
	for _, technology := range techSurfaceRows(rows) {
		byID[technology.ID] = technology
	}
	for _, assignment := range assignments {
		technology, found := byID[assignment.TechnologyID]
		if !found || timelineAssignmentNeedsRepair(technology, assignment) {
			return true, nil
		}
	}
	return false, nil
}

func prepareTechSurfaceBranch(current techsurface.Manifest, isSubmod bool, resolved techsurfaceprofiles.Profile, compatibilityMode string, input TechSurfaceBranchCreateInput) (techsurface.Manifest, techsurface.Branch, CreateInput, error) {
	if isSubmod {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{
			Detail: "Technology surface branches cannot patch a dependency's research GUI in P0.",
		}
	}
	profile, exists := techsurface.ProfileByID(resolved.ID)
	if !exists {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, fmt.Errorf("resolved technology surface profile is unavailable")
	}
	if !techsurface.IsEnabled(current) {
		current = techsurface.Manifest{
			SchemaVersion: techsurface.SchemaVersion, Profile: profile.ID, BaselineSHA256: profile.BaselineSHA256,
			CompatibilityMode: compatibilityMode, Branches: []techsurface.Branch{},
		}
	}
	if current.CompatibilityMode != "" && current.CompatibilityMode != compatibilityMode {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{
			Detail: "This project's existing technology branches use a different player compatibility policy.",
		}
	}
	if current.Profile != profile.ID || techsurface.NormalizedSHA256(current.BaselineSHA256) != profile.BaselineSHA256 {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{
			Detail: "This project is bound to a different technology baseline and must be rebased before adding a branch.",
		}
	}
	if input.Root.Folder != nil || len(input.Root.FolderPlacements) > 0 || input.Root.TechLine != nil || input.Root.StartYear != nil || len(input.Root.Path) > 0 {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{
			Detail: "A surface branch chooses its certified research area and root slot; root folder, path, and free placement are not accepted.",
		}
	}
	branch := techsurface.Branch{
		ID: strings.TrimSpace(input.ID), Surface: strings.TrimSpace(input.EntryID),
		RootTechnologyID: strings.TrimSpace(input.Root.TechID),
	}
	if branch.RootTechnologyID == "" {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: "A surface branch needs a stable root technology ID."}
	}
	target, targetErr := resolved.AuthoringTarget(strings.TrimSpace(input.EntryID), strings.TrimSpace(input.TargetID))
	if targetErr != nil {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: targetErr.Error()}
	}
	if len(target.Slots) != 1 {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: "The certified research target does not expose exactly one P0 branch slot."}
	}
	branch.Folder = target.FolderID
	branch.Slot = target.Slots[0].ID
	branch.Placements = []techsurface.BranchPlacement{{Folder: branch.Folder, Slot: branch.Slot}}
	if compatibilityMode == TechSurfaceCompatibilityAllPlayers {
		compatibleTargets, compatibilityErr := techsurfaceprofiles.AllPlayerTargets(strings.TrimSpace(input.EntryID), strings.TrimSpace(input.TargetID))
		if compatibilityErr != nil {
			return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: compatibilityErr.Error()}
		}
		for _, compatibleTarget := range compatibleTargets {
			if compatibleTarget.FolderID == branch.Folder {
				continue
			}
			if len(compatibleTarget.Slots) != 1 {
				return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: "A player-compatible research page does not expose exactly one certified P0 branch slot."}
			}
			branch.Placements = append(branch.Placements, techsurface.BranchPlacement{Folder: compatibleTarget.FolderID, Slot: compatibleTarget.Slots[0].ID})
		}
	}
	folder, folderOK := profile.Folders[branch.Folder]
	if !folderOK || folder.Surface != branch.Surface {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: "Choose a certified research target under one of the four fixed entrances."}
	}
	if !hasSurfaceDescription(input.Root) {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, surfaceError{Detail: "A visible branch technology needs an English description so the game never displays a raw _desc key."}
	}
	cell, cellOK := folder.TimelineCellAt(0)
	if !cellOK {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, fmt.Errorf("certified surface folder %q has no root timeline slot", branch.Folder)
	}
	branch.TechnologyIDs = []string{branch.RootTechnologyID}
	candidate := current
	candidate.Branches = append(append([]techsurface.Branch{}, current.Branches...), branch)
	if findings := techsurface.ValidateShape(candidate, false); len(findings) > 0 {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, techsurface.FindingsError{Findings: findings}
	}
	root := input.Root
	created := "created"
	techLine := folder.TechLine
	if techLine == "" {
		return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, fmt.Errorf("certified surface folder %q has no technology line", branch.Folder)
	}
	root.Source = &created
	root.TechLine = &techLine
	root.Folder = &branch.Folder
	root.StartYear = &cell.StartYear
	root.FolderPlacements = []map[string]any{}
	for _, placement := range branch.Placements {
		placementFolder, exists := profile.Folders[placement.Folder]
		if !exists {
			return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, fmt.Errorf("certified surface folder %q is unavailable", placement.Folder)
		}
		placementCell, exists := placementFolder.TimelineCellAt(0)
		if !exists {
			return techsurface.Manifest{}, techsurface.Branch{}, CreateInput{}, fmt.Errorf("certified surface folder %q has no root timeline slot", placement.Folder)
		}
		root.FolderPlacements = append(root.FolderPlacements, map[string]any{"folder": placement.Folder, "x": placementCell.X, "y": placementCell.Y})
	}
	root.PositionX = float64(cell.X)
	root.PositionY = float64(cell.Y)
	root.PositionYIsExplicit = true
	root.Path = []map[string]any{}
	root.Normalize()
	return candidate, branch, root, nil
}

func hasSurfaceDescription(input CreateInput) bool {
	if strings.TrimSpace(input.DescEn) != "" {
		return true
	}
	return strings.TrimSpace(input.DescriptionLocs["en"]) != ""
}

func techSurfaceRows(rows []project.Row) []techsurface.Technology {
	result := []techsurface.Technology{}
	for _, row := range rows {
		technology := techsurface.Technology{
			ID: text(row, "tech_id", ""), Source: text(row, "source", ""),
			Folder: text(row, "folder", ""), StartYear: integer(row, "start_year", 1936),
		}
		var placements, paths []map[string]any
		decodeColumn(row["folder_placements"], &placements)
		decodeColumn(row["path"], &paths)
		for _, placement := range placements {
			technology.FolderPlacements = append(technology.FolderPlacements, techsurface.FolderPlacement{
				Folder: strings.TrimSpace(stringFrom(placement["folder"])), X: folderPlacementFloat(placement["x"]), Y: folderPlacementFloat(placement["y"]),
			})
		}
		for _, path := range paths {
			if target := strings.TrimSpace(stringFrom(path["leads_to_tech"])); target != "" {
				technology.PathTargets = append(technology.PathTargets, target)
			}
		}
		result = append(result, technology)
	}
	return result
}

func branchIndex(manifest techsurface.Manifest, branchID string) int {
	branchID = strings.TrimSpace(branchID)
	for index, branch := range manifest.Branches {
		if strings.TrimSpace(branch.ID) == branchID {
			return index
		}
	}
	return -1
}

func surfaceBranchPlacements(branch techsurface.Branch) []techsurface.BranchPlacement {
	if len(branch.Placements) > 0 {
		return branch.Placements
	}
	return []techsurface.BranchPlacement{{Folder: branch.Folder, Slot: branch.Slot}}
}

func surfaceManagedTechnology(manifest techsurface.Manifest, technologyID string) bool {
	technologyID = strings.TrimSpace(technologyID)
	for _, branch := range manifest.Branches {
		for _, memberID := range branch.TechnologyIDs {
			if strings.TrimSpace(memberID) == technologyID {
				return true
			}
		}
	}
	return false
}

func protectedSurfaceFolder(manifest techsurface.Manifest, input CreateInput) bool {
	if !techsurface.IsEnabled(manifest) {
		return false
	}
	profile, exists := techsurface.ProfileByID(manifest.Profile)
	if !exists {
		return false
	}
	techLine := strings.TrimSpace(workbench.StrDefault(input.TechLine, "infantry"))
	folder := strings.TrimSpace(workbench.StrDefault(input.Folder, ""))
	if folder == "" {
		folder = techLineFolders[techLine]
		if folder == "" {
			folder = techLine + "_folder"
		}
	}
	if placementFolder, _, _, ok := firstFolderPlacement(normalizedFolderPlacements(input.FolderPlacements, folder)); ok {
		folder = placementFolder
	}
	_, isCertified := profile.Folders[folder]
	return isCertified
}

func prepareTechSurfaceBranchTechnology(manifest techsurface.Manifest, branch techsurface.Branch, input TechSurfaceBranchTechnologyCreateInput) (CreateInput, string, error) {
	parentID := strings.TrimSpace(input.ParentTechnologyID)
	if parentID == "" {
		return CreateInput{}, "", surfaceError{Detail: "Choose the preceding technology in this research branch."}
	}
	isMember := false
	for _, technologyID := range branch.TechnologyIDs {
		if strings.TrimSpace(technologyID) == parentID {
			isMember = true
			break
		}
	}
	if !isMember {
		return CreateInput{}, "", surfaceError{Detail: "The preceding technology must belong to the selected surface branch."}
	}
	child := input.Technology
	child.TechID = strings.TrimSpace(child.TechID)
	if child.TechID == "" {
		return CreateInput{}, "", surfaceError{Detail: "The next branch technology needs a stable technology ID."}
	}
	if child.Folder != nil || len(child.FolderPlacements) > 0 || child.TechLine != nil || child.StartYear != nil || len(child.Path) > 0 || child.PositionX != 0 || child.PositionY != 0 || child.PositionYIsExplicit {
		return CreateInput{}, "", surfaceError{Detail: "Branch technology placement and visual paths are assigned by the certified research surface, not by a free canvas."}
	}
	profile, exists := techsurface.ProfileByID(manifest.Profile)
	if !exists {
		return CreateInput{}, "", surfaceError{Detail: "The project's certified technology surface profile is unavailable."}
	}
	folder, exists := profile.Folders[branch.Folder]
	if !exists {
		return CreateInput{}, "", surfaceError{Detail: "The branch's certified research area is unavailable."}
	}
	if !hasSurfaceDescription(child) {
		return CreateInput{}, "", surfaceError{Detail: "A visible branch technology needs an English description so the game never displays a raw _desc key."}
	}
	cell, cellOK := folder.TimelineCellAt(len(branch.TechnologyIDs))
	if !cellOK {
		return CreateInput{}, "", surfaceError{Detail: "This certified branch has no remaining original-game timeline slot. Create a new certified branch instead of placing a free node."}
	}
	created := "created"
	techLine := folder.TechLine
	child.Source = &created
	child.TechLine = &techLine
	child.Folder = &branch.Folder
	child.StartYear = &cell.StartYear

	child.FolderPlacements = []map[string]any{}
	for _, placement := range surfaceBranchPlacements(branch) {
		placementFolder, exists := profile.Folders[placement.Folder]
		if !exists {
			return CreateInput{}, "", surfaceError{Detail: "A generated player-compatible research page is unavailable."}
		}
		placementCell, exists := placementFolder.TimelineCellAt(len(branch.TechnologyIDs))
		if !exists {
			return CreateInput{}, "", surfaceError{Detail: "A compatible research page has no remaining certified timeline slot. Create a new branch instead."}
		}
		child.FolderPlacements = append(child.FolderPlacements, map[string]any{"folder": placement.Folder, "x": placementCell.X, "y": placementCell.Y})
	}
	child.PositionX = float64(cell.X)
	child.PositionY = float64(cell.Y)
	child.PositionYIsExplicit = true
	child.Path = []map[string]any{}
	child.Normalize()
	return child, parentID, nil
}

func isSubmod(record project.Row) bool {
	var deps []string
	decodeColumn(record["dependencies"], &deps)
	return len(deps) > 0
}

func findTechByID(rows []project.Row, techID string) project.Row {
	for _, row := range rows {
		if text(row, "tech_id", "") == techID {
			return row
		}
	}
	return nil
}

func (h *handlers) createBranch(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	var input TechSurfaceBranchCreateInput
	if err := decodeBody(w, r, apiutil.MaxJSONBody, false, &input); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology surface branch payload")
		return
	}
	vanilla := h.game.techIDs()
	var result TechSurfaceBranchResult
	err := p.Write([]string{table, assetsTable}, func(tx *project.Tx) error {

		record := tx.Record()
		submod := isSubmod(record)
		current, err := recordManifest(record)
		if err != nil {
			return err
		}
		selection, err := technologySurfaceEnvironment(tx.Rows(assetsTable))
		if err != nil {
			return err
		}
		resolved, err := techsurfaceprofiles.Resolve(selection.Environment)
		if err != nil {
			return surfaceError{Detail: err.Error()}
		}
		candidate, branch, rootInput, err := prepareTechSurfaceBranch(current, submod, resolved, selection.CompatibilityMode, input)
		if err != nil {
			return err
		}
		root, err := createTechnology(tx, rootInput, vanilla)
		if err != nil {
			return err
		}
		if findings := techsurface.Validate(candidate, techSurfaceRows(tx.Rows(table)), submod); len(findings) > 0 {
			return techsurface.FindingsError{Findings: findings}
		}
		tx.SetRecord(func(record project.Row) {
			record["tech_surface_manifest"] = manifestValue(candidate)
			record["updated_at"] = workbench.NowTimestamp()
		})
		result = TechSurfaceBranchResult{Manifest: manifestResult(candidate), Branch: branch, Root: root}
		return nil
	})
	if err != nil {
		writeSurfaceError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusCreated, result)
}

func (h *handlers) createBranchTechnology(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	branchID := r.PathValue("branch")
	var input TechSurfaceBranchTechnologyCreateInput
	if err := decodeBody(w, r, apiutil.MaxJSONBody, false, &input); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid branch technology payload")
		return
	}
	vanilla := h.game.techIDs()
	var result TechSurfaceBranchTechnologyResult
	err := p.Write([]string{table, assetsTable}, func(tx *project.Tx) error {

		record := tx.Record()
		if isSubmod(record) {
			return surfaceError{Detail: "Technology surface branches cannot patch a dependency's research GUI in P0."}
		}
		manifest, err := recordManifest(record)
		if err != nil {
			return err
		}
		if !techsurface.IsEnabled(manifest) {
			return surfaceError{Detail: "Create a certified extension branch before adding technology to it."}
		}
		selection, err := technologySurfaceEnvironment(tx.Rows(assetsTable))
		if err != nil {
			return err
		}
		resolved, err := techsurfaceprofiles.Resolve(selection.Environment)
		if err != nil {
			return surfaceError{Detail: err.Error()}
		}
		if manifest.Profile != resolved.ID {
			return surfaceError{Detail: "The confirmed DLC environment no longer matches this branch profile; rebase before adding technology."}
		}
		index := branchIndex(manifest, branchID)
		if index < 0 {
			return surfaceError{Detail: "The selected technology surface branch does not exist."}
		}
		branch := manifest.Branches[index]
		childInput, parentID, err := prepareTechSurfaceBranchTechnology(manifest, branch, input)
		if err != nil {
			return err
		}
		child, err := createTechnology(tx, childInput, vanilla)
		if err != nil {
			return err
		}
		if err := appendSurfacePath(tx, parentID, child.TechID); err != nil {
			return err
		}
		branch.TechnologyIDs = append(append([]string{}, branch.TechnologyIDs...), child.TechID)
		manifest.Branches[index] = branch
		if findings := techsurface.Validate(manifest, techSurfaceRows(tx.Rows(table)), false); len(findings) > 0 {
			return techsurface.FindingsError{Findings: findings}
		}
		tx.SetRecord(func(record project.Row) {
			record["tech_surface_manifest"] = manifestValue(manifest)
			record["updated_at"] = workbench.NowTimestamp()
		})
		result = TechSurfaceBranchTechnologyResult{Manifest: manifestResult(manifest), Branch: branch, Technology: child}
		return nil
	})
	if err != nil {
		writeSurfaceError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusCreated, result)
}

func appendSurfacePath(tx *project.Tx, parentID, childID string) error {
	parent := findTechByID(tx.Rows(table), parentID)
	if parent == nil {
		return surfaceError{Detail: "The preceding technology no longer exists in this project."}
	}
	var paths []map[string]any
	decodeColumn(parent["path"], &paths)
	for _, path := range paths {
		if strings.TrimSpace(stringFrom(path["leads_to_tech"])) == childID {
			return surfaceError{Detail: "The preceding technology already points to this branch technology."}
		}
	}
	paths = append(paths, map[string]any{"leads_to_tech": childID})
	_, err := tx.Update(table, project.Int64(parent["id"]), func(row project.Row) { row["path"] = jsonValue(paths) })
	return err
}

func (h *handlers) surfaceRepair(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	result, err := repairTechnologySurfaceTimeline(p)
	if err != nil {
		writeSurfaceError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}

func repairTechnologySurfaceTimeline(p *project.Project) (TechSurfaceRepairResult, error) {
	result := TechSurfaceRepairResult{RepairedTechnologyIDs: []string{}}
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		record := tx.Record()
		if isSubmod(record) {
			return surfaceError{Detail: "Technology surface branches cannot patch a dependency's research GUI in P0."}
		}
		manifest, err := recordManifest(record)
		if err != nil {
			return err
		}
		assignments, err := technologySurfaceTimelineAssignments(manifest)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			row := findTechByID(tx.Rows(table), assignment.TechnologyID)
			if row == nil {
				return surfaceError{Detail: "A branch technology no longer exists and cannot be repaired automatically."}
			}
			descriptionLocs := map[string]string{}
			decodeColumn(row["description_locs"], &descriptionLocs)
			if descriptionLocs == nil {
				descriptionLocs = map[string]string{}
			}
			description := strings.TrimSpace(descriptionLocs["en"])
			if description == "" {
				description = strings.TrimSpace(text(row, "desc_en", ""))
			}
			if description == "" {
				description = strings.TrimSpace(text(row, "name", ""))
			}
			if description == "" {
				description = assignment.TechnologyID
			}
			descriptionLocs["en"] = description
			placements := []map[string]any{}
			for _, placement := range assignment.Placements {
				placements = append(placements, map[string]any{"folder": placement.Folder, "x": placement.Cell.X, "y": placement.Cell.Y})
			}
			if _, err := tx.Update(table, project.Int64(row["id"]), func(row project.Row) {
				row["folder"] = assignment.Folder
				row["start_year"] = int64(assignment.Cell.StartYear)
				row["position_x"] = float64(assignment.Cell.X)
				row["position_y"] = float64(assignment.Cell.Y)
				row["position_y_is_explicit"] = true
				row["folder_placements"] = jsonValue(placements)
				row["desc_en"] = description
				row["description_locs"] = jsonValue(descriptionLocs)
			}); err != nil {
				return err
			}
			result.RepairedTechnologyIDs = append(result.RepairedTechnologyIDs, assignment.TechnologyID)
		}
		if findings := techsurface.Validate(manifest, techSurfaceRows(tx.Rows(table)), false); len(findings) > 0 {
			return techsurface.FindingsError{Findings: findings}
		}
		result.Manifest = manifestResult(manifest)
		return nil
	})
	return result, err
}
