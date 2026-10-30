package ideas

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importsvc"
	ideaimport "haganeworks/internal/hoi4/importsvc/ideas"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/project"
)

const maxSingleImportBytes = 256 << 20

type importResult struct {
	Total           int            `json:"total"`
	Created         int            `json:"created"`
	Updated         int            `json:"updated"`
	Skipped         int            `json:"skipped"`
	PreservedFields map[string]int `json:"preserved_fields"`
	Warnings        []string       `json:"warnings"`
	ConflictPolicy  string         `json:"conflict_policy"`
}

func importIdeas(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	text, err := readImportUpload(w, r)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		return err
	}
	result, err := importIdeaScript(p, text, r.URL.Query().Get("country_tag"), r.FormValue("conflict_policy"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func readImportUpload(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSingleImportBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return "", apiutil.ValidationError("invalid multipart form")
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return "", apiutil.ValidationError("file is required")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxSingleImportBytes+1))
	if err != nil {
		return "", apiutil.ValidationError("failed to read upload")
	}
	if len(raw) > maxSingleImportBytes {
		return "", status(http.StatusRequestEntityTooLarge, "import file too large")
	}
	text, _, ok := loc.DecodeScriptText(raw)
	if !ok {
		return "", apiutil.ValidationError("file must be UTF-8 or Windows-1252 text")
	}
	return text, nil
}

func importPolicy(raw string) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(raw))
	if policy == "" {
		return "overwrite", nil
	}
	if policy != "overwrite" && policy != "skip" {
		return "", apiutil.BadRequestError(fmt.Sprintf("conflict_policy must be one of {'overwrite', 'skip'}, got %q", raw))
	}
	return policy, nil
}

type importPlan struct {
	Selected []int
	Created  int
	Updated  int
	Skipped  int
}

func planImport(keys []string, existing map[string]bool, policy string) importPlan {
	var plan importPlan
	seen := make(map[string]bool, len(existing)+len(keys))
	for key, present := range existing {
		seen[key] = present
	}
	selected := map[string]int{}
	for index, key := range keys {
		if key == "" {
			continue
		}
		if seen[key] {
			if policy == "skip" {
				plan.Skipped++
				continue
			}
			plan.Updated++
		} else {
			plan.Created++
			seen[key] = true
		}
		selected[key] = index
	}
	for _, index := range selected {
		plan.Selected = append(plan.Selected, index)
	}
	sort.Ints(plan.Selected)
	return plan
}

func summaryFields(summary *importsvc.ImportSummary) map[string]int {
	out := map[string]int{}
	if summary == nil || summary.PreservedFields == nil {
		return out
	}
	for _, key := range summary.PreservedFields.Keys {
		out[key] = summary.PreservedFields.Get(key)
	}
	return out
}

func importIdeaScript(p *project.Project, script, countryTag, rawPolicy string) (importResult, error) {
	policy, err := importPolicy(rawPolicy)
	if err != nil {
		return importResult{}, err
	}
	rows, summary, err := ideaimport.ImportIdeas(script)
	if err != nil {
		return importResult{}, apiutil.ValidationError(err.Error())
	}
	result := importResult{
		Total: summary.Total, PreservedFields: summaryFields(summary),
		Warnings: append([]string(nil), summary.Warnings...), ConflictPolicy: policy,
	}
	if len(rows) == 0 && len(result.Warnings) == 0 {
		result.Warnings = []string{"No ideas found in file"}
	}
	scopeTag := countryTag
	if scopeTag == "" {
		scopeTag, _ = p.Record()["country_tag"].(string)
	}
	parsed := make([]map[string]any, len(rows))
	keys := make([]string, len(rows))
	for i := range rows {
		parsed[i] = ideaimport.Native(rows[i]).(map[string]any)
		keys[i], _ = parsed[i]["idea_id"].(string)
	}

	err = p.Write([]string{table}, func(tx *project.Tx) error {
		existing := map[string]bool{}
		for _, row := range tx.Rows(table) {
			existing[text(row, "idea_id")] = true
		}
		plan := planImport(keys, existing, policy)
		result.Created, result.Updated, result.Skipped = plan.Created, plan.Updated, plan.Skipped
		if policy == "overwrite" && len(plan.Selected) > 0 {
			replaced := map[string]bool{}
			for _, index := range plan.Selected {
				replaced[keys[index]] = true
			}
			tx.Delete(table, func(row project.Row) bool { return replaced[text(row, "idea_id")] })
			for key := range replaced {
				delete(existing, key)
			}
		}
		for _, index := range plan.Selected {
			row, err := importedIdeaRow(parsed[index], scopeTag)
			if err != nil {
				return err
			}
			ideaID, _ := row["idea_id"].(string)
			if existing[ideaID] {
				continue
			}
			existing[ideaID] = true
			tx.Insert(table, row)
		}
		return nil
	})
	if err != nil {
		return importResult{}, err
	}
	return result, nil
}

