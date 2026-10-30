package exportengine

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

const (
	defaultBlobConcurrency = 12
	maxBlobConcurrency     = 32
	defaultDDSCacheBytes   = int64(512 << 20)
)

func blobConcurrency() int {
	n := defaultBlobConcurrency
	if v := strings.TrimSpace(os.Getenv("GO_EXPORT_BLOB_CONCURRENCY")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 1 && parsed <= maxBlobConcurrency {
			n = parsed
		}
	}
	return min(n, runtime.GOMAXPROCS(0))
}

type prefetchKey struct {
	Key   string
	Limit int64
}

type BlobRequest struct {
	ObjectKey string `json:"object_key"`
	MaxBytes  int64  `json:"max_bytes"`
}

func CollectBlobRequests(snapshot Snapshot, includeImported bool) []BlobRequest {
	keys := collectPrefetchKeys(snapshot, includeImported)
	out := make([]BlobRequest, 0, len(keys))
	for _, item := range keys {
		out = append(out, BlobRequest{ObjectKey: item.Key, MaxBytes: item.Limit})
	}
	return out
}

func collectPrefetchKeys(snapshot Snapshot, includeImported bool) []prefetchKey {
	seen := map[string]bool{}
	var out []prefetchKey
	add := func(key string, limit int64) {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, prefetchKey{Key: key, Limit: limit})
	}

	liveIDs := map[string]map[int64]bool{}
	for _, table := range entityOBSTables {
		if _, ok := liveIDs[table]; ok {
			continue
		}
		set := map[int64]bool{}
		for _, raw := range snapshot.Tables[table] {
			var row struct {
				ID int64 `json:"id"`
			}
			if json.Unmarshal(raw, &row) == nil && row.ID != 0 {
				set[row.ID] = true
			}
		}
		liveIDs[table] = set
	}
	lastByEntity := map[string]string{}
	var entityOrder []string
	for _, raw := range snapshot.Tables["entity_obs_paths"] {
		var row struct {
			EntityType string `json:"entity_type"`
			EntityID   int64  `json:"entity_id"`
			OBSKey     string `json:"obs_key"`
		}
		if json.Unmarshal(raw, &row) != nil || row.EntityID == 0 || row.OBSKey == "" {
			continue
		}
		table, known := entityOBSTables[row.EntityType]
		if !known || !liveIDs[table][row.EntityID] {
			continue
		}
		id := row.EntityType + "\x00" + strconv.FormatInt(row.EntityID, 10)
		if _, seen := lastByEntity[id]; !seen {
			entityOrder = append(entityOrder, id)
		}
		lastByEntity[id] = row.OBSKey
	}
	for _, id := range entityOrder {
		add(lastByEntity[id], 32<<20)
	}

	referencedIcons := referencedProjectIconNames(snapshot, includeImported)
	for _, raw := range snapshot.Tables["project_obs_icons"] {
		var row struct {
			GFXName string `json:"gfx_name"`
			OBSKey  string `json:"obs_key"`
		}
		if json.Unmarshal(raw, &row) == nil && row.GFXName != "" && referencedIcons[row.GFXName] {
			add(row.OBSKey, 32<<20)
		}
	}
	hasGUIOverlays := len(snapshot.Tables["gui_overlays"]) > 0

	hasTechSurface := false
	if project, err := emitmisc.DecodeEntityJSON(snapshot.Project); err == nil {
		if manifestRaw, ok := odToMap(project.A("tech_surface_manifest")); ok && len(manifestRaw) > 0 {
			hasTechSurface = true
		}
	}
	for _, raw := range snapshot.Tables["project_obs_assets"] {
		var row struct {
			AssetType string `json:"asset_type"`
			GFXName   string `json:"gfx_name"`
			OBSKey    string `json:"obs_key"`
		}
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		switch row.AssetType {
		case "gui_source":

			if row.GFXName == "snapshot.zip" && (includeImported || hasGUIOverlays || hasTechSurface) {
				add(row.OBSKey, guiSnapshotLimit)
			}
		case "gui_sprite_override":
			if hasGUIOverlays {
				add(row.OBSKey, 32<<20)
			}
		case guiVanillaScaffoldAsset:
			if hasGUIOverlays {
				add(row.OBSKey, guiVanillaScaffoldLimit)
			}
		case "ideology_source":
			_, _, sourceKey, active, _ := decodeIdeologySourceProject(snapshot.Project)
			if active && strings.TrimSpace(row.GFXName) == sourceKey {
				add(row.OBSKey, ideologySourceArchiveLimit)
			}
		case coverAssetType, "country_flag_ideology", "ideology_icon":
			add(row.OBSKey, 32<<20)
		}
	}
	for _, raw := range snapshot.Tables["project_raw_files"] {
		var row struct {
			Source        string `json:"source"`
			IsBinary      bool   `json:"is_binary"`
			ContentOBSKey string `json:"content_obs_key"`
			RawKind       string `json:"raw_kind"`
		}
		if json.Unmarshal(raw, &row) != nil || !row.IsBinary || row.ContentOBSKey == "" {
			continue
		}
		if !includeImported && row.Source == "imported" {
			continue
		}
		if row.RawKind == "empty_shadow" {
			continue
		}
		add(row.ContentOBSKey, 256<<20)
	}
	return out
}

