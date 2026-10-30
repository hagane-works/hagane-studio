package emitmisc

import (
	"math/big"
	"sort"
	"strings"
)

var equipCommonFloatStats = []string{
	"soft_attack", "hard_attack", "ap_attack", "air_attack", "breakthrough",
	"defense", "armor_value", "hardness", "max_strength",
	"build_cost_ic", "reliability", "maximum_speed",
}

var equipNavyFloatStats = []string{
	"naval_speed", "fire_range", "lg_armor_piercing", "lg_attack",
	"hg_armor_piercing", "hg_attack", "torpedo_attack", "anti_air_attack",
	"shore_bombardment", "evasion", "surface_detection", "sub_attack",
	"sub_detection", "surface_visibility", "sub_visibility", "naval_range",
	"port_capacity_usage", "search_and_destroy_coordination",
	"convoy_raiding_coordination",
}

var equipAirFloatStats = []string{
	"air_defence", "air_range", "air_agility", "air_ground_attack",
	"air_bombing", "air_superiority", "naval_strike_attack",
	"naval_strike_targetting", "carrier_size",
	"default_carrier_composition_weight",
}

var equipFloatStats = func() []string {
	out := make([]string, 0,
		len(equipCommonFloatStats)+len(equipNavyFloatStats)+len(equipAirFloatStats))
	out = append(out, equipCommonFloatStats...)
	out = append(out, equipNavyFloatStats...)
	out = append(out, equipAirFloatStats...)
	return out
}()

var equipRawExtrasTypedKeys = func() map[string]bool {
	m := map[string]bool{
		"year": true, "manpower": true, "is_buildable": true,
		"can_license": true, "carrier_capable": true,
	}
	for _, s := range equipFloatStats {
		m[s] = true
	}
	return m
}()

var subUnitStringFields = []string{
	"abbreviation", "sprite", "map_icon_category", "transport", "group",
}

var subUnitIntFields = []string{"priority", "ai_priority"}

var subUnitBoolFields = []string{
	"active", "cavalry", "special_forces", "marines",
	"can_exfiltrate_from_coast", "mountaineers", "can_be_parachuted",
	"affects_speed",
}

func fmtEquipNum(v any) string {
	if f, ok := asFloat(v); ok {
		return FmtHOI4Float(f)
	}
	return PyStr(v)
}

func fmtSubUnitScalar(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case float64:
		return FmtHOI4Float(t)
	case int, int64, *big.Int:
		return PyStr(t)
	}
	token := PyStr(v)
	if token == "" {
		return `""`
	}
	quote := false
	for _, r := range token {
		if pyIsSpace(r) {
			quote = true
			break
		}
	}
	if !quote && strings.ContainsAny(token, "\"/\\") {
		quote = true
	}
	if !quote {
		return token
	}

	escaped := strings.ReplaceAll(token, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func emitEquipmentNode(eq *NS, baseTabs int) []string {
	t := strings.Repeat("\t", baseTabs)
	t2 := strings.Repeat("\t", baseTabs+1)
	t3 := strings.Repeat("\t", baseTabs+2)
	lines := []string{t + eq.Str("equipment_id") + " = {"}

	if v := eq.A("year"); v != nil {
		lines = append(lines, t2+"year = "+PyStr(v))
	}
	if Truthy(eq.Attr("is_archetype", false)) {
		lines = append(lines, t2+"is_archetype = yes")
	}
	if picture := eq.Str("picture"); picture != "" {
		lines = append(lines, t2+"picture = "+picture)
	}
	if v := eq.A("is_buildable"); v != nil {
		lines = append(lines, t2+"is_buildable = "+yesNo(v))
	}
	if s := eq.Str("equipment_type"); s != "" {
		lines = append(lines, t2+"type = "+s)
	}
	if s := eq.Str("archetype"); s != "" {
		lines = append(lines, t2+"archetype = "+s)
	}
	if s := eq.Str("parent"); s != "" {
		lines = append(lines, t2+"parent = "+s)
	}

	for _, stat := range equipFloatStats {
		if v := eq.A(stat); v != nil {
			lines = append(lines, t2+stat+" = "+fmtEquipNum(v))
		}
	}
	if v := eq.A("manpower"); v != nil {
		lines = append(lines, t2+"manpower = "+PyStr(v))
	}
	if v := eq.A("can_license"); v != nil {
		lines = append(lines, t2+"can_license = "+yesNo(v))
	}
	if v := eq.A("carrier_capable"); v != nil {
		lines = append(lines, t2+"carrier_capable = "+yesNo(v))
	}

	resources := eq.Dict("resources")
	if resources.Len() > 0 {
		lines = append(lines, t2+"resources = {")
		for _, res := range resources.Keys() {
			amount, _ := resources.Get(res)
			amt := PyStr(amount)
			if isPyNumeric(amount) {
				amt = fmtEquipNum(amount)
			}
			lines = append(lines, t3+res+" = "+amt)
		}
		lines = append(lines, t2+"}")
	}

	rawExtras := eq.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		if key == "type" && eq.Str("equipment_type") != "" {
			continue
		}
		if key == "is_archetype" && Truthy(eq.Attr("is_archetype", false)) {
			continue
		}
		if equipRawExtrasTypedKeys[key] && eq.A(key) != nil {
			continue
		}
		if key == "resources" && Truthy(eq.A("resources")) {
			continue
		}
		EmitRawExtrasBlockOrScalar(&lines, key, value, t2)
	}

	lines = append(lines, t+"}")
	return lines
}

