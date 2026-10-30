package ideas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gameindex"
)

type vanillaEntry struct {
	raw      json.RawMessage
	hay      string
	ideaID   string
	category string
	picture  string
}

type vanillaIndex struct {
	total       int
	generatedAt any
	categories  int
	entries     []vanillaEntry
}

type vanillaIndexKey struct{}

func vanillaIdeas(ctx context.Context) (*vanillaIndex, error) {
	idx, err := gameindex.Load(ctx, vanillaIndexKey{}, gameindex.IdeaIndex, decodeVanillaIdeas)
	if err == gameindex.ErrNoGame {
		return &vanillaIndex{}, nil
	}
	return idx, err
}

func decodeVanillaIdeas(raw []byte) (*vanillaIndex, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	idx := &vanillaIndex{}
	if raw, ok := top["total"]; ok {
		_ = json.Unmarshal(raw, &idx.total)
	}
	if raw, ok := top["generated_at"]; ok {
		_ = json.Unmarshal(raw, &idx.generatedAt)
	}
	if raw, ok := top["by_category"]; ok {
		var groups map[string]json.RawMessage
		_ = json.Unmarshal(raw, &groups)
		idx.categories = len(groups)
	}
	var items []json.RawMessage
	if raw, ok := top["ideas"]; ok {
		_ = json.Unmarshal(raw, &items)
	}
	idx.entries = make([]vanillaEntry, 0, len(items))
	for _, item := range items {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		e := vanillaEntry{
			raw: item, ideaID: strField(m, "idea_id"),
			category: strField(m, "category"), picture: strField(m, "picture"),
		}
		e.hay = strings.ToLower(strings.Join([]string{e.ideaID, e.category, e.picture, strField(m, "file")}, "\n"))
		idx.entries = append(idx.entries, e)
	}
	return idx, nil
}

func strField(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func queryInt(r *http.Request, key string, def int) (int, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

func queryBool(r *http.Request, key string) (*bool, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil, true
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		t := true
		return &t, true
	case "false", "0", "no", "off":
		f := false
		return &f, true
	}
	return nil, false
}

func vanillaList(w http.ResponseWriter, r *http.Request) {
	limit, ok1 := queryInt(r, "limit", 50)
	offset, ok2 := queryInt(r, "offset", 0)
	hasPicture, ok3 := queryBool(r, "has_picture")
	if !ok1 || !ok2 || !ok3 {
		apiutil.Fail(w, apiutil.ValidationError("invalid query parameter"))
		return
	}
	if err := gameindex.CheckReady(gameindex.IdeaIndex); err != nil {
		apiutil.Fail(w, err)
		return
	}
	idx, err := vanillaIdeas(r.Context())
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	q := r.URL.Query()
	apiutil.JSON(w, http.StatusOK, listVanillaIdeas(idx, q.Get("q"), q.Get("category"), hasPicture, limit, offset))
}

func listVanillaIdeas(idx *vanillaIndex, q, category string, hasPicture *bool, limit, offset int) map[string]any {
	query := strings.ToLower(strings.TrimSpace(q))
	category = strings.TrimSpace(category)
	matched := []json.RawMessage{}
	for i := range idx.entries {
		e := &idx.entries[i]
		if category != "" && e.category != category {
			continue
		}
		if hasPicture != nil && *hasPicture == (e.picture == "") {
			continue
		}
		if query != "" && !strings.Contains(e.hay, query) {
			continue
		}
		matched = append(matched, e.raw)
	}
	limit = min(max(limit, 1), 200)
	offset = max(offset, 0)
	page := []json.RawMessage{}
	if offset < len(matched) {
		page = matched[offset:min(offset+limit, len(matched))]
	}
	return map[string]any{
		"total_matching": len(matched),
		"limit":          limit,
		"offset":         offset,
		"items":          page,
		"meta": map[string]any{
			"total":        idx.total,
			"generated_at": idx.generatedAt,
			"categories":   idx.categories,
		},
	}
}

func vanillaDetail(w http.ResponseWriter, r *http.Request) {
	category, ideaID := r.PathValue("category"), r.PathValue("idea_id")
	if err := gameindex.CheckReady(gameindex.IdeaIndex); err != nil {
		apiutil.Fail(w, err)
		return
	}
	idx, err := vanillaIdeas(r.Context())
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	for _, e := range idx.entries {
		if e.category == category && e.ideaID == ideaID {
			apiutil.JSON(w, http.StatusOK, map[string]any{"entry": e.raw})
			return
		}
	}
	apiutil.Fail(w, apiutil.NotFoundError("vanilla idea "+category+"/"+ideaID+" not found"))
}

var emptyIdeasUI = []byte(`{"meta":{"total":0,"source":"","groups":{}},"by_group":{},"by_country":{},"all_ids":[]}`)

func vanillaIdeasIndex(w http.ResponseWriter, r *http.Request) {
	raw, err := gameindex.Peek(gameindex.IdeasUI)
	if err == gameindex.ErrNoGame {
		raw, err = emptyIdeasUI, nil
	}
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
