package gamesave

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

const germanOOB = `division_template = {
	name = "Infanterie-Division"
	regiments = { infantry = { x = 0 y = 0 } }
}
division_template = {
	name = "Panzer-Division"
	regiments = { light_armor = { x = 0 y = 0 } }
}
units = { division = { division_template = "Panzer-Division" } division = { division_template="Panzer-Division" } division = { division_template = "Infanterie-Division" } }
`

type designerIndexFake struct{}

func (designerIndexFake) TemplateOOB(tag, _, template string) (string, bool) {
	if tag != "GER" || template == "AOK" {
		return "", false
	}
	return "GER_1936", true
}

func (designerIndexFake) OOBOf(tag, _ string) []string {
	if tag == "GER" {
		return []string{"GER_1936"}
	}
	return nil
}

func (designerIndexFake) OOBSource(name string) ([]byte, bool) {
	return []byte(germanOOB), name == "GER_1936"
}

func (designerIndexFake) RegimentalSupportAvailable(string, string) []string {
	return []string{"recon_regimental"}
}

func templateRows(t *testing.T, p *project.Project) []divisionTemplate {
	t.Helper()
	out := []divisionTemplate{}
	for _, row := range rows(t, p, divisionTemplateTable) {
		out = append(out, templateFromRow(row))
	}
	return out
}

func saveDesigner(p *project.Project, index designerSaveIndex, in designerSaveInput) (designerSaveResult, error) {
	return run(p, func(st *store) (designerSaveResult, error) { return saveDesignerTemplate(st, index, in) })
}

func TestDesignerSaveTakesTheBattleOrderInFirst(t *testing.T) {
	p := newProject(t)
	in := designerSaveInput{Tag: "GER", Date: "1936.1.1", Template: "Panzer-Division", Name: "Panzer-Division",
		Regiments: []designerSaveCell{{"light_armor", 0, 0}, {"light_armor", 0, 1}}}
	result, err := saveDesigner(p, designerIndexFake{}, in)
	if err != nil {
		t.Fatal(err)
	}

	saved := templateRows(t, p)
	if !result.Imported || len(saved) != 2 {
		t.Fatalf("imported=%v rows=%d", result.Imported, len(saved))
	}
	panzer := saved[1]
	if panzer.TemplateName != "Panzer-Division" || len(panzer.Regiments) != 2 || panzer.Regiments[1]["token"] != "light_armor" || panzer.Regiments[1]["y"] != 1.0 {
		t.Errorf("Panzer-Division = %+v", panzer)
	}
	if infantry := saved[0]; len(infantry.Regiments) != 1 || infantry.Regiments[0]["token"] != "infantry" || infantry.Source != "created" {
		t.Errorf("Infanterie-Division changed: %+v", infantry)
	}

	in.Name = "Leichte-Division"
	if result, err = saveDesigner(p, designerIndexFake{}, in); err != nil || result.Imported {
		t.Fatalf("second save: %v imported=%v", err, result.Imported)
	}
	saved = templateRows(t, p)
	if len(saved) != 2 || saved[1].TemplateName != "Leichte-Division" {
		t.Fatalf("rename = %+v", saved)
	}

	for _, row := range saved {
		if strings.Contains(row.OobUnitsTail, `"Panzer-Division"`) || strings.Count(row.OobUnitsTail, `"Leichte-Division"`) != 2 ||
			!strings.Contains(row.OobUnitsTail, `division_template = "Infanterie-Division"`) {
			t.Errorf("units after the rename: %s", row.OobUnitsTail)
		}
	}
}

