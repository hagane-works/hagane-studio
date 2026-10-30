package gamesave

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/domains/military"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type designerSaveCell struct {
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type designerSaveInput struct {
	Tag  string `json:"tag"`
	Date string `json:"date"`

	Template   string             `json:"template"`
	Name       string             `json:"name"`
	NamesGroup string             `json:"names_group"`
	Priority   *int               `json:"priority"`
	Regiments  []designerSaveCell `json:"regiments"`
	Support    []designerSaveCell `json:"support"`

	RegimentalSupport []designerSaveCell `json:"regimental_support"`
}

type designerSaveIndex interface {
	TemplateOOB(tag, date, template string) (string, bool)
	OOBSource(name string) ([]byte, bool)

	OOBOf(tag, date string) []string
	RegimentalSupportAvailable(tag, date string) []string
}

var designerTables = []string{divisionTemplateTable, divisionDeploymentTable, rawFilesTable}

func renameTemplateRefs(text, old, name string) string {
	pattern := regexp.MustCompile(`(?i)(division_template\s*=\s*)"` + regexp.QuoteMeta(old) + `"`)
	return pattern.ReplaceAllString(text, `${1}"`+strings.ReplaceAll(name, "$", "$$")+`"`)
}

func carryRename(st *store, oob string, rows []divisionTemplate, old, name string) error {
	for _, row := range rows {
		tail := renameTemplateRefs(row.OobUnitsTail, old, name)
		if tail == row.OobUnitsTail {
			continue
		}
		if _, err := military.UpdateDivisionTemplate(st.tx, row.ID, military.DivisionTemplatePatch{
			OobUnitsTail: workbench.Optional[string]{Set: true, Value: &tail},
		}); err != nil {
			return refusal(err)
		}
	}
	for _, deployment := range st.deployments(oob) {
		if !strings.EqualFold(deployment.DivisionTemplate, old) {
			continue
		}
		if err := st.setDeploymentTemplate(deployment.ID, name); err != nil {
			return err
		}
	}
	return nil
}

type designerSaveResult struct {
	Template divisionTemplate `json:"template"`
	OobFile  string           `json:"oob_file"`

	Imported bool `json:"imported"`

	StartedOOB bool `json:"started_oob,omitempty"`
}

func designerGrid(cells []designerSaveCell) []map[string]any {
	out := make([]map[string]any, 0, len(cells))
	for _, cell := range cells {
		out = append(out, map[string]any{"token": cell.Type, "x": cell.X, "y": cell.Y})
	}
	return out
}

func designerRegimentalExtras(existing map[string]any, cells []designerSaveCell) map[string]any {
	extras := make(map[string]any, len(existing)+1)
	for key, value := range existing {
		extras[key] = value
	}
	if cells == nil {
		return extras
	}

	delete(extras, "regimental_support")
	if len(cells) > 0 {
		block := cellsText("regimental_support", cells, "")
		extras["regimental_support"] = strings.TrimSuffix(strings.SplitN(block, "\n", 2)[1], "}")
	}
	return extras
}

func validateRegimentalSupport(index designerSaveIndex, in designerSaveInput) error {
	if len(in.RegimentalSupport) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, kind := range index.RegimentalSupportAvailable(upperTag(in.Tag), in.Date) {
		allowed[kind] = true
	}
	counts := map[int]int{}
	seenLine := map[[2]int]bool{}
	for _, cell := range in.Regiments {
		key := [2]int{cell.X, cell.Y}
		if cell.X >= 0 && cell.X < 5 && cell.Y >= 0 && cell.Y < 5 && !seenLine[key] {
			counts[cell.X]++
			seenLine[key] = true
		}
	}
	seen := map[int]bool{}
	for _, cell := range in.RegimentalSupport {
		if cell.X < 0 || cell.X >= 5 || cell.Y != 0 || seen[cell.X] || counts[cell.X] < 3 ||
			!allowed[cell.Type] || strings.ContainsAny(cell.Type, " \t\r\n{}=\"#") {
			return saveError{422, "invalid regimental support: one available company per regiment with at least three line battalions"}
		}
		seen[cell.X] = true
	}
	return nil
}

