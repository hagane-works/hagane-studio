package importjob

func legacyResult(sessionID, policy string, needed []string, gfxCount int64) map[string]any {
	return map[string]any{
		"characters":   map[string]any{"found_files": 0, "created": 0, "updated": 0, "skipped": 0, "warnings": []string{}},
		"focus_trees":  map[string]any{"found_files": 0, "created_trees": 0, "updated_trees": 0, "skipped_trees": 0, "created_nodes": 0, "warnings": []string{}},
		"technologies": map[string]any{"found_files": 0, "created": 0, "updated": 0, "skipped": 0, "warnings": []string{}},
		"ideas":        map[string]any{"found_files": 0, "created": 0, "updated": 0, "skipped": 0, "warnings": []string{}},
		"events":       map[string]any{"found_files": 0, "created_namespaces": 0, "reused_namespaces": 0, "created_events": 0, "updated_events": 0, "skipped_events": 0, "warnings": []string{}},
		"decisions":    map[string]any{"found_files": 0, "found_categories": 0, "created_categories": 0, "updated_categories": 0, "created_decisions": 0, "updated_decisions": 0, "skipped_decisions": 0, "warnings": []string{}},
		"warnings":     []string{}, "needed_images": needed, "gfx_mappings_count": gfxCount,
		"session_id": sessionID, "conflict_policy": policy,
	}
}

func applyResultCounts(result map[string]any, entities map[string]int64, trees TreeEntityCounts, world EventsWorldCounts) {
	result["characters"].(map[string]any)["updated"] = entities["character"]
	result["ideas"].(map[string]any)["updated"] = entities["idea"]
	f := result["focus_trees"].(map[string]any)
	f["updated_trees"], f["created_nodes"] = trees.FocusTrees, trees.FocusNodes
	result["technologies"].(map[string]any)["updated"] = trees.Technologies
	e := result["events"].(map[string]any)
	e["created_namespaces"], e["updated_events"] = world["event_namespace"], world["event"]
	d := result["decisions"].(map[string]any)
	d["found_categories"], d["updated_categories"], d["updated_decisions"] = world["decision_category"], world["decision_category"], world["decision"]
}
