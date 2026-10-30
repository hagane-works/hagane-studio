package content

import (
	"fmt"
	"net/http"
	"strings"

	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	superEventsTable = "super_events"
	superEventsGone  = "super event store unavailable"
)

type superEvent struct {
	ID                int64             `json:"id"`
	ProjectID         int64             `json:"project_id"`
	Number            int               `json:"number"`
	CountryTag        string            `json:"country_tag"`
	TitleEn           string            `json:"title_en"`
	TitleZh           string            `json:"title_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	QuoteEn           string            `json:"quote_en"`
	QuoteZh           string            `json:"quote_zh"`
	BtnEn             string            `json:"btn_en"`
	BtnZh             string            `json:"btn_zh"`
	PicturePath       string            `json:"picture_path"`
	SongToken         string            `json:"song_token"`
	AutoFireOnStartup bool              `json:"auto_fire_on_startup"`
	AutoFireAfterDays int               `json:"auto_fire_after_days"`
	Options           []any             `json:"options"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	QuoteLocs         map[string]string `json:"quote_locs"`
	ButtonLocs        map[string]string `json:"button_locs"`
	Source            string            `json:"source"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

func superEventFromRow(row project.Row) superEvent {
	item := superEvent{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		Number: intOr(row, "number", 0), CountryTag: text(row, "country_tag"),
		TitleEn: text(row, "title_en"), TitleZh: text(row, "title_zh"),
		DescEn: text(row, "desc_en"), DescZh: text(row, "desc_zh"),
		QuoteEn: text(row, "quote_en"), QuoteZh: text(row, "quote_zh"),
		BtnEn: text(row, "btn_en"), BtnZh: text(row, "btn_zh"),
		PicturePath: text(row, "picture_path"), SongToken: text(row, "song_token"),
		AutoFireOnStartup: boolOr(row, "auto_fire_on_startup", false),
		AutoFireAfterDays: intOr(row, "auto_fire_after_days", 0),
		DisplayNameLocs:   stringMap(row["display_name_locs"]),
		DescriptionLocs:   stringMap(row["description_locs"]),
		QuoteLocs:         stringMap(row["quote_locs"]),
		ButtonLocs:        stringMap(row["button_locs"]),
		Source:            textOr(row, "source", "created"),
		CreatedAt:         isoNaive(row["created_at"]), UpdatedAt: isoNaive(row["updated_at"]),
	}
	decode(row["options"], &item.Options)
	if item.Options == nil {
		item.Options = []any{}
	}
	return item
}

func cleanLocs(value map[string]string) map[string]string {
	out := map[string]string{}
	for lang, raw := range value {
		if text := strings.TrimSpace(raw); text != "" {
			out[lang] = text
		}
	}
	return out
}

func mergeLegacyLocs(value map[string]string, en, zh string) map[string]string {
	locs := cleanLocs(value)
	for lang, raw := range map[string]string{"en": en, "zh": zh} {
		if text := strings.TrimSpace(raw); text != "" && locs[lang] == "" {
			locs[lang] = text
		}
	}
	return locs
}

func cleanLocsAny(v any) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]any); ok {
		for lang, raw := range m {
			if s, ok := raw.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					out[lang] = t
				}
			}
		}
	}
	return out
}

func mergeLegacyLocsAny(v, en, zh any) map[string]string {
	locs := cleanLocsAny(v)
	for lang, raw := range map[string]any{"en": en, "zh": zh} {
		s, _ := raw.(string)
		if t := strings.TrimSpace(s); t != "" && locs[lang] == "" {
			locs[lang] = t
		}
	}
	return locs
}

func locsToAny(m map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func normalizeEventOptions(options []any) []any {
	normalized := make([]any, 0, len(options))
	for _, raw := range options {
		m, ok := raw.(map[string]any)
		if !ok {
			normalized = append(normalized, raw)
			continue
		}
		opt := map[string]any{}
		for k, v := range m {
			opt[k] = v
		}
		names := mergeLegacyLocsAny(opt["display_name_locs"], opt["name_en"], opt["name_zh"])
		displayText := ""
		if s, ok := opt["display_text"].(string); ok {
			displayText = strings.TrimSpace(s)
		}
		if displayText != "" && names["en"] == "" {
			names["en"] = displayText
		}
		if len(names) > 0 {
			opt["display_name_locs"] = locsToAny(names)
			if _, present := opt["name_en"]; !present {
				opt["name_en"] = names["en"]
			}
			if _, present := opt["name_zh"]; !present {
				opt["name_zh"] = names["zh"]
			}
		}
		tooltips := mergeLegacyLocsAny(opt["effect_tooltip_locs"], opt["effect_tooltip_en"], opt["effect_tooltip_zh"])
		if len(tooltips) > 0 {
			opt["effect_tooltip_locs"] = locsToAny(tooltips)
			if _, present := opt["effect_tooltip_en"]; !present {
				opt["effect_tooltip_en"] = tooltips["en"]
			}
			if _, present := opt["effect_tooltip_zh"]; !present {
				opt["effect_tooltip_zh"] = tooltips["zh"]
			}
		}
		normalized = append(normalized, opt)
	}
	return normalized
}

type superEventLocs struct {
	QuoteEn, QuoteZh, BtnEn, BtnZh string
	QuoteLocs, ButtonLocs          map[string]string
}

func (f *superEventLocs) normalize() {
	quote := mergeLegacyLocs(f.QuoteLocs, f.QuoteEn, f.QuoteZh)
	f.QuoteLocs = quote
	if len(quote) > 0 {
		if strings.TrimSpace(f.QuoteEn) == "" {
			f.QuoteEn = quote["en"]
		}
		if strings.TrimSpace(f.QuoteZh) == "" {
			f.QuoteZh = quote["zh"]
		}
	}
	button := mergeLegacyLocs(f.ButtonLocs, f.BtnEn, f.BtnZh)
	f.ButtonLocs = button
	if len(button) > 0 {
		if strings.TrimSpace(f.BtnEn) == "" {
			f.BtnEn = button["en"]
		}
		if strings.TrimSpace(f.BtnZh) == "" {
			f.BtnZh = button["zh"]
		}
	}
}

type superEventCreate struct {
	Number            *int              `json:"number"`
	CountryTag        string            `json:"country_tag"`
	TitleEn           string            `json:"title_en"`
	TitleZh           string            `json:"title_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	QuoteEn           string            `json:"quote_en"`
	QuoteZh           string            `json:"quote_zh"`
	BtnEn             string            `json:"btn_en"`
	BtnZh             string            `json:"btn_zh"`
	PicturePath       string            `json:"picture_path"`
	SongToken         string            `json:"song_token"`
	AutoFireOnStartup bool              `json:"auto_fire_on_startup"`
	AutoFireAfterDays int               `json:"auto_fire_after_days"`
	Options           []any             `json:"options"`
	DisplayNameLocs   map[string]string `json:"display_name_locs"`
	DescriptionLocs   map[string]string `json:"description_locs"`
	QuoteLocs         map[string]string `json:"quote_locs"`
	ButtonLocs        map[string]string `json:"button_locs"`
	Source            *string           `json:"source"`
}