func TestDesignerSaveNewTemplateAndRefusals(t *testing.T) {
	p := newProject(t)
	cell := []designerSaveCell{{"infantry", 0, 0}}
	created, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "GER", Name: "Gebirgs-Division", Regiments: cell})
	if err != nil {
		t.Fatal(err)
	}
	if created.Template.TemplateName != "Gebirgs-Division" || created.Template.Ordering != 2 || created.OobFile != "GER_1936" ||
		created.Template.CountryTag != "GER" || created.Template.ID == 0 {
		t.Errorf("new template = %+v", created.Template)
	}
	if len(templateRows(t, p)) != 3 {
		t.Errorf("rows = %+v", templateRows(t, p))
	}
	for _, c := range []struct {
		in   designerSaveInput
		want int
	}{
		{designerSaveInput{Tag: "GER", Name: "Infanterie-Division", Regiments: cell}, 409},
		{designerSaveInput{Tag: "GER", Template: "AOK", Name: "AOK", Regiments: cell}, 422},
		{designerSaveInput{Tag: "GER", Name: "Leer"}, 422},
		{designerSaveInput{Tag: "GER", Template: "Gone", Name: "Gone", Regiments: cell}, 404},
		{designerSaveInput{Tag: "GER", Name: "X", Regiments: []designerSaveCell{{"infantry", 0, 0}, {"infantry", 0, 1}, {"infantry", 0, 2}},
			RegimentalSupport: []designerSaveCell{{"not_available", 0, 0}}}, 422},
	} {
		if _, err := saveDesigner(p, designerIndexFake{}, c.in); statusOf(err) != c.want {
			t.Errorf("%+v = %v, want %d", c.in, err, c.want)
		}
	}

	if len(templateRows(t, p)) != 3 {
		t.Errorf("rows after refusals = %d", len(templateRows(t, p)))
	}
}

func TestDesignerSaveRegimentalSupport(t *testing.T) {
	p := newProject(t)
	three := []designerSaveCell{{"infantry", 0, 0}, {"infantry", 0, 1}, {"infantry", 0, 2}}
	out, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "GER", Name: "Mit Aufklärung", Regiments: three,
		RegimentalSupport: []designerSaveCell{{"recon_regimental", 0, 0}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Template.RawExtras["regimental_support"]; got != "\trecon_regimental = { x = 0 y = 0 }\n" {
		t.Errorf("raw_extras = %q", got)
	}
}

func TestDesignerRenameCarriesTheProjectDeployments(t *testing.T) {
	p := newProject(t)
	seed(t, p, divisionDeploymentTable,
		project.Row{"oob_file": "GER_1936", "division_template": "Panzer-Division", "ordering": int64(0)},
		project.Row{"oob_file": "GER_1936", "division_template": "Infanterie-Division", "ordering": int64(1)},
		project.Row{"oob_file": "GER_1939", "division_template": "Panzer-Division", "ordering": int64(0)})
	if _, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{
		Tag: "GER", Template: "Panzer-Division", Name: "Leichte-Division", Regiments: []designerSaveCell{{"light_armor", 0, 0}},
	}); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, row := range rows(t, p, divisionDeploymentTable) {
		got = append(got, text(row, "division_template"))
	}

	if strings.Join(got, ",") != "Leichte-Division,Infanterie-Division,Panzer-Division" {
		t.Errorf("deployments = %v", got)
	}
}

func TestTakeOOBInOnlyOnce(t *testing.T) {
	p := newProject(t)
	take := func(oob string) ([]divisionTemplate, bool, error) {
		var imported bool
		got, err := run(p, func(st *store) ([]divisionTemplate, error) {
			out, in, err := takeOOBIn(st, designerIndexFake{}, oob)
			imported = in
			return out, err
		})
		return got, imported, err
	}
	got, imported, err := take("GER_1936")
	if err != nil || !imported || len(got) != 2 {
		t.Fatalf("first take: rows=%d imported=%v err=%v", len(got), imported, err)
	}
	if got[0].OobFile != "GER_1936" || got[0].CountryTag != "GER" || !strings.HasPrefix(got[0].OobUnitsTail, "units = {") {
		t.Errorf("row = %+v", got[0])
	}
	got, imported, err = take("GER_1936")
	if err != nil || imported || len(got) != 2 || len(templateRows(t, p)) != 2 {
		t.Fatalf("second take: rows=%d imported=%v err=%v", len(got), imported, err)
	}
	if _, _, err := take("ENG_1936"); statusOf(err) != 422 {
		t.Fatalf("a file without text: %v", err)
	}
}

func TestDesignerSaveStartsABattleOrder(t *testing.T) {
	p := newProject(t)
	out, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "TST", Date: "1936.1.1", Name: "Erste", Regiments: []designerSaveCell{{"infantry", 0, 0}}})
	if err != nil {
		t.Fatal(err)
	}
	if out.OobFile != "TST_1936" || !out.StartedOOB || out.Imported || out.Template.Ordering != 0 {
		t.Errorf("result = %+v", out)
	}

	out, err = saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "TST", Date: "1936.1.1", Name: "Zweite", Regiments: []designerSaveCell{{"infantry", 0, 0}}})
	if err != nil || out.OobFile != "TST_1936" || out.StartedOOB {
		t.Errorf("second = %+v, %v", out, err)
	}
}

