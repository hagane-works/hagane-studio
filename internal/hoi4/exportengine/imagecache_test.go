package exportengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func imgperfPNG(t *testing.T, seed byte) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = seed ^ byte(i)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func imgperfSnapshot(n int) Snapshot {
	var ideas, paths []json.RawMessage
	for i := 1; i <= n; i++ {
		ideas = append(ideas, json.RawMessage(fmt.Sprintf(`{"id":%d,"idea_id":"GER_idea_%03d","country_tag":"GER","idea_type":"country","name_en":"Idea %d","source":"created"}`, i, i, i)))
		paths = append(paths, json.RawMessage(fmt.Sprintf(`{"entity_type":"idea_picture","entity_id":%d,"obs_key":"idea/%d.png"}`, i, i)))
	}

	paths = append(paths, json.RawMessage(`{"entity_type":"idea_picture","entity_id":999999,"obs_key":"idea/stale.png"}`))
	return Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"ideas":            ideas,
		"entity_obs_paths": paths,
	}}
}

func entriesDigest(entries []Entry) string {
	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00", e.Path, e.Origin, e.SourceEntity)
		h.Write(e.Bytes)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestBlobPrefetch_ConcurrentIsFasterAndByteIdentical(t *testing.T) {
	const n = 200
	snapshot := imgperfSnapshot(n)
	pngs := map[string][]byte{}
	for i := 1; i <= n; i++ {
		pngs[fmt.Sprintf("idea/%d.png", i)] = imgperfPNG(t, byte(i))
	}
	var calls int64
	slow := func(key string, maxBytes int64) ([]byte, error) {
		atomic.AddInt64(&calls, 1)
		time.Sleep(20 * time.Millisecond)
		if key == "idea/stale.png" {
			return nil, errors.New("stale row must not be fetched")
		}
		data, ok := pngs[key]
		if !ok {
			return nil, fmt.Errorf("unknown key %q", key)
		}
		return data, nil
	}

	t.Setenv("GO_EXPORT_BLOB_CONCURRENCY", "1")
	start := time.Now()
	serial, err := BuildNativeShadowWithBlobs(snapshot, nil, false, slow)
	if err != nil {
		t.Fatal(err)
	}
	serialDur := time.Since(start)
	serialCalls := atomic.SwapInt64(&calls, 0)

	t.Setenv("GO_EXPORT_BLOB_CONCURRENCY", "12")

	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(max(12, runtime.GOMAXPROCS(0))))
	start = time.Now()
	concurrent, err := BuildNativeShadowWithBlobs(snapshot, nil, false, slow)
	if err != nil {
		t.Fatal(err)
	}
	concurrentDur := time.Since(start)
	concurrentCalls := atomic.LoadInt64(&calls)
	t.Logf("serial=%s (%d fetches) concurrent=%s (%d fetches)", serialDur, serialCalls, concurrentDur, concurrentCalls)

	if serialCalls != n || concurrentCalls != n {
		t.Fatalf("expected exactly %d fetches per build (no stale row, no double fetch), got serial=%d concurrent=%d", n, serialCalls, concurrentCalls)
	}
	if serialDur < 4*time.Second {
		t.Fatalf("serial run should take >=4s with 200x20ms, took %s", serialDur)
	}
	if concurrentDur > 800*time.Millisecond {
		t.Fatalf("concurrent run should take <=0.8s, took %s", concurrentDur)
	}
	if len(serial.Entries) != len(concurrent.Entries) {
		t.Fatalf("entry count differs: %d vs %d", len(serial.Entries), len(concurrent.Entries))
	}
	for i := range serial.Entries {
		a, b := serial.Entries[i], concurrent.Entries[i]
		if a.Path != b.Path || a.Origin != b.Origin || a.SourceEntity != b.SourceEntity || !bytes.Equal(a.Bytes, b.Bytes) {
			t.Fatalf("entry %d differs: %s vs %s", i, a.Path, b.Path)
		}
	}
	if entriesDigest(serial.Entries) != entriesDigest(concurrent.Entries) {
		t.Fatal("digest differs")
	}
	dds := 0
	for _, e := range serial.Entries {
		if strings.HasSuffix(e.Path, ".dds") && bytes.HasPrefix(e.Bytes, []byte("DDS ")) {
			dds++
		}
	}
	if dds != n {
		t.Fatalf("expected %d idea DDS textures, got %d", n, dds)
	}
	if len(serial.Warnings) != 0 || len(concurrent.Warnings) != 0 {
		t.Fatalf("no image should be skipped: %v / %v", serial.Warnings, concurrent.Warnings)
	}
}