func GenerateEquipmentFileScript(equipments []*NS) string {
	sorted := append([]*NS{}, equipments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Str("equipment_id") < sorted[j].Str("equipment_id")
	})
	lines := []string{"equipments = {", ""}
	for _, eq := range sorted {
		lines = append(lines, emitEquipmentNode(eq, 1)...)
		lines = append(lines, "")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func GenerateSingleEquipmentScript(eq *NS) string {
	lines := emitEquipmentNode(eq, 0)
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\t", "    ")
	}
	return strings.Join(lines, "\n")
}

func emitSubUnitTokenBlock(lines *[]string, key string, values []any, indent string) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmtSubUnitScalar(v))
	}
	*lines = append(*lines, indent+key+" = { "+strings.Join(parts, " ")+" }")
}

func emitSubUnitNode(row *NS, baseTabs int) []string {
	t1 := strings.Repeat("\t", baseTabs)
	t2 := strings.Repeat("\t", baseTabs+1)
	t3 := strings.Repeat("\t", baseTabs+2)
	lines := []string{t1 + row.Str("sub_unit_id") + " = {"}

	for _, field := range subUnitStringFields {
		if v := row.Attr(field, ""); Truthy(v) {
			lines = append(lines, t2+field+" = "+fmtSubUnitScalar(v))
		}
	}
	for _, field := range subUnitIntFields {
		if v := row.A(field); v != nil {
			lines = append(lines, t2+field+" = "+PyStr(v))
		}
	}
	for _, field := range subUnitBoolFields {
		if v := row.A(field); v != nil {
			lines = append(lines, t2+field+" = "+yesNo(v))
		}
	}

	emitSubUnitTokenBlock(&lines, "type", row.List("unit_types"), t2)
	emitSubUnitTokenBlock(&lines, "categories", row.List("categories"), t2)
	emitSubUnitTokenBlock(&lines, "essential", row.List("essential"), t2)

	need := row.Dict("need")
	if need.Len() > 0 {
		lines = append(lines, t2+"need = {")
		for _, equipmentID := range need.Keys() {
			amount, _ := need.Get(equipmentID)
			lines = append(lines, t3+equipmentID+" = "+fmtSubUnitScalar(amount))
		}
		lines = append(lines, t2+"}")
	}
	stats := row.Dict("stats")
	for _, stat := range stats.Keys() {
		value, _ := stats.Get(stat)
		lines = append(lines, t2+stat+" = "+fmtSubUnitScalar(value))
	}
	terrainMods := row.Dict("terrain_modifiers")
	for _, terrain := range terrainMods.Keys() {
		raw, _ := terrainMods.Get(terrain)
		lines = append(lines, t2+terrain+" = {")
		modifiers, _ := asOD(raw)
		for _, modifier := range modifiers.Keys() {
			value, _ := modifiers.Get(modifier)
			lines = append(lines, t3+modifier+" = "+fmtSubUnitScalar(value))
		}
		lines = append(lines, t2+"}")
	}

	typedKeys := map[string]bool{}
	for _, field := range subUnitStringFields {
		if Truthy(row.Attr(field, "")) {
			typedKeys[field] = true
		}
	}
	for _, field := range subUnitIntFields {
		if row.A(field) != nil {
			typedKeys[field] = true
		}
	}
	for _, field := range subUnitBoolFields {
		if row.A(field) != nil {
			typedKeys[field] = true
		}
	}
	if Truthy(row.A("unit_types")) {
		typedKeys["type"] = true
	}
	if Truthy(row.A("categories")) {
		typedKeys["categories"] = true
	}
	if Truthy(row.A("essential")) {
		typedKeys["essential"] = true
	}
	if need.Len() > 0 {
		typedKeys["need"] = true
	}
	for _, k := range stats.Keys() {
		typedKeys[k] = true
	}
	for _, k := range terrainMods.Keys() {
		typedKeys[k] = true
	}

	rawExtras := row.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		if typedKeys[key] {
			continue
		}
		value, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, value, t2)
	}
	lines = append(lines, t1+"}")
	return lines
}

