package exportcheck

import (
	"strings"
)

type ExportIssue = map[string]any

type GuardrailProject struct {
	ID           int64
	CountryTag   string
	Dependencies []string
}

func (p GuardrailProject) isSubmod() bool { return len(p.Dependencies) > 0 }

type GuardrailCharacter struct {
	Tag            string
	Source         string
	Ideology       string
	RoleType       string
	CountryTag     string
	HasAdvisorRole bool
}

type GuardrailFocusNode struct{ FocusID string }

type GuardrailTree struct{ Nodes []GuardrailFocusNode }

type GuardrailIdea struct {
	IdeaID string
	Source string
}

type GuardrailDecision struct {
	DecisionID string
	Source     string
}

type GuardrailHistoryFile struct {
	CountryTag             string
	Source                 string
	SetPoliticsRulingParty string
}

type GuardrailDivisionTemplate struct {
	CountryTag   string
	Source       string
	TemplateName string
	OOBFile      string
}

type GuardrailTech struct {
	TechID string
	Folder string
}

type GuardrailDoctrine struct {
	EntityType string
	EntityKey  string
	TrackKey   string
}

type BaseIndexLookup func(entityType string) map[string]bool

func issueRef(kind, id string) map[string]any {
	return map[string]any{"type": kind, "id": id}
}

func notImported(source string) bool { return source != "imported" }

func GuardrailBaseCollision(
	project GuardrailProject, characters []GuardrailCharacter,
	trees []GuardrailTree, ideas []GuardrailIdea, decisions []GuardrailDecision,
	baseIndex BaseIndexLookup,
) []ExportIssue {
	out := []ExportIssue{}
	if !project.isSubmod() {
		return out
	}

	type source struct {
		entityType string
		label      string
		ids        []string
	}
	characterIDs := []string{}
	for _, c := range characters {
		if notImported(c.Source) {
			characterIDs = append(characterIDs, c.Tag)
		}
	}

	focusIDs := []string{}
	for _, t := range trees {
		for _, n := range t.Nodes {
			if n.FocusID != "" {
				focusIDs = append(focusIDs, n.FocusID)
			}
		}
	}
	ideaIDs := []string{}
	for _, i := range ideas {
		if notImported(i.Source) {
			ideaIDs = append(ideaIDs, i.IdeaID)
		}
	}
	decisionIDs := []string{}
	for _, d := range decisions {
		if notImported(d.Source) {
			decisionIDs = append(decisionIDs, d.DecisionID)
		}
	}
	for _, s := range []source{
		{"character", "character", characterIDs},
		{"focus", "focus", focusIDs},
		{"idea", "idea", ideaIDs},
		{"decision", "decision", decisionIDs},
	} {
		baseIDs := baseIndex(s.entityType)
		if len(baseIDs) == 0 {
			continue
		}
		for _, id := range s.ids {
			if id == "" || !baseIDs[id] {
				continue
			}
			out = append(out, ExportIssue{
				"level": "error", "code": "base_mod_id_collision",
				"entity_ref": issueRef(s.label, id), "severity": "blocker",
				"fix_hint": "Rename your " + s.label + " to a new, unique id, or reference the " +
					"base mod's existing '" + id + "' instead of re-creating it.",
				"message": "The " + s.label + " id '" + id + "' already exists in the base mod you " +
					"depend on — exporting it re-defines the base entity and HOI4 " +
					"crashes on load (duplicate id / 'multiple ... have the tag'). " +
					"Rename it, or reference the base one instead.",
			})
		}
	}
	return out
}

func GuardrailSubmodIdeology(
	project GuardrailProject, characters []GuardrailCharacter,
	historyFiles []GuardrailHistoryFile, isSubmod bool, baseIndex BaseIndexLookup,
) []ExportIssue {
	out := []ExportIssue{}
	if !isSubmod {
		return out
	}
	baseIdeo := baseIndex("ideology")
	if len(baseIdeo) == 0 {
		return out
	}
	known := map[string]bool{}
	for id := range vanillaSubIdeologyIDs() {
		known[id] = true
	}
	for id := range vanillaIdeologyGroupIDs() {
		known[id] = true
	}
	for id := range baseIdeo {
		known[id] = true
	}
	warned := map[string]bool{}
	check := func(value, who string, ref map[string]any) {
		x := strings.TrimSpace(value)
		if x == "" || known[x] || warned[x] {
			return
		}
		warned[x] = true
		out = append(out, ExportIssue{
			"level": "error", "code": "submod_ideology_undefined",
			"entity_ref": ref, "severity": "blocker",
			"fix_hint": "Reference a base-mod or vanilla ideology, or promote your submod " +
				"to a standalone mod that defines '" + x + "' in common/ideologies.",
			"message": "Custom ideology '" + x + "' is referenced by " + who + ", but a submod does not " +
				"emit common/ideologies (it inherits the base mod's) and neither the " +
				"base mod nor vanilla defines '" + x + "' — HOI4 crashes on load with an " +
				"undefined ideology. Use a base-mod ideology or make a standalone mod.",
		})
	}
	for _, c := range characters {
		check(c.Ideology, "character "+c.Tag, issueRef("character", c.Tag))
	}
	for _, hf := range historyFiles {
		check(hf.SetPoliticsRulingParty, hf.CountryTag+" ruling_party",
			issueRef("history_file", hf.CountryTag))
	}
	return out
}

