package gameview

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/guirender"
)

var guiDataRoutes = map[string]bool{
	"alerts": true, "bookmarks": true, "buildings": true, "frontend": true, "character": true, "characters": true,
	"country-stats": true, "country-profile": true, "equipment": true, "equipment-designer": true,
	"decisions": true, "designer": true, "diplomacy": true, "divisions": true, "doctrines": true,
	"event": true, "event-choices": true, "events": true, "focus-tree": true, "focus-trees": true, "focus-tree-choices": true,
	"idea-choices": true, "industrial-orgs": true, "industrial-org": true, "commander-roster": true, "leader-details": true,
	"officer-stats": true, "logistics": true, "officer-corps": true, "production": true, "sprites": true,
	"states": true, "tech-tree": true, "trade": true,
	"wargoal": true, "intel-agency": true, "subjects": true, "faction": true,
	"construction": true, "special-projects": true, "special-project": true, "sub-unit": true, "scientists": true,
	"army-overview": true, "navy-overview": true, "air-overview": true,
}

var guiDataParams = []string{"tag", "date", "id", "name", "template", "folder", "namespace", "party",
	"names", "q", "group", "kind", "cells", "modules", "upgrades", "options", "target"}

const (
	guiDataParamLimit = 256
	guiDataNamesLimit = 256
)

func guiDataQueryError(query url.Values) string {
	for _, name := range guiDataParams {
		value := query.Get(name)
		limit := guiDataParamLimit
		switch name {
		case "names":
			limit = guiDataParamLimit * guiDataNamesLimit
		case "cells", "modules":

			limit = guiDataParamLimit * 16
		}
		if len(value) > limit || strings.ContainsAny(value, "\r\n\x00") {
			return name + " is malformed"
		}
	}
	if names := query.Get("names"); names != "" && strings.Count(names, ",") >= guiDataNamesLimit {
		return "names lists too many sprites"
	}
	return ""
}

