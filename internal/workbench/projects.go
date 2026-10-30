package workbench

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CreateInput struct {
	Name                  string            `json:"name"`
	CountryTag            string            `json:"country_tag"`
	CountryName           string            `json:"country_name"`
	CountryTags           []string          `json:"country_tags"`
	IdeologyIconPaths     map[string]string `json:"ideology_icon_paths"`
	Version               *string           `json:"version"`
	Picture               string            `json:"picture"`
	Tags                  []string          `json:"tags"`
	Dependencies          []string          `json:"dependencies"`
	SupportedVersion      *string           `json:"supported_version"`
	ImportedReplacePaths  []string          `json:"imported_descriptor_replace_paths"`
	TemplateID            *int64            `json:"template_id"`
	Loaded                bool              `json:"is_loaded_existing_mod"`
	StartDate             *string           `json:"start_date"`
	EndDate               *string           `json:"end_date"`
	ScenarioHistoryPolicy *string           `json:"scenario_history_policy"`
	StageStateIDs         []int64           `json:"stage_state_ids"`
	ExportLanguages       *[]string         `json:"export_languages"`

	OwnedDLCs           *[]string      `json:"owned_dlcs"`
	IdeologyRegistry    map[string]any `json:"ideology_registry"`
	TechSurfaceManifest map[string]any `json:"tech_surface_manifest"`

	stageJSON *string
}

func (in *CreateInput) Normalize() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return fmt.Errorf("Project name is required")
	}
	if in.Version == nil {
		v := "1.0"
		in.Version = &v
	}
	if in.SupportedVersion == nil {
		v := "1.19.*"
		in.SupportedVersion = &v
	}
	if in.Tags == nil {
		in.Tags = []string{"Alternative History"}
	}
	if in.CountryTags == nil {
		in.CountryTags = []string{}
	}
	if in.Dependencies == nil {
		in.Dependencies = []string{}
	}
	if in.IdeologyIconPaths == nil {
		in.IdeologyIconPaths = map[string]string{}
	}
	if in.IdeologyRegistry == nil {
		in.IdeologyRegistry = map[string]any{}
	}
	if in.TechSurfaceManifest == nil {
		in.TechSurfaceManifest = map[string]any{}
	}
	in.ImportedReplacePaths = normalizeReplacePaths(in.ImportedReplacePaths)
	if in.ExportLanguages == nil {
		in.ExportLanguages = &[]string{"en"}
	} else {
		langs, err := normalizeExportLanguages(*in.ExportLanguages)
		if err != nil {
			return err
		}
		in.ExportLanguages = &langs
	}
	owned, err := NormalizeOwnedDLCs(in.OwnedDLCs)
	if err != nil {
		return err
	}
	in.OwnedDLCs = owned
	if in.StartDate != nil {
		v, err := validateStartDate(*in.StartDate)
		if err != nil {
			return err
		}
		in.StartDate = v
	}
	if in.EndDate != nil {
		v, err := validateEndDate(*in.EndDate)
		if err != nil {
			return err
		}
		in.EndDate = v
	}
	if in.ScenarioHistoryPolicy == nil {
		v := ScenarioHistoryPolicyInheritBaseHistory
		in.ScenarioHistoryPolicy = &v
	} else if err := validateScenarioHistoryPolicy(*in.ScenarioHistoryPolicy); err != nil {
		return err
	}
	ids, err := normalizeStageStateIDs(in.StageStateIDs)
	if err != nil {
		return err
	}
	if ids != nil {
		b, _ := json.Marshal(ids)
		s := string(b)
		in.stageJSON = &s
	}
	return nil
}

type countrySeed struct{ Tag, Name, NameEN, NameZH string }

func countrySeedPlan(in CreateInput) (string, []countrySeed) {
	if in.CountryTag == "" && len(in.CountryTags) == 0 {
		return "", nil
	}
	tags := append([]string(nil), in.CountryTags...)
	if len(tags) == 0 {
		tags = []string{in.CountryTag}
	}
	if in.CountryTag != "" {
		found := false
		for _, t := range tags {
			found = found || t == in.CountryTag
		}
		if !found {
			tags = append([]string{in.CountryTag}, tags...)
		}
	}
	primary := in.CountryTag
	if primary == "" {
		primary = tags[0]
	}
	seeds := make([]countrySeed, 0, len(tags))
	for _, tag := range tags {
		display := tag
		if tag == primary {
			display = in.CountryName
		}
		en, zh := splitLocName(display, tag)
		seeds = append(seeds, countrySeed{Tag: tag, Name: display, NameEN: en, NameZH: zh})
	}
	return primary, seeds
}

func splitLocName(displayName, tag string) (string, string) {
	if displayName == "" {
		return "", ""
	}
	ascii := true
	for _, r := range displayName {
		if r > 127 {
			ascii = false
			break
		}
	}
	if ascii {
		return displayName, ""
	}
	if tag != "" {
		return strings.ToUpper(tag), displayName
	}
	return "", displayName
}
