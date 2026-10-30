package states

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

const (
	codeGameUnconfigured = "vanilla_root_unconfigured"
	codeStatesDirMissing = "vanilla_states_dir_missing"
	codeStatesEmpty      = "vanilla_states_empty"
)

const expectedVanillaGameVersion = "1.19.2.0 (Operation Postern v1.19.2.0.a729 (d245))"

const (
	restoreSkipHasContent    = "already_has_content"
	restoreMissNotInVanilla  = "not_in_vanilla"
	restoreMissNoVanillaData = "vanilla_has_no_extras"
)

type VanillaStateContentCounts struct {
	NavalBases     int      `json:"naval_bases"`
	Bunkers        int      `json:"bunkers"`
	CoastalBunkers int      `json:"coastal_bunkers"`
	Dams           int      `json:"dams"`
	Landmarks      []string `json:"landmarks"`
	DatedBlocks    int      `json:"dated_blocks"`
	HasDLC         int      `json:"has_dlc"`
	ForceLink      bool     `json:"force_link"`
	Resistance     int      `json:"resistance"`
	OtherContent   int      `json:"other_content"`
}

func emptyContentCounts() VanillaStateContentCounts {
	return VanillaStateContentCounts{Landmarks: []string{}}
}

func (c *VanillaStateContentCounts) add(o VanillaStateContentCounts) {
	c.NavalBases += o.NavalBases
	c.Bunkers += o.Bunkers
	c.CoastalBunkers += o.CoastalBunkers
	c.Dams += o.Dams
	c.Landmarks = append(c.Landmarks, o.Landmarks...)
	c.DatedBlocks += o.DatedBlocks
	c.HasDLC += o.HasDLC
	if o.ForceLink {
		c.ForceLink = true
	}
	c.Resistance += o.Resistance
	c.OtherContent += o.OtherContent
}

type restoreDetail struct {
	ID       int64                     `json:"id"`
	StateID  int64                     `json:"state_id"`
	Restored bool                      `json:"restored"`
	Reason   string                    `json:"reason,omitempty"`
	Keys     []string                  `json:"keys"`
	Content  VanillaStateContentCounts `json:"content"`
}

type restoreResult struct {
	DryRun              bool                      `json:"dry_run"`
	Restored            int                       `json:"restored"`
	Skipped             int                       `json:"skipped"`
	Missing             int                       `json:"missing"`
	RestoredStateIDs    []int64                   `json:"restored_state_ids"`
	SkippedStateIDs     []int64                   `json:"skipped_state_ids"`
	MissingStateIDs     []int64                   `json:"missing_state_ids"`
	Totals              VanillaStateContentCounts `json:"totals"`
	Details             []restoreDetail           `json:"details"`
	VanillaFiles        int                       `json:"vanilla_files"`
	VanillaStates       int                       `json:"vanilla_states"`
	VanillaVersionKnown bool                      `json:"vanilla_version_known"`
	VanillaGameVersion  string                    `json:"vanilla_game_version,omitempty"`
	ExpectedGameVersion string                    `json:"expected_game_version"`
}

type restoreInput struct {
	StateDBIDs []int64 `json:"state_db_ids"`
	All        bool    `json:"all"`
	DryRun     bool    `json:"dry_run"`
}

func sourceError(code, message string, extra map[string]any) error {
	body := map[string]any{"code": code, "env": "VANILLA_ROOT", "path": "history/states/*.txt", "message": message}
	for k, v := range extra {
		body[k] = v
	}
	return detailError(http.StatusServiceUnavailable, body)
}

func (h *handlers) loadVanillaStates() (*vanillaStates, error) {
	if h.deps.Game == nil {
		return nil, sourceError(codeGameUnconfigured,
			"The Hearts of Iron IV folder is not set, so the base game's state files (history/states/*.txt) cannot be read. Set it in settings; nothing was changed.", nil)
	}
	game, err := h.deps.Game()
	if err != nil || game == nil {
		return nil, sourceError(codeGameUnconfigured,
			"The Hearts of Iron IV folder is not set, so the base game's state files (history/states/*.txt) cannot be read. Set it in settings; nothing was changed.", nil)
	}
	states := vanilla.stateFiles(game)
	if states.DirMissing {
		return nil, sourceError(codeStatesDirMissing,
			"The Hearts of Iron IV folder has no readable history/states directory — it should be the game's installation folder. Nothing was changed.",
			map[string]any{"path": "history/states"})
	}
	if states.States == 0 {
		return nil, sourceError(codeStatesEmpty,
			"history/states in the Hearts of Iron IV folder contains no readable state files. Nothing was changed.",
			map[string]any{"path": "history/states", "files": states.Files})
	}
	return states, nil
}

