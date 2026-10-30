package techsurface

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
)

const (
	SchemaVersion = 1
	BaselinePath  = "interface/countrytechtreeview.gui"
)

var (
	idRE     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	sha256RE = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Manifest struct {
	SchemaVersion     int      `json:"schema_version"`
	Profile           string   `json:"profile"`
	BaselineSHA256    string   `json:"baseline_sha256"`
	CompatibilityMode string   `json:"compatibility_mode,omitempty"`
	Branches          []Branch `json:"branches"`
}

type Branch struct {
	ID               string `json:"id"`
	Surface          string `json:"surface"`
	Folder           string `json:"folder"`
	Slot             string `json:"slot"`
	RootTechnologyID string `json:"root_technology_id"`

	TechnologyIDs []string `json:"technology_ids"`

	Placements []BranchPlacement `json:"placements,omitempty"`
}

type BranchPlacement struct {
	Folder string `json:"folder"`
	Slot   string `json:"slot"`
}

type Technology struct {
	ID               string
	Source           string
	Folder           string
	StartYear        int
	FolderPlacements []FolderPlacement
	PathTargets      []string
}

type FolderPlacement struct {
	Folder string
	X      float64
	Y      float64
}

type Slot struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
}

type TimelineCell struct {
	StartYear int `json:"start_year"`
	X         int `json:"x"`
	Y         int `json:"y"`
}

type FolderProfile struct {
	ID      string `json:"id"`
	Surface string `json:"surface"`

	TechLine string          `json:"tech_line"`
	Slots    map[string]Slot `json:"slots"`

	GridboxFormat string `json:"gridbox_format,omitempty"`
	SlotWidth     int    `json:"slot_width,omitempty"`
	SlotHeight    int    `json:"slot_height,omitempty"`

	Timeline []TimelineCell `json:"timeline"`
}

func (f FolderProfile) TimelineCellAt(index int) (TimelineCell, bool) {
	if index < 0 || index >= len(f.Timeline) {
		return TimelineCell{}, false
	}
	return f.Timeline[index], true
}

type Profile struct {
	ID             string                   `json:"id"`
	BaselineSHA256 string                   `json:"baseline_sha256"`
	Folders        map[string]FolderProfile `json:"folders"`
}

var profiles = productionProfilesFromCatalogue()

var legacyInfantryProfile = legacyInfantryProfileFromCatalogue()

func productionProfilesFromCatalogue() map[string]Profile {
	result := make(map[string]Profile, 8)
	for mask := 0; mask < 8; mask++ {
		state := techsurfaceprofiles.DLCState{
			NoStepBack: mask&1 != 0, ByBloodAlone: mask&2 != 0, ManTheGuns: mask&4 != 0,
		}
		selected, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(techsurfaceprofiles.AuditedSourceFingerprint(), state))
		if err != nil {
			panic(fmt.Sprintf("resolve certified technology surface profile: %v", err))
		}
		profile := Profile{ID: selected.ID, BaselineSHA256: techsurfaceprofiles.CountryTechTreeGUISHA256, Folders: map[string]FolderProfile{}}
		for _, entry := range techsurfaceprofiles.ProductionEntries() {
			for _, group := range entry.TargetGroups {
				for _, target := range group.Variants {
					if !target.AuthoringEnabled {
						continue
					}
					folder := FolderProfile{
						ID: target.FolderID, Surface: entry.ID, TechLine: target.TechLine,
						Slots: map[string]Slot{}, GridboxFormat: target.GridboxFormat,
						SlotWidth: target.SlotSize.Width, SlotHeight: target.SlotSize.Height,
					}
					for _, slot := range target.Slots {
						folder.Slots[slot.ID] = Slot{ID: slot.ID, X: slot.X, Y: slot.Y}
					}
					for _, cell := range target.Timeline {
						folder.Timeline = append(folder.Timeline, TimelineCell{StartYear: cell.StartYear, X: cell.X, Y: cell.Y})
					}
					profile.Folders[folder.ID] = folder
				}
			}
		}
		result[profile.ID] = profile
	}
	return result
}

