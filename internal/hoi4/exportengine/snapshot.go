package exportengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const SnapshotSchema = "hoi4.export-snapshot.v1"

type Snapshot struct {
	Schema      string                       `json:"schema"`
	ProjectID   int64                        `json:"project_id"`
	Project     json.RawMessage              `json:"project"`
	Tables      map[string][]json.RawMessage `json:"tables"`
	Fingerprint string                       `json:"fingerprint"`
	CapturedAt  time.Time                    `json:"captured_at"`

	versions map[string]capturedTableVersion
}

type snapshotSource struct {
	name  string
	query string
}

var coreSnapshotSources = []snapshotSource{
	{"characters", `SELECT to_jsonb(t) FROM characters t WHERE project_id=$1 ORDER BY id`},
	{"custom_traits", `SELECT to_jsonb(t) FROM custom_traits t WHERE project_id=$1 ORDER BY id`},
	{"opinion_modifiers", `SELECT to_jsonb(t) FROM opinion_modifiers t WHERE project_id=$1 ORDER BY id`},
	{"dynamic_modifiers", `SELECT to_jsonb(t) FROM dynamic_modifiers t WHERE project_id=$1 ORDER BY id`},
	{"focus_trees", `SELECT to_jsonb(t) FROM focus_trees t WHERE project_id=$1 ORDER BY id`},
	{"focus_nodes", `SELECT to_jsonb(n) FROM focus_nodes n JOIN focus_trees t ON t.id=n.focus_tree_id WHERE t.project_id=$1 ORDER BY n.id`},
	{"focus_file_policies", `SELECT to_jsonb(t) FROM focus_file_policies t WHERE project_id=$1 ORDER BY id`},
	{"technologies", `SELECT to_jsonb(t) FROM technologies t WHERE project_id=$1 ORDER BY id`},
	{"equipments", `SELECT to_jsonb(t) FROM equipments t WHERE project_id=$1 ORDER BY id`},
	{"sub_units", `SELECT to_jsonb(t) FROM sub_units t WHERE project_id=$1 ORDER BY id`},
	{"division_templates", `SELECT to_jsonb(t) FROM division_templates t WHERE project_id=$1 ORDER BY id`},
	{"division_deployments", `SELECT to_jsonb(t) FROM division_deployments t WHERE project_id=$1 ORDER BY id`},
	{"military_industrial_organizations", `SELECT to_jsonb(t) FROM military_industrial_organizations t WHERE project_id=$1 ORDER BY id`},
	{"doctrines", `SELECT to_jsonb(t) FROM doctrines t WHERE project_id=$1 ORDER BY id`},
	{"intelligence_agencies", `SELECT to_jsonb(t) FROM intelligence_agencies t WHERE project_id=$1 ORDER BY id`},
	{"oob_air_wing_groups", `SELECT to_jsonb(t) FROM oob_air_wing_groups t WHERE project_id=$1 ORDER BY id`},
	{"oob_air_wings", `SELECT to_jsonb(t) FROM oob_air_wings t WHERE project_id=$1 ORDER BY id`},
	{"oob_fleets", `SELECT to_jsonb(t) FROM oob_fleets t WHERE project_id=$1 ORDER BY id`},
	{"oob_task_forces", `SELECT to_jsonb(t) FROM oob_task_forces t WHERE project_id=$1 ORDER BY id`},
	{"oob_ships", `SELECT to_jsonb(t) FROM oob_ships t WHERE project_id=$1 ORDER BY id`},
	{"ideas", `SELECT to_jsonb(t) FROM ideas t WHERE project_id=$1 ORDER BY id`},
	{"event_namespaces", `SELECT to_jsonb(t) FROM event_namespaces t WHERE project_id=$1 ORDER BY id`},
	{"events", `SELECT to_jsonb(t) FROM events t WHERE project_id=$1 ORDER BY id`},
	{"super_events", `SELECT to_jsonb(t) FROM super_events t WHERE project_id=$1 ORDER BY id`},
	{"decision_categories", `SELECT to_jsonb(t) FROM decision_categories t WHERE project_id=$1 ORDER BY id`},
	{"decisions", `SELECT to_jsonb(t) FROM decisions t WHERE project_id=$1 ORDER BY id`},
	{"special_project_specializations", `SELECT to_jsonb(t) FROM special_project_specializations t WHERE project_id=$1 ORDER BY id`},
	{"special_projects", `SELECT to_jsonb(t) FROM special_projects t WHERE project_id=$1 ORDER BY id`},
	{"scripted_snippets", `SELECT to_jsonb(t) FROM scripted_snippets t WHERE project_id=$1 ORDER BY id`},
	{"scripted_localisations", `SELECT to_jsonb(t) FROM scripted_localisations t WHERE project_id=$1 ORDER BY id`},
	{"modifier_definitions", `SELECT to_jsonb(t) FROM modifier_definitions t WHERE project_id=$1 ORDER BY id`},
	{"ai_strategy_plans", `SELECT to_jsonb(t) FROM ai_strategy_plans t WHERE project_id=$1 ORDER BY id`},
	{"ai_templates", `SELECT to_jsonb(t) FROM ai_templates t WHERE project_id=$1 ORDER BY id`},
	{"wargoal_types", `SELECT to_jsonb(t) FROM wargoal_types t WHERE project_id=$1 ORDER BY id`},
	{"ui_panels", `SELECT to_jsonb(t) FROM ui_panels t WHERE project_id=$1 ORDER BY id`},
	{"gui_overlays", `SELECT to_jsonb(t) FROM gui_overlays t WHERE project_id=$1 ORDER BY id`},
	{"countries", `SELECT to_jsonb(t) FROM countries t WHERE project_id=$1 ORDER BY id`},
	{"states", `SELECT to_jsonb(t) FROM states t WHERE project_id=$1 ORDER BY id`},
	{"autonomy_states", `SELECT to_jsonb(t) FROM autonomy_states t WHERE project_id=$1 ORDER BY id`},
	{"history_files", `SELECT to_jsonb(t) FROM history_files t WHERE project_id=$1 ORDER BY id`},
	{"project_raw_files", `SELECT to_jsonb(t) FROM project_raw_files t WHERE project_id=$1 ORDER BY id`},
	{"bookmarks", `SELECT to_jsonb(t) FROM bookmarks t WHERE project_id=$1 ORDER BY id`},
	{"project_loc_entries", `SELECT to_jsonb(t) FROM project_loc_entries t WHERE project_id=$1 ORDER BY id`},
	{"project_cosmetic_tags", `SELECT to_jsonb(t) FROM project_cosmetic_tags t WHERE project_id=$1 ORDER BY id`},
	{"project_cosmetic_tag_names", `SELECT to_jsonb(n) FROM project_cosmetic_tag_names n JOIN project_cosmetic_tags t ON t.id=n.cosmetic_tag_id WHERE t.project_id=$1 ORDER BY n.id`},
	{"entity_obs_paths", `SELECT to_jsonb(t) FROM entity_obs_paths t WHERE project_id=$1 ORDER BY id`},
	{"project_obs_icons", `SELECT to_jsonb(t) FROM project_obs_icons t WHERE project_id=$1 ORDER BY id`},
	{"project_obs_assets", `SELECT to_jsonb(t) FROM project_obs_assets t WHERE project_id=$1 ORDER BY id`},
}

func CoreTables() []string {
	names := make([]string, len(coreSnapshotSources))
	for i, source := range coreSnapshotSources {
		names[i] = source.name
	}
	return names
}

func NewSnapshot(projectID int64, project json.RawMessage, tables map[string][]json.RawMessage) (Snapshot, error) {
	snapshot := Snapshot{
		Schema:     SnapshotSchema,
		ProjectID:  projectID,
		Project:    project,
		Tables:     tables,
		CapturedAt: time.Now().UTC(),
	}
	var err error
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	return snapshot, err
}

func SnapshotFingerprint(snapshot Snapshot) (string, error) {
	names := make([]string, 0, len(snapshot.Tables))
	for name := range snapshot.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(snapshot.Schema))
	_, _ = hasher.Write(snapshot.Project)
	for _, name := range names {
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(name))
		for _, row := range snapshot.Tables[name] {
			if !json.Valid(row) {
				return "", fmt.Errorf("snapshot table %s contains invalid JSON", name)
			}
			_, _ = hasher.Write([]byte{0})
			_, _ = hasher.Write(row)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
