package technology

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/techlifecycle"
	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

var lifecycleTables = []string{table, "equipments", "focus_trees", "focus_nodes", "events", "decisions", "ideas", "characters"}

func lifecycleTechnology(value Technology) techlifecycle.Technology {
	result := techlifecycle.Technology{
		DatabaseID: value.ID, ProjectID: value.ProjectID, TechID: value.TechID, Source: value.Source,
		Name: value.Name, NameEn: value.NameEn, NameZh: value.NameZh, DescEn: value.DescEn, DescZh: value.DescZh,
		DisplayNameLocs: value.DisplayNameLocs, DescriptionLocs: value.DescriptionLocs,
		EnableEquipments: value.EnableEquipments, TechLine: value.TechLine, Folder: value.Folder,
		StartYear: value.StartYear, PositionX: value.PositionX, PositionY: value.PositionY,
	}
	for _, placement := range value.FolderPlacements {
		result.FolderPlacements = append(result.FolderPlacements, techlifecycle.FolderPlacement{
			Folder: strings.TrimSpace(stringFrom(placement["folder"])),
			X:      folderPlacementFloat(placement["x"]), Y: folderPlacementFloat(placement["y"]),
		})
	}
	for _, path := range value.Path {
		if target := strings.TrimSpace(stringFrom(path["leads_to_tech"])); target != "" {
			result.Path = append(result.Path, techlifecycle.PathEdge{LeadsToTechnology: target})
		}
	}
	return result
}

func lifecycleState(projectID int64, record project.Row, rows func(string) []project.Row) (techlifecycle.State, error) {
	state := techlifecycle.State{Projects: []int64{projectID}, Branches: []techlifecycle.Branch{}, Technologies: []techlifecycle.Technology{}, Equipments: []techlifecycle.Equipment{}, ExternalReferences: []techlifecycle.ExternalReference{}}
	manifest, err := recordManifest(record)
	if err != nil {
		return state, err
	}
	for _, branch := range manifest.Branches {
		state.Branches = append(state.Branches, techlifecycle.Branch{
			ProjectID: projectID, ID: branch.ID, ProfileID: manifest.Profile,
			BaselineSHA256: manifest.BaselineSHA256, SchemaVersion: manifest.SchemaVersion,
			Surface: branch.Surface, Folder: branch.Folder, Slot: branch.Slot,
			RootTechnologyID: branch.RootTechnologyID, TechnologyIDs: append([]string(nil), branch.TechnologyIDs...),
		})
	}
	for _, row := range rows(table) {
		technology := lifecycleTechnology(techFromRow(row))
		technology.ProjectID = projectID
		state.Technologies = append(state.Technologies, technology)
	}
	equipmentIDs := []string{}
	for _, row := range rows("equipments") {
		equipmentIDs = append(equipmentIDs, text(row, "equipment_id", ""))
	}
	sort.Strings(equipmentIDs)
	for _, equipmentID := range equipmentIDs {
		state.Equipments = append(state.Equipments, techlifecycle.Equipment{ProjectID: projectID, EquipmentID: equipmentID})
	}
	seen := map[string]bool{}
	for _, source := range relationSources(rows) {
		for _, targetID := range technologyRefs(source.text) {
			if strings.TrimSpace(targetID) == "" {
				continue
			}
			sourceID := strings.TrimSpace(source.name)
			if sourceID == "" {
				sourceID = strconv.FormatInt(source.id, 10)
			}
			key := source.kind + "\x00" + sourceID + "\x00" + source.field + "\x00" + targetID
			if seen[key] {
				continue
			}
			seen[key] = true
			state.ExternalReferences = append(state.ExternalReferences, techlifecycle.ExternalReference{
				ProjectID: projectID, SourceKind: source.kind, SourceID: sourceID,
				Field: source.field, TargetTechnologyID: targetID,
			})
		}
	}
	return state, nil
}

func readLifecycleState(p *project.Project) (techlifecycle.State, error) {
	loaded := map[string][]project.Row{}
	for _, name := range lifecycleTables {
		rows, err := p.Rows(name)
		if err != nil {
			return techlifecycle.State{}, err
		}
		loaded[name] = rows
	}
	return lifecycleState(p.ID(), p.Record(), func(name string) []project.Row { return loaded[name] })
}

