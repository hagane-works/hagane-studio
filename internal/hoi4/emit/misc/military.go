package emitmisc

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/importsvc/units"
	"haganeworks/internal/hoi4/paradox"
)

func fmtFactor(v float64) string {
	if isIntegralFloat(v) {
		return PyStr(pyIntTrunc(v))
	}
	return pyRound4Repr(v)
}

func emitDivisionBlock(dep *NS, baseTabs int, nameOverride *string) []string {
	t := strings.Repeat("\t", baseTabs)
	t2 := strings.Repeat("\t", baseTabs+1)
	lines := []string{t + "division = {"}

	name := dep.Str("division_name")
	if nameOverride != nil {
		name = *nameOverride
	}
	if name != "" {
		lines = append(lines, t2+`name = "`+name+`"`)
	}

	lines = append(lines, t2+`division_template = "`+dep.Str("division_template")+`"`)

	if loc := dep.Attr("location", nil); loc != nil {
		lines = append(lines, t2+"location = "+PyStr(asInt64(loc)))
	}

	if v := dep.Attr("start_experience_factor", nil); v != nil {
		f, _ := asFloat(v)
		lines = append(lines, t2+"start_experience_factor = "+fmtFactor(f))
	}
	if v := dep.Attr("start_equipment_factor", nil); v != nil {
		f, _ := asFloat(v)
		lines = append(lines, t2+"start_equipment_factor = "+fmtFactor(f))
	}

	rawExtras := dep.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, value, t2)
	}

	lines = append(lines, t+"}")
	return lines
}

func GenerateUnitsBlock(deployments []*NS, baseTabs int) string {
	ordered := make([]*NS, len(deployments))
	copy(ordered, deployments)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		ao, bo := a.Int("ordering", 0), b.Int("ordering", 0)
		if ao != bo {
			return ao < bo
		}
		an, bn := a.Str("division_name"), b.Str("division_name")
		if an != bn {
			return an < bn
		}
		return a.Int("id", 0) < b.Int("id", 0)
	})
	t := strings.Repeat("\t", baseTabs)
	var inner []string
	for _, dep := range ordered {
		n := int64(1)
		if c := dep.Int("count", 0); c > 0 {
			n = c
		}
		baseName := pyStrip(dep.Str("division_name"))
		if n == 1 {
			inner = append(inner, emitDivisionBlock(dep, baseTabs+1, nil)...)
		} else {
			for i := int64(1); i <= n; i++ {
				var override *string
				if baseName != "" {
					s := baseName + " " + PyStr(i)
					override = &s
				}
				inner = append(inner, emitDivisionBlock(dep, baseTabs+1, override)...)
			}
		}
	}
	if len(inner) == 0 {
		return ""
	}
	return t + "units = {\n" + strings.Join(inner, "\n") + "\n" + t + "}"
}

func emitTemplateNode(tpl *NS, baseTabs int) []string {
	t := strings.Repeat("\t", baseTabs)
	t2 := strings.Repeat("\t", baseTabs+1)
	t3 := strings.Repeat("\t", baseTabs+2)
	lines := []string{t + "division_template = {"}
	lines = append(lines, t2+`name = "`+tpl.Str("template_name")+`"`)
	if Truthy(tpl.A("division_names_group")) {
		lines = append(lines, t2+"division_names_group = "+tpl.Str("division_names_group"))
	}

	lines = append(lines, t2+"regiments = {")
	for _, rAny := range tpl.List("regiments") {
		r, ok := asOD(rAny)
		if !ok {
			continue
		}
		lines = append(lines, t3+PyStr(r.GetDef("token", nil))+" = { x = "+
			PyStr(asInt64(r.GetDef("x", int64(0))))+" y = "+
			PyStr(asInt64(r.GetDef("y", int64(0))))+" }")
	}
	lines = append(lines, t2+"}")

	support := tpl.List("support")
	if len(support) > 0 {
		lines = append(lines, t2+"support = {")
		for _, rAny := range support {
			r, ok := asOD(rAny)
			if !ok {
				continue
			}
			lines = append(lines, t3+PyStr(r.GetDef("token", nil))+" = { x = "+
				PyStr(asInt64(r.GetDef("x", int64(0))))+" y = "+
				PyStr(asInt64(r.GetDef("y", int64(0))))+" }")
		}
		lines = append(lines, t2+"}")
	}

	if v := tpl.Attr("priority", nil); v != nil {
		lines = append(lines, t2+"priority = "+PyStr(v))
	}
	if v := tpl.Attr("is_locked", nil); v != nil {
		lines = append(lines, t2+"is_locked = "+yn(Truthy(v)))
	}

	rawExtras := tpl.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, value, t2)
	}

	lines = append(lines, t+"}")
	return lines
}

