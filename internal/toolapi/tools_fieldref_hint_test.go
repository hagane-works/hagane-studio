package toolapi

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func fieldRefTestDeps(t *testing.T) func(host string) Deps {
	t.Helper()
	return func(host string) Deps {
		return Deps{
			Proxy:      NewRESTProxy("http://unused.invalid"),
			ExportJobs: NewRESTProxy("http://unused.invalid"),
		}
	}
}

func listAllTools(t *testing.T) []*Tool {
	t.Helper()
	srv := newAuthedTestServer(t, fieldRefTestDeps(t))
	session := connectClient(t, srv.URL)
	result, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	if len(result.Tools) < 80 {
		t.Fatalf("ListTools returned only %d tools — the test Deps is not building the "+
			"full surface, so coverage assertions here would pass by not looking", len(result.Tools))
	}
	return result.Tools
}

func TestToolFieldRefTargetsExist(t *testing.T) {
	docs := fieldDocs()
	for tool, workbench := range toolFieldRef {
		if _, ok := docs[workbench]; !ok {
			t.Errorf("tool %q points at %s, but fieldsdata/%s.md does not exist",
				tool, fieldRefURI(workbench), workbench)
		}
	}
}

func TestToolFieldRefNamesAreRealTools(t *testing.T) {
	registered := map[string]bool{}
	for _, tool := range listAllTools(t) {
		registered[tool.Name] = true
	}
	for tool := range toolFieldRef {
		if !registered[tool] {
			t.Errorf("toolFieldRef has a row for %q, which is not a registered tool — "+
				"renamed or removed? A dead row reads as coverage it does not provide", tool)
		}
	}
	for tool := range toolFieldRefExempt {
		if tool == "test" {
			continue
		}
		if !registered[tool] {
			t.Errorf("toolFieldRefExempt has a row for %q, which is not a registered tool", tool)
		}
	}
}

func TestToolFieldRefCoversEveryWriteTool(t *testing.T) {
	var uncovered []string
	for _, tool := range listAllTools(t) {
		name := tool.Name
		if isReadOnlyToolName(name) {
			continue
		}
		if _, mapped := toolFieldRef[name]; mapped {
			continue
		}
		if _, exempt := toolFieldRefExempt[name]; exempt {
			continue
		}
		uncovered = append(uncovered, name)
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 {
		t.Errorf("%d write tools have no field-reference pointer and no exemption: %s\n"+
			"Add each to toolFieldRef (with its fieldsdata stem) or to toolFieldRefExempt "+
			"(with the reason no doc covers it).", len(uncovered), strings.Join(uncovered, ", "))
	}
}

func TestFieldReferenceHintInToolsList(t *testing.T) {
	byName := map[string]*Tool{}
	for _, tool := range listAllTools(t) {
		byName[tool.Name] = tool
	}

	for _, tc := range []struct{ tool, workbench string }{
		{"create_focus", "focus"},
		{"create_event", "events"},
		{"create_technology", "technologies"},
		{"create_wargoal", "wargoals"},
		{"create_ai_strategy_plan", "ai-strategy-plans"},
		{"update_ai_strategy_plan", "ai-strategy-plans"},
		{"create_ai_template", "ai-templates"},
		{"update_ai_template", "ai-templates"},
		{"export_project", "_timeline-and-replace-path"},
	} {
		tool, ok := byName[tc.tool]
		if !ok {
			t.Errorf("tool %q not registered", tc.tool)
			continue
		}
		if !strings.Contains(tool.Description, fieldRefURI(tc.workbench)) {
			t.Errorf("tool %q description does not name %s:\n  %s",
				tc.tool, fieldRefURI(tc.workbench), tool.Description)
		}
	}

	if focus := byName["create_focus"]; focus != nil && !strings.Contains(focus.Description, "explicit duplicate focus_id") {
		t.Errorf("create_focus lost its original description text — the hint must append, not replace:\n  %s",
			focus.Description)
	}

	if list := byName["list_projects"]; list != nil && strings.Contains(list.Description, "hoi4://fields/") {
		t.Errorf("read-only tool list_projects got a field-reference pointer:\n  %s", list.Description)
	}

	for _, name := range []string{
		"create_ideology_root", "create_ideology_subtype", "update_ideology",
		"preview_delete_ideology", "apply_delete_ideology", "discard_ideology_conflict",
	} {
		tool := byName[name]
		if tool == nil {
			t.Errorf("politics tool %q not registered", name)
			continue
		}
		if strings.Contains(tool.Description, fieldRefURI("ideologies")) {
			t.Errorf("politics tool %q points at the unsafe legacy ideology field doc:\n  %s", name, tool.Description)
		}
	}
}

func TestFieldReferenceHintIsIdempotent(t *testing.T) {
	srv := newAuthedTestServer(t, fieldRefTestDeps(t))
	session := connectClient(t, srv.URL)

	var descriptions []string
	for i := 0; i < 3; i++ {
		result, err := session.ListTools(context.Background(), &ListToolsParams{})
		if err != nil {
			t.Fatalf("ListTools #%d: %v", i+1, err)
		}
		for _, tool := range result.Tools {
			if tool.Name == "create_focus" {
				descriptions = append(descriptions, tool.Description)
			}
		}
	}
	if len(descriptions) != 3 {
		t.Fatalf("expected create_focus in all 3 listings, got %d", len(descriptions))
	}
	for i, desc := range descriptions {
		if n := strings.Count(desc, fieldRefURI("focus")); n != 1 {
			t.Errorf("listing #%d: create_focus names hoi4://fields/focus %d times, want exactly 1:\n  %s",
				i+1, n, desc)
		}
		if desc != descriptions[0] {
			t.Errorf("listing #%d description differs from the first — tools/list is not stable", i+1)
		}
	}
}

func TestFieldRefWorkbenchCoverage(t *testing.T) {
	pointed := map[string]bool{}
	for _, wb := range fieldRefWorkbenches() {
		pointed[wb] = true
	}
	var unpointed []string
	for stem := range fieldDocs() {
		if stem == "README" || pointed[stem] {
			continue
		}
		unpointed = append(unpointed, stem)
	}
	sort.Strings(unpointed)
	t.Logf("%d/%d field-reference docs are pointed at by a tool; not pointed at: %s",
		len(pointed), len(fieldDocs())-1, strings.Join(unpointed, ", "))
}