type scriptSource struct {
	kind, name, field, text string
	id                      int64
}

func relationSources(rows func(string) []project.Row) []scriptSource {
	var out []scriptSource
	emit := func(kind string, row project.Row, name string, fields ...string) {
		for _, field := range fields {
			if value := text(row, field, ""); value != "" {
				out = append(out, scriptSource{kind, name, field, value, project.Int64(row["id"])})
			}
		}
	}
	nodesByTree := map[int64][]project.Row{}
	for _, node := range rows("focus_nodes") {
		treeID := project.Int64(node["focus_tree_id"])
		nodesByTree[treeID] = append(nodesByTree[treeID], node)
	}
	for _, tree := range rows("focus_trees") {
		for _, node := range nodesByTree[project.Int64(tree["id"])] {
			emit("focus", node, text(node, "focus_id", ""), "completion_reward", "select_effect", "available", "bypass", "cancel")
		}
	}
	for _, event := range rows("events") {
		name := text(event, "event_id", "")
		emit("event", event, name, "trigger", "immediate", "after", "mean_time_to_happen")
		var options []map[string]any
		decodeColumn(event["options"], &options)
		for index, option := range options {
			for _, field := range []string{"trigger", "effects", "hidden_effects"} {
				if value, ok := option[field].(string); ok && value != "" {
					out = append(out, scriptSource{"event", name, "option[" + strconv.Itoa(index) + "]." + field, value, project.Int64(event["id"])})
				}
			}
		}
	}
	for _, decision := range rows("decisions") {
		emit("decision", decision, text(decision, "decision_id", ""), "complete_effect", "remove_effect", "timeout_effect", "cancel_effect",
			"allowed", "available", "visible", "cancel_trigger", "activation", "ai_will_do")
	}
	for _, idea := range rows("ideas") {
		emit("idea", idea, text(idea, "idea_id", ""), "on_add", "on_remove", "do_effect", "allowed", "available",
			"visible", "allowed_civil_war", "allowed_to_remove", "cancel")
	}
	for _, character := range rows("characters") {
		name := text(character, "tag", "")
		if name == "" {
			name = text(character, "name", "")
		}
		emit("character", character, name, "advisor_on_add", "advisor_on_remove", "advisor_available", "advisor_visible", "advisor_modifier")
	}
	return out
}

func technologyRefs(script string) []string {
	if strings.TrimSpace(script) == "" {
		return nil
	}
	block, recoveries, err := paradox.Parse(script)
	if err != nil || block == nil {
		return nil
	}
	for _, recovery := range recoveries {
		if recovery.Code == paradox.RecoveryUnclosedBlock {
			return nil
		}
	}
	var out []string
	var walk func([]*paradox.Entry)
	walk = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry.Key != nil {
				switch *entry.Key {
				case "has_tech", "can_research":
					if entry.Block == nil {
						out = append(out, entry.Scalar.StrValue())
					}
				case "set_technology":
					if entry.Block != nil {
						for _, child := range entry.Block.Entries {
							if child.Key != nil && *child.Key != "popup" {
								out = append(out, *child.Key)
							}
						}
					}
				}
			}
			if entry.Block != nil {
				walk(entry.Block.Entries)
			}
		}
	}
	walk(block.Entries)
	return out
}

func writeLifecycleError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var contract techlifecycle.ContractError
	if errors.As(err, &contract) {
		status := http.StatusUnprocessableEntity
		switch contract.Code {
		case techlifecycle.CodeProjectNotFound, techlifecycle.CodeBranchNotFound,
			techlifecycle.CodeBranchProjectMismatch, techlifecycle.CodeBranchMemberMissing,
			techlifecycle.CodeBranchMemberProject:
			status = http.StatusNotFound
		}
		apiutil.JSON(w, status, contract)
		return true
	}
	var blocked techlifecycle.DeleteBlockedError
	if errors.As(err, &blocked) {
		apiutil.JSON(w, http.StatusConflict, blocked)
		return true
	}
	detail(w, http.StatusServiceUnavailable, "technology surface unavailable")
	return true
}