func GenerateOOBFileScript(templates []*NS, tail string) string {
	ordered := make([]*NS, len(templates))
	copy(ordered, templates)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		ao, bo := a.Int("ordering", 0), b.Int("ordering", 0)
		if ao != bo {
			return ao < bo
		}
		return a.Str("template_name") < b.Str("template_name")
	})
	var lines []string
	for _, tpl := range ordered {
		lines = append(lines, emitTemplateNode(tpl, 0)...)
		lines = append(lines, "")
	}
	out := strings.Join(lines, "\n")
	tail = pyStrip(tail)
	if tail != "" {
		out = out + "\n" + tail + "\n"
	}
	return out
}

func GenerateSingleDivisionTemplateScript(tpl *NS) string {
	lines := emitTemplateNode(tpl, 0)
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\t", "    ")
	}
	return strings.Join(lines, "\n")
}

type oobSink struct{ notices []string }

func (s *oobSink) warn(msg string) {
	if s != nil {
		s.notices = append(s.notices, msg)
	}
}

func oobQuote(value string) string {
	return importsvc.EmitQuoted(value, false)
}

func oobRenderToken(token string) (string, bool) {
	rendered, err := importsvc.RenderHOI4ScalarToken(token)
	if err != nil {
		return "", false
	}
	return rendered, true
}

func oobTokenEntry(token string) (*paradox.Entry, bool) {
	entry, err := importsvc.ParseHOI4ScalarToken(token, false)
	if err != nil {
		return nil, false
	}
	return entry, true
}

func oobRowRef(kind string, row *NS) string {
	ref := kind
	if id := row.Int("id", 0); id != 0 {
		ref += " #" + PyStr(id)
	}
	var ctx []string
	if tag := pyStrip(row.Str("country_tag")); tag != "" {
		ctx = append(ctx, tag)
	}
	if file := pyStrip(row.Str("oob_file")); file != "" {
		ctx = append(ctx, file)
	}
	if len(ctx) > 0 {
		ref += " (" + strings.Join(ctx, " · ") + ")"
	}
	return ref
}

func oobShow(value string) string {
	v := pyStrip(value)
	runes := []rune(v)
	if len(runes) > 60 {
		v = string(runes[:60]) + "…"
	}
	return "\"" + v + "\""
}

func oobBadTokenReason(label, value string) string {
	if pyStrip(value) == "" {
		return "its " + label + " is empty"
	}
	return "its " + label + " " + oobShow(value) + " is not a single HOI4 token"
}

func patchScalar(entries *[]*paradox.Entry, key, value string, token bool,
	rowRef string, sink *oobSink) {

	matchIdx := -1
	for idx, entry := range *entries {
		if entry.Key != nil && *entry.Key == key && entry.Block == nil {
			matchIdx = idx
			break
		}
	}
	dropKey := func() {
		if matchIdx >= 0 {
			*entries = append((*entries)[:matchIdx:matchIdx], (*entries)[matchIdx+1:]...)
		}
	}
	if value == "" {
		dropKey()
		return
	}
	k := key
	op := "="
	var replacement *paradox.Entry
	if token {
		parsed, ok := oobTokenEntry(value)
		if !ok {
			sink.warn(rowRef + ": `" + key + "` was left out because " + oobShow(value) +
				" is not a single HOI4 token. The rest of the row still exports, but the " +
				"game will not receive that setting — fix the value in the OOB workbench " +
				"and re-export.")
			dropKey()
			return
		}
		replacement = &paradox.Entry{Key: &k, Op: &op, Scalar: parsed.Scalar, Quoted: parsed.Quoted}
	} else {
		replacement = &paradox.Entry{Key: &k, Op: &op,
			Scalar: paradox.Scalar{Kind: paradox.ScalarStr, Str: value}, Quoted: true}
	}
	if matchIdx >= 0 {
		(*entries)[matchIdx] = replacement
	} else {
		*entries = append(*entries, replacement)
	}
}