func TestBlobPrefetch_OneFailureCostsOneImage(t *testing.T) {
	const n = 40
	snapshot := imgperfSnapshot(n)
	good := imgperfPNG(t, 7)
	var origin int64
	resolver := func(key string, maxBytes int64) ([]byte, error) {
		atomic.AddInt64(&origin, 1)
		if key == "idea/17.png" {
			return nil, errors.New("object storage: 404")
		}
		return good, nil
	}
	t.Setenv("GO_EXPORT_BLOB_CONCURRENCY", "8")
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, resolver)
	if err != nil {
		t.Fatalf("one bad image must not fail the export: %v", err)
	}
	dds, refs := 0, 0
	for _, e := range artifact.Entries {
		if strings.HasSuffix(e.Path, ".dds") {
			dds++
			if strings.Contains(e.Path, "ger_idea_017") {
				t.Fatalf("failed image must not ship: %s", e.Path)
			}
		}
		if e.Path == "Demo/common/ideas/ger_ger.txt" {
			refs = strings.Count(string(e.Bytes), "picture = ")
			if strings.Contains(string(e.Bytes), "picture = GER_ger_idea_017") {
				t.Fatal("failed image must not be referenced")
			}
		}
	}
	if dds != n-1 || refs != n-1 {
		t.Fatalf("expected %d textures and refs, got dds=%d refs=%d", n-1, dds, refs)
	}
	joined := strings.Join(artifact.Warnings, "\n")
	if len(artifact.Warnings) != 1 || !strings.Contains(joined, "GER_idea_017") {
		t.Fatalf("expected exactly one skipped-image warning for GER_idea_017, got %v", artifact.Warnings)
	}

	if got := atomic.LoadInt64(&origin); got != n+1 {
		t.Fatalf("expected %d origin calls (n + one fallback retry), got %d", n+1, got)
	}
}

func TestBlobPrefetch_NilResolverStaysNil(t *testing.T) {
	if r := newBlobPrefetch(t.Context(), nil, []prefetchKey{{Key: "x", Limit: 1}}, 4); r != nil {
		t.Fatal("nil origin must stay nil so the fail-closed branches keep firing")
	}
}

func TestBlobPrefetch_TighterBoundFallsBackToOrigin(t *testing.T) {
	big := bytes.Repeat([]byte{1}, 100)
	var calls []int64
	origin := func(key string, maxBytes int64) ([]byte, error) {
		calls = append(calls, maxBytes)
		if int64(len(big)) > maxBytes {
			return nil, errors.New("too large")
		}
		return big, nil
	}
	resolve := newBlobPrefetch(t.Context(), origin, []prefetchKey{{Key: "k", Limit: 1000}}, 1)
	if data, err := resolve("k", 1000); err != nil || len(data) != 100 {
		t.Fatalf("hit expected: %v %d", err, len(data))
	}
	if _, err := resolve("k", 10); err == nil {
		t.Fatal("a tighter bound than the fetched size must re-ask the origin and surface its error")
	}
	if len(calls) != 2 || calls[1] != 10 {
		t.Fatalf("expected prefetch + one fallback with the caller's bound, got %v", calls)
	}
}

