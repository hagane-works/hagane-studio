package gameview

import (
	"container/list"
	"context"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/guirender"
)

const (
	cacheIdle = 20 * time.Minute

	sweepEvery = 2 * time.Minute
)

type indexEntry struct {
	index *guirender.Index
	used  time.Time
}

type indexFlight struct {
	done  chan struct{}
	index *guirender.Index
	err   error
	stale bool
}

type indexCache struct {
	mu         sync.Mutex
	projects   int
	perProject int
	entries    map[string]indexEntry
	flights    map[string]*indexFlight
}

func newIndexCache(projects, perProject int) *indexCache {
	return &indexCache{
		projects: projects, perProject: perProject,
		entries: map[string]indexEntry{}, flights: map[string]*indexFlight{},
	}
}

var (
	projectIndexes = newIndexCache(3, 2)

	importedIndexes = newIndexCache(3, 3)
)

func cacheProject(key string) string {
	if at := strings.Index(key, "|"); at >= 0 {
		return key[:at]
	}
	return key
}

func (c *indexCache) load(ctx context.Context, key string, loader func(context.Context) (*guirender.Index, error)) (*guirender.Index, error) {
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		entry.used = time.Now()
		c.entries[key] = entry
		c.mu.Unlock()
		return entry.index, nil
	}
	flight := c.flights[key]
	if flight == nil {
		flight = &indexFlight{done: make(chan struct{})}
		c.flights[key] = flight
		go c.finish(key, flight, context.WithoutCancel(ctx), loader)
	}
	c.mu.Unlock()
	select {
	case <-flight.done:
		return flight.index, flight.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *indexCache) finish(key string, flight *indexFlight, ctx context.Context, loader func(context.Context) (*guirender.Index, error)) {
	flight.index, flight.err = loader(ctx)
	c.mu.Lock()
	current := c.flights[key] == flight && !flight.stale
	if current {
		delete(c.flights, key)
		if flight.err == nil && flight.index != nil {
			c.entries[key] = indexEntry{index: flight.index, used: time.Now()}
			c.evictLocked(key)
			startSweeper()
		}
	}
	c.mu.Unlock()
	close(flight.done)
}

func (c *indexCache) evictLocked(added string) {
	project := cacheProject(added)
	for {
		count, oldest := 0, ""
		var oldestAt time.Time
		for key, entry := range c.entries {
			if cacheProject(key) != project {
				continue
			}
			count++
			if key != added && (oldest == "" || entry.used.Before(oldestAt)) {
				oldest, oldestAt = key, entry.used
			}
		}
		if count <= c.perProject || oldest == "" {
			break
		}
		delete(c.entries, oldest)
	}
	for {
		lastUse := map[string]time.Time{}
		for key, entry := range c.entries {
			p := cacheProject(key)
			if entry.used.After(lastUse[p]) {
				lastUse[p] = entry.used
			}
		}
		if len(lastUse) <= c.projects {
			return
		}
		victim := ""
		for p, at := range lastUse {
			if p != project && (victim == "" || at.Before(lastUse[victim])) {
				victim = p
			}
		}
		if victim == "" {
			return
		}
		c.dropProjectLocked(victim)
	}
}

func (c *indexCache) dropProjectLocked(project string) {
	for key := range c.entries {
		if cacheProject(key) == project {
			delete(c.entries, key)
		}
	}

	for key, flight := range c.flights {
		if cacheProject(key) == project {
			flight.stale = true
			delete(c.flights, key)
		}
	}
}

func (c *indexCache) forget(project string) {
	c.mu.Lock()
	c.dropProjectLocked(project)
	c.mu.Unlock()
}

func (c *indexCache) dropIdle(idle time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	dropped := false
	for key, entry := range c.entries {
		if time.Since(entry.used) >= idle {
			delete(c.entries, key)
			dropped = true
		}
	}
	return dropped
}

type bytesCache struct {
	mu    sync.Mutex
	limit int
	size  int
	order *list.List
	items map[string]*list.Element
}

type bytesItem struct {
	key  string
	body []byte
	used time.Time
}

func newBytesCache(limit int) *bytesCache {
	return &bytesCache{limit: limit, order: list.New(), items: map[string]*list.Element{}}
}

func (c *bytesCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.items[key]
	if !ok {
		return nil, false
	}
	item := element.Value.(*bytesItem)
	item.used = time.Now()
	c.order.MoveToFront(element)
	return item.body, true
}

func (c *bytesCache) put(key string, body []byte) {
	if len(body) > c.limit/4 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		c.size -= len(element.Value.(*bytesItem).body)
		c.order.Remove(element)
	}
	c.items[key] = c.order.PushFront(&bytesItem{key: key, body: body, used: time.Now()})
	c.size += len(body)
	for c.size > c.limit {
		last := c.order.Back()
		item := last.Value.(*bytesItem)
		c.order.Remove(last)
		delete(c.items, item.key)
		c.size -= len(item.body)
	}
	startSweeper()
}

func (c *bytesCache) dropIdle(idle time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	dropped := false
	for element := c.order.Back(); element != nil; {
		item := element.Value.(*bytesItem)
		previous := element.Prev()
		if time.Since(item.used) >= idle {
			c.order.Remove(element)
			delete(c.items, item.key)
			c.size -= len(item.body)
			dropped = true
		}
		element = previous
	}
	return dropped
}

var (
	vanillaPNGs = newBytesCache(64 << 20)

	projectPNGs = newBytesCache(48 << 20)
)

var sweeperOnce sync.Once

func startSweeper() {
	sweeperOnce.Do(func() {
		go func() {
			for range time.Tick(sweepEvery) {
				sweepIdle(cacheIdle)
			}
		}()
	})
}

func sweepIdle(idle time.Duration) {
	dropped := projectIndexes.dropIdle(idle)
	dropped = importedIndexes.dropIdle(idle) || dropped
	dropped = vanillaPNGs.dropIdle(idle) || dropped
	dropped = projectPNGs.dropIdle(idle) || dropped
	dropped = vanillaLanguages.dropIdle(idle) || dropped
	dropped = dropIdleGame(idle) || dropped
	if dropped {

		guirender.ReleaseCaches()
		debug.FreeOSMemory()
	}
}
