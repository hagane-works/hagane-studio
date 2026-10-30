package exportengine

import (
	"encoding/json"
	"testing"
)

func TestPrefetchIncludesGUISnapshotForTechSurfaceWithoutOverlays(t *testing.T) {
	assets := []json.RawMessage{
		json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/snap.zip"}`),
	}

	t.Run("有科技面板但没有 gui_overlays 也要签", func(t *testing.T) {
		snapshot := Snapshot{
			Project: json.RawMessage(`{"tech_surface_manifest":{"root":"gridbox"}}`),
			Tables:  map[string][]json.RawMessage{"project_obs_assets": assets},
		}
		if limit, ok := prefetchLimitFor(snapshot, "gui/snap.zip"); !ok || limit != guiSnapshotLimit {
			t.Fatalf("科技面板项目应拿到快照签名额度,got ok=%v limit=%d", ok, limit)
		}
	})

	t.Run("两者都没有时仍然不签(不白下几百 MB)", func(t *testing.T) {
		snapshot := Snapshot{
			Project: json.RawMessage(`{}`),
			Tables:  map[string][]json.RawMessage{"project_obs_assets": assets},
		}
		if _, ok := prefetchLimitFor(snapshot, "gui/snap.zip"); ok {
			t.Fatal("既无 gui_overlays 又无 tech_surface_manifest,不该签这个大对象")
		}
	})

	t.Run("空 manifest 等同于没有", func(t *testing.T) {
		snapshot := Snapshot{
			Project: json.RawMessage(`{"tech_surface_manifest":{}}`),
			Tables:  map[string][]json.RawMessage{"project_obs_assets": assets},
		}
		if _, ok := prefetchLimitFor(snapshot, "gui/snap.zip"); ok {
			t.Fatal("空 manifest 时 tech_surface.go 会直接返回,不该签")
		}
	})

	t.Run("只有 gui_overlays 时照旧要签", func(t *testing.T) {
		snapshot := Snapshot{
			Project: json.RawMessage(`{}`),
			Tables: map[string][]json.RawMessage{
				"project_obs_assets": assets,
				"gui_overlays":       {json.RawMessage(`{"id":1}`)},
			},
		}
		if _, ok := prefetchLimitFor(snapshot, "gui/snap.zip"); !ok {
			t.Fatal("原有行为不能被改坏")
		}
	})
}

func prefetchLimitFor(snapshot Snapshot, key string) (int64, bool) {
	for _, k := range collectPrefetchKeys(snapshot, false) {
		if k.Key == key {
			return k.Limit, true
		}
	}
	return 0, false
}
