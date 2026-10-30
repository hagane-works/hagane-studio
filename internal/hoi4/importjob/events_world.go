package importjob

import (
	"encoding/json"
	"fmt"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"haganeworks/internal/hoi4/importsvc"
	eventsvc "haganeworks/internal/hoi4/importsvc/events"
	worldsvc "haganeworks/internal/hoi4/importsvc/world"
	"haganeworks/internal/hoi4/paradox"
)

type EventsWorldProgress struct {
	FilesDone, FilesTotal int
	Rows                  int64
}
type EventsWorldCounts map[string]int64

func (c EventsWorldCounts) Total() int64 {
	var n int64
	for _, v := range c {
		n += v
	}
	return n
}

type EventsWorldStageOptions struct {
	ProjectID                                                                        int64
	CountryTag                                                                       string
	ImportEvents, ImportDecisions, ImportBookmarks, ImportHistoryFiles, ImportStates bool
}

func defaultEventsWorldStageOptions() EventsWorldStageOptions {
	return EventsWorldStageOptions{ImportEvents: true, ImportDecisions: true, ImportBookmarks: true, ImportHistoryFiles: true, ImportStates: true}
}

type eventsWorldRow struct {
	kind, key, parent, source string
	payload                   []byte
}

func eventsWorldFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		n := strings.ToLower(filepath.ToSlash(rel))
		if !strings.HasSuffix(n, ".txt") {
			return nil
		}
		if strings.HasPrefix(n, "events/") || strings.HasPrefix(n, "common/decisions/") || strings.HasPrefix(n, "common/decisions/categories/") || strings.HasPrefix(n, "common/bookmarks/") || strings.HasPrefix(n, "history/countries/") || strings.HasPrefix(n, "history/states/") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func parseEventsWorld(rel, text string) ([]eventsWorldRow, error) {
	n := strings.ToLower(filepath.ToSlash(rel))
	var rows []eventsWorldRow
	add := func(kind, key, parent string, v any) error {
		if strings.TrimSpace(key) == "" {
			return nil
		}
		b, e := marshalPGJSON(nativeEW(v))
		if e == nil {
			rows = append(rows, eventsWorldRow{kind, key, parent, rel, b})
		}
		return e
	}
	switch {
	case strings.HasPrefix(n, "events/"):
		se, _, err := worldsvc.ImportSuperEvents(text)
		if err != nil {
			return nil, err
		}
		if len(se) > 0 {
			for _, d := range se {
				m := nativeEW(d).(map[string]any)
				c := upperString(m["country_tag"])
				num := jsonNumber(m["number"])
				if e := add("super_event", c+":"+num, "", m); e != nil {
					return nil, e
				}
			}
			return rows, nil
		}
		nss, evs, _, err := eventsvc.ImportEvents(text)
		if err != nil {
			return nil, err
		}
		for _, ns := range nss {
			if e := add("event_namespace", ns, "", map[string]any{"namespace": ns}); e != nil {
				return nil, e
			}
		}
		for _, ev := range evs {
			id := stringValue(ev["event_id"])
			parent := ""
			if p := strings.IndexByte(id, '.'); p > 0 {
				parent = id[:p]
			}
			ev["source_file"] = rel
			if err = add("event", id, parent, ev); err != nil {
				return nil, err
			}
		}
	case strings.HasPrefix(n, "common/decisions/categories/"):
		cats, _, err := eventsvc.ImportDecisionCategories(text)
		if err != nil {
			return nil, err
		}
		for _, d := range cats {
			if err = add("decision_category", stringValue(d["category_id"]), "", d); err != nil {
				return nil, err
			}
		}
	case strings.HasPrefix(n, "common/decisions/"):
		cats, decs, _, err := eventsvc.ImportDecisions(text)
		if err != nil {
			return nil, err
		}
		for _, d := range cats {
			if err = add("decision_category", stringValue(d["category_id"]), "", d); err != nil {
				return nil, err
			}
		}
		for _, d := range decs {
			d["source_file"] = rel
			if err = add("decision", stringValue(d["decision_id"]), stringValue(d["category_id"]), d); err != nil {
				return nil, err
			}
		}
	case strings.HasPrefix(n, "common/bookmarks/"):
		items, _ := worldsvc.ImportBookmarks(text)
		for _, d := range items {
			m := nativeEW(d).(map[string]any)
			if e := add("bookmark", stringValue(m["bookmark_id"]), "", m); e != nil {
				return nil, e
			}
		}
	case strings.HasPrefix(n, "history/countries/"):
		items, _, err := worldsvc.ImportHistoryFiles(text, rel)
		if err != nil {
			return nil, err
		}
		for _, d := range items {
			m := nativeEW(d).(map[string]any)
			if err = add("history", upperString(m["country_tag"]), "", m); err != nil {
				return nil, err
			}
		}
	case strings.HasPrefix(n, "history/states/"):
		items, _ := worldsvc.ImportStateFiles(text, rel)
		for _, d := range items {
			m := nativeEW(d).(map[string]any)
			m["source_file"] = rel
			if e := add("state", jsonNumber(m["state_id"]), "", m); e != nil {
				return nil, e
			}
		}
	}
	return rows, nil
}

