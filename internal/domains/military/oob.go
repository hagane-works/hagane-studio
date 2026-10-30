package military

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	airWingGroupTable = "oob_air_wing_groups"
	airWingTable      = "oob_air_wings"
	fleetTable        = "oob_fleets"
	taskForceTable    = "oob_task_forces"
	shipTable         = "oob_ships"
)

var (
	errAirWingGroupMissing = apiutil.NotFoundError("Air wing group not found")
	errAirWingMissing      = apiutil.NotFoundError("Air wing not found")
	errFleetMissing        = apiutil.NotFoundError("Fleet not found")
	errTaskForceMissing    = apiutil.NotFoundError("Task force not found")
	errShipMissing         = apiutil.NotFoundError("Ship not found")
)

func validateOOBToken(value string, allowEmpty bool) (string, error) {
	token := strings.TrimSpace(value)
	if _, err := importsvc.ParseHOI4ScalarToken(token, allowEmpty); err != nil {
		return "", detailError(http.StatusUnprocessableEntity, err.Error())
	}
	return token, nil
}

func normaliseOOBTop(oobFile, sourceFile, countryTag string) (string, string, string) {
	oobFile = strings.ReplaceAll(strings.TrimSpace(oobFile), "\\", "/")
	sourceFile = strings.ReplaceAll(strings.TrimSpace(sourceFile), "\\", "/")
	tag := upper(countryTag)
	if len(tag) > 10 {
		tag = tag[:10]
	}
	return oobFile, sourceFile, tag
}

type AirWingGroup struct {
	ID         int64            `json:"id"`
	ProjectID  int64            `json:"project_id"`
	OobFile    string           `json:"oob_file"`
	SourceFile string           `json:"source_file"`
	CountryTag string           `json:"country_tag"`
	Location   string           `json:"location"`
	Ordering   int              `json:"ordering"`
	Layout     []map[string]any `json:"layout"`
	Source     string           `json:"source"`

	Warnings []string `json:"warnings"`
}

type AirWing struct {
	ID            int64            `json:"id"`
	ProjectID     int64            `json:"project_id"`
	GroupID       int64            `json:"group_id"`
	EquipmentType string           `json:"equipment_type"`
	Name          string           `json:"name"`
	Owner         string           `json:"owner"`
	Amount        string           `json:"amount"`
	Ordering      int              `json:"ordering"`
	BodyAst       []map[string]any `json:"body_ast"`
	Source        string           `json:"source"`
}

type Fleet struct {
	ID         int64            `json:"id"`
	ProjectID  int64            `json:"project_id"`
	OobFile    string           `json:"oob_file"`
	SourceFile string           `json:"source_file"`
	CountryTag string           `json:"country_tag"`
	Name       string           `json:"name"`
	NavalBase  string           `json:"naval_base"`
	Ordering   int              `json:"ordering"`
	Layout     []map[string]any `json:"layout"`
	Source     string           `json:"source"`
	Warnings   []string         `json:"warnings"`
}

type TaskForce struct {
	ID        int64            `json:"id"`
	ProjectID int64            `json:"project_id"`
	FleetID   int64            `json:"fleet_id"`
	Name      string           `json:"name"`
	Location  string           `json:"location"`
	Ordering  int              `json:"ordering"`
	Layout    []map[string]any `json:"layout"`
	Source    string           `json:"source"`
	Warnings  []string         `json:"warnings"`
}

type Ship struct {
	ID                    int64            `json:"id"`
	ProjectID             int64            `json:"project_id"`
	TaskForceID           int64            `json:"task_force_id"`
	Name                  string           `json:"name"`
	Definition            string           `json:"definition"`
	StartExperienceFactor string           `json:"start_experience_factor"`
	EquipmentType         string           `json:"equipment_type"`
	Amount                string           `json:"amount"`
	Owner                 string           `json:"owner"`
	VersionName           string           `json:"version_name"`
	Ordering              int              `json:"ordering"`
	BodyAst               []map[string]any `json:"body_ast"`
	Source                string           `json:"source"`
}

type OOBSnapshot struct {
	AirWingGroups []AirWingGroup `json:"air_wing_groups"`
	AirWings      []AirWing      `json:"air_wings"`
	Fleets        []Fleet        `json:"fleets"`
	TaskForces    []TaskForce    `json:"task_forces"`
	Ships         []Ship         `json:"ships"`
}

func airWingGroupFromRow(row project.Row) AirWingGroup {
	var g AirWingGroup
	fromRow(row, nil, &g)
	if g.Layout == nil {
		g.Layout = []map[string]any{}
	}
	g.Warnings = []string{}
	return g
}

func airWingFromRow(row project.Row) AirWing {
	var w AirWing
	fromRow(row, nil, &w)
	if w.BodyAst == nil {
		w.BodyAst = []map[string]any{}
	}
	return w
}

