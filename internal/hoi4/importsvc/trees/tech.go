package trees

import (
	"math"
	"math/big"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var techKnownFields = map[string]bool{
	"research_cost": true, "start_year": true, "folder": true, "categories": true,
	"path": true, "enable_equipments": true, "enable_subunits": true,
	"enable_equipment_modules": true,
	"allow":                    true, "on_research_complete": true, "on_research_complete_limit": true,
	"ai_will_do": true, "modifier": true,
	"enable_building": true, "enable_tactic": true, "xor": true,
	"dependencies":     true,
	"sub_technologies": true, "doctrine": true, "allow_branch": true,
	"xp_research_type": true, "xp_boost_cost": true, "xp_research_bonus": true,
	"force_use_small_tech_layout": true,
}

var folderToTechLine = map[string]string{
	"infantry_folder":                "infantry",
	"support_folder":                 "support",
	"artillery_folder":               "artillery",
	"armour_folder":                  "armor",
	"nsb_armour_folder":              "nsb_armor",
	"air_techs_folder":               "air_techs",
	"bba_air_techs_folder":           "bba_air_techs",
	"naval_folder":                   "naval",
	"mtgnavalfolder":                 "mtg_naval",
	"mtgnavalsupportfolder":          "naval_support",
	"industry_folder":                "industry",
	"electronics_folder":             "electronic_mechanical_engineering",
	"special_forces_doctrine_folder": "special_forces_doctrine",
}

func isUnitModifierBlock(children []*paradox.Entry) (bool, error) {
	if len(children) == 0 {
		return false, nil
	}
	for _, e := range children {
		if e.Key == nil || e.Block != nil {
			return false, nil
		}
		if _, err := pyFloatScalar(e.Scalar); err != nil {
			if err == errFloatOverflow {
				return false, err
			}
			return false, nil
		}
	}
	return true, nil
}

type orderedIntVars struct {
	keys []string
	m    map[string]*big.Int
}

func resolveVarNum(val paradox.Scalar, variables *orderedIntVars) any {
	if val.Kind == paradox.ScalarStr && strings.HasPrefix(val.Str, "@") {
		if resolved, ok := variables.get(val.Str); ok {
			return resolved
		}
		return big.NewInt(0)
	}
	switch val.Kind {
	case paradox.ScalarInt:
		return val.Int
	case paradox.ScalarFloat:
		return val.Float
	default:
		text := pyStr(val)
		if strings.Contains(text, ".") {
			if number, ok := pyFloat(text); ok {
				return number
			}
			return big.NewInt(0)
		}
		if number, ok := pyIntFromStr(text); ok {
			return number
		}
		return big.NewInt(0)
	}
}

func newOrderedIntVars() *orderedIntVars {
	return &orderedIntVars{m: map[string]*big.Int{}}
}

func (v *orderedIntVars) set(name string, z *big.Int) {
	if _, ok := v.m[name]; !ok {
		v.keys = append(v.keys, name)
	}
	v.m[name] = z
}

func (v *orderedIntVars) get(name string) (any, bool) {
	z, ok := v.m[name]
	if !ok {
		return nil, false
	}
	return z, true
}

type YearColumn struct {
	Name string
	Row  *big.Int
}

func isYearVarName(name string) bool {
	rs := []rune(name)
	if len(rs) != 5 || rs[0] != '@' {
		return false
	}
	for _, r := range rs[1:] {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func harvestIntVar(sc paradox.Scalar) (*big.Int, bool, error) {
	f, ok := pyFloat(pyStr(sc))
	if !ok {
		return nil, false, nil
	}
	z, ok := pyIntFromFloat(f)
	if !ok {
		if math.IsNaN(f) {
			return nil, false, nil
		}
		return nil, false, errIntInf
	}
	return z, true, nil
}

func ImportTechnologies(text, techLine, folderName string) ([]map[string]any, []YearColumn, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("technology")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, nil, err
	}

	expandAtConstants(ast.Entries)

	techsBlock := ppBlock(ast.Entries, "technologies")
	if techsBlock == nil {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_technologies",
			nil,
			"A file in the Technologies folder had no technologies in it — skipped.",
		))
		return []map[string]any{}, []YearColumn{}, summary, nil
	}

	variables := newOrderedIntVars()
	for _, level := range [][]*paradox.Entry{ast.Entries, techsBlock.Entries} {
		for _, e := range level {
			if e.Key != nil && *e.Key != "" && strings.HasPrefix(*e.Key, "@") && e.Block == nil {
				z, keep, err := harvestIntVar(e.Scalar)
				if err != nil {
					return nil, nil, nil, err
				}
				if keep {
					variables.set(*e.Key, z)
				}
			}
		}
	}

	detectedYearColumns := []YearColumn{}
	for _, name := range variables.keys {
		if isYearVarName(name) {
			detectedYearColumns = append(detectedYearColumns, YearColumn{
				Name: name, Row: variables.m[name],
			})
		}
	}
	sort.SliceStable(detectedYearColumns, func(i, j int) bool {
		return detectedYearColumns[i].Row.Cmp(detectedYearColumns[j].Row) < 0
	})

	prereqMap := map[string][]string{}
	for _, e := range techsBlock.Entries {
		if e.Key == nil || strings.HasPrefix(*e.Key, "@") || e.Block == nil {
			continue
		}
		for _, p := range findAll(e.Block.Entries, "path") {
			if p.Block == nil {
				continue
			}
			target := scalarOr(p.Block.Entries, "leads_to_tech", strScalar(""))
			if pyTruthy(target) {
				key := pyStr(target)
				prereqMap[key] = append(prereqMap[key], *e.Key)
			}
		}
	}

	results := []map[string]any{}
	for _, e := range techsBlock.Entries {
		if e.Key == nil || strings.HasPrefix(*e.Key, "@") || e.Block == nil {
			continue
		}
		tech, err := mapOneTech(*e.Key, e.Block.Entries, techLine, folderName, variables, summary)
		if err != nil {
			return nil, nil, nil, err
		}
		if lst, ok := prereqMap[*e.Key]; ok {
			tech["prerequisites"] = lst
		} else {
			tech["prerequisites"] = []string{}
		}
		results = append(results, tech)
	}

	summary.Total = len(results)
	return results, detectedYearColumns, summary, nil
}