func nativeEW(v any) any {
	switch x := v.(type) {
	case *worldsvc.PyDict:
		m := map[string]any{}
		for _, k := range x.Keys() {
			v, _ := x.Get(k)
			m[k] = nativeEW(v)
		}
		return m
	case *importsvc.RawExtras:
		m := map[string]any{}
		for _, k := range x.Keys {
			v, _ := x.Get(k)
			m[k] = nativeEW(v)
		}
		return m
	case importsvc.ScopeVerdict:
		k, c, s := x.StorageTuple()
		return map[string]any{"kind": k, "country_tag": c, "shared_country_tags": s}
	case *big.Int:
		return json.Number(x.String())
	case paradox.Scalar:
		return importsvc.PyScalarStr(x)
	case []any:
		r := make([]any, len(x))
		for i := range x {
			r[i] = nativeEW(x[i])
		}
		return r
	case map[string]any:
		m := map[string]any{}
		for k, v := range x {
			m[k] = nativeEW(v)
		}
		return m
	default:
		return v
	}
}
func stringValue(v any) string { s, _ := v.(string); return strings.TrimSpace(s) }
func upperString(v any) string { return strings.ToUpper(stringValue(v)) }
func jsonNumber(v any) string {
	switch x := v.(type) {
	case json.Number:
		return x.String()
	case *big.Int:
		return x.String()
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(x)
	}
}

var eventDLCNamespacePrefixes = []string{"mtg_", "wtt_", "aat_", "bba_", "dod_", "goe_", "got_", "nsb_", "lar_", "bftb_", "bop_", "tfv_", "ted_", "tsr_", "war_", "aof_", "mdp_"}
var eventGlobalNamespaceTokens = map[string]bool{"news": true, "generic": true, "shared": true, "world": true, "global": true, "achievement": true, "achievements": true, "election": true, "elections": true, "decisions": true, "panay": true, "capitulation": true}

func namespaceTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool { return r == '_' || r == '.' || r == '-' })
}

type eventScope struct {
	allowed map[string]bool
	primary string
	aliases countryAliasData

	mu      sync.Mutex
	foreign map[string]bool
}

func newEventScope(allowed map[string]bool, primary string, aliases countryAliasData) *eventScope {
	return &eventScope{allowed: allowed, primary: primary, aliases: aliases, foreign: map[string]bool{}}
}

func (e *eventScope) foreignNamespace(ns string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	foreign, ok := e.foreign[ns]
	if !ok {
		foreign = namespaceForeignToProject(ns, e.allowed, e.aliases.aliases)
		e.foreign[ns] = foreign
	}
	return foreign
}

