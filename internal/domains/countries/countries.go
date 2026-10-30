package countries

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	const base = "/api/go/projects/{id}"
	mux.HandleFunc("PUT "+base+"/countries/{cid}", h.handle(updateCountryHandler))
	mux.HandleFunc("DELETE "+base+"/countries/{cid}", h.handle(deleteCountryHandler))
	mux.HandleFunc("GET "+base+"/countries/{cid}/delete-preview", h.handle(deletePreviewHandler))
	mux.HandleFunc("POST "+base+"/countries/{cid}/upload-flag", h.handle(uploadFlagHandler))
	mux.HandleFunc("GET "+base+"/countries/{cid}/flag", h.handle(readFlagHandler))
	mux.HandleFunc("POST "+base+"/flip-all", h.handle(flipAllHandler))
	mux.HandleFunc("GET "+base+"/base-mod-vocabulary", h.handle(vocabularyHandler))

	mux.HandleFunc("GET "+base+"/cover", h.handle(readCoverHandler))
	mux.HandleFunc("POST "+base+"/cover", h.handle(uploadCoverHandler))
	mux.HandleFunc("DELETE "+base+"/cover", h.handle(deleteCoverHandler))

	mux.HandleFunc("GET "+base+"/cosmetic-tags", h.handle(listCosmeticHandler))
	mux.HandleFunc("POST "+base+"/cosmetic-tags", h.handle(createCosmeticHandler))
	mux.HandleFunc("GET "+base+"/cosmetic-tags/{tag}", h.handle(getCosmeticHandler))
	mux.HandleFunc("PATCH "+base+"/cosmetic-tags/{tag}", h.handle(patchCosmeticHandler))
	mux.HandleFunc("POST "+base+"/cosmetic-tags/{tag}/resolve", h.handle(resolveCosmeticHandler))

	mux.HandleFunc("GET "+base+"/relations/graph", h.handle(relationGraphHandler))
	mux.HandleFunc("GET "+base+"/relations/entity/{kind}/{sid}", h.handle(relationEntityHandler))
	mux.HandleFunc("GET "+base+"/relations/country/{tag}", h.handle(relationCountryHandler))

	mux.HandleFunc("GET "+base+"/politics/references", h.handle(politicsReferencesHandler))
	mux.HandleFunc("POST "+base+"/politics/roots", h.handle(politicsCreateRootHandler))
	mux.HandleFunc("POST "+base+"/politics/subtypes", h.handle(politicsCreateSubtypeHandler))
	mux.HandleFunc("PATCH "+base+"/politics/definitions/{kind}/{script}", h.handle(politicsUpdateHandler))
	mux.HandleFunc("POST "+base+"/politics/delete-preview", h.handle(politicsDeletePreviewHandler))
	mux.HandleFunc("POST "+base+"/politics/delete-apply", h.handle(politicsDeleteApplyHandler))
	mux.HandleFunc("DELETE "+base+"/politics/conflicts/{script}", h.handle(politicsDiscardConflictHandler))

	mux.HandleFunc("GET "+base+"/ideology-icons/{group}", h.handle(readIdeologyIconHandler))
	mux.HandleFunc("POST "+base+"/ideology-icons/{group}/upload", h.handle(uploadIdeologyIconHandler))
}

type handlers struct {
	deps domains.Deps
}

type handlerFunc func(p *project.Project, w http.ResponseWriter, r *http.Request) error

func (h *handlers) handle(fn handlerFunc) http.HandlerFunc {
	return apiutil.Handler(h.deps.Open, fn)
}

func detail(status int, message string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": message}}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

var (
	errCountryMissing = detail(http.StatusNotFound, workbench.ErrCountryMissing.Error())
	errLastCountry    = detail(http.StatusBadRequest, "Cannot delete the last country from a project")
)

func countryID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("cid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errCountryMissing
	}
	return id, nil
}