func validateRestoreInput(in restoreInput) error {
	if !in.All && len(in.StateDBIDs) == 0 {
		return detailError(422, "state_db_ids must not be empty (or pass all: true)")
	}
	if in.All && len(in.StateDBIDs) > 0 {
		return detailError(422, "pass either state_db_ids or all: true, not both")
	}
	if len(in.StateDBIDs) > stateBulkMaxIDs {
		return detailError(422, fmt.Sprintf("state_db_ids exceeds the %d limit", stateBulkMaxIDs))
	}
	return nil
}

func isEmptyRawExtras(v any) bool {
	if v == nil {
		return true
	}
	m, ok := v.(map[string]any)
	return ok && len(m) == 0
}

func restoreVanillaContent(p *project.Project, idx *vanillaStates, in restoreInput) (*restoreResult, error) {
	if err := validateRestoreInput(in); err != nil {
		return nil, err
	}
	res := &restoreResult{
		DryRun:              in.DryRun,
		Totals:              emptyContentCounts(),
		Details:             []restoreDetail{},
		RestoredStateIDs:    []int64{},
		SkippedStateIDs:     []int64{},
		MissingStateIDs:     []int64{},
		VanillaFiles:        idx.Files,
		VanillaStates:       idx.States,
		VanillaVersionKnown: idx.Version.Known,
		VanillaGameVersion:  idx.Version.Display,
		ExpectedGameVersion: expectedVanillaGameVersion,
	}
	if res.VanillaVersionKnown && res.VanillaGameVersion == "" {
		res.VanillaGameVersion = idx.Version.Raw
	}
	err := p.Write([]string{statesTable}, func(tx *project.Tx) error {
		ids := in.StateDBIDs
		if in.All {
			ids = projectStateIDs(tx.Rows(statesTable))
			if len(ids) > stateBulkMaxIDs {
				return detailError(422, fmt.Sprintf("project has %d states, over the %d limit", len(ids), stateBulkMaxIDs))
			}
		}
		for _, id := range ids {
			row := tx.Find(statesTable, id)
			if row == nil {
				return errStateMissing
			}
			stateID, _ := whole(row["state_id"])
			detail := restoreDetail{ID: id, StateID: stateID, Keys: []string{}, Content: emptyContentCounts()}
			if !isEmptyRawExtras(row["raw_extras"]) {
				detail.Reason = restoreSkipHasContent
				res.Skipped++
				res.SkippedStateIDs = append(res.SkippedStateIDs, stateID)
				res.Details = append(res.Details, detail)
				continue
			}
			entry := idx.ByID[stateID]
			var extras map[string]any
			if entry != nil {
				extras, _ = entry.Row["raw_extras"].(map[string]any)
			}
			if entry == nil || len(extras) == 0 {
				detail.Reason = restoreMissNotInVanilla
				if entry != nil {
					detail.Reason = restoreMissNoVanillaData
				}
				res.Missing++
				res.MissingStateIDs = append(res.MissingStateIDs, stateID)
				res.Details = append(res.Details, detail)
				continue
			}

			if _, err := tx.Update(statesTable, id, func(r project.Row) { r["raw_extras"] = plain(extras) }); err != nil {
				return err
			}
			detail.Restored = true
			detail.Keys = append([]string{}, entry.RawExtrasKeys...)
			detail.Content = entry.Content
			res.Restored++
			res.RestoredStateIDs = append(res.RestoredStateIDs, stateID)
			res.Totals.add(entry.Content)
			res.Details = append(res.Details, detail)
		}
		sort.Strings(res.Totals.Landmarks)
		if in.DryRun {
			return errDryRun
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return nil, err
	}
	return res, nil
}

func projectStateIDs(rows []project.Row) []int64 {
	type pair struct{ stateID, id int64 }
	pairs := make([]pair, 0, len(rows))
	for _, row := range rows {
		sid, _ := whole(row["state_id"])
		pairs = append(pairs, pair{sid, project.Int64(row["id"])})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].stateID != pairs[j].stateID {
			return pairs[i].stateID < pairs[j].stateID
		}
		return pairs[i].id < pairs[j].id
	})
	out := make([]int64, len(pairs))
	for i, p := range pairs {
		out[i] = p.id
	}
	return out
}