type designerRawIndex struct{ text *string }

func (designerRawIndex) TemplateOOB(tag, _, _ string) (string, bool) { return "GER_1936", tag == "GER" }
func (designerRawIndex) OOBOf(tag, _ string) []string {
	if tag == "GER" {
		return []string{"GER_1936"}
	}
	return nil
}
func (i designerRawIndex) OOBSource(name string) ([]byte, bool) {
	return []byte(*i.text), name == "GER_1936"
}
func (designerRawIndex) RegimentalSupportAvailable(string, string) []string { return nil }

func TestDesignerSaveWritesIntoTheImportedModsFile(t *testing.T) {

	p := newProject(t)
	seedRaw(t, p, gameview.RawFileInsert{DestPath: "history/units/GER_1936.txt", Content: rawOOB, RawKind: "verbatim_override", Source: "imported"})
	shown := rawOOB
	index := designerRawIndex{text: &shown}
	in := designerSaveInput{Tag: "GER", Template: "Panzer-Division", Name: "Panzer-Division", NamesGroup: "GER_Arm_01",
		Priority: intPtr(1), Regiments: []designerSaveCell{{"light_armor", 0, 0}, {"light_armor", 0, 1}}}
	result, err := saveDesigner(p, index, in)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported || result.Template.TemplateName != "Panzer-Division" || result.OobFile != "GER_1936" {
		t.Errorf("result = %+v", result)
	}
	if len(templateRows(t, p)) != 0 {
		t.Errorf("rows written beside the raw file: %+v", templateRows(t, p))
	}
	files := rawFiles(t, p)
	if len(files) != 1 || files[0].Note != designerRawFileNote {
		t.Fatalf("raw files = %+v", files)
	}
	saved := files[0].Content
	if strings.Contains(saved, "motorized") || strings.Contains(saved, "engineer") || strings.Count(saved, "light_armor") != 2 ||
		strings.Contains(saved, "priority") {
		t.Errorf("saved file:\n%s", saved)
	}

	shown = saved
	in.Name = "Leichte-Division"
	if _, err := saveDesigner(p, index, in); err != nil {
		t.Fatal(err)
	}
	if files := rawFiles(t, p); !strings.Contains(files[0].Content, `division_template = "Leichte-Division"`) {
		t.Errorf("rename not carried into the units:\n%s", files[0].Content)
	}

	before := rawFiles(t, p)[0].Content
	in.Regiments = []designerSaveCell{{"light_armor } units = {", 0, 0}}
	if _, err := saveDesigner(p, index, in); statusOf(err) != 422 || rawFiles(t, p)[0].Content != before {
		t.Fatalf("script in a battalion type: %v", err)
	}
}

func TestDesignerSaveWithoutARawFileStillUsesTheRows(t *testing.T) {
	p := newProject(t)
	seedRaw(t, p, gameview.RawFileInsert{DestPath: "history/units/ENG_1936.txt", Content: "units = { }", RawKind: "verbatim_override"})
	result, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "GER", Template: "Panzer-Division", Name: "Panzer-Division",
		Regiments: []designerSaveCell{{"light_armor", 0, 0}}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Imported || len(templateRows(t, p)) != 2 || rawFiles(t, p)[0].Content != "units = { }" {
		t.Errorf("imported=%v rows=%d", result.Imported, len(templateRows(t, p)))
	}
}

const hqFile = "history/general/taog_hq_template.txt"

