package emitscripts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case string:
		return t, nil
	case json.Number:
		return jsonNumber(t), nil
	case json.Delim:
		switch t {
		case '{':
			d := NewDict()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, errors.New("object key is not a string")
				}
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				d.Set(key, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return d, nil
		case '[':
			out := []any{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				out = append(out, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

func jsonNumber(n json.Number) any {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if z, ok := new(big.Int).SetString(s, 10); ok {
			return z
		}
	}
	f, err := n.Float64()
	if err != nil {
		return float64(0)
	}
	return f
}

func decodeRequest(body []byte) (*Dict, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	d, ok := v.(*Dict)
	if !ok {
		return nil, errors.New("request body is not a JSON object")
	}
	return d, nil
}

func gStr(d *Dict, k string) string {
	v := d.Get(k)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return PyStr(v)
}

func gBool(d *Dict, k string) bool { return pyTruthy(d.Get(k)) }

func gAny(d *Dict, k string) any { return d.Get(k) }

func gDict(d *Dict, k string) *Dict {
	if inner, ok := d.Get(k).(*Dict); ok {
		return inner
	}
	return nil
}

func gList(d *Dict, k string) []any {
	if l, ok := d.Get(k).([]any); ok {
		return l
	}
	return nil
}

func gStrList(d *Dict, k string) []string {
	l := gList(d, k)
	if l == nil {
		return nil
	}
	out := make([]string, 0, len(l))
	for _, item := range l {
		out = append(out, PyStr(item))
	}
	return out
}

func gBoolDef(d *Dict, k string, def bool) bool {
	v := d.Get(k)
	if v == nil {
		return def
	}
	return pyTruthy(v)
}

func reqObjList(req *Dict, key string) []*Dict {
	l := gList(req, key)
	out := make([]*Dict, 0, len(l))
	for _, item := range l {
		if d, ok := item.(*Dict); ok {
			out = append(out, d)
		} else {
			out = append(out, NewDict())
		}
	}
	return out
}

func reqStrSet(req *Dict, key string) map[string]bool {
	v := req.Get(key)
	if v == nil {
		return nil
	}
	l, ok := v.([]any)
	if !ok {
		return nil
	}
	set := make(map[string]bool, len(l))
	for _, item := range l {
		set[PyStr(item)] = true
	}
	return set
}

func stateFrom(d *Dict) *State {
	return &State{
		StateID:                 gAny(d, "state_id"),
		Name:                    gStr(d, "name"),
		Manpower:                gAny(d, "manpower"),
		Impassable:              gBool(d, "impassable"),
		ImpassableIgnoredLinks:  gAny(d, "impassable_ignored_links"),
		Resources:               gDict(d, "resources"),
		StateCategory:           gStr(d, "state_category"),
		Owner:                   gStr(d, "owner"),
		Controller:              gAny(d, "controller"),
		AddCoreOf:               gList(d, "add_core_of"),
		AddClaimBy:              gList(d, "add_claim_by"),
		RemoveCoreOf:            gList(d, "remove_core_of"),
		SetProvinceController:   gList(d, "set_province_controller"),
		VictoryPoints:           gList(d, "victory_points"),
		RawExtras:               gDict(d, "raw_extras"),
		Buildings:               gDict(d, "buildings"),
		Provinces:               gList(d, "provinces"),
		LocalSupplies:           gAny(d, "local_supplies"),
		BuildingsMaxLevelFactor: gAny(d, "buildings_max_level_factor"),
	}
}

func customTraitFrom(d *Dict) *CustomTrait {
	return &CustomTrait{
		TraitID:          gStr(d, "trait_id"),
		TraitType:        gStr(d, "trait_type"),
		Modifier:         gDict(d, "modifier"),
		AiWillDo:         gStr(d, "ai_will_do"),
		Random:           gBoolDef(d, "random", true),
		ExtraFields:      gDict(d, "extra_fields"),
		RawExtras:        gDict(d, "raw_extras"),
		TargetedModifier: gList(d, "targeted_modifier"),
	}
}

func dynamicModifierFrom(d *Dict) *DynamicModifier {
	return &DynamicModifier{
		DmID:          gStr(d, "dm_id"),
		Icon:          gStr(d, "icon"),
		Enable:        gStr(d, "enable"),
		RemoveTrigger: gStr(d, "remove_trigger"),
		Modifier:      gDict(d, "modifier"),
		RawExtras:     gDict(d, "raw_extras"),
	}
}

func autonomyStateFrom(d *Dict) *AutonomyState {
	return &AutonomyState{
		AutonomyID:        gStr(d, "autonomy_id"),
		IsPuppet:          gBool(d, "is_puppet"),
		IsDefault:         gBool(d, "is_default"),
		MinFreedomLevel:   gAny(d, "min_freedom_level"),
		ManpowerInfluence: gAny(d, "manpower_influence"),
		Rule:              gStr(d, "rule"),
		Modifier:          gDict(d, "modifier"),
		AiWeights:         gDict(d, "ai_weights"),
		RawExtras:         gDict(d, "raw_extras"),
	}
}

func mioFrom(d *Dict) *MIO {
	return &MIO{
		MioID:              gStr(d, "mio_id"),
		Include:            gStr(d, "include"),
		Icon:               gStr(d, "icon"),
		EquipmentType:      gList(d, "equipment_type"),
		ResearchCategories: gList(d, "research_categories"),
		Allowed:            gStr(d, "allowed"),
		Available:          gStr(d, "available"),
		Visible:            gStr(d, "visible"),
		Traits:             gList(d, "traits"),
		RawExtras:          gDict(d, "raw_extras"),
		Source:             gStr(d, "source"),
		ScopeKind:          gStr(d, "scope_kind"),
		CountryTag:         gStr(d, "country_tag"),
		SharedCountryTags:  gStrList(d, "shared_country_tags"),
	}
}

func doctrineFrom(d *Dict) *Doctrine {
	return &Doctrine{
		EntityKey:             gStr(d, "entity_key"),
		EntityType:            gStr(d, "entity_type"),
		FolderKey:             gStr(d, "folder_key"),
		TrackKey:              gStr(d, "track_key"),
		AllowInMultipleTracks: gAny(d, "allow_in_multiple_tracks"),
		Name:                  gStr(d, "name"),
		Description:           gStr(d, "description"),
		Icon:                  gStr(d, "icon"),
		XpCost:                gAny(d, "xp_cost"),
		XpType:                gStr(d, "xp_type"),
		Xor:                   gAny(d, "xor"),
		RawExtras:             gDict(d, "raw_extras"),
		SourceFile:            gStr(d, "source_file"),
	}
}

func scriptedSnippetFrom(d *Dict) *ScriptedSnippet {
	return &ScriptedSnippet{
		Kind:       gStr(d, "kind"),
		Name:       gStr(d, "name"),
		Body:       gAny(d, "body"),
		SourceFile: gStr(d, "source_file"),
	}
}

func scriptedLocalisationFrom(d *Dict) *ScriptedLocalisation {
	return &ScriptedLocalisation{
		Body:       gAny(d, "body"),
		SourceFile: gStr(d, "source_file"),
	}
}

func intelAgencyFrom(d *Dict) *IntelligenceAgency {
	return &IntelligenceAgency{
		AgencyID:        gStr(d, "agency_id"),
		Picture:         gStr(d, "picture"),
		Names:           gList(d, "names"),
		DefaultTrigger:  gStr(d, "default_trigger"),
		Available:       gStr(d, "available"),
		UpgradeBranches: gList(d, "upgrade_branches"),
		RawExtras:       gDict(d, "raw_extras"),
	}
}

func opinionModifierFrom(d *Dict) *OpinionModifier {
	return &OpinionModifier{
		ModifierID: gStr(d, "modifier_id"),
		Body:       gDict(d, "body"),
		RawExtras:  gDict(d, "raw_extras"),
	}
}

func specializationFrom(d *Dict) *Specialization {
	return &Specialization{
		SpecializationID:   gStr(d, "specialization_id"),
		BlueprintImage:     gStr(d, "blueprint_image"),
		Color:              gList(d, "color"),
		ProgramBackground:  gStr(d, "program_background"),
		FacilityModel:      gStr(d, "facility_model"),
		SubSpecializations: gList(d, "sub_specializations"),
		RawExtras:          gDict(d, "raw_extras"),
	}
}

func specialProjectFrom(d *Dict) *SpecialProject {
	return &SpecialProject{
		ProjectToken:            gStr(d, "project_token"),
		Specialization:          gStr(d, "specialization"),
		Icon:                    gStr(d, "icon"),
		BlueprintImage:          gStr(d, "blueprint_image"),
		Complexity:              gStr(d, "complexity"),
		PrototypeTime:           gStr(d, "prototype_time"),
		ProjectTags:             gAny(d, "project_tags"),
		SpecialProjectParent:    gList(d, "special_project_parent"),
		Allowed:                 gStr(d, "allowed"),
		Visible:                 gStr(d, "visible"),
		Available:               gStr(d, "available"),
		ResourceCost:            gStr(d, "resource_cost"),
		ProjectOutput:           gStr(d, "project_output"),
		GenericPrototypeRewards: gList(d, "generic_prototype_rewards"),
		RawExtras:               gDict(d, "raw_extras"),
	}
}

func fileEntriesJSON(entries []FileEntry) [][]string {
	out := make([][]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, []string{e.Path, e.Content})
	}
	return out
}

func mapRows[T any](req *Dict, key string, conv func(*Dict) T) []T {
	objs := reqObjList(req, key)
	out := make([]T, 0, len(objs))
	for _, o := range objs {
		out = append(out, conv(o))
	}
	return out
}

func EmitStateLinesJSON(body []byte) ([]string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return nil, err
	}
	st := NewDict()
	if inner := gDict(req, "state"); inner != nil {
		st = inner
	}
	lines := EmitStateLines(stateFrom(st))
	if lines == nil {
		lines = []string{}
	}
	return lines, nil
}

func EmitCustomUnitLeaderTraitsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateCustomUnitLeaderTraitsScript(mapRows(req, "traits", customTraitFrom)), nil
}

func EmitCustomScientistTraitsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateCustomScientistTraitsScript(mapRows(req, "traits", customTraitFrom)), nil
}

func EmitCustomCountryLeaderTraitsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateCustomCountryLeaderTraitsScript(mapRows(req, "traits", customTraitFrom)), nil
}

func EmitDynamicModifiersJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateDynamicModifiersScript(mapRows(req, "modifiers", dynamicModifierFrom)), nil
}

func EmitAutonomyStatesJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateAutonomyStatesScript(mapRows(req, "states", autonomyStateFrom)), nil
}

func EmitOpinionModifiersJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateOpinionModifiersScript(mapRows(req, "modifiers", opinionModifierFrom)), nil
}

func EmitMIOJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateMIOScript(
		mapRows(req, "mios", mioFrom),
		reqStrSet(req, "resolved_managed_icons"),
		reqStrSet(req, "resolved_managed_trait_icons"),
	), nil
}

func EmitDoctrineJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateDoctrineScript(mapRows(req, "doctrines", doctrineFrom)), nil
}

func EmitIntelAgenciesJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateIntelligenceAgenciesScript(
		mapRows(req, "agencies", intelAgencyFrom),
		reqStrSet(req, "resolved_managed_logos"),
	), nil
}

func EmitIntelAgencyUpgradesJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateIntelligenceAgencyUpgradesScript(
		mapRows(req, "agencies", intelAgencyFrom)), nil
}

func EmitSpecializationsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateSpecializationsScript(mapRows(req, "specs", specializationFrom)), nil
}

func EmitSpecialProjectsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateSpecialProjectsScript(mapRows(req, "projects", specialProjectFrom)), nil
}

func EmitProjectTagsJSON(body []byte) (string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return "", err
	}
	return GenerateProjectTagsScript(mapRows(req, "projects", specialProjectFrom)), nil
}

func EmitScriptedSnippetsJSON(body []byte) ([][]string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return nil, err
	}
	return fileEntriesJSON(GenerateScriptedSnippetsScripts(
		mapRows(req, "snippets", scriptedSnippetFrom),
		gStr(req, "mod_dir_name"))), nil
}

func EmitScriptedLocalisationsJSON(body []byte) ([][]string, error) {
	req, err := decodeRequest(body)
	if err != nil {
		return nil, err
	}
	return fileEntriesJSON(GenerateScriptedLocalisationsScripts(
		mapRows(req, "locs", scriptedLocalisationFrom),
		gStr(req, "mod_dir_name"))), nil
}