func oobIndent(text string, tabs int) []string {
	prefix := strings.Repeat("\t", tabs)
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			out = append(out, prefix+line)
		} else {
			out = append(out, line)
		}
	}
	return out
}

func oobRawLines(item *OD, tabs int) []string {
	entryAny, _ := item.Get("entry")
	data, ok := entryAny.(*units.EntryData)
	if !ok {
		return nil
	}
	script := importsvc.ToScriptEntries(units.DataToEntries([]*units.EntryData{data}), 0)
	return oobIndent(script, tabs)
}

func bodyAST(n *NS) []*units.EntryData {
	v := n.Attr("body_ast", nil)
	if d, ok := v.([]*units.EntryData); ok {
		return d
	}
	return nil
}

func airEquipmentLines(wing *NS, tabs int, groupRef string, sink *oobSink) ([]string, bool) {
	wingRef := oobRowRef("air wing", wing) + " in " + groupRef
	equipmentType, ok := oobRenderToken(wing.Str("equipment_type"))
	if !ok {
		sink.warn(wingRef + " was skipped because " +
			oobBadTokenReason("aircraft type", wing.Str("equipment_type")) +
			". HOI4 writes an air wing as `<aircraft type> = { ... }`, so it cannot be " +
			"exported without one — set the aircraft type (for example " +
			"fighter_equipment_1) in the OOB workbench, or these aircraft will keep " +
			"missing from the mod. Everything else in this order of battle still exported.")
		return nil, false
	}
	body := units.DataToEntries(bodyAST(wing))
	patchScalar(&body, "owner", wing.Str("owner"), true, wingRef, sink)
	patchScalar(&body, "amount", wing.Str("amount"), true, wingRef, sink)
	t := strings.Repeat("\t", tabs)
	lines := []string{t + equipmentType + " = {"}
	script := importsvc.ToScriptEntries(body, 0)
	if script != "" {
		lines = append(lines, oobIndent(script, tabs+1)...)
	}
	lines = append(lines, t+"}")
	return lines, true
}

func oobSort(rows []*NS) []*NS {
	out := make([]*NS, len(rows))
	copy(out, rows)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ao, bo := a.Int("ordering", 0), b.Int("ordering", 0)
		if ao != bo {
			return ao < bo
		}
		return a.Int("id", 0) < b.Int("id", 0)
	})
	return out
}

