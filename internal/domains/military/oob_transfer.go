package military

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/importsvc"
	unitsparser "haganeworks/internal/hoi4/importsvc/units"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type oobImportInput struct {
	OobFile    string
	SourceFile string
	CountryTag string
	Text       string
	Policy     string
}

type OOBImportResult struct {
	AirWingGroups int      `json:"air_wing_groups"`
	AirWings      int      `json:"air_wings"`
	Fleets        int      `json:"fleets"`
	TaskForces    int      `json:"task_forces"`
	Ships         int      `json:"ships"`
	Skipped       int      `json:"skipped"`
	Warnings      []string `json:"warnings"`
}

func decodeOOBImport(w http.ResponseWriter, r *http.Request) (oobImportInput, bool) {
	invalid := func() (oobImportInput, bool) {
		detail(w, http.StatusUnprocessableEntity, "Invalid OOB import payload")
		return oobImportInput{}, false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return invalid()
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return invalid()
	}
	readString := func(key string, required bool) (string, bool) {
		raw, present := fields[key]
		if !present {
			return "", !required
		}
		if string(raw) == "null" {
			return "", false
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", false
		}
		return value, true
	}
	oobFile, ok := readString("oob_file", false)
	if !ok {
		return invalid()
	}
	sourceFile, ok := readString("source_file", false)
	if !ok {
		return invalid()
	}
	countryTag, ok := readString("country_tag", false)
	if !ok {
		return invalid()
	}
	text, ok := readString("text", true)
	if !ok {
		return invalid()
	}
	policy := "overwrite"
	if _, present := fields["policy"]; present {
		if policy, ok = readString("policy", true); !ok {
			return invalid()
		}
	}
	if utf8.RuneCountInString(oobFile) > 255 || utf8.RuneCountInString(sourceFile) > 500 ||
		utf8.RuneCountInString(countryTag) > 10 || utf8.RuneCountInString(text) > 5_000_000 ||
		(policy != "overwrite" && policy != "skip") {
		return invalid()
	}
	return oobImportInput{OobFile: oobFile, SourceFile: sourceFile, CountryTag: countryTag, Text: text, Policy: policy}, true
}

func pyOOBSpace(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) }

func pyOOBStrip(value string) string { return strings.TrimFunc(value, pyOOBSpace) }

func normaliseOOBSourceFile(sourceFile, oobFile string) string {
	sourceFile = strings.TrimLeft(strings.ReplaceAll(sourceFile, "\\", "/"), "/")
	if sourceFile != "" {
		return sourceFile
	}
	stem := pyOOBStrip(oobFile)
	if stem == "" {
		stem = "oob"
	}
	return "history/units/" + stem + ".txt"
}

func purePosixStem(value string) string {
	base := path.Base(value)
	if strings.Trim(base, ".") == "" {
		return base
	}
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		return base[:i]
	}
	return base
}

func normaliseOOBImportTop(in oobImportInput) (sourceFile, oobFile, countryTag string) {
	sourceFile = normaliseOOBSourceFile(in.SourceFile, in.OobFile)
	if in.OobFile != "" {
		oobFile = pyOOBStrip(in.OobFile)
	} else {
		oobFile = purePosixStem(sourceFile)
	}
	if in.CountryTag != "" {
		countryTag = strings.ToUpper(pyOOBStrip(in.CountryTag))
	} else {
		countryTag = strings.ToUpper(pyOOBStrip(strings.SplitN(oobFile, "_", 2)[0]))
	}
	if runes := []rune(countryTag); len(runes) > 10 {
		countryTag = string(runes[:10])
	}
	return
}

func oobWireSlice(value any) []any {
	items, _ := value.([]any)
	if items == nil {
		return []any{}
	}
	return items
}

func oobWireMap(value any) map[string]any {
	item, _ := value.(map[string]any)
	if item == nil {
		return map[string]any{}
	}
	return item
}

func oobWireInt(item map[string]any, key string) int {
	switch value := item[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		n, _ := strconv.Atoi(value.String())
		return n
	default:
		return 0
	}
}