type superEventPatch struct {
	Number            workbench.Optional[int]               `json:"number"`
	CountryTag        workbench.Optional[string]            `json:"country_tag"`
	TitleEn           workbench.Optional[string]            `json:"title_en"`
	TitleZh           workbench.Optional[string]            `json:"title_zh"`
	DescEn            workbench.Optional[string]            `json:"desc_en"`
	DescZh            workbench.Optional[string]            `json:"desc_zh"`
	QuoteEn           workbench.Optional[string]            `json:"quote_en"`
	QuoteZh           workbench.Optional[string]            `json:"quote_zh"`
	BtnEn             workbench.Optional[string]            `json:"btn_en"`
	BtnZh             workbench.Optional[string]            `json:"btn_zh"`
	PicturePath       workbench.Optional[string]            `json:"picture_path"`
	SongToken         workbench.Optional[string]            `json:"song_token"`
	AutoFireOnStartup workbench.Optional[bool]              `json:"auto_fire_on_startup"`
	AutoFireAfterDays workbench.Optional[int]               `json:"auto_fire_after_days"`
	Options           workbench.Optional[[]any]             `json:"options"`
	DisplayNameLocs   workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs   workbench.Optional[map[string]string] `json:"description_locs"`
	QuoteLocs         workbench.Optional[map[string]string] `json:"quote_locs"`
	ButtonLocs        workbench.Optional[map[string]string] `json:"button_locs"`
}