func generateAirWingsBlock(groups []*NS, baseTabs int, sink *oobSink) string {
	if len(groups) == 0 {
		return ""
	}
	t := strings.Repeat("\t", baseTabs)
	var lines []string
	for _, group := range oobSort(groups) {
		groupRef := oobRowRef("air wing group", group)
		location, locOK := oobRenderToken(group.Str("location"))
		if !locOK {
			lost := len(nsList(group.List("wings")))
			sink.warn(groupRef + " was skipped because " +
				oobBadTokenReason("spawn state", group.Str("location")) +
				". HOI4 keys every air_wings entry by the state holding the airbase " +
				"(`500 = { ... }`), so the entry cannot be exported without one — open " +
				"the group in the OOB workbench and set the state that has the air base. " +
				"Until then its " + PyStr(int64(lost)) + " air wing(s) will not exist in " +
				"game; the rest of this order of battle exported normally.")
			continue
		}
		lines = append(lines, t+"\t"+location+" = {")
		wings := oobSort(nsList(group.List("wings")))
		byOrdinal := map[int64]*NS{}
		for _, wing := range wings {
			byOrdinal[wing.Int("ordering", 0)] = wing
		}
		emittedEquipment := map[int64]bool{}
		emittedName := map[int64]bool{}
		for _, itemAny := range group.List("layout") {
			item, isOD := asOD(itemAny)
			kind := ""
			ordinal := int64(-1)
			if isOD {
				kind = PyStr(item.GetDef("kind", nil))
				ordinal = asInt64(item.GetDef("ordinal", int64(-1)))
			}
			var wing *NS
			if isOD {
				wing = byOrdinal[ordinal]
			}
			if kind == "wing_equipment" && wing != nil {
				if wingLines, wingOK := airEquipmentLines(wing, baseTabs+2, groupRef, sink); wingOK {
					lines = append(lines, wingLines...)
				}
				emittedEquipment[ordinal] = true
			} else if kind == "wing_name" && wing != nil {
				name := wing.Str("name")
				if name != "" {
					lines = append(lines, t+"\t\tname = "+oobQuote(name))
				}
				emittedName[ordinal] = true
			} else if kind == "raw" && isOD {
				lines = append(lines, oobRawLines(item, baseTabs+2)...)
			}
		}
		for _, wing := range wings {
			ordinal := wing.Int("ordering", 0)
			if !emittedEquipment[ordinal] {
				if wingLines, wingOK := airEquipmentLines(wing, baseTabs+2, groupRef, sink); wingOK {
					lines = append(lines, wingLines...)
				}
			}
			if !emittedName[ordinal] && wing.Str("name") != "" {
				lines = append(lines, t+"\t\tname = "+oobQuote(wing.Str("name")))
			}
		}
		lines = append(lines, t+"\t}")
	}

	if len(lines) == 0 {
		return ""
	}
	out := append([]string{t + "air_wings = {"}, lines...)
	out = append(out, t+"}")
	return strings.Join(out, "\n")
}

func shipLines(ship *NS, tabs int, parentRef string, sink *oobSink) []string {
	shipRef := oobRowRef("ship", ship) + " in " + parentRef
	body := units.DataToEntries(bodyAST(ship))
	patchScalar(&body, "name", ship.Str("name"), false, shipRef, sink)
	patchScalar(&body, "definition", ship.Str("definition"), true, shipRef, sink)
	patchScalar(&body, "start_experience_factor", ship.Str("start_experience_factor"), true,
		shipRef, sink)

	var equipmentEntry *paradox.Entry
	for _, entry := range body {
		if entry.Key != nil && *entry.Key == "equipment" && entry.Block != nil {
			equipmentEntry = entry
			break
		}
	}
	equipmentType := pyStrip(ship.Str("equipment_type"))
	if equipmentType != "" {
		if equipmentEntry == nil {
			k := "equipment"
			op := "="
			equipmentEntry = &paradox.Entry{Key: &k, Op: &op, Block: &paradox.Block{}}
			body = append(body, equipmentEntry)
		}
		var equipmentNode *paradox.Entry
		for _, entry := range equipmentEntry.Block.Entries {
			if entry.Key != nil && entry.Block != nil {
				equipmentNode = entry
				break
			}
		}
		if equipmentNode == nil {
			k := equipmentType
			op := "="
			equipmentNode = &paradox.Entry{Key: &k, Op: &op, Block: &paradox.Block{}}
			equipmentEntry.Block.Entries = append(equipmentEntry.Block.Entries, equipmentNode)
		} else {
			k := equipmentType
			equipmentNode.Key = &k
		}
		patchScalar(&equipmentNode.Block.Entries, "amount", ship.Str("amount"), true,
			shipRef, sink)
		patchScalar(&equipmentNode.Block.Entries, "owner", ship.Str("owner"), true,
			shipRef, sink)
		patchScalar(&equipmentNode.Block.Entries, "version_name", ship.Str("version_name"), false,
			shipRef, sink)
	}

	t := strings.Repeat("\t", tabs)
	lines := []string{t + "ship = {"}
	script := importsvc.ToScriptEntries(body, 0)
	if script != "" {
		lines = append(lines, oobIndent(script, tabs+1)...)
	}
	lines = append(lines, t+"}")
	return lines
}

