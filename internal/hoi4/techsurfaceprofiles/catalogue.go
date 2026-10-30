package techsurfaceprofiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	CountryTechTreeGUIPath = "interface/countrytechtreeview.gui"
	TechnologyTagsPath     = "common/technology_tags/00_technology.txt"

	CountryTechTreeGUISHA256 = "fec641fb738ab59369cac40ae902fdba793bed7cbaa2c807756207a2e1f01045"
	TechnologyTagsSHA256     = "fccf0824abcf0d741f740704d92310c430b84d062a5b43e65119dcdb2896eeea"

	GridboxRootSuffix = "_tree"

	EntryArmy              = "army"
	EntryAir               = "air"
	EntryNavy              = "navy"
	EntryIndustryEquipment = "industry_equipment"

	DLCNoStepBack   DLC = "No Step Back"
	DLCByBloodAlone DLC = "By Blood Alone"
	DLCManTheGuns   DLC = "Man the Guns"
)

var idPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Environment struct {
	Source         SourceFingerprint `json:"source"`
	DLC            DLCState          `json:"dlc"`
	DLCFingerprint string            `json:"dlc_fingerprint"`
}

type SourceFingerprint struct {
	Files []SourceFile `json:"files"`
}

type SourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type DLC string

type DLCState struct {
	NoStepBack   bool `json:"no_step_back"`
	ByBloodAlone bool `json:"by_blood_alone"`
	ManTheGuns   bool `json:"man_the_guns"`
}

func (s DLCState) Active(dlc DLC) bool {
	switch dlc {
	case DLCNoStepBack:
		return s.NoStepBack
	case DLCByBloodAlone:
		return s.ByBloodAlone
	case DLCManTheGuns:
		return s.ManTheGuns
	default:
		return false
	}
}

