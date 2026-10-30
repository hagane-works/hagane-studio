package gamesave

import (
	"encoding/json"
	"net/http"
	"strings"

	"haganeworks/internal/domains/military"
	"haganeworks/internal/hoi4/importsvc/ideas"
	"haganeworks/internal/project"
)

const mioTable = "military_industrial_organizations"

const mioSourceKey = "__source_file"

type mioTakeIndex interface {
	IndustrialOrgSource(name string) (string, []byte, bool)
}

func takeMIOFileIn(st *store, index mioTakeIndex, id string) (string, int, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", 0, saveError{422, "id is required"}
	}
	rows := st.sortedRows(mioTable, nil, byText("mio_id"))
	have := map[string]bool{}
	for _, row := range rows {
		have[text(row, "mio_id")] = true
	}
	if have[id] {
		return mioRowFile(rows, id), 0, nil
	}
	file, body, ok := index.IndustrialOrgSource(id)
	if !ok {
		return "", 0, saveError{422, "no file declares this organization"}
	}
	for _, row := range rows {
		extras, _ := row["raw_extras"].(map[string]any)
		if source, _ := extras[mioSourceKey].(string); strings.EqualFold(source, file) {
			return file, 0, nil
		}
	}
	parsed, _, err := ideas.ImportMIO(string(body))
	if err != nil {
		return "", 0, saveError{422, "the organization file could not be read: " + err.Error()}
	}
	inputs := []military.MIOCreateInput{}
	for _, entity := range ideas.NativeEntities(parsed) {
		in, err := mioInputFromImport(entity, file)
		if err != nil {
			return "", 0, err
		}
		if in.MioID == "" || have[in.MioID] {
			continue
		}
		inputs = append(inputs, in)
	}

	for _, in := range inputs {
		if _, err := military.InsertMIO(st.tx, in); err != nil {
			return "", 0, err
		}
	}
	return file, len(inputs), nil
}

func mioInputFromImport(entity map[string]any, file string) (military.MIOCreateInput, error) {
	encoded, err := json.Marshal(entity)
	if err != nil {
		return military.MIOCreateInput{}, err
	}
	var in military.MIOCreateInput
	if err := json.Unmarshal(encoded, &in); err != nil {
		return military.MIOCreateInput{}, err
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	in.RawExtras[mioSourceKey] = file

	if verdict, ok := entity["_scope_verdict"].(map[string]any); ok {
		in.CountryTag, _ = verdict["country_tag"].(string)
		if kind, _ := verdict["kind"].(string); kind != "" {
			in.ScopeKind = &kind
		}
		if shared, ok := verdict["shared_country_tags"].([]any); ok {
			for _, tag := range shared {
				if text, ok := tag.(string); ok {
					in.SharedCountryTags = append(in.SharedCountryTags, text)
				}
			}
		}
	}
	created := "created"
	in.Source = &created
	return in, nil
}

func mioRowFile(rows []project.Row, id string) string {
	for _, row := range rows {
		if text(row, "mio_id") == id {
			extras, _ := row["raw_extras"].(map[string]any)
			file, _ := extras[mioSourceKey].(string)
			return file
		}
	}
	return ""
}

func (h *handlers) mioTake(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(w, r, 1<<16, &in); err != nil || strings.TrimSpace(in.ID) == "" {
		detail(w, http.StatusUnprocessableEntity, "id is required")
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	type taken struct {
		file  string
		count int
	}
	out, err := write(p, []string{mioTable}, func(st *store) (taken, error) {
		file, count, err := takeMIOFileIn(st, index, in.ID)
		return taken{file, count}, err
	})
	h.finish(w, p, map[string]any{"file": out.file, "imported": out.count}, err, "taking the organization file in", out.count > 0)
}