func TestCollectPrefetchKeys_MirrorsCallSites(t *testing.T) {
	snapshot := Snapshot{Tables: map[string][]json.RawMessage{
		"characters":   {json.RawMessage(`{"id":11}`)},
		"technologies": {json.RawMessage(`{"id":7}`)},
		"entity_obs_paths": {
			json.RawMessage(`{"entity_type":"character_portrait","entity_id":11,"obs_key":"old/p.png"}`),
			json.RawMessage(`{"entity_type":"character_portrait","entity_id":11,"obs_key":"p.png"}`),
			json.RawMessage(`{"entity_type":"tech_icon","entity_id":7,"obs_key":"tech.png"}`),
			json.RawMessage(`{"entity_type":"tech_icon","entity_id":8,"obs_key":"gone.png"}`),
			json.RawMessage(`{"entity_type":"unknown_kind","entity_id":7,"obs_key":"never.png"}`),
		},
		"focus_trees":  {json.RawMessage(`{"id":1,"country_tag":"GER"}`)},
		"focus_nodes":  {json.RawMessage(`{"id":1,"focus_tree_id":1,"focus_id":"GER_x","text_icon":"GFX_text_y","dynamic_icon":[{"icon":"GFX_dyn_z"}]}`)},
		"doctrines":    {json.RawMessage(`{"id":1,"doctrine_id":"d","icon":"GFX_doc_w","source":"created"}`)},
		"gui_overlays": {json.RawMessage(`{"id":1}`)},
		"project_obs_icons": {
			json.RawMessage(`{"gfx_name":"GFX_focus_ger_x","obs_key":"focus/x.png"}`),
			json.RawMessage(`{"gfx_name":"GFX_text_y","obs_key":"focus/y.png"}`),
			json.RawMessage(`{"gfx_name":"GFX_dyn_z","obs_key":"focus/z.png"}`),
			json.RawMessage(`{"gfx_name":"GFX_doc_w","obs_key":"doc/w.png"}`),
			json.RawMessage(`{"gfx_name":"GFX_focus_unreferenced","obs_key":"focus/unreferenced.png"}`),
		},
		"project_obs_assets": {
			json.RawMessage(`{"asset_type":"project_cover","gfx_name":"thumbnail.png","obs_key":"cover.png"}`),
			json.RawMessage(`{"asset_type":"country_flag_ideology","gfx_name":"GER_fascism","obs_key":"flag_f.png"}`),
			json.RawMessage(`{"asset_type":"ideology_icon","gfx_name":"communism","obs_key":"ideo.png"}`),
			json.RawMessage(`{"asset_type":"gui_sprite_override","gfx_name":"a.png","obs_key":"gui/a.png"}`),
			json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/snap.zip"}`),
			json.RawMessage(`{"asset_type":"event_picture","gfx_name":"GFX_x","obs_key":"unused.png"}`),
		},
		"project_raw_files": {
			json.RawMessage(`{"id":1,"is_binary":true,"content_obs_key":"raw/a.bin","source":"created"}`),
			json.RawMessage(`{"id":2,"is_binary":true,"content_obs_key":"raw/imported.bin","source":"imported"}`),
			json.RawMessage(`{"id":3,"is_binary":false,"content_obs_key":"raw/text.bin","source":"created"}`),
		},
	}}
	got := map[string]int64{}
	for _, k := range collectPrefetchKeys(snapshot, false) {
		got[k.Key] = k.Limit
	}
	want := map[string]int64{
		"p.png": 32 << 20, "tech.png": 32 << 20,
		"focus/x.png": 32 << 20, "focus/y.png": 32 << 20, "focus/z.png": 32 << 20, "doc/w.png": 32 << 20,
		"cover.png": 32 << 20, "flag_f.png": 32 << 20, "ideo.png": 32 << 20, "gui/a.png": 32 << 20,
		"gui/snap.zip": guiSnapshotLimit, "raw/a.bin": 256 << 20,
	}
	if len(got) != len(want) {
		t.Fatalf("keys: got %v want %v", got, want)
	}
	for k, limit := range want {
		if got[k] != limit {
			t.Fatalf("key %s: got limit %d want %d (all: %v)", k, got[k], limit, got)
		}
	}
	public := CollectBlobRequests(snapshot, false)
	if len(public) != len(want) || public[0].ObjectKey == "" || public[0].MaxBytes <= 0 {
		t.Fatalf("public blob requests = %+v", public)
	}
	withImported := collectPrefetchKeys(snapshot, true)
	found := false
	for _, k := range withImported {
		if k.Key == "raw/imported.bin" {
			found = true
		}
	}
	if !found {
		t.Fatal("includeImported=true must prefetch imported raw binaries")
	}

	delete(snapshot.Tables, "gui_overlays")
	for _, k := range collectPrefetchKeys(snapshot, false) {
		if strings.HasPrefix(k.Key, "gui/") {
			t.Fatalf("gui asset %s must not be prefetched without gui_overlays rows", k.Key)
		}
	}
	fullRequests := CollectBlobRequests(snapshot, true)
	foundSource, foundOverride := false, false
	for _, request := range fullRequests {
		switch request.ObjectKey {
		case "gui/snap.zip":
			foundSource = request.MaxBytes == guiSnapshotLimit
		case "gui/a.png":
			foundOverride = true
		}
	}
	if !foundSource {
		t.Fatalf("full export did not request its imported UI baseline: %+v", fullRequests)
	}
	if foundOverride {
		t.Fatalf("unreferenced GUI sprite override was requested without overlays: %+v", fullRequests)
	}
}

