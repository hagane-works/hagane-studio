package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/gameindex"
)

func registerVanilla(mux *http.ServeMux, deps domains.Deps) {
	g := newGameData(deps.Game)
	mux.HandleFunc("GET /api/go/vanilla/equipment-archetypes", indexRoute(gameindex.Equipment, func(raw []byte) (any, error) {
		var data struct {
			Source     string   `json:"source"`
			Archetypes []string `json:"archetypes"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		archetypes := orEmpty(data.Archetypes)
		return map[string]any{
			"source":       "common/units/equipment/*.txt · is_archetype=yes top-level keys",
			"extracted_at": indexDate(),
			"total":        len(archetypes),
			"archetypes":   archetypes,
		}, nil
	}))
	mux.HandleFunc("GET /api/go/vanilla/subunits", func(w http.ResponseWriter, r *http.Request) {
		subunits := g.subunits()
		writeJSON(w, http.StatusOK, map[string]any{
			"source":       "common/units/*.txt + dlc/*/common/units/*.txt · sub_units{} top-level scalars · land only",
			"extracted_at": g.extractedAt(),
			"total":        len(subunits),
			"stat_keys":    subunitStatKeys,
			"subunits":     subunits,
		})
	})
	mux.HandleFunc("GET /api/go/vanilla/events", func(w http.ResponseWriter, r *http.Request) {
		limit, ok1 := queryInt(r, "limit", 50)
		offset, ok2 := queryInt(r, "offset", 0)
		if !ok1 || !ok2 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "limit/offset must be integers"})
			return
		}
		q := r.URL.Query()
		writeJSON(w, http.StatusOK, listVanillaEvents(g.events(),
			q.Get("q"), q.Get("event_type"), q.Get("namespace_prefix"), limit, offset))
	})
	mux.HandleFunc("GET /api/go/vanilla/events/{ns}/{num}", func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("ns")
		num, err := strconv.Atoi(r.PathValue("num"))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "event number must be an integer"})
			return
		}
		index := g.events()
		target := fmt.Sprintf("%s.%d", ns, num)
		for _, entry := range index.entries {
			if entry.ID == target {
				writeJSON(w, http.StatusOK, map[string]any{"entry": entry, "body": g.eventBody(entry)})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{
			"detail": "vanilla event " + ns + "." + strconv.Itoa(num) + " not found"})
	})
	mux.HandleFunc("GET /api/go/vanilla/id-collisions", indexRoute(gameindex.IDCollisions, func(raw []byte) (any, error) {
		var ids struct {
			Tech             []string `json:"tech_ids"`
			FocusTree        []string `json:"focus_tree_ids"`
			DecisionCategory []string `json:"decision_category_ids"`
			DynamicModifier  []string `json:"dynamic_modifier_ids"`
		}
		if err := json.Unmarshal(raw, &ids); err != nil {
			return nil, err
		}
		at := indexDate()
		return map[string]any{
			"tech_ids":                       orEmpty(ids.Tech),
			"tech_total":                     len(ids.Tech),
			"tech_extracted_at":              at,
			"focus_tree_ids":                 orEmpty(ids.FocusTree),
			"focus_tree_total":               len(ids.FocusTree),
			"focus_extracted_at":             at,
			"decision_category_ids":          orEmpty(ids.DecisionCategory),
			"decision_category_total":        len(ids.DecisionCategory),
			"decision_category_extracted_at": at,
			"dynamic_modifier_ids":           orEmpty(ids.DynamicModifier),
			"dynamic_modifier_total":         len(ids.DynamicModifier),
			"dynamic_modifier_extracted_at":  at,
		}, nil
	}))
	mux.HandleFunc("GET /api/go/vanilla/script-names", func(w http.ResponseWriter, r *http.Request) {
		names, err := vanillaScriptNames(r.Context())
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, names)
	})
	mux.HandleFunc("GET /api/go/vanilla/structural-fields", indexRoute(gameindex.StructuralFields, func(raw []byte) (any, error) {
		var payload struct {
			Fields   []string                   `json:"fields"`
			ByEntity map[string]json.RawMessage `json:"by_entity"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		if payload.ByEntity == nil {
			payload.ByEntity = map[string]json.RawMessage{}
		}
		return map[string]any{"fields": orEmpty(payload.Fields), "by_entity": payload.ByEntity}, nil
	}))
	mux.HandleFunc("GET /api/go/vanilla/country-tags", indexRoute(gameindex.CountryTags, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string            `json:"game_version"`
			Tags        []json.RawMessage `json:"tags"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.Tags == nil {
			data.Tags = []json.RawMessage{}
		}
		return map[string]any{"game_version": data.GameVersion, "total": len(data.Tags), "tags": data.Tags}, nil
	}))

	mux.HandleFunc("GET /api/go/vanilla/country-aliases", indexRoute(gameindex.CountryAliases, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string              `json:"game_version"`
			Aliases     map[string][]string `json:"aliases"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.Aliases == nil {
			data.Aliases = map[string][]string{}
		}
		return map[string]any{"game_version": data.GameVersion, "aliases": data.Aliases}, nil
	}))

	mux.HandleFunc("GET /api/go/vanilla/state-content", indexRoute(gameindex.StateContent, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string                     `json:"game_version"`
			States      map[string]json.RawMessage `json:"states"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.States == nil {
			data.States = map[string]json.RawMessage{}
		}
		return map[string]any{"game_version": data.GameVersion, "states": data.States}, nil
	}))

	mux.HandleFunc("GET /api/go/vanilla/dlc-states", indexRoute(gameindex.DLCStates, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string              `json:"game_version"`
			DLCNames    []string            `json:"dlc_names"`
			States      map[string][]string `json:"states"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.States == nil {
			data.States = map[string][]string{}
		}
		return map[string]any{"game_version": data.GameVersion, "dlc_names": orEmpty(data.DLCNames), "states": data.States}, nil
	}))

	mux.HandleFunc("GET /api/go/vanilla/tech-sections", indexRoute(gameindex.TechSections, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string                     `json:"game_version"`
			Folders     map[string]json.RawMessage `json:"folders"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.Folders == nil {
			data.Folders = map[string]json.RawMessage{}
		}
		return map[string]any{"game_version": data.GameVersion, "folders": data.Folders}, nil
	}))

	mux.HandleFunc("GET /api/go/vanilla/unit-names", indexRoute(gameindex.UnitNames, func(raw []byte) (any, error) {
		var data struct {
			GameVersion string                       `json:"game_version"`
			Subunits    map[string]map[string]string `json:"subunits"`
			Equipment   map[string]map[string]string `json:"equipment"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		if data.Subunits == nil {
			data.Subunits = map[string]map[string]string{}
		}
		if data.Equipment == nil {
			data.Equipment = map[string]map[string]string{}
		}
		return map[string]any{"game_version": data.GameVersion, "subunits": data.Subunits, "equipment": data.Equipment}, nil
	}))
}

