package military

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	divisionTemplateTable   = "division_templates"
	divisionDeploymentTable = "division_deployments"
)

var (
	errDivisionTemplateMissing   = apiutil.NotFoundError("Division template not found")
	errDivisionDeploymentMissing = apiutil.NotFoundError("Division deployment not found")
	errDivisionUnavailable       = detailError(http.StatusServiceUnavailable, "division store unavailable")
)

func tagFromOOB(oobFile string) string {
	stem := upper(strings.SplitN(oobFile, "_", 2)[0])
	if len(stem) > 10 {
		stem = stem[:10]
	}
	return stem
}

type DivisionTemplate struct {
	ID                 int64            `json:"id"`
	ProjectID          int64            `json:"project_id"`
	OobFile            string           `json:"oob_file"`
	TemplateName       string           `json:"template_name"`
	CountryTag         string           `json:"country_tag"`
	DivisionNamesGroup string           `json:"division_names_group"`
	Regiments          []map[string]any `json:"regiments"`
	Support            []map[string]any `json:"support"`
	IsLocked           *bool            `json:"is_locked"`
	Priority           *int             `json:"priority"`
	Ordering           int              `json:"ordering"`
	OobUnitsTail       string           `json:"oob_units_tail"`
	RawExtras          map[string]any   `json:"raw_extras"`
	Source             string           `json:"source"`
	Warnings           []string         `json:"warnings"`
}

func divisionTemplateFromRow(row project.Row) DivisionTemplate {
	var t DivisionTemplate
	fromRow(row, map[string]any{"source": "created"}, &t)
	if t.Regiments == nil {
		t.Regiments = []map[string]any{}
	}
	if t.Support == nil {
		t.Support = []map[string]any{}
	}
	if t.RawExtras == nil {
		t.RawExtras = map[string]any{}
	}
	t.Warnings = []string{}
	return t
}

func listDivisionTemplates(p *project.Project, oobFile, countryTag *string) ([]DivisionTemplate, error) {
	rows, err := p.Rows(divisionTemplateTable)
	if err != nil {
		return nil, err
	}
	out := []DivisionTemplate{}
	for _, row := range rows {
		if oobFile != nil && !stringEquals(row["oob_file"], *oobFile) {
			continue
		}
		if countryTag != nil && !stringEquals(row["country_tag"], *countryTag) {
			continue
		}
		out = append(out, divisionTemplateFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.OobFile != b.OobFile {
			return a.OobFile < b.OobFile
		}
		if a.Ordering != b.Ordering {
			return a.Ordering < b.Ordering
		}
		return a.TemplateName < b.TemplateName
	})
	return out, nil
}

func getDivisionTemplate(p *project.Project, id int64) (DivisionTemplate, error) {
	rows, err := p.Rows(divisionTemplateTable)
	if err != nil {
		return DivisionTemplate{}, err
	}
	if row := findRow(rows, id); row != nil {
		return divisionTemplateFromRow(row), nil
	}
	return DivisionTemplate{}, errDivisionTemplateMissing
}

type DivisionTemplateCreateInput struct {
	OobFile            string           `json:"oob_file"`
	TemplateName       *string          `json:"template_name"`
	CountryTag         string           `json:"country_tag"`
	DivisionNamesGroup string           `json:"division_names_group"`
	Regiments          []map[string]any `json:"regiments"`
	Support            []map[string]any `json:"support"`
	RawExtras          map[string]any   `json:"raw_extras"`
	IsLocked           *bool            `json:"is_locked"`
	Priority           *int             `json:"priority"`
	Ordering           int              `json:"ordering"`
	Source             *string          `json:"source"`
}