func GenerateSubUnitFileScript(rows []*NS) string {
	sorted := append([]*NS{}, rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		oi, oj := sorted[i].Int("ordering", 0), sorted[j].Int("ordering", 0)
		if oi != oj {
			return oi < oj
		}
		return sorted[i].Str("sub_unit_id") < sorted[j].Str("sub_unit_id")
	})
	lines := []string{"sub_units = {", ""}
	for _, row := range sorted {
		lines = append(lines, emitSubUnitNode(row, 1)...)
		lines = append(lines, "")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func GenerateSingleSubUnitScript(row *NS) string {
	lines := emitSubUnitNode(row, 0)
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\t", "    ")
	}
	return strings.Join(lines, "\n")
}

type UnitEmissionPlan struct {
	FileKeys        []string
	Files           map[string]string
	StructuredPaths map[string]bool
	RawReplayPaths  map[string]bool
}

func newUnitEmissionPlan() *UnitEmissionPlan {
	return &UnitEmissionPlan{
		Files:           map[string]string{},
		StructuredPaths: map[string]bool{},
		RawReplayPaths:  map[string]bool{},
	}
}

func rawImportedPaths(rawFiles []*NS) map[string]bool {
	out := map[string]bool{}
	for _, row := range rawFiles {
		source, _ := row.Attr("source", nil).(string)
		if source != "imported" {
			continue
		}
		out[pyStripChars(strings.ReplaceAll(row.Str("dest_path"), "\\", "/"), "/")] = true
	}
	return out
}

func addUnitGroups(plan *UnitEmissionPlan, groups *FileGroups,
	importedRawPaths map[string]bool, generator func([]*NS) string) {

	for _, key := range groups.Keys {
		rows := groups.Get(key)
		path := pyStripChars(strings.ReplaceAll(key, "\\", "/"), "/")
		importedOnly := true
		for _, row := range rows {
			source, _ := row.Attr("source", "created").(string)
			if source != "imported" {
				importedOnly = false
				break
			}
		}
		if importedOnly && importedRawPaths[path] {

			plan.RawReplayPaths[path] = true
			continue
		}
		if _, seen := plan.Files[path]; !seen {
			plan.FileKeys = append(plan.FileKeys, path)
		}
		plan.Files[path] = generator(rows)
		plan.StructuredPaths[path] = true
	}
}

func PlanUnitEmission(equipment, subUnits, rawFiles []*NS, tag string) *UnitEmissionPlan {
	plan := newUnitEmissionPlan()
	importedRawPaths := rawImportedPaths(rawFiles)
	equipmentGroups := GroupBySourceFile(equipment, func(*NS) string {

		return "common/units/equipment/zz_" + tag + "_equipment.txt"
	}, "common/units/equipment")
	subUnitGroups := GroupBySourceFile(subUnits, func(*NS) string {
		return "common/units/" + tag + "_sub_units.txt"
	}, "common/units")
	addUnitGroups(plan, equipmentGroups, importedRawPaths, GenerateEquipmentFileScript)
	addUnitGroups(plan, subUnitGroups, importedRawPaths, GenerateSubUnitFileScript)
	return plan
}

func EmitUnits(env *Env, equipment, subUnits, rawFiles []*NS) *UnitEmissionPlan {
	plan := PlanUnitEmission(equipment, subUnits, rawFiles, env.Tag)
	authorPaths := AuthorRawOverridePaths(rawFiles)
	if env.StructuredUnitPaths == nil {
		env.StructuredUnitPaths = map[string]bool{}
	}
	for _, path := range plan.FileKeys {
		if authorPaths[path] {
			continue
		}
		env.write(env.ModDir+path, plan.Files[path])
		env.StructuredUnitPaths[path] = true
	}
	return plan
}

func yesNo(v any) string {
	if Truthy(v) {
		return "yes"
	}
	return "no"
}

func isPyNumeric(v any) bool {
	switch v.(type) {
	case bool, int, int64, *big.Int, float64:
		return true
	}
	return false
}