var (
	vanillaDateKeyRE     = regexp.MustCompile(`^\d{3,4}\.\d{1,2}\.\d{1,2}(\.\d{1,2})?$`)
	vanillaProvinceKeyRE = regexp.MustCompile(`^\d+$`)
)

var vanillaNotAProvinceBuilding = map[string]bool{"level": true, "has_dlc": true}

var vanillaResistanceKeys = []string{"set_compliance", "set_resistance", "start_resistance"}

func childBlocks(block *paradox.Block) []*paradox.Entry {
	out := []*paradox.Entry{}
	if block == nil {
		return out
	}
	for _, ent := range block.Entries {
		if ent.Key != nil && ent.Block != nil {
			out = append(out, ent)
		}
	}
	return out
}

func scalarAssignmentsDeep(block *paradox.Block, fn func(key string)) {
	if block == nil {
		return
	}
	for _, ent := range block.Entries {
		if ent.Key == nil {
			continue
		}
		if ent.Block != nil {
			scalarAssignmentsDeep(ent.Block, fn)
			continue
		}
		fn(strings.TrimSpace(*ent.Key))
	}
}

func scanBuildings(block *paradox.Block, hit *VanillaStateContentCounts) {
	for _, prov := range childBlocks(block) {
		if !vanillaProvinceKeyRE.MatchString(strings.TrimSpace(*prov.Key)) {
			continue
		}
		scalarAssignmentsDeep(prov.Block, func(name string) {
			switch name {
			case "naval_base":
				hit.NavalBases++
			case "bunker":
				hit.Bunkers++
			case "coastal_bunker":
				hit.CoastalBunkers++
			case "dam":
				hit.Dams++
			default:
				if !vanillaNotAProvinceBuilding[name] {
					hit.OtherContent++
				}
			}
		})
		for _, sub := range childBlocks(prov.Block) {
			if name := strings.TrimSpace(*sub.Key); strings.HasPrefix(name, "landmark_") {
				hit.Landmarks = append(hit.Landmarks, name)
			} else {
				hit.OtherContent++
			}
		}
	}
}

func scanHistory(block *paradox.Block, hit *VanillaStateContentCounts) {
	for _, ent := range childBlocks(block) {
		key := strings.TrimSpace(*ent.Key)
		if vanillaDateKeyRE.MatchString(key) {
			hit.DatedBlocks++
		}
		if key == "buildings" {
			scanBuildings(ent.Block, hit)
		} else {
			scanHistory(ent.Block, hit)
		}
	}
}

func countKeyOccurrences(block *paradox.Block, want string) int {
	if block == nil {
		return 0
	}
	n := 0
	for _, ent := range block.Entries {
		if ent.Key != nil && strings.TrimSpace(*ent.Key) == want {
			n++
		}
		if ent.Block != nil {
			n += countKeyOccurrences(ent.Block, want)
		}
	}
	return n
}

func scanVanillaStateContent(stateBlock *paradox.Block) VanillaStateContentCounts {
	hit := emptyContentCounts()
	hit.HasDLC = countKeyOccurrences(stateBlock, "has_dlc")
	for _, key := range vanillaResistanceKeys {
		hit.Resistance += countKeyOccurrences(stateBlock, key)
	}
	for _, ent := range childBlocks(stateBlock) {
		switch strings.TrimSpace(*ent.Key) {
		case "history":
			scanHistory(ent.Block, &hit)
		case "buildings":
			scanBuildings(ent.Block, &hit)
		}
	}
	hit.ForceLink = countKeyOccurrences(stateBlock, "force_link_ownership_to") > 0
	sort.Strings(hit.Landmarks)
	return hit
}
