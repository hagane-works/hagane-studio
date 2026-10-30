package units

import (
	"fmt"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

type EntryData struct {
	Key       *string
	Op        *string
	Value     paradox.Scalar
	Children  []*EntryData
	IsBlock   bool
	Line      int
	Quoted    bool
	Qualifier *string
}

func EntryToData(e *paradox.Entry) *EntryData {
	d := &EntryData{
		Key:       e.Key,
		Op:        e.Op,
		Line:      e.Line,
		Quoted:    e.Quoted,
		Qualifier: e.Qualifier,
	}
	if e.Block != nil {
		d.IsBlock = true
		d.Children = make([]*EntryData, 0, len(e.Block.Entries))
		for _, c := range e.Block.Entries {
			d.Children = append(d.Children, EntryToData(c))
		}
	} else {
		d.Value = e.Scalar
	}
	return d
}

func DataToEntry(d *EntryData) *paradox.Entry {
	e := &paradox.Entry{
		Key:       d.Key,
		Op:        d.Op,
		Line:      d.Line,
		Quoted:    d.Quoted,
		Qualifier: d.Qualifier,
	}
	if d.IsBlock {
		b := &paradox.Block{Entries: make([]*paradox.Entry, 0, len(d.Children))}
		for _, c := range d.Children {
			b.Entries = append(b.Entries, DataToEntry(c))
		}
		e.Block = b
	} else {
		e.Scalar = d.Value
	}
	return e
}

func EntriesToData(entries []*paradox.Entry) []*EntryData {
	out := make([]*EntryData, 0, len(entries))
	for _, e := range entries {
		out = append(out, EntryToData(e))
	}
	return out
}

func DataToEntries(entries []*EntryData) []*paradox.Entry {
	out := make([]*paradox.Entry, 0, len(entries))
	for _, d := range entries {
		out = append(out, DataToEntry(d))
	}
	return out
}

func ScalarToken(e *paradox.Entry) string {
	if e == nil || e.Block != nil {
		return ""
	}
	bare := &paradox.Entry{Scalar: e.Scalar, Quoted: e.Quoted}
	return pyStrip(importsvc.ToScriptEntries([]*paradox.Entry{bare}, 0))
}

func ScalarText(e *paradox.Entry) string {
	if e == nil || e.Block != nil {
		return ""
	}
	return importsvc.PyScalarStr(e.Scalar)
}

func firstScalar(entries []*paradox.Entry, key string) *paradox.Entry {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key && e.Block == nil {
			return e
		}
	}
	return nil
}

func ppBlock(entries []*paradox.Entry, key string) ([]*paradox.Entry, bool) {
	for _, e := range entries {
		if e.Key != nil && *e.Key == key {
			if e.Block == nil {
				return nil, false
			}
			return e.Block.Entries, true
		}
	}
	return nil, false
}

func validRequiredToken(value string, warning string, warnings *[]string) bool {
	if _, err := importsvc.ValidateHOI4ScalarToken(value, false); err != nil {
		*warnings = append(*warnings, warning)
		return false
	}
	return true
}

const (
	LayoutWingEquipment = "wing_equipment"
	LayoutWingName      = "wing_name"
	LayoutShip          = "ship"
	LayoutTaskForce     = "task_force"
	LayoutField         = "field"
	LayoutRaw           = "raw"
)

type LayoutItem struct {
	Kind    string
	Ordinal int
	Field   string
	Entry   *EntryData
}

type AirWing struct {
	EquipmentType string
	Name          string
	Owner         string
	Amount        string
	Ordering      int
	BodyAST       []*EntryData
}

type AirWingGroup struct {
	Location string
	Ordering int
	Layout   []*LayoutItem
	Wings    []*AirWing
}

type Ship struct {
	Name                  string
	Definition            string
	StartExperienceFactor string
	EquipmentType         string
	Amount                string
	Owner                 string
	VersionName           string
	Ordering              int
	BodyAST               []*EntryData
}

type TaskForce struct {
	Name     string
	Location string
	Ordering int
	Layout   []*LayoutItem
	Ships    []*Ship
}

