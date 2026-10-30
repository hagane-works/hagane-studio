package exportcheck

type ValidateProjectInput struct {
	Core          ValidateInput
	States        []ValidateState
	HistoryFiles  []ValidateHistoryFile
	StageStateIDs []int64

	IdeologyRegistry []byte
	IsSubmod         bool

	DisplayNameKinds []DisplayNameKind
}

func ideologyRefsFor(
	characters []ValidateCharacter, historyFiles []ValidateHistoryFile,
) []ValidateIdeologyRef {
	refs := []ValidateIdeologyRef{}
	for _, c := range characters {
		refs = append(refs, ValidateIdeologyRef{
			Value: c.Ideology, Who: "character " + c.Tag,
			Ref: map[string]any{"type": "character", "id": c.Tag},
		})
	}
	for _, hf := range historyFiles {
		refs = append(refs, ValidateIdeologyRef{
			Value: hf.SetPoliticsRulingParty,
			Who:   hf.CountryTag + " ruling_party",
			Ref:   map[string]any{"type": "history_file", "id": hf.CountryTag},
		})
	}
	return refs
}

func ValidateProject(in ValidateProjectInput) []map[string]any {
	issues := validateProjectCore(in.Core)
	issues = append(issues, validateCapitalNotOwned(in.States, in.HistoryFiles)...)
	issues = append(issues, validateProvinceDoubleMapping(in.States)...)
	issues = append(issues, validateIdeologyChecks(in.IdeologyRegistry, in.IsSubmod,
		ideologyRefsFor(in.Core.Characters, in.HistoryFiles))...)
	issues = append(issues, validateCountryLeaderIdeologies(in.IdeologyRegistry,
		in.Core.Characters)...)
	issues = append(issues, validateHistoryPopularities(in.IdeologyRegistry,
		in.IsSubmod, in.HistoryFiles)...)
	issues = append(issues, validateTechCircularPrereq(in.Core.Techs)...)
	issues = append(issues, validateDisplayNamePlaceholder(in.DisplayNameKinds)...)
	tagOutOfStageIssues(issues, in.StageStateIDs)
	return appendSuccessLine(issues)
}

func tagOutOfStageIssues(issues []map[string]any, stageStateIDs []int64) {
	if len(stageStateIDs) == 0 {
		return
	}
	stage := map[int64]bool{}
	for _, id := range stageStateIDs {
		stage[id] = true
	}
	if len(stage) == 0 {
		return
	}
	for _, issue := range issues {
		if anyToString(issue["level"]) != "warning" {
			continue
		}
		ref, ok := issue["entity_ref"].(map[string]any)
		if !ok {
			continue
		}
		sid, ok := stateIDFromRef(ref["state_id"])
		if !ok {
			continue
		}
		if !stage[sid] {
			issue["out_of_stage"] = true
		}
	}
}

func stateIDFromRef(v any) (int64, bool) {
	switch value := v.(type) {
	case bool:
		return 0, false
	case int:
		return int64(value), true
	case int64:
		return value, true
	case float64:

		if value == float64(int64(value)) {
			return int64(value), true
		}
	}
	return 0, false
}