func GuardrailBaseTagWrites(
	project GuardrailProject, historyFiles []GuardrailHistoryFile,
	divisionTemplates []GuardrailDivisionTemplate, isSubmod bool,
	baseIndex BaseIndexLookup,
) []ExportIssue {
	out := []ExportIssue{}
	if !isSubmod {
		return out
	}
	baseTags := baseIndex("country_tag")
	if len(baseTags) == 0 {
		return out
	}
	for _, hf := range historyFiles {
		tag := hf.CountryTag
		if tag == "" || !baseTags[tag] || !notImported(hf.Source) {
			continue
		}
		out = append(out, ExportIssue{
			"level": "warning", "code": "history_writes_base_tag",
			"entity_ref": issueRef("history_file", tag), "severity": "warning",
			"fix_hint": "Keep it to additive lines (e.g. recruit_character) — those stack. " +
				"A full history rewrite can shadow the base country's own setup.",
			"message": "You're writing history for base-mod country " + tag + ". Additive lines " +
				"(like recruiting a character) stack safely on top of the base, but " +
				"a full rewrite of " + tag + "'s history can shadow the base mod's own " +
				"setup for that country. Prefer additive changes.",
		})
	}
	for _, dt := range divisionTemplates {
		tag := dt.CountryTag
		if tag == "" || !baseTags[tag] || !notImported(dt.Source) {
			continue
		}
		out = append(out, ExportIssue{
			"level": "warning", "code": "division_stem_base_collision",
			"entity_ref": map[string]any{
				"type": "division_template", "id": dt.TemplateName,
				"oob_file": dt.OOBFile, "country_tag": tag,
			},
			"severity": "warning",
			"fix_hint": "Use a NEW OOB file name (stem) for your divisions so you don't " +
				"replace the base country's whole order of battle.",
			"message": "Division template '" + dt.TemplateName + "' targets base-mod " +
				"country " + tag + " via OOB file '" + dt.OOBFile + "'. HOI4 loads an " +
				"OOB file whole — if this stem matches the base's, the base army is " +
				"replaced. Use a distinct OOB file name to be safe.",
		})
	}
	return out
}

func GuardrailRecruitRoleFilter(
	project GuardrailProject, characters []GuardrailCharacter,
	baseIndex BaseIndexLookup,
) []ExportIssue {
	out := []ExportIssue{}
	if !project.isSubmod() {
		return out
	}
	baseTags := baseIndex("country_tag")
	for _, c := range characters {
		role := c.RoleType
		tag := c.CountryTag
		if tag == "" {
			tag = project.CountryTag
		}

		if role == "advisor" && !c.HasAdvisorRole {
			continue
		}
		if role != "advisor" || len(baseTags) == 0 || !baseTags[tag] {
			continue
		}
		out = append(out, ExportIssue{
			"level": "warning", "code": "recruit_pure_advisor_base",
			"entity_ref": issueRef("character", c.Tag), "severity": "warning",
			"fix_hint": "Advisors become available automatically — no recruit needed. " +
				"Give the character a leader/general role if you meant it to be recruited.",
			"message": "Character " + c.Tag + " is a pure advisor for base-mod country " +
				tag + ". Advisors self-enable via their slot, so a recruit into the base " +
				"country's history is a no-op ('already assigned' in-game).",
		})
	}
	return out
}

var vanillaTechFolders = map[string]bool{
	"air_doctrine_folder": true, "air_techs_folder": true, "armour_folder": true,
	"artillery_folder": true, "bba_air_techs_folder": true, "electronics_folder": true,
	"industry_folder": true, "infantry_folder": true, "land_doctrine_folder": true,
	"mtgnavalfolder": true, "mtgnavalsupportfolder": true, "naval_doctrine_folder": true,
	"naval_folder": true, "nsb_armour_folder": true,
	"special_forces_doctrine_folder": true, "support_folder": true,
}