func dataRead(w http.ResponseWriter, r *http.Request, route string, index *guirender.Index, found bool) {
	if !found || index == nil {
		detail(w, http.StatusNotFound, "no game data for this project")
		return
	}
	query := r.URL.Query()
	param := func(name string) string { return strings.TrimSpace(query.Get(name)) }
	tag, date := param("tag"), param("date")

	switch route {
	case "alerts":
		writeJSON(w, http.StatusOK, index.AlertsPayload())
	case "bookmarks":
		writeJSON(w, http.StatusOK, index.BookmarksPayload())
	case "frontend":

		writeJSON(w, http.StatusOK, map[string]any{
			"backgrounds": index.FrontendBackgrounds(), "loading": index.LoadingPictures(),
			"version": index.GameVersion(),
		})
	case "buildings":
		writeJSON(w, http.StatusOK, index.BuildingsPayload(tag, date))
	case "character":
		nameKey, origin, ok := index.CharacterNameKey(param("token"))
		if !ok {
			detail(w, http.StatusNotFound, "no character with that token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": param("token"), "name_key": nameKey, "origin": origin})
	case "characters":
		people, ok := index.CharactersPayloadAt(tag, param("party"), date)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "no characters for tag", "tags": index.ListCharacterTags()})
			return
		}
		writeJSON(w, http.StatusOK, people)
	case "country-stats":
		writeJSON(w, http.StatusOK, index.CountryStatsPayload(tag, date))
	case "equipment-designer":
		draft := guirender.ParseDesignDraft(param("modules"), param("upgrades"), param("options") == "1")
		writeJSON(w, http.StatusOK, index.EquipmentDesignerDraftPayload(tag, date, param("name"), draft))
	case "wargoal":
		writeJSON(w, http.StatusOK, index.WarGoalPayload(tag, param("target"), date))
	case "scientists":
		writeJSON(w, http.StatusOK, index.ScientistsPayload(tag, date))
	case "army-overview":
		writeJSON(w, http.StatusOK, index.ArmyOverviewPayload(tag, date))
	case "navy-overview":
		writeJSON(w, http.StatusOK, index.NavyOverviewPayload(tag, date))
	case "air-overview":
		writeJSON(w, http.StatusOK, index.AirOverviewPayload(tag, date))
	case "special-projects":
		writeJSON(w, http.StatusOK, index.SpecialProjectsPayload(tag, date))
	case "sub-unit":
		writeJSON(w, http.StatusOK, index.SubUnitEditPayload(param("id")))
	case "special-project":
		writeJSON(w, http.StatusOK, index.SpecialProjectDetailPayload(tag, date, param("id")))
	case "construction":
		writeJSON(w, http.StatusOK, index.ConstructionPayload(tag, date))
	case "faction":
		writeJSON(w, http.StatusOK, index.FactionPayload(tag, date))
	case "subjects":
		writeJSON(w, http.StatusOK, index.SubjectsPayload(tag, date))
	case "intel-agency":
		writeJSON(w, http.StatusOK, index.IntelAgencyPayload(tag, date))
	case "equipment":
		writeJSON(w, http.StatusOK, index.EquipmentPayload(tag, param("name")))
	case "country-profile":
		writeJSON(w, http.StatusOK, index.CountryProfilePayload(tag, date))
	case "decisions":
		writeJSON(w, http.StatusOK, index.DecisionsPayloadAt(tag, date))
	case "idea-choices":
		writeJSON(w, http.StatusOK, index.IdeaChoicesPayload(tag, date, param("group")))
	case "industrial-orgs":
		writeJSON(w, http.StatusOK, index.IndustrialOrgsPayload(tag, date))
	case "industrial-org":
		writeJSON(w, http.StatusOK, index.IndustrialOrgPayload(tag, date, param("id")))
	case "doctrines":
		writeJSON(w, http.StatusOK, index.DoctrinesPayload(tag, date, param("folder")))
	case "commander-roster":
		writeJSON(w, http.StatusOK, index.CommanderRosterPayload(tag, date, param("kind")))
	case "leader-details":
		writeJSON(w, http.StatusOK, index.LeaderDetailsPayload(tag, date, param("id")))
	case "officer-stats":
		writeJSON(w, http.StatusOK, index.OfficerStatsPayload(tag, date))
	case "designer":
		writeJSON(w, http.StatusOK, index.DesignerPayloadFor(tag, date, param("template"), param("cells")))
	case "diplomacy":
		writeJSON(w, http.StatusOK, index.DiplomacyPayload(tag, date))
	case "divisions":
		writeJSON(w, http.StatusOK, index.DivisionsPayload(tag, date))
	case "event":
		event := index.EventPayloadFor(param("id"), tag, date)
		if event == nil {
			detail(w, http.StatusNotFound, "event not found: "+param("id"))
			return
		}
		writeJSON(w, http.StatusOK, event)
	case "events":
		writeJSON(w, http.StatusOK, map[string]any{"events": index.ListEvents(param("namespace"))})
	case "event-choices":

		offset, _ := strconv.Atoi(param("offset"))
		limit, _ := strconv.Atoi(param("limit"))
		if limit <= 0 {
			limit = 40
		}
		scope := "all"
		var events []guirender.EventChoice
		var total int
		if tag != "" && param("scope") != "all" {
			scope = "country"
			events, total = index.SearchCountryEventsPage(param("q"), max(0, offset), min(limit, 10000), tag)
		} else {
			events, total = index.SearchEventsPage(param("q"), max(0, offset), min(limit, 10000))
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events, "total": total, "offset": max(0, offset), "scope": scope})
	case "focus-tree":

		id := param("id")
		if id == "" && tag != "" {
			id = index.FocusTreeForTag(tag)
		}
		tree, ok := index.FocusTreePayload(id)
		if !ok {
			detail(w, http.StatusNotFound, "focus tree not found")
			return
		}
		if tag != "" {
			if panel := index.ContinuousFocusPanel(tag, date, id); panel != nil {
				tree["continuous"] = panel
			}
		}
		writeJSON(w, http.StatusOK, tree)
	case "focus-trees":
		writeJSON(w, http.StatusOK, map[string]any{"trees": index.ListFocusTrees()})
	case "focus-tree-choices":
		choices := index.FocusTreeChoices(tag)
		if choices == nil {
			choices = []guirender.FocusTreeChoice{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"trees": choices})
	case "logistics":
		writeJSON(w, http.StatusOK, index.LogisticsPayload(tag, date))
	case "officer-corps":
		people, ok := index.OfficerCorpsPayload(tag, date)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "no characters"})
			return
		}
		writeJSON(w, http.StatusOK, people)
	case "production":
		writeJSON(w, http.StatusOK, index.ProductionPayload(tag, date))
	case "sprites":
		names := strings.Split(query.Get("names"), ",")
		payload := map[string]any{"sprites": index.SpritesPayload(names)}
		if query.Get("include_usage") == "true" {
			if len(names) > 100 {
				detail(w, http.StatusUnprocessableEntity, "at most 100 sprite names for usage inspection")
				return
			}
			payload["usage_evidence"] = index.SpriteUsagePayload(names)
		}
		writeJSON(w, http.StatusOK, payload)
	case "states":
		id, _ := strconv.Atoi(param("id"))
		writeJSON(w, http.StatusOK, index.StatesOfPayload(tag, date, id))
	case "tech-tree":
		writeJSON(w, http.StatusOK, index.TechTreePayload(tag, date, param("folder")))
	case "trade":
		writeJSON(w, http.StatusOK, index.TradePayload(tag, date))
	default:
		detail(w, http.StatusNotFound, "Not Found")
	}
}