func (s DLCState) Fingerprint() string {
	canonical := fmt.Sprintf("%s=%t\n%s=%t\n%s=%t\n",
		DLCNoStepBack, s.NoStepBack,
		DLCByBloodAlone, s.ByBloodAlone,
		DLCManTheGuns, s.ManTheGuns,
	)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

type EntryDefinition struct {
	ID           string        `json:"id"`
	Ledger       string        `json:"ledger"`
	TargetGroups []TargetGroup `json:"target_groups"`
}

type TargetGroup struct {
	ID       string   `json:"id"`
	Optional bool     `json:"optional,omitempty"`
	Variants []Target `json:"variants"`
}

type Target struct {
	ID               string         `json:"id"`
	VariantID        string         `json:"variant_id"`
	FolderID         string         `json:"folder_id"`
	ContainerID      string         `json:"container_id"`
	TechLine         string         `json:"tech_line"`
	GridboxNameRule  string         `json:"gridbox_name_rule"`
	GridboxFormat    string         `json:"gridbox_format"`
	SlotSize         Size           `json:"slot_size"`
	Slots            []Slot         `json:"slots"`
	Timeline         []TimelineCell `json:"timeline"`
	Condition        DLCCondition   `json:"condition"`
	AuthoringEnabled bool           `json:"authoring_enabled"`
	Certification    Certification  `json:"certification"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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

type DLCCondition struct {
	Requires []DLC `json:"requires"`
	Forbids  []DLC `json:"forbids"`
}

func (c DLCCondition) matches(state DLCState) bool {
	for _, dlc := range c.Requires {
		if !state.Active(dlc) {
			return false
		}
	}
	for _, dlc := range c.Forbids {
		if state.Active(dlc) {
			return false
		}
	}
	return true
}

type Certification struct {
	State      string `json:"state"`
	EvidenceID string `json:"evidence_id"`
	Detail     string `json:"detail,omitempty"`
}

type Profile struct {
	ID             string            `json:"id"`
	Source         SourceFingerprint `json:"source"`
	DLC            DLCState          `json:"dlc"`
	DLCFingerprint string            `json:"dlc_fingerprint"`
	Entries        []Entry           `json:"entries"`
}

type Entry struct {
	ID      string   `json:"id"`
	Ledger  string   `json:"ledger"`
	Targets []Target `json:"targets"`
}

type ErrorCode string

const (
	CodeSourceMissing     ErrorCode = "tech_profile_source_missing"
	CodeSourceDrift       ErrorCode = "tech_profile_source_drift"
	CodeDLCFingerprint    ErrorCode = "tech_profile_dlc_fingerprint_invalid"
	CodeCatalogueInvalid  ErrorCode = "tech_profile_catalogue_invalid"
	CodeTargetUnknown     ErrorCode = "tech_profile_target_unknown"
	CodeTargetUncertified ErrorCode = "tech_profile_target_uncertified"
)

type SelectionError struct {
	Code   ErrorCode
	Detail string
}

func (e SelectionError) Error() string { return string(e.Code) + ": " + e.Detail }

func ErrorCodeOf(err error) ErrorCode {
	if typed, ok := err.(SelectionError); ok {
		return typed.Code
	}
	return ""
}

func AuditedSourceFingerprint() SourceFingerprint {
	return SourceFingerprint{Files: []SourceFile{
		{Path: CountryTechTreeGUIPath, SHA256: CountryTechTreeGUISHA256},
		{Path: TechnologyTagsPath, SHA256: TechnologyTagsSHA256},
	}}
}

func NewEnvironment(source SourceFingerprint, dlc DLCState) Environment {
	return Environment{Source: cloneSource(source), DLC: dlc, DLCFingerprint: dlc.Fingerprint()}
}

func ProductionEntries() []EntryDefinition {
	return cloneDefinitions(catalogue)
}

func Resolve(environment Environment) (Profile, error) {
	if err := validateCatalogue(catalogue); err != nil {
		return Profile{}, SelectionError{Code: CodeCatalogueInvalid, Detail: err.Error()}
	}
	if err := verifySource(environment.Source); err != nil {
		return Profile{}, err
	}
	wantDLC := environment.DLC.Fingerprint()
	if environment.DLCFingerprint == "" || !strings.EqualFold(strings.TrimSpace(environment.DLCFingerprint), wantDLC) {
		return Profile{}, SelectionError{Code: CodeDLCFingerprint, Detail: "re-scan the active DLC set before selecting a technology surface profile"}
	}

	profile := Profile{
		ID:             profileID(environment.DLC),
		Source:         AuditedSourceFingerprint(),
		DLC:            environment.DLC,
		DLCFingerprint: wantDLC,
		Entries:        make([]Entry, 0, len(catalogue)),
	}
	for _, definition := range catalogue {
		entry := Entry{ID: definition.ID, Ledger: definition.Ledger, Targets: []Target{}}
		for _, group := range definition.TargetGroups {
			for _, variant := range group.Variants {
				if variant.Condition.matches(environment.DLC) {
					entry.Targets = append(entry.Targets, cloneTarget(variant))
					break
				}
			}
		}
		sort.Slice(entry.Targets, func(i, j int) bool { return entry.Targets[i].ID < entry.Targets[j].ID })
		profile.Entries = append(profile.Entries, entry)
	}
	return profile, nil
}

func (p Profile) AuthoringTarget(entryID, targetID string) (Target, error) {
	for _, entry := range p.Entries {
		if entry.ID != entryID {
			continue
		}
		for _, target := range entry.Targets {
			if target.ID != targetID {
				continue
			}
			if !target.AuthoringEnabled {
				return Target{}, SelectionError{Code: CodeTargetUncertified, Detail: fmt.Sprintf("%s/%s has no engine-certified branch slot", entryID, targetID)}
			}
			return cloneTarget(target), nil
		}
	}
	return Target{}, SelectionError{Code: CodeTargetUnknown, Detail: fmt.Sprintf("unknown technology surface target %s/%s", entryID, targetID)}
}

func AllPlayerTargets(entryID, targetID string) ([]Target, error) {
	for _, entry := range catalogue {
		if entry.ID != entryID {
			continue
		}
		for _, group := range entry.TargetGroups {
			if group.ID != targetID {
				continue
			}
			result := []Target{}
			hasBase := false
			for _, variant := range group.Variants {
				if len(variant.Condition.Requires) == 0 {
					hasBase = true
				}
				if !variant.AuthoringEnabled || variant.Certification.State != "engine_verified_geometry" {
					return nil, SelectionError{Code: CodeTargetUncertified, Detail: fmt.Sprintf("%s/%s does not have certified geometry for every player-owned DLC variant", entryID, targetID)}
				}
				result = append(result, cloneTarget(variant))
			}
			if !hasBase {
				return nil, SelectionError{Code: CodeTargetUncertified, Detail: fmt.Sprintf("%s/%s exists only on a DLC research page", entryID, targetID)}
			}
			return result, nil
		}
	}
	return nil, SelectionError{Code: CodeTargetUnknown, Detail: fmt.Sprintf("unknown technology surface target %s/%s", entryID, targetID)}
}

func AllPlayerTargetsForFolder(entryID, folderID string) ([]Target, error) {
	for _, entry := range catalogue {
		if entry.ID != entryID {
			continue
		}
		for _, group := range entry.TargetGroups {
			for _, variant := range group.Variants {
				if variant.FolderID == folderID {
					return AllPlayerTargets(entryID, group.ID)
				}
			}
		}
	}
	return nil, SelectionError{Code: CodeTargetUnknown, Detail: fmt.Sprintf("unknown technology surface folder %s/%s", entryID, folderID)}
}

func (t Target) GridboxName(rootTechnologyID string) (string, error) {
	rootTechnologyID = strings.TrimSpace(rootTechnologyID)
	if !idPattern.MatchString(rootTechnologyID) {
		return "", fmt.Errorf("invalid root technology id %q", rootTechnologyID)
	}
	return rootTechnologyID + GridboxRootSuffix, nil
}

func profileID(state DLCState) string {
	return fmt.Sprintf("p0_prod_nsb%d_bba%d_mtg%d_fec641fb", boolInt(state.NoStepBack), boolInt(state.ByBloodAlone), boolInt(state.ManTheGuns))
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func verifySource(source SourceFingerprint) error {
	required := AuditedSourceFingerprint().Files
	if len(source.Files) != len(required) {
		return SelectionError{Code: CodeSourceMissing, Detail: "both audited GUI and technology_tags fingerprints are required"}
	}
	seen := make(map[string]string, len(source.Files))
	for _, file := range source.Files {
		path := normalizedPath(file.Path)
		if path == "" || seen[path] != "" {
			return SelectionError{Code: CodeSourceMissing, Detail: "source fingerprint paths must be present and unique"}
		}
		seen[path] = strings.ToLower(strings.TrimSpace(file.SHA256))
	}
	for _, file := range required {
		got, ok := seen[normalizedPath(file.Path)]
		if !ok {
			return SelectionError{Code: CodeSourceMissing, Detail: fmt.Sprintf("missing source fingerprint for %s", file.Path)}
		}
		if got != file.SHA256 {
			return SelectionError{Code: CodeSourceDrift, Detail: fmt.Sprintf("source fingerprint drifted for %s", file.Path)}
		}
	}
	return nil
}

func normalizedPath(path string) string {
	return strings.ToLower(strings.Trim(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/"))
}

func validateCatalogue(definitions []EntryDefinition) error {
	if len(definitions) != 4 {
		return fmt.Errorf("entry count=%d, want 4", len(definitions))
	}
	entryIDs := map[string]bool{}
	for _, entry := range definitions {
		if entry.ID == "" || entryIDs[entry.ID] {
			return fmt.Errorf("entry id %q is empty or duplicated", entry.ID)
		}
		entryIDs[entry.ID] = true
		for _, group := range entry.TargetGroups {
			if group.ID == "" || len(group.Variants) == 0 {
				return fmt.Errorf("entry %s has an empty target group", entry.ID)
			}
			for _, target := range group.Variants {
				if target.ID == "" || target.FolderID == "" || target.ContainerID != target.FolderID || target.GridboxNameRule != "root_technology_id"+GridboxRootSuffix {
					return fmt.Errorf("target %s/%s has an invalid fixed surface contract", entry.ID, target.ID)
				}
				if target.AuthoringEnabled && (len(target.Slots) == 0 || len(target.Timeline) == 0 || target.SlotSize.Width <= 0 || target.SlotSize.Height <= 0) {
					return fmt.Errorf("authoring target %s/%s lacks certified geometry", entry.ID, target.ID)
				}
			}
			for mask := 0; mask < 8; mask++ {
				state := DLCState{NoStepBack: mask&1 != 0, ByBloodAlone: mask&2 != 0, ManTheGuns: mask&4 != 0}
				matches := 0
				for _, target := range group.Variants {
					if target.Condition.matches(state) {
						matches++
					}
				}
				if matches > 1 || (matches == 0 && !group.Optional) {
					return fmt.Errorf("target group %s/%s has %d variants for DLC mask %d", entry.ID, group.ID, matches, mask)
				}
			}
		}
	}
	return nil
}

func cloneSource(source SourceFingerprint) SourceFingerprint {
	return SourceFingerprint{Files: append([]SourceFile(nil), source.Files...)}
}

func cloneDefinitions(in []EntryDefinition) []EntryDefinition {
	out := make([]EntryDefinition, len(in))
	for i, entry := range in {
		out[i] = EntryDefinition{ID: entry.ID, Ledger: entry.Ledger, TargetGroups: make([]TargetGroup, len(entry.TargetGroups))}
		for j, group := range entry.TargetGroups {
			out[i].TargetGroups[j] = TargetGroup{ID: group.ID, Optional: group.Optional, Variants: make([]Target, len(group.Variants))}
			for k, variant := range group.Variants {
				out[i].TargetGroups[j].Variants[k] = cloneTarget(variant)
			}
		}
	}
	return out
}

func cloneTarget(in Target) Target {
	in.Slots = append([]Slot(nil), in.Slots...)
	in.Timeline = append([]TimelineCell(nil), in.Timeline...)
	in.Condition.Requires = append([]DLC(nil), in.Condition.Requires...)
	in.Condition.Forbids = append([]DLC(nil), in.Condition.Forbids...)
	return in
}

var catalogue = []EntryDefinition{
	{
		ID: EntryArmy, Ledger: "army",
		TargetGroups: []TargetGroup{
			{ID: "infantry", Variants: []Target{infantryTarget()}},
			{ID: "armour", Variants: []Target{baseArmourTarget(), nsbArmourTarget()}},
		},
	},
	{
		ID: EntryAir, Ledger: "air",
		TargetGroups: []TargetGroup{{ID: "airframes", Variants: []Target{baseAirTarget(), bbaAirTarget()}}},
	},
	{
		ID: EntryNavy, Ledger: "navy",
		TargetGroups: []TargetGroup{
			{ID: "naval_main", Variants: []Target{baseNavalTarget(), mtgNavalMainTarget()}},
			{ID: "naval_support", Optional: true, Variants: []Target{mtgNavalSupportTarget()}},
		},
	},
	{
		ID: EntryIndustryEquipment, Ledger: "civilian",
		TargetGroups: []TargetGroup{
			{ID: "industry", Variants: []Target{industryTarget()}},
			{ID: "electronics", Variants: []Target{electronicsTarget()}},
		},
	},
}

func certified(evidence string) Certification {
	return Certification{State: "engine_verified_geometry", EvidenceID: evidence}
}

func pending(detail string) Certification {
	return Certification{State: "source_locked_pending_engine", EvidenceID: "countrytechtreeview.gui@fec641fb", Detail: detail}
}

func target(id, variant, folder, line, format string, width, height int, slot Slot, timeline []TimelineCell, condition DLCCondition, enabled bool, certification Certification) Target {
	target := Target{
		ID: id, VariantID: variant, FolderID: folder, ContainerID: folder, TechLine: line,
		GridboxNameRule: "root_technology_id" + GridboxRootSuffix,
		GridboxFormat:   format, SlotSize: Size{Width: width, Height: height},
		Timeline: timeline, Condition: condition, AuthoringEnabled: enabled, Certification: certification,
	}
	if slot.ID != "" {
		target.Slots = []Slot{slot}
	}
	return target
}

func infantryTarget() Target {
	return target("infantry", "base", "infantry_folder", "infantry", "LEFT", 70, 70,
		Slot{ID: "p0_infantry_extension_1", X: 140, Y: 1225},
		[]TimelineCell{{StartYear: 1936, X: 0, Y: 2}, {StartYear: 1939, X: 0, Y: 6}, {StartYear: 1943, X: 0, Y: 14}},
		DLCCondition{}, true, certified("hoi4-engine/infantry/base"))
}

func baseArmourTarget() Target {
	return target("armour", "base", "armour_folder", "armor", "UP", 70, 70, Slot{}, nil,
		DLCCondition{Forbids: []DLC{DLCNoStepBack}}, false, pending("No engine-certified P0 branch slot exists for the base armour canvas."))
}

func nsbArmourTarget() Target {
	return target("armour", "nsb", "nsb_armour_folder", "armor", "UPPER_LEFT", 70, 70, Slot{}, nil,
		DLCCondition{Requires: []DLC{DLCNoStepBack}}, false, pending("NSB replaces armour_folder and uses UPPER_LEFT; no P0 engine-certified branch slot exists yet."))
}

func baseAirTarget() Target {
	return target("airframes", "base", "air_techs_folder", "air", "UP", 60, 70,
		Slot{ID: "p0_base_air_slot", X: 2240, Y: 32},
		[]TimelineCell{{StartYear: 1933, X: 0, Y: 2}, {StartYear: 1940, X: 0, Y: 6}},
		DLCCondition{Forbids: []DLC{DLCByBloodAlone}}, true, certified("hoi4-engine/airframes/base"))
}

func bbaAirTarget() Target {
	return target("airframes", "bba", "bba_air_techs_folder", "air", "UP", 70, 70,
		Slot{ID: "p0_dlc_air_slot", X: 2100, Y: 32},
		[]TimelineCell{{StartYear: 1933, X: 0, Y: 2}, {StartYear: 1940, X: 0, Y: 6}},
		DLCCondition{Requires: []DLC{DLCByBloodAlone}}, true, certified("hoi4-engine/airframes/bba"))
}

func baseNavalTarget() Target {
	return target("naval_main", "base", "naval_folder", "naval", "LEFT", 70, 70,
		Slot{ID: "p0_base_naval_slot", X: 150, Y: 400},
		[]TimelineCell{{StartYear: 1936, X: 0, Y: 7}, {StartYear: 1940, X: 0, Y: 13}},
		DLCCondition{Forbids: []DLC{DLCManTheGuns}}, true, certified("hoi4-engine/naval/base"))
}

func mtgNavalMainTarget() Target {
	return target("naval_main", "mtg", "mtgnavalfolder", "naval", "LEFT", 70, 70,
		Slot{ID: "p0_dlc_naval_main_slot", X: 70, Y: 1160},
		[]TimelineCell{{StartYear: 1936, X: 0, Y: 8}, {StartYear: 1939, X: 0, Y: 14}},
		DLCCondition{Requires: []DLC{DLCManTheGuns}}, true, certified("hoi4-engine/naval/mtg-main"))
}

func mtgNavalSupportTarget() Target {
	return target("naval_support", "mtg", "mtgnavalsupportfolder", "naval", "LEFT", 70, 70,
		Slot{ID: "p0_dlc_naval_support_slot", X: 150, Y: 1170},
		[]TimelineCell{{StartYear: 1922, X: 0, Y: 1}, {StartYear: 1936, X: 0, Y: 5}},
		DLCCondition{Requires: []DLC{DLCManTheGuns}}, true, certified("hoi4-engine/naval/mtg-support"))
}

func industryTarget() Target {
	return target("industry", "base", "industry_folder", "industry", "UP", 70, 70,
		Slot{ID: "p0_base_industry_slot", X: 1350, Y: 172},
		[]TimelineCell{{StartYear: 1936, X: 0, Y: 0}, {StartYear: 1937, X: 0, Y: 4}},
		DLCCondition{}, true, certified("hoi4-engine/industry/base"))
}

func electronicsTarget() Target {
	return target("electronics", "base", "electronics_folder", "electronics", "UP", 60, 60,
		Slot{ID: "p0_base_electronics_slot", X: 1550, Y: 172},
		[]TimelineCell{{StartYear: 1936, X: -1, Y: 0}, {StartYear: 1938, X: -1, Y: 6}},
		DLCCondition{}, true, certified("hoi4-engine/electronics/base"))
}
