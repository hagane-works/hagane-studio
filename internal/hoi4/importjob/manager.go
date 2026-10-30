package importjob

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

var sessionIDRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type State struct {
	ProjectID  int64          `json:"project_id"`
	SessionID  string         `json:"session_id"`
	Status     string         `json:"status"`
	Phase      string         `json:"phase"`
	FilesDone  int            `json:"files_done"`
	FilesTotal int            `json:"files_total"`
	Percent    float64        `json:"percent"`
	Rows       int64          `json:"rows"`
	Error      string         `json:"error,omitempty"`
	ErrorCode  string         `json:"error_code,omitempty"`
	HTTPStatus int            `json:"http_status,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
}

type Options struct {
	EditRevisions                                                                      map[string]string `json:"edit_revisions,omitempty"`
	CountryTag                                                                         string            `json:"country_tag"`
	ConflictPolicy                                                                     string            `json:"conflict_policy"`
	Characters, Focus, Tech, Ideas, Events, Decisions, Bookmarks, HistoryFiles, States bool
}

func AllOptions() Options {
	return Options{ConflictPolicy: "overwrite", Characters: true, Focus: true, Tech: true, Ideas: true, Events: true, Decisions: true, Bookmarks: true, HistoryFiles: true, States: true}
}

type runningJob struct {
	state       State
	cancel      context.CancelFunc
	gfxMappings map[string]string
}

type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*runningJob

	AfterImport func(*project.Project) error
}

func NewManager() *Manager {
	return &Manager{jobs: map[string]*runningJob{}}
}

func jobKey(projectID int64, sessionID string) string {
	return fmt.Sprintf("%d:%s", projectID, sessionID)
}

func ValidSessionID(id string) bool { return sessionIDRE.MatchString(id) }

func (m *Manager) Start(p *project.Project, sessionID, root string, options Options) (State, error) {
	projectID := p.ID()
	if !sessionIDRE.MatchString(sessionID) {
		return State{}, errors.New("invalid project_id or session_id")
	}
	options.CountryTag = strings.ToUpper(strings.TrimSpace(options.CountryTag))
	options.ConflictPolicy = strings.ToLower(strings.TrimSpace(options.ConflictPolicy))
	if options.ConflictPolicy == "" {
		options.ConflictPolicy = "overwrite"
	}
	if options.ConflictPolicy != "overwrite" && options.ConflictPolicy != "skip" {
		return State{}, errors.New("invalid conflict_policy")
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return State{}, errors.New("staged session not found")
	}
	key := jobKey(projectID, sessionID)
	m.mu.Lock()
	m.pruneLocked(time.Now().UTC().Add(-24 * time.Hour))
	if existing := m.jobs[key]; existing != nil && existing.state.Status == "running" {
		state := existing.state
		m.mu.Unlock()
		return state, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &runningJob{state: State{ProjectID: projectID, SessionID: sessionID, Status: "running", Phase: "localisation", StartedAt: time.Now().UTC()}, cancel: cancel}
	m.jobs[key] = job
	state := job.state
	m.mu.Unlock()
	go m.run(ctx, job, p, root, options)
	return state, nil
}

func (m *Manager) pruneLocked(before time.Time) {
	for key, job := range m.jobs {
		if job.state.FinishedAt != nil && job.state.FinishedAt.Before(before) {
			delete(m.jobs, key)
		}
	}
}

func (m *Manager) Get(projectID int64, sessionID string) (State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job := m.jobs[jobKey(projectID, sessionID)]
	if job == nil {
		return State{}, false
	}
	return job.state, true
}

func (m *Manager) Cancel(projectID int64, sessionID string) bool {
	m.mu.RLock()
	job := m.jobs[jobKey(projectID, sessionID)]
	m.mu.RUnlock()
	if job == nil || job.state.Status != "running" {
		return false
	}
	job.cancel()
	return true
}

func (m *Manager) AssetContext(projectID int64, sessionID string) (map[string]string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job := m.jobs[jobKey(projectID, sessionID)]
	if job == nil || job.gfxMappings == nil {
		return nil, false
	}
	return job.gfxMappings, true
}

func (m *Manager) update(job *runningJob, change func(*State)) {
	m.mu.Lock()
	change(&job.state)
	m.mu.Unlock()
}

func stagePercent(base, span float64, done, total int) float64 {
	if total <= 0 {
		return base
	}
	return base + span*float64(done)/float64(total)
}

func (m *Manager) run(ctx context.Context, job *runningJob, p *project.Project, root string, options Options) {
	defer os.RemoveAll(root)
	s := newSession(ctx, root, options, p.Record())
	var rows int64
	setPhase := func(phase string, percent float64) {
		m.update(job, func(st *State) {
			st.Phase, st.Percent, st.FilesDone, st.FilesTotal = phase, percent, 0, 0
		})
	}
	progress := func(base, span float64) func(done, total int, staged int64) {
		return func(done, total int, staged int64) {
			m.update(job, func(st *State) {
				st.FilesDone, st.FilesTotal, st.Rows = done, total, rows+staged
				st.Percent = stagePercent(base, span, done, total)
			})
		}
	}
	entityProgress := func(base, span float64) func(EntityProgress) {
		report := progress(base, span)
		return func(p EntityProgress) { report(p.FilesDone, p.FilesTotal, p.Rows) }
	}

	var (
		gfxCount       int64
		entityCounts   map[string]int64
		treeCounts     TreeEntityCounts
		worldCounts    EventsWorldCounts
		structuredMisc StructuredMiscResult
		mapFiles       int
		err            error
	)
	locRows, err := s.stageLocalisation(func(p LocProgress) {
		m.update(job, func(st *State) {
			st.FilesDone, st.FilesTotal, st.Rows = p.FilesDone, p.FilesTotal, p.Rows
			st.Percent = stagePercent(0, 15, p.FilesDone, p.FilesTotal)
		})
	})
	rows += locRows
	if err == nil {
		setPhase("gfx", 15)
		gfxCount, err = s.stageGFX()
	}
	if err == nil {
		setPhase("entities", 18)
		var n int64
		n, err = s.stageSimpleEntities(entityProgress(18, 12))
		rows += n
	}
	if err == nil {
		setPhase("characters_ideas", 30)
		var n int64
		n, err = s.stageCharsExtras(CharsExtrasStageOptions{ImportCharacters: options.Characters, ImportIdeas: options.Ideas, CountryTag: options.CountryTag}, entityProgress(30, 12))
		rows += n
	}
	if err == nil {
		setPhase("focus_technology", 42)
		report := progress(42, 13)
		treeCounts, err = s.stageTrees(TreeStageOptions{ProjectID: p.ID(), CountryTag: options.CountryTag, ImportFocus: options.Focus, ImportTech: options.Tech}, func(p TreeEntityProgress) {
			report(p.FilesDone, p.FilesTotal, p.Rows)
		})
		rows += treeCounts.FocusTrees + treeCounts.FocusNodes + treeCounts.Technologies
	}
	if err == nil {

		if gameIndexBuilding() {
			setPhase("game_index", 55)
		}
		_, err = s.gameFacts()
	}
	if err == nil {
		setPhase("events_world", 55)
		report := progress(55, 15)
		worldCounts, err = s.stageEventsWorld(EventsWorldStageOptions{ProjectID: p.ID(), CountryTag: options.CountryTag, ImportEvents: options.Events, ImportDecisions: options.Decisions, ImportBookmarks: options.Bookmarks, ImportHistoryFiles: options.HistoryFiles, ImportStates: options.States}, func(p EventsWorldProgress) {
			report(p.FilesDone, p.FilesTotal, p.Rows)
		})
		rows += worldCounts.Total()
	}
	if err == nil {
		setPhase("structured_entities", 70)
		structuredMisc, err = s.stageStructuredMisc(entityProgress(70, 8))
		for _, warning := range structuredMisc.Warnings {
			s.warn(warning)
		}
		for _, count := range structuredMisc.Counts {
			rows += count
		}
	}
	if err == nil {
		setPhase("countries_factions", 78)
		var countries CountryCounts
		countries, err = s.stageCountries(CountryStageOptions{ImportCountries: true, ImportFactions: true})
		rows += countries.Countries + countries.Factions
		for _, warning := range countries.Warnings {
			s.warn(warning)
		}
	}
	if err == nil {
		err = s.stageCosmeticTags()
	}
	if err == nil {
		setPhase("units", 82)
		var units UnitCounts
		units, err = s.stageUnits(options.CountryTag)
		rows += units.RawFiles + units.AirWingGroups + units.AirWings + units.Fleets + units.TaskForces + units.Ships
	}
	if err == nil {
		setPhase("misc_entities", 86)
		var misc MiscCounts
		misc, err = s.stageMiscEntities(entityProgress(86, 5))
		for _, count := range misc {
			rows += count
		}
	}
	blobs := p.Blobs()
	if err == nil {
		mapFiles, err = s.stageMapFiles(blobs)
	}
	if err == nil {
		err = s.reportUnrouted()
	}
	if err == nil {
		s.stageCharsExtrasNeededImages()
		entityCounts = s.entityCounts()
		m.mu.Lock()
		job.gfxMappings = s.gfx
		m.mu.Unlock()
	}
	if err == nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	if err == nil {
		setPhase("merge", 92)
		err = p.Write(schema.Tables(), func(tx *project.Tx) error {
			steps := []func() error{
				func() error { return s.mergeSimpleEntities(tx) },
				func() error {
					return s.mergeCharsExtras(tx, CharsExtrasMergeOptions{ConflictPolicy: options.ConflictPolicy})
				},
				func() error { return s.mergeTrees(tx, TreeMergeOptions{ConflictPolicy: options.ConflictPolicy}) },
				func() error {
					return s.mergeEventsWorld(tx, EventsWorldMergeOptions{ConflictPolicy: options.ConflictPolicy})
				},
				func() error {
					return s.mergeStructuredMisc(tx, MiscMergeOptions{ConflictPolicy: options.ConflictPolicy})
				},
				func() error { return s.mergeCountries(tx, CountryMergeOptions{ConflictPolicy: options.ConflictPolicy}) },
				func() error { return s.mergeUnits(tx, options.ConflictPolicy) },
				func() error { return s.mergeLocalisation(tx) },
				func() error { return s.mergeCosmeticTags(tx) },
				func() error { return s.mergeMapFiles(tx, MiscMergeOptions{ConflictPolicy: options.ConflictPolicy}) },
			}
			for _, step := range steps {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := step(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	var guiSourceFiles, ideologySourceFiles, countryFlags int
	var warnings []string
	if err == nil {
		guiSourceFiles, err = s.persistGUISourceSnapshot(p, blobs)
		if err != nil {
			s.warn("GUI source snapshot was not persisted: " + err.Error())
			err = nil
		}
		ideologySourceFiles, err = s.persistIdeologySourceSnapshot(p, blobs)
		if err != nil {
			s.warn("Ideology source snapshot was not persisted: " + err.Error())
			err = nil
		}
		if m.AfterImport != nil {
			if hookErr := m.AfterImport(p); hookErr != nil {
				s.warn("Imported ideologies were not added to the politics registry: " + hookErr.Error())
			}
		}
		err = p.Write(schema.Tables(), func(tx *project.Tx) error {
			var notes []string
			var flagErr error
			countryFlags, notes, flagErr = s.persistImportedCountryFlags(tx, blobs)
			if flagErr != nil {
				s.warn("Country flags were not kept: " + flagErr.Error())
			}
			if len(notes) > 10 {
				notes = append(notes[:10], fmt.Sprintf("…and %d more country flags were not kept", len(notes)-10))
			}
			for _, note := range notes {
				s.warn(note)
			}
			return nil
		})
		warnings = s.sortedWarnings()
	}

	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	job.state.FinishedAt = &now
	job.state.Rows = rows
	if err != nil {
		if errors.Is(err, context.Canceled) {
			job.state.Status, job.state.Phase = "cancelled", "cancelled"
		} else {
			job.state.Status, job.state.Phase, job.state.Error = "failed", "failed", err.Error()
		}
		return
	}
	job.state.Status, job.state.Phase, job.state.Percent = "complete", "complete", 100
	result := legacyResult(job.state.SessionID, options.ConflictPolicy, s.neededImageList(), gfxCount)
	result["edit_revisions"] = map[string]string{}
	result["warnings"] = warnings
	result["public_images_reused"] = 0
	result["gui_source_files"] = guiSourceFiles
	result["ideology_source_files"] = ideologySourceFiles
	result["country_flags"] = countryFlags
	result["map_files"] = mapFiles
	applyResultCounts(result, entityCounts, treeCounts, worldCounts)
	job.state.Result = result
}
