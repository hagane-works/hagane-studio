package gameview

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/project"
)

const (
	textOverrideFile = "localisation/english/replace/zz_game_view_l_english.yml"
	textMaxLen       = 8000
)

var errTextInvalid = errors.New("invalid GUI text override")

func validTextKey(key string) bool {
	if key == "" || len(key) > 512 {
		return false
	}
	for _, ch := range key {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '_', ch == '.', ch == '-':
		default:
			return false
		}
	}
	return true
}

func textTarget(lang string) (locale, file string, ok bool) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" || lang == "en" {
		return "l_english", textOverrideFile, true
	}
	hoi4, ok := locout.LangCodeToHOI4[lang]
	if !ok {
		return "", "", false
	}
	return "l_" + hoi4, "localisation/" + hoi4 + "/replace/zz_game_view_l_" + hoi4 + ".yml", true
}

func isTextOverride(row project.Row, locale, file string) bool {
	return str(row["locale"]) == locale && str(row["source_file"]) == file && str(row["source"]) == "created"
}

func textOverrides(p *project.Project, lang string) (map[string]string, error) {
	locale, file, ok := textTarget(lang)
	if !ok {
		return nil, errTextInvalid
	}
	rows, err := p.Rows("project_loc_entries")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, row := range rows {
		if isTextOverride(row, locale, file) {
			out[str(row["key"])] = str(row["value"])
		}
	}
	return out, nil
}

func setTextOverride(p *project.Project, lang, key, value string) error {
	key = strings.TrimSpace(key)
	locale, file, ok := textTarget(lang)
	if !ok || !validTextKey(key) || utf8.RuneCountInString(value) > textMaxLen || strings.ContainsRune(value, '\x00') {
		return errTextInvalid
	}
	return p.Write([]string{"project_loc_entries"}, func(tx *project.Tx) error {
		for _, row := range tx.Rows("project_loc_entries") {
			if str(row["locale"]) == locale && str(row["key"]) == key && str(row["source_file"]) == file &&
				project.Int64(row["entry_order"]) == 0 {
				_, err := tx.Update("project_loc_entries", project.Int64(row["id"]), func(r project.Row) {
					r["value"], r["source"] = value, "created"
				})
				return err
			}
		}
		tx.Insert("project_loc_entries", project.Row{
			"locale": locale, "key": key, "value": value, "version": nil, "source": "created",
			"source_file": file, "replace_priority": int64(1), "source_order": int64(0), "entry_order": int64(0),
			"is_replace": true,
		})
		return nil
	})
}

func clearTextOverride(p *project.Project, lang, key string) (bool, error) {
	locale, file, ok := textTarget(lang)
	if !ok {
		return false, errTextInvalid
	}
	key = strings.TrimSpace(key)
	removed := 0
	err := p.Write([]string{"project_loc_entries"}, func(tx *project.Tx) error {
		removed = tx.Delete("project_loc_entries", func(row project.Row) bool {
			return str(row["key"]) == key && isTextOverride(row, locale, file)
		})
		return nil
	})
	return removed > 0, err
}

func (s *Service) text(w http.ResponseWriter, r *http.Request, p *project.Project) {

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if _, _, ok := textTarget(lang); !ok {
		detail(w, http.StatusUnprocessableEntity, "unknown language: "+lang)
		return
	}
	switch r.Method {
	case http.MethodGet:
		overrides, err := textOverrides(p, lang)
		if err != nil {
			detail(w, http.StatusInternalServerError, "could not read text overrides")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"overrides": overrides})
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var in struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			Lang  string `json:"lang"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			detail(w, http.StatusUnprocessableEntity, "invalid JSON body")
			return
		}
		in.Key = strings.TrimSpace(in.Key)
		if strings.TrimSpace(in.Lang) != "" {
			lang = in.Lang
		}

		if file, owned := s.textOwnedByData(r, p, lang, in.Key); owned {
			writeJSON(w, http.StatusConflict, map[string]string{
				"detail": "this text is written by the project's own data; change it there",
				"code":   "owned_by_project_data", "file": file,
			})
			return
		}
		if err := setTextOverride(p, lang, in.Key, in.Value); err != nil {
			if errors.Is(err, errTextInvalid) {
				detail(w, http.StatusUnprocessableEntity, "the key must be a localisation key (A-Z a-z 0-9 _ . -) and the text at most 8000 characters")
				return
			}
			detail(w, http.StatusInternalServerError, "could not save the text")
			return
		}
		s.Forget(p)
		writeJSON(w, http.StatusOK, map[string]any{"key": in.Key, "value": in.Value})
	case http.MethodDelete:
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		if !validTextKey(key) {
			detail(w, http.StatusUnprocessableEntity, "key is required")
			return
		}
		removed, err := clearTextOverride(p, lang, key)
		if err != nil {
			detail(w, http.StatusInternalServerError, "could not restore the text")
			return
		}
		s.Forget(p)
		writeJSON(w, http.StatusOK, map[string]any{"key": key, "removed": removed})
	default:
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (s *Service) textOwnedByData(r *http.Request, p *project.Project, lang, key string) (string, bool) {
	if !validTextKey(key) {
		return "", false
	}
	header, _, ok := textTarget(lang)
	if !ok {
		return "", false
	}
	index, _, err := s.loadIndex(r.Context(), p)
	if err != nil || index == nil {
		return "", false
	}
	return index.ProjectDataLocFile(key, header)
}
