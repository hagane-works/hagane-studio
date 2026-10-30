package exportcheck

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func clearAssetHeadSuccessCacheForTest() {
	assetHeadSuccessCache.Range(func(key, _ any) bool {
		assetHeadSuccessCache.Delete(key)
		return true
	})
	assetHeadCacheWrites.Store(0)
}

func TestCandidateAssetKeysOnlyIncludesReachableAuditSlice(t *testing.T) {
	idx := AssetObsIndex{
		ByKind: map[string]map[int64]string{
			"idea_picture": {1: "wanted", 2: "imported", 99: "unrelated"},
		},
		Icons: map[string]string{},
	}
	in := AssetAuditInput{
		ProjectTag: "GER",
		Ideas: []AssetAuditEntity{
			{ID: 1, Source: "created", CountryTag: "GER", IdeaID: "GER_custom", Picture: "GFX_idea_custom"},
			{ID: 2, Source: "imported", CountryTag: "GER", IdeaID: "GER_old", Picture: "GFX_idea_old"},
			{ID: 3, Source: "created", CountryTag: "GER", IdeaID: "GER_vanilla", Picture: "GFX_idea_vanilla"},
		},
	}
	keys := candidateAssetKeys(in, idx,
		func(name string) bool { return name == "vanilla" },
		func(string) bool { return false })
	if len(keys) != 1 || keys[0] != "wanted" {
		t.Fatalf("candidate keys=%v, want [wanted]", keys)
	}
}

func TestPrefetchAssetHeadsIsDedupedBoundedAndPositivelyCached(t *testing.T) {
	clearAssetHeadSuccessCacheForTest()
	t.Cleanup(clearAssetHeadSuccessCacheForTest)
	keys := make([]string, 0, 32)
	for i := 0; i < 24; i++ {
		keys = append(keys, fmt.Sprintf("key-%02d", i))
	}
	keys = append(keys, "key-00", "key-01")

	var calls, active, maximum atomic.Int64
	head := func(string) (bool, error) {
		calls.Add(1)
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(8 * time.Millisecond)
		active.Add(-1)
		return true, nil
	}
	got := prefetchAssetHeads(context.Background(), keys, head)
	if len(got) != 24 || calls.Load() != 24 {
		t.Fatalf("result=%d calls=%d, want 24 unique probes", len(got), calls.Load())
	}

	bound := int64(min(assetHeadWorkers, runtime.GOMAXPROCS(0)))
	if maximum.Load() < min(2, bound) || maximum.Load() > bound {
		t.Fatalf("max concurrency=%d, want 2..%d", maximum.Load(), bound)
	}
	second := prefetchAssetHeads(context.Background(), keys, head)
	if len(second) != 24 || calls.Load() != 24 {
		t.Fatalf("positive cache missed: result=%d calls=%d", len(second), calls.Load())
	}
}

func TestPrefetchAssetHeadsDoesNotCacheMissOrError(t *testing.T) {
	clearAssetHeadSuccessCacheForTest()
	t.Cleanup(clearAssetHeadSuccessCacheForTest)
	var mu sync.Mutex
	calls := map[string]int{}
	head := func(key string) (bool, error) {
		mu.Lock()
		calls[key]++
		mu.Unlock()
		if key == "error" {
			return false, fmt.Errorf("temporary")
		}
		return false, nil
	}
	for i := 0; i < 2; i++ {
		got := prefetchAssetHeads(context.Background(), []string{"missing", "error"}, head)
		if got["missing"] || got["error"] {
			t.Fatalf("unexpected reachable result: %#v", got)
		}
	}
	if calls["missing"] != 2 || calls["error"] != 2 {
		t.Fatalf("miss/error must be retried, calls=%v", calls)
	}
}