type countryPatch struct {
	Tag                workbench.Optional[string]                       `json:"tag"`
	Name               workbench.Optional[string]                       `json:"name"`
	Color              workbench.Optional[[]int64]                      `json:"color"`
	ColorUI            workbench.Optional[[]int64]                      `json:"color_ui"`
	Ideology           workbench.Optional[string]                       `json:"ideology"`
	GraphicalCulture   workbench.Optional[string]                       `json:"graphical_culture"`
	GraphicalCulture2D workbench.Optional[string]                       `json:"graphical_culture_2d"`
	UseLegacyAIPPSpend workbench.Optional[bool]                         `json:"use_legacy_ai_pp_spend"`
	NameEn             workbench.Optional[string]                       `json:"name_en"`
	NameZh             workbench.Optional[string]                       `json:"name_zh"`
	AdjectiveEn        workbench.Optional[string]                       `json:"adjective_en"`
	AdjectiveZh        workbench.Optional[string]                       `json:"adjective_zh"`
	FlagImagePath      workbench.Optional[string]                       `json:"flag_image_path"`
	NameDemocraticEn   workbench.Optional[string]                       `json:"name_democratic_en"`
	NameDemocraticZh   workbench.Optional[string]                       `json:"name_democratic_zh"`
	NameFascismEn      workbench.Optional[string]                       `json:"name_fascism_en"`
	NameFascismZh      workbench.Optional[string]                       `json:"name_fascism_zh"`
	NameCommunismEn    workbench.Optional[string]                       `json:"name_communism_en"`
	NameCommunismZh    workbench.Optional[string]                       `json:"name_communism_zh"`
	NameNeutralityEn   workbench.Optional[string]                       `json:"name_neutrality_en"`
	NameNeutralityZh   workbench.Optional[string]                       `json:"name_neutrality_zh"`
	DisplayNameLocs    workbench.Optional[map[string]string]            `json:"display_name_locs"`
	AdjectiveLocs      workbench.Optional[map[string]string]            `json:"adjective_locs"`
	IdeologyNameLocs   workbench.Optional[map[string]map[string]string] `json:"ideology_name_locs"`
}

func (p *countryPatch) touchesLocs() bool {
	return p.AdjectiveLocs.Set || p.AdjectiveEn.Set || p.AdjectiveZh.Set ||
		p.IdeologyNameLocs.Set ||
		p.NameDemocraticEn.Set || p.NameDemocraticZh.Set ||
		p.NameFascismEn.Set || p.NameFascismZh.Set ||
		p.NameCommunismEn.Set || p.NameCommunismZh.Set ||
		p.NameNeutralityEn.Set || p.NameNeutralityZh.Set
}

func setString(o *workbench.Optional[string], v string) {
	*o = workbench.Optional[string]{Set: true, Value: &v}
}

func (p *countryPatch) normalizeLocs(cur project.Row) {
	pick := func(o workbench.Optional[string], column string) string {
		if o.Set && o.Value != nil {
			return *o.Value
		}
		return str(cur[column])
	}
	locs := &workbench.CountryLocsFields{
		AdjectiveEn: pick(p.AdjectiveEn, "adjective_en"),
		AdjectiveZh: pick(p.AdjectiveZh, "adjective_zh"),
		IdeologyScalars: map[string]*[2]string{
			"democratic": {pick(p.NameDemocraticEn, "name_democratic_en"), pick(p.NameDemocraticZh, "name_democratic_zh")},
			"fascism":    {pick(p.NameFascismEn, "name_fascism_en"), pick(p.NameFascismZh, "name_fascism_zh")},
			"communism":  {pick(p.NameCommunismEn, "name_communism_en"), pick(p.NameCommunismZh, "name_communism_zh")},
			"neutrality": {pick(p.NameNeutralityEn, "name_neutrality_en"), pick(p.NameNeutralityZh, "name_neutrality_zh")},
		},
	}
	if p.AdjectiveLocs.Set && p.AdjectiveLocs.Value != nil {
		locs.AdjectiveLocs = *p.AdjectiveLocs.Value
	} else {
		locs.AdjectiveLocs = stringMapOf(cur["adjective_locs"])
	}
	if p.IdeologyNameLocs.Set && p.IdeologyNameLocs.Value != nil {
		locs.IdeologyNameLocs = *p.IdeologyNameLocs.Value
	} else {
		locs.IdeologyNameLocs = nestedStringMapOf(cur["ideology_name_locs"])
	}
	workbench.NormalizeCountryLocs(locs)
	setString(&p.AdjectiveEn, locs.AdjectiveEn)
	setString(&p.AdjectiveZh, locs.AdjectiveZh)
	p.AdjectiveLocs = workbench.Optional[map[string]string]{Set: true, Value: &locs.AdjectiveLocs}
	p.IdeologyNameLocs = workbench.Optional[map[string]map[string]string]{Set: true, Value: &locs.IdeologyNameLocs}
	setString(&p.NameDemocraticEn, locs.IdeologyScalars["democratic"][0])
	setString(&p.NameDemocraticZh, locs.IdeologyScalars["democratic"][1])
	setString(&p.NameFascismEn, locs.IdeologyScalars["fascism"][0])
	setString(&p.NameFascismZh, locs.IdeologyScalars["fascism"][1])
	setString(&p.NameCommunismEn, locs.IdeologyScalars["communism"][0])
	setString(&p.NameCommunismZh, locs.IdeologyScalars["communism"][1])
	setString(&p.NameNeutralityEn, locs.IdeologyScalars["neutrality"][0])
	setString(&p.NameNeutralityZh, locs.IdeologyScalars["neutrality"][1])
}

