package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type createBookmarkIn struct {
	ProjectID           int64             `json:"project_id"`
	BookmarkID          string            `json:"bookmark_id" jsonschema:"required (empty → error, duplicate → 409, NO auto-suffix)"`
	NameEn              string            `json:"name_en,omitempty"`
	NameZh              string            `json:"name_zh,omitempty"`
	DescEn              string            `json:"desc_en,omitempty"`
	DescZh              string            `json:"desc_zh,omitempty"`
	DisplayNameLocs     map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs     map[string]string `json:"description_locs,omitempty"`
	Date                string            `json:"date,omitempty" jsonschema:"default 1936.1.1.12"`
	Picture             string            `json:"picture,omitempty" jsonschema:"default GFX_select_date_1936"`
	DefaultCountry      string            `json:"default_country,omitempty"`
	SelectableCountries []map[string]any  `json:"selectable_countries,omitempty" jsonschema:"list of {tag, history_en, minor, ...} dicts"`
	Filters             []string          `json:"filters,omitempty"`
}

type createBookmarkOut struct {
	Diagnostics  *[]saveDiagnostic `json:"diagnostics,omitempty"`
	BookmarkDBID int64             `json:"bookmark_db_id"`
	BookmarkID   string            `json:"bookmark_id"`
}

func registerCreateBookmark(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_bookmark",
		Description: "Create a bookmark (scenario entry on the country-select screen). bookmark_id required " +
			"(empty→error, duplicate→409, NO auto-suffix). selectable_countries = list of " +
			"{tag, history_en, minor, ...} dicts.",
	}, func(ctx context.Context, _ *CallToolRequest, in createBookmarkIn) (*CallToolResult, createBookmarkOut, error) {
		date := in.Date
		if date == "" {
			date = "1936.1.1.12"
		}
		picture := in.Picture
		if picture == "" {
			picture = "GFX_select_date_1936"
		}
		body := map[string]any{
			"bookmark_id":     in.BookmarkID,
			"name_en":         in.NameEn,
			"name_zh":         in.NameZh,
			"desc_en":         in.DescEn,
			"desc_zh":         in.DescZh,
			"date":            date,
			"picture":         picture,
			"default_country": in.DefaultCountry,
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}

		if in.SelectableCountries != nil {
			body["selectable_countries"] = in.SelectableCountries
		}
		if in.Filters != nil {
			body["filters"] = in.Filters
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/bookmarks", in.ProjectID), body)
		if err != nil {
			return nil, createBookmarkOut{}, err
		}
		out := createBookmarkOut{
			Diagnostics: saveDiagnosticsField(resp),
			BookmarkID:  stringField(resp, "bookmark_id"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.BookmarkDBID = int64(id)
		}
		return nil, out, nil
	})
}