func (h *handlers) listBranches(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	state, err := readLifecycleState(p)
	if writeLifecycleError(w, err) {
		return
	}
	result, err := techlifecycle.List(state, p.ID())
	if writeLifecycleError(w, err) {
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}

func (h *handlers) getBranch(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	state, err := readLifecycleState(p)
	if writeLifecycleError(w, err) {
		return
	}
	result, err := techlifecycle.Get(state, p.ID(), r.PathValue("branch"))
	if writeLifecycleError(w, err) {
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}

func (h *handlers) updateBranch(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	branchID, projectID := r.PathValue("branch"), p.ID()
	var input techlifecycle.UpdateRequest
	if err := decodeBody(w, r, apiutil.MaxJSONBody, false, &input); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology branch update payload")
		return
	}
	err := p.Write(lifecycleTables, func(tx *project.Tx) error {
		state, err := lifecycleState(projectID, tx.Record(), tx.Rows)
		if err != nil {
			return err
		}
		plan, err := techlifecycle.Update(state, projectID, branchID, input)
		if err != nil {
			return err
		}
		return applyLifecycleUpdate(tx, plan)
	})
	if writeLifecycleError(w, err) {
		return
	}
	state, err := readLifecycleState(p)
	if writeLifecycleError(w, err) {
		return
	}
	result, err := techlifecycle.Get(state, p.ID(), branchID)
	if writeLifecycleError(w, err) {
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}

func applyLifecycleUpdate(tx *project.Tx, plan techlifecycle.UpdatePlan) error {
	if len(plan.ChangedFields) == 0 {
		return nil
	}
	values := map[string]any{
		"name": plan.Mutable.Name, "name_en": plan.Mutable.NameEn, "name_zh": plan.Mutable.NameZh,
		"desc_en": plan.Mutable.DescEn, "desc_zh": plan.Mutable.DescZh,
		"display_name_locs": jsonValue(plan.Mutable.DisplayNameLocs), "description_locs": jsonValue(plan.Mutable.DescriptionLocs),
		"enable_equipments": jsonValue(plan.Mutable.EnableEquipments),
	}
	for _, field := range plan.ChangedFields {
		if _, ok := values[field]; !ok {
			return fmt.Errorf("unsupported technology lifecycle update field %q", field)
		}
	}
	row := tx.Find(table, plan.TechnologyDatabaseID)
	if row == nil || text(row, "tech_id", "") != plan.RootTechnologyID {
		return fmt.Errorf("technology lifecycle update matched no technology row")
	}
	_, err := tx.Update(table, plan.TechnologyDatabaseID, func(row project.Row) {
		for _, field := range plan.ChangedFields {
			row[field] = values[field]
		}
	})
	return err
}

func (h *handlers) deleteBranch(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}

	branchID, projectID := r.PathValue("branch"), p.ID()
	err := p.Write(lifecycleTables, func(tx *project.Tx) error {
		state, err := lifecycleState(projectID, tx.Record(), tx.Rows)
		if err != nil {
			return err
		}
		plan, err := techlifecycle.Delete(state, projectID, branchID)
		if err != nil {
			return err
		}
		return applyLifecycleDelete(tx, plan)
	})
	if writeLifecycleError(w, err) {
		return
	}
	apiutil.NoContent(w)
}

func applyLifecycleDelete(tx *project.Tx, plan techlifecycle.DeletePlan) error {
	manifest, err := recordManifest(tx.Record())
	if err != nil {
		return err
	}
	kept := make([]techsurface.Branch, 0, len(manifest.Branches))
	removed := 0
	for _, branch := range manifest.Branches {
		if strings.TrimSpace(branch.ID) == plan.BranchID {
			removed++
			continue
		}
		kept = append(kept, branch)
	}
	if removed != 1 {
		return fmt.Errorf("technology lifecycle delete matched %d manifest branches", removed)
	}
	deleted := tx.Delete(table, func(row project.Row) bool {
		return project.Int64(row["id"]) == plan.TechnologyDatabaseID && text(row, "tech_id", "") == plan.RootTechnologyID
	})
	if deleted != 1 {
		return fmt.Errorf("technology lifecycle delete matched %d technology rows", deleted)
	}
	manifest.Branches = kept
	value := manifestValue(manifest)
	if len(kept) == 0 {

		value = map[string]any{}
	}
	tx.SetRecord(func(record project.Row) {
		record["tech_surface_manifest"] = value
		record["updated_at"] = workbench.NowTimestamp()
	})
	return nil
}