func oobWireWarnings(value any) []string {
	switch warnings := value.(type) {
	case []string:
		return append([]string{}, warnings...)
	case []any:
		out := make([]string, 0, len(warnings))
		for _, warning := range warnings {
			if text, ok := warning.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return []string{}
	}
}

func countOOBWire(groups, fleets []any) OOBImportResult {
	result := OOBImportResult{Warnings: []string{}, AirWingGroups: len(groups), Fleets: len(fleets)}
	for _, rawGroup := range groups {
		result.AirWings += len(oobWireSlice(oobWireMap(rawGroup)["wings"]))
	}
	for _, rawFleet := range fleets {
		for _, rawTaskForce := range oobWireSlice(oobWireMap(rawFleet)["task_forces"]) {
			result.TaskForces++
			result.Ships += len(oobWireSlice(oobWireMap(rawTaskForce)["ships"]))
		}
	}
	return result
}

func wireJSONValue(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var out any
	err = decoder.Decode(&out)
	return out, err
}

func importOOBUnits(p *project.Project, in oobImportInput) (OOBImportResult, error) {
	result := OOBImportResult{Warnings: []string{}}
	sourceFile, oobFile, countryTag := normaliseOOBImportTop(in)
	tables := []string{airWingGroupTable, airWingTable, fleetTable, taskForceTable, shipTable}
	err := p.Write(tables, func(tx *project.Tx) error {
		inSource := func(row project.Row) bool { return stringEquals(row["source_file"], sourceFile) }
		existing := 0
		for _, table := range []string{airWingGroupTable, fleetTable} {
			for _, row := range tx.Rows(table) {
				if inSource(row) {
					existing++
				}
			}
		}
		if in.Policy == "skip" && existing > 0 {
			result.Skipped = existing
			return nil
		}
		wire := unitsparser.AirNavalOOBJSON(in.Text)
		groupsRaw, err := wireJSONValue(oobWireSlice(wire["groups"]))
		if err != nil {
			return err
		}
		fleetsRaw, err := wireJSONValue(oobWireSlice(wire["fleets"]))
		if err != nil {
			return err
		}
		groups, fleets := oobWireSlice(groupsRaw), oobWireSlice(fleetsRaw)
		result.Warnings = oobWireWarnings(wire["warnings"])
		if len(result.Warnings) > 0 && len(groups) == 0 && len(fleets) == 0 {
			return nil
		}
		deleteOOBRows(tx, airWingGroupTable, inSource)
		deleteOOBRows(tx, fleetTable, inSource)
		counts := countOOBWire(groups, fleets)
		counts.Warnings = result.Warnings
		result = counts

		for _, rawGroup := range groups {
			g := oobWireMap(rawGroup)
			group := tx.Insert(airWingGroupTable, project.Row{
				"oob_file": oobFile, "source_file": sourceFile, "country_tag": countryTag,
				"location": wireString(g, "location"), "ordering": wireOrdering(g),
				"layout": wireJSON(g, "layout", []any{}), "source": "created",
			})
			for _, rawWing := range oobWireSlice(g["wings"]) {
				wing := oobWireMap(rawWing)
				tx.Insert(airWingTable, project.Row{
					"group_id": group["id"], "equipment_type": wireString(wing, "equipment_type"),
					"name": wireString(wing, "name"), "owner": wireString(wing, "owner"),
					"amount": wireString(wing, "amount"), "ordering": wireOrdering(wing),
					"body_ast": wireJSON(wing, "body_ast", []any{}), "source": "created",
				})
			}
		}
		for _, rawFleet := range fleets {
			f := oobWireMap(rawFleet)
			fleet := tx.Insert(fleetTable, project.Row{
				"oob_file": oobFile, "source_file": sourceFile, "country_tag": countryTag,
				"name": wireString(f, "name"), "naval_base": wireString(f, "naval_base"),
				"ordering": wireOrdering(f), "layout": wireJSON(f, "layout", []any{}), "source": "created",
			})
			for _, rawForce := range oobWireSlice(f["task_forces"]) {
				tf := oobWireMap(rawForce)
				force := tx.Insert(taskForceTable, project.Row{
					"fleet_id": fleet["id"], "name": wireString(tf, "name"), "location": wireString(tf, "location"),
					"ordering": wireOrdering(tf), "layout": wireJSON(tf, "layout", []any{}), "source": "created",
				})
				for _, rawShip := range oobWireSlice(tf["ships"]) {
					sh := oobWireMap(rawShip)
					tx.Insert(shipTable, project.Row{
						"task_force_id": force["id"], "name": wireString(sh, "name"),
						"definition":              wireString(sh, "definition"),
						"start_experience_factor": wireString(sh, "start_experience_factor"),
						"equipment_type":          wireString(sh, "equipment_type"),
						"amount":                  wireString(sh, "amount"), "owner": wireString(sh, "owner"),
						"version_name": wireString(sh, "version_name"), "ordering": wireOrdering(sh),
						"body_ast": wireJSON(sh, "body_ast", []any{}), "source": "created",
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return OOBImportResult{Warnings: []string{}}, err
	}
	return result, nil
}

func wireOrdering(item map[string]any) any {
	return jsonValue(oobWireInt(item, "ordering"))
}

func oobStringPointer(value any) *string {
	if value == nil {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		text = fmt.Sprint(value)
	}
	return &text
}

func oobScalar(value any) paradox.Scalar {
	if tagged, ok := value.(map[string]any); ok {
		tag, _ := tagged["t"].(string)
		raw := tagged["v"]
		switch tag {
		case "int":
			if n, ok := new(big.Int).SetString(fmt.Sprint(raw), 10); ok {
				return paradox.Scalar{Kind: paradox.ScalarInt, Int: n}
			}
		case "float":
			f, _ := strconv.ParseFloat(fmt.Sprint(raw), 64)
			return paradox.Scalar{Kind: paradox.ScalarFloat, Float: f}
		case "bool":
			if truth, _ := raw.(bool); truth {
				return paradox.Scalar{Kind: paradox.ScalarStr, Str: "True"}
			}
			return paradox.Scalar{Kind: paradox.ScalarStr, Str: "False"}
		case "str":
			return paradox.Scalar{Kind: paradox.ScalarStr, Str: fmt.Sprint(raw)}
		}
	}
	switch scalar := value.(type) {
	case string:
		return paradox.Scalar{Kind: paradox.ScalarStr, Str: scalar}
	case float64:
		if !math.IsNaN(scalar) && !math.IsInf(scalar, 0) && scalar == math.Trunc(scalar) {
			return paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(int64(scalar))}
		}
		return paradox.Scalar{Kind: paradox.ScalarFloat, Float: scalar}
	case json.Number:
		if n, ok := new(big.Int).SetString(scalar.String(), 10); ok {
			return paradox.Scalar{Kind: paradox.ScalarInt, Int: n}
		}
		f, _ := scalar.Float64()
		return paradox.Scalar{Kind: paradox.ScalarFloat, Float: f}
	case bool:
		if scalar {
			return paradox.Scalar{Kind: paradox.ScalarStr, Str: "True"}
		}
		return paradox.Scalar{Kind: paradox.ScalarStr, Str: "False"}
	case nil:
		return paradox.Scalar{Kind: paradox.ScalarStr, Str: "None"}
	default:
		return paradox.Scalar{Kind: paradox.ScalarStr, Str: fmt.Sprint(scalar)}
	}
}

func oobEntryData(item map[string]any) *unitsparser.EntryData {
	entry := &unitsparser.EntryData{
		Key: oobStringPointer(item["key"]), Op: oobStringPointer(item["op"]),
		Line: oobWireInt(item, "line"), Quoted: item["quoted"] == true,
		Qualifier: oobStringPointer(item["qualifier"]),
	}
	if children, ok := item["value"].([]any); ok {
		entry.IsBlock = true
		entry.Children = make([]*unitsparser.EntryData, 0, len(children))
		for _, child := range children {
			entry.Children = append(entry.Children, oobEntryData(oobWireMap(child)))
		}
	} else {
		entry.Value = oobScalar(item["value"])
	}
	return entry
}

func oobBodyAST(items []map[string]any) []*unitsparser.EntryData {
	out := make([]*unitsparser.EntryData, 0, len(items))
	for _, item := range items {
		out = append(out, oobEntryData(item))
	}
	return out
}

func oobLayout(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		entry := emitmisc.NewOD().Set("kind", item["kind"])
		switch item["kind"] {
		case "raw":
			entry.Set("entry", oobEntryData(oobWireMap(item["entry"])))
		case "field":
			entry.Set("field", item["field"])
		default:
			entry.Set("ordinal", item["ordinal"])
		}
		out = append(out, entry)
	}
	return out
}

func oobGroupNode(group AirWingGroup, wings []AirWing) *emitmisc.NS {
	node := emitmisc.NewNS().Set("id", group.ID).Set("project_id", group.ProjectID).
		Set("oob_file", group.OobFile).Set("source_file", group.SourceFile).
		Set("country_tag", group.CountryTag).Set("location", group.Location).
		Set("ordering", int64(group.Ordering)).Set("layout", oobLayout(group.Layout)).Set("source", group.Source)
	children := make([]any, 0, len(wings))
	for _, wing := range wings {
		children = append(children, emitmisc.NewNS().Set("id", wing.ID).Set("project_id", wing.ProjectID).
			Set("group_id", wing.GroupID).Set("equipment_type", wing.EquipmentType).Set("name", wing.Name).
			Set("owner", wing.Owner).Set("amount", wing.Amount).Set("ordering", int64(wing.Ordering)).
			Set("body_ast", oobBodyAST(wing.BodyAst)).Set("source", wing.Source))
	}
	return node.Set("wings", children)
}

func oobFleetNode(fleet Fleet, taskForces []TaskForce, shipsByTaskForce map[int64][]Ship) *emitmisc.NS {
	node := emitmisc.NewNS().Set("id", fleet.ID).Set("project_id", fleet.ProjectID).
		Set("oob_file", fleet.OobFile).Set("source_file", fleet.SourceFile).
		Set("country_tag", fleet.CountryTag).Set("name", fleet.Name).Set("naval_base", fleet.NavalBase).
		Set("ordering", int64(fleet.Ordering)).Set("layout", oobLayout(fleet.Layout)).Set("source", fleet.Source)
	children := make([]any, 0, len(taskForces))
	for _, taskForce := range taskForces {
		tfNode := emitmisc.NewNS().Set("id", taskForce.ID).Set("project_id", taskForce.ProjectID).
			Set("fleet_id", taskForce.FleetID).Set("name", taskForce.Name).Set("location", taskForce.Location).
			Set("ordering", int64(taskForce.Ordering)).Set("layout", oobLayout(taskForce.Layout)).Set("source", taskForce.Source)
		shipNodes := make([]any, 0, len(shipsByTaskForce[taskForce.ID]))
		for _, ship := range shipsByTaskForce[taskForce.ID] {
			shipNodes = append(shipNodes, emitmisc.NewNS().Set("id", ship.ID).Set("project_id", ship.ProjectID).
				Set("task_force_id", ship.TaskForceID).Set("name", ship.Name).Set("definition", ship.Definition).
				Set("start_experience_factor", ship.StartExperienceFactor).Set("equipment_type", ship.EquipmentType).
				Set("amount", ship.Amount).Set("owner", ship.Owner).Set("version_name", ship.VersionName).
				Set("ordering", int64(ship.Ordering)).Set("body_ast", oobBodyAST(ship.BodyAst)).Set("source", ship.Source))
		}
		children = append(children, tfNode.Set("ships", shipNodes))
	}
	return node.Set("task_forces", children)
}

func validateOOBPreviewSnapshot(snapshot OOBSnapshot) error {
	wingByGroup := map[int64][]AirWing{}
	for _, wing := range snapshot.AirWings {
		wingByGroup[wing.GroupID] = append(wingByGroup[wing.GroupID], wing)
	}
	for _, group := range snapshot.AirWingGroups {
		if _, err := importsvc.RenderHOI4ScalarToken(group.Location); err != nil {
			suffix := ""
			if group.ID != 0 {
				suffix = fmt.Sprintf(" #%d", group.ID)
			}
			return fmt.Errorf("Air wing group%s requires a valid spawn state/location token", suffix)
		}
		for _, wing := range wingByGroup[group.ID] {
			if _, err := importsvc.RenderHOI4ScalarToken(wing.EquipmentType); err != nil {
				suffix := ""
				if wing.ID != 0 {
					suffix = fmt.Sprintf(" #%d", wing.ID)
				}
				return fmt.Errorf("Air wing%s requires a valid equipment type token", suffix)
			}
			for _, token := range []string{wing.Owner, wing.Amount} {
				if token != "" {
					if _, err := importsvc.RenderHOI4ScalarToken(token); err != nil {
						return err
					}
				}
			}
		}
	}
	shipsByTF := map[int64][]Ship{}
	for _, ship := range snapshot.Ships {
		shipsByTF[ship.TaskForceID] = append(shipsByTF[ship.TaskForceID], ship)
	}
	for _, fleet := range snapshot.Fleets {
		if fleet.NavalBase != "" {
			if _, err := importsvc.RenderHOI4ScalarToken(fleet.NavalBase); err != nil {
				return err
			}
		}
		for _, taskForce := range snapshot.TaskForces {
			if taskForce.FleetID != fleet.ID {
				continue
			}
			if taskForce.Location != "" {
				if _, err := importsvc.RenderHOI4ScalarToken(taskForce.Location); err != nil {
					return err
				}
			}
			for _, ship := range shipsByTF[taskForce.ID] {
				for _, token := range []string{ship.Definition, ship.StartExperienceFactor} {
					if token != "" {
						if _, err := importsvc.RenderHOI4ScalarToken(token); err != nil {
							return err
						}
					}
				}
				if pyOOBStrip(ship.EquipmentType) != "" {
					for _, token := range []string{ship.Amount, ship.Owner} {
						if token != "" {
							if _, err := importsvc.RenderHOI4ScalarToken(token); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func renderOOBPreview(snapshot OOBSnapshot) (string, error) {
	if err := validateOOBPreviewSnapshot(snapshot); err != nil {
		return "", detailError(http.StatusUnprocessableEntity, err.Error())
	}
	wingsByGroup := map[int64][]AirWing{}
	for _, wing := range snapshot.AirWings {
		wingsByGroup[wing.GroupID] = append(wingsByGroup[wing.GroupID], wing)
	}
	groups := make([]*emitmisc.NS, 0, len(snapshot.AirWingGroups))
	for _, group := range snapshot.AirWingGroups {
		groups = append(groups, oobGroupNode(group, wingsByGroup[group.ID]))
	}
	shipsByTaskForce := map[int64][]Ship{}
	for _, ship := range snapshot.Ships {
		shipsByTaskForce[ship.TaskForceID] = append(shipsByTaskForce[ship.TaskForceID], ship)
	}
	taskForcesByFleet := map[int64][]TaskForce{}
	for _, taskForce := range snapshot.TaskForces {
		taskForcesByFleet[taskForce.FleetID] = append(taskForcesByFleet[taskForce.FleetID], taskForce)
	}
	fleets := make([]*emitmisc.NS, 0, len(snapshot.Fleets))
	for _, fleet := range snapshot.Fleets {
		fleets = append(fleets, oobFleetNode(fleet, taskForcesByFleet[fleet.ID], shipsByTaskForce))
	}
	blocks := []string{}
	if air, _ := emitmisc.GenerateAirWingsBlockTolerant(groups, 0); air != "" {
		blocks = append(blocks, air)
	}
	if naval, _ := emitmisc.GenerateFleetsBlockTolerant(fleets, 0); naval != "" {
		blocks = append(blocks, naval)
	}
	if len(blocks) == 0 {
		return "", nil
	}
	return strings.Join(blocks, "\n\n") + "\n", nil
}

func oobPreviewScript(p *project.Project, oobFile, sourceFile *string) (string, error) {
	snapshot, err := oobSnapshot(p, oobFile, sourceFile)
	if err != nil {
		return "", err
	}
	return renderOOBPreview(snapshot)
}
