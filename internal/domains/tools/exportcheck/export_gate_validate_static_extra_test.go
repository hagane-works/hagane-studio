package exportcheck

func tagOutOfStage(issues []map[string]any, stageStateIDs []int64) {
	if len(stageStateIDs) == 0 {
		return
	}
	stage := map[int64]bool{}
	for _, s := range stageStateIDs {
		stage[s] = true
	}
	if len(stage) == 0 {
		return
	}
	for _, issue := range issues {
		if issue["level"] != "warning" {
			continue
		}
		ref, ok := issue["entity_ref"].(map[string]any)
		if !ok {
			continue
		}
		sid, ok := toInt64(ref["state_id"])
		if !ok {
			continue
		}
		if !stage[sid] {
			issue["out_of_stage"] = true
		}
	}
}

func toInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:

		return int64(t), true
	}
	return 0, false
}
