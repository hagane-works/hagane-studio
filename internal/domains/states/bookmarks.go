package states

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/savediagnostics"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const bookmarksTable = "bookmarks"

var errBookmarkMissing = apiutil.NotFoundError("Bookmark not found")

type bookmark struct {
	EditRevision        string                        `json:"edit_revision"`
	Diagnostics         *[]savediagnostics.Diagnostic `json:"diagnostics,omitempty"`
	ID                  int64                         `json:"id"`
	ProjectID           int64                         `json:"project_id"`
	BookmarkID          string                        `json:"bookmark_id"`
	NameEn              string                        `json:"name_en"`
	NameZh              string                        `json:"name_zh"`
	DescEn              string                        `json:"desc_en"`
	DescZh              string                        `json:"desc_zh"`
	Date                string                        `json:"date"`
	Picture             string                        `json:"picture"`
	DefaultCountry      string                        `json:"default_country"`
	DefaultFlag         bool                          `json:"default_flag"`
	SelectableCountries []map[string]any              `json:"selectable_countries"`
	Filters             []string                      `json:"filters"`
	DisplayNameLocs     map[string]string             `json:"display_name_locs"`
	DescriptionLocs     map[string]string             `json:"description_locs"`
	Source              string                        `json:"source"`
	CreatedAt           string                        `json:"created_at"`
	UpdatedAt           string                        `json:"updated_at"`
}

func registerBookmarks(mux *http.ServeMux, h *handlers) {
	base := "/api/go/projects/{id}/bookmarks"
	mux.HandleFunc("GET "+base, h.handle(h.listBookmarks))
	mux.HandleFunc("POST "+base, h.handle(h.createBookmark))
	mux.HandleFunc("GET "+base+"/{bid}", h.handle(h.getBookmark))
	mux.HandleFunc("PUT "+base+"/{bid}", h.handle(h.updateBookmark))
	mux.HandleFunc("DELETE "+base+"/{bid}", h.handle(h.deleteBookmark))
	mux.HandleFunc("POST "+base+"/{bid}/upload-picture", h.handle(h.uploadBookmarkPicture))
	mux.HandleFunc("GET "+base+"/{bid}/picture", h.handle(h.bookmarkPicture))
}

func textOr(v any, def string) string {
	if v == nil {
		return def
	}
	return str(v)
}

func stringMap(v any) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]any); ok {
		for k, value := range m {
			if s, ok := value.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func bookmarkFromRow(row project.Row) bookmark {
	b := bookmark{
		EditRevision: strconv.FormatInt(revisionOfRow(row), 10),
		ID:           project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		BookmarkID: str(row["bookmark_id"]),
		NameEn:     str(row["name_en"]), NameZh: str(row["name_zh"]),
		DescEn: str(row["desc_en"]), DescZh: str(row["desc_zh"]),
		Date:           textOr(row["date"], "1936.1.1.12"),
		Picture:        textOr(row["picture"], "GFX_select_date_1936"),
		DefaultCountry: str(row["default_country"]),
		DefaultFlag:    row["default_flag"] == true,
		Filters:        []string{},
		Source:         textOr(row["source"], "created"),
		CreatedAt:      str(row["created_at"]), UpdatedAt: str(row["updated_at"]),
	}
	b.SelectableCountries = []map[string]any{}
	if list, ok := row["selectable_countries"].([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				b.SelectableCountries = append(b.SelectableCountries, m)
			}
		}
	}
	b.SelectableCountries = presentBookmarkCountries(b.SelectableCountries)
	if list, ok := row["filters"].([]any); ok {
		for _, item := range list {
			if s, ok := item.(string); ok {
				b.Filters = append(b.Filters, s)
			}
		}
	}
	b.DisplayNameLocs = stringMap(row["display_name_locs"])
	b.DescriptionLocs = stringMap(row["description_locs"])
	return b
}

func normalizeBookmarkCountries(entries []map[string]any) []map[string]any {
	if entries == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(entries))
	for _, raw := range entries {
		if raw == nil {
			out = append(out, raw)
			continue
		}
		entry := map[string]any{}
		for k, v := range raw {
			entry[k] = v
		}
		locs := map[string]string{}
		if given, ok := entry["history_locs"].(map[string]any); ok {
			for lang, value := range given {
				if text, ok := value.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						locs[lang] = trimmed
					}
				}
			}
		}
		for _, pair := range [][2]string{{"en", "history_en"}, {"zh", "history_zh"}} {
			if text := strings.TrimSpace(str(entry[pair[1]])); text != "" && locs[pair[0]] == "" {
				locs[pair[0]] = text
			}
		}
		if len(locs) > 0 {
			entry["history_locs"] = locs
			if _, present := entry["history_en"]; !present {
				entry["history_en"] = locs["en"]
			}
			if _, present := entry["history_zh"]; !present {
				entry["history_zh"] = locs["zh"]
			}
		}
		out = append(out, entry)
	}
	return out
}