func saveDesignerTemplate(st *store, index designerSaveIndex, in designerSaveInput) (designerSaveResult, error) {
	tag := upperTag(in.Tag)
	name := strings.TrimSpace(in.Name)
	if tag == "" || name == "" || len(name) > 255 || strings.ContainsAny(name, "\"\r\n") {
		return designerSaveResult{}, saveError{422, "tag and a template name are required"}
	}
	if len(in.Regiments) == 0 && len(in.Support) == 0 {
		return designerSaveResult{}, saveError{422, "a template needs at least one battalion"}
	}
	for _, cell := range append(append([]designerSaveCell{}, in.Regiments...), in.Support...) {
		if strings.TrimSpace(cell.Type) == "" || cell.X < 0 || cell.Y < 0 || cell.X > 10 || cell.Y > 10 {
			return designerSaveResult{}, saveError{422, "invalid battalion cell"}
		}
	}
	if err := validateRegimentalSupport(index, in); err != nil {
		return designerSaveResult{}, err
	}
	oob, ok := index.TemplateOOB(tag, in.Date, in.Template)
	started := false
	if !ok {
		if in.Template != "" {
			return designerSaveResult{}, saveError{422, "this template is not in a battle order file (Army HQ templates cannot be saved here yet)"}
		}
		var err error
		if oob, started, err = newTemplateOOB(st, index, tag, in.Date); err != nil {
			return designerSaveResult{}, err
		}
	}

	if result, handled, err := saveDesignerTemplateInFile(st, index, in, tag, name, oob); handled || err != nil {
		return result, err
	}

	var rows []divisionTemplate
	imported := false
	if !started {
		var err error
		if rows, imported, err = takeOOBIn(st, index, oob); err != nil {
			return designerSaveResult{}, err
		}
	}
	var target *divisionTemplate
	ordering := 0
	for i := range rows {
		if rows[i].Ordering >= ordering {
			ordering = rows[i].Ordering + 1
		}
		if in.Template != "" && strings.EqualFold(rows[i].TemplateName, in.Template) {
			target = &rows[i]
		}
	}
	for i := range rows {
		if strings.EqualFold(rows[i].TemplateName, name) && (target == nil || rows[i].ID != target.ID) {
			return designerSaveResult{}, saveError{409, "a template with this name already exists"}
		}
	}
	regiments, support := designerGrid(in.Regiments), designerGrid(in.Support)
	if target == nil {
		if in.Template != "" {
			return designerSaveResult{}, saveError{404, "template not found in the project"}
		}
		created, err := military.CreateDivisionTemplate(st.tx, military.DivisionTemplateCreateInput{
			OobFile: oob, TemplateName: &name, CountryTag: tag, DivisionNamesGroup: in.NamesGroup,
			Regiments: regiments, Support: support, Priority: in.Priority, Ordering: ordering,
			RawExtras: designerRegimentalExtras(nil, in.RegimentalSupport),
		})
		if err != nil {
			return designerSaveResult{}, refusal(err)
		}
		return designerSaveResult{Template: created, OobFile: oob, Imported: imported, StartedOOB: started}, nil
	}
	patch := military.DivisionTemplatePatch{
		TemplateName:       workbench.Optional[string]{Set: true, Value: &name},
		DivisionNamesGroup: workbench.Optional[string]{Set: true, Value: &in.NamesGroup},
		Regiments:          workbench.Optional[[]map[string]any]{Set: true, Value: &regiments},
		Support:            workbench.Optional[[]map[string]any]{Set: true, Value: &support},
	}
	if in.RegimentalSupport != nil {
		extras := designerRegimentalExtras(target.RawExtras, in.RegimentalSupport)
		patch.RawExtras = workbench.Optional[map[string]any]{Set: true, Value: &extras}
	}
	if in.Priority != nil {
		patch.Priority = workbench.Optional[int]{Set: true, Value: in.Priority}
	}
	updated, err := military.UpdateDivisionTemplate(st.tx, target.ID, patch)
	if err != nil {
		return designerSaveResult{}, refusal(err)
	}
	if target.TemplateName != name {
		if err := carryRename(st, oob, rows, target.TemplateName, name); err != nil {
			return designerSaveResult{}, err
		}
	}
	return designerSaveResult{Template: updated, OobFile: oob, Imported: imported}, nil
}