func createDivisionTemplate(write writer, in DivisionTemplateCreateInput) (DivisionTemplate, error) {
	templateName := strings.TrimSpace(stringPtr(in.TemplateName, "New Division"))
	if templateName == "" {
		templateName = "New Division"
	}
	oobFile := strings.TrimSpace(in.OobFile)
	if oobFile == "" {
		oobFile = "untitled"
	}
	countryTag := in.CountryTag
	if countryTag == "" {
		countryTag = tagFromOOB(oobFile)
	}
	if in.Regiments == nil {
		in.Regiments = []map[string]any{}
	}
	if in.Support == nil {
		in.Support = []map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	var created DivisionTemplate
	err := write([]string{divisionTemplateTable}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(divisionTemplateTable) {
			if text(row, "oob_file") == oobFile {
				taken[text(row, "template_name")] = true
			}
		}
		base, suffix := templateName, 0
		for taken[templateName] {
			suffix++
			templateName = fmt.Sprintf("%s %d", base, suffix)
			if suffix > 1000 {
				return detailError(http.StatusConflict, "template_name collision (cap 1000)")
			}
		}
		row := tx.Insert(divisionTemplateTable, project.Row{
			"oob_file": oobFile, "template_name": templateName, "country_tag": countryTag,
			"division_names_group": in.DivisionNamesGroup,
			"regiments":            jsonValue(in.Regiments), "support": jsonValue(in.Support),
			"is_locked": jsonValue(in.IsLocked), "priority": jsonValue(in.Priority),
			"ordering": jsonValue(in.Ordering), "oob_units_tail": "",
			"raw_extras": jsonValue(in.RawExtras), "source": stringPtr(in.Source, "created"),
		})
		created = divisionTemplateFromRow(row)
		return nil
	})
	return created, err
}

type DivisionTemplatePatch struct {
	OobFile            workbench.Optional[string]           `json:"oob_file"`
	TemplateName       workbench.Optional[string]           `json:"template_name"`
	CountryTag         workbench.Optional[string]           `json:"country_tag"`
	DivisionNamesGroup workbench.Optional[string]           `json:"division_names_group"`
	Regiments          workbench.Optional[[]map[string]any] `json:"regiments"`
	Support            workbench.Optional[[]map[string]any] `json:"support"`
	IsLocked           workbench.Optional[bool]             `json:"is_locked"`
	Priority           workbench.Optional[int]              `json:"priority"`
	Ordering           workbench.Optional[int]              `json:"ordering"`
	OobUnitsTail       workbench.Optional[string]           `json:"oob_units_tail"`
	RawExtras          workbench.Optional[map[string]any]   `json:"raw_extras"`
	Source             workbench.Optional[string]           `json:"source"`
}

func sourceChange(changes map[string]any, current project.Row, source workbench.Optional[string]) {
	flip := false
	if textOr(current, "source", "created") == "imported" {
		final := "imported"
		if source.Set && source.Value != nil {
			final = *source.Value
		}
		flip = final == "imported"
	}
	switch {
	case flip:
		changes["source"] = "created"
	case source.Set:
		changes["source"] = opt(source)
	}
}

func updateDivisionTemplate(write writer, id int64, patch DivisionTemplatePatch) (DivisionTemplate, error) {
	var updated DivisionTemplate
	err := write([]string{divisionTemplateTable}, func(tx *project.Tx) error {
		current := tx.Find(divisionTemplateTable, id)
		if current == nil {
			return errDivisionTemplateMissing
		}
		changes := map[string]any{}
		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		if patch.TemplateName.Set {
			changes["template_name"] = strings.TrimSpace(optStr(patch.TemplateName))
		}
		if patch.OobFile.Set {
			changes["oob_file"] = strings.TrimSpace(optStr(patch.OobFile))
		}
		set("country_tag", patch.CountryTag.Set, opt(patch.CountryTag))
		set("division_names_group", patch.DivisionNamesGroup.Set, opt(patch.DivisionNamesGroup))
		set("regiments", patch.Regiments.Set, opt(patch.Regiments))
		set("support", patch.Support.Set, opt(patch.Support))
		set("is_locked", patch.IsLocked.Set, opt(patch.IsLocked))
		set("priority", patch.Priority.Set, opt(patch.Priority))
		set("ordering", patch.Ordering.Set, opt(patch.Ordering))
		set("oob_units_tail", patch.OobUnitsTail.Set, opt(patch.OobUnitsTail))
		set("raw_extras", patch.RawExtras.Set, opt(patch.RawExtras))
		sourceChange(changes, current, patch.Source)
		if len(changes) == 0 {
			updated = divisionTemplateFromRow(current)
			return nil
		}
		if nullViolation(divisionTemplateTable, changes) {
			return errDivisionUnavailable
		}
		oobFile, name := text(current, "oob_file"), text(current, "template_name")
		if v, ok := changes["oob_file"].(string); ok {
			oobFile = v
		}
		if v, ok := changes["template_name"].(string); ok {
			name = v
		}
		for _, row := range tx.Rows(divisionTemplateTable) {
			if project.Int64(row["id"]) != id && text(row, "oob_file") == oobFile && text(row, "template_name") == name {
				return detailError(http.StatusConflict, "division template collision")
			}
		}
		row, err := tx.Update(divisionTemplateTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = divisionTemplateFromRow(row)
		return nil
	})
	return updated, err
}

func deleteRow(p *project.Project, table string, id int64, missing error) error {
	return p.Write([]string{table}, func(tx *project.Tx) error {
		if tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return missing
		}
		return nil
	})
}