func presentBookmarkCountries(entries []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, raw := range entries {
		entry := make(map[string]any, len(raw)+6)
		for k, v := range raw {
			entry[k] = v
		}
		defaults := map[string]any{"ideology": "", "ideas": []any{}, "focuses": []any{}, "minor": false, "label": "", "available": nil}
		for k, v := range defaults {
			if _, ok := entry[k]; !ok {
				entry[k] = v
			}
		}
		out = append(out, entry)
	}
	return out
}

func sortedBookmarks(rows []project.Row) []bookmark {
	out := make([]bookmark, 0, len(rows))
	for _, row := range rows {
		out = append(out, bookmarkFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].BookmarkID < out[j].BookmarkID })
	return out
}

func bookmarksByID(rows []project.Row) []bookmark {
	out := make([]bookmark, 0, len(rows))
	for _, row := range rows {
		out = append(out, bookmarkFromRow(row))
	}
	return out
}

func listBookmarksWithDiagnostics(p *project.Project) ([]bookmark, error) {
	rows, err := p.Rows(bookmarksTable)
	if err != nil {
		return nil, err
	}
	out := sortedBookmarks(rows)
	all := bookmarkSaveDiagnostics(p.ID(), out)
	for i := range out {
		out[i].Diagnostics = bookmarkRowDiagnostics(all, out[i])
	}
	return out, nil
}

func (h *handlers) listBookmarks(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	items, err := listBookmarksWithDiagnostics(p)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func bookmarkID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("bid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errBookmarkMissing
	}
	return id, nil
}

func (h *handlers) getBookmark(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := bookmarkID(r)
	if err != nil {
		return err
	}
	items, err := listBookmarksWithDiagnostics(p)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == id {
			apiutil.JSON(w, http.StatusOK, item)
			return nil
		}
	}
	return errBookmarkMissing
}

type bookmarkCreateInput struct {
	BookmarkID          string            `json:"bookmark_id"`
	NameEn              string            `json:"name_en"`
	NameZh              string            `json:"name_zh"`
	DescEn              string            `json:"desc_en"`
	DescZh              string            `json:"desc_zh"`
	Date                *string           `json:"date"`
	Picture             *string           `json:"picture"`
	DefaultCountry      string            `json:"default_country"`
	DefaultFlag         bool              `json:"default_flag"`
	SelectableCountries []map[string]any  `json:"selectable_countries"`
	Filters             []string          `json:"filters"`
	DisplayNameLocs     map[string]string `json:"display_name_locs"`
	DescriptionLocs     map[string]string `json:"description_locs"`
	Source              *string           `json:"source"`
}

type bookmarkPatch struct {
	BookmarkID          workbench.Optional[string]            `json:"bookmark_id"`
	NameEn              workbench.Optional[string]            `json:"name_en"`
	NameZh              workbench.Optional[string]            `json:"name_zh"`
	DescEn              workbench.Optional[string]            `json:"desc_en"`
	DescZh              workbench.Optional[string]            `json:"desc_zh"`
	Date                workbench.Optional[string]            `json:"date"`
	Picture             workbench.Optional[string]            `json:"picture"`
	DefaultCountry      workbench.Optional[string]            `json:"default_country"`
	DefaultFlag         workbench.Optional[bool]              `json:"default_flag"`
	SelectableCountries workbench.Optional[[]map[string]any]  `json:"selectable_countries"`
	Filters             workbench.Optional[[]string]          `json:"filters"`
	DisplayNameLocs     workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs     workbench.Optional[map[string]string] `json:"description_locs"`
}

func decodeBookmarkBody(r *http.Request, v any) error {
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20)).Decode(v); err != nil {
		return detailError(http.StatusUnprocessableEntity, "Invalid bookmark payload")
	}
	return nil
}

func bookmarkIDTaken(tx *project.Tx, bookmarkID string, except int64) bool {
	for _, row := range tx.Rows(bookmarksTable) {
		if project.Int64(row["id"]) != except && str(row["bookmark_id"]) == bookmarkID {
			return true
		}
	}
	return false
}

func withDiagnostics(projectID int64, before []savediagnostics.Diagnostic, after []bookmark, b bookmark) bookmark {
	diagnostics := savediagnostics.Compare(before, bookmarkSaveDiagnostics(projectID, after))
	b.Diagnostics = bookmarkRowDiagnostics(diagnostics, b)
	return b
}