func namespaceForeignForTag(ns, tag string, aliases map[string][]string) bool {
	tokens := namespaceTokens(ns)
	set := map[string]bool{}
	for _, x := range tokens {
		set[x] = true
	}
	if set[strings.ToLower(tag)] {
		return false
	}
	for _, alias := range aliases[strings.ToUpper(tag)] {
		at := strings.Split(alias, "_")
		if len(at) == 1 && set[alias] {
			return false
		}
		for i := 0; i+len(at) <= len(tokens); i++ {
			if strings.Join(tokens[i:i+len(at)], "_") == alias {
				return false
			}
		}
	}
	lower := strings.ToLower(strings.TrimSpace(ns))
	for _, p := range eventDLCNamespacePrefixes {
		if strings.HasPrefix(lower, p) {
			rest := namespaceTokens(strings.TrimPrefix(lower, p))
			for _, x := range rest {
				if x == "news" {
					return false
				}
			}
			break
		}
	}
	for _, x := range tokens {
		if eventGlobalNamespaceTokens[x] {
			return false
		}
	}
	return true
}
func namespaceForeignToProject(ns string, allowed map[string]bool, aliases map[string][]string) bool {
	if len(allowed) == 0 {
		return false
	}
	for tag := range allowed {
		if !namespaceForeignForTag(ns, tag, aliases) {
			return false
		}
	}
	return true
}
func knownForeignPrefix(id string, allowed, baseTags map[string]bool) bool {
	if i := strings.IndexByte(id, '_'); i >= 2 && i <= 5 {
		p := strings.ToUpper(id[:i])
		return baseTags[p] && !allowed[p]
	}
	return false
}
func namespacePromotesOwn(ns, primary string, aliases map[string][]string) bool {
	t := namespaceTokens(ns)
	set := map[string]bool{}
	for _, x := range t {
		set[x] = true
	}
	if set[strings.ToLower(primary)] {
		return true
	}
	for _, a := range aliases[strings.ToUpper(primary)] {
		if !strings.Contains(a, "_") && set[a] {
			return true
		}
	}
	return false
}
func filterEventRows(rows []eventsWorldRow, scope *eventScope) []eventsWorldRow {
	allowed, primary := scope.allowed, scope.primary
	if len(allowed) == 0 {
		return rows
	}
	out := make([]eventsWorldRow, 0, len(rows))
	keptNS := map[string]bool{}
	for _, r := range rows {
		if r.kind != "event" {
			continue
		}
		if r.parent != "" && scope.foreignNamespace(r.parent) {
			continue
		}
		if knownForeignPrefix(r.key, allowed, scope.aliases.tags) {
			continue
		}
		var p map[string]any
		if json.Unmarshal(r.payload, &p) != nil {
			continue
		}
		v, _ := p["_scope_verdict"].(map[string]any)
		kind := stringValue(v["kind"])
		ctag := upperString(v["country_tag"])
		switch kind {
		case "own":
			if ctag != "" && !allowed[ctag] {
				continue
			}
		case "global_except":
			for _, x := range anyStrings(v["shared_country_tags"]) {
				if strings.EqualFold(x, primary) {
					goto dropped
				}
			}
		case "global":
			if namespacePromotesOwn(r.parent, primary, scope.aliases.aliases) {
				v["kind"] = "own"
				v["country_tag"] = primary
				v["shared_country_tags"] = nil
			}
		}
		r.payload, _ = marshalPGJSON(p)
		out = append(out, r)
		keptNS[r.parent] = true
		continue
	dropped:
	}
	for _, r := range rows {
		if r.kind != "event" && r.kind != "event_namespace" {
			out = append(out, r)
		} else if r.kind == "event_namespace" && keptNS[r.key] {
			out = append(out, r)
		}
	}
	return filterDecisionRows(out, allowed, scope.aliases.tags)
}