type setCountryHistoryIn struct {
	ProjectID                    int64                   `json:"project_id"`
	CountryTag                   string                  `json:"country_tag" jsonschema:"upserted by country_tag"`
	Capital                      *int                    `json:"capital,omitempty" jsonschema:"state id"`
	SetPoliticsRulingParty       *string                 `json:"set_politics_ruling_party,omitempty"`
	SetPoliticsLastElection      *string                 `json:"set_politics_last_election,omitempty" jsonschema:"HOI4 date YYYY.M.D"`
	SetPoliticsElectionFrequency *int                    `json:"set_politics_election_frequency,omitempty" jsonschema:"months between elections"`
	SetPoliticsElectionsAllowed  *bool                   `json:"set_politics_elections_allowed,omitempty"`
	SetPopularities              map[string]int          `json:"set_popularities,omitempty" jsonschema:"canonical ROOT ideology id to percentage map; non-empty map must total 100"`
	SetPopularitiesDemocratic    *int                    `json:"set_popularities_democratic,omitempty" jsonschema:"legacy vanilla-root compatibility field; prefer set_popularities"`
	SetPopularitiesCommunism     *int                    `json:"set_popularities_communism,omitempty" jsonschema:"legacy vanilla-root compatibility field; prefer set_popularities"`
	SetPopularitiesFascism       *int                    `json:"set_popularities_fascism,omitempty" jsonschema:"legacy vanilla-root compatibility field; prefer set_popularities"`
	SetPopularitiesNeutrality    *int                    `json:"set_popularities_neutrality,omitempty" jsonschema:"legacy vanilla-root compatibility field; prefer set_popularities"`
	SetResearchSlots             *int                    `json:"set_research_slots,omitempty"`
	SetConvoys                   *int                    `json:"set_convoys,omitempty"`
	SetStability                 *float64                `json:"set_stability,omitempty"`
	SetWarSupport                *float64                `json:"set_war_support,omitempty"`
	AddIdeasExtra                []string                `json:"add_ideas_extra,omitempty" jsonschema:"starting national spirit/idea tokens; REPLACES when supplied"`
	RemoveIdeas                  []string                `json:"remove_ideas,omitempty" jsonschema:"REPLACES when supplied"`
	CompleteNationalFocusIDs     []string                `json:"complete_national_focus_ids,omitempty" jsonschema:"REPLACES when supplied"`
	SetTechnologyIDs             []string                `json:"set_technology_ids,omitempty" jsonschema:"REPLACES when supplied"`
	SetCosmeticTag               *string                 `json:"set_cosmetic_tag,omitempty"`
	SetOOB                       *string                 `json:"set_oob,omitempty"`
	SetAirOOB                    *string                 `json:"set_air_oob,omitempty"`
	SetNavalOOB                  *string                 `json:"set_naval_oob,omitempty"`
	StartupFlags                 []string                `json:"startup_flags,omitempty" jsonschema:"set_country_flag tokens; REPLACES when supplied"`
	GiveGuaranteeTags            []string                `json:"give_guarantee_tags,omitempty" jsonschema:"country tags; REPLACES when supplied"`
	AddToFactionTags             []string                `json:"add_to_faction_tags,omitempty" jsonschema:"country tags; REPLACES when supplied and clears the legacy scalar"`
	AddToFaction                 *string                 `json:"add_to_faction,omitempty" jsonschema:"legacy single-tag field; prefer add_to_faction_tags"`
	SetAutonomy                  []map[string]any        `json:"set_autonomy,omitempty" jsonschema:"REPLACES entries such as {target, autonomy_state, freedom_level}"`
	DiplomaticRelation           []map[string]any        `json:"diplomatic_relation,omitempty" jsonschema:"REPLACES entries such as {country, relation, active}"`
	DateBlocks                   []historyDateBlockInput `json:"date_blocks,omitempty" jsonschema:"REPLACES dated history blocks in stable order"`
	RawScript                    *string                 `json:"raw_script,omitempty" jsonschema:"escape hatch for arbitrary history blocks"`
}

type historyDateBlockInput struct {
	Date   string `json:"date" jsonschema:"HOI4 date YYYY.M.D"`
	Script string `json:"script" jsonschema:"raw script inside the dated block"`
}

type setCountryHistoryOut struct {
	HistoryFileID int64  `json:"history_file_id"`
	CountryTag    string `json:"country_tag"`
	Created       bool   `json:"created"`
}