func fleetFromRow(row project.Row) Fleet {
	var f Fleet
	fromRow(row, nil, &f)
	if f.Layout == nil {
		f.Layout = []map[string]any{}
	}
	f.Warnings = []string{}
	return f
}

func taskForceFromRow(row project.Row) TaskForce {
	var t TaskForce
	fromRow(row, nil, &t)
	if t.Layout == nil {
		t.Layout = []map[string]any{}
	}
	t.Warnings = []string{}
	return t
}

func shipFromRow(row project.Row) Ship {
	var s Ship
	fromRow(row, nil, &s)
	if s.BodyAst == nil {
		s.BodyAst = []map[string]any{}
	}
	return s
}

func sortedRows(p *project.Project, table string, keep func(project.Row) bool) ([]project.Row, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	out := []project.Row{}
	for _, row := range rows {
		if keep == nil || keep(row) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := number(out[i]["ordering"]), number(out[j]["ordering"])
		if a != b {
			return a < b
		}
		return project.Int64(out[i]["id"]) < project.Int64(out[j]["id"])
	})
	return out, nil
}

func annotateOOBWarnings(p *project.Project, groups []*AirWingGroup, fleets []*Fleet, taskForces []*TaskForce) error {
	if len(groups) == 0 && len(fleets) == 0 && len(taskForces) == 0 {
		return nil
	}
	index, err := loadDeploymentValidationIndex(p, true, false)
	if err != nil {
		return err
	}
	for _, g := range groups {
		g.Warnings = oobTokenOwnershipWarnings(stateOwnership, oobTagOf(g.CountryTag, g.OobFile), g.Location, index)
	}
	for _, f := range fleets {
		f.Warnings = oobTokenOwnershipWarnings(provinceOwnership, oobTagOf(f.CountryTag, f.OobFile), f.NavalBase, index)
	}
	if len(taskForces) > 0 {

		rows, err := p.Rows(fleetTable)
		if err != nil {
			return err
		}
		fleetTags := map[int64]string{}
		for _, row := range rows {
			fleetTags[project.Int64(row["id"])] = oobTagOf(text(row, "country_tag"), text(row, "oob_file"))
		}
		for _, t := range taskForces {
			t.Warnings = oobTokenOwnershipWarnings(provinceOwnership, fleetTags[t.FleetID], t.Location, index)
		}
	}
	return nil
}

func ptrsOf[T any](items []T) []*T {
	out := make([]*T, 0, len(items))
	for i := range items {
		out = append(out, &items[i])
	}
	return out
}

func oobSnapshot(p *project.Project, oobFile, sourceFile *string) (OOBSnapshot, error) {
	snap := OOBSnapshot{
		AirWingGroups: []AirWingGroup{}, AirWings: []AirWing{},
		Fleets: []Fleet{}, TaskForces: []TaskForce{}, Ships: []Ship{},
	}
	top := func(row project.Row) bool {
		return (oobFile == nil || stringEquals(row["oob_file"], *oobFile)) &&
			(sourceFile == nil || stringEquals(row["source_file"], *sourceFile))
	}
	groups, err := sortedRows(p, airWingGroupTable, top)
	if err != nil {
		return snap, err
	}
	groupIDs := map[int64]bool{}
	for _, row := range groups {
		snap.AirWingGroups = append(snap.AirWingGroups, airWingGroupFromRow(row))
		groupIDs[project.Int64(row["id"])] = true
	}
	fleets, err := sortedRows(p, fleetTable, top)
	if err != nil {
		return snap, err
	}
	fleetIDs := map[int64]bool{}
	for _, row := range fleets {
		snap.Fleets = append(snap.Fleets, fleetFromRow(row))
		fleetIDs[project.Int64(row["id"])] = true
	}
	if len(groupIDs) > 0 {
		wings, err := sortedRows(p, airWingTable, func(row project.Row) bool { return groupIDs[project.Int64(row["group_id"])] })
		if err != nil {
			return snap, err
		}
		for _, row := range wings {
			snap.AirWings = append(snap.AirWings, airWingFromRow(row))
		}
	}
	taskForceIDs := map[int64]bool{}
	if len(fleetIDs) > 0 {
		forces, err := sortedRows(p, taskForceTable, func(row project.Row) bool { return fleetIDs[project.Int64(row["fleet_id"])] })
		if err != nil {
			return snap, err
		}
		for _, row := range forces {
			snap.TaskForces = append(snap.TaskForces, taskForceFromRow(row))
			taskForceIDs[project.Int64(row["id"])] = true
		}
	}
	if len(taskForceIDs) > 0 {
		ships, err := sortedRows(p, shipTable, func(row project.Row) bool { return taskForceIDs[project.Int64(row["task_force_id"])] })
		if err != nil {
			return snap, err
		}
		for _, row := range ships {
			snap.Ships = append(snap.Ships, shipFromRow(row))
		}
	}
	err = annotateOOBWarnings(p, ptrsOf(snap.AirWingGroups), ptrsOf(snap.Fleets), ptrsOf(snap.TaskForces))
	return snap, err
}