type Fleet struct {
	Name       string
	NavalBase  string
	Ordering   int
	Layout     []*LayoutItem
	TaskForces []*TaskForce
}

type Result struct {
	AirWingGroups []*AirWingGroup
	Fleets        []*Fleet
}

func parseAirGroup(entry *paradox.Entry, ordering int, warnings *[]string) *AirWingGroup {
	var body []*paradox.Entry
	if entry.Block != nil {
		body = entry.Block.Entries
	}
	group := &AirWingGroup{
		Ordering: ordering,
		Layout:   []*LayoutItem{},
		Wings:    []*AirWing{},
	}
	if entry.Key != nil {
		group.Location = *entry.Key
	}
	var awaitingName []int

	for _, child := range body {
		if child.Key != nil && child.Block != nil {
			if !validRequiredToken(
				*child.Key,
				"air wing skipped: one valid equipment type token is required",
				warnings,
			) {
				continue
			}
			wingIndex := len(group.Wings)
			group.Wings = append(group.Wings, &AirWing{
				EquipmentType: *child.Key,
				Name:          "",
				Owner:         ScalarToken(firstScalar(child.Block.Entries, "owner")),
				Amount:        ScalarToken(firstScalar(child.Block.Entries, "amount")),
				Ordering:      wingIndex,
				BodyAST:       EntriesToData(child.Block.Entries),
			})
			awaitingName = append(awaitingName, wingIndex)
			group.Layout = append(group.Layout,
				&LayoutItem{Kind: LayoutWingEquipment, Ordinal: wingIndex})
			continue
		}

		if child.Key != nil && *child.Key == "name" && child.Block == nil &&
			len(awaitingName) > 0 {
			wingIndex := awaitingName[len(awaitingName)-1]
			awaitingName = awaitingName[:len(awaitingName)-1]
			group.Wings[wingIndex].Name = ScalarText(child)
			group.Layout = append(group.Layout,
				&LayoutItem{Kind: LayoutWingName, Ordinal: wingIndex})
			continue
		}

		group.Layout = append(group.Layout,
			&LayoutItem{Kind: LayoutRaw, Entry: EntryToData(child)})
	}
	return group
}

func parseShip(body []*paradox.Entry, ordering int) *Ship {
	ship := &Ship{Ordering: ordering}
	if equipment, ok := ppBlock(body, "equipment"); ok {
		for _, node := range equipment {
			if node.Key == nil || node.Block == nil {
				continue
			}
			ship.EquipmentType = *node.Key
			ship.Amount = ScalarToken(firstScalar(node.Block.Entries, "amount"))
			ship.Owner = ScalarToken(firstScalar(node.Block.Entries, "owner"))
			ship.VersionName = ScalarText(firstScalar(node.Block.Entries, "version_name"))
			break
		}
	}
	ship.Name = ScalarText(firstScalar(body, "name"))
	ship.Definition = ScalarToken(firstScalar(body, "definition"))
	ship.StartExperienceFactor = ScalarToken(firstScalar(body, "start_experience_factor"))
	ship.BodyAST = EntriesToData(body)
	return ship
}

func parseTaskForce(body []*paradox.Entry, ordering int) *TaskForce {
	tf := &TaskForce{
		Ordering: ordering,
		Layout:   []*LayoutItem{},
		Ships:    []*Ship{},
	}
	seen := map[string]bool{}
	for _, child := range body {
		switch {
		case child.Key != nil && *child.Key == "ship" && child.Block != nil:
			shipIndex := len(tf.Ships)
			tf.Ships = append(tf.Ships, parseShip(child.Block.Entries, shipIndex))
			tf.Layout = append(tf.Layout,
				&LayoutItem{Kind: LayoutShip, Ordinal: shipIndex})
		case child.Key != nil &&
			(*child.Key == "name" || *child.Key == "location") &&
			!seen[*child.Key] && child.Block == nil:
			seen[*child.Key] = true
			if *child.Key == "name" {
				tf.Name = ScalarText(child)
			} else {
				tf.Location = ScalarToken(child)
			}
			tf.Layout = append(tf.Layout,
				&LayoutItem{Kind: LayoutField, Field: *child.Key})
		default:
			tf.Layout = append(tf.Layout,
				&LayoutItem{Kind: LayoutRaw, Entry: EntryToData(child)})
		}
	}
	return tf
}