func (h *handlers) createBookmark(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	var in bookmarkCreateInput
	if err := decodeBookmarkBody(r, &in); err != nil {
		return err
	}
	id := strings.TrimSpace(in.BookmarkID)
	if id == "" {
		return detailError(http.StatusBadRequest, "bookmark_id is required (e.g. 'the_modern_usa_arrived')")
	}
	if in.SelectableCountries == nil {
		in.SelectableCountries = []map[string]any{}
	}
	in.SelectableCountries = normalizeBookmarkCountries(in.SelectableCountries)
	if in.Filters == nil {
		in.Filters = []string{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	var result bookmark
	err = p.Write([]string{bookmarksTable}, func(tx *project.Tx) error {
		before := bookmarkSaveDiagnostics(p.ID(), bookmarksByID(tx.Rows(bookmarksTable)))
		if bookmarkIDTaken(tx, id, 0) {
			return detailError(http.StatusConflict, fmt.Sprintf(
				"bookmark_id '%s' already exists in this project. Pick a unique bookmark_id.", id))
		}
		row := tx.Insert(bookmarksTable, project.Row{
			"edit_revision": int64(1), "bookmark_id": id,
			"name_en": in.NameEn, "name_zh": in.NameZh, "desc_en": in.DescEn, "desc_zh": in.DescZh,
			"date": workbench.StrDefault(in.Date, "1936.1.1.12"), "picture": workbench.StrDefault(in.Picture, "GFX_select_date_1936"),
			"default_country": in.DefaultCountry, "default_flag": in.DefaultFlag,
			"selectable_countries": plain(in.SelectableCountries), "filters": plain(in.Filters),
			"display_name_locs": plain(in.DisplayNameLocs), "description_locs": plain(in.DescriptionLocs),
			"source": workbench.StrDefault(in.Source, "created"), "raw_extras": map[string]any{},
		})
		rv.created(project.Int64(row["id"]))
		result = withDiagnostics(p.ID(), before, bookmarksByID(tx.Rows(bookmarksTable)), bookmarkFromRow(row))
		return nil
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	rv.header(w)
	apiutil.JSON(w, http.StatusCreated, result)
	return nil
}

func optionalJSON[T any](o workbench.Optional[T], null any) any {
	if o.Value == nil {
		return null
	}
	return plain(*o.Value)
}

func (h *handlers) updateBookmark(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := bookmarkID(r)
	if err != nil {
		return err
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	var patch bookmarkPatch
	if err := decodeBookmarkBody(r, &patch); err != nil {
		return err
	}
	if patch.BookmarkID.Set {
		trimmed := ""
		if patch.BookmarkID.Value != nil {
			trimmed = strings.TrimSpace(*patch.BookmarkID.Value)
		}
		if trimmed == "" {
			return detailError(http.StatusBadRequest, "bookmark_id cannot be empty")
		}
		patch.BookmarkID.Value = &trimmed
	}
	if patch.SelectableCountries.Set && patch.SelectableCountries.Value != nil {
		normalized := normalizeBookmarkCountries(*patch.SelectableCountries.Value)
		patch.SelectableCountries.Value = &normalized
	}
	changes := map[string]any{}
	setText := func(column string, o workbench.Optional[string]) {
		if o.Set {
			if o.Value == nil {
				changes[column] = nil
			} else {
				changes[column] = *o.Value
			}
		}
	}
	setText("bookmark_id", patch.BookmarkID)
	setText("name_en", patch.NameEn)
	setText("name_zh", patch.NameZh)
	setText("desc_en", patch.DescEn)
	setText("desc_zh", patch.DescZh)
	setText("date", patch.Date)
	setText("picture", patch.Picture)
	setText("default_country", patch.DefaultCountry)
	if patch.DefaultFlag.Set {
		changes["default_flag"] = patch.DefaultFlag.Value != nil && *patch.DefaultFlag.Value
	}
	if patch.SelectableCountries.Set {
		changes["selectable_countries"] = optionalJSON(patch.SelectableCountries, nil)
	}
	if patch.Filters.Set {
		changes["filters"] = optionalJSON(patch.Filters, nil)
	}
	if patch.DisplayNameLocs.Set {
		changes["display_name_locs"] = optionalJSON(patch.DisplayNameLocs, map[string]any{})
	}
	if patch.DescriptionLocs.Set {
		changes["description_locs"] = optionalJSON(patch.DescriptionLocs, map[string]any{})
	}
	var result bookmark
	err = p.Write([]string{bookmarksTable}, func(tx *project.Tx) error {
		if _, err := rv.advance(tx, id); err != nil {
			return err
		}
		cur := tx.Find(bookmarksTable, id)
		before := bookmarkSaveDiagnostics(p.ID(), bookmarksByID(tx.Rows(bookmarksTable)))
		if newID, ok := changes["bookmark_id"].(string); ok && bookmarkIDTaken(tx, newID, id) {
			return detailError(http.StatusConflict, "bookmark_id conflict in this project")
		}
		flip := textOr(cur["source"], "created") == "imported"
		row, err := tx.Update(bookmarksTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
			if flip {
				row["source"] = "created"
			}
		})
		if err != nil {
			return err
		}
		result = withDiagnostics(p.ID(), before, bookmarksByID(tx.Rows(bookmarksTable)), bookmarkFromRow(row))
		return nil
	})
	if err != nil {
		return err
	}
	rv.commit(p)
	rv.header(w)
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func (h *handlers) deleteBookmark(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := bookmarkID(r)
	if err != nil {
		return err
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	err = p.Write([]string{bookmarksTable}, func(tx *project.Tx) error {
		if _, err := rv.advance(tx, id); err != nil {
			return err
		}
		if tx.Delete(bookmarksTable, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errBookmarkMissing
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

const codeSaveConflict = "save_conflict"

type revisionScope struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

type revisionProblem struct {
	Code             string        `json:"code"`
	Scope            revisionScope `json:"scope"`
	ExpectedRevision string        `json:"expected_revision,omitempty"`
	CurrentRevision  string        `json:"current_revision,omitempty"`
}

func problemError(status int, problem revisionProblem) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": problem}}
}

func parseRevision(raw string) (int64, bool) {
	if raw == "" || len(raw) > 19 || raw[0] < '1' || raw[0] > '9' {
		return 0, false
	}
	for i := 1; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return 0, false
		}
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	return value, err == nil
}

func revisionOfRow(row project.Row) int64 {
	if n := project.Int64(row["edit_revision"]); n > 0 {
		return n
	}
	return 1
}

type revisions struct {
	expected  map[string]string
	advanced  map[int64]string
	committed map[string]string
}

func invalidRevisions() error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity,
		Body: map[string]any{"detail": map[string]string{"code": "invalid_edit_revisions"}}}
}

func readRevisions(r *http.Request) (*revisions, error) {
	rv := &revisions{expected: map[string]string{}, advanced: map[int64]string{}, committed: map[string]string{}}
	raw := r.Header.Get("X-Edit-Revisions")
	if len(raw) > 32768 {
		return nil, invalidRevisions()
	}
	if raw == "" {
		return rv, nil
	}
	var expected map[string]string
	if err := json.Unmarshal([]byte(raw), &expected); err != nil || expected == nil {
		return nil, invalidRevisions()
	}
	for key, value := range expected {
		kind, rawID, found := strings.Cut(key, ":")
		if _, ok := parseRevision(rawID); !found || (kind != "bookmark" && kind != "focus_tree") || !ok {
			return nil, invalidRevisions()
		}
		if _, ok := parseRevision(value); !ok {
			return nil, invalidRevisions()
		}
	}
	rv.expected = expected
	return rv, nil
}

func bookmarkScopeKey(id int64) string { return "bookmark:" + strconv.FormatInt(id, 10) }

func (rv *revisions) advance(tx *project.Tx, id int64) (string, error) {
	if revision, done := rv.advanced[id]; done {
		return revision, nil
	}
	scope := revisionScope{Kind: "bookmark", ID: id}
	row := tx.Find(bookmarksTable, id)
	if row == nil {
		return "", problemError(http.StatusNotFound, revisionProblem{Code: "save_target_not_found", Scope: scope})
	}
	current := revisionOfRow(row)
	currentText := strconv.FormatInt(current, 10)
	if expected, ok := rv.expected[bookmarkScopeKey(id)]; ok && expected != currentText {
		return "", problemError(http.StatusConflict, revisionProblem{
			Code: codeSaveConflict, Scope: scope, ExpectedRevision: expected, CurrentRevision: currentText,
		})
	}
	if current == math.MaxInt64 {
		return "", problemError(http.StatusInternalServerError, revisionProblem{Code: "save_revision_exhausted", Scope: scope})
	}
	next := current + 1
	if _, err := tx.Update(bookmarksTable, id, func(r project.Row) { r["edit_revision"] = next }); err != nil {
		return "", err
	}
	rv.advanced[id] = strconv.FormatInt(next, 10)
	return rv.advanced[id], nil
}

func (rv *revisions) created(id int64) { rv.advanced[id] = "1" }

func (rv *revisions) commit(p *project.Project) {
	if len(rv.advanced) == 0 {
		return
	}
	rows, err := p.Rows(bookmarksTable)
	if err != nil {
		return
	}
	for _, row := range rows {
		id := project.Int64(row["id"])
		if _, touched := rv.advanced[id]; touched {
			rv.committed[bookmarkScopeKey(id)] = strconv.FormatInt(revisionOfRow(row), 10)
		}
	}
}

func (rv *revisions) header(w http.ResponseWriter) {
	if rv == nil || len(rv.committed) == 0 {
		return
	}
	payload, _ := json.Marshal(rv.committed)
	w.Header().Set("X-Edit-Revisions", string(payload))
}
