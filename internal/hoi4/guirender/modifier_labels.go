package guirender

import "strings"

var modifierLocKeys = map[string]string{
	"conscription":                  "MODIFIER_CONSCRIPTION_FACTOR",
	"conscription_factor":           "MODIFIER_CONSCRIPTION_TOTAL_FACTOR",
	"industrial_capacity_factory":   "MODIFIER_INDUSTRIAL_CAPACITY_FACTOR",
	"industrial_capacity_dockyard":  "MODIFIER_INDUSTRIAL_CAPACITY_DOCKYARD_FACTOR",
	"min_export":                    "MODIFIER_MIN_EXPORT_FACTOR",
	"refit_speed":                   "MODIFIER_INDUSTRIAL_REFIT_SPEED_FACTOR",
	"refit_ic_cost":                 "MODIFIER_INDUSTRIAL_REFIT_IC_COST_FACTOR",
	"non_core_manpower":             "MODIFIER_GLOBAL_NON_CORE_MANPOWER",
	"monthly_population":            "MODIFIER_GLOBAL_MONTHLY_POPULATION",
	"join_faction_tension":          "MODIFIER_JOIN_FACTION_TENSION_LIMIT",
	"experience_gain_factor":        "MODIFIER_XP_GAIN_FACTOR",
	"experience_gain_army":          "MODIFIER_XP_GAIN_ARMY",
	"experience_gain_navy":          "MODIFIER_XP_GAIN_NAVY",
	"experience_gain_air":           "MODIFIER_XP_GAIN_AIR",
	"experience_gain_army_factor":   "MODIFIER_XP_GAIN_ARMY_FACTOR",
	"experience_gain_navy_factor":   "MODIFIER_XP_GAIN_NAVY_FACTOR",
	"experience_gain_air_factor":    "MODIFIER_XP_GAIN_AIR_FACTOR",
	"resistance_damage_to_garrison": "MODIFIER_RESISTANCE_DAMAGE_TO_GARRISONS",
	"resistance_damage_to_garrison_on_our_occupied_states": "MODIFIER_RESISTANCE_DAMAGE_TO_GARRISONS_ON_OUR_OCCUPIED_STATES",
	"foreign_subversive_activites":                         "MODIFIER_FOREIGN_SUBVERSIVE_ACTIVITIES",
	"war_stability_factor":                                 "MODIFIER_STABILITY_WAR_FACTOR",
	"offensive_war_stability_factor":                       "MODIFIER_STABILITY_OFFENSIVE_WAR_FACTOR",
	"defensive_war_stability_factor":                       "MODIFIER_STABILITY_DEFENSIVE_WAR_FACTOR",
	"party_popularity_stability_factor":                    "MODIFIER_STABILITY_POPULARITY_FACTOR",
	"subjects_autonomy_gain":                               "MODIFIER_AUTONOMY_SUBJECT_GAIN",
	"trade_cost_for_target_factor":                         "MODIFIER_TRADE_COST_TO_TARGET_FACTOR",
	"military_industrial_organization_funds_gain":          "MODIFIER_MIO_FUNDS_GAIN",
	"compliance_gain":                                      "MODIFIER_COMPLIANCE_GAIN_ADD",
	"naval_invasion_prep_speed":                            "MODIFIER_NAVAL_INVASION_PREPARATION_SPEED",
}

var modifierPercentNames = map[string]bool{
	"planning_speed": true, "stability_weekly": true, "war_support_weekly": true,
	"conscription": true, "industrial_capacity_factory": true, "industrial_capacity_dockyard": true,
	"min_export": true, "refit_speed": true, "refit_ic_cost": true,
	"non_core_manpower": true, "monthly_population": true,
}

func modifierReadsPercent(key string) bool {
	key = strings.ToLower(key)
	return strings.HasSuffix(key, "_factor") || strings.HasSuffix(key, "_tension") || modifierPercentNames[key]
}

func (idx *Index) modifierLabel(key string) string {
	lower := strings.ToLower(strings.TrimSpace(key))
	upper := strings.ToUpper(lower)
	for _, candidate := range []string{
		modifierLocKeys[lower], "MODIFIER_" + upper, "MODIFIERS_" + upper, "MODIFIER_GLOBAL_" + upper,
		"MODIFIER_" + upper + "_LIMIT", "MODIFIER_" + upper + "_FACTOR", "modifier_" + lower, lower,
	} {
		if candidate == "" {
			continue
		}
		if text, ok := idx.Localise(candidate); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(stripColour(text))
		}
	}
	if label, ok := idx.doctrineMasteryLabel(lower); ok {
		return label
	}
	return lower
}

func (idx *Index) doctrineMasteryLabel(key string) (string, bool) {
	named := func(template, fallback, nameKey, token string) (string, bool) {
		if nameKey == "" {
			nameKey = token
		}
		name := idx.fillTooltipLoc(template, fallback, map[string]any{"ARG": idx.localisedOr(nameKey)})
		return strings.TrimSpace(stripColour(name)), true
	}
	if token, ok := strings.CutSuffix(key, "_track_mastery_gain_factor"); ok {
		if track, found := idx.doctrineTrackFor(token); found {
			return named("mod_track_mastery_gain_factor", "£doctrine_mod $ARG|H$ track mastery gain factor", track.Name, token)
		}
		return "", false
	}
	token, ok := strings.CutSuffix(key, "_mastery_gain_factor")
	if !ok || token == "" {
		return "", false
	}
	for _, folder := range idx.doctrineFolders() {
		if folder.Token == token {
			return named("mod_folder_mastery_gain_factor", "£doctrine_mod $ARG|H$ mastery gain", folder.Name, token)
		}
	}
	if grand, found := idx.grandDoctrineFor(token); found {
		return named("mod_grand_doctrine_mastery_gain_factor", "£doctrine_mod $ARG|H$ (grand doctrine) mastery gain", grand.Name, token)
	}
	if sub, found := idx.subDoctrineFor(token); found {
		return named("mod_subdoctrine_mastery_gain_factor", "£doctrine_mod $ARG|H$ (subdoctrine) mastery gain", sub.Name, token)
	}

	if text, found := idx.Localise(token); found && strings.TrimSpace(text) != "" {
		return named("mod_grand_doctrine_mastery_gain_factor", "£doctrine_mod $ARG|H$ (grand doctrine) mastery gain", token, token)
	}
	return "", false
}
