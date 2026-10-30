package preview

import (
	"math/big"
	"sort"
	"strings"
)

type YearRow struct {
	Name string
	Row  *big.Int
}

var vanillaYearRows = []YearRow{
	{"@1918", big.NewInt(0)}, {"@1936", big.NewInt(2)},
	{"@1938", big.NewInt(4)}, {"@1939", big.NewInt(6)},
	{"@1940", big.NewInt(8)}, {"@1941", big.NewInt(10)},
	{"@1942", big.NewInt(12)}, {"@1943", big.NewInt(14)},
	{"@1944", big.NewInt(16)}, {"@1945", big.NewInt(18)},
	{"@1946", big.NewInt(20)},
}

type Project struct {
	TechYearColumns []any
}

func resolveYearRows(project *Project) []YearRow {
	if project == nil {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	raw := project.TechYearColumns
	if len(raw) == 0 {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	var rows []YearRow
	for _, entryAny := range raw {
		entry := asList(entryAny)
		if entry == nil || len(entry) != 2 {
			continue
		}
		z, ok := pyIntFromVal(entry[1])
		if !ok {
			continue
		}
		rows = append(rows, YearRow{pyStr(entry[0]), z})
	}
	if len(rows) == 0 {
		return append([]YearRow(nil), vanillaYearRows...)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Row.Cmp(rows[j].Row) < 0
	})
	return rows
}

func yearToRow(year any, yearRows []YearRow) *big.Int {
	rows := yearRows
	if rows == nil {
		rows = vanillaYearRows
	}
	for i := len(rows) - 1; i >= 0; i-- {
		name := rows[i].Name
		if len(name) < 1 {
			continue
		}
		yr, ok := pyIntFromVal(name[1:])
		if !ok {
			continue
		}
		if numCmp(year, yr) >= 0 {
			return rows[i].Row
		}
	}
	return big.NewInt(0)
}

func lineUsesStoredRow(allTechs []*OD) bool {
	for _, t := range allTechs {
		if attr(t, "position_y_is_explicit") != nil {
			continue
		}
		py := attr(t, "position_y")
		if py == nil || !pyTruthy(py) {
			continue
		}
		if z, ok := pyIntFromVal(py); ok {
			if z.Sign() != 0 {
				return true
			}
		}
	}
	return false
}

func techUsesStoredRow(tech *OD, allTechs []*OD) bool {
	intent := attr(tech, "position_y_is_explicit")
	if b, ok := intent.(bool); ok {
		return b
	}
	return lineUsesStoredRow(allTechs)
}

func resolveTechRow(tech *OD, allTechs []*OD, yearRows []YearRow) *big.Int {
	if techUsesStoredRow(tech, allTechs) {
		py := attr(tech, "position_y")
		if py == nil || !pyTruthy(py) {
			return big.NewInt(0)
		}
		if z, ok := pyIntFromVal(py); ok {
			return z
		}
		return big.NewInt(0)
	}
	return yearToRow(attr(tech, "start_year"), yearRows)
}

func generateTechScriptLines(tech *OD, allTechs []*OD, yearRows []YearRow) []string {
	var lines []string
	lines = append(lines, "\t"+getStr(tech, "tech_id")+" = {")
	lines = append(lines, "")

	modifiers := getOD(tech, "modifiers")
	if modifiers == nil {
		modifiers = NewOD()
	}

	enableEquip := getList(tech, "enable_equipments")
	enableSub := getList(tech, "enable_subunits")
	enableMods := getList(tech, "enable_equipment_modules")

	emitEnableList := func(key string, items []any) {
		lines = append(lines, "\t\t"+key+" = {")
		for _, item := range items {
			if pyTruthy(item) {
				lines = append(lines, "\t\t\t"+pyStr(item))
			}
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}
	if len(enableEquip) > 0 {
		emitEnableList("enable_equipments", enableEquip)
	}
	if len(enableMods) > 0 {
		emitEnableList("enable_equipment_modules", enableMods)
	}
	if len(enableSub) > 0 {
		emitEnableList("enable_subunits", enableSub)
	}

	enableBuilding := orEmptyStr(attr(tech, "enable_building"))
	if pyStrip(enableBuilding) != "" {
		lines = append(lines, "\t\tenable_building = { "+pyStrip(enableBuilding)+" }")
	}
	enableTactic := orEmptyStr(attr(tech, "enable_tactic"))
	if pyStrip(enableTactic) != "" {
		lines = append(lines, "\t\tenable_tactic = "+pyStrip(enableTactic))
	}

	xorList := getList(tech, "xor")
	if len(xorList) > 0 {
		var parts []string
		for _, x := range xorList {
			if pyTruthy(x) {
				parts = append(parts, pyStr(x))
			}
		}
		xorStr := strings.Join(parts, " ")
		if xorStr != "" {
			lines = append(lines, "\t\txor = { "+xorStr+" }")
		}
	}

	dependencies := getOD(tech, "dependencies")
	if dependencies != nil && dependencies.Len() > 0 {
		lines = append(lines, "\t\tdependencies = {")
		for _, depID := range dependencies.Keys() {
			if depID != "" {
				depVal, _ := dependencies.Get(depID)
				lines = append(lines, "\t\t\t"+depID+" = "+formatModifierStat(depVal))
			}
		}
		lines = append(lines, "\t\t}")
	}

	subTechs := getList(tech, "sub_technologies")
	if len(subTechs) > 0 {
		var parts []string
		for _, t := range subTechs {
			if pyTruthy(t) {
				parts = append(parts, pyStr(t))
			}
		}
		subStr := strings.Join(parts, " ")
		if subStr != "" {
			lines = append(lines, "\t\tsub_technologies = { "+subStr+" }")
		}
	}

	if pyTruthy(attr(tech, "doctrine")) {
		lines = append(lines, "\t\tdoctrine = yes")
	}

	allowText := orEmptyStr(attr(tech, "allow"))
	if pyStrip(allowText) != "" {
		lines = append(lines, "\t\tallow = {")
		lines = append(lines, indentBlock(allowText, 3)...)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	allowBranchText := orEmptyStr(attr(tech, "allow_branch"))
	if pyStrip(allowBranchText) != "" {
		lines = append(lines, "\t\tallow_branch = {")
		lines = append(lines, indentBlock(allowBranchText, 3)...)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	orcLimit := orEmptyStr(attr(tech, "on_research_complete_limit"))
	orcText := orEmptyStr(attr(tech, "on_research_complete"))
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

	xpType := orEmptyStr(attr(tech, "xp_research_type"))
	if xpType != "" {
		lines = append(lines, "\t\txp_research_type = "+xpType)
	}
	xpCost := attr(tech, "xp_boost_cost")
	if xpCost != nil && pyTruthy(xpCost) {
		lines = append(lines, "\t\txp_boost_cost = "+pyStr(xpCost))
	}
	xpBonus := getOD(tech, "xp_research_bonus")
	if xpBonus != nil && xpBonus.Has("__scalar") {
		v, _ := xpBonus.Get("__scalar")
		lines = append(lines, "\t\txp_research_bonus = "+formatModifierStat(v))
		lines = append(lines, "")
	} else if xpBonus != nil && xpBonus.Len() > 0 {
		var firstVal any
		for _, k := range xpBonus.Keys() {
			v, _ := xpBonus.Get(k)
			switch v.(type) {
			case *big.Int, float64:
				firstVal = v
			case bool:
				firstVal = v
			}
			if firstVal != nil {
				break
			}
		}
		if firstVal != nil {
			lines = append(lines, "\t\txp_research_bonus = "+formatModifierStat(firstVal))
			lines = append(lines, "")
		}
	}

	modifierText := orEmptyStr(modifiers.GetDef("__modifier", ""))
	if modifierText != "" && pyStrip(modifierText) != "" {
		lines = append(lines, indentBlock(modifierText, 2)...)
		lines = append(lines, "")
	}

	for _, category := range modifiers.Keys() {
		if strings.HasPrefix(category, "__") {
			continue
		}
		statsAny, _ := modifiers.Get(category)
		stats := asOD(statsAny)
		if stats != nil && stats.Len() > 0 {
			lines = append(lines, "\t\t"+category+" = {")
			for _, stat := range stats.Keys() {
				value, _ := stats.Get(stat)
				lines = append(lines, "\t\t\t"+stat+" = "+formatModifierStat(value))
			}
			lines = append(lines, "\t\t}")
			lines = append(lines, "")
		}
	}

	type edge struct {
		leadsTo string
		coeff   float64
	}
	var edges []edge
	idx := map[string]int{}
	for _, entryAny := range getList(tech, "path") {
		entry := asOD(entryAny)
		if entry == nil {
			continue
		}
		leads := entry.GetDef("leads_to_tech", nil)
		coeffAny := entry.GetDef("research_cost_coeff", 1.0)
		if pyTruthy(leads) {
			coeff := 1.0
			switch c := coeffAny.(type) {
			case float64:
				coeff = c
			case *big.Int:
				coeff, _ = new(big.Float).SetInt(c).Float64()
			case bool:
				if c {
					coeff = 1.0
				} else {
					coeff = 0.0
				}
			case string:
				if f, ok := pyFloatParse(c); ok {
					coeff = f
				} else {
					coeff = 1.0
				}
			}
			key := pyStr(leads)
			if i, ok := idx[key]; ok {
				edges[i].coeff = coeff
			} else {
				idx[key] = len(edges)
				edges = append(edges, edge{key, coeff})
			}
		}
	}
	techID := getStr(tech, "tech_id")
	for _, follower := range allTechs {
		prereqs := getList(follower, "prerequisites")
		found := false
		for _, p := range prereqs {
			if s, ok := p.(string); ok && s == techID {
				found = true
				break
			}
		}
		if found {
			key := getStr(follower, "tech_id")
			if _, ok := idx[key]; !ok {
				idx[key] = len(edges)
				edges = append(edges, edge{key, 1.0})
			}
		}
	}
	for _, e := range edges {
		lines = append(lines, "\t\tpath = {")
		lines = append(lines, "\t\t\tleads_to_tech = "+e.leadsTo)
		var coeffStr string
		if e.coeff == float64(int64(e.coeff)) {
			z, _ := pyIntFromVal(e.coeff)
			coeffStr = z.String()
		} else {
			coeffStr = pyStr(e.coeff)
		}
		lines = append(lines, "\t\t\tresearch_cost_coeff = "+coeffStr)
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	lines = append(lines, "\t\tresearch_cost = "+pyStr(attr(tech, "research_cost")))
	lines = append(lines, "\t\tstart_year = "+pyStr(attr(tech, "start_year")))

	folderName := pyStrip(orEmptyStr(attr(tech, "folder")))
	if folderName != "" {
		row := resolveTechRow(tech, allTechs, yearRows)
		lines = append(lines, "\t\tfolder = {")
		lines = append(lines, "\t\t\tname = "+folderName)
		lines = append(lines, "\t\t\tposition = { x = "+pyStr(attr(tech, "position_x"))+" y = "+row.String()+" }")
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	if pyTruthy(attr(tech, "force_use_small_tech_layout")) {
		lines = append(lines, "\t\tforce_use_small_tech_layout = yes")
		lines = append(lines, "")
	}

	cats := getList(tech, "categories")
	if len(cats) > 0 {
		lines = append(lines, "\t\tcategories = {")
		for _, cat := range cats {
			lines = append(lines, "\t\t\t"+pyStr(cat))
		}
		lines = append(lines, "\t\t}")
		lines = append(lines, "")
	}

	aiText := getStr(tech, "ai_will_do")
	if !pyTruthy(attr(tech, "ai_will_do")) {
		aiText = "factor = 1"
	}
	lines = append(lines, "\t\tai_will_do = {")
	lines = append(lines, indentBlock(aiText, 3)...)
	lines = append(lines, "\t\t}")

	rawExtras := getOD(tech, "raw_extras")
	if rawExtras != nil {
		for _, key := range rawExtras.Keys() {
			value, _ := rawExtras.Get(key)
			before := len(lines)
			emitRawExtras(&lines, key, value, "\t\t")
			if len(lines) > before {
				lines = append(lines, "")
			}
		}
	}

	lines = append(lines, "\t}")
	return lines
}

func GenerateSingleTechScript(tech *OD, allTechs []*OD, project *Project) string {
	yearRows := resolveYearRows(project)
	lines := generateTechScriptLines(tech, allTechs, yearRows)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(out, "\n")
}

func GenerateTechLineScript(techs []*OD, techLine string, project *Project, siblings []*OD) string {
	yearRows := resolveYearRows(project)

	var lines []string
	lines = append(lines, "technologies = {")
	lines = append(lines, "")

	for _, yr := range yearRows {
		lines = append(lines, "\t"+yr.Name+" = "+yr.Row.String())
	}
	lines = append(lines, "")

	sortedTechs := append([]*OD(nil), techs...)
	sort.SliceStable(sortedTechs, func(i, j int) bool {
		a, b := sortedTechs[i], sortedTechs[j]
		c := numCmp(attr(a, "start_year"), attr(b, "start_year"))
		if c != 0 {
			return c < 0
		}
		return numCmp(attr(a, "position_x"), attr(b, "position_x")) < 0
	})
	siblingSet := sortedTechs
	if siblings != nil {
		siblingSet = append([]*OD(nil), siblings...)
	}
	for _, tech := range sortedTechs {
		lines = append(lines, generateTechScriptLines(tech, siblingSet, yearRows)...)
		lines = append(lines, "")
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}