func (p *countryPatch) apply(row project.Row) bool {
	changed := false
	text := func(column string, o workbench.Optional[string]) {
		if o.Set {
			changed = true
			if o.Value == nil {
				row[column] = nil
			} else {
				row[column] = *o.Value
			}
		}
	}
	ints := func(column string, o workbench.Optional[[]int64]) {
		if o.Set {
			changed = true
			if o.Value == nil {
				row[column] = nil
			} else {
				list := make([]any, len(*o.Value))
				for i, v := range *o.Value {
					list[i] = v
				}
				row[column] = list
			}
		}
	}
	locs := func(column string, o workbench.Optional[map[string]string]) {
		if o.Set {
			changed = true
			if o.Value == nil {
				row[column] = nil
			} else {
				row[column] = anyMap(*o.Value)
			}
		}
	}
	text("tag", p.Tag)
	text("name", p.Name)
	ints("color", p.Color)
	ints("color_ui", p.ColorUI)
	text("ideology", p.Ideology)
	text("graphical_culture", p.GraphicalCulture)
	text("graphical_culture_2d", p.GraphicalCulture2D)
	if p.UseLegacyAIPPSpend.Set {
		changed = true
		if p.UseLegacyAIPPSpend.Value == nil {
			row["use_legacy_ai_pp_spend"] = nil
		} else {
			row["use_legacy_ai_pp_spend"] = *p.UseLegacyAIPPSpend.Value
		}
	}
	text("name_en", p.NameEn)
	text("name_zh", p.NameZh)
	text("adjective_en", p.AdjectiveEn)
	text("adjective_zh", p.AdjectiveZh)
	text("flag_image_path", p.FlagImagePath)
	text("name_democratic_en", p.NameDemocraticEn)
	text("name_democratic_zh", p.NameDemocraticZh)
	text("name_fascism_en", p.NameFascismEn)
	text("name_fascism_zh", p.NameFascismZh)
	text("name_communism_en", p.NameCommunismEn)
	text("name_communism_zh", p.NameCommunismZh)
	text("name_neutrality_en", p.NameNeutralityEn)
	text("name_neutrality_zh", p.NameNeutralityZh)
	locs("display_name_locs", p.DisplayNameLocs)
	locs("adjective_locs", p.AdjectiveLocs)
	if p.IdeologyNameLocs.Set {
		changed = true
		if p.IdeologyNameLocs.Value == nil {
			row["ideology_name_locs"] = nil
		} else {
			nested := map[string]any{}
			for group, values := range *p.IdeologyNameLocs.Value {
				nested[group] = anyMap(values)
			}
			row["ideology_name_locs"] = nested
		}
	}
	return changed
}

func updateCountryHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, err := countryID(r)
	if err != nil {
		return err
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	var patch countryPatch
	if err := decodeJSON(r, &patch, false); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid country payload")
	}
	item, err := updateCountry(p, cid, patch, rv)
	if err != nil {
		return err
	}
	rv.header(w)
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func updateCountry(p *project.Project, cid int64, patch countryPatch, rv *revisions) (project.Row, error) {
	err := p.Write(renameTables(), func(tx *project.Tx) error {
		cur := tx.Find("countries", cid)
		if cur == nil {
			return errCountryMissing
		}
		if patch.touchesLocs() {
			patch.normalizeLocs(cur)
		}
		oldTag := str(cur["tag"])
		newTag := oldTag
		if patch.Tag.Set && patch.Tag.Value != nil {
			newTag = *patch.Tag.Value
		}
		tagChanging := newTag != oldTag
		if tagChanging && !workbench.CountryTagRE.MatchString(newTag) {

			return apiutil.ValidationError(workbench.CountryTagPatternMessage)
		}
		if tagChanging {
			for _, row := range tx.Rows("countries") {
				if str(row["tag"]) == newTag && project.Int64(row["id"]) != cid {
					return apiutil.ConflictError(workbench.ErrCountryConflict{Tag: newTag}.Error())
				}
			}
		}
		next := project.Row{}
		if patch.apply(next) {

			for _, column := range []string{"tag", "name", "display_name_locs", "adjective_locs", "ideology_name_locs"} {
				if value, set := next[column]; set && value == nil {
					return detail(http.StatusServiceUnavailable, "country store unavailable")
				}
			}
			if _, err := tx.Update("countries", cid, func(row project.Row) {
				for k, v := range next {
					row[k] = v
				}
			}); err != nil {
				return err
			}
		}
		if tagChanging {
			if err := renameCountryTagCascade(tx, rv, oldTag, newTag); err != nil {
				return err
			}
			if str(tx.Record()["country_tag"]) == oldTag {
				tx.SetRecord(func(record project.Row) { record["country_tag"] = newTag })
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := workbench.SyncProjectCountryTags(p); err != nil {
		return nil, err
	}
	rv.commit(p)
	return workbench.GetCountry(p, cid)
}

func deletePreviewHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, err := countryID(r)
	if err != nil {
		return err
	}
	report, err := countryDeletePreview(p, cid)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, report)
	return nil
}

func countryDeletePreview(p *project.Project, cid int64) (map[string]any, error) {
	countries, err := p.Rows("countries")
	if err != nil {
		return nil, err
	}
	var country project.Row
	for _, row := range countries {
		if project.Int64(row["id"]) == cid {
			country = row
		}
	}
	if country == nil {
		return nil, errCountryMissing
	}
	tag := str(country["tag"])
	counts := make(workbench.OrderedCounts, 0, len(countryTagTables))
	var sum int64
	for _, item := range countryTagTables {
		rows, err := p.Rows(item.Table)
		if err != nil {
			return nil, err
		}
		var n int64
		for _, row := range rows {
			if str(row["country_tag"]) == tag {
				n++
			}
		}
		counts = append(counts, struct {
			Label string
			N     int64
		}{item.Label, n})
		sum += n
	}
	states, err := p.Rows("states")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"country_id": project.Int64(country["id"]), "tag": tag, "name": str(country["name"]),
		"will_be_last_country": len(countries) <= 1,
		"cascade_counts":       counts, "total": sum,

		"owned_states": ownedStates(states, tag), "vanilla_tag": workbench.VanillaCountryTag(tag),
	}, nil
}

func ownedStates(states []project.Row, tag string) int64 {
	var n int64
	for _, row := range states {
		if strings.EqualFold(str(row["owner"]), tag) {
			n++
		}
	}
	return n
}

func deleteCountryHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	cid, err := countryID(r)
	if err != nil {
		return err
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	if err := deleteCountry(p, cid, rv); err != nil {
		return err
	}
	rv.commit(p)
	rv.header(w)
	apiutil.NoContent(w)
	return nil
}

func deleteCountry(p *project.Project, cid int64, rv *revisions) error {
	tables := []string{"countries", "states", "focus_nodes", "bookmarks"}
	for _, t := range countryTagTables {
		tables = append(tables, t.Table)
	}
	err := p.Write(tables, func(tx *project.Tx) error {
		country := tx.Find("countries", cid)
		if country == nil {
			return errCountryMissing
		}
		tag := str(country["tag"])
		if len(tx.Rows("countries")) <= 1 {
			return errLastCountry
		}
		if !workbench.VanillaCountryTag(tag) {
			if owned := ownedStates(tx.Rows("states"), tag); owned > 0 {
				return &apiutil.StatusError{Status: http.StatusConflict, Body: map[string]any{
					"detail":       fmt.Sprintf("Country %s still owns %d states in this project; give them to another country first", tag, owned),
					"code":         "country_owns_states",
					"owned_states": owned,
				}}
			}
		}
		for _, row := range tx.Rows("focus_trees") {
			if str(row["country_tag"]) == tag {
				if err := rv.advance(tx, "focus_trees", project.Int64(row["id"])); err != nil {
					return err
				}
			}
		}
		ofTag := func(row project.Row) bool { return str(row["country_tag"]) == tag }
		for _, t := range countryTagTables {
			removed := idsOf(tx.Rows(t.Table), ofTag)
			tx.Delete(t.Table, ofTag)

			switch t.Table {
			case "focus_trees":
				tx.Delete("focus_nodes", func(row project.Row) bool { return removed[project.Int64(row["focus_tree_id"])] })
			case "event_namespaces":
				tx.Delete("events", func(row project.Row) bool { return removed[project.Int64(row["namespace_id"])] })
			case "decision_categories":
				tx.Delete("decisions", func(row project.Row) bool { return removed[project.Int64(row["category_id"])] })
			}
		}
		tx.Delete("countries", func(row project.Row) bool { return project.Int64(row["id"]) == cid })
		return nil
	})
	if err != nil {
		return err
	}
	return workbench.SyncProjectCountryTags(p)
}

func idsOf(rows []project.Row, match func(project.Row) bool) map[int64]bool {
	out := map[int64]bool{}
	for _, row := range rows {
		if match(row) {
			out[project.Int64(row["id"])] = true
		}
	}
	return out
}

var flipWorkbenches = []struct{ Label, Table string }{
	{"characters", "characters"},
	{"technologies", "technologies"},
	{"ideas", "ideas"},
	{"focus_trees", "focus_trees"},
	{"events", "events"},
	{"decisions", "decisions"},
	{"states", "states"},
	{"dynamic_modifiers", "dynamic_modifiers"},
	{"military_industrial_organizations", "military_industrial_organizations"},
	{"intelligence_agencies", "intelligence_agencies"},
	{"bookmarks", "bookmarks"},
	{"super_events", "super_events"},
	{"history_files", "history_files"},
	{"doctrines", "doctrines"},
	{"scripted_snippets", "scripted_snippets"},
	{"scripted_localisations", "scripted_localisations"},
	{"equipment", "equipments"},
	{"sub_units", "sub_units"},
	{"division_templates", "division_templates"},
	{"division_deployments", "division_deployments"},
	{"raw_files", "project_raw_files"},
	{"autonomy_states", "autonomy_states"},
	{"opinion_modifiers", "opinion_modifiers"},
	{"modifier_definitions", "modifier_definitions"},
	{"ai_strategy_plans", "ai_strategy_plans"},
	{"ai_templates", "ai_templates"},
	{"wargoals", "wargoal_types"},
	{"special_projects", "special_projects"},
	{"special_project_specializations", "special_project_specializations"},
	{"ui_panels", "ui_panels"},
	{"oob_air_wing_groups", "oob_air_wing_groups"},
	{"oob_air_wings", "oob_air_wings"},
	{"oob_fleets", "oob_fleets"},
	{"oob_task_forces", "oob_task_forces"},
	{"oob_ships", "oob_ships"},
}

func flipAllHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var body struct {
		DryRun bool `json:"dry_run"`
	}
	if err := decodeJSON(r, &body, true); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid flip-all payload")
	}
	rv, err := readRevisions(r)
	if err != nil {
		return err
	}
	counts, err := flipAll(p, body.DryRun, rv)
	if err != nil {
		return err
	}
	rv.commit(p)
	rv.header(w)
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"project_id": p.ID(), "country_tag": str(p.Record()["country_tag"]),
		"flipped": counts, "total": counts.Total(), "dry_run": body.DryRun,
	})
	return nil
}