func filterEquals(column string, value any) func(project.Row) bool {
	if value == nil {
		return nil
	}
	return func(row project.Row) bool {
		switch want := value.(type) {
		case string:
			return stringEquals(row[column], want)
		case int64:
			return row[column] != nil && project.Int64(row[column]) == want
		}
		return false
	}
}

func listOOB[T any](p *project.Project, table string, keep func(project.Row) bool, convert func(project.Row) T) ([]T, error) {
	rows, err := sortedRows(p, table, keep)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		out = append(out, convert(row))
	}
	return out, nil
}

func getOOB[T any](p *project.Project, table string, id int64, missing error, convert func(project.Row) T) (T, error) {
	var zero T
	rows, err := p.Rows(table)
	if err != nil {
		return zero, err
	}
	row := findRow(rows, id)
	if row == nil {
		return zero, missing
	}
	return convert(row), nil
}

var oobChildren = map[string][]struct{ table, parentColumn string }{
	airWingGroupTable: {{airWingTable, "group_id"}},
	fleetTable:        {{taskForceTable, "fleet_id"}},
	taskForceTable:    {{shipTable, "task_force_id"}},
}

func oobCascadeTables(table string) []string {
	out := []string{table}
	for _, child := range oobChildren[table] {
		out = append(out, oobCascadeTables(child.table)...)
	}
	return out
}

func deleteOOBRows(tx *project.Tx, table string, match func(project.Row) bool) int {
	ids := map[int64]bool{}
	for _, row := range tx.Rows(table) {
		if match(row) {
			ids[project.Int64(row["id"])] = true
		}
	}
	if len(ids) == 0 {
		return 0
	}
	for _, child := range oobChildren[table] {
		column := child.parentColumn
		deleteOOBRows(tx, child.table, func(row project.Row) bool { return ids[project.Int64(row[column])] })
	}
	return tx.Delete(table, func(row project.Row) bool { return ids[project.Int64(row["id"])] })
}

func deleteOOB(p *project.Project, table string, id int64, missing error) error {
	return p.Write(oobCascadeTables(table), func(tx *project.Tx) error {
		if deleteOOBRows(tx, table, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return missing
		}
		return nil
	})
}

func requireRow(tx *project.Tx, table string, id int64, missing error) error {
	if tx.Find(table, id) == nil {
		return missing
	}
	return nil
}

var errOOBWrite = errors.New("null value in column violates not-null constraint")

func applyOOBUpdate(tx *project.Tx, table string, id int64, current project.Row, changes map[string]any, source workbench.Optional[string]) (project.Row, error) {
	sourceChange(changes, current, source)
	if len(changes) == 0 {
		return current, nil
	}
	if nullViolation(table, changes) {
		return nil, errOOBWrite
	}
	return tx.Update(table, id, func(row project.Row) {
		for column, value := range changes {
			row[column] = value
		}
	})
}

type airWingGroupInput struct {
	OobFile    string           `json:"oob_file"`
	SourceFile string           `json:"source_file"`
	CountryTag string           `json:"country_tag"`
	Location   *string          `json:"location"`
	Ordering   int              `json:"ordering"`
	Layout     []map[string]any `json:"layout"`
	Source     *string          `json:"source"`
}

type airWingGroupPatch struct {
	OobFile    workbench.Optional[string]           `json:"oob_file"`
	SourceFile workbench.Optional[string]           `json:"source_file"`
	CountryTag workbench.Optional[string]           `json:"country_tag"`
	Location   workbench.Optional[string]           `json:"location"`
	Ordering   workbench.Optional[int]              `json:"ordering"`
	Layout     workbench.Optional[[]map[string]any] `json:"layout"`
	Source     workbench.Optional[string]           `json:"source"`
}

func getAirWingGroup(p *project.Project, id int64) (AirWingGroup, error) {
	g, err := getOOB(p, airWingGroupTable, id, errAirWingGroupMissing, airWingGroupFromRow)
	if err != nil {
		return g, err
	}
	err = annotateOOBWarnings(p, []*AirWingGroup{&g}, nil, nil)
	return g, err
}