var errImportUnavailable = status(http.StatusServiceUnavailable, "idea import unavailable")

func importedIdeaRow(parsed map[string]any, scopeTag string) (project.Row, error) {
	ideaID, ok := jsonText(parsed, "idea_id")
	if !ok {
		return nil, errImportUnavailable
	}
	ideaType, ok := jsonText(parsed, "idea_type")
	if !ok {
		return nil, errImportUnavailable
	}
	name, ok := jsonText(parsed, "name")
	if !ok {
		name = ideaID
	}
	picture, _ := jsonText(parsed, "picture")
	aiWillDo, _ := jsonText(parsed, "ai_will_do")
	allowed, _ := jsonText(parsed, "allowed")
	allowedCivilWar, _ := jsonText(parsed, "allowed_civil_war")
	removalCost := int64(-1)
	if raw, ok := jsonText(parsed, "removal_cost"); ok {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < math.MinInt32 || n > math.MaxInt32 {
			return nil, errImportUnavailable
		}
		removalCost = n
	}
	isDesigner := false
	if raw, ok := jsonText(parsed, "is_designer"); ok {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, errImportUnavailable
		}
		isDesigner = b
	}
	jsonOr := func(key string, fallback any) any {
		if value, ok := parsed[key]; ok {
			return jsonValue(value)
		}
		return fallback
	}
	return project.Row{
		"country_tag": scopeTag, "scope_kind": "own", "shared_country_tags": nil,
		"idea_id": ideaID, "idea_type": ideaType, "name": name, "picture": picture,
		"modifier": jsonOr("modifier", map[string]any{}), "ai_will_do": aiWillDo,
		"removal_cost": removalCost, "auto_attach_on_start": true,
		"allowed": allowed, "allowed_civil_war": allowedCivilWar, "is_designer": isDesigner,
		"research_bonus":  jsonOr("research_bonus", map[string]any{}),
		"designer_traits": jsonOr("designer_traits", []any{}),
		"available":       "", "visible": "", "allowed_to_remove": "", "cancel": "",
		"do_effect": "", "on_add": "", "on_remove": "", "cancel_if_invalid": true,
		"targeted_modifier": []any{}, "equipment_bonus": map[string]any{}, "rule": map[string]any{},
		"level": int64(0), "ledger": "", "cost": int64(0),
		"raw_extras": jsonOr("raw_extras", map[string]any{}), "category_flags": nil,
		"source": "created", "source_snapshot": jsonValue(ideaSnapshot(parsed, scopeTag)), "source_file": "",
		"name_en": name, "name_zh": "", "desc_en": "", "desc_zh": "",
		"display_name_locs": map[string]any{}, "description_locs": map[string]any{}, "has_obs": false,
	}, nil
}

func jsonText(m map[string]any, key string) (string, bool) {
	value, ok := m[key]
	if !ok || value == nil {
		return "", false
	}
	if s, ok := value.(string); ok {
		return s, true
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func ideaSnapshot(parsed map[string]any, countryTag string) map[string]any {
	stringValue := func(key, fallback string) string {
		if value, ok := parsed[key].(string); ok {
			return value
		}
		return fallback
	}
	boolValue := func(key string, fallback bool) bool {
		if value, ok := parsed[key].(bool); ok {
			return value
		}
		return fallback
	}
	valueOr := func(key string, fallback any) any {
		if value, ok := parsed[key]; ok && value != nil {
			return value
		}
		return fallback
	}
	ideaID := stringValue("idea_id", "")
	name := stringValue("name", ideaID)
	return map[string]any{
		"allowed": stringValue("allowed", ""), "allowed_civil_war": stringValue("allowed_civil_war", ""),
		"allowed_to_remove": "", "available": "", "cancel": "", "cancel_if_invalid": true,
		"category_flags": nil, "country_tag": countryTag, "description_locs": map[string]any{},
		"desc_en": "", "desc_zh": "", "designer_traits": valueOr("designer_traits", []any{}),
		"display_name_locs": map[string]any{}, "do_effect": "", "equipment_bonus": map[string]any{},
		"idea_id": ideaID, "idea_type": stringValue("idea_type", "country"),
		"is_designer": boolValue("is_designer", false), "ledger": "", "level": 0,
		"modifier": valueOr("modifier", map[string]any{}), "name": name, "name_en": name,
		"name_zh": "", "on_add": "", "on_remove": "", "picture": stringValue("picture", ""),
		"raw_extras":   valueOr("raw_extras", map[string]any{}),
		"removal_cost": valueOr("removal_cost", int64(-1)), "research_bonus": valueOr("research_bonus", map[string]any{}),
		"rule": map[string]any{}, "scope_kind": "own", "shared_country_tags": nil,
		"targeted_modifier": []any{}, "visible": "",
	}
}
