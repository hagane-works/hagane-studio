package exportengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/hoi4/emit/worldout"
)

type capturedTableVersion struct {
	version string
	rows    []json.RawMessage
}

type stageSpec struct {
	name   string
	tables []string

	writesWorldMeta bool
}

type stageOutput struct {
	entries         []Entry
	traitNamespaces map[string]string

	preservedLocRows []preservedLocRow
	sink             stageSinkDelta
}

type stageSinkDelta struct {
	warnings             []string
	ideaPictureDrops     []IdeaPictureDrop
	sharedPlacementPaths []string
	worldMeta            *worldout.Meta
}

func (d stageSinkDelta) applyTo(sink *imageSink) {
	if sink == nil {
		return
	}
	sink.warnings = append(sink.warnings, d.warnings...)
	sink.ideaPictureDrops = append(sink.ideaPictureDrops, d.ideaPictureDrops...)
	if len(d.sharedPlacementPaths) > 0 {
		if sink.sharedPlacementPaths == nil {
			sink.sharedPlacementPaths = map[string]bool{}
		}
		for _, name := range d.sharedPlacementPaths {
			sink.sharedPlacementPaths[name] = true
		}
	}
	if d.worldMeta != nil {
		sink.worldMeta = *d.worldMeta
	}
}

func sinkDeltaOf(local *imageSink, spec stageSpec) stageSinkDelta {
	delta := stageSinkDelta{warnings: local.warnings, ideaPictureDrops: local.ideaPictureDrops}
	for name := range local.sharedPlacementPaths {
		delta.sharedPlacementPaths = append(delta.sharedPlacementPaths, name)
	}
	sort.Strings(delta.sharedPlacementPaths)
	if spec.writesWorldMeta {
		meta := local.worldMeta
		delta.worldMeta = &meta
	}
	return delta
}

type stageMemo struct {
	snapshot  Snapshot
	enabled   bool
	verify    bool
	slot      string
	base      string
	hashes    map[string]string
	reused    int
	ran       int
	uncovered int
}

func newStageMemo(snapshot Snapshot, includeImported bool, opts nativeBuildOptions) *stageMemo {
	m := &stageMemo{snapshot: snapshot}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GO_GUI_STAGE_CACHE")))
	if !opts.gameFilesOnly || snapshot.versions == nil || snapshot.ProjectID <= 0 ||
		mode == "0" || mode == "off" || mode == "false" {
		return m
	}
	m.enabled = true
	m.verify = mode == "verify"
	m.slot = fmt.Sprintf("imported=%t", includeImported)
	hasher := sha256.New()
	fmt.Fprintf(hasher, "stagecache/v1\x00project=%d\x00%s\x00game-files-manifest-only\x00", snapshot.ProjectID, m.slot)
	_, _ = hasher.Write(snapshot.Project)
	m.base = hex.EncodeToString(hasher.Sum(nil))
	m.hashes = map[string]string{}
	return m
}

func (m *stageMemo) tableVersion(name string) (string, bool) {
	captured, ok := m.snapshot.versions[name]
	if !ok {
		return "", false
	}
	rows := m.snapshot.Tables[name]
	if sameRows(rows, captured.rows) {
		return "v:" + captured.version, true
	}
	if hash, ok := m.hashes[name]; ok {
		return hash, true
	}
	hasher := sha256.New()
	for _, row := range rows {
		fmt.Fprintf(hasher, "%d\x00", len(row))
		_, _ = hasher.Write(row)
	}
	hash := "h:" + hex.EncodeToString(hasher.Sum(nil))
	m.hashes[name] = hash
	return hash, true
}

func sameRows(a, b []json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0]
}

func (m *stageMemo) key(spec stageSpec) (string, bool) {
	hasher := sha256.New()
	fmt.Fprintf(hasher, "%s\x00%s", m.base, spec.name)
	for _, name := range spec.tables {
		version, ok := m.tableVersion(name)
		if !ok {
			return "", false
		}
		fmt.Fprintf(hasher, "\x00%s=%s", name, version)
	}
	return hex.EncodeToString(hasher.Sum(nil)), true
}

func (m *stageMemo) scoped(spec stageSpec) Snapshot {
	scoped := Snapshot{Schema: m.snapshot.Schema, ProjectID: m.snapshot.ProjectID, Project: m.snapshot.Project,
		Tables: make(map[string][]json.RawMessage, len(spec.tables))}
	for _, name := range spec.tables {
		if rows, ok := m.snapshot.Tables[name]; ok {
			scoped.Tables[name] = rows
		}
	}
	return scoped
}