func legacyInfantryProfileFromCatalogue() Profile {
	const legacyID = "p0_vanilla_infantry_fec641fb"
	base, ok := profiles["p0_prod_nsb0_bba0_mtg0_fec641fb"]
	if !ok {
		panic("certified base technology surface profile missing")
	}
	infantry, ok := base.Folders["infantry_folder"]
	if !ok {
		panic("certified infantry folder missing")
	}
	return Profile{ID: legacyID, BaselineSHA256: base.BaselineSHA256, Folders: map[string]FolderProfile{infantry.ID: infantry}}
}

func Profiles() []Profile {
	out := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, profile)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func ProfileByID(id string) (Profile, bool) {
	if id == legacyInfantryProfile.ID {
		return legacyInfantryProfile, true
	}
	profile, ok := profiles[id]
	return profile, ok
}

func RegisterProfileForTest(profile Profile) func() {
	previous, existed := profiles[profile.ID]
	profiles[profile.ID] = profile
	return func() {
		if existed {
			profiles[profile.ID] = previous
		} else {
			delete(profiles, profile.ID)
		}
	}
}

type Finding struct {
	Code    string
	Message string
	Branch  string
	TechID  string
}

func Parse(raw map[string]any) (Manifest, error) {
	if len(raw) == 0 {
		return Manifest{}, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode tech surface manifest: %w", err)
	}
	return normalizeManifest(manifest), nil
}

func normalizeManifest(manifest Manifest) Manifest {
	for index := range manifest.Branches {
		branch := &manifest.Branches[index]
		if len(branch.Placements) == 0 && strings.TrimSpace(branch.Folder) != "" && strings.TrimSpace(branch.Slot) != "" {
			branch.Placements = []BranchPlacement{{Folder: strings.TrimSpace(branch.Folder), Slot: strings.TrimSpace(branch.Slot)}}
		}
		if len(manifest.Branches[index].TechnologyIDs) == 0 && strings.TrimSpace(manifest.Branches[index].RootTechnologyID) != "" {

			manifest.Branches[index].TechnologyIDs = []string{strings.TrimSpace(manifest.Branches[index].RootTechnologyID)}
		}
	}
	return manifest
}

func branchPlacements(branch Branch) []BranchPlacement {
	if len(branch.Placements) > 0 {
		return branch.Placements
	}
	if strings.TrimSpace(branch.Folder) == "" || strings.TrimSpace(branch.Slot) == "" {
		return nil
	}
	return []BranchPlacement{{Folder: branch.Folder, Slot: branch.Slot}}
}

func IsEnabled(manifest Manifest) bool {
	return manifest.SchemaVersion != 0 || manifest.Profile != "" ||
		manifest.BaselineSHA256 != "" || len(manifest.Branches) > 0
}

