package toolapi

import (
	"context"
	"fmt"
)

func emptyIfNilDicts(s []map[string]any) []map[string]any {
	if s == nil {
		return []map[string]any{}
	}
	return s
}

type createDivisionTemplateIn struct {
	ProjectID          int64            `json:"project_id"`
	OobFile            string           `json:"oob_file,omitempty" jsonschema:"the OOB group stem, e.g. 'USA_1936' → history/units/USA_1936.txt; shared with add_division_deployment"`
	TemplateName       string           `json:"template_name,omitempty" jsonschema:"defaults to 'New Division'; AUTO-SUFFIXES on collision — read back the response"`
	CountryTag         string           `json:"country_tag,omitempty" jsonschema:"derived from the oob_file prefix when blank"`
	DivisionNamesGroup string           `json:"division_names_group,omitempty"`
	Regiments          []map[string]any `json:"regiments,omitempty" jsonschema:"line battalions: list of {token, x, y} dicts"`
	Support            []map[string]any `json:"support,omitempty" jsonschema:"support companies: list of {token, x, y} dicts"`
	IsLocked           *bool            `json:"is_locked,omitempty"`
	Priority           *int             `json:"priority,omitempty"`
	Ordering           int              `json:"ordering,omitempty"`
}

type createDivisionTemplateOut struct {
	TemplateID   int64  `json:"template_id"`
	TemplateName string `json:"template_name"`
	OobFile      string `json:"oob_file"`
	CountryTag   string `json:"country_tag"`
}

func registerCreateDivisionTemplate(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_division_template",
		Description: "Create an OOB division template (a country's starting-army division design).\n\n" +
			"oob_file = the OOB group stem (e.g. 'USA_1936' → history/units/USA_1936.txt); " +
			"also derives country_tag from its prefix when country_tag is blank. Shared " +
			"group key with add_division_deployment — use the SAME oob_file for both.\n\n" +
			"regiments = line battalions, support = support companies; each is a list of " +
			"{token, x, y} dicts (subunit token from the vanilla subunit list + its " +
			"combat-width column/row). There is NO modifier field for this domain.\n\n" +
			"template_name AUTO-SUFFIXES on an (oob_file, template_name) collision " +
			"(' 1', ' 2', …) — READ BACK the returned template_name; it may differ from " +
			"what you sent. Reference that returned name from add_division_deployment.",
	}, func(ctx context.Context, _ *CallToolRequest, in createDivisionTemplateIn) (*CallToolResult, createDivisionTemplateOut, error) {
		templateName := in.TemplateName
		if templateName == "" {
			templateName = "New Division"
		}
		body := map[string]any{
			"oob_file":             in.OobFile,
			"template_name":        templateName,
			"country_tag":          in.CountryTag,
			"division_names_group": in.DivisionNamesGroup,
			"regiments":            emptyIfNilDicts(in.Regiments),
			"support":              emptyIfNilDicts(in.Support),
			"is_locked":            in.IsLocked,
			"priority":             in.Priority,
			"ordering":             in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/division-templates", in.ProjectID), body)
		if err != nil {
			return nil, createDivisionTemplateOut{}, err
		}

		return nil, createDivisionTemplateOut{
			TemplateID:   int64Field(resp, "id"),
			TemplateName: stringField(resp, "template_name"),
			OobFile:      stringField(resp, "oob_file"),
			CountryTag:   stringField(resp, "country_tag"),
		}, nil
	})
}