func (h *handlers) divisionTemplates(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	const missing = "Division template not found"
	failDivision := func(err error, op string) { fail(w, err, op, "division store unavailable") }
	if len(rest) == 1 && rest[0] == "import" {
		h.divisionTemplateImport(w, r, p)
		return
	}
	if len(rest) == 2 && rest[0] == "preview" {
		divisionTemplatePreview(w, r, p, rest[1])
		return
	}
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := listDivisionTemplates(p, queryPtr(r, "oob_file"), queryPtr(r, "country_tag"))
			if err != nil {
				failDivision(err, "list division templates")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var in DivisionTemplateCreateInput
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid division template payload")
				return
			}
			item, err := createDivisionTemplate(p.Write, in)
			if err != nil {
				failDivision(err, "create division template")
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			methodError(w, "GET or POST only")
		}
		return
	}
	id, ok := itemID(rest)
	if !ok {
		detail(w, http.StatusNotFound, missing)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := getDivisionTemplate(p, id)
		if err != nil {
			failDivision(err, "get division template")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch DivisionTemplatePatch
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid division template payload")
			return
		}
		item, err := updateDivisionTemplate(p.Write, id, patch)
		if err != nil {
			failDivision(err, "update division template")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, divisionTemplateTable, id, errDivisionTemplateMissing); err != nil {
			failDivision(err, "delete division template")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}

func divisionTemplatePreview(w http.ResponseWriter, r *http.Request, p *project.Project, rawID string) {
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, ok := previewID(w, rawID, "template_id", "Division template not found")
	if !ok {
		return
	}
	item, err := getDivisionTemplate(p, id)
	if err != nil {
		previewFail(w, err, "division template preview")
		return
	}
	writePreview(w, item, emitmisc.GenerateSingleDivisionTemplateScript, "division template preview emit")
}

type DivisionDeployment struct {
	ID                    int64          `json:"id"`
	ProjectID             int64          `json:"project_id"`
	OobFile               string         `json:"oob_file"`
	CountryTag            string         `json:"country_tag"`
	DivisionTemplate      string         `json:"division_template"`
	Location              *int           `json:"location"`
	Count                 int            `json:"count"`
	DivisionName          string         `json:"division_name"`
	StartExperienceFactor *float64       `json:"start_experience_factor"`
	StartEquipmentFactor  *float64       `json:"start_equipment_factor"`
	Ordering              int            `json:"ordering"`
	RawExtras             map[string]any `json:"raw_extras"`
	Source                string         `json:"source"`
	Warnings              []string       `json:"warnings"`
}

func divisionDeploymentFromRow(row project.Row) DivisionDeployment {
	var d DivisionDeployment
	fromRow(row, map[string]any{"source": "created", "count": 1}, &d)
	if d.RawExtras == nil {
		d.RawExtras = map[string]any{}
	}
	d.Warnings = []string{}
	return d
}

type deploymentValidationIndex struct {
	ownedByTag map[string]map[int]bool

	ownedStatesByTag map[string]map[int]bool

	knownTags      map[string]bool
	templatesByOOB map[string]map[string]bool
	allTemplates   map[string]bool
}

func loadDeploymentValidationIndex(p *project.Project, needOwned, needTemplates bool) (deploymentValidationIndex, error) {
	index := deploymentValidationIndex{
		ownedByTag: map[string]map[int]bool{}, ownedStatesByTag: map[string]map[int]bool{},
		knownTags:      map[string]bool{},
		templatesByOOB: map[string]map[string]bool{}, allTemplates: map[string]bool{},
	}
	if needOwned {
		states, err := p.Rows("states")
		if err != nil {
			return index, err
		}
		for _, state := range states {
			tag := upper(text(state, "owner"))
			if tag == "" {
				continue
			}
			owned := index.ownedByTag[tag]
			if owned == nil {
				owned = map[int]bool{}
				index.ownedByTag[tag] = owned
			}
			ownedStates := index.ownedStatesByTag[tag]
			if ownedStates == nil {
				ownedStates = map[int]bool{}
				index.ownedStatesByTag[tag] = ownedStates
			}
			if state["state_id"] != nil {
				ownedStates[int(project.Int64(state["state_id"]))] = true
			}
			index.knownTags[tag] = true
			provinces, _ := state["provinces"].([]any)
			for _, province := range provinces {
				switch value := province.(type) {
				case string:
					var id int
					if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &id); err == nil {
						owned[id] = true
					}
				default:
					if province != nil {
						owned[int(number(value))] = true
					}
				}
			}
		}
		countries, err := p.Rows("countries")
		if err != nil {
			return index, err
		}
		for _, country := range countries {
			if tag := upper(text(country, "tag")); tag != "" {
				index.knownTags[tag] = true
			}
		}
	}
	if needTemplates {
		templates, err := p.Rows(divisionTemplateTable)
		if err != nil {
			return index, err
		}
		for _, template := range templates {
			oobFile, name := text(template, "oob_file"), text(template, "template_name")
			if name == "" {
				continue
			}
			index.allTemplates[name] = true
			byOOB := index.templatesByOOB[oobFile]
			if byOOB == nil {
				byOOB = map[string]bool{}
				index.templatesByOOB[oobFile] = byOOB
			}
			byOOB[name] = true
		}
	}
	return index, nil
}

