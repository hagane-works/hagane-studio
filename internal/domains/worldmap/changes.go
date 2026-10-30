package worldmap

import (
	"fmt"
	"net/http"
	"time"

	"haganeworks/internal/project"
)

type worldMapChange struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type worldMapChanges struct {
	Count   int              `json:"count"`
	World   bool             `json:"world"`
	Records []worldMapChange `json:"records"`

	Confirmation string `json:"confirmation"`
}

func mapChangesOf(additions []WorldMapAddition) *worldMapChanges {
	if len(additions) == 0 {
		return nil
	}
	out := &worldMapChanges{Count: len(additions), Records: make([]worldMapChange, 0, len(additions))}
	var latest int64
	for _, a := range additions {
		out.Records = append(out.Records, worldMapChange{ID: a.ID, Kind: a.Kind, Name: a.Name, CreatedAt: a.CreatedAt})
		out.World = out.World || a.Kind == "world"
		latest = max(latest, a.ID)
	}
	out.Confirmation = fmt.Sprintf("restore-%d-%d", out.Count, latest)
	return out
}

type mapChangedProblem struct {
	*mapProblem
	MapChanges *worldMapChanges `json:"map_changes"`
}

func refuseMapChanged(w http.ResponseWriter, changes *worldMapChanges) {
	problem := newMapProblem(problemMapChanged, fmt.Sprintf("项目的地图已经有 %d 处改动(新增土地、省份改动或上一次的新世界):先「恢复原版地图」,再创作整个世界", changes.Count)).with("count", changes.Count)
	writeJSON(w, http.StatusConflict, mapChangedProblem{mapProblem: problem, MapChanges: changes})
}

func (h *handlers) confirmedRestore(w http.ResponseWriter, p *project.Project, store *mapStore, confirmation string) {
	additions, err := store.WorldMapAdditions()
	if err != nil {
		refuseMap(w, http.StatusServiceUnavailable, problemUnreadable, "读不到地图改动的记录")
		return
	}
	changes := mapChangesOf(additions)
	if changes == nil {
		writeJSON(w, http.StatusOK, map[string]any{"restored": 0, "removed": []worldMapChange{}})
		return
	}
	if changes.Confirmation != confirmation {
		problem := newMapProblem(problemMapChangesMoved, "地图的改动和卡片上列的不一样了,没有恢复:重新预览新世界,再确认一次").with("count", changes.Count)
		writeJSON(w, http.StatusConflict, mapChangedProblem{mapProblem: problem, MapChanges: changes})
		return
	}
	h.restore(w, p, store)
}