func sameSuperEventSlot(row project.Row, countryTag any, number any) bool {
	tag, ok := countryTag.(string)
	if !ok || row["country_tag"] == nil {
		return false
	}
	return text(row, "country_tag") == tag && project.Int64(row["number"]) == project.Int64(number)
}

func (h *handlers) superEvents(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	if h.assetRoute(p, w, r, rest, "upload-picture", "picture", superEventPictureSpec, func(w http.ResponseWriter, id int64, _ assetSpec) error {
		row, err := getRow(p, superEventsTable, id, "SuperEvent not found")
		if err != nil {
			return statusErr(http.StatusServiceUnavailable, superEventsGone)
		}
		writeJSON(w, http.StatusOK, superEventFromRow(row))
		return nil
	}) {
		return
	}
	if len(rest) > 1 {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	const missing = "SuperEvent not found"
	crud{
		listKey: "super_events", missing: missing, unavailable: superEventsGone, bothKeys: true,
		list: func(p *project.Project, _ *http.Request) (any, int, error) {
			items, err := listRows(p, superEventsTable, func(a, b project.Row) bool {
				return project.Int64(a["number"]) < project.Int64(b["number"])
			}, superEventFromRow)
			return items, len(items), err
		},
		create: createSuperEvent,
		get: func(p *project.Project, id int64) (any, error) {
			row, err := getRow(p, superEventsTable, id, missing)
			if err != nil {
				return nil, err
			}
			return superEventFromRow(row), nil
		},
		update: updateSuperEvent,
		remove: removeRow(superEventsTable, missing),
	}.serve(p, w, r, rest)
}

func superEventPictureSpec(p *project.Project, id int64) (assetSpec, error) {
	rows, err := p.Rows(superEventsTable)
	if err != nil {
		return assetSpec{}, err
	}
	row := findRow(rows, id)
	if row == nil {
		return assetSpec{}, errAssetTargetMissing
	}
	filename := fmt.Sprintf("SE_%04d.png", intOr(row, "number", 0))
	return assetSpec{
		table: superEventsTable, entityType: "super_event_picture", obsKind: "super_event",
		filename: filename,
		apply: func(row project.Row) {
			row["picture_path"] = "super_events/" + filename
			row["has_obs"] = true
		},
	}, nil
}

func createSuperEvent(p *project.Project, r *http.Request) (any, error) {
	var in superEventCreate
	if err := decodeBody(r, &in); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid super event payload")
	}
	if in.Number == nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "number is required")
	}
	if in.Options == nil {
		in.Options = []any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	locs := &superEventLocs{
		QuoteEn: in.QuoteEn, QuoteZh: in.QuoteZh, BtnEn: in.BtnEn, BtnZh: in.BtnZh,
		QuoteLocs: in.QuoteLocs, ButtonLocs: in.ButtonLocs,
	}
	locs.normalize()
	row, err := writeRow(p, superEventsTable, func(tx *project.Tx) (project.Row, error) {
		if taken(tx.Rows(superEventsTable), 0, func(row project.Row) bool {
			return sameSuperEventSlot(row, in.CountryTag, int64(*in.Number))
		}) {
			return nil, statusErr(http.StatusConflict, fmt.Sprintf(
				"superevent number %d already exists for this country in the project. Pick a unique 4-digit number.", *in.Number))
		}
		return insert(tx, superEventsTable, project.Row{
			"number": int64(*in.Number), "country_tag": in.CountryTag,
			"title_en": in.TitleEn, "title_zh": in.TitleZh, "desc_en": in.DescEn, "desc_zh": in.DescZh,
			"quote_en": locs.QuoteEn, "quote_zh": locs.QuoteZh, "btn_en": locs.BtnEn, "btn_zh": locs.BtnZh,
			"picture_path": in.PicturePath, "song_token": in.SongToken,
			"auto_fire_on_startup": in.AutoFireOnStartup, "auto_fire_after_days": int64(in.AutoFireAfterDays),
			"options":           jsonValue(normalizeEventOptions(in.Options)),
			"display_name_locs": jsonValue(in.DisplayNameLocs),
			"description_locs":  jsonValue(in.DescriptionLocs),
			"quote_locs":        jsonValue(locs.QuoteLocs), "button_locs": jsonValue(locs.ButtonLocs),
			"source": workbench.StrDefault(in.Source, "created"), "has_obs": false,
		})
	})
	if err != nil {
		return nil, err
	}
	return superEventFromRow(row), nil
}