var vanillaDoctrineTracks = map[string]bool{
	"infantry": true, "armor": true, "air": true, "artillery": true, "support": true,
	"carriers": true, "screens": true, "capital_ships": true, "submarines": true,
}

func GuardrailTechAndDoctrine(
	project GuardrailProject, techs []GuardrailTech,
	doctrines []GuardrailDoctrine, isSubmod bool,
) []ExportIssue {
	out := []ExportIssue{}
	if isSubmod {
		return out
	}
	projectFolders := map[string]bool{}
	for _, t := range techs {
		if t.Folder != "" {
			projectFolders[t.Folder] = true
		}
	}
	for _, t := range techs {
		folder := t.Folder

		if folder == "" || vanillaTechFolders[folder] || !projectFolders[folder] {
			continue
		}
		delete(projectFolders, folder)
		out = append(out, ExportIssue{
			"level": "warning", "code": "tech_folder_unknown",
			"entity_ref": issueRef("technology", t.TechID), "severity": "warning",
			"fix_hint": "Pick one of HOI4's tech folders, or double-check the spelling.",
			"message": "Technology '" + t.TechID + "' uses folder '" + folder + "', which " +
				"isn't a vanilla tech folder. If that's a typo the tech won't appear in " +
				"any research tree window.",
		})
	}
	projectTracks := map[string]bool{}
	for _, d := range doctrines {
		if d.EntityType == "track" && d.EntityKey != "" {
			projectTracks[d.EntityKey] = true
		}
	}
	for _, d := range doctrines {
		if d.EntityType != "subdoctrine" {
			continue
		}
		track := d.TrackKey
		if track == "" || projectTracks[track] || vanillaDoctrineTracks[track] {
			continue
		}
		out = append(out, ExportIssue{
			"level": "warning", "code": "doctrine_track_unresolved",
			"entity_ref": issueRef("doctrine", d.EntityKey), "severity": "warning",
			"fix_hint": "Define the track in this project, or use a known base/vanilla track.",
			"message": "Subdoctrine '" + d.EntityKey + "' sits on track '" + track + "', which " +
				"this project doesn't define. If the base mod removed that track, the " +
				"doctrine tree can misbehave.",
		})
	}
	return out
}

func NormalizeExportIssue(raw map[string]any) ExportIssue {
	level := anyToString(raw["level"])
	if level == "success" || level == "info" {
		return nil
	}
	severity := anyToString(raw["severity"])
	if severity != "blocker" && severity != "warning" {

		severity = "warning"
	}
	code := anyToString(raw["code"])
	if code == "" {
		code = "content_check"
	}
	detail := raw["detail"]
	if !anyTruthy(detail) {
		detail = map[string]any{}
	}
	out := ExportIssue{
		"code":       code,
		"entity_ref": raw["entity_ref"],
		"detail":     detail,
		"severity":   severity,
		"message":    anyToString(raw["message"]),
		"fix_hint":   anyToString(raw["fix_hint"]),
	}

	if anyTruthy(raw["source"]) {
		out["source"] = raw["source"]
	}
	if anyTruthy(raw["evidence"]) {
		out["evidence"] = raw["evidence"]
	}
	if anyTruthy(raw["repair_route"]) {
		out["repair_route"] = raw["repair_route"]
	}
	if anyTruthy(raw["out_of_stage"]) {
		out["out_of_stage"] = true
	}
	return out
}

func NormalizeExportIssues(raw []map[string]any) []ExportIssue {
	out := []ExportIssue{}
	for _, item := range raw {
		if n := NormalizeExportIssue(item); n != nil {
			out = append(out, n)
		}
	}
	return out
}

func SummarizeExportIssues(issues []ExportIssue) map[string]any {
	blockers := []ExportIssue{}
	warnings := []ExportIssue{}
	for _, i := range issues {
		switch anyToString(i["severity"]) {
		case "blocker":
			blockers = append(blockers, i)
		case "warning":
			warnings = append(warnings, i)
		}
	}
	if issues == nil {
		issues = []ExportIssue{}
	}
	return map[string]any{
		"ok":            len(blockers) == 0,
		"blocker_count": len(blockers),
		"warning_count": len(warnings),
		"blockers":      blockers,
		"warnings":      warnings,
		"issues":        issues,
	}
}

func EffectiveIncludeImported(
	requested bool, hasDependencies, isLoadedExistingMod, hasImportedRows bool,
) bool {
	if !requested {
		return false
	}
	if !hasDependencies {
		return true
	}
	return isLoadedExistingMod || hasImportedRows
}