func verdictExcludesAll(v map[string]any, allowed map[string]bool) bool {
	if len(allowed) == 0 {
		return false
	}
	kind := stringValue(v["kind"])
	ct := upperString(v["country_tag"])
	tags := map[string]bool{}
	for _, x := range anyStrings(v["shared_country_tags"]) {
		tags[strings.ToUpper(x)] = true
	}
	for a := range allowed {
		excluded := false
		switch kind {
		case "own":
			excluded = ct != "" && ct != a
		case "explicit":
			excluded = len(tags) > 0 && !tags[a]
		case "global_except":
			excluded = tags[a]
		}
		if !excluded {
			return false
		}
	}
	return true
}
func filterDecisionRows(rows []eventsWorldRow, allowed, baseTags map[string]bool) []eventsWorldRow {
	excludedCats := map[string]bool{}
	for _, r := range rows {
		if r.kind != "decision_category" {
			continue
		}
		var p map[string]any
		if json.Unmarshal(r.payload, &p) == nil {
			if v, ok := p["_scope_verdict"].(map[string]any); ok && verdictExcludesAll(v, allowed) {
				excludedCats[r.key] = true
			}
		}
	}
	out := make([]eventsWorldRow, 0, len(rows))
	referenced := map[string]bool{}
	for _, r := range rows {
		if r.kind != "decision" {
			continue
		}
		if knownForeignPrefix(r.key, allowed, baseTags) || excludedCats[r.parent] {
			continue
		}
		var p map[string]any
		if json.Unmarshal(r.payload, &p) == nil {
			if v, ok := p["_scope_verdict"].(map[string]any); ok && verdictExcludesAll(v, allowed) {
				continue
			}
		}
		out = append(out, r)
		referenced[r.parent] = true
	}
	for _, r := range rows {
		if r.kind != "decision" && r.kind != "decision_category" {
			out = append(out, r)
		} else if r.kind == "decision_category" && (referenced[r.key] || !excludedCats[r.key]) {

			out = append(out, r)
		}
	}
	return out
}
func anyStrings(v any) []string {
	a, _ := v.([]any)
	out := make([]string, 0, len(a))
	for _, x := range a {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	if s, ok := v.([]string); ok {
		return s
	}
	return out
}

type EventsWorldMergeOptions struct{ ConflictPolicy string }

type worldStage struct {
	rows []worldRow
}

type worldRow struct {
	kind, key, parent, source string
	data                      payload
}

func (s *session) stageEventsWorld(options EventsWorldStageOptions, progress func(EventsWorldProgress)) (EventsWorldCounts, error) {
	if options == (EventsWorldStageOptions{}) {
		options = defaultEventsWorldStageOptions()
	}

	if options.CountryTag == "" && options.ProjectID > 0 {
		options.CountryTag, _ = s.record["country_tag"].(string)
	}
	options.CountryTag = strings.ToUpper(strings.TrimSpace(options.CountryTag))
	allowed := map[string]bool{}
	if options.CountryTag != "" {
		allowed[options.CountryTag] = true
	}
	if options.ProjectID > 0 {
		for _, tag := range anyStrings(s.record["country_tags"]) {
			if tag = strings.ToUpper(strings.TrimSpace(tag)); tag != "" {
				allowed[tag] = true
			}
		}
	}
	facts, err := s.gameFacts()
	if err != nil {
		return nil, err
	}
	scope := newEventScope(allowed, options.CountryTag, facts.aliases)
	files, err := eventsWorldFiles(s.root)
	if err != nil {
		return nil, err
	}
	counts := EventsWorldCounts{}
	var staged []eventsWorldRow
	err = inFileOrder(s.ctx, len(files), func(i int) (fileRows[eventsWorldRow], error) {
		rel, _ := filepath.Rel(s.root, files[i])
		rel = filepath.ToSlash(rel)
		lower := strings.ToLower(rel)
		if (strings.HasPrefix(lower, "events/") && !options.ImportEvents) || (strings.HasPrefix(lower, "common/decisions/") && !options.ImportDecisions) || (strings.HasPrefix(lower, "common/bookmarks/") && !options.ImportBookmarks) || (strings.HasPrefix(lower, "history/countries/") && !options.ImportHistoryFiles) || (strings.HasPrefix(lower, "history/states/") && !options.ImportStates) {
			return fileRows[eventsWorldRow]{}, nil
		}
		body, err := os.ReadFile(files[i])
		if err != nil {
			return fileRows[eventsWorldRow]{}, fmt.Errorf("read %s: %w", rel, err)
		}
		rows, err := parseEventsWorld(rel, decodeScriptBytes(body))
		body = nil
		if err != nil {
			return fileRows[eventsWorldRow]{rel: rel, invalid: err}, nil
		}
		rows = filterEventRows(rows, scope)
		for i := range rows {
			if options.CountryTag != "" && rows[i].kind == "super_event" {
				var m map[string]any
				if json.Unmarshal(rows[i].payload, &m) == nil && stringValue(m["country_tag"]) == "" {
					m["country_tag"] = options.CountryTag
					rows[i].key = options.CountryTag + ":" + jsonNumber(m["number"])
					rows[i].payload, _ = marshalPGJSON(m)
				}
			}
			if options.CountryTag != "" && rows[i].kind == "history" && rows[i].key == "" {
				rows[i].key = options.CountryTag
			}
		}
		return fileRows[eventsWorldRow]{rel: rel, rows: rows}, nil
	}, func(i int, file fileRows[eventsWorldRow]) error {
		switch {
		case file.rel == "":
			return nil
		case file.invalid != nil:
			s.warnParse(file.rel, file.invalid)
			return nil
		}
		for _, r := range file.rows {
			staged = append(staged, r)
			counts[r.kind]++
		}
		if progress != nil {
			progress(EventsWorldProgress{i + 1, len(files), counts.Total()})
		}
		return nil
	})
	if err != nil {
		return counts, err
	}

	staged = worldLastWins(staged)
	s.world.rows = make([]worldRow, 0, len(staged))
	for _, r := range staged {
		s.world.rows = append(s.world.rows, worldRow{kind: r.kind, key: r.key, parent: r.parent, source: r.source, data: decodePayload(r.payload)})
	}
	s.recordWorldImages()
	return counts, nil
}

func (s *session) recordWorldImages() {
	for _, r := range s.world.rows {
		switch r.kind {
		case "event":
			s.needImage(importassets.ResolveSprite("event_picture", r.data.str("picture"), s.gfx))
		case "bookmark":
			s.needImage(importassets.ResolveSprite("bookmark_picture", r.data.str("picture"), s.gfx))
		case "decision":
			s.needImage(importassets.ResolveSprite("decision_icon", r.data.str("icon"), s.gfx))
		case "decision_category":
			s.needImage(importassets.ResolveSprite("decision_category_icon", r.data.str("icon"), s.gfx))
		case "super_event":
			s.needImage(worldSuperEventPicture(r.data.str("picture_path")))
		}
	}
}

func worldLastWins(rows []eventsWorldRow) []eventsWorldRow {
	last := make(map[string]int, len(rows))
	for i, r := range rows {
		if r.kind != "decision_category" {
			last[r.kind+"\x00"+r.key] = i
		}
	}
	out := make([]eventsWorldRow, 0, len(rows))
	for i, r := range rows {
		if r.kind == "decision_category" || last[r.kind+"\x00"+r.key] == i {
			out = append(out, r)
		}
	}
	return out
}

func worldSuperEventPicture(path string) string {
	path = strings.ReplaceAll(path, `\\`, "/")
	path = strings.ReplaceAll(path, `\`, "/")
	return strings.Trim(path, `"`)
}

func worldEventFileNamespace(source string) string {
	if i := strings.LastIndex(source, "/"); i >= 0 {
		source = source[i+1:]
	}
	return strings.TrimSuffix(source, ".txt")
}

func worldCategoryDefinition(source string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(source, `\`, "/")), "decisions/categories/")
}

func worldScope(p payload) (countryTag, kind string, shared any) {
	verdict, _ := p["_scope_verdict"].(map[string]any)
	countryTag, kind = "", "global"
	if value := verdict["country_tag"]; value != nil {
		countryTag = jsonText(value)
	}
	if value := verdict["kind"]; value != nil {
		kind = jsonText(value)
	}
	return countryTag, kind, verdict["shared_country_tags"]
}

func worldNullIfEmpty(p payload, key string) any {
	if text, ok := p.text(key); ok && text != "" {
		return text
	}
	return nil
}

func worldNullableBool(p payload, key string) any {
	text, ok := p.text(key)
	if !ok || text == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "t", "yes", "y", "on", "1":
		return true
	case "false", "f", "no", "n", "off", "0":
		return false
	}
	return nil
}

func worldLpad4(text string) string {
	runes := []rune(text)
	if len(runes) >= 4 {
		return string(runes[:4])
	}
	return strings.Repeat("0", 4-len(runes)) + text
}

func (s *session) mergeEventsWorld(tx *project.Tx, options EventsWorldMergeOptions) error {
	skip := strings.EqualFold(strings.TrimSpace(options.ConflictPolicy), "skip")
	write := func(table *keyedTable, row project.Row, update []string) {
		if skip {
			table.insertNew(row)
		} else {
			table.upsert(row, update)
		}
	}
	rows := s.world.rows

	projectCountry, _ := tx.Record()["country_tag"].(string)

	namespaces := newKeyedTable(tx, "event_namespaces", "namespace")
	namespace := func(name string) project.Row {
		return project.Row{"namespace": name, "country_tag": projectCountry, "scope_kind": "own", "shared_country_tags": nil}
	}

	for _, r := range rows {
		if r.kind == "event_namespace" {
			write(namespaces, namespace(r.key), nil)
		}
	}

	for _, r := range rows {
		if r.kind == "event" && r.parent != "" {
			namespaces.insertNew(namespace(r.parent))
		}
	}

	for _, r := range rows {
		if r.kind == "event" && r.parent == "" {
			namespaces.insertNew(namespace(worldEventFileNamespace(r.source)))
		}
	}

	wanted := map[string]bool{}
	for _, r := range rows {
		if r.kind == "decision" && r.parent != "" {
			wanted[r.parent] = true
		}
	}
	wantedIDs := make([]string, 0, len(wanted))
	for id := range wanted {
		wantedIDs = append(wantedIDs, id)
	}
	sort.Strings(wantedIDs)
	chosen := map[string]*worldRow{}
	for i := range rows {
		r := &rows[i]
		if r.kind != "decision_category" || !wanted[r.key] {
			continue
		}

		if current := chosen[r.key]; current == nil || worldCategoryDefinition(r.source) || !worldCategoryDefinition(current.source) {
			chosen[r.key] = r
		}
	}
	categories := newKeyedTable(tx, "decision_categories", "category_id")
	categoryOverwrite := []string{"icon", "picture", "priority", "on_map_area", "allowed", "visible", "raw_extras", "country_tag", "scope_kind", "shared_country_tags"}
	for _, key := range wantedIDs {
		r := chosen[key]
		if r == nil {
			continue
		}
		p := r.data
		names, descs := s.locNames(key), s.locNames(key+"_desc")
		countryTag, kind, shared := worldScope(p)
		row := project.Row{
			"category_id": key, "icon": p.str("icon"), "picture": p.str("picture"), "priority": p.intOr("priority", 0),
			"on_map_area": p.str("on_map_area"), "allowed": p.str("allowed"), "visible": p.str("visible"),
			"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "country_tag": countryTag, "scope_kind": kind, "shared_country_tags": shared,
			"name_en": s.locStr(key, "en"), "name_zh": s.locStr(key, "zh"), "desc_en": s.locStr(key+"_desc", "en"), "desc_zh": s.locStr(key+"_desc", "zh"),
			"display_name_locs": s.locNamesOrEmpty(key), "description_locs": s.locNamesOrEmpty(key + "_desc"), "visibility": "visible",
		}
		id, exists := categories.find(key)
		if !exists {
			categories.insertNew(row)
			continue
		}
		if skip {
			continue
		}

		if _, err := tx.Update("decision_categories", id, func(existing project.Row) {
			for _, column := range categoryOverwrite {
				existing[column] = row[column]
			}
			for _, column := range []string{"name_en", "name_zh", "desc_en", "desc_zh"} {
				if row[column] != "" {
					existing[column] = row[column]
				}
			}
			if names != nil {
				existing["display_name_locs"] = names
			}
			if descs != nil {
				existing["description_locs"] = descs
			}
		}); err != nil {
			return err
		}
	}

	for _, key := range wantedIDs {
		categories.insertNew(project.Row{
			"category_id": key, "country_tag": "", "scope_kind": "global",
			"name_en": s.locStr(key, "en"), "name_zh": s.locStr(key, "zh"), "desc_en": s.locStr(key+"_desc", "en"), "desc_zh": s.locStr(key+"_desc", "zh"),
			"display_name_locs": s.locNamesOrEmpty(key), "description_locs": s.locNamesOrEmpty(key + "_desc"), "visibility": "visible",
		})
	}

	events := newKeyedTable(tx, "events", "event_id")
	eventUpdate := []string{"namespace_id", "event_number", "event_type", "title", "description", "picture", "is_triggered_only", "fire_only_once", "hidden", "major", "trigger", "mean_time_to_happen", "immediate", "after", "options", "timeout_days", "raw_extras", "country_tag", "scope_kind", "shared_country_tags", "source", "title_en", "title_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs", "source_file"}
	for _, r := range rows {
		if r.kind != "event" {
			continue
		}
		name := r.parent
		if name == "" {
			name = worldEventFileNamespace(r.source)
		}
		namespaceID, ok := namespaces.find(name)
		if !ok {
			continue
		}
		p := r.data
		titleKey, descKey := r.key+".t", r.key+".d"
		titleEN, ok := s.locText(titleKey, "en")
		if !ok {
			titleEN = p.str("title_en")
		}
		descEN, ok := s.locText(descKey, "en")
		if !ok {
			descEN = p.str("desc_en")
		}
		countryTag, kind, shared := worldScope(p)
		write(events, project.Row{
			"namespace_id": namespaceID, "event_id": r.key, "event_number": p.intOr("event_number", 0),
			"event_type": p.strOr("event_type", "country_event"), "title": p.jsonOr("title", ""), "description": p.jsonOr("description", ""),
			"picture": p.str("picture"), "is_triggered_only": p.boolOr("is_triggered_only", false), "fire_only_once": p.boolOr("fire_only_once", false),
			"hidden": p.boolOr("hidden", false), "major": p.boolOr("major", false), "trigger": p.str("trigger"),
			"mean_time_to_happen": p.str("mean_time_to_happen"), "immediate": p.str("immediate"), "after": p.str("after"),
			"options": p.jsonOr("options", []any{}), "timeout_days": p.intOr("timeout_days", 0), "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
			"country_tag": countryTag, "scope_kind": kind, "shared_country_tags": shared, "source": "imported",
			"title_en": titleEN, "title_zh": s.locStr(titleKey, "zh"), "desc_en": descEN, "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(titleKey), "description_locs": s.locNamesOrEmpty(descKey),
			"has_obs": false, "source_file": r.source, "auto_fire_on_startup": false, "auto_fire_after_days": 0, "on_action_attach": "",
		}, eventUpdate)
	}

	decisions := newKeyedTable(tx, "decisions", "decision_id")
	decisionUpdate := []string{"category_id", "icon", "allowed", "available", "visible", "cost", "fire_only_once", "cancel_if_not_visible", "days_remove", "days_re_enable", "complete_effect", "remove_effect", "cancel_trigger", "cancel_effect", "modifier", "activation", "timeout_effect", "days_mission_timeout", "selectable_mission", "is_good", "targeted_modifier", "targets", "target_array", "target_root_trigger", "target_trigger", "state_target", "highlight_states", "priority", "custom_cost_trigger", "custom_cost_text", "on_map_mode", "war_with_on_add", "war_with_on_remove", "ai_will_do", "raw_extras", "country_tag", "scope_kind", "shared_country_tags", "source", "source_file", "name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs"}
	for _, r := range rows {
		if r.kind != "decision" {
			continue
		}
		categoryID, ok := categories.find(r.parent)
		if !ok {
			continue
		}
		p := r.data
		nameEN, ok := s.locText(r.key, "en")
		if !ok {
			nameEN = r.key
		}
		descKey := r.key + "_desc"
		countryTag, kind, shared := worldScope(p)
		row := project.Row{
			"category_id": categoryID, "decision_id": r.key, "cost": p.intOr("cost", 0),
			"fire_only_once": p.boolOr("fire_only_once", false), "cancel_if_not_visible": p.boolOr("cancel_if_not_visible", false),
			"days_remove": p.intOr("days_remove", 0), "days_re_enable": p.intOr("days_re_enable", 0),
			"days_mission_timeout": p.intOr("days_mission_timeout", 0), "selectable_mission": p.boolOr("selectable_mission", false),
			"is_good": p.boolOr("is_good", false), "targeted_modifier": p.jsonOr("targeted_modifier", map[string]any{}),
			"targets": p.jsonOr("targets", []any{}), "target_array": p.jsonOr("target_array", []any{}),
			"highlight_states": p.jsonOr("highlight_states", []any{}), "priority": p.intOr("priority", 0),
			"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "country_tag": countryTag, "scope_kind": kind, "shared_country_tags": shared,
			"source": "imported", "source_file": r.source,
			"name_en": nameEN, "name_zh": s.locStr(r.key, "zh"), "desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(r.key), "description_locs": s.locNamesOrEmpty(descKey),
			"effect_tooltip_locs": map[string]any{}, "has_obs": false,
		}

		for _, column := range []string{"icon", "allowed", "available", "visible", "complete_effect", "remove_effect", "cancel_trigger", "cancel_effect", "modifier", "activation", "timeout_effect", "target_root_trigger", "target_trigger", "state_target", "custom_cost_trigger", "custom_cost_text", "on_map_mode", "war_with_on_add", "war_with_on_remove", "ai_will_do"} {
			row[column] = p.str(column)
		}
		write(decisions, row, decisionUpdate)
	}

	superEvents := newKeyedTable(tx, "super_events", "country_tag", "number")
	superEventUpdate := []string{"title_en", "title_zh", "desc_en", "desc_zh", "quote_en", "quote_zh", "btn_en", "btn_zh", "display_name_locs", "description_locs", "picture_path", "song_token", "options", "auto_fire_on_startup", "auto_fire_after_days", "source"}
	for _, r := range rows {
		if r.kind != "super_event" {
			continue
		}
		p := r.data
		number, ok := p.int("number")
		if !ok {

			return fmt.Errorf("merge super event %s: no number", r.key)
		}
		var countryTag any
		if tag := strings.ToUpper(p.str("country_tag")); tag != "" {
			countryTag = tag
		} else if tag, ok := tx.Record()["country_tag"].(string); ok {
			countryTag = strings.ToUpper(tag)
		}
		locKey := func(part string) string {
			if countryTag == nil {
				return ""
			}
			return countryTag.(string) + "_SE_" + part + "_" + worldLpad4(p.str("number"))
		}
		title, desc, quote, button := locKey("title"), locKey("desc"), locKey("quote"), locKey("btn")
		row := project.Row{
			"number": number, "country_tag": countryTag,
			"title_en": s.locStr(title, "en"), "title_zh": s.locStr(title, "zh"), "desc_en": s.locStr(desc, "en"), "desc_zh": s.locStr(desc, "zh"),
			"quote_en": s.locStr(quote, "en"), "quote_zh": s.locStr(quote, "zh"), "btn_en": s.locStr(button, "en"), "btn_zh": s.locStr(button, "zh"),
			"display_name_locs": s.locNamesOrEmpty(title), "description_locs": s.locNamesOrEmpty(desc),

			"quote_locs": map[string]any{}, "button_locs": map[string]any{},
			"picture_path": p.str("picture_path"), "song_token": p.str("song_token"), "options": p.jsonOr("options", []any{}),
			"auto_fire_on_startup": p.boolOr("auto_fire_on_startup", false), "auto_fire_after_days": p.intOr("auto_fire_after_days", 0),
			"source": "imported", "has_obs": false,
		}
		if countryTag == nil {

			tx.Insert("super_events", row)
			continue
		}
		write(superEvents, row, superEventUpdate)
	}

	bookmarks := newKeyedTable(tx, "bookmarks", "bookmark_id")
	bookmarkUpdate := []string{"name_en", "name_zh", "desc_en", "desc_zh", "display_name_locs", "description_locs", "date", "picture", "default_country", "default_flag", "selectable_countries", "filters", "source", "raw_extras"}
	for _, r := range rows {
		if r.kind != "bookmark" {
			continue
		}
		p := r.data
		nameKey, descKey := strings.ToUpper(r.key)+"_NAME", strings.ToUpper(r.key)+"_DESC"
		write(bookmarks, project.Row{
			"bookmark_id": r.key,
			"name_en":     s.locStr(nameKey, "en"), "name_zh": s.locStr(nameKey, "zh"), "desc_en": s.locStr(descKey, "en"), "desc_zh": s.locStr(descKey, "zh"),
			"display_name_locs": s.locNamesOrEmpty(nameKey), "description_locs": s.locNamesOrEmpty(descKey),
			"date": p.strOr("date", "1936.1.1.12"), "picture": p.strOr("picture", "GFX_select_date_1936"),
			"default_country": p.str("default_country"), "default_flag": p.boolOr("default_flag", false),
			"selectable_countries": p.jsonOr("selectable_countries", []any{}), "filters": p.jsonOr("filters", []any{}),
			"source": "imported", "raw_extras": p.jsonOr("raw_extras", map[string]any{}),
		}, bookmarkUpdate)
	}

	history := newKeyedTable(tx, "history_files", "country_tag")
	historyUpdate := []string{"capital", "set_politics_ruling_party", "set_politics_last_election", "set_politics_election_frequency", "set_politics_elections_allowed", "set_popularities_democratic", "set_popularities_communism", "set_popularities_fascism", "set_popularities_neutrality", "set_popularities", "set_research_slots", "set_convoys", "set_stability", "set_war_support", "add_ideas_extra", "remove_ideas", "complete_national_focus_ids", "set_technology_ids", "startup_flags", "give_guarantee_tags", "add_to_faction", "set_autonomy", "diplomatic_relation", "add_to_faction_tags", "set_cosmetic_tag", "set_oob", "set_air_oob", "set_naval_oob", "date_blocks", "raw_script", "source"}
	for _, r := range rows {
		if r.kind != "history" || r.key == "" {
			continue
		}
		p := r.data

		floatOr := func(key string) float64 {
			if f, ok := p.float(key); ok {
				return f
			}
			return -1
		}

		factionTags, ok := p.json("add_to_faction_tags")
		if !ok {
			factionTags = []any{}
			if faction := p.str("add_to_faction"); faction != "" {
				factionTags = []any{faction}
			}
		}
		row := project.Row{
			"country_tag": r.key, "capital": p.intOr("capital", 0),
			"set_politics_election_frequency": p.intOr("set_politics_election_frequency", 0),
			"set_politics_elections_allowed":  worldNullableBool(p, "set_politics_elections_allowed"),
			"set_popularities":                p.jsonOr("set_popularities", map[string]any{}),
			"set_stability":                   floatOr("set_stability"), "set_war_support": floatOr("set_war_support"),
			"add_to_faction_tags": factionTags, "source": "imported",
		}
		for _, column := range []string{"set_popularities_democratic", "set_popularities_communism", "set_popularities_fascism", "set_popularities_neutrality", "set_research_slots", "set_convoys"} {
			row[column] = p.intOr(column, 0)
		}
		for _, column := range []string{"set_politics_ruling_party", "set_politics_last_election", "add_to_faction", "set_cosmetic_tag", "set_oob", "set_air_oob", "set_naval_oob", "raw_script"} {
			row[column] = p.str(column)
		}
		for _, column := range []string{"add_ideas_extra", "remove_ideas", "complete_national_focus_ids", "set_technology_ids", "startup_flags", "give_guarantee_tags", "set_autonomy", "diplomatic_relation", "date_blocks"} {
			row[column] = p.jsonOr(column, []any{})
		}
		write(history, row, historyUpdate)
	}

	states := newKeyedTable(tx, "states", "state_id")
	stateUpdate := []string{"name", "manpower", "state_category", "resources", "provinces", "owner", "controller", "add_core_of", "add_claim_by", "remove_core_of", "victory_points", "set_province_controller", "buildings", "local_supplies", "buildings_max_level_factor", "impassable", "impassable_ignored_links", "force_link_ownership_to", "raw_extras", "source_file", "source_snapshot", "source"}
	for _, r := range rows {
		if r.kind != "state" {
			continue
		}
		p := r.data
		stateID, ok := p.int("state_id")
		if !ok {

			return fmt.Errorf("merge state %s: no state_id", r.key)
		}
		var sourcePath any
		if r.source != "" {
			sourcePath = r.source
		}
		row := project.Row{
			"state_id": stateID, "name": p.str("name"), "display_name_locs": map[string]any{},
			"manpower": p.intOr("manpower", 0), "state_category": p.str("state_category"),
			"resources": p.jsonOr("resources", map[string]any{}), "provinces": p.jsonOr("provinces", []any{}),
			"owner": p.str("owner"), "controller": worldNullIfEmpty(p, "controller"),
			"buildings":      p.jsonOr("buildings", map[string]any{}),
			"local_supplies": p.nullableFloat("local_supplies"), "buildings_max_level_factor": p.nullableFloat("buildings_max_level_factor"),
			"impassable": worldNullableBool(p, "impassable"), "impassable_ignored_links": worldNullIfEmpty(p, "impassable_ignored_links"),
			"force_link_ownership_to": p.nullableInt("force_link_ownership_to"),
			"raw_extras":              p.jsonOr("raw_extras", map[string]any{}), "source_file": r.source,
			"source_snapshot": map[string]any{
				"schema_version": "hagane.state-source-snapshot.v1", "state_id": stateID,
				"province_ids": p.jsonOr("provinces", []any{}), "owner": worldNullIfEmpty(p, "owner"),
				"controller": worldNullIfEmpty(p, "controller"), "source_path": sourcePath,
				"source_line": nil, "source_content_sha256": nil,
			},
			"source": "imported",
		}
		for _, column := range []string{"add_core_of", "add_claim_by", "remove_core_of", "victory_points", "set_province_controller"} {
			row[column] = p.jsonOr(column, []any{})
		}
		write(states, row, stateUpdate)
	}
	return nil
}