var entityOBSTables = map[string]string{
	"character_portrait":     "characters",
	"character_advisor":      "characters",
	"tech_icon":              "technologies",
	"decision_icon":          "decisions",
	"decision_category_icon": "decision_categories",
	"super_event_picture":    "super_events",
	"idea_picture":           "ideas",
	"event_picture":          "events",
	"bookmark_picture":       "bookmarks",
	"country_flag":           "countries",
	"mio_icon":               "military_industrial_organizations",
	"intel_agency_logo":      "intelligence_agencies",
	"custom_trait_icon":      "custom_traits",
}

func referencedProjectIconNames(snapshot Snapshot, includeImported bool) map[string]bool {
	names := map[string]bool{}
	if len(snapshot.Tables["focus_trees"]) > 0 {
		for _, raw := range snapshot.Tables["focus_nodes"] {
			var node struct {
				FocusID     string `json:"focus_id"`
				TextIcon    string `json:"text_icon"`
				DynamicIcon []struct {
					Icon string `json:"icon"`
				} `json:"dynamic_icon"`
			}
			if json.Unmarshal(raw, &node) != nil {
				continue
			}
			names[emitmisc.FocusIconSpriteName(node.FocusID)] = true
			for _, d := range node.DynamicIcon {
				if n := strings.TrimSpace(d.Icon); n != "" {
					names[n] = true
				}
			}
			if n := strings.TrimSpace(node.TextIcon); n != "" {
				names[n] = true
			}
		}
	}
	for _, raw := range snapshot.Tables["military_industrial_organizations"] {
		var mio struct {
			Traits []struct {
				Token string `json:"token"`
				Icon  string `json:"icon"`
			} `json:"traits"`
		}
		if json.Unmarshal(raw, &mio) != nil {
			continue
		}
		for _, t := range mio.Traits {
			if t.Token != "" && t.Icon != "" {
				names[t.Icon] = true
			}
		}
	}
	for _, raw := range snapshot.Tables["doctrines"] {
		var row struct {
			Icon   string `json:"icon"`
			Source string `json:"source"`
		}
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		if !includeImported && row.Source == "imported" {
			continue
		}
		if n := strings.TrimSpace(row.Icon); n != "" {
			names[n] = true
		}
	}
	return names
}

type blobResult struct {
	data []byte
	err  error
}

type blobPrefetch struct {
	origin  BlobResolver
	mu      sync.Mutex
	results map[string]blobResult
}

func newBlobPrefetch(ctx context.Context, origin BlobResolver, keys []prefetchKey, concurrency int) BlobResolver {
	if origin == nil {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}
	p := &blobPrefetch{origin: origin, results: make(map[string]blobResult, len(keys))}
	if len(keys) > 0 {
		g, _ := errgroup.WithContext(ctx)
		g.SetLimit(concurrency)
		for _, k := range keys {
			k := k
			g.Go(func() error {
				data, err := origin(k.Key, k.Limit)
				p.mu.Lock()
				p.results[k.Key] = blobResult{data: data, err: err}
				p.mu.Unlock()
				return nil
			})
		}
		_ = g.Wait()
	}
	return p.resolve
}

func (p *blobPrefetch) resolve(key string, maxBytes int64) ([]byte, error) {
	p.mu.Lock()
	r, ok := p.results[key]
	p.mu.Unlock()
	if ok && r.err == nil && int64(len(r.data)) <= maxBytes {
		return r.data, nil
	}

	data, err := p.origin(key, maxBytes)
	if err == nil {
		p.mu.Lock()
		p.results[key] = blobResult{data: data}
		p.mu.Unlock()
	}
	return data, err
}

type ddsCache struct {
	mu       sync.Mutex
	maxBytes int64
	bytes    int64
	order    *list.List
	items    map[string]*list.Element
}

type ddsCacheItem struct {
	key string
	dds []byte
}

func newDDSCache(maxBytes int64) *ddsCache {
	return &ddsCache{maxBytes: maxBytes, order: list.New(), items: map[string]*list.Element{}}
}

func ddsCacheKey(raw []byte, filename string) string {
	sum := sha256.Sum256(raw)
	return string(sum[:]) + "\x00" + strings.ToLower(path.Ext(filename))
}

func (c *ddsCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*ddsCacheItem).dds, true
}

func (c *ddsCache) put(key string, dds []byte) {
	size := int64(len(dds))
	if size > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return
	}
	el := c.order.PushFront(&ddsCacheItem{key: key, dds: dds})
	c.items[key] = el
	c.bytes += size
	for c.bytes > c.maxBytes {
		last := c.order.Back()
		if last == nil {
			break
		}
		item := last.Value.(*ddsCacheItem)
		c.order.Remove(last)
		delete(c.items, item.key)
		c.bytes -= int64(len(item.dds))
	}
}

func ddsCacheLimit() int64 {
	if v := strings.TrimSpace(os.Getenv("GO_EXPORT_DDS_CACHE_MB")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			return n << 20
		}
	}
	return defaultDDSCacheBytes
}

var processDDSCache = newDDSCache(ddsCacheLimit())