func newTemplateOOB(st *store, index designerSaveIndex, tag, date string) (string, bool, error) {
	named := index.OOBOf(tag, date)
	for _, row := range st.templates(nil, &tag) {
		if row.OobFile != "" {
			return row.OobFile, false, nil
		}
	}
	for _, oob := range named {
		if oob = strings.Trim(oob, "\""); oob != "" {
			if _, exists := index.OOBSource(oob); !exists {
				return oob, true, nil
			}
		}
	}
	base := tag
	if year, _, _ := strings.Cut(strings.TrimSpace(date), "."); len(year) == 4 && strings.Trim(year, "0123456789") == "" {
		base += "_" + year
	}
	for n := 1; n < 100; n++ {
		name := base
		if n > 1 {
			name = fmt.Sprintf("%s_%d", base, n)
		}
		if _, exists := index.OOBSource(name); exists {
			continue
		}
		if len(st.templates(&name, nil)) == 0 {
			return name, true, nil
		}
	}
	return "", false, saveError{409, "no free battle order file name for " + tag}
}

func takeOOBIn(st *store, index designerSaveIndex, oob string) ([]divisionTemplate, bool, error) {
	if rows := st.templates(&oob, nil); len(rows) > 0 {
		return rows, false, nil
	}
	source, ok := index.OOBSource(oob)
	if !ok {
		return nil, false, saveError{422, "the battle order file's text is not available"}
	}
	if _, err := military.ImportDivisionTemplatesText(st.tx, string(source), oob, "skip"); err != nil {
		return nil, false, refusal(err)
	}
	return st.templates(&oob, nil), true, nil
}

func (h *handlers) oobTake(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in struct {
		Tag  string `json:"tag"`
		Date string `json:"date"`
	}
	if err := decodeJSON(w, r, 1<<16, &in); err != nil || strings.TrimSpace(in.Tag) == "" {
		detail(w, http.StatusUnprocessableEntity, "tag is required")
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}
	oob, ok := index.TemplateOOB(upperTag(in.Tag), in.Date, "")
	if !ok {
		detail(w, http.StatusUnprocessableEntity, "the country has no battle order file")
		return
	}
	imported, err := write(p, designerTables, func(st *store) (bool, error) {
		_, imported, err := takeOOBIn(st, index, oob)
		return imported, err
	})
	h.finish(w, p, map[string]any{"oob_file": oob, "imported": imported}, err, "taking the battle order in", imported)
}

func (h *handlers) designerSave(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in designerSaveInput
	if !decodeBody(w, r, 1<<20, &in, "invalid designer payload") {
		return
	}
	index, ok := h.loadIndex(w, r, p)
	if !ok {
		return
	}

	isHQ := false
	if strings.TrimSpace(in.Template) != "" {
		_, inOOB := index.TemplateOOB(upperTag(in.Tag), in.Date, in.Template)
		isHQ = !inOOB
	}

	op := "designer save"
	out, err := write(p, designerTables, func(st *store) (any, error) {
		if isHQ {
			hq, handled, err := saveHQTemplate(st, index, in)
			if handled {
				op = "army HQ template save"
				return hq, err
			}
		}
		return saveDesignerTemplate(st, index, in)
	})
	h.finish(w, p, out, err, op, true)
}
