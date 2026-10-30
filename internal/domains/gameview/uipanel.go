package gameview

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"haganeworks/internal/project"
)

type uiPanelWindowEnvelope struct {
	PanelID    int64                                   `json:"panel_id"`
	Token      string                                  `json:"token"`
	Revision   string                                  `json:"revision"`
	WindowName string                                  `json:"window_name"`
	Controls   map[string]uiPanelWindowControlEnvelope `json:"controls"`
}

type uiPanelWindowControlEnvelope struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type uiPanel struct {
	ID        int64
	Token     string
	Controls  []map[string]any
	UpdatedAt time.Time
}

var panelNamePattern = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func panelNameSlug(raw, fallback string) string {
	s := strings.Trim(panelNamePattern.ReplaceAllString(strings.TrimSpace(raw), "_"), "_")
	if s == "" {
		return fallback
	}
	return s
}

func panelNames(token, controlName string, withControl bool) map[string]string {
	t := panelNameSlug(token, "panel")
	names := map[string]string{
		"window_name": t + "_panel",
		"gui_path":    "interface/zz_uipanel_" + t + ".gui",
	}
	if withControl {
		names["control_name"] = t + "_" + panelNameSlug(controlName, "control")
	}
	return names
}

func panelText(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprint(v)
}

func uiPanelWindowEnvelopeFor(panels []uiPanel, file, window string) (uiPanelWindowEnvelope, bool) {
	file = strings.ToLower(strings.Trim(strings.ReplaceAll(file, `\`, "/"), "/"))
	window = strings.TrimSpace(window)
	var matched *uiPanel
	var rows []map[string]any
	var rowPath []string
	for i := range panels {
		names := panelNames(panels[i].Token, "", false)
		if strings.ToLower(names["gui_path"]) != file {
			continue
		}
		var found []map[string]any
		var foundPath []string
		if names["window_name"] == window {
			found = panels[i].Controls
		} else if entry, path, ok := uiPanelListEntry(panels[i].Token, panels[i].Controls, window, nil); ok {
			found, foundPath = entry, path
		} else {
			continue
		}
		if matched != nil {
			return uiPanelWindowEnvelope{}, false
		}
		matched, rows, rowPath = &panels[i], found, foundPath
	}
	if matched == nil || matched.ID <= 0 || matched.UpdatedAt.IsZero() {
		return uiPanelWindowEnvelope{}, false
	}
	out := uiPanelWindowEnvelope{
		PanelID: matched.ID, Token: matched.Token,
		Revision: matched.UpdatedAt.UTC().Format(time.RFC3339Nano), WindowName: window,
		Controls: map[string]uiPanelWindowControlEnvelope{},
	}
	ambiguous := map[string]bool{}
	var walk func([]map[string]any, []string, []string)
	walk = func(rows []map[string]any, guiParents, controlParents []string) {
		for _, control := range rows {
			name := strings.TrimSpace(panelText(control["name"]))
			if name == "" || name == "<nil>" {
				continue
			}
			controlName := panelNames(matched.Token, name, true)["control_name"]
			guiParts := append(append([]string(nil), guiParents...), controlName)
			controlParts := append(append([]string(nil), controlParents...), name)
			namePath := strings.Join(append([]string{out.WindowName}, guiParts...), "/")
			if ambiguous[namePath] {
			} else if _, duplicate := out.Controls[namePath]; duplicate {

				delete(out.Controls, namePath)
				ambiguous[namePath] = true
			} else {
				out.Controls[namePath] = uiPanelWindowControlEnvelope{Name: name, Path: strings.Join(controlParts, "/")}
			}
			if strings.EqualFold(strings.TrimSpace(panelText(control["kind"])), "container") {
				if uiPanelIsList(control) {
					continue
				}
				childGui := guiParts
				if scroll, _ := control["scroll"].(bool); scroll {
					childGui = append(append([]string(nil), guiParts...), controlName+"_content")
				}
				walk(uiPanelControlRows(control["children"]), childGui, controlParts)
			}
		}
	}
	walk(rows, nil, rowPath)
	return out, true
}

func uiPanelIsList(control map[string]any) bool {
	array, _ := control["array"].(string)
	return strings.TrimSpace(array) != "" && strings.EqualFold(strings.TrimSpace(panelText(control["kind"])), "container")
}

func uiPanelListEntry(token string, rows []map[string]any, window string, parents []string) ([]map[string]any, []string, bool) {
	for _, control := range rows {
		name := strings.TrimSpace(panelText(control["name"]))
		if name == "" || name == "<nil>" || !strings.EqualFold(strings.TrimSpace(panelText(control["kind"])), "container") {
			continue
		}
		path := append(append([]string(nil), parents...), name)
		children := uiPanelControlRows(control["children"])
		if uiPanelIsList(control) && panelNames(token, name, true)["control_name"]+"_entry" == window {
			return children, path, true
		}
		if found, foundPath, ok := uiPanelListEntry(token, children, window, path); ok {
			return found, foundPath, true
		}
	}
	return nil, nil, false
}

func uiPanelControlRows(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		out := make([]map[string]any, 0, len(rows))
		for _, value := range rows {
			if row, ok := value.(map[string]any); ok {
				out = append(out, row)
			}
		}
		return out
	default:
		return nil
	}
}

func attachUIPanelEnvelope(p *project.Project, file, window string, result map[string]any) map[string]any {
	rows, err := p.Rows("ui_panels")
	if err != nil {
		return result
	}
	panels := make([]uiPanel, 0, len(rows))
	for _, row := range rows {
		panel := uiPanel{ID: project.Int64(row["id"]), Token: str(row["token"]), Controls: uiPanelControlRows(row["controls"])}
		if raw := strings.TrimSpace(str(row["updated_at"])); raw != "" {
			if t, err := time.Parse(timestampLayout, strings.TrimSuffix(raw, "+00:00")); err == nil {
				panel.UpdatedAt = t
			}
		}
		panels = append(panels, panel)
	}
	envelope, ok := uiPanelWindowEnvelopeFor(panels, file, window)
	if !ok {
		return result
	}
	out := make(map[string]any, len(result)+1)
	for key, value := range result {
		out[key] = value
	}
	out["ui_panel"] = envelope
	return out
}