func flipAll(p *project.Project, dryRun bool, rv *revisions) (workbench.OrderedCounts, error) {
	tables := make([]string, len(flipWorkbenches))
	for i, wb := range flipWorkbenches {
		tables[i] = wb.Table
	}
	counts := make(workbench.OrderedCounts, 0, len(flipWorkbenches))
	err := p.Write(tables, func(tx *project.Tx) error {
		for _, wb := range flipWorkbenches {
			imported := func(row project.Row) bool { return str(row["source"]) == "imported" }
			var n int64
			for _, row := range tx.Rows(wb.Table) {
				if imported(row) {
					n++
				}
			}
			counts = append(counts, struct {
				Label string
				N     int64
			}{wb.Label, n})
			if n == 0 || dryRun {
				continue
			}
			if wb.Table == "bookmarks" || wb.Table == "focus_trees" {
				for id := range idsOf(tx.Rows(wb.Table), imported) {
					if err := rv.advance(tx, wb.Table, id); err != nil {
						return err
					}
				}
			}
			rewriteRows(tx, wb.Table, func(row project.Row) project.Row {
				if !imported(row) {
					return nil
				}
				next := cloneShallow(row)
				next["source"] = "created"
				return next
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func vocabularyHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	distinct := func(table, column string) ([]string, error) {
		rows, err := p.Rows(table)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		out := []string{}
		for _, row := range rows {
			v := str(row[column])
			if str(row["source"]) != "imported" || v == "" || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
		sortStrings(out)
		return out, nil
	}
	slots, err := distinct("characters", "advisor_slot")
	if err != nil {
		return err
	}
	types, err := distinct("ideas", "idea_type")
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"advisor_slots": slots, "idea_types": types})
	return nil
}