var oobFieldGloss = map[string]string{
	"naval_base": "the fleet's home port province",
	"location":   "the spawn province",
}

func oobRenderFieldLine(field, value, rowRef string, sink *oobSink) (string, bool) {
	if field == "name" {
		return oobQuote(value), true
	}
	rendered, ok := oobRenderToken(value)
	if !ok {
		gloss := ""
		if g := oobFieldGloss[field]; g != "" {
			gloss = " (" + g + ")"
		}
		sink.warn(rowRef + ": `" + field + "` was left out because " + oobShow(value) +
			" is not a single HOI4 token. The row and its ships still export, but HOI4 " +
			"receives no `" + field + "`" + gloss + " — set a single token (usually one " +
			"province ID) in the OOB workbench and re-export.")
		return "", false
	}
	return rendered, true
}

func taskForceLines(taskForce *NS, tabs int, parentRef string, sink *oobSink) []string {
	taskForceRef := oobRowRef("task force", taskForce) + " in " + parentRef
	t := strings.Repeat("\t", tabs)
	lines := []string{t + "task_force = {"}
	ships := oobSort(nsList(taskForce.List("ships")))
	byOrdinal := map[int64]*NS{}
	for _, ship := range ships {
		byOrdinal[ship.Int("ordering", 0)] = ship
	}
	emittedFields := map[string]bool{}
	emittedShips := map[int64]bool{}
	for _, itemAny := range taskForce.List("layout") {
		item, isOD := asOD(itemAny)
		kind := ""
		if isOD {
			kind = PyStr(item.GetDef("kind", nil))
		}
		if kind == "field" {
			field := PyStr(item.GetDef("field", ""))
			value := taskForce.Str(field)
			if value != "" {
				if rendered, ok := oobRenderFieldLine(field, value, taskForceRef, sink); ok {
					lines = append(lines, t+"\t"+field+" = "+rendered)
				}
			}
			emittedFields[field] = true
		} else if kind == "ship" {
			ordinal := asInt64(item.GetDef("ordinal", int64(-1)))
			ship := byOrdinal[ordinal]
			if ship != nil {
				lines = append(lines, shipLines(ship, tabs+1, taskForceRef, sink)...)
				emittedShips[ordinal] = true
			}
		} else if kind == "raw" && isOD {
			lines = append(lines, oobRawLines(item, tabs+1)...)
		}
	}
	for _, field := range []string{"name", "location"} {
		value := taskForce.Str(field)
		if !emittedFields[field] && value != "" {
			if rendered, ok := oobRenderFieldLine(field, value, taskForceRef, sink); ok {
				lines = append(lines, t+"\t"+field+" = "+rendered)
			}
		}
	}
	for _, ship := range ships {
		ordinal := ship.Int("ordering", 0)
		if !emittedShips[ordinal] {
			lines = append(lines, shipLines(ship, tabs+1, taskForceRef, sink)...)
		}
	}
	lines = append(lines, t+"}")
	return lines
}

