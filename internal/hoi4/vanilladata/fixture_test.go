package vanilladata

import (
	"testing"

	"haganeworks/internal/gameindex"
)

func useFixture(t *testing.T) {
	t.Helper()
	restore := gameindex.UseStatic(map[string][]byte{
		gameindex.IdeaIndex: []byte(`{
 "picture_sprites": ["generic_fascism_drift_2", "war_economy", "civilian_economy", "tot_economic_mobilisation", "BEL_rexist_legion", "new_economic_policy"],
 "picture_sprites_orphans": ["idea_PRC_sinochem"],
 "ideas": [
  {"category": "country", "idea_id": "a", "picture": "GFX_idea_generic_fascism_drift_2", "file": "common/ideas/x.txt", "line": 1},
  {"category": "country", "idea_id": "b", "picture": "idea_PRC_sinochem", "file": "common/ideas/x.txt", "line": 5},
  {"category": "country", "idea_id": "c", "picture": null, "file": "common/ideas/x.txt", "line": 9}
 ]}`),
		gameindex.DLCNames: []byte(`{"game_version": "1.19.3", "sources": ["fixture"], "dlcs": [
  {"name": "La Resistance", "kind": "expansion", "prefix": "LAR", "source": "dlc028"},
  {"name": "Man the Guns", "kind": "expansion", "prefix": "MTG", "source": "dlc023"},
  {"name": "Poland: United and Ready", "kind": "content_pack", "prefix": "", "source": "dlc002"}]}`),
		gameindex.VanillaEffects:   []byte(`{"total": 3, "ids": ["add_ideas", "country_event", "set_country_flag"]}`),
		gameindex.VanillaTriggers:  []byte(`{"total": 3, "ids": ["has_idea", "has_war", "is_ai"]}`),
		gameindex.StructuralFields: []byte(`{"fields": ["cancel_if_invalid", "is_triggered_only", "available_if_capitulated"], "by_entity": {}}`),
		gameindex.YesNoKeys:        []byte(`{"total": 3, "ids": ["instant_build", "keep_completed", "designer"]}`),
	})
	t.Cleanup(restore)
}
