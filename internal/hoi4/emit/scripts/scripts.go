package emitscripts

import (
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
)

var bareNameToken = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)

func FormatHoi4NameScalar(value any) string {
	text := pyStrip(PyStr(value))
	if bareNameToken.MatchString(text) {
		return text
	}
	return `"` + strings.ReplaceAll(text, `"`, `\"`) + `"`
}

type AutoFire struct {
	EventID string
	Days    int
	Hook    string
	Legacy  bool
}

var leaderScopedOnActions = map[string]bool{
	"on_unit_leader_created":                    true,
	"on_army_leader_daily":                      true,
	"on_army_leader_won_combat":                 true,
	"on_army_leader_lost_combat":                true,
	"on_unit_leader_level_up":                   true,
	"on_army_leader_promoted":                   true,
	"on_unit_leader_promote_from_ranks_veteran": true,
	"on_unit_leader_promote_from_ranks_green":   true,
}

func GenerateOnActions(
	countryTag string,
	hasTraitOverlay bool,
	traitOverlayNamespace string,
	countryIdeaIDs []string,
	autoFireEvents []AutoFire,
	loadFocusTreeIDs []string,
) string {
	type evDays struct {
		ID   string
		Days int
	}
	var normalized []struct {
		ID   string
		Days int
		Hook string
	}
	push := func(id string, days int, hook string) {
		normalized = append(normalized, struct {
			ID   string
			Days int
			Hook string
		}{id, days, hook})
	}
	for _, entry := range autoFireEvents {
		if entry.Legacy {
			push(entry.EventID, entry.Days, "on_startup")
			continue
		}
		hook := entry.Hook
		if hook == "" {
			hook = "on_startup"
		}
		if hook == "on_unit_leader_promote_from_ranks" {
			push(entry.EventID, entry.Days, "on_unit_leader_promote_from_ranks_veteran")
			push(entry.EventID, entry.Days, "on_unit_leader_promote_from_ranks_green")
		} else {
			if hook == "on_subject_annexed_by_overlord" {
				hook = "on_subject_annexed"
			}
			push(entry.EventID, entry.Days, hook)
		}
	}

	var startupEvents []evDays
	otherHooks := map[string][]evDays{}
	var otherOrder []string
	for _, n := range normalized {
		if n.Hook == "on_startup" {
			startupEvents = append(startupEvents, evDays{n.ID, n.Days})
		} else {
			if _, ok := otherHooks[n.Hook]; !ok {
				otherOrder = append(otherOrder, n.Hook)
			}
			otherHooks[n.Hook] = append(otherHooks[n.Hook], evDays{n.ID, n.Days})
		}
	}

	hasStartup := hasTraitOverlay || len(countryIdeaIDs) > 0 ||
		len(startupEvents) > 0 || len(loadFocusTreeIDs) > 0

	if !hasStartup && len(otherHooks) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "on_actions = {")

	if hasStartup {
		lines = append(lines, "\ton_startup = {")

		if len(loadFocusTreeIDs) > 0 {
			lines = append(lines, "\t\teffect = {")
			lines = append(lines, "\t\t\t"+countryTag+" = {")
			for _, treeID := range loadFocusTreeIDs {
				lines = append(lines, "\t\t\t\tload_focus_tree = "+treeID)
			}
			lines = append(lines, "\t\t\t}")
			lines = append(lines, "\t\t}")
		}

		if hasTraitOverlay && traitOverlayNamespace != "" {
			lines = append(lines, "\t\teffect = {")
			lines = append(lines, "\t\t\t"+countryTag+" = {")
			lines = append(lines, "\t\t\t\tcountry_event = "+traitOverlayNamespace+".1")
			lines = append(lines, "\t\t\t}")
			lines = append(lines, "\t\t}")
		}

		if len(countryIdeaIDs) > 0 {
			lines = append(lines, "\t\teffect = {")
			lines = append(lines, "\t\t\t"+countryTag+" = {")
			lines = append(lines, "\t\t\t\tadd_ideas = {")
			for _, ideaID := range countryIdeaIDs {
				lines = append(lines, "\t\t\t\t\t"+ideaID)
			}
			lines = append(lines, "\t\t\t\t}")
			lines = append(lines, "\t\t\t}")
			lines = append(lines, "\t\t}")
		}

		if len(startupEvents) > 0 {
			lines = append(lines, "\t\teffect = {")
			lines = append(lines, "\t\t\t"+countryTag+" = {")
			for _, ev := range startupEvents {
				if ev.Days > 0 {
					lines = append(lines,
						"\t\t\t\tcountry_event = { id = "+ev.ID+" days = "+strconv.Itoa(ev.Days)+" }")
				} else {
					lines = append(lines, "\t\t\t\tcountry_event = "+ev.ID)
				}
			}
			lines = append(lines, "\t\t\t}")
			lines = append(lines, "\t\t}")
		}

		lines = append(lines, "\t}")
	}

	sortedHooks := append([]string(nil), otherOrder...)
	sort.Strings(sortedHooks)
	for _, hook := range sortedHooks {
		eventsForHook := otherHooks[hook]
		if len(eventsForHook) == 0 {
			continue
		}
		if hook == "on_yearly" {
			lines = append(lines,
				"\t# WARNING: legacy on_yearly has no HOI4 1.19 equivalent; "+
					"preserved without silent remapping")
		}
		lines = append(lines, "\t"+hook+" = {")
		lines = append(lines, "\t\teffect = {")

		pad := "\t\t\t"
		leaderScoped := leaderScopedOnActions[hook]
		if leaderScoped {
			lines = append(lines, pad+"owner = {")
			pad += "\t"
		}
		lines = append(lines, pad+"if = {")
		lines = append(lines, pad+"\tlimit = { tag = "+countryTag+" }")
		for _, ev := range eventsForHook {
			if ev.Days > 0 {
				lines = append(lines,
					pad+"\tcountry_event = { id = "+ev.ID+" days = "+strconv.Itoa(ev.Days)+" }")
			} else {
				lines = append(lines, pad+"\tcountry_event = "+ev.ID)
			}
		}
		lines = append(lines, pad+"}")
		if leaderScoped {
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "\t}")
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func BookmarkWeatherSeed(bmID string) int {
	s := bmID
	if s == "" {
		s = "bookmark"
	}
	var h uint64 = 2166136261
	for _, ch := range s {
		h = (h*16777619 + uint64(ch)) & 0xFFFFFFFF
	}
	return 10000 + int(h%90000)
}

func BookmarkHasWeather(rawExtras *Dict) bool {
	for _, k := range rawExtras.Keys() {
		v := rawExtras.Get(k)
		if pyLower(pyStrip(k)) == "effect" {
			return true
		}
		if v != nil && strings.Contains(PyStr(v), "randomize_weather") {
			return true
		}
	}
	return false
}

func EmitVerbatimInnerBlock(lines *[]string, key string, rawText any, indent string) {
	src := ""
	if pyTruthy(rawText) {
		src = PyStr(rawText)
	}
	var body []string
	for _, ln := range pySplitlines(src) {
		if pyStrip(ln) != "" {
			body = append(body, pyStrip(ln))
		}
	}
	if len(body) == 0 {
		return
	}
	if len(body) == 1 {
		*lines = append(*lines, indent+key+" = { "+body[0]+" }")
		return
	}
	*lines = append(*lines, indent+key+" = {")
	for _, ln := range body {
		*lines = append(*lines, indent+"\t"+ln)
	}
	*lines = append(*lines, indent+"}")
}

type State struct {
	StateID                 any
	Name                    string
	Manpower                any
	Impassable              bool
	ImpassableIgnoredLinks  any
	Resources               *Dict
	StateCategory           string
	Owner                   string
	Controller              any
	AddCoreOf               []any
	AddClaimBy              []any
	RemoveCoreOf            []any
	SetProvinceController   []any
	VictoryPoints           []any
	RawExtras               *Dict
	Buildings               *Dict
	Provinces               []any
	LocalSupplies           any
	BuildingsMaxLevelFactor any
}

func EmitStateLines(st *State) []string {
	sid := st.StateID
	sname := pyStrip(st.Name)
	if sname == "" {
		sname = "STATE_" + PyStr(sid)
	}
	lines := []string{"state = {"}
	lines = append(lines, "\tid = "+PyStr(sid))
	lines = append(lines, "\tname = \""+sname+"\"")

	if pyTruthy(st.Manpower) {
		lines = append(lines, "\tmanpower = "+pyIntStr(st.Manpower))
	}

	if st.Impassable {
		lines = append(lines, "\timpassable = yes")
	}

	if st.ImpassableIgnoredLinks != nil {
		EmitVerbatimInnerBlock(&lines, "impassable_ignored_links", st.ImpassableIgnoredLinks, "\t")
	}

	if st.Resources.Truthy() {
		lines = append(lines, "\tresources = {")
		for _, rkey := range st.Resources.Keys() {
			lines = append(lines, "\t\t"+rkey+" = "+PyStr(st.Resources.Get(rkey)))
		}
		lines = append(lines, "\t}")
	}

	if st.StateCategory != "" {
		lines = append(lines, "\tstate_category = "+st.StateCategory)
	}

	var historyLines []string
	if st.Owner != "" {
		historyLines = append(historyLines, "\t\towner = "+st.Owner)
	}
	if pyTruthy(st.Controller) {
		historyLines = append(historyLines, "\t\tcontroller = "+PyStr(st.Controller))
	}
	for _, core := range st.AddCoreOf {
		if pyTruthy(core) {
			historyLines = append(historyLines, "\t\tadd_core_of = "+PyStr(core))
		}
	}
	for _, claim := range st.AddClaimBy {
		if pyTruthy(claim) {
			historyLines = append(historyLines, "\t\tadd_claim_by = "+PyStr(claim))
		}
	}
	for _, rc := range st.RemoveCoreOf {
		if pyTruthy(rc) {
			historyLines = append(historyLines, "\t\tremove_core_of = "+PyStr(rc))
		}
	}

	for _, spcAny := range st.SetProvinceController {
		spc, ok := spcAny.(*Dict)
		if !ok {
			continue
		}
		pid := spc.Get("province_id")
		ctrl := spc.Get("controller")
		if pid != nil && pyTruthy(ctrl) {
			historyLines = append(historyLines,
				"\t\tset_province_controller = { province = "+PyStr(pid)+
					" controller = "+PyStr(ctrl)+" }")
		}
	}

	for _, vpAny := range st.VictoryPoints {
		vp, ok := vpAny.(*Dict)
		if !ok {
			continue
		}
		pid := vp.Get("province_id")
		pval := vp.Get("value")
		if pid == nil || pval == nil {
			continue
		}
		historyLines = append(historyLines,
			"\t\tvictory_points = { "+PyStr(pid)+" "+PyStr(pval)+" }")
	}

	rawExtras := st.RawExtras
	if rawExtras == nil {
		rawExtras = NewDict()
	}
	stateBuildings := st.Buildings
	provinceBuildingsRaw := ""
	if v := rawExtras.Get("buildings"); pyTruthy(v) {
		provinceBuildingsRaw = pyStrip(PyStr(v))
	}
	if stateBuildings.Truthy() || provinceBuildingsRaw != "" {
		historyLines = append(historyLines, "\t\tbuildings = {")
		for _, bkey := range stateBuildings.Keys() {
			historyLines = append(historyLines, "\t\t\t"+bkey+" = "+PyStr(stateBuildings.Get(bkey)))
		}
		for _, vl := range pySplitlines(provinceBuildingsRaw) {
			if pyStrip(vl) != "" {
				if strings.HasPrefix(vl, "\t") {
					historyLines = append(historyLines, "\t\t\t"+pyRStrip(vl))
				} else {
					historyLines = append(historyLines, "\t\t\t"+pyStrip(vl))
				}
			}
		}
		historyLines = append(historyLines, "\t\t}")
	}

	for _, rk := range rawExtras.Keys() {
		if !strings.HasPrefix(rk, "history.") {
			continue
		}
		innerKey := rk[len("history."):]
		EmitRawExtrasBlockOrScalar(&historyLines, innerKey, rawExtras.Get(rk), "\t\t")
	}

	if len(historyLines) > 0 {
		lines = append(lines, "\thistory = {")
		lines = append(lines, historyLines...)
		lines = append(lines, "\t}")
	}

	if len(st.Provinces) > 0 {
		parts := make([]string, 0, len(st.Provinces))
		for _, p := range st.Provinces {
			parts = append(parts, PyStr(p))
		}
		lines = append(lines, "\tprovinces = {")
		lines = append(lines, "\t\t"+strings.Join(parts, " "))
		lines = append(lines, "\t}")
	}

	if st.LocalSupplies != nil {
		lines = append(lines, "\tlocal_supplies = "+FmtHoi4Float(pyFloatOf(st.LocalSupplies)))
	}

	if st.BuildingsMaxLevelFactor != nil {
		lines = append(lines, "\tbuildings_max_level_factor = "+FmtHoi4Float(pyFloatOf(st.BuildingsMaxLevelFactor)))
	}

	for _, rk := range rawExtras.Keys() {
		if rk == "buildings" || strings.HasPrefix(rk, "history.") {
			continue
		}
		if rk == "impassable_ignored_links" && st.ImpassableIgnoredLinks != nil {
			continue
		}
		EmitRawExtrasBlockOrScalar(&lines, rk, rawExtras.Get(rk), "\t")
	}

	lines = append(lines, "}")
	return lines
}

var ulTraitSkillKeys = []string{
	"attack_skill", "defense_skill", "logistics_skill",
	"planning_skill", "maneuvering_skill", "coordination_skill",
}

var ulTraitSkillFactorKeys = []string{
	"attack_skill_factor", "defense_skill_factor", "logistics_skill_factor",
	"planning_skill_factor", "maneuvering_skill_factor", "coordination_skill_factor",
}

var ulTraitStringKeys = []string{
	"trait_type", "mutually_exclusive", "enable_ability",
	"override_effect_tooltip", "custom_effect_tooltip",
	"custom_prerequisite_tooltip", "custom_gain_xp_trigger_tooltip",
}

var ulTraitIntKeys = []string{"gui_row", "gui_column", "gain_xp_on_spotting", "cost"}

var ulTraitBoolKeys = []string{"show_in_combat"}

var ulTraitScriptKeys = []string{
	"allowed", "prerequisites", "gain_xp", "gain_xp_leader",
	"on_add", "on_remove", "daily_effect",
}

var ulTraitModifierKeys = []string{
	"non_shared_modifier", "corps_commander_modifier", "field_marshal_modifier",
}

var ulTraitRawBlockKeys = []string{
	"parent", "any_parent", "all_parents", "sub_unit_modifiers",
	"trait_xp_factor", "new_commander_weight",
}

var ulTraitTypeValues = map[string]bool{
	"land": true, "navy": true, "all": true,
	"corps_commander": true, "field_marshal": true,
}

var clTraitStringKeys = []string{"mutually_exclusive", "custom_modifier_tooltip"}
var clTraitIntKeys = []string{"cost"}
var clTraitRawBlockKeys = []string{"equipment_bonus", "custom_modifier_tooltip"}

func FormatTraitIntScalar(value any) *string {
	switch t := value.(type) {
	case bool:
		return nil
	case *big.Int:
		s := t.String()
		return &s
	case float64:
		s := pyIntStr(t)
		return &s
	case string:
		if pyStrip(t) != "" && !strings.Contains(t, "\n") && !strings.Contains(t, "\r") {
			s := pyStrip(t)
			return &s
		}
	}
	return nil
}

func isNumberNotBool(v any) bool {
	switch v.(type) {
	case *big.Int, float64:
		return true
	}
	return false
}

func pyStrOr(v any) string {
	if pyTruthy(v) {
		return PyStr(v)
	}
	return ""
}

type CustomTrait struct {
	TraitID          string
	TraitType        string
	Modifier         *Dict
	AiWillDo         string
	Random           bool
	ExtraFields      *Dict
	RawExtras        *Dict
	TargetedModifier []any
}

func GenerateCustomUnitLeaderTraitsScript(traits []*CustomTrait) string {
	var ul []*CustomTrait
	for _, t := range traits {
		if t.TraitType == "unit_leader" {
			ul = append(ul, t)
		}
	}
	if len(ul) == 0 {
		return ""
	}
	lines := []string{"leader_traits = {", ""}
	for _, t := range ul {
		lines = append(lines, "\t"+t.TraitID+" = {")
		extras := t.ExtraFields
		if extras == nil {
			extras = NewDict()
		}

		typeVal := "land"
		if tv, ok := extras.Get("type").(string); ok && ulTraitTypeValues[tv] {
			typeVal = tv
		}
		lines = append(lines, "\t\ttype = "+typeVal)

		for _, k := range ulTraitStringKeys {
			if val, ok := extras.Get(k).(string); ok && pyStrip(val) != "" {
				lines = append(lines, "\t\t"+k+" = "+pyStrip(val))
			}
		}

		for _, skillKey := range ulTraitSkillKeys {
			if rendered := FormatTraitIntScalar(extras.Get(skillKey)); rendered != nil {
				lines = append(lines, "\t\t"+skillKey+" = "+*rendered)
			}
		}

		for _, fkey := range ulTraitSkillFactorKeys {
			val := extras.Get(fkey)
			if isNumberNotBool(val) && pyTruthy(val) {
				lines = append(lines, "\t\t"+fkey+" = "+PyStr(val))
			}
		}

		for _, ikey := range ulTraitIntKeys {
			if rendered := FormatTraitIntScalar(extras.Get(ikey)); rendered != nil {
				lines = append(lines, "\t\t"+ikey+" = "+*rendered)
			}
		}

		for _, bkey := range ulTraitBoolKeys {
			if val, ok := extras.Get(bkey).(bool); ok {
				yn := "no"
				if val {
					yn = "yes"
				}
				lines = append(lines, "\t\t"+bkey+" = "+yn)
			}
		}

		for _, mkey := range ulTraitModifierKeys {
			if mod, ok := extras.Get(mkey).(*Dict); ok && mod.Truthy() {
				lines = append(lines, "\t\t"+mkey+" = {")
				for _, k := range mod.Keys() {
					EmitMioValue(&lines, k, mod.Get(k), "\t\t\t")
				}
				lines = append(lines, "\t\t}")
			}
		}

		if t.Modifier.Truthy() {
			lines = append(lines, "\t\tmodifier = {")
			for _, k := range t.Modifier.Keys() {
				EmitMioValue(&lines, k, t.Modifier.Get(k), "\t\t\t")
			}
			lines = append(lines, "\t\t}")
		}

		emitTargetedModifiers(&lines, t.TargetedModifier)

		for _, skey := range ulTraitScriptKeys {
			if text, ok := extras.Get(skey).(string); ok && pyStrip(text) != "" {
				lines = append(lines, "\t\t"+skey+" = {")
				for _, rawLine := range splitNL(text) {
					lines = append(lines, "\t\t\t"+pyRStrip(rawLine))
				}
				lines = append(lines, "\t\t}")
			}
		}

		if t.RawExtras != nil {
			for _, rk := range ulTraitRawBlockKeys {
				if rv, ok := t.RawExtras.Get(rk).(string); ok && pyStrip(rv) != "" {
					EmitRawScriptBlock(&lines, rk, rv, "\t\t")
				}
			}
		}

		aiText := pyStrip(t.AiWillDo)
		if aiText != "" {
			lines = append(lines, "\t\tai_will_do = {")
			for _, rawLine := range splitNL(aiText) {
				lines = append(lines, "\t\t\t"+pyRStrip(rawLine))
			}
			lines = append(lines, "\t\t}")
		}
		lines = append(lines, "\t}")
		lines = append(lines, "")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func emitTargetedModifiers(lines *[]string, entries []any) {
	for _, entryAny := range entries {
		entry, ok := entryAny.(*Dict)
		if !ok {
			continue
		}
		tag := ""
		if v := entry.Get("tag"); pyTruthy(v) {
			tag = pyStrip(PyStr(v))
		}
		mods, modsOK := entry.Get("modifier").(*Dict)
		if tag == "" || !modsOK || !mods.Truthy() {
			continue
		}
		*lines = append(*lines, "\t\ttargeted_modifier = {")
		*lines = append(*lines, "\t\t\ttag = "+tag)
		for _, mk := range mods.Keys() {
			EmitMioValue(lines, mk, mods.Get(mk), "\t\t\t")
		}
		*lines = append(*lines, "\t\t}")
	}
}

var scientistTraitSpecializations = map[string]bool{
	"specialization_air": true, "specialization_land": true,
	"specialization_naval": true, "specialization_nuclear": true,
}

func GenerateCustomScientistTraitsScript(traits []*CustomTrait) string {
	var sci []*CustomTrait
	for _, t := range traits {
		if t.TraitType == "scientist" {
			sci = append(sci, t)
		}
	}
	if len(sci) == 0 {
		return ""
	}
	var blocks []string
	for _, t := range sci {
		lines := []string{t.TraitID + " = {"}
		extras := t.ExtraFields
		if extras == nil {
			extras = NewDict()
		}
		if spec, ok := extras.Get("specialization").(string); ok && scientistTraitSpecializations[spec] {
			lines = append(lines, "\tspecialization = { "+spec+" }")
		}
		if iconRef, ok := extras.Get("icon").(string); ok && pyStrip(iconRef) != "" {
			lines = append(lines, "\ticon = "+pyStrip(iconRef))
		}
		if t.Modifier.Truthy() {
			lines = append(lines, "\tmodifier = {")
			for _, k := range t.Modifier.Keys() {
				EmitMioValue(&lines, k, t.Modifier.Get(k), "\t\t")
			}
			lines = append(lines, "\t}")
		}
		lines = append(lines, "}")
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

type DynamicModifier struct {
	DmID          string
	Icon          string
	Enable        string
	RemoveTrigger string
	Modifier      *Dict
	RawExtras     *Dict
}

func emitStrippedRawBody(lines *[]string, key, inner string) {
	if strings.HasPrefix(inner, "{") && strings.HasSuffix(inner, "}") {
		*lines = append(*lines, "\t"+key+" = "+inner)
		return
	}
	*lines = append(*lines, "\t"+key+" = {")
	for _, rawLine := range splitNL(inner) {
		stripped := pyRStrip(rawLine)
		if stripped != "" {
			*lines = append(*lines, "\t\t"+stripped)
		}
	}
	*lines = append(*lines, "\t}")
}

func GenerateDynamicModifiersScript(modifiers []*DynamicModifier) string {
	if len(modifiers) == 0 {
		return ""
	}
	var lines []string
	for _, m := range modifiers {
		if m.DmID == "" {
			continue
		}
		lines = append(lines, m.DmID+" = {")
		if m.Icon != "" {
			lines = append(lines, "\ticon = "+m.Icon)
		}
		if m.Enable != "" && pyStrip(m.Enable) != "" {
			emitStrippedRawBody(&lines, "enable", pyStrip(m.Enable))
		}
		if m.RemoveTrigger != "" && pyStrip(m.RemoveTrigger) != "" {
			emitStrippedRawBody(&lines, "remove_trigger", pyStrip(m.RemoveTrigger))
		}
		for _, k := range m.Modifier.Keys() {
			EmitMioValue(&lines, k, m.Modifier.Get(k), "\t")
		}
		for _, k := range m.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, m.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type AutonomyState struct {
	AutonomyID        string
	IsPuppet          bool
	IsDefault         bool
	MinFreedomLevel   any
	ManpowerInfluence any
	Rule              string
	Modifier          *Dict
	AiWeights         *Dict
	RawExtras         *Dict
}

func GenerateAutonomyStatesScript(states []*AutonomyState) string {
	if len(states) == 0 {
		return ""
	}
	var lines []string
	for _, s := range states {
		if s.AutonomyID == "" {
			continue
		}
		lines = append(lines, "autonomy_state = {")
		lines = append(lines, "\tid = "+s.AutonomyID)
		if s.IsPuppet {
			lines = append(lines, "\tis_puppet = yes")
		}
		if s.IsDefault {
			lines = append(lines, "\tdefault = yes")
		}
		if pyTruthy(s.MinFreedomLevel) {
			mfl := FmtHoi4Float(pyFloatOf(s.MinFreedomLevel) / 1000.0)
			lines = append(lines, "\tmin_freedom_level = "+mfl)
		}
		if s.ManpowerInfluence != nil {
			lines = append(lines, "\tmanpower_influence = "+FmtHoi4Float(pyFloatOf(s.ManpowerInfluence)))
		}
		if s.Rule != "" && pyStrip(s.Rule) != "" {
			emitStrippedRawBody(&lines, "rule", pyStrip(s.Rule))
		}
		if s.Modifier.Truthy() {
			lines = append(lines, "\tmodifier = {")
			for _, k := range s.Modifier.Keys() {
				EmitMioValue(&lines, k, s.Modifier.Get(k), "\t\t")
			}
			lines = append(lines, "\t}")
		}
		for _, k := range s.AiWeights.Keys() {
			rawText := s.AiWeights.Get(k)
			if d, ok := rawText.(*Dict); ok {
				EmitMioValue(&lines, k, d, "\t")
			} else {
				EmitRawExtrasBlockOrScalar(&lines, k, rawText, "\t")
			}
		}
		for _, k := range s.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, s.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type MIO struct {
	MioID              string
	Include            string
	Icon               string
	EquipmentType      []any
	ResearchCategories []any
	Allowed            string
	Available          string
	Visible            string
	Traits             []any
	RawExtras          *Dict
	Source             string
	ScopeKind          string
	CountryTag         string
	SharedCountryTags  []string
}

func SynthMioAllowed(m *MIO) string {
	if pyStrip(m.Allowed) != "" {
		return m.Allowed
	}
	body, ok := importsvc.SynthCountryGate(m.ScopeKind, m.CountryTag, m.SharedCountryTags)
	if ok && body != "" {
		return "{ " + body + " }"
	}
	return "{ always = yes }"
}

func GenerateMIOScript(mios []*MIO, resolvedManagedIcons, resolvedManagedTraitIcons map[string]bool) string {
	if len(mios) == 0 {
		return ""
	}
	var lines []string
	for _, m := range mios {
		if m.MioID == "" {
			continue
		}
		hasInclude := pyStrip(m.Include) != ""
		lines = append(lines, m.MioID+" = {")
		if m.Include != "" {
			lines = append(lines, "\tinclude = "+m.Include)
		}
		if m.Icon != "" {
			managed := strings.HasPrefix(m.Icon, "GFX_idea_mio_")
			unresolved := managed && resolvedManagedIcons != nil && !resolvedManagedIcons[m.Icon]
			if !unresolved {
				lines = append(lines, "\ticon = "+m.Icon)
			}
		}
		if len(m.EquipmentType) > 0 {
			EmitMioValue(&lines, "equipment_type", append([]any(nil), m.EquipmentType...), "\t")
		}
		if len(m.ResearchCategories) > 0 {
			EmitMioValue(&lines, "research_categories", append([]any(nil), m.ResearchCategories...), "\t")
		}
		allowedText := m.Allowed
		if m.Source == "created" {
			allowedText = SynthMioAllowed(m)
		}
		EmitRawScriptBlock(&lines, "allowed", allowedText, "\t")
		EmitRawScriptBlock(&lines, "available", m.Available, "\t")
		EmitRawScriptBlock(&lines, "visible", m.Visible, "\t")
		for _, traitAny := range m.Traits {
			trait, ok := traitAny.(*Dict)
			if !ok {
				continue
			}
			kind, _ := trait.Get("kind").(string)
			var blockKey string
			switch {
			case kind == "initial":
				blockKey = "initial_trait"
			case kind == "override":
				blockKey = "override_trait"
			case hasInclude:
				blockKey = "add_trait"
			default:
				blockKey = "trait"
			}
			lines = append(lines, "\t"+blockKey+" = {")
			for _, k := range trait.Keys() {
				v := trait.Get(k)
				if k == "kind" || k == "raw_extras" || v == nil {
					continue
				}
				if k == "icon" {
					if vs, isStr := v.(string); isStr && strings.HasPrefix(vs, "GFX_idea_mio_trait_") {
						if resolvedManagedTraitIcons != nil && !resolvedManagedTraitIcons[vs] {
							continue
						}
					}
				}
				if vs, isStr := v.(string); isStr && k == "name" {
					lines = append(lines, "\t\tname = "+FormatHoi4NameScalar(vs))
				} else if _, isStr := v.(string); isStr && mioTraitScriptFields[k] {
					EmitRawScriptBlock(&lines, k, v, "\t\t")
				} else {
					EmitMioValue(&lines, k, v, "\t\t")
				}
			}
			if re, ok := trait.Get("raw_extras").(*Dict); ok {
				for _, k := range re.Keys() {
					EmitRawExtrasBlockOrScalar(&lines, k, re.Get(k), "\t\t")
				}
			}
			lines = append(lines, "\t}")
		}
		for _, k := range m.RawExtras.Keys() {

			if strings.HasPrefix(k, "__") {
				continue
			}
			if k == "equipment_type" && len(m.EquipmentType) > 0 {
				continue
			}
			if k == "research_categories" && len(m.ResearchCategories) > 0 {
				continue
			}
			EmitRawExtrasBlockOrScalar(&lines, k, m.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type Doctrine struct {
	EntityKey             string
	EntityType            string
	FolderKey             string
	TrackKey              string
	AllowInMultipleTracks any
	Name                  string
	Description           string
	Icon                  string
	XpCost                any
	XpType                string
	Xor                   any
	RawExtras             *Dict
	SourceFile            string
}

func DoctrineDefaultSourceFile(d *Doctrine, modDirName string) string {
	et := d.EntityType
	folder := pyStrip(d.FolderKey)
	if folder == "" {
		folder = "land"
	}
	switch et {
	case "folder":
		return "folders/zz_" + modDirName + "_doctrine_folders.txt"
	case "grand_doctrine":
		return "grand_doctrines/zz_" + modDirName + "_" + folder + "_grand_doctrines.txt"
	case "track":
		return "tracks/zz_" + modDirName + "_" + folder + "_doctrine_tracks.txt"
	case "subdoctrine":
		track := pyStrip(d.TrackKey)
		if track == "" {
			track = "infantry"
		}
		return "subdoctrines/" + folder + "/zz_" + modDirName + "_" + track + "_subdoctrines.txt"
	}
	return et + "/zz_" + modDirName + "_" + d.EntityKey + ".txt"
}

type FileGroups[T any] struct {
	Paths []string
	Rows  map[string][]T
}

func newFileGroups[T any]() *FileGroups[T] {
	return &FileGroups[T]{Rows: map[string][]T{}}
}

func (g *FileGroups[T]) add(path string, row T) {
	if _, ok := g.Rows[path]; !ok {
		g.Paths = append(g.Paths, path)
	}
	g.Rows[path] = append(g.Rows[path], row)
}

func DoctrinesByFile(doctrines []*Doctrine, modDirName string) *FileGroups[*Doctrine] {
	byFile := newFileGroups[*Doctrine]()
	for _, d := range doctrines {
		rel := pyStrip(d.SourceFile)
		if rel == "" {
			rel = DoctrineDefaultSourceFile(d, modDirName)
		}
		rel = strings.TrimLeft(strings.ReplaceAll(rel, "\\", "/"), "/")
		byFile.add(rel, d)
	}
	return byFile
}

func GenerateDoctrineScript(doctrines []*Doctrine) string {
	if len(doctrines) == 0 {
		return ""
	}
	var lines []string
	for _, d := range doctrines {
		if d.EntityKey == "" {
			continue
		}
		lines = append(lines, d.EntityKey+" = {")
		isSubdoctrine := d.EntityType == "subdoctrine"
		if d.EntityType == "grand_doctrine" && pyStrip(d.FolderKey) != "" {
			lines = append(lines, "\tfolder = "+pyStrip(d.FolderKey))
		}
		if isSubdoctrine && pyStrip(d.TrackKey) != "" {
			lines = append(lines, "\ttrack = "+pyStrip(d.TrackKey))
		}
		if aimt, ok := d.AllowInMultipleTracks.(bool); isSubdoctrine && ok && aimt {
			lines = append(lines, "\tallow_in_multiple_tracks = yes")
		}
		if pyStrip(d.Name) != "" {
			lines = append(lines, "\tname = "+FormatHoi4NameScalar(d.Name))
		}
		if pyStrip(d.Description) != "" {
			lines = append(lines, "\tdescription = "+pyStrip(d.Description))
		}
		if pyStrip(d.Icon) != "" {
			lines = append(lines, "\ticon = "+pyStrip(d.Icon))
		}
		if pyTruthy(d.XpCost) {
			lines = append(lines, "\txp_cost = "+pyIntStr(d.XpCost))
		}
		if pyStrip(d.XpType) != "" {
			lines = append(lines, "\txp_type = "+pyStrip(d.XpType))
		}
		if isSubdoctrine && d.Xor != nil {
			EmitVerbatimInnerBlock(&lines, "xor", d.Xor, "\t")
		}
		for _, k := range d.RawExtras.Keys() {
			rawText := d.RawExtras.Get(k)
			if k == "__bare__" {
				vals, isList := rawText.([]any)
				if !isList {
					vals = []any{rawText}
				}
				for _, v := range vals {
					if pyStrip(PyStr(v)) != "" {
						lines = append(lines, "\t"+pyStrip(PyStr(v)))
					}
				}
				continue
			}
			if k == "xor" && isSubdoctrine && d.Xor != nil {
				continue
			}
			if k == "allow_in_multiple_tracks" && isSubdoctrine && d.AllowInMultipleTracks != nil {
				continue
			}
			EmitRawExtrasBlockOrScalar(&lines, k, rawText, "\t")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type ScriptedSnippet struct {
	Kind       string
	Name       string
	Body       any
	SourceFile string
}

func ScriptedSnippetDefaultSourceFile(s *ScriptedSnippet, modDirName string) string {
	if s.Kind == "trigger" {
		return "common/scripted_triggers/zz_" + modDirName + "_scripted_triggers.txt"
	}
	return "common/scripted_effects/zz_" + modDirName + "_scripted_effects.txt"
}

func ScriptedSnippetsByFile(snippets []*ScriptedSnippet, modDirName string) *FileGroups[*ScriptedSnippet] {
	byFile := newFileGroups[*ScriptedSnippet]()
	for _, s := range snippets {
		rel := pyStrip(s.SourceFile)
		if rel == "" {
			rel = ScriptedSnippetDefaultSourceFile(s, modDirName)
		}
		rel = strings.TrimLeft(strings.ReplaceAll(rel, "\\", "/"), "/")
		byFile.add(rel, s)
	}
	return byFile
}

type FileEntry struct {
	Path    string
	Content string
}

func GenerateScriptedSnippetsScripts(snippets []*ScriptedSnippet, modDirName string) []FileEntry {
	if len(snippets) == 0 {
		return nil
	}
	var out []FileEntry
	groups := ScriptedSnippetsByFile(snippets, modDirName)
	for _, relPath := range groups.Paths {
		var parts []string
		for _, s := range groups.Rows[relPath] {
			name := pyStrip(s.Name)
			if name == "" {
				continue
			}
			body := ""
			if s.Body != nil {
				body = PyStr(s.Body)
			}
			body = pyStripChar(body, '\n')
			parts = append(parts, name+" = {\n"+body+"\n}\n\n")
		}
		if len(parts) > 0 {
			out = append(out, FileEntry{relPath, strings.Join(parts, "")})
		}
	}
	return out
}

type ScriptedLocalisation struct {
	Body       any
	SourceFile string
}

func ScriptedLocalisationDefaultSourceFile(modDirName string) string {
	return "common/scripted_localisation/zz_" + modDirName + "_scripted_localisation.txt"
}

func ScriptedLocalisationsByFile(locs []*ScriptedLocalisation, modDirName string) *FileGroups[*ScriptedLocalisation] {
	byFile := newFileGroups[*ScriptedLocalisation]()
	for _, s := range locs {
		rel := pyStrip(s.SourceFile)
		if rel == "" {
			rel = ScriptedLocalisationDefaultSourceFile(modDirName)
		}
		rel = strings.TrimLeft(strings.ReplaceAll(rel, "\\", "/"), "/")
		byFile.add(rel, s)
	}
	return byFile
}

func GenerateScriptedLocalisationsScripts(locs []*ScriptedLocalisation, modDirName string) []FileEntry {
	if len(locs) == 0 {
		return nil
	}
	var out []FileEntry
	groups := ScriptedLocalisationsByFile(locs, modDirName)
	for _, relPath := range groups.Paths {
		var parts []string
		for _, s := range groups.Rows[relPath] {
			body := ""
			if s.Body != nil {
				body = PyStr(s.Body)
			}
			body = pyStripChar(body, '\n')
			if pyStrip(body) == "" {
				continue
			}
			parts = append(parts, "defined_text = {\n"+body+"\n}\n\n")
		}
		if len(parts) > 0 {
			out = append(out, FileEntry{relPath, strings.Join(parts, "")})
		}
	}
	return out
}

func EmitNamesBlock(lines *[]string, names []any, indent string) {
	var toks []string
	for _, n := range names {
		s := PyStr(n)
		if s == "" {
			continue
		}
		if bareNameToken.MatchString(s) {
			toks = append(toks, s)
		} else {
			toks = append(toks, `"`+s+`"`)
		}
	}
	*lines = append(*lines, indent+"names = { "+strings.Join(toks, " ")+" }")
}

type IntelligenceAgency struct {
	AgencyID        string
	Picture         string
	Names           []any
	DefaultTrigger  string
	Available       string
	UpgradeBranches []any
	RawExtras       *Dict
}

func EffectiveIntelligenceAgencyNames(a *IntelligenceAgency) []any {
	if len(a.Names) > 0 {
		return append([]any(nil), a.Names...)
	}
	if a.AgencyID != "" {
		return []any{a.AgencyID}
	}
	return []any{}
}

func GenerateIntelligenceAgenciesScript(agencies []*IntelligenceAgency, resolvedManagedLogos map[string]bool) string {
	if len(agencies) == 0 {
		return ""
	}
	var lines []string
	for _, a := range agencies {
		lines = append(lines, "intelligence_agency = {")
		if a.Picture != "" {
			managed := a.Picture == IntelAgencyLogoSpriteName(a.AgencyID)
			unresolved := managed && resolvedManagedLogos != nil && !resolvedManagedLogos[a.Picture]
			if !unresolved {
				lines = append(lines, "\tpicture = "+a.Picture)
			}
		}
		names := EffectiveIntelligenceAgencyNames(a)
		EmitNamesBlock(&lines, names, "\t")
		EmitRawScriptBlock(&lines, "default", a.DefaultTrigger, "\t")
		EmitRawScriptBlock(&lines, "available", a.Available, "\t")
		for _, k := range a.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, a.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func GenerateIntelligenceAgencyUpgradesScript(agencies []*IntelligenceAgency) string {
	if len(agencies) == 0 {
		return ""
	}
	var branchOrder []string
	branchUpgrades := map[string][]*Dict{}
	seenTokens := map[string]bool{}
	for _, a := range agencies {
		for _, branchAny := range a.UpgradeBranches {
			branch, ok := branchAny.(*Dict)
			if !ok {
				continue
			}
			bname := pyStrip(pyStrOr(branch.Get("branch")))
			if bname == "" {
				continue
			}
			if _, ok := branchUpgrades[bname]; !ok {
				branchUpgrades[bname] = []*Dict{}
				branchOrder = append(branchOrder, bname)
			}
			upsAny, _ := branch.Get("upgrades").([]any)
			for _, upAny := range upsAny {
				up, ok := upAny.(*Dict)
				if !ok {
					continue
				}
				token := pyStrip(pyStrOr(up.Get("token")))
				if token == "" || seenTokens[token] {
					continue
				}
				seenTokens[token] = true
				branchUpgrades[bname] = append(branchUpgrades[bname], up)
			}
		}
	}

	if len(branchOrder) == 0 {
		return ""
	}

	var lines []string
	for _, bname := range branchOrder {
		ups := branchUpgrades[bname]
		if len(ups) == 0 {
			continue
		}
		lines = append(lines, bname+" = {")
		lines = append(lines, "")
		for _, up := range ups {
			token := pyStrip(pyStrOr(up.Get("token")))
			lines = append(lines, "\t"+token+" = {")
			if pyTruthy(up.Get("picture")) {
				lines = append(lines, "\t\tpicture = "+PyStr(up.Get("picture")))
			}
			if pyTruthy(up.Get("frame")) {
				lines = append(lines, "\t\tframe = "+PyStr(up.Get("frame")))
			}
			if pyTruthy(up.Get("sound")) {
				lines = append(lines, "\t\tsound = "+PyStr(up.Get("sound")))
			}
			EmitRawScriptBlock(&lines, "ai_will_do", up.Get("ai_will_do"), "\t\t")
			EmitRawScriptBlock(&lines, "modifiers_during_progress", up.Get("modifiers_during_progress"), "\t\t")
			EmitRawScriptBlock(&lines, "visible", up.Get("visible"), "\t\t")
			EmitRawScriptBlock(&lines, "available", up.Get("available"), "\t\t")
			levels, _ := up.Get("levels").([]any)
			for _, levelBody := range levels {
				body := ""
				if pyTruthy(levelBody) {
					body = PyStr(levelBody)
				}
				body = pyRStrip(body)
				lines = append(lines, "\t\tlevel = {")
				for _, rawLine := range splitNL(body) {
					stripped := pyRStrip(rawLine)
					if stripped != "" {
						lines = append(lines, "\t\t\t"+stripped)
					}
				}
				lines = append(lines, "\t\t}")
			}
			if re, ok := up.Get("raw_extras").(*Dict); ok {
				for _, k := range re.Keys() {
					EmitRawExtrasBlockOrScalar(&lines, k, re.Get(k), "\t\t")
				}
			}
			lines = append(lines, "\t}")
			lines = append(lines, "")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type OpinionModifier struct {
	ModifierID string
	Body       *Dict
	RawExtras  *Dict
}

func GenerateOpinionModifiersScript(modifiers []*OpinionModifier) string {
	if len(modifiers) == 0 {
		return ""
	}
	lines := []string{"opinion_modifiers = {", ""}
	for _, m := range modifiers {
		lines = append(lines, "\t"+m.ModifierID+" = {")
		for _, k := range m.Body.Keys() {
			EmitMioValue(&lines, k, m.Body.Get(k), "\t\t")
		}
		for _, k := range m.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, m.RawExtras.Get(k), "\t\t")
		}
		lines = append(lines, "\t}")
		lines = append(lines, "")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func GenerateCustomCountryLeaderTraitsScript(traits []*CustomTrait) string {
	var cl []*CustomTrait
	for _, t := range traits {
		if t.TraitType == "country_leader" {
			cl = append(cl, t)
		}
	}
	if len(cl) == 0 {
		return ""
	}
	lines := []string{"leader_traits = {", ""}
	for _, t := range cl {
		lines = append(lines, "\t"+t.TraitID+" = {")
		if !t.Random {
			lines = append(lines, "\t\trandom = no")
		}
		extras := t.ExtraFields
		if extras == nil {
			extras = NewDict()
		}
		spriteVal := extras.Get("sprite")
		if spriteVal != nil && pyStrip(PyStr(spriteVal)) != "" {
			lines = append(lines, "\t\tsprite = "+pyStrip(PyStr(spriteVal)))
		}
		if t.Modifier != nil {
			for _, k := range t.Modifier.Keys() {
				EmitMioValue(&lines, k, t.Modifier.Get(k), "\t\t")
			}
		}

		for _, sk := range clTraitStringKeys {
			if sv, ok := extras.Get(sk).(string); ok && pyStrip(sv) != "" {
				lines = append(lines, "\t\t"+sk+" = "+pyStrip(sv))
			}
		}
		for _, ik := range clTraitIntKeys {
			if rendered := FormatTraitIntScalar(extras.Get(ik)); rendered != nil {
				lines = append(lines, "\t\t"+ik+" = "+*rendered)
			}
		}

		emitTargetedModifiers(&lines, t.TargetedModifier)

		aiText := pyStrip(t.AiWillDo)
		if aiText != "" {
			lines = append(lines, "\t\tai_will_do = {")
			for _, rawLine := range splitNL(aiText) {
				lines = append(lines, "\t\t\t"+pyRStrip(rawLine))
			}
			lines = append(lines, "\t\t}")
		}

		if t.RawExtras != nil {
			for _, rk := range clTraitRawBlockKeys {
				if rv, ok := t.RawExtras.Get(rk).(string); ok && pyStrip(rv) != "" {
					EmitRawScriptBlock(&lines, rk, rv, "\t\t")
				}
			}
		}

		lines = append(lines, "\t}")
		lines = append(lines, "")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

type Specialization struct {
	SpecializationID   string
	BlueprintImage     string
	Color              []any
	ProgramBackground  string
	FacilityModel      string
	SubSpecializations []any
	RawExtras          *Dict
}

func EmitSpecialColor(lines *[]string, color []any, indent string) {
	if len(color) != 3 {
		return
	}
	parts := make([]string, 0, 3)
	for _, c := range color {
		s, ok := intOfLoose(c)
		if !ok {
			return
		}
		parts = append(parts, s)
	}
	*lines = append(*lines, indent+"color = { "+parts[0]+" "+parts[1]+" "+parts[2]+" }")
}

func intOfLoose(v any) (string, bool) {
	switch t := v.(type) {
	case bool:
		return pyIntStr(t), true
	case *big.Int:
		return t.String(), true
	case float64:
		return pyIntStr(t), true
	case string:
		s := pyStrip(t)
		if s == "" {
			return "", false
		}
		neg := false
		if s[0] == '+' || s[0] == '-' {
			neg = s[0] == '-'
			s = s[1:]
		}
		if s == "" {
			return "", false
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return "", false
			}
		}
		s = strings.TrimLeft(s, "0")
		if s == "" {
			s = "0"
		}
		if neg && s != "0" {
			s = "-" + s
		}
		return s, true
	}
	return "", false
}

func GenerateSpecializationsScript(specs []*Specialization) string {
	if len(specs) == 0 {
		return ""
	}
	var blocks []string
	for _, s := range specs {
		lines := []string{s.SpecializationID + " = {"}
		if s.BlueprintImage != "" && pyStrip(s.BlueprintImage) != "" {
			lines = append(lines, "\tblueprint_image = "+pyStrip(s.BlueprintImage))
		}
		EmitSpecialColor(&lines, s.Color, "\t")
		if s.ProgramBackground != "" && pyStrip(s.ProgramBackground) != "" {
			bg := pyStrip(s.ProgramBackground)
			if !(strings.HasPrefix(bg, `"`) && strings.HasSuffix(bg, `"`)) {
				bg = `"` + bg + `"`
			}
			lines = append(lines, "\tprogram_background = "+bg)
		}
		if s.FacilityModel != "" && pyStrip(s.FacilityModel) != "" {
			lines = append(lines, "\tfacility_model = "+pyStrip(s.FacilityModel))
		}
		if len(s.SubSpecializations) > 0 {
			parts := make([]string, 0, len(s.SubSpecializations))
			for _, x := range s.SubSpecializations {
				if pyStrip(PyStr(x)) != "" {
					parts = append(parts, pyStrip(PyStr(x)))
				}
			}
			joined := strings.Join(parts, " ")
			if joined != "" {
				lines = append(lines, "\tspecial_project_sub_specializations = { "+joined+" }")
			}
		}
		for _, k := range s.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, s.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

type SpecialProject struct {
	ProjectToken            string
	Specialization          string
	Icon                    string
	BlueprintImage          string
	Complexity              string
	PrototypeTime           string
	ProjectTags             any
	SpecialProjectParent    []any
	Allowed                 string
	Visible                 string
	Available               string
	ResourceCost            string
	ProjectOutput           string
	GenericPrototypeRewards []any
	RawExtras               *Dict
}

func GenerateSpecialProjectsScript(projects []*SpecialProject) string {
	if len(projects) == 0 {
		return ""
	}
	var blocks []string
	for _, p := range projects {
		lines := []string{p.ProjectToken + " = {"}
		if p.Specialization != "" && pyStrip(p.Specialization) != "" {
			lines = append(lines, "\tspecialization = "+pyStrip(p.Specialization))
		}
		if p.Icon != "" && pyStrip(p.Icon) != "" {
			lines = append(lines, "\ticon = "+pyStrip(p.Icon))
		}
		if p.BlueprintImage != "" && pyStrip(p.BlueprintImage) != "" {
			lines = append(lines, "\tblueprint_image = "+pyStrip(p.BlueprintImage))
		}
		if p.Complexity != "" && pyStrip(p.Complexity) != "" {
			lines = append(lines, "\tcomplexity = "+pyStrip(p.Complexity))
		}
		if p.PrototypeTime != "" && pyStrip(p.PrototypeTime) != "" {
			lines = append(lines, "\tprototype_time = "+pyStrip(p.PrototypeTime))
		}

		switch tags := p.ProjectTags.(type) {
		case []any:
			if len(tags) > 0 {
				var cleaned []string
				for _, t := range tags {
					if s := pyStrip(PyStr(t)); s != "" {
						cleaned = append(cleaned, s)
					}
				}
				if len(cleaned) == 1 {
					lines = append(lines, "\tproject_tags = "+cleaned[0])
				} else if len(cleaned) > 0 {
					lines = append(lines, "\tproject_tags = { "+strings.Join(cleaned, " ")+" }")
				}
			}
		case string:
			if pyStrip(tags) != "" {
				lines = append(lines, "\tproject_tags = "+pyStrip(tags))
			}
		}
		if len(p.SpecialProjectParent) > 0 {
			var parts []string
			for _, x := range p.SpecialProjectParent {
				if pyStrip(PyStr(x)) != "" {
					parts = append(parts, pyStrip(PyStr(x)))
				}
			}
			joined := strings.Join(parts, " ")
			if joined != "" {
				lines = append(lines, "\tspecial_project_parent = { "+joined+" }")
			}
		}
		EmitRawScriptBlock(&lines, "allowed", p.Allowed, "\t")
		EmitRawScriptBlock(&lines, "visible", p.Visible, "\t")
		EmitRawScriptBlock(&lines, "available", p.Available, "\t")
		EmitRawScriptBlock(&lines, "resource_cost", p.ResourceCost, "\t")
		EmitRawScriptBlock(&lines, "project_output", p.ProjectOutput, "\t")
		if len(p.GenericPrototypeRewards) > 0 {
			lines = append(lines, "\tgeneric_prototype_rewards = {")
			for _, tok := range p.GenericPrototypeRewards {
				t := pyStrip(PyStr(tok))
				if t != "" {
					lines = append(lines, "\t\t"+t)
				}
			}
			lines = append(lines, "\t}")
		}
		for _, k := range p.RawExtras.Keys() {
			EmitRawExtrasBlockOrScalar(&lines, k, p.RawExtras.Get(k), "\t")
		}
		lines = append(lines, "}")
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

func GenerateProjectTagsScript(projects []*SpecialProject) string {
	var seen []string
	seenSet := map[string]bool{}
	for _, p := range projects {
		var tags []any
		switch t := p.ProjectTags.(type) {
		case string:
			tags = []any{t}
		case []any:
			tags = t
		}
		for _, tk := range tags {
			tok := pyStrip(PyStr(tk))
			if tok != "" && !seenSet[tok] {
				seenSet[tok] = true
				seen = append(seen, tok)
			}
		}
	}
	if len(seen) == 0 {
		return ""
	}
	lines := []string{"project_tags = {"}
	for _, tok := range seen {
		lines = append(lines, "\t"+tok)
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n") + "\n"
}