func parseFleet(body []*paradox.Entry, ordering int) *Fleet {
	fleet := &Fleet{
		Ordering:   ordering,
		Layout:     []*LayoutItem{},
		TaskForces: []*TaskForce{},
	}
	seen := map[string]bool{}
	for _, child := range body {
		switch {
		case child.Key != nil && *child.Key == "task_force" && child.Block != nil:
			tfIndex := len(fleet.TaskForces)
			fleet.TaskForces = append(fleet.TaskForces,
				parseTaskForce(child.Block.Entries, tfIndex))
			fleet.Layout = append(fleet.Layout,
				&LayoutItem{Kind: LayoutTaskForce, Ordinal: tfIndex})
		case child.Key != nil &&
			(*child.Key == "name" || *child.Key == "naval_base") &&
			!seen[*child.Key] && child.Block == nil:
			seen[*child.Key] = true
			if *child.Key == "name" {
				fleet.Name = ScalarText(child)
			} else {
				fleet.NavalBase = ScalarToken(child)
			}
			fleet.Layout = append(fleet.Layout,
				&LayoutItem{Kind: LayoutField, Field: *child.Key})
		default:
			fleet.Layout = append(fleet.Layout,
				&LayoutItem{Kind: LayoutRaw, Entry: EntryToData(child)})
		}
	}
	return fleet
}

func ImportAirNavalOOB(text string) (*Result, []string) {
	result := &Result{
		AirWingGroups: []*AirWingGroup{},
		Fleets:        []*Fleet{},
	}
	warnings := []string{}

	ast, recoveries, err := paradox.Parse(text)
	if err != nil {
		return result, []string{"air/naval OOB parse failed: " + err.Error()}
	}

	for _, recovery := range recoveries {
		if recovery.Code != paradox.RecoveryUnclosedBlock {
			continue
		}
		eofLine := recovery.Line
		switch value := recovery.Details["eof_line"].(type) {
		case int:
			eofLine = value
		case int64:
			eofLine = int(value)
		case float64:
			eofLine = int(value)
		}
		return result, []string{fmt.Sprintf(
			"air/naval OOB parse failed: Expected RBRACE, got EOF '' at line %d", eofLine,
		)}
	}

	collectAir := func(block []*paradox.Entry) {
		for _, airEntry := range block {
			if airEntry.Key == nil || airEntry.Block == nil {
				continue
			}
			if !validRequiredToken(
				*airEntry.Key,
				"air_wings group skipped: one valid spawn state/location token is required",
				&warnings,
			) {
				continue
			}
			result.AirWingGroups = append(result.AirWingGroups,
				parseAirGroup(airEntry, len(result.AirWingGroups), &warnings))
		}
	}

	collectFleets := func(block []*paradox.Entry) {
		for _, fleetEntry := range block {
			if fleetEntry.Key == nil || *fleetEntry.Key != "fleet" ||
				fleetEntry.Block == nil {
				continue
			}
			result.Fleets = append(result.Fleets,
				parseFleet(fleetEntry.Block.Entries, len(result.Fleets)))
		}
	}

	for _, top := range ast.Entries {
		switch {
		case top.Key != nil && *top.Key == "air_wings" && top.Block != nil:
			collectAir(top.Block.Entries)
		case top.Key != nil && *top.Key == "units" && top.Block != nil:
			collectFleets(top.Block.Entries)

			for _, nestedAir := range top.Block.Entries {
				if nestedAir.Key != nil && *nestedAir.Key == "air_wings" &&
					nestedAir.Block != nil {
					collectAir(nestedAir.Block.Entries)
				}
			}
		case top.Key != nil && *top.Key == "fleet" && top.Block != nil:
			result.Fleets = append(result.Fleets,
				parseFleet(top.Block.Entries, len(result.Fleets)))
		}
	}
	return result, warnings
}

var ParseAirNavalOOB = ImportAirNavalOOB