func fleetLines(fleet *NS, tabs int, sink *oobSink) []string {
	fleetRef := oobRowRef("fleet", fleet)
	t := strings.Repeat("\t", tabs)
	lines := []string{t + "fleet = {"}
	taskForces := oobSort(nsList(fleet.List("task_forces")))
	byOrdinal := map[int64]*NS{}
	for _, tf := range taskForces {
		byOrdinal[tf.Int("ordering", 0)] = tf
	}
	emittedFields := map[string]bool{}
	emittedTaskForces := map[int64]bool{}
	for _, itemAny := range fleet.List("layout") {
		item, isOD := asOD(itemAny)
		kind := ""
		if isOD {
			kind = PyStr(item.GetDef("kind", nil))
		}
		if kind == "field" {
			field := PyStr(item.GetDef("field", ""))
			value := fleet.Str(field)
			if value != "" {
				if rendered, ok := oobRenderFieldLine(field, value, fleetRef, sink); ok {
					lines = append(lines, t+"\t"+field+" = "+rendered)
				}
			}
			emittedFields[field] = true
		} else if kind == "task_force" {
			ordinal := asInt64(item.GetDef("ordinal", int64(-1)))
			taskForce := byOrdinal[ordinal]
			if taskForce != nil {
				lines = append(lines, taskForceLines(taskForce, tabs+1, fleetRef, sink)...)
				emittedTaskForces[ordinal] = true
			}
		} else if kind == "raw" && isOD {
			lines = append(lines, oobRawLines(item, tabs+1)...)
		}
	}
	for _, field := range []string{"name", "naval_base"} {
		value := fleet.Str(field)
		if !emittedFields[field] && value != "" {
			if rendered, ok := oobRenderFieldLine(field, value, fleetRef, sink); ok {
				lines = append(lines, t+"\t"+field+" = "+rendered)
			}
		}
	}
	for _, taskForce := range taskForces {
		ordinal := taskForce.Int("ordering", 0)
		if !emittedTaskForces[ordinal] {
			lines = append(lines, taskForceLines(taskForce, tabs+1, fleetRef, sink)...)
		}
	}
	lines = append(lines, t+"}")
	return lines
}

func generateFleetsBlock(fleets []*NS, baseTabs int, sink *oobSink) string {
	if len(fleets) == 0 {
		return ""
	}
	t := strings.Repeat("\t", baseTabs)
	lines := []string{t + "units = {"}
	for _, fleet := range oobSort(fleets) {
		lines = append(lines, fleetLines(fleet, baseTabs+1, sink)...)
	}
	lines = append(lines, t+"}")
	return strings.Join(lines, "\n")
}

func mergeUnitsBlocks(deployments, fleets []*NS, sink *oobSink) string {
	var entries []*paradox.Entry

	for _, part := range []struct{ label, script string }{
		{"divisions", GenerateUnitsBlock(deployments, 0)},
		{"fleets", generateFleetsBlock(fleets, 0, sink)},
	} {
		label, script := part.label, part.script
		if script == "" {
			continue
		}
		ast, _, err := paradox.Parse(script)
		if err != nil {

			sink.warn("exporter fault: the generated `" + label + "` block did not parse " +
				"and was dropped, so this order of battle is missing its " + label + ". " +
				"Your data is not at fault — please report this with the project id so it " +
				"can be fixed; re-exporting will not change the result.")
			continue
		}

		var unitsBlock *paradox.Block
		for _, e := range ast.Entries {
			if e.Key != nil && *e.Key == "units" {
				unitsBlock = e.Block
				break
			}
		}
		if unitsBlock != nil && len(unitsBlock.Entries) > 0 {
			entries = append(entries, unitsBlock.Entries...)
		}
	}
	if len(entries) == 0 {
		return ""
	}
	k := "units"
	op := "="
	wrapper := &paradox.Entry{Key: &k, Op: &op, Block: &paradox.Block{Entries: entries}}
	return pyRStrip(importsvc.ToScriptEntries([]*paradox.Entry{wrapper}, 0))
}

func GenerateAirWingsBlockTolerant(groups []*NS, baseTabs int) (string, []string) {
	sink := &oobSink{}
	return generateAirWingsBlock(groups, baseTabs, sink), sink.notices
}

func GenerateFleetsBlockTolerant(fleets []*NS, baseTabs int) (string, []string) {
	sink := &oobSink{}
	return generateFleetsBlock(fleets, baseTabs, sink), sink.notices
}

func MergeUnitsBlocksTolerant(deployments, fleets []*NS) (string, []string) {
	sink := &oobSink{}
	return mergeUnitsBlocks(deployments, fleets, sink), sink.notices
}

func GenerateAirWingsBlock(groups []*NS, baseTabs int) string {
	return generateAirWingsBlock(groups, baseTabs, nil)
}

func GenerateFleetsBlock(fleets []*NS, baseTabs int) string {
	return generateFleetsBlock(fleets, baseTabs, nil)
}

func MergeUnitsBlocks(deployments, fleets []*NS) string {
	return mergeUnitsBlocks(deployments, fleets, nil)
}