func createAirWingGroup(p *project.Project, in airWingGroupInput) (AirWingGroup, error) {
	if in.Location == nil {
		return AirWingGroup{}, detailError(http.StatusUnprocessableEntity, "location is required")
	}
	location, err := validateOOBToken(*in.Location, false)
	if err != nil {
		return AirWingGroup{}, err
	}
	oobFile, sourceFile, tag := normaliseOOBTop(in.OobFile, in.SourceFile, in.CountryTag)
	if in.Layout == nil {
		in.Layout = []map[string]any{}
	}
	var id int64
	err = p.Write([]string{airWingGroupTable}, func(tx *project.Tx) error {
		row := tx.Insert(airWingGroupTable, project.Row{
			"oob_file": oobFile, "source_file": sourceFile, "country_tag": tag,
			"location": location, "ordering": jsonValue(in.Ordering),
			"layout": jsonValue(in.Layout), "source": stringPtr(in.Source, "created"),
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return AirWingGroup{}, err
	}
	return getAirWingGroup(p, id)
}

func updateAirWingGroup(p *project.Project, id int64, patch airWingGroupPatch) (AirWingGroup, error) {
	err := p.Write([]string{airWingGroupTable}, func(tx *project.Tx) error {
		current := tx.Find(airWingGroupTable, id)
		if current == nil {
			return errAirWingGroupMissing
		}
		changes := map[string]any{}
		if patch.Location.Set {
			location, err := validateOOBToken(optStr(patch.Location), false)
			if err != nil {
				return err
			}
			changes["location"] = location
		}
		setOOBTop(changes, patch.OobFile, patch.SourceFile, patch.CountryTag)
		if patch.Ordering.Set {
			changes["ordering"] = opt(patch.Ordering)
		}
		if patch.Layout.Set {
			changes["layout"] = opt(patch.Layout)
		}
		_, err := applyOOBUpdate(tx, airWingGroupTable, id, current, changes, patch.Source)
		return err
	})
	if err != nil {
		return AirWingGroup{}, err
	}
	return getAirWingGroup(p, id)
}

func setOOBTop(changes map[string]any, oobFile, sourceFile, countryTag workbench.Optional[string]) {
	if !oobFile.Set && !sourceFile.Set && !countryTag.Set {
		return
	}
	oob, source, tag := normaliseOOBTop(optStr(oobFile), optStr(sourceFile), optStr(countryTag))
	if oobFile.Set {
		changes["oob_file"] = oob
	}
	if sourceFile.Set {
		changes["source_file"] = source
	}
	if countryTag.Set {
		changes["country_tag"] = tag
	}
}

type airWingInput struct {
	GroupID       int64            `json:"group_id"`
	EquipmentType *string          `json:"equipment_type"`
	Name          string           `json:"name"`
	Owner         string           `json:"owner"`
	Amount        string           `json:"amount"`
	Ordering      int              `json:"ordering"`
	BodyAst       []map[string]any `json:"body_ast"`
	Source        *string          `json:"source"`
}

type airWingPatch struct {
	GroupID       workbench.Optional[int64]            `json:"group_id"`
	EquipmentType workbench.Optional[string]           `json:"equipment_type"`
	Name          workbench.Optional[string]           `json:"name"`
	Owner         workbench.Optional[string]           `json:"owner"`
	Amount        workbench.Optional[string]           `json:"amount"`
	Ordering      workbench.Optional[int]              `json:"ordering"`
	BodyAst       workbench.Optional[[]map[string]any] `json:"body_ast"`
	Source        workbench.Optional[string]           `json:"source"`
}

func createAirWing(p *project.Project, in airWingInput) (AirWing, error) {
	var created AirWing
	err := p.Write([]string{airWingGroupTable, airWingTable}, func(tx *project.Tx) error {
		if err := requireRow(tx, airWingGroupTable, in.GroupID, errAirWingGroupMissing); err != nil {
			return err
		}
		if in.EquipmentType == nil {
			return detailError(http.StatusUnprocessableEntity, "equipment_type is required")
		}
		equipmentType, err := validateOOBToken(*in.EquipmentType, false)
		if err != nil {
			return err
		}
		owner, err := validateOOBToken(in.Owner, true)
		if err != nil {
			return err
		}
		amount, err := validateOOBToken(in.Amount, true)
		if err != nil {
			return err
		}
		if in.BodyAst == nil {
			in.BodyAst = []map[string]any{}
		}
		created = airWingFromRow(tx.Insert(airWingTable, project.Row{
			"group_id": in.GroupID, "equipment_type": equipmentType, "name": in.Name,
			"owner": owner, "amount": amount, "ordering": jsonValue(in.Ordering),
			"body_ast": jsonValue(in.BodyAst), "source": stringPtr(in.Source, "created"),
		}))
		return nil
	})
	return created, err
}

func updateAirWing(p *project.Project, id int64, patch airWingPatch) (AirWing, error) {
	var updated AirWing
	err := p.Write([]string{airWingGroupTable, airWingTable}, func(tx *project.Tx) error {
		current := tx.Find(airWingTable, id)
		if current == nil {
			return errAirWingMissing
		}
		if patch.GroupID.Set && patch.GroupID.Value != nil {
			if err := requireRow(tx, airWingGroupTable, *patch.GroupID.Value, errAirWingGroupMissing); err != nil {
				return err
			}
		}
		changes := map[string]any{}
		if patch.GroupID.Set {
			changes["group_id"] = opt(patch.GroupID)
		}
		if patch.EquipmentType.Set {
			v, err := validateOOBToken(optStr(patch.EquipmentType), false)
			if err != nil {
				return err
			}
			changes["equipment_type"] = v
		}
		if patch.Name.Set {
			changes["name"] = opt(patch.Name)
		}
		if err := setTokens(changes, map[string]workbench.Optional[string]{"owner": patch.Owner, "amount": patch.Amount}, []string{"owner", "amount"}); err != nil {
			return err
		}
		if patch.Ordering.Set {
			changes["ordering"] = opt(patch.Ordering)
		}
		if patch.BodyAst.Set {
			changes["body_ast"] = opt(patch.BodyAst)
		}
		row, err := applyOOBUpdate(tx, airWingTable, id, current, changes, patch.Source)
		if err != nil {
			return err
		}
		updated = airWingFromRow(row)
		return nil
	})
	return updated, err
}

func setTokens(changes map[string]any, fields map[string]workbench.Optional[string], order []string) error {
	for _, column := range order {
		field := fields[column]
		if !field.Set {
			continue
		}
		v, err := validateOOBToken(optStr(field), true)
		if err != nil {
			return err
		}
		changes[column] = v
	}
	return nil
}

type fleetInput struct {
	OobFile    string           `json:"oob_file"`
	SourceFile string           `json:"source_file"`
	CountryTag string           `json:"country_tag"`
	Name       string           `json:"name"`
	NavalBase  string           `json:"naval_base"`
	Ordering   int              `json:"ordering"`
	Layout     []map[string]any `json:"layout"`
	Source     *string          `json:"source"`
}

type fleetPatch struct {
	OobFile    workbench.Optional[string]           `json:"oob_file"`
	SourceFile workbench.Optional[string]           `json:"source_file"`
	CountryTag workbench.Optional[string]           `json:"country_tag"`
	Name       workbench.Optional[string]           `json:"name"`
	NavalBase  workbench.Optional[string]           `json:"naval_base"`
	Ordering   workbench.Optional[int]              `json:"ordering"`
	Layout     workbench.Optional[[]map[string]any] `json:"layout"`
	Source     workbench.Optional[string]           `json:"source"`
}

func getFleet(p *project.Project, id int64) (Fleet, error) {
	f, err := getOOB(p, fleetTable, id, errFleetMissing, fleetFromRow)
	if err != nil {
		return f, err
	}
	err = annotateOOBWarnings(p, nil, []*Fleet{&f}, nil)
	return f, err
}

func createFleet(p *project.Project, in fleetInput) (Fleet, error) {
	navalBase, err := validateOOBToken(in.NavalBase, true)
	if err != nil {
		return Fleet{}, err
	}
	oobFile, sourceFile, tag := normaliseOOBTop(in.OobFile, in.SourceFile, in.CountryTag)
	if in.Layout == nil {
		in.Layout = []map[string]any{}
	}
	var id int64
	err = p.Write([]string{fleetTable}, func(tx *project.Tx) error {
		row := tx.Insert(fleetTable, project.Row{
			"oob_file": oobFile, "source_file": sourceFile, "country_tag": tag,
			"name": in.Name, "naval_base": navalBase, "ordering": jsonValue(in.Ordering),
			"layout": jsonValue(in.Layout), "source": stringPtr(in.Source, "created"),
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return Fleet{}, err
	}
	return getFleet(p, id)
}

func updateFleet(p *project.Project, id int64, patch fleetPatch) (Fleet, error) {
	err := p.Write([]string{fleetTable}, func(tx *project.Tx) error {
		current := tx.Find(fleetTable, id)
		if current == nil {
			return errFleetMissing
		}
		changes := map[string]any{}
		if err := setTokens(changes, map[string]workbench.Optional[string]{"naval_base": patch.NavalBase}, []string{"naval_base"}); err != nil {
			return err
		}
		setOOBTop(changes, patch.OobFile, patch.SourceFile, patch.CountryTag)
		if patch.Name.Set {
			changes["name"] = opt(patch.Name)
		}
		if patch.Ordering.Set {
			changes["ordering"] = opt(patch.Ordering)
		}
		if patch.Layout.Set {
			changes["layout"] = opt(patch.Layout)
		}
		_, err := applyOOBUpdate(tx, fleetTable, id, current, changes, patch.Source)
		return err
	})
	if err != nil {
		return Fleet{}, err
	}
	return getFleet(p, id)
}

type taskForceInput struct {
	FleetID  int64            `json:"fleet_id"`
	Name     string           `json:"name"`
	Location string           `json:"location"`
	Ordering int              `json:"ordering"`
	Layout   []map[string]any `json:"layout"`
	Source   *string          `json:"source"`
}

type taskForcePatch struct {
	FleetID  workbench.Optional[int64]            `json:"fleet_id"`
	Name     workbench.Optional[string]           `json:"name"`
	Location workbench.Optional[string]           `json:"location"`
	Ordering workbench.Optional[int]              `json:"ordering"`
	Layout   workbench.Optional[[]map[string]any] `json:"layout"`
	Source   workbench.Optional[string]           `json:"source"`
}

func getTaskForce(p *project.Project, id int64) (TaskForce, error) {
	t, err := getOOB(p, taskForceTable, id, errTaskForceMissing, taskForceFromRow)
	if err != nil {
		return t, err
	}
	err = annotateOOBWarnings(p, nil, nil, []*TaskForce{&t})
	return t, err
}

func createTaskForce(p *project.Project, in taskForceInput) (TaskForce, error) {
	var id int64
	err := p.Write([]string{fleetTable, taskForceTable}, func(tx *project.Tx) error {
		if err := requireRow(tx, fleetTable, in.FleetID, errFleetMissing); err != nil {
			return err
		}
		location, err := validateOOBToken(in.Location, true)
		if err != nil {
			return err
		}
		if in.Layout == nil {
			in.Layout = []map[string]any{}
		}
		row := tx.Insert(taskForceTable, project.Row{
			"fleet_id": in.FleetID, "name": in.Name, "location": location,
			"ordering": jsonValue(in.Ordering), "layout": jsonValue(in.Layout),
			"source": stringPtr(in.Source, "created"),
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return TaskForce{}, err
	}
	return getTaskForce(p, id)
}

func updateTaskForce(p *project.Project, id int64, patch taskForcePatch) (TaskForce, error) {
	err := p.Write([]string{fleetTable, taskForceTable}, func(tx *project.Tx) error {
		current := tx.Find(taskForceTable, id)
		if current == nil {
			return errTaskForceMissing
		}
		if patch.FleetID.Set && patch.FleetID.Value != nil {
			if err := requireRow(tx, fleetTable, *patch.FleetID.Value, errFleetMissing); err != nil {
				return err
			}
		}
		changes := map[string]any{}
		if patch.FleetID.Set {
			changes["fleet_id"] = opt(patch.FleetID)
		}
		if patch.Name.Set {
			changes["name"] = opt(patch.Name)
		}
		if err := setTokens(changes, map[string]workbench.Optional[string]{"location": patch.Location}, []string{"location"}); err != nil {
			return err
		}
		if patch.Ordering.Set {
			changes["ordering"] = opt(patch.Ordering)
		}
		if patch.Layout.Set {
			changes["layout"] = opt(patch.Layout)
		}
		_, err := applyOOBUpdate(tx, taskForceTable, id, current, changes, patch.Source)
		return err
	})
	if err != nil {
		return TaskForce{}, err
	}
	return getTaskForce(p, id)
}

type shipInput struct {
	TaskForceID           int64            `json:"task_force_id"`
	Name                  string           `json:"name"`
	Definition            string           `json:"definition"`
	StartExperienceFactor string           `json:"start_experience_factor"`
	EquipmentType         string           `json:"equipment_type"`
	Amount                string           `json:"amount"`
	Owner                 string           `json:"owner"`
	VersionName           string           `json:"version_name"`
	Ordering              int              `json:"ordering"`
	BodyAst               []map[string]any `json:"body_ast"`
	Source                *string          `json:"source"`
}

type shipPatch struct {
	TaskForceID           workbench.Optional[int64]            `json:"task_force_id"`
	Name                  workbench.Optional[string]           `json:"name"`
	Definition            workbench.Optional[string]           `json:"definition"`
	StartExperienceFactor workbench.Optional[string]           `json:"start_experience_factor"`
	EquipmentType         workbench.Optional[string]           `json:"equipment_type"`
	Amount                workbench.Optional[string]           `json:"amount"`
	Owner                 workbench.Optional[string]           `json:"owner"`
	VersionName           workbench.Optional[string]           `json:"version_name"`
	Ordering              workbench.Optional[int]              `json:"ordering"`
	BodyAst               workbench.Optional[[]map[string]any] `json:"body_ast"`
	Source                workbench.Optional[string]           `json:"source"`
}

var shipTokenColumns = []string{"definition", "start_experience_factor", "amount", "owner"}

func createShip(p *project.Project, in shipInput) (Ship, error) {
	var created Ship
	err := p.Write([]string{taskForceTable, shipTable}, func(tx *project.Tx) error {
		if err := requireRow(tx, taskForceTable, in.TaskForceID, errTaskForceMissing); err != nil {
			return err
		}
		tokens := map[string]*string{
			"definition": &in.Definition, "start_experience_factor": &in.StartExperienceFactor,
			"amount": &in.Amount, "owner": &in.Owner,
		}
		for _, field := range shipTokenColumns {
			v, err := validateOOBToken(*tokens[field], true)
			if err != nil {
				return err
			}
			*tokens[field] = v
		}
		if in.BodyAst == nil {
			in.BodyAst = []map[string]any{}
		}
		created = shipFromRow(tx.Insert(shipTable, project.Row{
			"task_force_id": in.TaskForceID, "name": in.Name, "definition": in.Definition,
			"start_experience_factor": in.StartExperienceFactor, "equipment_type": in.EquipmentType,
			"amount": in.Amount, "owner": in.Owner, "version_name": in.VersionName,
			"ordering": jsonValue(in.Ordering), "body_ast": jsonValue(in.BodyAst),
			"source": stringPtr(in.Source, "created"),
		}))
		return nil
	})
	return created, err
}

func updateShip(p *project.Project, id int64, patch shipPatch) (Ship, error) {
	var updated Ship
	err := p.Write([]string{taskForceTable, shipTable}, func(tx *project.Tx) error {
		current := tx.Find(shipTable, id)
		if current == nil {
			return errShipMissing
		}
		if patch.TaskForceID.Set && patch.TaskForceID.Value != nil {
			if err := requireRow(tx, taskForceTable, *patch.TaskForceID.Value, errTaskForceMissing); err != nil {
				return err
			}
		}
		changes := map[string]any{}
		if patch.TaskForceID.Set {
			changes["task_force_id"] = opt(patch.TaskForceID)
		}
		if patch.Name.Set {
			changes["name"] = opt(patch.Name)
		}
		if err := setTokens(changes, map[string]workbench.Optional[string]{
			"definition": patch.Definition, "start_experience_factor": patch.StartExperienceFactor,
			"amount": patch.Amount, "owner": patch.Owner,
		}, shipTokenColumns); err != nil {
			return err
		}
		for column, field := range map[string]workbench.Optional[string]{"equipment_type": patch.EquipmentType, "version_name": patch.VersionName} {
			if field.Set {
				changes[column] = opt(field)
			}
		}
		if patch.Ordering.Set {
			changes["ordering"] = opt(patch.Ordering)
		}
		if patch.BodyAst.Set {
			changes["body_ast"] = opt(patch.BodyAst)
		}
		row, err := applyOOBUpdate(tx, shipTable, id, current, changes, patch.Source)
		if err != nil {
			return err
		}
		updated = shipFromRow(row)
		return nil
	})
	return updated, err
}

type oobEntity struct {
	missing string
	list    func(r *http.Request) (any, error)
	get     func(id int64) (any, error)
	create  func(r *http.Request) (any, error)
	update  func(r *http.Request, id int64) (any, error)
	remove  func(id int64) error
}

var errDecoded = errors.New("invalid OOB payload")

func decodeOOB[T any](r *http.Request) (T, error) {
	var in T
	if err := decodeBody(r, 8<<20, &in); err != nil {
		return in, errDecoded
	}
	return in, nil
}

func writeOOBError(w http.ResponseWriter, err error, op string) {
	var (
		status   *apiutil.StatusError
		notFound apiutil.NotFoundError
	)
	switch {
	case errors.Is(err, errDecoded):
		detail(w, http.StatusUnprocessableEntity, "Invalid OOB payload")
	case errors.As(err, &status), errors.As(err, &notFound):
		apiutil.Fail(w, err)
	default:
		log.Printf("%s: %v", op, err)
		detail(w, http.StatusInternalServerError,
			op+" failed: "+err.Error()+" — this is an internal error of the program (logged); the OOB store itself is reachable")
	}
}

func (h *handlers) oobEntities(p *project.Project) map[string]oobEntity {
	return map[string]oobEntity{
		"air-wing-groups": {
			missing: "Air wing group not found",
			list: func(r *http.Request) (any, error) {
				out, err := listOOB(p, airWingGroupTable, filterEquals("oob_file", strOrNil(queryPtr(r, "oob_file"))), airWingGroupFromRow)
				if err == nil {
					err = annotateOOBWarnings(p, ptrsOf(out), nil, nil)
				}
				return out, err
			},
			get: func(id int64) (any, error) { return getAirWingGroup(p, id) },
			create: func(r *http.Request) (any, error) {
				in, err := decodeOOB[airWingGroupInput](r)
				if err != nil {
					return nil, err
				}
				return createAirWingGroup(p, in)
			},
			update: func(r *http.Request, id int64) (any, error) {
				in, err := decodeOOB[airWingGroupPatch](r)
				if err != nil {
					return nil, err
				}
				return updateAirWingGroup(p, id, in)
			},
			remove: func(id int64) error { return deleteOOB(p, airWingGroupTable, id, errAirWingGroupMissing) },
		},
		"air-wings": {
			missing: "Air wing not found",
			list: func(r *http.Request) (any, error) {
				return listOOB(p, airWingTable, filterEquals("group_id", int64OrNil(queryInt64Ptr(r, "group_id"))), airWingFromRow)
			},
			get: func(id int64) (any, error) { return getOOB(p, airWingTable, id, errAirWingMissing, airWingFromRow) },
			create: func(r *http.Request) (any, error) {
				in, err := decodeOOB[airWingInput](r)
				if err != nil {
					return nil, err
				}
				return createAirWing(p, in)
			},
			update: func(r *http.Request, id int64) (any, error) {
				in, err := decodeOOB[airWingPatch](r)
				if err != nil {
					return nil, err
				}
				return updateAirWing(p, id, in)
			},
			remove: func(id int64) error { return deleteOOB(p, airWingTable, id, errAirWingMissing) },
		},
		"fleets": {
			missing: "Fleet not found",
			list: func(r *http.Request) (any, error) {
				out, err := listOOB(p, fleetTable, filterEquals("oob_file", strOrNil(queryPtr(r, "oob_file"))), fleetFromRow)
				if err == nil {
					err = annotateOOBWarnings(p, nil, ptrsOf(out), nil)
				}
				return out, err
			},
			get: func(id int64) (any, error) { return getFleet(p, id) },
			create: func(r *http.Request) (any, error) {
				in, err := decodeOOB[fleetInput](r)
				if err != nil {
					return nil, err
				}
				return createFleet(p, in)
			},
			update: func(r *http.Request, id int64) (any, error) {
				in, err := decodeOOB[fleetPatch](r)
				if err != nil {
					return nil, err
				}
				return updateFleet(p, id, in)
			},
			remove: func(id int64) error { return deleteOOB(p, fleetTable, id, errFleetMissing) },
		},
		"task-forces": {
			missing: "Task force not found",
			list: func(r *http.Request) (any, error) {
				out, err := listOOB(p, taskForceTable, filterEquals("fleet_id", int64OrNil(queryInt64Ptr(r, "fleet_id"))), taskForceFromRow)
				if err == nil {
					err = annotateOOBWarnings(p, nil, nil, ptrsOf(out))
				}
				return out, err
			},
			get: func(id int64) (any, error) { return getTaskForce(p, id) },
			create: func(r *http.Request) (any, error) {
				in, err := decodeOOB[taskForceInput](r)
				if err != nil {
					return nil, err
				}
				return createTaskForce(p, in)
			},
			update: func(r *http.Request, id int64) (any, error) {
				in, err := decodeOOB[taskForcePatch](r)
				if err != nil {
					return nil, err
				}
				return updateTaskForce(p, id, in)
			},
			remove: func(id int64) error { return deleteOOB(p, taskForceTable, id, errTaskForceMissing) },
		},
		"ships": {
			missing: "Ship not found",
			list: func(r *http.Request) (any, error) {
				return listOOB(p, shipTable, filterEquals("task_force_id", int64OrNil(queryInt64Ptr(r, "task_force_id"))), shipFromRow)
			},
			get: func(id int64) (any, error) { return getOOB(p, shipTable, id, errShipMissing, shipFromRow) },
			create: func(r *http.Request) (any, error) {
				in, err := decodeOOB[shipInput](r)
				if err != nil {
					return nil, err
				}
				return createShip(p, in)
			},
			update: func(r *http.Request, id int64) (any, error) {
				in, err := decodeOOB[shipPatch](r)
				if err != nil {
					return nil, err
				}
				return updateShip(p, id, in)
			},
			remove: func(id int64) error { return deleteOOB(p, shipTable, id, errShipMissing) },
		},
	}
}

func strOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func int64OrNil(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (h *handlers) oobUnits(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	if len(rest) == 0 {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	if rest[0] == "snapshot" && len(rest) == 1 && r.Method == http.MethodGet {
		snap, err := oobSnapshot(p, queryPtr(r, "oob_file"), queryPtr(r, "source_file"))
		if err != nil {
			writeOOBError(w, err, "oob snapshot")
			return
		}
		writeJSON(w, http.StatusOK, snap)
		return
	}
	if rest[0] == "preview" && len(rest) == 1 {
		if r.Method != http.MethodGet {
			detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}
		script, err := oobPreviewScript(p, queryPtr(r, "oob_file"), queryPtr(r, "source_file"))
		if err != nil {
			writeOOBError(w, err, "oob preview")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"script": script})
		return
	}
	if rest[0] == "import" && len(rest) == 1 {
		if r.Method != http.MethodPost {
			detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}
		payload, ok := decodeOOBImport(w, r)
		if !ok {
			return
		}
		result, err := importOOBUnits(p, payload)
		if err != nil {
			writeOOBError(w, err, "oob import")
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	entity, ok := h.oobEntities(p)[rest[0]]
	if !ok {
		detail(w, http.StatusNotFound, "Not Found")
		return
	}
	if len(rest) == 1 {
		switch r.Method {
		case http.MethodGet:
			items, err := entity.list(r)
			if err != nil {
				writeOOBError(w, err, "list "+rest[0])
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			item, err := entity.create(r)
			if err != nil {
				writeOOBError(w, err, "create "+rest[0])
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			methodError(w, "GET or POST only")
		}
		return
	}
	id, ok := itemID(rest[1:])
	if !ok {
		detail(w, http.StatusNotFound, entity.missing)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := entity.get(id)
		if err != nil {
			writeOOBError(w, err, "get "+rest[0])
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		item, err := entity.update(r, id)
		if err != nil {
			writeOOBError(w, err, "update "+rest[0])
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := entity.remove(id); err != nil {
			writeOOBError(w, err, "delete "+rest[0])
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}