func registerSetCountryHistory(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "set_country_history",
		Description: "Set a country's complete day-0/scenario history (upsert by country_tag): " +
			"politics and elections, canonical ideology popularities, research/convoys, stability/war " +
			"support, starting or removed ideas, completed focuses, technologies, cosmetic tag, land/air/naval " +
			"OOBs, startup flags, guarantees, factions, autonomy, diplomatic relations and dated blocks. " +
			"Only supplied fields change; every supplied list/map replaces that field. Read the history_file first.",
	}, func(ctx context.Context, _ *CallToolRequest, in setCountryHistoryIn) (*CallToolResult, setCountryHistoryOut, error) {
		tag := strings.ToUpper(strings.TrimSpace(in.CountryTag))

		body := map[string]any{"country_tag": tag}
		if in.Capital != nil {
			body["capital"] = *in.Capital
		}
		if in.SetPoliticsRulingParty != nil {
			body["set_politics_ruling_party"] = *in.SetPoliticsRulingParty
		}
		if in.SetPoliticsLastElection != nil {
			body["set_politics_last_election"] = *in.SetPoliticsLastElection
		}
		if in.SetPoliticsElectionFrequency != nil {
			body["set_politics_election_frequency"] = *in.SetPoliticsElectionFrequency
		}
		if in.SetPoliticsElectionsAllowed != nil {
			body["set_politics_elections_allowed"] = *in.SetPoliticsElectionsAllowed
		}
		if in.SetPopularities != nil {
			body["set_popularities"] = in.SetPopularities
		}
		if in.SetPopularitiesDemocratic != nil {
			body["set_popularities_democratic"] = *in.SetPopularitiesDemocratic
		}
		if in.SetPopularitiesCommunism != nil {
			body["set_popularities_communism"] = *in.SetPopularitiesCommunism
		}
		if in.SetPopularitiesFascism != nil {
			body["set_popularities_fascism"] = *in.SetPopularitiesFascism
		}
		if in.SetPopularitiesNeutrality != nil {
			body["set_popularities_neutrality"] = *in.SetPopularitiesNeutrality
		}
		if in.SetResearchSlots != nil {
			body["set_research_slots"] = *in.SetResearchSlots
		}
		if in.SetConvoys != nil {
			body["set_convoys"] = *in.SetConvoys
		}
		if in.SetStability != nil {
			body["set_stability"] = *in.SetStability
		}
		if in.SetWarSupport != nil {
			body["set_war_support"] = *in.SetWarSupport
		}
		if in.AddIdeasExtra != nil {
			body["add_ideas_extra"] = in.AddIdeasExtra
		}
		if in.RemoveIdeas != nil {
			body["remove_ideas"] = in.RemoveIdeas
		}
		if in.CompleteNationalFocusIDs != nil {
			body["complete_national_focus_ids"] = in.CompleteNationalFocusIDs
		}
		if in.SetTechnologyIDs != nil {
			body["set_technology_ids"] = in.SetTechnologyIDs
		}
		if in.SetCosmeticTag != nil {
			body["set_cosmetic_tag"] = *in.SetCosmeticTag
		}
		if in.SetOOB != nil {
			body["set_oob"] = *in.SetOOB
		}
		if in.SetAirOOB != nil {
			body["set_air_oob"] = *in.SetAirOOB
		}
		if in.SetNavalOOB != nil {
			body["set_naval_oob"] = *in.SetNavalOOB
		}
		if in.StartupFlags != nil {
			body["startup_flags"] = in.StartupFlags
		}
		if in.GiveGuaranteeTags != nil {
			body["give_guarantee_tags"] = in.GiveGuaranteeTags
		}
		if in.AddToFactionTags != nil {
			body["add_to_faction_tags"] = in.AddToFactionTags
		}
		if in.AddToFaction != nil {
			body["add_to_faction"] = *in.AddToFaction
		}
		if in.SetAutonomy != nil {
			body["set_autonomy"] = in.SetAutonomy
		}
		if in.DiplomaticRelation != nil {
			body["diplomatic_relation"] = in.DiplomaticRelation
		}
		if in.DateBlocks != nil {
			body["date_blocks"] = in.DateBlocks
		}
		if in.RawScript != nil {
			body["raw_script"] = *in.RawScript
		}

		existingID, lookupErr := findHistoryFileID(ctx, proxy, in.ProjectID, tag)
		if lookupErr != nil {
			return nil, setCountryHistoryOut{}, lookupErr
		}

		var resp map[string]any
		var created bool
		var err error
		if existingID == 0 {
			resp, err = proxyAuthedCall(ctx, proxy, "POST",
				fmt.Sprintf("/api/projects/%d/history-files", in.ProjectID), body)
			created = true
		} else {
			resp, err = proxyAuthedCall(ctx, proxy, "PUT",
				fmt.Sprintf("/api/projects/%d/history-files/%d", in.ProjectID, existingID), body)
			created = false
		}
		if err != nil {
			return nil, setCountryHistoryOut{}, err
		}

		out := setCountryHistoryOut{
			CountryTag: stringField(resp, "country_tag"),
			Created:    created,
		}
		if id, ok := resp["id"].(float64); ok {
			out.HistoryFileID = int64(id)
		}
		return nil, out, nil
	})
}

func setHistoryRulingParty(ctx context.Context, proxy *RESTProxy, projectID int64, tag, party string) (id int64, created bool, err error) {
	existingID, err := findHistoryFileID(ctx, proxy, projectID, tag)
	if err != nil {
		return 0, false, err
	}
	body := map[string]any{"country_tag": tag, "set_politics_ruling_party": party}
	var resp map[string]any
	if existingID == 0 {
		resp, err = proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/history-files", projectID), body)
		created = true
	} else {
		resp, err = proxyAuthedCall(ctx, proxy, "PUT", fmt.Sprintf("/api/projects/%d/history-files/%d", projectID, existingID), body)
	}
	if err != nil {
		return 0, false, err
	}
	if rowID, ok := resp["id"].(float64); ok {
		id = int64(rowID)
	}
	return id, created, nil
}

func findHistoryFileID(ctx context.Context, proxy *RESTProxy, projectID int64, tag string) (int64, error) {
	raw, err := proxy.callRaw(ctx, http.MethodGet,
		fmt.Sprintf("/api/projects/%d/history-files", projectID), nil)
	if err != nil {
		return 0, err
	}
	if len(raw) == 0 {
		return 0, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0, fmt.Errorf("decoding upstream history-files list: %w", err)
	}
	for _, row := range rows {
		if ct, ok := row["country_tag"].(string); ok && strings.EqualFold(ct, tag) {
			if id, ok := row["id"].(float64); ok {
				return int64(id), nil
			}
		}
	}
	return 0, nil
}