const warnNoLocation = "actionable:no_location"

type oobOwnershipScope struct {
	unownedPrefix string
	byState       bool
}

var (
	provinceOwnership = oobOwnershipScope{unownedPrefix: "actionable:unowned_province|province="}
	stateOwnership    = oobOwnershipScope{unownedPrefix: "actionable:unowned_state|state=", byState: true}
)

func (index deploymentValidationIndex) hasOwnedStates() bool {
	return len(index.ownedByTag) > 0
}

func (index deploymentValidationIndex) ownership(scope oobOwnershipScope, tag string) (map[int]bool, bool) {
	source := index.ownedByTag
	if scope.byState {
		source = index.ownedStatesByTag
	}
	owned, owns := source[tag]
	return owned, owns || index.knownTags[tag]
}

func oobOwnershipWarning(scope oobOwnershipScope, tag string, location int, owned map[int]bool, knownTag, hasOwnedStates bool) string {
	if !hasOwnedStates {
		return ""
	}
	if !knownTag || tag == "" {
		displayTag := tag
		if displayTag == "" {
			displayTag = "?"
		}
		return "actionable:unknown_tag|tag=" + displayTag
	}
	if len(owned) > 0 && !owned[location] {
		return scope.unownedPrefix + strconv.Itoa(location) + "|tag=" + tag
	}
	return ""
}

func deploymentLocationWarning(tag string, location int, owned map[int]bool, knownTag, hasOwnedStates bool) string {
	return oobOwnershipWarning(provinceOwnership, tag, location, owned, knownTag, hasOwnedStates)
}

func oobTokenOwnershipWarnings(scope oobOwnershipScope, tag, rawLocation string, index deploymentValidationIndex) []string {
	warnings := []string{}
	token := strings.TrimSpace(rawLocation)
	if token == "" {
		return append(warnings, warnNoLocation)
	}
	owned, knownTag := index.ownership(scope, tag)
	location, err := strconv.Atoi(token)
	if err != nil {
		location, owned = -1, nil
	}
	if w := oobOwnershipWarning(scope, tag, location, owned, knownTag, index.hasOwnedStates()); w != "" {
		warnings = append(warnings, w)
	}
	return warnings
}

func oobTagOf(countryTag, oobFile string) string {
	tag := upper(countryTag)
	if tag == "" {
		tag = upper(tagFromOOB(oobFile))
	}
	return tag
}

