package lint

import (
	"sort"
	"strings"

	"haganeworks/internal/project"
)

func profileProject(p *project.Project) (map[string]any, error) {
	tables := map[string][]project.Row{}
	for _, table := range []string{"event_namespaces", "events", "decision_categories", "decisions",
		"ideas", "focus_trees", "focus_nodes", "history_files", "characters"} {
		rows, err := p.Rows(table)
		if err != nil {
			return nil, err
		}
		tables[table] = rows
	}
	findings := []LintFinding{}

	nsByID := map[int64][2]*string{}
	for _, row := range tables["event_namespaces"] {
		id, _ := rowID(row, "id")
		nsByID[id] = [2]*string{textPtr(row, "namespace"), textPtr(row, "country_tag")}
	}

	eventCount := 0
	perNS := map[int64]int{}
	for _, row := range tables["events"] {
		eventID := text(row, "event_id")
		titleEn, ctag := textPtr(row, "title_en"), textPtr(row, "country_tag")
		isTriggered := boolOr(row, "is_triggered_only", true)
		mtth, trigger := textPtr(row, "mean_time_to_happen"), textPtr(row, "trigger")
		immediate, after := textPtr(row, "immediate"), textPtr(row, "after")
		eventCount++
		file := "events/events.txt"
		if nsID, ok := rowID(row, "namespace_id"); ok {
			perNS[nsID]++
			if pair, ok := nsByID[nsID]; ok {
				base := deref(ctag)
				if base == "" {
					base = deref(pair[0])
					if base == "" {
						base = "events"
					}
				}
				file = "events/" + base + ".txt"
			}
		}
		eid := eventID
		ref := perfEntityRef("event", &eid, perfRefNameStr(deref(titleEn), eventID), nilIfEmpty(ctag))
		if !isTriggered && deref(mtth) != "" {
			m := deref(mtth)
			perfScanHotField(m, ref, file, "Event MTTH", &findings)
			hasFloor := perfDaysRe.MatchString(m) || perfMYRe.MatchString(m)
			if perfFactorRe.MatchString(m) && !hasFloor {
				f := file
				findings = append(findings, LintFinding{
					Severity: "warning", Code: "perf.mtth_no_floor",
					Title:      "Polled event MTTH has no base interval",
					Reason:     "This non-triggered event's `mean_time_to_happen` uses a `factor` modifier but defines no base `days`/`months`/`years`. The engine polls it aggressively for every eligible scope.",
					Suggestion: "Add a base `days = N` (or `months`/`years`) so the engine polls less often, or make the event `is_triggered_only` and fire it explicitly.",
					EntityRef:  ref, File: &f,
					Detail: map[string]any{"field": "mean_time_to_happen"},
				})
			}
		}
		if !isTriggered && deref(trigger) != "" {
			perfScanHotField(deref(trigger), ref, file, "Event trigger (polled)", &findings)
		}
		perfScanNested(deref(immediate), ref, file, "Event immediate effect", &findings)
		perfScanNested(deref(after), ref, file, "Event after effect", &findings)
	}

	catByID := map[int64][3]*string{}
	for _, row := range tables["decision_categories"] {
		id, _ := rowID(row, "id")
		catByID[id] = [3]*string{textPtr(row, "category_id"), textPtr(row, "name_en"), textPtr(row, "country_tag")}
	}

	decisionCount := 0
	perCat := map[int64]int{}
	for _, row := range tables["decisions"] {
		decisionID := text(row, "decision_id")
		nameEn, ctag := textPtr(row, "name_en"), textPtr(row, "country_tag")
		decisionCount++
		catToken := "decisions"
		if catID, ok := rowID(row, "category_id"); ok {
			perCat[catID]++
			if c, ok := catByID[catID]; ok && deref(c[0]) != "" {
				catToken = deref(c[0])
			}
		}
		base := deref(ctag)
		if base == "" {
			base = catToken
		}
		file := "common/decisions/" + base + ".txt"
		did := decisionID
		ref := perfEntityRef("decision", &did, perfRefNameStr(deref(nameEn), decisionID), nilIfEmpty(ctag))
		for _, fld := range []struct{ text, label string }{
			{text(row, "available"), "Decision available"},
			{text(row, "visible"), "Decision visible"},
			{text(row, "activation"), "Decision activation (mission auto-trigger)"},
		} {
			perfScanHotField(fld.text, ref, file, fld.label, &findings)
			if fld.text != "" && perfFanoutRe.MatchString(fld.text) {
				var hitNames []string
				for _, m := range perfFanoutRe.FindAllStringSubmatch(fld.text, -1) {
					hitNames = append(hitNames, m[1])
				}
				hits := perfSortedUnique(hitNames)
				names := make([]string, len(hits))
				for i, h := range hits {
					names[i] = h.(string)
				}
				f := file
				findings = append(findings, LintFinding{
					Severity: "info", Code: "perf.expensive_available_recheck",
					Title: "Database-wide check re-runs on every UI refresh",
					Reason: fld.label + " uses " + pyListRepr(names) + ", which scans the game " +
						"database. Decision availability is re-checked whenever its tab is open.",
					Suggestion: "If the value rarely changes, cache it in a country variable updated on a slower cadence and compare the variable here instead.",
					EntityRef:  ref, File: &f,
					Detail: map[string]any{"field": fld.label, "checks": hits},
				})
			}
		}
		perfScanNested(text(row, "complete_effect"), ref, file, "Decision complete_effect", &findings)
	}

	ideaCount := 0
	for _, row := range tables["ideas"] {
		ideaID := text(row, "idea_id")
		ctag := textPtr(row, "country_tag")
		ideaCount++
		base := deref(ctag)
		if base == "" {
			base = "ideas"
		}
		file := "common/ideas/" + base + ".txt"
		iid := ideaID
		ref := perfEntityRef("idea", &iid, perfRefNameStr(text(row, "name_en"), text(row, "name"), ideaID), nilIfEmpty(ctag))
		perfScanHotField(text(row, "do_effect"), ref, file, "Idea do_effect (continuous / on-tick)", &findings)
		perfScanHotField(text(row, "available"), ref, file, "Idea available", &findings)
		perfScanHotField(text(row, "visible"), ref, file, "Idea visible", &findings)
	}

	treeList := tables["focus_trees"]
	treeCount, totalNodes := len(treeList), 0
	nodesByTree := map[int64][]project.Row{}
	treeIDs := map[int64]bool{}
	for _, tree := range treeList {
		id, _ := rowID(tree, "id")
		treeIDs[id] = true
	}
	for _, node := range tables["focus_nodes"] {
		treeID, _ := rowID(node, "focus_tree_id")
		if treeIDs[treeID] {
			nodesByTree[treeID] = append(nodesByTree[treeID], node)
		}
	}
	for _, t := range treeList {
		id, _ := rowID(t, "id")
		treeIDText, ctag := text(t, "tree_id"), textPtr(t, "country_tag")
		raw := text(t, "source_file")
		if raw == "" {
			raw = treeIDText + ".txt"
		}
		raw = strings.ReplaceAll(raw, "\\", "/")
		base := raw
		if i := strings.LastIndexByte(raw, '/'); i >= 0 {
			base = raw[i+1:]
		}
		if base == "" {
			base = treeIDText + ".txt"
		}
		file := "common/national_focus/" + base
		treeNodes := nodesByTree[id]
		nodeCount := len(treeNodes)
		for _, node := range treeNodes {
			fid := text(node, "focus_id")
			nRef := perfEntityRef("focus", &fid,
				perfRefNameStr(text(node, "name_en"), text(node, "name"), fid), nilIfEmpty(ctag))
			perfScanHotField(text(node, "available"), nRef, file, "Focus available (polled)", &findings)
			perfScanHotField(text(node, "bypass"), nRef, file, "Focus bypass (polled)", &findings)
			perfScanNested(text(node, "completion_reward"), nRef, file, "Focus completion_reward", &findings)
		}
		totalNodes += nodeCount
		if nodeCount > perfThreshFocusNodesPerTree {
			tid := treeIDText
			tRef := perfEntityRef("focus", &tid, perfRefNameStr(treeIDText), nilIfEmpty(ctag))
			f := file
			findings = append(findings, LintFinding{
				Severity: "warning", Code: "perf.oversized_focus_tree",
				Title: "Very large focus tree",
				Reason: "Focus tree '" + tid + "' has " + itoaInt(nodeCount) + " focuses " +
					"(> " + itoaInt(perfThreshFocusNodesPerTree) + "). The generated " +
					"national_focus file is large to parse on load and hard to maintain.",
				Suggestion: "Split shared branches into a `shared_focus` pool or a second tree, or gate late-game branches behind `allow_branch` so they don't all load eagerly.",
				EntityRef:  tRef, File: &f,
				Detail: map[string]any{"focus_count": nodeCount},
			})
		}
	}
	if totalNodes > perfThreshTotalFocusNodes {
		findings = append(findings, LintFinding{
			Severity: "info", Code: "perf.total_focus_weight",
			Title: "Project has a very large number of focuses",
			Reason: "The project defines " + itoaInt(totalNodes) + " focuses across " +
				itoaInt(treeCount) + " tree(s). Large focus databases add to load time and save size.",
			Suggestion: "Consider whether all trees need to load for every country, and lean on shared pools where branches repeat.",
			Detail:     map[string]any{"total_focus_count": totalNodes, "tree_count": treeCount},
		})
	}

	nsIDs := make([]int64, 0, len(perNS))
	for id := range perNS {
		nsIDs = append(nsIDs, id)
	}
	sort.Slice(nsIDs, func(i, j int) bool { return nsIDs[i] < nsIDs[j] })
	for _, nsID := range nsIDs {
		count := perNS[nsID]
		if count <= perfThreshEventsPerNamespace {
			continue
		}
		name := "namespace#" + itoa64(nsID)
		var ctag *string
		if pair, ok := nsByID[nsID]; ok {
			if deref(pair[0]) != "" {
				name = deref(pair[0])
			}
			ctag = pair[1]
		}
		f := "events/" + name + ".txt"
		n := name
		findings = append(findings, LintFinding{
			Severity: "warning", Code: "perf.oversized_event_file",
			Title: "Very large event namespace",
			Reason: "Event namespace '" + name + "' has " + itoaInt(count) + " events " +
				"(> " + itoaInt(perfThreshEventsPerNamespace) + "); the generated " +
				"events/" + name + ".txt is large to parse on load.",
			Suggestion: "Split the namespace into themed sub-namespaces so each generated event file stays small and editable.",
			EntityRef:  perfEntityRef("event", &n, &n, nilIfEmpty(ctag)),
			File:       &f,
			Detail:     map[string]any{"event_count": count},
		})
	}
	catIDs := make([]int64, 0, len(perCat))
	for id := range perCat {
		catIDs = append(catIDs, id)
	}
	sort.Slice(catIDs, func(i, j int) bool { return catIDs[i] < catIDs[j] })
	for _, catID := range catIDs {
		count := perCat[catID]
		if count <= perfThreshDecisionsPerCat {
			continue
		}
		name := "category#" + itoa64(catID)
		var nameEn, ctag *string
		if c, ok := catByID[catID]; ok {
			if deref(c[0]) != "" {
				name = deref(c[0])
			}
			nameEn, ctag = c[1], c[2]
		}
		f := "common/decisions/categories/" + name + ".txt"
		n := name
		findings = append(findings, LintFinding{
			Severity: "info", Code: "perf.oversized_decision_category",
			Title: "Very large decision category",
			Reason: "Decision category '" + name + "' contains " + itoaInt(count) + " decisions " +
				"(> " + itoaInt(perfThreshDecisionsPerCat) + "). Every visible " +
				"decision's `available`/`visible` is re-checked while the tab is open.",
			Suggestion: "Group decisions into more categories so fewer are evaluated/visible at once, and gate rarely-relevant ones behind a category `visible` trigger.",
			EntityRef:  perfEntityRef("decision", &n, perfRefNameStr(deref(nameEn), name), nilIfEmpty(ctag)),
			File:       &f,
			Detail:     map[string]any{"decision_count": count},
		})
	}

	historyCount := 0
	for _, row := range tables["history_files"] {
		historyCount++
		rawScript := text(row, "raw_script")
		if rawScript == "" {
			continue
		}
		ctag := textPtr(row, "country_tag")
		tag := deref(ctag)
		ref := perfEntityRef("history_file", &tag, perfRefNameStr(tag), ctag)
		perfScanNested(rawScript, ref, "history/countries/"+tag+".txt", "Country history raw_script", &findings)
	}

	characterCount := 0
	for _, row := range tables["characters"] {
		characterCount++
		advAvail, advVis := text(row, "advisor_available"), text(row, "advisor_visible")
		if advAvail == "" && advVis == "" {
			continue
		}
		tag, ctag := textPtr(row, "tag"), textPtr(row, "country_tag")
		base := deref(ctag)
		if base == "" {
			base = "characters"
		}
		file := "common/characters/" + base + ".txt"
		ref := perfEntityRef("character", tag, perfRefNameStr(text(row, "name_en"), text(row, "name"), deref(tag)), nilIfEmpty(ctag))
		perfScanHotField(advAvail, ref, file, "Advisor available", &findings)
		perfScanHotField(advVis, ref, file, "Advisor visible", &findings)
	}

	scanned := map[string]any{
		"focus_trees": treeCount, "focus_nodes": totalNodes,
		"events": eventCount, "decisions": decisionCount, "ideas": ideaCount,
		"characters": characterCount, "history_files": historyCount,
	}
	return LintFindingsReport(findings, map[string]any{
		"scanned": scanned, "export_profile": nil,
	}), nil
}

func perfRefNameStr(candidates ...string) *string {
	for _, c := range candidates {
		if strings.TrimSpace(c) != "" {
			v := strings.TrimSpace(c)
			return &v
		}
	}
	return nil
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