func mapOneTech(
	techID string,
	body []*paradox.Entry,
	techLine string,
	folderName string,
	variables *orderedIntVars,
	summary *importsvc.ImportSummary,
) (map[string]any, error) {
	rawCost := scalarOr(body, "research_cost", floatScalar(1.0))
	rawYear := scalarOr(body, "start_year", intScalar(1936))
	researchCost, err := resolveVarFloat(rawCost, variables)
	if err != nil {
		return nil, err
	}
	startYear, err := resolveVar(rawYear, variables)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"tech_id":                     techID,
		"name":                        techID,
		"name_en":                     techID,
		"tech_line":                   techLine,
		"research_cost":               researchCost,
		"start_year":                  startYear,
		"folder":                      "",
		"position_x":                  big.NewInt(0),
		"position_y":                  big.NewInt(0),
		"position_y_is_explicit":      false,
		"prerequisites":               []string{},
		"categories":                  []string{},
		"modifiers":                   map[string]any{},
		"ai_will_do":                  "",
		"raw_extras":                  NewRawDict(),
		"enable_equipments":           []string{},
		"enable_subunits":             []string{},
		"enable_equipment_modules":    []string{},
		"enable_building":             "",
		"enable_tactic":               "",
		"xor":                         []string{},
		"dependencies":                map[string]any{},
		"sub_technologies":            []string{},
		"doctrine":                    false,
		"allow":                       "",
		"allow_branch":                "",
		"on_research_complete":        "",
		"on_research_complete_limit":  "",
		"path":                        []map[string]any{},
		"xp_research_type":            "",
		"xp_boost_cost":               big.NewInt(0),
		"xp_research_bonus":           map[string]any{},
		"force_use_small_tech_layout": false,
	}
	rawExtras := data["raw_extras"].(*RawDict)

	placements := []map[string]any{}
	for _, folderEntry := range findAll(body, "folder") {
		if folderEntry.Block == nil {
			continue
		}
		folder := pyStr(scalarOr(folderEntry.Block.Entries, "name", strScalar(folderName)))
		px, py := any(big.NewInt(0)), any(big.NewInt(0))
		explicit := false
		if posBlock := ppBlock(folderEntry.Block.Entries, "position"); truthyBlock(posBlock) {
			explicit = true
			px = resolveVarNum(scalarOr(posBlock.Entries, "x", intScalar(0)), variables)
			py = resolveVarNum(scalarOr(posBlock.Entries, "y", intScalar(0)), variables)
		}
		placements = append(placements, map[string]any{"folder": folder, "x": px, "y": py})
		if len(placements) == 1 {
			data["folder"] = folder
			if explicit {
				data["position_y_is_explicit"] = true
				data["position_x"] = px
				data["position_y"] = py
			}
		}
	}
	if len(placements) > 0 {
		data["folder_placements"] = placements
	}

	if folderStr, ok := data["folder"].(string); ok {
		if tl, hit := folderToTechLine[folderStr]; hit {
			data["tech_line"] = tl
		}
	}

	if catsBlock := ppBlock(body, "categories"); truthyBlock(catsBlock) {
		data["categories"] = bareValues(catsBlock.Entries)
	}

	modifiers := map[string]any{}
	for _, fieldName := range []string{
		"enable_equipments", "enable_subunits", "enable_equipment_modules",
	} {
		if blk := ppBlock(body, fieldName); truthyBlock(blk) {
			vals := []string{}
			for _, v := range bareValues(blk.Entries) {
				if v != "" {
					vals = append(vals, v)
				}
			}
			data[fieldName] = vals
		}
	}

	if ebBlk := ppBlock(body, "enable_building"); ebBlk != nil {
		parts := []string{}
		for _, e := range ebBlk.Entries {
			if e.Block != nil {
				continue
			}
			if e.Key != nil && *e.Key != "" {
				parts = append(parts, *e.Key+" = "+pyStr(e.Scalar))
			} else if !(e.Scalar.Kind == paradox.ScalarStr && e.Scalar.Str == "") {

				parts = append(parts, pyStr(e.Scalar))
			}
		}
		if len(parts) > 0 {
			data["enable_building"] = strings.Join(parts, " ")
		}
	}

	etScalar := scalarOr(body, "enable_tactic", strScalar(""))
	etExcluded := false
	switch etScalar.Kind {
	case paradox.ScalarStr:
		etExcluded = etScalar.Str == ""
	case paradox.ScalarInt:
		etExcluded = etScalar.Int.Sign() == 0
	case paradox.ScalarFloat:
		etExcluded = etScalar.Float == 0
	}
	if !etExcluded {
		data["enable_tactic"] = pyStr(etScalar)
	} else if etBlk := ppBlock(body, "enable_tactic"); truthyBlock(etBlk) {
		vals := []string{}
		for _, v := range bareValues(etBlk.Entries) {
			if v != "" {
				vals = append(vals, v)
			}
		}
		if len(vals) > 0 {
			data["enable_tactic"] = vals[0]
		}
	}

	for _, fieldName := range []string{"xor", "sub_technologies"} {
		if blk := ppBlock(body, fieldName); truthyBlock(blk) {
			vals := []string{}
			for _, v := range bareValues(blk.Entries) {
				if v != "" {
					vals = append(vals, v)
				}
			}
			data[fieldName] = vals
		}
	}

	if depBlock := ppBlock(body, "dependencies"); truthyBlock(depBlock) {
		deps := map[string]any{}
		for _, e := range depBlock.Entries {
			if e.Key != nil && *e.Key != "" && e.Block == nil {
				f, err := pyFloatScalar(e.Scalar)
				if err != nil {
					if err == errFloatOverflow {
						return nil, err
					}
					continue
				}
				deps[*e.Key] = f
			}
		}
		if len(deps) > 0 {
			data["dependencies"] = deps
		}
	}

	if dv := scalarOpt(body, "doctrine"); dv != nil {
		s := pyLower(pyStrip(pyStr(*dv)))
		data["doctrine"] = s == "yes" || s == "true" || s == "1"
	}

	if fv := scalarOpt(body, "force_use_small_tech_layout"); fv != nil {
		s := pyLower(pyStrip(pyStr(*fv)))
		data["force_use_small_tech_layout"] = s == "yes" || s == "true" || s == "1"
	}

	for _, fieldName := range []string{
		"allow", "allow_branch", "on_research_complete",
		"on_research_complete_limit",
	} {
		if blk := ppBlock(body, fieldName); truthyBlock(blk) {
			data[fieldName] = importsvc.BlockToText(blk, 0)
		}
	}

	pathEntries := []map[string]any{}
	for _, p := range findAll(body, "path") {
		if p.Block == nil {
			continue
		}
		leads := scalarOr(p.Block.Entries, "leads_to_tech", strScalar(""))
		coeff := scalarOr(p.Block.Entries, "research_cost_coeff", floatScalar(1.0))
		if pyTruthy(leads) {
			coeffF, err := pyFloatScalar(coeff)
			if err != nil {
				if err == errFloatOverflow {
					return nil, err
				}
				coeffF = 1.0
			}
			pathEntries = append(pathEntries, map[string]any{
				"leads_to_tech":       pyStr(leads),
				"research_cost_coeff": coeffF,
			})
		}
	}
	if len(pathEntries) > 0 {
		data["path"] = pathEntries
	}

	xpType := scalarOr(body, "xp_research_type", strScalar(""))
	if pyTruthy(xpType) {
		data["xp_research_type"] = pyStr(xpType)
	}
	xpCost := scalarOr(body, "xp_boost_cost", intScalar(0))
	xpCostI := big.NewInt(0)
	if pyTruthy(xpCost) {
		z, err := pyIntScalar(xpCost)
		if err != nil {
			if err == errIntInf {
				return nil, err
			}
			z = big.NewInt(0)
		}
		xpCostI = z
	}
	if xpCostI.Sign() != 0 {
		data["xp_boost_cost"] = xpCostI
	}
	if xpbBlock := ppBlock(body, "xp_research_bonus"); xpbBlock != nil {
		bonus := map[string]any{}
		for _, e := range xpbBlock.Entries {
			if e.Key != nil && *e.Key != "" && e.Block == nil {
				f, err := pyFloatScalar(e.Scalar)
				if err != nil {
					if err == errFloatOverflow {
						return nil, err
					}
					continue
				}
				bonus[*e.Key] = f
			}
		}
		if len(bonus) > 0 {
			data["xp_research_bonus"] = bonus
		}
	} else if xpbScalar := scalarOpt(body, "xp_research_bonus"); xpbScalar != nil {
		switch xpbScalar.Kind {
		case paradox.ScalarInt:
			data["xp_research_bonus"] = map[string]any{"__scalar": xpbScalar.Int}
		case paradox.ScalarFloat:
			data["xp_research_bonus"] = map[string]any{"__scalar": xpbScalar.Float}
		}
	}

	if modBlock := ppBlock(body, "modifier"); truthyBlock(modBlock) {
		modifiers["__modifier"] = importsvc.BlockToText(modBlock, 0)
	}

	for _, e := range body {
		if e.Key == nil || *e.Key == "" || techKnownFields[*e.Key] {
			continue
		}
		if e.Block != nil {
			isMod, err := isUnitModifierBlock(e.Block.Entries)
			if err != nil {
				return nil, err
			}
			if isMod {
				m := map[string]any{}
				for _, c := range e.Block.Entries {
					f, ferr := pyFloatScalar(c.Scalar)
					if ferr != nil {
						return nil, ferr
					}
					m[*c.Key] = f
				}
				modifiers[*e.Key] = m
			} else {
				rawExtras.Accumulate(*e.Key, importsvc.BlockToText(e.Block, 0))
				summary.AddPreserved(*e.Key)
			}
		} else {
			rawExtras.Accumulate(*e.Key, e.Scalar)
			summary.AddPreserved(*e.Key)
		}
	}

	data["modifiers"] = modifiers

	if aiBlock := ppBlock(body, "ai_will_do"); truthyBlock(aiBlock) {
		data["ai_will_do"] = importsvc.BlockToText(aiBlock, 0)
	}

	return data, nil
}