func validateDeploymentIndexed(d DivisionDeployment, index deploymentValidationIndex) []string {
	warnings := []string{}
	tag := oobTagOf(d.CountryTag, d.OobFile)
	if d.Location == nil {
		warnings = append(warnings, warnNoLocation)
	} else {
		owned, knownTag := index.ownership(provinceOwnership, tag)
		if w := deploymentLocationWarning(tag, *d.Location, owned, knownTag, index.hasOwnedStates()); w != "" {
			warnings = append(warnings, w)
		}
	}
	template := strings.TrimSpace(d.DivisionTemplate)
	if template == "" {
		warnings = append(warnings, "info:no_template")
	} else if !index.templatesByOOB[d.OobFile][template] && !index.allTemplates[template] {

		warnings = append(warnings, "info:missing_template|template="+template)
	}
	return warnings
}

func withDeploymentWarnings(p *project.Project, deployments []DivisionDeployment) error {
	needOwned, needTemplates := false, false
	for _, d := range deployments {
		needOwned = needOwned || d.Location != nil
		needTemplates = needTemplates || strings.TrimSpace(d.DivisionTemplate) != ""
	}
	if len(deployments) == 0 {
		return nil
	}
	index, err := loadDeploymentValidationIndex(p, needOwned, needTemplates)
	if err != nil {
		return err
	}
	for i := range deployments {
		deployments[i].Warnings = validateDeploymentIndexed(deployments[i], index)
	}
	return nil
}

func listDivisionDeployments(p *project.Project, oobFile, countryTag *string) ([]DivisionDeployment, error) {
	rows, err := p.Rows(divisionDeploymentTable)
	if err != nil {
		return nil, err
	}
	out := []DivisionDeployment{}
	for _, row := range rows {
		if oobFile != nil && !stringEquals(row["oob_file"], *oobFile) {
			continue
		}
		if countryTag != nil && !stringEquals(row["country_tag"], *countryTag) {
			continue
		}
		out = append(out, divisionDeploymentFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.OobFile != b.OobFile {
			return a.OobFile < b.OobFile
		}
		if a.Ordering != b.Ordering {
			return a.Ordering < b.Ordering
		}
		return a.ID < b.ID
	})
	if err := withDeploymentWarnings(p, out); err != nil {
		return nil, err
	}
	return out, nil
}

func getDivisionDeployment(p *project.Project, id int64) (DivisionDeployment, error) {
	rows, err := p.Rows(divisionDeploymentTable)
	if err != nil {
		return DivisionDeployment{}, err
	}
	row := findRow(rows, id)
	if row == nil {
		return DivisionDeployment{}, errDivisionDeploymentMissing
	}
	items := []DivisionDeployment{divisionDeploymentFromRow(row)}
	if err := withDeploymentWarnings(p, items); err != nil {
		return DivisionDeployment{}, err
	}
	return items[0], nil
}

type divisionDeploymentCreateInput struct {
	OobFile               string         `json:"oob_file"`
	CountryTag            string         `json:"country_tag"`
	DivisionTemplate      string         `json:"division_template"`
	Location              *int           `json:"location"`
	Count                 *int           `json:"count"`
	DivisionName          string         `json:"division_name"`
	StartExperienceFactor *float64       `json:"start_experience_factor"`
	StartEquipmentFactor  *float64       `json:"start_equipment_factor"`
	Ordering              int            `json:"ordering"`
	RawExtras             map[string]any `json:"raw_extras"`
	Source                *string        `json:"source"`
}

const countRangeDetail = "count must be between 1 and 1000"