func (m *stageMemo) run(spec stageSpec, sink *imageSink, compute func(Snapshot, *imageSink) (stageOutput, error)) (stageOutput, error) {
	if !m.enabled {
		return compute(m.snapshot, sink)
	}
	key, ok := m.key(spec)
	if !ok {
		m.uncovered++
		return compute(m.snapshot, sink)
	}
	slot := spec.name + "/" + m.slot
	if cached, hit := stageOutputs.get(m.snapshot.ProjectID, slot, key); hit {
		if m.verify {
			fresh, err := isolatedStage(m.snapshot, spec, compute)
			if err != nil {
				return stageOutput{}, err
			}
			if diff := diffStageOutputs(cached, fresh); diff != "" {
				log.Printf("[gui] stage cache: project %d stage %s differs from a full build: %s", m.snapshot.ProjectID, spec.name, diff)
				stageOutputs.put(m.snapshot.ProjectID, slot, key, fresh)
				cached = fresh
			}
		}
		m.reused++
		cached.sink.applyTo(sink)
		return cached.clone(), nil
	}
	fresh, err := isolatedStage(m.scoped(spec), spec, compute)
	if err != nil {
		return stageOutput{}, err
	}
	stageOutputs.put(m.snapshot.ProjectID, slot, key, fresh)
	m.ran++
	fresh.sink.applyTo(sink)
	return fresh.clone(), nil
}

func isolatedStage(snapshot Snapshot, spec stageSpec, compute func(Snapshot, *imageSink) (stageOutput, error)) (stageOutput, error) {
	local := &imageSink{}
	out, err := compute(snapshot, local)
	if err != nil {
		return stageOutput{}, err
	}
	out.entries = slices.Clip(out.entries)
	out.sink = sinkDeltaOf(local, spec)
	return out, nil
}

func (o stageOutput) clone() stageOutput {
	clone := o
	clone.entries = append([]Entry(nil), o.entries...)
	if o.traitNamespaces != nil {
		clone.traitNamespaces = make(map[string]string, len(o.traitNamespaces))
		for name, value := range o.traitNamespaces {
			clone.traitNamespaces[name] = value
		}
	}
	return clone
}

func diffStageOutputs(a, b stageOutput) string {
	left, right := sortedEntries(a.entries), sortedEntries(b.entries)
	if len(left) != len(right) {
		return fmt.Sprintf("%d entries, full build has %d", len(left), len(right))
	}
	for i := range left {
		x, y := left[i], right[i]
		if x.Path != y.Path {
			return fmt.Sprintf("entry %q, full build has %q", x.Path, y.Path)
		}
		if !bytes.Equal(x.Bytes, y.Bytes) || x.Origin != y.Origin || x.SourceEntity != y.SourceEntity ||
			!slices.Equal(x.CosmeticColorTags, y.CosmeticColorTags) {
			return fmt.Sprintf("entry %q differs", x.Path)
		}
	}
	if !reflect.DeepEqual(a.preservedLocRows, b.preservedLocRows) {
		return "decoded localisation rows differ"
	}
	if !reflect.DeepEqual(a.traitNamespaces, b.traitNamespaces) {
		return "trait overlay namespaces differ"
	}
	if !slices.Equal(a.sink.warnings, b.sink.warnings) {
		return "warnings differ"
	}
	if !reflect.DeepEqual(a.sink.ideaPictureDrops, b.sink.ideaPictureDrops) ||
		!slices.Equal(a.sink.sharedPlacementPaths, b.sink.sharedPlacementPaths) ||
		!reflect.DeepEqual(a.sink.worldMeta, b.sink.worldMeta) {
		return "image sink differs"
	}
	return ""
}

func sortedEntries(entries []Entry) []Entry {
	sorted := append([]Entry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	return sorted
}

const stageCacheProjects = 100

type stageCacheEntry struct {
	key string
	out stageOutput
}

type stageCacheStore struct {
	mu       sync.Mutex
	order    []int64
	used     map[int64]time.Time
	projects map[int64]map[string]stageCacheEntry
	sweeper  sync.Once
}

var stageOutputs = &stageCacheStore{projects: map[int64]map[string]stageCacheEntry{}}

func (c *stageCacheStore) get(projectID int64, slot, key string) (stageOutput, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropIdleLocked(previewCacheNow())
	entry, ok := c.projects[projectID][slot]
	if !ok || entry.key != key {
		return stageOutput{}, false
	}
	c.touchLocked(projectID)
	return entry.out, true
}

func (c *stageCacheStore) put(projectID int64, slot, key string, out stageOutput) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stages := c.projects[projectID]
	if stages == nil {
		stages = map[string]stageCacheEntry{}
		c.projects[projectID] = stages
	}
	stages[slot] = stageCacheEntry{key: key, out: out}
	c.touchLocked(projectID)
	for len(c.order) > stageCacheProjects {
		c.dropLocked(c.order[0])
	}
	c.sweeper.Do(func() { go sweepIdle(c.sweep) })
}