func updateSuperEvent(p *project.Project, r *http.Request, id int64) (any, error) {
	var patch superEventPatch
	if err := decodeBody(r, &patch); err != nil {
		return nil, statusErr(http.StatusUnprocessableEntity, "Invalid super event payload")
	}
	curRow, err := getRow(p, superEventsTable, id, "SuperEvent not found")
	if err != nil {
		return nil, err
	}
	cur := superEventFromRow(curRow)
	if patch.QuoteLocs.Set || patch.QuoteEn.Set || patch.QuoteZh.Set ||
		patch.ButtonLocs.Set || patch.BtnEn.Set || patch.BtnZh.Set || patch.Options.Set {
		pick := func(o workbench.Optional[string], current string) string {
			if o.Set && o.Value != nil {
				return *o.Value
			}
			return current
		}
		locs := &superEventLocs{
			QuoteEn: pick(patch.QuoteEn, cur.QuoteEn), QuoteZh: pick(patch.QuoteZh, cur.QuoteZh),
			BtnEn: pick(patch.BtnEn, cur.BtnEn), BtnZh: pick(patch.BtnZh, cur.BtnZh),
			QuoteLocs: cur.QuoteLocs, ButtonLocs: cur.ButtonLocs,
		}
		if patch.QuoteLocs.Set && patch.QuoteLocs.Value != nil {
			locs.QuoteLocs = *patch.QuoteLocs.Value
		}
		if patch.ButtonLocs.Set && patch.ButtonLocs.Value != nil {
			locs.ButtonLocs = *patch.ButtonLocs.Value
		}
		locs.normalize()
		set := func(o *workbench.Optional[string], v string) { *o = workbench.Optional[string]{Set: true, Value: &v} }
		set(&patch.QuoteEn, locs.QuoteEn)
		set(&patch.QuoteZh, locs.QuoteZh)
		set(&patch.BtnEn, locs.BtnEn)
		set(&patch.BtnZh, locs.BtnZh)
		patch.QuoteLocs = workbench.Optional[map[string]string]{Set: true, Value: &locs.QuoteLocs}
		patch.ButtonLocs = workbench.Optional[map[string]string]{Set: true, Value: &locs.ButtonLocs}
		options := cur.Options
		if patch.Options.Set && patch.Options.Value != nil {
			options = *patch.Options.Value
		}
		options = normalizeEventOptions(options)
		patch.Options = workbench.Optional[[]any]{Set: true, Value: &options}
	}
	sets := project.Row{}
	setOpt(sets, "number", patch.Number)
	setOpt(sets, "country_tag", patch.CountryTag)
	setOpt(sets, "title_en", patch.TitleEn)
	setOpt(sets, "title_zh", patch.TitleZh)
	setOpt(sets, "desc_en", patch.DescEn)
	setOpt(sets, "desc_zh", patch.DescZh)
	setOpt(sets, "quote_en", patch.QuoteEn)
	setOpt(sets, "quote_zh", patch.QuoteZh)
	setOpt(sets, "btn_en", patch.BtnEn)
	setOpt(sets, "btn_zh", patch.BtnZh)
	setOpt(sets, "picture_path", patch.PicturePath)
	setOpt(sets, "song_token", patch.SongToken)
	setOpt(sets, "auto_fire_on_startup", patch.AutoFireOnStartup)
	setOpt(sets, "auto_fire_after_days", patch.AutoFireAfterDays)
	setOptJSON(sets, "options", patch.Options)
	setOptJSON(sets, "display_name_locs", patch.DisplayNameLocs)
	setOptJSON(sets, "description_locs", patch.DescriptionLocs)
	setOptJSON(sets, "quote_locs", patch.QuoteLocs)
	setOptJSON(sets, "button_locs", patch.ButtonLocs)
	if cur.Source == "imported" {
		sets["source"] = "created"
	}
	row, err := updateRow(p, superEventsTable, id, sets, func(tx *project.Tx) error {
		next := project.Row{"country_tag": curRow["country_tag"], "number": curRow["number"]}
		for _, column := range []string{"country_tag", "number"} {
			if value, ok := sets[column]; ok {
				next[column] = value
			}
		}
		if next["number"] != nil && taken(tx.Rows(superEventsTable), id, func(row project.Row) bool {
			return sameSuperEventSlot(row, next["country_tag"], next["number"])
		}) {
			return statusErr(http.StatusConflict, "superevent number conflict")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return superEventFromRow(row), nil
}
