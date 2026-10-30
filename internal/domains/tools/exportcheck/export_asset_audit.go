package exportcheck

import (
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	emitpreview "haganeworks/internal/hoi4/emit/preview"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
	"regexp"
	"strings"
)

func ideaPictureFilename(countryTag, ideaID string) string {
	return "idea_" + strings.ToUpper(countryTag) + "_" + emitmisc.Slugify(ideaID) + ".png"
}

func eventPictureFilename(countryTag, eventID string) string {
	return "event_" + strings.ToUpper(countryTag) + "_" + emitmisc.Slugify(eventID) + ".png"
}

func decisionIconFilename(decisionID string) string {
	return "decision_" + emitmisc.Slugify(decisionID) + ".png"
}

func techIconFilename(techID string) string {
	return emitmisc.Slugify(techID) + ".png"
}

func focusIconSpriteName(focusID string) string { return emitmisc.FocusIconSpriteName(focusID) }

func portraitFilename(countryTag, slug string) string {
	return emitscripts.PortraitFilename(countryTag, slug)
}

func extractPortraitSlug(characterID, countryTag string) string {
	return emitpreview.ExtractSlug(characterID, countryTag)
}

type AssetAudit struct {
	IdeaPicturesMissing  int    `json:"idea_pictures_missing"`
	EventPicturesMissing int    `json:"event_pictures_missing"`
	DecisionIconsMissing int    `json:"decision_icons_missing"`
	TechIconsMissing     int    `json:"tech_icons_missing"`
	FocusIconsMissing    int    `json:"focus_icons_missing"`
	PortraitsMissing     int    `json:"portraits_missing"`
	TotalMissing         int    `json:"total_missing"`
	Error                string `json:"_error,omitempty"`
}

type AssetAuditEntity struct {
	ID         int64
	Source     string
	CountryTag string

	Picture       string
	Icon          string
	IdeaID        string
	EventID       string
	DecisionID    string
	TechID        string
	Tag           string
	LargePortrait string
	FocusID       string
}

type AssetAuditTree struct {
	Source string
	Nodes  []AssetAuditEntity
}

type AssetAuditInput struct {
	ProjectTag      string
	HasDependencies bool
	IncludeImported bool
	Characters      []AssetAuditEntity
	Trees           []AssetAuditTree
	Techs           []AssetAuditEntity
	Ideas           []AssetAuditEntity
	Events          []AssetAuditEntity
	Decisions       []AssetAuditEntity
}

type AssetResolver func(kind string, entityID int64, spriteOrFile string) bool

type VanillaNameSet func(name string) bool

func createdOnly(rows []AssetAuditEntity) []AssetAuditEntity {
	out := make([]AssetAuditEntity, 0, len(rows))
	for _, r := range rows {
		if r.Source == "" || r.Source == "created" {
			out = append(out, r)
		}
	}
	return out
}

func AuditProjectAssets(
	in AssetAuditInput, resolve AssetResolver,
	vanillaIdeaPictures, vanillaFocusIcons VanillaNameSet,
) AssetAudit {
	audit := AssetAudit{}
	characters, techs, ideas := in.Characters, in.Techs, in.Ideas
	events, decisions := in.Events, in.Decisions
	trees := in.Trees

	if !in.IncludeImported {
		characters, techs, ideas = createdOnly(characters), createdOnly(techs), createdOnly(ideas)
		events, decisions = createdOnly(events), createdOnly(decisions)
		kept := make([]AssetAuditTree, 0, len(trees))
		for _, t := range trees {
			if t.Source == "" || t.Source == "created" {
				kept = append(kept, t)
			}
		}
		trees = kept
	}

	for _, idea := range ideas {
		raw := strings.TrimSpace(idea.Picture)
		if raw == "" {
			continue
		}
		stripped := strings.TrimPrefix(raw, "GFX_idea_")
		if stripped != "" && vanillaIdeaPictures(stripped) {
			continue
		}
		if in.HasDependencies {
			continue
		}
		ctag := idea.CountryTag
		if ctag == "" {
			ctag = in.ProjectTag
		}
		if !resolve("idea_picture", idea.ID, ideaPictureFilename(ctag, idea.IdeaID)) {
			audit.IdeaPicturesMissing++
		}
	}

	for _, ev := range events {
		raw := strings.TrimSpace(ev.Picture)
		if raw == "" || !strings.HasPrefix(raw, "GFX_") {
			continue
		}
		if in.HasDependencies {
			continue
		}
		ctag := ev.CountryTag
		if ctag == "" {
			ctag = in.ProjectTag
		}
		if !resolve("event_picture", ev.ID, eventPictureFilename(ctag, ev.EventID)) {
			audit.EventPicturesMissing++
		}
	}

	for _, d := range decisions {
		if strings.TrimSpace(d.Icon) == "" {
			continue
		}
		if !resolve("decision_icon", d.ID, decisionIconFilename(d.DecisionID)) {
			audit.DecisionIconsMissing++
		}
	}
	for _, t := range techs {
		if strings.TrimSpace(t.TechID) == "" {
			continue
		}
		if !resolve("tech_icon", t.ID, techIconFilename(t.TechID)) {
			audit.TechIconsMissing++
		}
	}

	seenFocus := map[string]bool{}
	for _, tree := range trees {
		for _, node := range tree.Nodes {
			if node.FocusID == "" {
				continue
			}
			sprite := focusIconSpriteName(node.FocusID)
			if seenFocus[sprite] {
				continue
			}
			seenFocus[sprite] = true

			nodeIcon := strings.TrimSpace(node.Icon)
			if nodeIcon != "" && vanillaFocusIcons(nodeIcon) {
				continue
			}
			if in.HasDependencies {
				continue
			}
			if !resolve("focus_icon", 0, sprite) {
				audit.FocusIconsMissing++
			}
		}
	}

	for _, char := range characters {
		ref := strings.TrimSpace(char.LargePortrait)

		if ref == "" || strings.HasPrefix(ref, "GFX_") || strings.HasPrefix(ref, "gfx/") {
			continue
		}
		if in.HasDependencies {
			continue
		}
		ctag := char.CountryTag
		if ctag == "" {
			ctag = in.ProjectTag
		}
		if !resolve("character_portrait", char.ID,
			portraitFilename(ctag, extractPortraitSlug(char.Tag, ctag))) {
			audit.PortraitsMissing++
		}
	}

	audit.TotalMissing = audit.IdeaPicturesMissing + audit.EventPicturesMissing +
		audit.DecisionIconsMissing + audit.TechIconsMissing +
		audit.FocusIconsMissing + audit.PortraitsMissing
	return audit
}

var nonContentTables = map[string]bool{
	"entity_obs_paths": true, "project_obs_assets": true, "project_obs_icons": true,
	"project_file_baselines": true, "project_overlay_state": true,
}

var countKeyAliases = map[string]string{
	"wargoal_types": "wargoals", "project_raw_files": "raw_files",
	"equipments": "equipment",
}

var sqlIdentifierRE = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
