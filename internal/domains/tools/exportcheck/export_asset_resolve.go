package exportcheck

import (
	"context"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/vanilladata"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type AssetObsIndex struct {
	ByKind map[string]map[int64]string

	Icons map[string]string
}

type ObsHead func(key string) (bool, error)

const assetHeadWorkers = 12

const assetHeadSuccessTTL = 5 * time.Minute

const assetHeadCacheLimit = 50_000

type cachedAssetHead struct {
	expiresAt time.Time
}

var assetHeadSuccessCache sync.Map

var assetHeadCacheWrites atomic.Uint64

func cachedSuccessfulAssetHead(key string, now time.Time) bool {
	value, ok := assetHeadSuccessCache.Load(key)
	if !ok {
		return false
	}
	entry, ok := value.(cachedAssetHead)
	if !ok || !now.Before(entry.expiresAt) {
		assetHeadSuccessCache.Delete(key)
		return false
	}
	return true
}

func cacheSuccessfulAssetHead(key string, now time.Time) {
	assetHeadSuccessCache.Store(key, cachedAssetHead{expiresAt: now.Add(assetHeadSuccessTTL)})
	if assetHeadCacheWrites.Add(1)%256 != 0 {
		return
	}
	retained := 0
	assetHeadSuccessCache.Range(func(cacheKey, raw any) bool {
		entry, valid := raw.(cachedAssetHead)
		if !valid || !now.Before(entry.expiresAt) || retained >= assetHeadCacheLimit {
			assetHeadSuccessCache.Delete(cacheKey)
		} else {
			retained++
		}
		return true
	})
}

func assetObsKey(idx AssetObsIndex, kind string, entityID int64, spriteOrFile string) string {
	if kind == "focus_icon" {
		return idx.Icons[strings.TrimSpace(spriteOrFile)]
	}
	if byID := idx.ByKind[kind]; byID != nil {
		return byID[entityID]
	}
	return ""
}

func candidateAssetKeys(
	in AssetAuditInput, idx AssetObsIndex,
	vanillaIdeaPictures, vanillaFocusIcons VanillaNameSet,
) []string {
	seen := map[string]bool{}
	order := []string{}
	collect := func(kind string, entityID int64, spriteOrFile string) bool {
		key := strings.TrimSpace(assetObsKey(idx, kind, entityID, spriteOrFile))
		if key != "" && !seen[key] {
			seen[key] = true
			order = append(order, key)
		}

		return true
	}
	_ = AuditProjectAssets(in, collect, vanillaIdeaPictures, vanillaFocusIcons)
	return order
}

func prefetchAssetHeads(ctx context.Context, keys []string, head ObsHead) map[string]bool {
	result := make(map[string]bool, len(keys))
	if head == nil || len(keys) == 0 {
		return result
	}
	unique := make([]string, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	now := time.Now()
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if cachedSuccessfulAssetHead(key, now) {
			result[key] = true
			continue
		}
		unique = append(unique, key)
	}
	if len(unique) == 0 {
		return result
	}
	workers := min(assetHeadWorkers, runtime.GOMAXPROCS(0), len(unique))
	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range jobs {
				if ctx.Err() != nil {
					return
				}
				exists, err := head(key)
				reachable := err == nil && exists
				mu.Lock()
				result[key] = reachable
				mu.Unlock()
				if reachable {
					cacheSuccessfulAssetHead(key, time.Now())
				}
			}
		}()
	}
	func() {
		defer close(jobs)
		for _, key := range unique {
			select {
			case jobs <- key:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	return result
}

func (s *Store) AuditProjectAssetsFor(
	ctx context.Context, projectID int64, includeImported bool, head ObsHead,
) (AssetAudit, error) {
	var out AssetAudit

	project, found, err := s.GetMetadata(ctx, projectID)
	if err != nil || !found {
		return out, err
	}
	idx, err := s.LoadAssetObsIndex(ctx, projectID)
	if err != nil {
		return out, err
	}
	in := AssetAuditInput{
		ProjectTag:      project.CountryTag,
		HasDependencies: len(project.Dependencies) > 0,
		IncludeImported: includeImported,
	}
	chars, err := s.ListCharacters(ctx, projectID, "")
	if err != nil {
		return out, err
	}
	for _, c := range chars {
		in.Characters = append(in.Characters, AssetAuditEntity{
			ID: c.ID, Source: c.Source, CountryTag: c.CountryTag,
			Tag: c.Tag, LargePortrait: c.LargePortrait})
	}
	ideas, err := s.loadGateIdeas(ctx, projectID)
	if err != nil {
		return out, err
	}
	for _, i := range ideas {
		in.Ideas = append(in.Ideas, AssetAuditEntity{
			ID: i.ID, Source: i.Source, CountryTag: i.CountryTag,
			Picture: i.Picture, IdeaID: i.IdeaID})
	}
	events, err := s.ListEvents(ctx, projectID, nil, "")
	if err != nil {
		return out, err
	}
	for _, e := range events {
		in.Events = append(in.Events, AssetAuditEntity{
			ID: e.ID, Source: e.Source, CountryTag: e.CountryTag,
			Picture: e.Picture, EventID: e.EventID})
	}
	decisions, err := s.ListDecisions(ctx, projectID, nil, "")
	if err != nil {
		return out, err
	}
	for _, d := range decisions {
		in.Decisions = append(in.Decisions, AssetAuditEntity{
			ID: d.ID, Source: d.Source, Icon: d.Icon, DecisionID: d.DecisionID})
	}
	techs, err := s.loadGateTechs(ctx, projectID)
	if err != nil {
		return out, err
	}
	for _, t := range techs {
		in.Techs = append(in.Techs, AssetAuditEntity{
			ID: t.ID, Source: t.Source, TechID: t.TechID})
	}
	trees, err := s.loadGateTrees(ctx, projectID)
	if err != nil {
		return out, err
	}
	for _, t := range trees {
		at := AssetAuditTree{Source: t.Source}
		for _, n := range t.Nodes {
			at.Nodes = append(at.Nodes, AssetAuditEntity{
				FocusID: n.FocusID, Icon: n.Icon})
		}
		in.Trees = append(in.Trees, at)
	}

	vanillaIdea := vanillaIdeaPictureNames()
	vanillaFocus := vanillaFocusIconNames()
	keys := candidateAssetKeys(in, idx,
		vanillaIdea, vanillaFocus)
	reachable := prefetchAssetHeads(ctx, keys, head)
	resolver := func(kind string, entityID int64, spriteOrFile string) bool {
		return reachable[assetObsKey(idx, kind, entityID, spriteOrFile)]
	}
	return AuditProjectAssets(in, resolver,
		vanillaIdea, vanillaFocus), nil
}

func vanillaIdeaPictureNames() VanillaNameSet { return vanilladata.IsVanillaIdeaPicture }

func vanillaFocusIconNames() VanillaNameSet {
	icons := exportengine.FocusIconGate(false, nil)
	return func(name string) bool { return icons[name] }
}
