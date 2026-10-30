package emitmisc

import (
	"sort"
	"strconv"
	"strings"
)

type YearRow struct {
	Var string
	Row int64
}

var vanillaYearRows = []YearRow{
	{"@1918", 0}, {"@1936", 2}, {"@1938", 4}, {"@1939", 6},
	{"@1940", 8}, {"@1941", 10}, {"@1942", 12}, {"@1943", 14},
	{"@1944", 16}, {"@1945", 18}, {"@1946", 20},
}

func resolveYearRows(project *NS) []YearRow {
	if project == nil {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	raw := project.List("tech_year_columns")
	if len(raw) == 0 {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	var rows []YearRow
	for _, entryAny := range raw {
		entry, ok := entryAny.([]any)
		if !ok || len(entry) != 2 {
			continue
		}
		name := PyStr(entry[0])
		var row int64
		switch v := entry[1].(type) {
		case int:
			row = int64(v)
		case int64:
			row = v
		case float64:
			row = pyIntTrunc(v)
		case string:

			n, err := strconv.ParseInt(pyStrip(v), 10, 64)
			if err != nil {
				continue
			}
			row = n
		default:
			continue
		}
		rows = append(rows, YearRow{name, row})
	}
	if len(rows) == 0 {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Row < rows[j].Row })
	return rows
}

func yearToRow(year int64, yearRows []YearRow) int64 {
	rows := yearRows
	if rows == nil {
		rows = vanillaYearRows
	}
	for i := len(rows) - 1; i >= 0; i-- {
		varName := rows[i].Var
		if len(varName) < 1 {
			continue
		}
		yr, err := strconv.ParseInt(varName[1:], 10, 64)
		if err != nil {
			continue
		}
		if year >= yr {
			return rows[i].Row
		}
	}
	return 0
}

func lineUsesStoredRow(allTechs []*NS) bool {
	for _, t := range allTechs {
		if t.Attr("position_y_is_explicit", nil) != nil {
			continue
		}
		if t.Int("position_y", 0) != 0 {
			return true
		}
	}
	return false
}

func techUsesStoredRow(tech *NS, allTechs []*NS) bool {
	intent := tech.Attr("position_y_is_explicit", nil)
	if b, ok := intent.(bool); ok {
		return b
	}
	return lineUsesStoredRow(allTechs)
}

func resolveTechRow(tech *NS, allTechs []*NS, yearRows []YearRow) int64 {
	if techUsesStoredRow(tech, allTechs) {
		return tech.Int("position_y", 0)
	}
	return yearToRow(tech.Int("start_year", 0), yearRows)
}

func formatModifierStat(v any) string {
	switch t := v.(type) {
	case bool:
		return yn(t)
	case float64:
		return FmtHOI4Float(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	}
	s := PyStr(v)
	if strings.ContainsAny(s, " \t/\\\"") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

func generateTechScript(tech *NS, allTechs []*NS, yearRows []YearRow) []string {
	var lines []string
	lines = append(lines, "\t"+tech.Str("tech_id")+" = {")
	lines = append(lines, "")

	modifiers := tech.Dict("modifiers")

	enableEquip := tech.List("enable_equipments")
	enableSub := tech.List("enable_subunits")
	enableMods := tech.List("enable_equipment_modules")

	if len(enableEquip) > 0 {
		lines = append(lines, "\t\tenable_equipments = {")
		for _, item := range enableEquip {
			if Truthy(item) {
				lines = append(lines, "\t\t\t"+PyStr(item))
			}
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	if len(enableMods) > 0 {
		lines = append(lines, "\t\tenable_equipment_modules = {")
		for _, item := range enableMods {
			if Truthy(item) {
				lines = append(lines, "\t\t\t"+PyStr(item))
			}
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	if len(enableSub) > 0 {
		lines = append(lines, "\t\tenable_subunits = {")
		for _, item := range enableSub {
			if Truthy(item) {
				lines = append(lines, "\t\t\t"+PyStr(item))
			}
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	enableBuilding := tech.Str("enable_building")
	if pyStrip(enableBuilding) != "" {
		lines = append(lines, "\t\tenable_building = { "+pyStrip(enableBuilding)+" }")
	}
	enableTactic := tech.Str("enable_tactic")
	if pyStrip(enableTactic) != "" {
		lines = append(lines, "\t\tenable_tactic = "+pyStrip(enableTactic))
	}

	xorList := tech.List("xor")
	if len(xorList) > 0 {
		var parts []string
		for _, x := range xorList {
			if Truthy(x) {
				parts = append(parts, PyStr(x))
			}
		}
		xorStr := strings.Join(parts, " ")
		if xorStr != "" {
			lines = append(lines, "\t\txor = { "+xorStr+" }")
		}
	}

	dependencies := tech.Dict("dependencies")
	if dependencies.Len() > 0 {
		lines = append(lines, "\t\tdependencies = {")
		for _, depID := range dependencies.Keys() {
			depVal, _ := dependencies.Get(depID)
			if depID != "" {
				lines = append(lines, "\t\t\t"+depID+" = "+formatModifierStat(depVal))
			}
		}
		lines = append(lines, "\t\t}")
	}

	subTechs := tech.List("sub_technologies")
	if len(subTechs) > 0 {
		var parts []string
		for _, t := range subTechs {
			if Truthy(t) {
				parts = append(parts, PyStr(t))
			}
		}
		subStr := strings.Join(parts, " ")
		if subStr != "" {
			lines = append(lines, "\t\tsub_technologies = { "+subStr+" }")
		}
	}

	if tech.Bool("doctrine", false) {
		lines = append(lines, "\t\tdoctrine = yes")
	}

	allowText := tech.Str("allow")
	if pyStrip(allowText) != "" {
		lines = append(lines, "\t\tallow = {")
		lines = append(lines, indentBlock(allowText, 3)...)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	allowBranchText := tech.Str("allow_branch")
	if pyStrip(allowBranchText) != "" {
		lines = append(lines, "\t\tallow_branch = {")
		lines = append(lines, indentBlock(allowBranchText, 3)...)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	orcLimit := tech.Str("on_research_complete_limit")
	orcText := tech.Str("on_research_complete")
	if pyStrip(orcLimit) != "" {
		lines = append(lines, "\t\ton_research_complete_limit = {")
		lines = append(lines, indentBlock(orcLimit, 3)...)
		lines = append(lines, "\t\t}")
	}
	if pyStrip(orcText) != "" {
		lines = append(lines, "\t\ton_research_complete = {")
		lines = append(lines, indentBlock(orcText, 3)...)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	xpType := tech.Str("xp_research_type")
	if xpType != "" {
		lines = append(lines, "\t\txp_research_type = "+xpType)
	}
	xpCost := tech.A("xp_boost_cost")
	if Truthy(xpCost) {
		lines = append(lines, "\t\txp_boost_cost = "+PyStr(xpCost))
	}
	xpBonus := tech.Dict("xp_research_bonus")
	if scalarVal, hasScalar := xpBonus.Get("__scalar"); hasScalar {
		lines = append(lines, "\t\txp_research_bonus = "+formatModifierStat(scalarVal))
		lines = append(lines, "")
	} else if xpBonus.Len() > 0 {
		var firstVal any
		for _, k := range xpBonus.Keys() {
			v, _ := xpBonus.Get(k)
			if isPyNumber(v) {
				firstVal = v
				break
			}
		}
		if firstVal != nil {
			lines = append(lines, "\t\txp_research_bonus = "+formatModifierStat(firstVal))
			lines = append(lines, "")
		}
	}

	modifierText := PyStr(modifiers.GetDef("__modifier", ""))
	if modifierText != "" && pyStrip(modifierText) != "" {
		lines = append(lines, indentBlock(modifierText, 2)...)
		lines = append(lines, "")
	}

	for _, category := range modifiers.Keys() {
		if strings.HasPrefix(category, "__") {
			continue
		}
		statsAny, _ := modifiers.Get(category)
		if stats, ok := asOD(statsAny); ok && stats.Len() > 0 {
			lines = append(lines, "\t\t"+category+" = {")
			for _, stat := range stats.Keys() {
				value, _ := stats.Get(stat)
				lines = append(lines, "\t\t\t"+stat+" = "+formatModifierStat(value))
			}
			lines = append(lines, "\t\t}")
			lines = append(lines, "")
		}
	}

	type fwd struct {
		leads string
		coeff float64
	}
	var forwardEdges []fwd
	fwdIdx := map[string]int{}
	for _, entryAny := range tech.List("path") {
		entry, ok := asOD(entryAny)
		if !ok {
			continue
		}
		leadsAny := entry.GetDef("leads_to_tech", nil)
		coeffAny := entry.GetDef("research_cost_coeff", 1.0)
		if !Truthy(leadsAny) {
			continue
		}
		leads := PyStr(leadsAny)
		coeff, ok2 := asFloat(coeffAny)
		if !ok2 {
			coeff = 1.0
		}
		if i, exists := fwdIdx[leads]; exists {
			forwardEdges[i].coeff = coeff
		} else {
			fwdIdx[leads] = len(forwardEdges)
			forwardEdges = append(forwardEdges, fwd{leads, coeff})
		}
	}
	techID := tech.Str("tech_id")
	for _, follower := range allTechs {
		isFollower := false
		for _, p := range follower.List("prerequisites") {
			if PyStr(p) == techID {
				isFollower = true
				break
			}
		}
		if !isFollower {
			continue
		}
		fid := follower.Str("tech_id")
		if _, exists := fwdIdx[fid]; !exists {
			fwdIdx[fid] = len(forwardEdges)
			forwardEdges = append(forwardEdges, fwd{fid, 1.0})
		}
	}
	for _, edge := range forwardEdges {
		lines = append(lines, "\t\tpath = {")
		lines = append(lines, "\t\t\tleads_to_tech = "+edge.leads)
		coeffStr := ""
		if isIntegralFloat(edge.coeff) {
			coeffStr = strconv.FormatInt(pyIntTrunc(edge.coeff), 10)
		} else {
			coeffStr = pyFloatRepr(edge.coeff)
		}
		lines = append(lines, "\t\t\tresearch_cost_coeff = "+coeffStr)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	lines = append(lines, "\t\tresearch_cost = "+PyStr(tech.A("research_cost")))
	lines = append(lines, "\t\tstart_year = "+PyStr(tech.A("start_year")))

	folderName := pyStrip(tech.Str("folder"))
	emittedFolderPlacement := false
	for _, placementAny := range tech.List("folder_placements") {
		placement, ok := asOD(placementAny)
		if !ok {
			continue
		}
		placementFolder := pyStrip(PyStr(placement.GetDef("folder", folderName)))
		if placementFolder == "" {
			placementFolder = folderName
		}
		if placementFolder == "" {
			continue
		}
		x := placement.GetDef("x", 0)
		y := placement.GetDef("y", 0)
		lines = append(lines, "\t\tfolder = {")
		lines = append(lines, "\t\t\tname = "+placementFolder)
		lines = append(lines, "\t\t\tposition = { x = "+PyStr(x)+" y = "+PyStr(y)+" }")
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
		emittedFolderPlacement = true
	}
	if !emittedFolderPlacement && folderName != "" {
		row := resolveTechRow(tech, allTechs, yearRows)
		lines = append(lines, "\t\tfolder = {")
		lines = append(lines, "\t\t\tname = "+folderName)
		lines = append(lines, "\t\t\tposition = { x = "+PyStr(tech.A("position_x"))+
			" y = "+strconv.FormatInt(row, 10)+" }")
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	if tech.Bool("force_use_small_tech_layout", false) {
		lines = append(lines, "\t\tforce_use_small_tech_layout = yes")
		lines = append(lines, "")
	}

	cats := tech.List("categories")
	if len(cats) > 0 {
		lines = append(lines, "\t\tcategories = {")
		for _, cat := range cats {
			lines = append(lines, "\t\t\t"+PyStr(cat))
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	aiText := tech.StrOr("ai_will_do", "factor = 1")
	lines = append(lines, "\t\tai_will_do = {")
	lines = append(lines, indentBlock(aiText, 3)...)
	lines = append(lines, "\t\t}")

	rawExtras := tech.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		before := len(lines)
		EmitRawExtrasBlockOrScalar(&lines, key, value, "\t\t")
		if len(lines) > before {
			lines = append(lines, "")
		}
	}

	lines = append(lines, "\t}")
	return lines
}

func isPyNumber(v any) bool {
	switch v.(type) {
	case bool, int, int64, float64:
		return true
	}

	if _, ok := v.(interface{ Sign() int }); ok {
		return ok
	}
	return false
}

func GenerateTechLineScript(techs []*NS, techLine string, project *NS, siblings []*NS) string {
	yearRows := resolveYearRows(project)

	var lines []string
	lines = append(lines, "technologies = {")
	lines = append(lines, "")

	for _, yr := range yearRows {
		lines = append(lines, "\t"+yr.Var+" = "+strconv.FormatInt(yr.Row, 10))
	}
	lines = append(lines, "")

	sortedTechs := make([]*NS, len(techs))
	copy(sortedTechs, techs)
	sort.SliceStable(sortedTechs, func(i, j int) bool {
		a, b := sortedTechs[i], sortedTechs[j]
		ay, by := a.Int("start_year", 0), b.Int("start_year", 0)
		if ay != by {
			return ay < by
		}
		return a.Int("position_x", 0) < b.Int("position_x", 0)
	})
	siblingSet := sortedTechs
	if siblings != nil {
		siblingSet = siblings
	}
	for _, tech := range sortedTechs {
		lines = append(lines, generateTechScript(tech, siblingSet, yearRows)...)
		lines = append(lines, "")
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}