type addDivisionDeploymentIn struct {
	ProjectID             int64          `json:"project_id"`
	OobFile               string         `json:"oob_file,omitempty" jsonschema:"the SAME OOB group stem used by create_division_template"`
	CountryTag            string         `json:"country_tag,omitempty" jsonschema:"derived from the oob_file prefix when blank"`
	DivisionTemplate      string         `json:"division_template,omitempty" jsonschema:"NAME string ref to a template's template_name (NOT an FK) — create the template FIRST"`
	Location              *int64         `json:"location,omitempty" jsonschema:"the province id (a bare int) where the division spawns"`
	Count                 *int           `json:"count,omitempty" jsonschema:"batch-expands into N identical divisions on export (default 1, max 1000)"`
	DivisionName          string         `json:"division_name,omitempty"`
	StartExperienceFactor *float64       `json:"start_experience_factor,omitempty"`
	StartEquipmentFactor  *float64       `json:"start_equipment_factor,omitempty"`
	Ordering              int            `json:"ordering,omitempty"`
	RawExtras             map[string]any `json:"raw_extras,omitempty"`
}

type addDivisionDeploymentOut struct {
	DeploymentID     int64    `json:"deployment_id"`
	DivisionTemplate string   `json:"division_template"`
	Location         *int64   `json:"location"`
	Count            int64    `json:"count"`
	OobFile          string   `json:"oob_file"`
	CountryTag       string   `json:"country_tag"`
	Warnings         []string `json:"warnings"`
}

func nullableInt64Field(m map[string]any, key string) *int64 {
	if f, ok := m[key].(float64); ok {
		v := int64(f)
		return &v
	}
	return nil
}

func registerAddDivisionDeployment(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "add_division_deployment",
		Description: "Place a division template at a province — a country's starting-army unit.\n\n" +
			"oob_file = the SAME OOB group stem used by create_division_template (both " +
			"export into history/units/<oob_file>.txt). country_tag auto-derives from the " +
			"oob_file prefix when blank.\n\n" +
			"division_template is a NAME string ref to a template's template_name (create " +
			"the template FIRST, then reference the returned name here) — NOT an FK. " +
			"location = the province id (a bare int) where the division spawns. " +
			"count batch-expands into N identical divisions on export (default 1).\n\n" +
			"Never a hard error — every add succeeds and returns a two-tier `warnings` " +
			"list of machine-parsable codes (`<tier>:<code>[|param=value...]`) you " +
			"SHOULD surface to the author:\n" +
			"  - 'actionable:no_location' → no spawn province set.\n" +
			"  - 'actionable:unowned_province|province=..|tag=..' → the division " +
			"would spawn on foreign/unowned territory.\n" +
			"  - 'actionable:unknown_tag|tag=..' → the country tag this deployment " +
			"resolves to is not one this project defines. Usually the oob_file has " +
			"no country prefix (the '<TAG>_<year>' convention, e.g. 'USA_1936'), so " +
			"the tag was derived from a meaningless stem.\n" +
			"  - 'info:missing_template|template=..' / 'info:no_template' → the " +
			"referenced template isn't defined yet in the Division Templates " +
			"workbench (define it there or the OOB won't load).",
	}, func(ctx context.Context, _ *CallToolRequest, in addDivisionDeploymentIn) (*CallToolResult, addDivisionDeploymentOut, error) {
		count := 1
		if in.Count != nil {
			count = *in.Count
		}
		body := map[string]any{
			"oob_file":                in.OobFile,
			"country_tag":             in.CountryTag,
			"division_template":       in.DivisionTemplate,
			"location":                in.Location,
			"count":                   count,
			"division_name":           in.DivisionName,
			"start_experience_factor": in.StartExperienceFactor,
			"start_equipment_factor":  in.StartEquipmentFactor,
			"ordering":                in.Ordering,
			"raw_extras":              emptyIfNilMap(in.RawExtras),
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/division-deployments", in.ProjectID), body)
		if err != nil {
			return nil, addDivisionDeploymentOut{}, err
		}

		return nil, addDivisionDeploymentOut{
			DeploymentID:     int64Field(resp, "id"),
			DivisionTemplate: stringField(resp, "division_template"),
			Location:         nullableInt64Field(resp, "location"),
			Count:            int64Field(resp, "count"),
			OobFile:          stringField(resp, "oob_file"),
			CountryTag:       stringField(resp, "country_tag"),
			Warnings:         emptyIfNilStrings(stringSliceField(resp, "warnings")),
		}, nil
	})
}