func (c *stageCacheStore) touchLocked(projectID int64) {
	for i, id := range c.order {
		if id == projectID {
			c.order = append(c.order[:i:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, projectID)
	if c.used == nil {
		c.used = map[int64]time.Time{}
	}
	c.used[projectID] = previewCacheNow()
}

func (c *stageCacheStore) dropLocked(projectID int64) {
	delete(c.projects, projectID)
	delete(c.used, projectID)
	c.order = slices.DeleteFunc(c.order, func(id int64) bool { return id == projectID })
}

func (c *stageCacheStore) dropIdleLocked(now time.Time) {
	for _, id := range slices.Clone(c.order) {
		if now.Sub(c.used[id]) > previewCacheIdle {
			c.dropLocked(id)
		}
	}
}

func (c *stageCacheStore) sweep(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropIdleLocked(now)
}

const previewCacheIdle = 10 * time.Minute

var previewCacheNow = time.Now

func sweepIdle(sweep func(time.Time)) {
	for range time.Tick(time.Minute) {
		sweep(previewCacheNow())
	}
}

type StageReuse struct {
	Reused, Ran, Uncached int
}

func (m *stageMemo) reuse() StageReuse {
	return StageReuse{Reused: m.reused, Ran: m.ran, Uncached: m.uncovered}
}

var (
	stageCharacters       = stageSpec{name: "characters", tables: []string{"characters", "entity_obs_paths"}}
	stageTraitOverlays    = stageSpec{name: "trait_overlays", tables: []string{"characters", "custom_traits", "event_namespaces", "entity_obs_paths"}}
	stageAnimatedPortrait = stageSpec{name: "animated_portraits", tables: []string{"characters", "project_obs_assets"}}
	stageTechnologies     = stageSpec{name: "technologies", tables: []string{
		"technologies", "equipments", "sub_units", "division_templates", "division_deployments",
		"oob_air_wing_groups", "oob_air_wings", "oob_fleets", "oob_task_forces", "oob_ships",
		"project_raw_files", "entity_obs_paths",
	}}
	stageDecisions = stageSpec{name: "decisions", tables: []string{
		"decisions", "decision_categories", "super_events", "events", "focus_trees", "focus_nodes",
		"project_raw_files", "entity_obs_paths",
	}}
	stageFocus       = stageSpec{name: "focus", tables: []string{"focus_trees", "focus_nodes", "focus_file_policies", "project_obs_icons"}}
	stageIdeasEvents = stageSpec{name: "ideas_events", tables: []string{"ideas", "event_namespaces", "events", "entity_obs_paths"}}
	stageModifiers   = stageSpec{name: "modifiers", tables: []string{"opinion_modifiers", "dynamic_modifiers", "autonomy_states", "modifier_definitions"}}
	stageAdvanced    = stageSpec{name: "advanced_scripts", tables: []string{
		"custom_traits", "military_industrial_organizations", "intelligence_agencies", "doctrines",
		"special_project_specializations", "special_projects", "entity_obs_paths", "project_obs_icons",
	}}
	stageAIPlanning    = stageSpec{name: "ai_planning", tables: []string{"ai_strategy_plans", "ai_templates"}}
	stageWargoals      = stageSpec{name: "wargoals", tables: []string{"wargoal_types"}}
	stagePanels        = stageSpec{name: "panels", tables: []string{"ui_panels"}}
	stageScriptedFiles = stageSpec{name: "scripted_files", tables: []string{"scripted_snippets", "scripted_localisations"}}

	stageWorld = stageSpec{name: "world", writesWorldMeta: true, tables: []string{
		"countries", "history_files", "states", "bookmarks", "project_cosmetic_tags", "project_cosmetic_tag_names",
		"ideas", "focus_nodes", "characters", "division_templates", "division_deployments",
		"oob_air_wing_groups", "oob_fleets", "oob_task_forces",
		"project_raw_files", "project_loc_entries", "entity_obs_paths", "project_obs_assets",
	}}
	stagePreservedLocRows = stageSpec{name: "preserved_loc_rows", tables: []string{"project_loc_entries"}}
	stageOnActions        = stageSpec{name: "on_actions", tables: []string{
		"characters", "ideas", "event_namespaces", "events", "super_events", "focus_trees",
		"custom_traits", "entity_obs_paths",
	}}
)