func NormalizedSHA256(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func Hash(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

func isCreated(source string) bool {
	source = strings.TrimSpace(strings.ToLower(source))
	return source == "" || source == "created"
}

func hasFolder(technology Technology, folder string) bool {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return false
	}
	if len(technology.FolderPlacements) > 0 {
		for _, placement := range technology.FolderPlacements {
			if strings.TrimSpace(placement.Folder) == folder {
				return true
			}
		}
		return false
	}
	return strings.TrimSpace(technology.Folder) == folder
}

func byID(techs []Technology) map[string]Technology {
	result := make(map[string]Technology, len(techs))
	for _, technology := range techs {
		id := strings.TrimSpace(technology.ID)
		if id != "" {
			result[id] = technology
		}
	}
	return result
}

func hasIncomingPath(techs []Technology, folder, target string) bool {

	byID := byID(techs)
	for _, technology := range techs {
		if !hasFolder(technology, folder) {
			continue
		}
		for _, pathTarget := range technology.PathTargets {
			if strings.TrimSpace(pathTarget) != target {
				continue
			}
			if linked, exists := byID[target]; exists && hasFolder(linked, folder) {
				return true
			}
		}
	}
	return false
}

type rootRef struct {
	Folder string
	TechID string
}

func projectRoots(profile Profile, techs []Technology) []rootRef {
	roots := []rootRef{}
	for folder := range profile.Folders {
		for _, technology := range techs {
			id := strings.TrimSpace(technology.ID)
			if !isCreated(technology.Source) || id == "" || !hasFolder(technology, folder) {
				continue
			}
			if !hasIncomingPath(techs, folder, id) {
				roots = append(roots, rootRef{Folder: folder, TechID: id})
			}
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Folder == roots[j].Folder {
			return roots[i].TechID < roots[j].TechID
		}
		return roots[i].Folder < roots[j].Folder
	})
	return roots
}

func finding(code, message, branch, techID string) Finding {
	return Finding{Code: code, Message: message, Branch: branch, TechID: techID}
}

func Validate(manifest Manifest, techs []Technology, isSubmod bool) []Finding {
	manifest = normalizeManifest(manifest)
	findings := ValidateShape(manifest, isSubmod)
	if !IsEnabled(manifest) || isSubmod {
		return findings
	}
	profile, ok := ProfileByID(strings.TrimSpace(manifest.Profile))
	if !ok {
		return findings
	}
	technologies := byID(techs)
	declaredRoots := map[string]bool{}
	declaredMembers := map[string]string{}
	for _, branch := range manifest.Branches {
		folder, folderOK := profile.Folders[branch.Folder]
		members := map[string]bool{}
		for _, techID := range branch.TechnologyIDs {
			techID = strings.TrimSpace(techID)
			if techID == "" || members[techID] {
				findings = append(findings, finding("tech_surface_branch_member_invalid",
					"Each technology can appear once in a declared surface branch.", branch.ID, techID))
				continue
			}
			members[techID] = true
			if previous, alreadyDeclared := declaredMembers[techID]; alreadyDeclared {
				findings = append(findings, finding("tech_surface_branch_member_duplicate",
					"A technology can belong to only one surface branch.", branch.ID, techID))
				_ = previous
			} else {
				declaredMembers[techID] = branch.ID
			}
		}
		if !members[branch.RootTechnologyID] {
			findings = append(findings, finding("tech_surface_root_not_member",
				"The declared root must be the first member of its surface branch.", branch.ID, branch.RootTechnologyID))
		}
		root, exists := technologies[branch.RootTechnologyID]
		if !exists || !isCreated(root.Source) {
			findings = append(findings, finding("tech_surface_root_missing",
				"A surface branch must point to an existing project-owned technology root.", branch.ID, branch.RootTechnologyID))
			continue
		}
		if !hasFolder(root, branch.Folder) {
			findings = append(findings, finding("tech_surface_root_folder_mismatch",
				"The declared root must be placed in the same certified folder as its generated gridbox.", branch.ID, branch.RootTechnologyID))
		}
		if hasIncomingPath(techs, branch.Folder, branch.RootTechnologyID) {
			findings = append(findings, finding("tech_surface_root_not_root",
				"The declared root is already reached by a path in this folder; only a path root gets its own gridbox.", branch.ID, branch.RootTechnologyID))
		}
		rootKey := branch.Folder + "\x00" + branch.RootTechnologyID
		if declaredRoots[rootKey] {
			findings = append(findings, finding("tech_surface_root_duplicate",
				"A root technology can declare only one gridbox in a given folder.", branch.ID, branch.RootTechnologyID))
		}
		declaredRoots[rootKey] = true
		for _, placement := range branchPlacements(branch)[1:] {
			alternateFolder, alternateOK := profile.Folders[placement.Folder]
			alternateRootKey := placement.Folder + "\x00" + branch.RootTechnologyID
			if declaredRoots[alternateRootKey] {
				findings = append(findings, finding("tech_surface_root_duplicate",
					"A root technology can declare only one gridbox in a given folder.", branch.ID, branch.RootTechnologyID))
			}
			declaredRoots[alternateRootKey] = true
			if !hasFolder(root, placement.Folder) {
				findings = append(findings, finding("tech_surface_root_folder_mismatch",
					"The declared root must be placed on every player-compatible research page generated for this branch.", branch.ID, branch.RootTechnologyID))
			}
			if hasIncomingPath(techs, placement.Folder, branch.RootTechnologyID) {
				findings = append(findings, finding("tech_surface_root_not_root",
					"The declared root is already reached by a path on a generated research page.", branch.ID, branch.RootTechnologyID))
			}
			if !alternateOK {
				continue
			}
			for memberIndex, memberID := range branch.TechnologyIDs {
				memberID = strings.TrimSpace(memberID)
				member, exists := technologies[memberID]
				if !exists || !isCreated(member.Source) {
					continue
				}
				if !hasFolder(member, placement.Folder) {
					findings = append(findings, finding("tech_surface_branch_member_folder_mismatch",
						"Every branch technology must be emitted on every generated research page.", branch.ID, memberID))
					continue
				}
				cell, cellOK := alternateFolder.TimelineCellAt(memberIndex)
				if !cellOK {
					findings = append(findings, finding("tech_surface_timeline_exhausted",
						"A compatible research page has no remaining certified timeline slot for this technology.", branch.ID, memberID))
					continue
				}
				if !hasTimelinePlacement(member, placement.Folder, cell) {
					findings = append(findings, finding("tech_surface_timeline_position_mismatch",
						"This branch technology must use the certified position on every generated research page.", branch.ID, memberID))
				}
				if memberID != branch.RootTechnologyID && !hasIncomingPathFromMembers(techs, placement.Folder, memberID, members) {
					findings = append(findings, finding("tech_surface_branch_member_unlinked",
						"Every non-root technology needs its in-branch visual path on every generated research page.", branch.ID, memberID))
				}
			}
		}

		for memberIndex, memberID := range branch.TechnologyIDs {
			memberID = strings.TrimSpace(memberID)
			if memberID == "" {
				continue
			}
			member, exists := technologies[memberID]
			if !exists || !isCreated(member.Source) {
				findings = append(findings, finding("tech_surface_branch_member_missing",
					"Every declared branch technology must be an existing project-owned technology.", branch.ID, memberID))
				continue
			}
			if !hasFolder(member, branch.Folder) {
				findings = append(findings, finding("tech_surface_branch_member_folder_mismatch",
					"Every branch technology must stay in its branch's certified folder.", branch.ID, memberID))
			}
			if !folderOK {
				continue
			}
			cell, cellOK := folder.TimelineCellAt(memberIndex)
			if !cellOK {
				findings = append(findings, finding("tech_surface_timeline_exhausted",
					"This certified research branch has no remaining original-game timeline slot for another technology.", branch.ID, memberID))
				continue
			}
			if member.StartYear != cell.StartYear {
				findings = append(findings, finding("tech_surface_timeline_year_mismatch",
					"This branch technology must use the research year assigned by its certified timeline slot.", branch.ID, memberID))
			}
			if !hasTimelinePlacement(member, branch.Folder, cell) {
				findings = append(findings, finding("tech_surface_timeline_position_mismatch",
					"This branch technology must use the grid position assigned by its certified timeline slot.", branch.ID, memberID))
			}
			for _, target := range member.PathTargets {
				if !members[strings.TrimSpace(target)] {
					findings = append(findings, finding("tech_surface_branch_path_external",
						"A P0 branch path may point only to a technology in the same declared branch.", branch.ID, memberID))
				}
			}
			if memberID == branch.RootTechnologyID {
				continue
			}
			if !hasIncomingPathFromMembers(techs, branch.Folder, memberID, members) {
				findings = append(findings, finding("tech_surface_branch_member_unlinked",
					"Every non-root branch technology needs an in-branch visual path from the preceding technology.", branch.ID, memberID))
			}
		}
	}
	for folder := range profile.Folders {
		for _, technology := range techs {
			techID := strings.TrimSpace(technology.ID)
			if !isCreated(technology.Source) || techID == "" || !hasFolder(technology, folder) {
				continue
			}
			if _, belongs := declaredMembers[techID]; !belongs {
				findings = append(findings, finding("tech_surface_branch_member_missing",
					"Project-owned technologies in a certified P0 folder must be added through a declared branch.", "", techID))
			}
		}
	}

	if len(manifest.Branches) == 0 && len(projectRoots(profile, techs)) > 0 {
		findings = append(findings, finding("tech_surface_branch_missing",
			"A technology surface needs one declared branch for every project-owned root technology.", "", ""))
	}

	for _, root := range projectRoots(profile, techs) {
		key := root.Folder + "\x00" + root.TechID
		if !declaredRoots[key] {
			findings = append(findings, finding("tech_surface_gridbox_missing",
				"This project-owned root technology has no generated <root>_tree gridbox, so it would not be visible in the game.", "", root.TechID))
		}
	}
	return findings
}

func hasTimelinePlacement(technology Technology, folder string, cell TimelineCell) bool {
	for _, placement := range technology.FolderPlacements {
		if strings.TrimSpace(placement.Folder) == folder &&
			placement.X == float64(cell.X) && placement.Y == float64(cell.Y) {
			return true
		}
	}
	return false
}

func hasIncomingPathFromMembers(techs []Technology, folder, target string, members map[string]bool) bool {
	for _, technology := range techs {
		if !members[strings.TrimSpace(technology.ID)] || !hasFolder(technology, folder) {
			continue
		}
		for _, pathTarget := range technology.PathTargets {
			if strings.TrimSpace(pathTarget) == target {
				return true
			}
		}
	}
	return false
}

func ValidateShape(manifest Manifest, isSubmod bool) []Finding {
	manifest = normalizeManifest(manifest)
	findings := []Finding{}
	if !IsEnabled(manifest) {
		return findings
	}
	if isSubmod {
		return append(findings, finding("tech_surface_submod_unsupported",
			"This technology surface profile is pinned to vanilla countrytechtreeview.gui and cannot safely patch a dependency's interface yet.", "", ""))
	}
	if manifest.SchemaVersion != SchemaVersion {
		findings = append(findings, finding("tech_surface_schema_invalid",
			fmt.Sprintf("Technology surface schema_version must be %d.", SchemaVersion), "", ""))
	}
	if manifest.CompatibilityMode != "" && manifest.CompatibilityMode != "all_players" && manifest.CompatibilityMode != "requires_dlc" {
		findings = append(findings, finding("tech_surface_compatibility_mode_invalid",
			"Choose either compatibility for all players or a Mod that requires the selected DLC.", "", ""))
	}
	profile, ok := ProfileByID(strings.TrimSpace(manifest.Profile))
	if !ok {
		return append(findings, finding("tech_surface_profile_unknown",
			"Choose a certified technology surface profile instead of a free-form GUI target.", "", ""))
	}
	if got := NormalizedSHA256(manifest.BaselineSHA256); !sha256RE.MatchString(got) || got != profile.BaselineSHA256 {
		findings = append(findings, finding("tech_surface_baseline_mismatch",
			"The surface manifest must be bound to the exact GUI baseline certified by its profile.", "", ""))
	}
	if len(manifest.Branches) > 32 {
		findings = append(findings, finding("tech_surface_branch_limit",
			"This P0 surface supports at most 32 declared root branches per project.", "", ""))
	}

	branchIDs := map[string]bool{}
	slots := map[string]bool{}
	for _, branch := range manifest.Branches {
		branch.ID = strings.TrimSpace(branch.ID)
		branch.Folder = strings.TrimSpace(branch.Folder)
		branch.Surface = strings.TrimSpace(branch.Surface)
		branch.Slot = strings.TrimSpace(branch.Slot)
		branch.RootTechnologyID = strings.TrimSpace(branch.RootTechnologyID)
		if len(branch.TechnologyIDs) == 0 || strings.TrimSpace(branch.TechnologyIDs[0]) != branch.RootTechnologyID {
			findings = append(findings, finding("tech_surface_root_not_first_member",
				"A surface branch must list its root as the first technology in its ordered chain.", branch.ID, branch.RootTechnologyID))
		}
		memberIDs := map[string]bool{}
		for _, memberID := range branch.TechnologyIDs {
			memberID = strings.TrimSpace(memberID)
			if !idRE.MatchString(memberID) || memberIDs[memberID] {
				findings = append(findings, finding("tech_surface_branch_member_invalid",
					"Each branch technology ID must be unique and contain only letters, numbers, and underscores.", branch.ID, memberID))
			}
			memberIDs[memberID] = true
		}
		if !idRE.MatchString(branch.ID) || branchIDs[branch.ID] {
			findings = append(findings, finding("tech_surface_branch_id_invalid",
				"Each surface branch needs a unique identifier containing only letters, numbers, and underscores.", branch.ID, ""))
		}
		branchIDs[branch.ID] = true
		placements := branchPlacements(branch)
		if len(placements) == 0 || placements[0].Folder != branch.Folder || placements[0].Slot != branch.Slot {
			findings = append(findings, finding("tech_surface_primary_placement_invalid",
				"The first generated research-page placement must match the branch's certified primary target.", branch.ID, branch.RootTechnologyID))
		}
		placementFolders := map[string]bool{}
		for _, placement := range placements {
			placement.Folder = strings.TrimSpace(placement.Folder)
			placement.Slot = strings.TrimSpace(placement.Slot)
			folder, folderOK := profile.Folders[placement.Folder]
			if !folderOK || folder.Surface != branch.Surface || placementFolders[placement.Folder] {
				findings = append(findings, finding("tech_surface_folder_not_certified",
					"Every generated research page must be a unique certified target under the branch entrance.", branch.ID, branch.RootTechnologyID))
				continue
			}
			placementFolders[placement.Folder] = true
			if _, slotOK := folder.Slots[placement.Slot]; !slotOK {
				findings = append(findings, finding("tech_surface_slot_not_certified",
					"Every generated research page must use its named certified slot; free coordinates are not supported.", branch.ID, branch.RootTechnologyID))
			}
			slotKey := placement.Folder + "\x00" + placement.Slot
			if slots[slotKey] {
				findings = append(findings, finding("tech_surface_slot_occupied",
					"Two root branches cannot occupy the same certified gridbox slot.", branch.ID, branch.RootTechnologyID))
			}
			slots[slotKey] = true
		}
		productionCompatibilityContract := strings.HasPrefix(manifest.Profile, "p0_prod_")
		if productionCompatibilityContract && manifest.CompatibilityMode == "requires_dlc" && len(placements) != 1 {
			findings = append(findings, finding("tech_surface_required_dlc_placement_invalid",
				"A Mod that requires selected DLC must export only its selected certified research page.", branch.ID, branch.RootTechnologyID))
		}
		if productionCompatibilityContract && manifest.CompatibilityMode == "all_players" {
			expectedTargets, err := techsurfaceprofiles.AllPlayerTargetsForFolder(branch.Surface, branch.Folder)
			if err != nil {
				findings = append(findings, finding("tech_surface_all_players_unsupported",
					"This research target cannot be exported for players with and without DLC.", branch.ID, branch.RootTechnologyID))
			} else {
				for _, expectedTarget := range expectedTargets {
					if !placementFolders[expectedTarget.FolderID] {
						findings = append(findings, finding("tech_surface_all_players_placement_missing",
							"Compatibility for all players requires the certified Base and relevant DLC research-page placements.", branch.ID, branch.RootTechnologyID))
					}
				}
				if len(placementFolders) != len(expectedTargets) {
					findings = append(findings, finding("tech_surface_all_players_placement_extra",
						"Compatibility for all players may emit only the certified Base and relevant DLC research pages.", branch.ID, branch.RootTechnologyID))
				}
			}
		}
	}
	return findings
}

func Compile(source []byte, manifest Manifest, techs []Technology, isSubmod bool) ([]byte, error) {
	if !IsEnabled(manifest) {
		return source, nil
	}
	if findings := Validate(manifest, techs, isSubmod); len(findings) > 0 {
		return nil, FindingsError{Findings: findings}
	}
	profile, _ := ProfileByID(manifest.Profile)
	if Hash(source) != profile.BaselineSHA256 {
		return nil, fmt.Errorf("tech surface GUI baseline hash differs from profile %q", profile.ID)
	}
	if _, recoveries, err := paradox.Parse(string(source)); err != nil || len(recoveries) > 0 {
		return nil, fmt.Errorf("tech surface source GUI is not a strict Paradox document")
	}

	byFolder := map[string][]Branch{}
	for _, branch := range manifest.Branches {
		for _, placement := range branchPlacements(branch) {
			placed := branch
			placed.Folder = placement.Folder
			placed.Slot = placement.Slot
			byFolder[placement.Folder] = append(byFolder[placement.Folder], placed)
		}
	}
	insertions := []insertion{}
	for folderName, branches := range byFolder {
		folder := profile.Folders[folderName]
		container, err := findNamedContainer(source, folderName)
		if err != nil {
			return nil, err
		}
		existing := directGridboxNames(source, container)
		sort.Slice(branches, func(i, j int) bool { return branches[i].ID < branches[j].ID })
		blocks := make([]string, 0, len(branches))
		for _, branch := range branches {
			gridName := branch.RootTechnologyID + "_tree"
			if existing[gridName] {
				return nil, fmt.Errorf("tech surface gridbox %q already exists in %q", gridName, folderName)
			}
			slot := folder.Slots[branch.Slot]
			block, err := emitGridbox(container.childIndent, gridName, slot, folder)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
		insertions = append(insertions, insertion{offset: container.closeOffset, text: "\n" + strings.Join(blocks, "\n\n") + "\n" + container.indent})
	}
	sort.Slice(insertions, func(i, j int) bool { return insertions[i].offset > insertions[j].offset })
	patched := append([]byte(nil), source...)
	for _, insert := range insertions {
		patched = append(patched[:insert.offset], append([]byte(insert.text), patched[insert.offset:]...)...)
	}
	if _, recoveries, err := paradox.Parse(string(patched)); err != nil || len(recoveries) > 0 {
		return nil, fmt.Errorf("generated technology surface GUI is not a strict Paradox document")
	}
	return patched, nil
}

type FindingsError struct{ Findings []Finding }

func (e FindingsError) Error() string {
	if len(e.Findings) == 0 {
		return "technology surface validation failed"
	}
	return e.Findings[0].Code + ": " + e.Findings[0].Message
}

type tokenKind uint8

const (
	tokenWord tokenKind = iota
	tokenEquals
	tokenLBrace
	tokenRBrace
)

type token struct {
	kind  tokenKind
	value string
	start int
	depth int
}

func lex(source []byte) []token {
	tokens := []token{}
	depth := 0
	for i := 0; i < len(source); {
		c := source[i]
		if c == '#' {
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		switch c {
		case '=':
			tokens = append(tokens, token{kind: tokenEquals, start: i, depth: depth})
			i++
			continue
		case '{':
			tokens = append(tokens, token{kind: tokenLBrace, start: i, depth: depth})
			depth++
			i++
			continue
		case '}':
			if depth > 0 {
				depth--
			}
			tokens = append(tokens, token{kind: tokenRBrace, start: i, depth: depth})
			i++
			continue
		case '"':
			start := i
			i++
			var value strings.Builder
			for i < len(source) {
				if source[i] == '\\' && i+1 < len(source) {
					value.WriteByte(source[i+1])
					i += 2
					continue
				}
				if source[i] == '"' {
					i++
					break
				}
				value.WriteByte(source[i])
				i++
			}
			tokens = append(tokens, token{kind: tokenWord, value: value.String(), start: start, depth: depth})
			continue
		}
		start := i
		for i < len(source) && !strings.ContainsRune(" \t\r\n#={}", rune(source[i])) {
			i++
		}
		if start != i {
			tokens = append(tokens, token{kind: tokenWord, value: string(source[start:i]), start: start, depth: depth})
		} else {
			i++
		}
	}
	return tokens
}

func matchingClose(tokens []token, open int) int {
	if open < 0 || open >= len(tokens) || tokens[open].kind != tokenLBrace {
		return -1
	}
	depth := 1
	for i := open + 1; i < len(tokens); i++ {
		switch tokens[i].kind {
		case tokenLBrace:
			depth++
		case tokenRBrace:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func nextNonComment(tokens []token, index int) int { return index + 1 }

type containerLocation struct {
	openOffset  int
	closeOffset int
	depth       int
	indent      string
	childIndent string
}

func findNamedContainer(source []byte, name string) (containerLocation, error) {
	tokens := lex(source)
	matches := []containerLocation{}
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].kind != tokenWord || !strings.EqualFold(tokens[i].value, "containerWindowType") || tokens[i+1].kind != tokenEquals || tokens[i+2].kind != tokenLBrace {
			continue
		}
		close := matchingClose(tokens, i+2)
		if close < 0 {
			return containerLocation{}, fmt.Errorf("unterminated GUI container")
		}
		baseDepth := tokens[i+2].depth
		for j := i + 3; j+2 < close; j++ {
			if tokens[j].depth != baseDepth+1 || tokens[j].kind != tokenWord || !strings.EqualFold(tokens[j].value, "name") || tokens[j+1].kind != tokenEquals || tokens[j+2].kind != tokenWord {
				continue
			}
			if tokens[j+2].value != name {
				continue
			}
			indent := lineIndent(source, tokens[close].start)
			matches = append(matches, containerLocation{openOffset: tokens[i+2].start, closeOffset: tokens[close].start, depth: baseDepth, indent: indent, childIndent: indent + indentUnit(source, tokens[i+2].start)})
			break
		}
	}
	if len(matches) == 0 {
		return containerLocation{}, fmt.Errorf("technology surface target container %q is absent from countrytechtreeview.gui", name)
	}
	if len(matches) > 1 {
		return containerLocation{}, fmt.Errorf("technology surface target container %q is ambiguous", name)
	}
	return matches[0], nil
}

func lineIndent(source []byte, offset int) string {
	start := offset
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	end := start
	for end < len(source) && (source[end] == ' ' || source[end] == '\t') {
		end++
	}
	return string(source[start:end])
}

func indentUnit(source []byte, offset int) string {
	lineStart := offset
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < len(source) && source[i] != '\n'; i++ {
		if source[i] == '\t' {
			return "\t"
		}
	}
	return "    "
}

func directGridboxNames(source []byte, container containerLocation) map[string]bool {
	tokens := lex(source)
	names := map[string]bool{}
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].start < container.openOffset || tokens[i].start > container.closeOffset || tokens[i].depth != container.depth+1 || tokens[i].kind != tokenWord || !strings.EqualFold(tokens[i].value, "gridboxtype") || tokens[i+1].kind != tokenEquals || tokens[i+2].kind != tokenLBrace {
			continue
		}
		close := matchingClose(tokens, i+2)
		if close < 0 || tokens[close].start > container.closeOffset {
			continue
		}
		baseDepth := tokens[i+2].depth
		for j := i + 3; j+2 < close; j++ {
			if tokens[j].depth == baseDepth+1 && tokens[j].kind == tokenWord && strings.EqualFold(tokens[j].value, "name") && tokens[j+1].kind == tokenEquals && tokens[j+2].kind == tokenWord {
				names[tokens[j+2].value] = true
				break
			}
		}
	}
	return names
}

func emitGridbox(indent, name string, slot Slot, folder FolderProfile) (string, error) {
	format := strings.ToUpper(strings.TrimSpace(folder.GridboxFormat))
	if format == "" {
		format = "LEFT"
	}
	if format != "LEFT" && format != "UP" {
		return "", fmt.Errorf("technology surface folder %q has unsupported gridbox format %q", folder.ID, folder.GridboxFormat)
	}
	width := folder.SlotWidth
	if width == 0 {
		width = 70
	}
	height := folder.SlotHeight
	if height == 0 {
		height = 70
	}
	if width < 1 || width > 500 || height < 1 || height > 500 {
		return "", fmt.Errorf("technology surface folder %q has invalid gridbox slot size %dx%d", folder.ID, width, height)
	}
	inner := indent + "\t"
	return indent + "gridboxtype = {\n" +
		inner + "name = \"" + name + "\"\n" +
		inner + fmt.Sprintf("position = { x = %d y = %d }\n", slot.X, slot.Y) +
		inner + fmt.Sprintf("slotsize = { width = %d height = %d }\n", width, height) +
		inner + "format = \"" + format + "\"\n" +
		indent + "}", nil
}

type insertion struct {
	offset int
	text   string
}
