package worldout

import (
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

var stateFnIllegalRe = regexp.MustCompile(`[:*?"<>|\x00-\x1f]`)

func stateExportFilename(st *OMap, sid int64, sname string, vanillaFns map[int64]string) string {
	raw := strings.ReplaceAll(pyStrOrEmpty(eGet(st, "source_file")), "\\", "/")
	base := raw
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		base = raw[i+1:]
	}
	fn := pyStrip(stateFnIllegalRe.ReplaceAllString(base, ""))
	if fn != "" && strings.Trim(fn, ". ") != "" &&
		strings.HasSuffix(toLower(fn), ".txt") {
		return fn
	}
	if v, ok := vanillaFns[sid]; ok && v != "" {
		return v
	}
	return fmtInt(sid) + "-" + sname + ".txt"
}

func emitVerbatimInnerBlock(lines *[]string, key string, rawText Value, indent string) {
	var body []string
	for _, ln := range pySplitlines(pyStrOrEmpty(rawText)) {
		if s := pyStrip(ln); s != "" {
			body = append(body, s)
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

func emitStateLines(st *OMap, overridesStartTransfers bool) []string {
	sidV, _ := st.Get("state_id")
	sname := pyStrip(eStrOrEmpty(st, "name"))
	if sname == "" {
		sname = "STATE_" + pyStr(sidV)
	}
	lines := []string{"state = {"}
	lines = append(lines, "\tid = "+pyStr(sidV))
	lines = append(lines, "\tname = \""+sname+"\"")

	if pyTruthy(eGet(st, "manpower")) {
		lines = append(lines, "\tmanpower = "+fmtInt(eInt(st, "manpower")))
	}

	if pyTruthy(eGet(st, "impassable")) {
		lines = append(lines, "\timpassable = yes")
	}

	if iil, ok := st.Get("impassable_ignored_links"); ok && iil != nil {
		emitVerbatimInnerBlock(&lines, "impassable_ignored_links", iil, "\t")
	}

	resources := eMap(st, "resources")
	if resources.Len() > 0 {
		lines = append(lines, "\tresources = {")
		for _, rkey := range resources.Keys() {
			rval, _ := resources.Get(rkey)
			lines = append(lines, "\t\t"+rkey+" = "+pyStr(rval))
		}
		lines = append(lines, "\t}")
	}

	if pyTruthy(eGet(st, "state_category")) {
		lines = append(lines, "\tstate_category = "+pyStr(eGet(st, "state_category")))
	}

	var historyLines []string
	if pyTruthy(eGet(st, "owner")) {
		historyLines = append(historyLines, "\t\towner = "+pyStr(eGet(st, "owner")))
	}
	if pyTruthy(eGet(st, "controller")) {
		historyLines = append(historyLines, "\t\tcontroller = "+pyStr(eGet(st, "controller")))
	}
	for _, core := range eList(st, "add_core_of") {
		if pyTruthy(core) {
			historyLines = append(historyLines, "\t\tadd_core_of = "+pyStr(core))
		}
	}
	for _, claim := range eList(st, "add_claim_by") {
		if pyTruthy(claim) {
			historyLines = append(historyLines, "\t\tadd_claim_by = "+pyStr(claim))
		}
	}
	for _, rc := range eList(st, "remove_core_of") {
		if pyTruthy(rc) {
			historyLines = append(historyLines, "\t\tremove_core_of = "+pyStr(rc))
		}
	}

	for _, spcV := range eList(st, "set_province_controller") {
		spc, ok := spcV.(*OMap)
		if !ok {
			continue
		}
		pid, hasPid := spc.Get("province_id")
		ctrl, _ := spc.Get("controller")
		if hasPid && pid != nil && pyTruthy(ctrl) {
			historyLines = append(historyLines,
				"\t\tset_province_controller = { province = "+pyStr(pid)+
					" controller = "+pyStr(ctrl)+" }")
		}
	}

	for _, vpV := range eList(st, "victory_points") {
		vp, ok := vpV.(*OMap)
		if !ok {
			continue
		}
		pid, hasPid := vp.Get("province_id")
		pval, hasVal := vp.Get("value")
		if !hasPid || pid == nil || !hasVal || pval == nil {
			continue
		}
		historyLines = append(historyLines,
			"\t\tvictory_points = { "+pyStr(pid)+" "+pyStr(pval)+" }")
	}

	rawExtras := eMap(st, "raw_extras")
	stateBuildings := eMap(st, "buildings")
	provinceBuildingsRaw := ""
	if rawExtras != nil {
		if bv, ok := rawExtras.Get("buildings"); ok {
			provinceBuildingsRaw = pyStrip(pyStrOrEmpty(bv))
		}
	}
	if stateBuildings.Len() > 0 || provinceBuildingsRaw != "" {
		historyLines = append(historyLines, "\t\tbuildings = {")
		if stateBuildings != nil {
			for _, bkey := range stateBuildings.Keys() {
				bval, _ := stateBuildings.Get(bkey)
				historyLines = append(historyLines, "\t\t\t"+bkey+" = "+pyStr(bval))
			}
		}
		for _, vl := range pySplitlines(provinceBuildingsRaw) {
			if pyStrip(vl) != "" {
				if strings.HasPrefix(vl, "\t") {
					historyLines = append(historyLines,
						"\t\t\t"+strings.TrimRight(vl, " \t\n\r\v\f"))
				} else {
					historyLines = append(historyLines, "\t\t\t"+pyStrip(vl))
				}
			}
		}
		historyLines = append(historyLines, "\t\t}")
	}

	if rawExtras != nil {
		elseFollows := false
		for _, rk := range rawExtras.Keys() {
			if k := toLower(rk); k == "history.else" || k == "history.else_if" {
				elseFollows = true
			}
		}
		var leftOut []string
		for _, rk := range rawExtras.Keys() {
			if !strings.HasPrefix(rk, "history.") {
				continue
			}
			innerKey := rk[len("history."):]
			rv, _ := rawExtras.Get(rk)
			if overridesStartTransfers {
				out, targets, keep := historyWithoutStartTransfers(innerKey, rv, elseFollows)
				leftOut = append(leftOut, targets...)
				if !keep {
					continue
				}
				rv = out
			}
			emitRawExtrasBlockOrScalar(&historyLines, innerKey, rv, "\t\t")
		}
		if len(leftOut) > 0 {
			historyLines = append(historyLines, "\t\t"+startTransferNote(leftOut, pyStr(eGet(st, "owner"))))
		}
	}

	if len(historyLines) > 0 {
		lines = append(lines, "\thistory = {")
		lines = append(lines, historyLines...)
		lines = append(lines, "\t}")
	}

	provinces := eList(st, "provinces")
	if len(provinces) > 0 {
		var provTokens []string
		for _, p := range provinces {
			provTokens = append(provTokens, pyStr(p))
		}
		lines = append(lines, "\tprovinces = {")
		lines = append(lines, "\t\t"+strings.Join(provTokens, " "))
		lines = append(lines, "\t}")
	}

	if ls, ok := st.Get("local_supplies"); ok && ls != nil {
		lines = append(lines, "\tlocal_supplies = "+hoi4FloatOf(ls))
	}

	if bmlf, ok := st.Get("buildings_max_level_factor"); ok && bmlf != nil {
		lines = append(lines, "\tbuildings_max_level_factor = "+hoi4FloatOf(bmlf))
	}

	if rawExtras != nil {
		for _, rk := range rawExtras.Keys() {
			if rk == "buildings" || strings.HasPrefix(rk, "history.") {
				continue
			}
			if rk == "impassable_ignored_links" {
				if v, ok := st.Get("impassable_ignored_links"); ok && v != nil {
					continue
				}
			}
			rv, _ := rawExtras.Get(rk)
			emitRawExtrasBlockOrScalar(&lines, rk, rv, "\t")
		}
	}

	lines = append(lines, "}")
	return lines
}

func generateStatesLocalisation(states []*OMap, lang, suffix string) string {
	var lines []string
	for _, st := range states {
		sidV, hasSid := st.Get("state_id")
		if !hasSid || sidV == nil {
			continue
		}
		locs := eMap(st, "display_name_locs")
		val := ""
		if locs != nil {
			if v, ok := locs.Get(lang); ok {
				val = pyStrip(pyStrOrEmpty(v))
			}
		}
		key := "STATE_" + pyStr(sidV)
		if isPlaceholderDisplayName(val, key) {
			continue
		}
		lines = append(lines, ymlLine(key, val, false))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(append([]string{suffix + ":"}, lines...), "\n")
}

func (c *emitCtx) emitStates() {
	modDir := c.modDir
	project := c.s.Project

	knownTags := map[string]bool{}
	for tag := range c.t.VanillaTags {
		knownTags[tag] = true
	}
	for _, country := range c.s.Countries {
		if pyTruthy(eGet(country, "tag")) {
			knownTags[toUpper(pyStrip(eStr(country, "tag")))] = true
		}
	}
	if pyTruthy(eGet(project, "country_tag")) {
		knownTags[toUpper(pyStrip(eStr(project, "country_tag")))] = true
	}

	emittedStateIDs := map[int64]bool{}
	var bufferOrder []string
	buffers := map[string][]string{}

	for _, st := range c.s.States {
		sidV, hasSid := st.Get("state_id")
		if !hasSid || sidV == nil {
			continue
		}
		sid, _ := intVal(sidV)
		if emittedStateIDs[sid] {
			continue
		}
		var name any = exportmsg.Message{Key: "orphan.unnamed"}
		if named := pyStrip(eStrOrEmpty(st, "name")); named != "" {
			name = named
		}
		orphanFinding := func(field, tag string) exportmsg.Message {
			return exportmsg.Message{Key: "orphan.state", Args: exportmsg.Args{
				"state": pyStr(sidV), "name": name, "field": field, "tag": tag,
			}}
		}
		for _, fieldTag := range [][2]string{
			{"owner", pyStrOrEmpty(eGet(st, "owner"))},
			{"controller", pyStrOrEmpty(eGet(st, "controller"))},
		} {
			t := pyStrip(fieldTag[1])
			if t != "" && !knownTags[toUpper(t)] {
				c.orphan = append(c.orphan, orphanFinding(fieldTag[0], t))
			}
		}
		for _, coreV := range eList(st, "add_core_of") {
			c2 := pyStrip(pyStrOrEmpty(coreV))
			if c2 != "" && !knownTags[toUpper(c2)] {
				c.orphan = append(c.orphan, orphanFinding("add_core_of", c2))
			}
		}
		stLines := emitStateLines(st, ownerOverridesStartTransfers(st, c.t.StateOwners[sid]))
		nonComment := false
		for _, ln := range stLines {
			s := pyStrip(ln)
			if s != "" && !strings.HasPrefix(s, "#") {
				nonComment = true
				break
			}
		}
		if !nonComment {
			continue
		}
		sname := pyStrip(eStrOrEmpty(st, "name"))
		if sname == "" {
			sname = "STATE_" + fmtInt(sid)
		}
		emittedStateIDs[sid] = true
		stateFn := stateExportFilename(st, sid, sname, c.t.StateFilenames)
		if buf, ok := buffers[stateFn]; ok {
			buf = append(buf, "")
			buf = append(buf, stLines...)
			buffers[stateFn] = buf
		} else {
			bufferOrder = append(bufferOrder, stateFn)
			buffers[stateFn] = append([]string{}, stLines...)
		}
	}

	for _, stateFn := range bufferOrder {
		c.write(modDir+"history/states/"+stateFn,
			strings.Join(buffers[stateFn], "\n")+"\n")
	}

	for _, tr := range c.triples {
		statesLoc := generateStatesLocalisation(c.s.States, tr.Code, tr.Suffix)
		if statesLoc == "" {
			continue
		}
		c.write(
			modDir+"localisation/"+tr.Subdir+"/"+c.s.ModDirName+"_states_"+tr.Suffix+".yml",
			bomStr+statesLoc)
	}
}