func createDivisionDeployment(p *project.Project, in divisionDeploymentCreateInput) (DivisionDeployment, error) {
	count := 1
	if in.Count != nil {
		count = *in.Count
	}
	if count < 1 || count > 1000 {
		return DivisionDeployment{}, detailError(http.StatusUnprocessableEntity, countRangeDetail)
	}
	oobFile := strings.TrimSpace(in.OobFile)
	countryTag := in.CountryTag
	if countryTag == "" {
		countryTag = tagFromOOB(oobFile)
	} else {
		countryTag = upper(countryTag)
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	var id int64
	err := p.Write([]string{divisionDeploymentTable}, func(tx *project.Tx) error {
		row := tx.Insert(divisionDeploymentTable, project.Row{
			"oob_file": oobFile, "country_tag": countryTag,
			"division_template": strings.TrimSpace(in.DivisionTemplate),
			"location":          jsonValue(in.Location), "count": jsonValue(count),
			"division_name":           strings.TrimSpace(in.DivisionName),
			"start_experience_factor": jsonValue(in.StartExperienceFactor),
			"start_equipment_factor":  jsonValue(in.StartEquipmentFactor),
			"ordering":                jsonValue(in.Ordering), "raw_extras": jsonValue(in.RawExtras),
			"source": stringPtr(in.Source, "created"),
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return DivisionDeployment{}, err
	}
	return getDivisionDeployment(p, id)
}

type divisionDeploymentPatch struct {
	OobFile               workbench.Optional[string]         `json:"oob_file"`
	CountryTag            workbench.Optional[string]         `json:"country_tag"`
	DivisionTemplate      workbench.Optional[string]         `json:"division_template"`
	Location              workbench.Optional[int]            `json:"location"`
	Count                 workbench.Optional[int]            `json:"count"`
	DivisionName          workbench.Optional[string]         `json:"division_name"`
	StartExperienceFactor workbench.Optional[float64]        `json:"start_experience_factor"`
	StartEquipmentFactor  workbench.Optional[float64]        `json:"start_equipment_factor"`
	Ordering              workbench.Optional[int]            `json:"ordering"`
	RawExtras             workbench.Optional[map[string]any] `json:"raw_extras"`
	Source                workbench.Optional[string]         `json:"source"`
}

func updateDivisionDeployment(p *project.Project, id int64, patch divisionDeploymentPatch) (DivisionDeployment, error) {
	if patch.Count.Set && patch.Count.Value != nil && *patch.Count.Value > 1000 {
		return DivisionDeployment{}, detailError(http.StatusUnprocessableEntity, countRangeDetail)
	}
	err := p.Write([]string{divisionDeploymentTable}, func(tx *project.Tx) error {
		current := tx.Find(divisionDeploymentTable, id)
		if current == nil {
			return errDivisionDeploymentMissing
		}
		changes := map[string]any{}
		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		for column, field := range map[string]workbench.Optional[string]{
			"oob_file": patch.OobFile, "division_template": patch.DivisionTemplate, "division_name": patch.DivisionName,
		} {
			set(column, field.Set, strings.TrimSpace(optStr(field)))
		}
		set("country_tag", patch.CountryTag.Set, upper(optStr(patch.CountryTag)))
		set("location", patch.Location.Set, opt(patch.Location))
		if patch.Count.Set {
			count := 1
			if patch.Count.Value != nil && *patch.Count.Value >= 1 {
				count = *patch.Count.Value
			}
			changes["count"] = jsonValue(count)
		}
		set("start_experience_factor", patch.StartExperienceFactor.Set, opt(patch.StartExperienceFactor))
		set("start_equipment_factor", patch.StartEquipmentFactor.Set, opt(patch.StartEquipmentFactor))
		set("ordering", patch.Ordering.Set, opt(patch.Ordering))
		set("raw_extras", patch.RawExtras.Set, opt(patch.RawExtras))
		sourceChange(changes, current, patch.Source)
		if len(changes) == 0 {
			return nil
		}
		if nullViolation(divisionDeploymentTable, changes) {
			return errDivisionUnavailable
		}
		_, err := tx.Update(divisionDeploymentTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		return err
	})
	if err != nil {
		return DivisionDeployment{}, err
	}
	return getDivisionDeployment(p, id)
}

func (h *handlers) divisionDeployments(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	const missing = "Division deployment not found"
	failDivision := func(err error, op string) { fail(w, err, op, "division store unavailable") }
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := listDivisionDeployments(p, queryPtr(r, "oob_file"), queryPtr(r, "country_tag"))
			if err != nil {
				failDivision(err, "list division deployments")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var in divisionDeploymentCreateInput
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid division deployment payload")
				return
			}
			item, err := createDivisionDeployment(p, in)
			if err != nil {
				failDivision(err, "create division deployment")
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			methodError(w, "GET or POST only")
		}
		return
	}
	id, ok := itemID(rest)
	if !ok {
		detail(w, http.StatusNotFound, missing)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := getDivisionDeployment(p, id)
		if err != nil {
			failDivision(err, "get division deployment")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch divisionDeploymentPatch
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid division deployment payload")
			return
		}
		item, err := updateDivisionDeployment(p, id, patch)
		if err != nil {
			failDivision(err, "update division deployment")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, divisionDeploymentTable, id, errDivisionDeploymentMissing); err != nil {
			failDivision(err, "delete division deployment")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}