func indexRoute(name string, view func(raw []byte) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := gameindex.Peek(name)
		if errors.Is(err, gameindex.ErrNoGame) {
			raw, err = []byte("{}"), nil
		}
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		body, err := view(raw)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	}
}

func indexDate() any {
	built := gameindex.Default().Status().BuiltAt
	if len(built) < 10 {
		return nil
	}
	return built[:10]
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func queryInt(r *http.Request, key string, def int) (int, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

type idSetFile struct {
	Total       int      `json:"total"`
	IDs         []string `json:"ids"`
	ExtractedAt any      `json:"extracted_at"`
}

func parseIDSet(raw []byte) idSetFile {
	var f idSetFile
	_ = json.Unmarshal(raw, &f)
	if f.IDs == nil {
		f.IDs = []string{}
	}
	return f
}

func vanillaScriptNames(ctx context.Context) (map[string]any, error) {
	read := func(name string) ([]byte, error) {
		if err := gameindex.CheckReady(name); err != nil {
			return nil, err
		}
		raw, err := gameindex.Get(ctx, name)
		if errors.Is(err, gameindex.ErrNoGame) {
			return []byte("{}"), nil
		}
		return raw, err
	}
	var sets [3]idSetFile
	for i, name := range []string{gameindex.VanillaTriggers, gameindex.VanillaEffects, gameindex.MathFunctions} {
		raw, err := read(name)
		if err != nil {
			return nil, err
		}
		sets[i] = parseIDSet(raw)
	}
	triggers, effects, math := sets[0], sets[1], sets[2]
	extractedAt := triggers.ExtractedAt
	if s, ok := extractedAt.(string); extractedAt == nil || ok && s == "" {
		extractedAt = effects.ExtractedAt
	}
	return map[string]any{
		"triggers":            triggers.IDs,
		"effects":             effects.IDs,
		"math_functions":      math.IDs,
		"trigger_total":       triggers.Total,
		"effect_total":        effects.Total,
		"math_function_total": len(math.IDs),
		"extracted_at":        extractedAt,
	}, nil
}

func listVanillaEvents(index *eventIndex, q, eventType, namespacePrefix string, limit, offset int) map[string]any {
	qLower := strings.ToLower(strings.TrimSpace(q))
	nsPrefix := strings.ToLower(strings.TrimSpace(namespacePrefix))
	typ := strings.TrimSpace(eventType)
	matched := []vanillaEvent{}
	for _, entry := range index.entries {
		if typ != "" && entry.Type != typ {
			continue
		}
		if nsPrefix != "" && !strings.HasPrefix(strings.ToLower(entry.NS), nsPrefix) {
			continue
		}
		if qLower != "" && !strings.Contains(entry.hay, qLower) {
			continue
		}
		matched = append(matched, entry)
	}
	lim, off := min(max(limit, 1), 200), max(offset, 0)
	page := []vanillaEvent{}
	if off < len(matched) {
		page = matched[off:min(off+lim, len(matched))]
	}
	return map[string]any{
		"total_matching": len(matched),
		"limit":          lim,
		"offset":         off,
		"items":          page,
		"meta": map[string]any{
			"total":        len(index.entries),
			"generated_at": index.generatedAt,
			"namespaces":   index.namespaces,
		},
	}
}

func (g *gameData) eventBody(entry vanillaEvent) string {
	game := g.current()
	if game == nil || entry.File == "" || entry.Line <= 0 {
		return ""
	}
	clean := strings.ReplaceAll(entry.File, "\\", "/")
	for _, seg := range strings.Split(clean, "/") {
		if seg == ".." {
			return ""
		}
	}
	raw, err := game.ReadFile(path.Join("events", clean))
	if err != nil {
		return ""
	}
	lines := splitLinesKeepEnds(strings.TrimPrefix(string(raw), "\uFEFF"))
	if entry.Line > len(lines) {
		return ""
	}
	var body strings.Builder
	depth, started := 0, false
	for i := entry.Line - 1; i < len(lines); i++ {
		body.WriteString(lines[i])
		for _, ch := range lines[i] {
			switch ch {
			case '{':
				depth++
				started = true
			case '}':
				depth--
				if started && depth == 0 {
					return body.String()
				}
			}
		}
		if (i+1)-entry.Line > 500 {
			break
		}
	}
	return body.String()
}

func splitLinesKeepEnds(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i+1])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}