func TestDDSCache_SameBytesSameOutput(t *testing.T) {
	raw := imgperfPNG(t, 3)
	uncached, err := imageBytesToDDSUncached(raw, "a.png")
	if err != nil {
		t.Fatal(err)
	}
	first, err := imageBytesToDDS(raw, "a.png")
	if err != nil {
		t.Fatal(err)
	}
	second, err := imageBytesToDDS(raw, "b.PNG")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(uncached, first) || !bytes.Equal(first, second) {
		t.Fatal("cached DDS must equal the encoder output byte for byte")
	}
	if _, ok := processDDSCache.get(ddsCacheKey(raw, "x.png")); !ok {
		t.Fatal("expected a cache hit for the same sha+extension")
	}
	if ddsCacheKey(raw, "x.png") == ddsCacheKey(raw, "x.tga") {
		t.Fatal("extension must be part of the key (DecodeImage dispatches on it)")
	}
	if !bytes.HasPrefix(first, []byte("DDS ")) {
		t.Fatal("output must still be a DDS container")
	}
}

func TestDDSCache_LRUBoundedByBytes(t *testing.T) {
	c := newDDSCache(10)
	c.put("a", bytes.Repeat([]byte{1}, 4))
	c.put("b", bytes.Repeat([]byte{2}, 4))
	c.get("a")
	c.put("c", bytes.Repeat([]byte{3}, 4))
	if _, ok := c.get("b"); ok {
		t.Fatal("b should have been evicted (LRU)")
	}
	if _, ok := c.get("a"); !ok {
		t.Fatal("a should survive (recently used)")
	}
	if _, ok := c.get("c"); !ok {
		t.Fatal("c should be present")
	}
	c.put("huge", bytes.Repeat([]byte{4}, 11))
	if _, ok := c.get("huge"); ok {
		t.Fatal("an item larger than the cache must not be stored")
	}
	if c.bytes > 10 {
		t.Fatalf("cache exceeded its byte bound: %d", c.bytes)
	}
}

func TestSnap47_EntriesManifest(t *testing.T) {
	src := os.Getenv("HOI4_SNAP47")
	if src == "" {
		t.Skip("HOI4_SNAP47 not set")
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	fixed := imgperfPNG(t, 42)
	var calls int64
	start := time.Now()
	sleepMS, _ := strconv.Atoi(os.Getenv("HOI4_SNAP47_SLEEP_MS"))
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, true, func(key string, maxBytes int64) ([]byte, error) {
		atomic.AddInt64(&calls, 1)
		if sleepMS > 0 {
			time.Sleep(time.Duration(sleepMS) * time.Millisecond)
		}
		return fixed, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("entries=%d fetches=%d warnings=%d build=%s", len(artifact.Entries), calls, len(artifact.Warnings), time.Since(start))
	var out strings.Builder
	for _, e := range artifact.Entries {
		sum := sha256.Sum256(e.Bytes)
		fmt.Fprintf(&out, "%s %s\n", hex.EncodeToString(sum[:]), e.Path)
	}
	if dst := os.Getenv("HOI4_SNAP47_OUT"); dst != "" {
		if err := os.WriteFile(dst, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