func TestHQSaveTakesTheFileInThenEditsTheProjectsCopy(t *testing.T) {
	p := newProject(t)
	base := index(t, map[string]string{hqFile: hqFixture, "history/countries/GER - Germany.txt": "capital = 64\n"})
	in := designerSaveInput{Tag: "GER", Date: "1936.1.1.12", Template: "AOK", Name: "AOK",
		Regiments: []designerSaveCell{{"hq_light_armor", 0, 0}}, Support: []designerSaveCell{{"hq_support_company", 0, 0}}}
	save := func(idx hqSaveIndex, in designerSaveInput) (map[string]any, bool, error) {
		var handled bool
		out, err := run(p, func(st *store) (map[string]any, error) {
			out, h, err := saveHQTemplate(st, idx, in)
			handled = h
			return out, err
		})
		return out, handled, err
	}
	result, handled, err := save(base, in)
	if err != nil || !handled || result["imported"] != true || result["oob_file"] != hqFile {
		t.Fatalf("first save = %v, %v, %v", result, handled, err)
	}
	files := rawFiles(t, p)
	if len(files) != 1 || files[0].Note != hqRawFileNote || !strings.Contains(files[0].Content, "hq_light_armor = { x = 0 y = 0 }\n\t\t\t}") {
		t.Fatalf("raw file = %+v", files)
	}

	top := indexOver(t, map[string]string{hqFile: files[0].Content}, base)
	in.Regiments = append(in.Regiments, designerSaveCell{"hq_light_armor", 0, 1})
	if result, _, err := save(top, in); err != nil || result["imported"] != false {
		t.Fatalf("second save = %v, %v", result, err)
	}
	if files := rawFiles(t, p); len(files) != 1 || !strings.Contains(files[0].Content, "hq_light_armor = { x = 0 y = 1 }") {
		t.Fatalf("after the second save: %+v", files)
	}

	if _, handled, _ := save(top, designerSaveInput{Tag: "GER", Template: "Infanterie-Division"}); handled {
		t.Error("a battle order template was taken for an Army HQ one")
	}
}

func TestDivisionSaveTakesTheFileIn(t *testing.T) {
	p := newProject(t)
	save := func(in divisionSaveInput) (map[string]any, error) {
		return run(p, func(st *store) (map[string]any, error) { return saveDivision(st, designerIndexFake{}, in) })
	}
	out, err := save(divisionSaveInput{OOB: "GER_1936", Ordinal: 2, Experience: floatPtr(0.9)})
	if err != nil || len(templateRows(t, p)) != 2 {
		t.Fatalf("save: %v %v", out, err)
	}
	tail := templateRows(t, p)[0].OobUnitsTail

	third := tail[strings.LastIndex(tail, "division = {"):]
	if !strings.Contains(third, `"Infanterie-Division"`) || !strings.Contains(third, "start_experience_factor = 0.9") ||
		strings.Count(tail, "start_experience_factor") != 1 {
		t.Errorf("tail = %s", tail)
	}
	unknown := "Kavallerie-Division"
	for _, wrong := range []divisionSaveInput{
		{OOB: "GER_1936", Ordinal: 0},
		{OOB: "GER_1936", Ordinal: 0, Template: &unknown},
		{OOB: "GER_1936", Ordinal: 0, Equipment: floatPtr(1.5)},
		{OOB: "../GER_1936", Ordinal: 0, Experience: floatPtr(0.5)},
		{OOB: "GER_1936", Ordinal: 9, Experience: floatPtr(0.5)},
	} {
		if _, err := save(wrong); statusOf(err) != 422 {
			t.Errorf("%+v: %v", wrong, err)
		}
	}
}

func TestDesignerResultShape(t *testing.T) {
	p := newProject(t)
	out, err := saveDesigner(p, designerIndexFake{}, designerSaveInput{Tag: "GER", Name: "Neu", Priority: intPtr(2), Regiments: []designerSaveCell{{"infantry", 0, 0}}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	var shape map[string]any
	_ = json.Unmarshal(raw, &shape)
	template, _ := shape["template"].(map[string]any)
	for _, key := range []string{"id", "project_id", "oob_file", "template_name", "country_tag", "division_names_group",
		"regiments", "support", "is_locked", "priority", "ordering", "oob_units_tail", "raw_extras", "source", "warnings"} {
		if _, ok := template[key]; !ok {
			t.Errorf("template has no %s: %v", key, template)
		}
	}
	if template["priority"] != 2.0 || shape["imported"] != true || shape["oob_file"] != "GER_1936" {
		t.Errorf("answer = %s", raw)
	}
	if _, present := shape["started_oob"]; present {
		t.Errorf("started_oob written for an existing file: %s", raw)
	}
}
