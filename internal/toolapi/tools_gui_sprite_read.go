package toolapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type getGUISpritesIn struct {
	ProjectID      int64            `json:"project_id"`
	Names          []string         `json:"names" jsonschema:"1 to 100 exact GFX sprite names"`
	IncludeUsage   bool             `json:"include_usage,omitempty" jsonschema:"default false; optional source usage evidence when required to resolve missing subject information; known selected targets do not need this read"`
	SelectedFrames map[string][]int `json:"selected_frames,omitempty" jsonschema:"exact GFX name to nonempty zero-based selected frame indices; with include_usage only these frame meanings are returned, never the whole atlas"`
}

type guiSpriteSummary struct {
	UsageEvidence map[string]any `json:"usage_evidence,omitempty"`
	Name          string         `json:"name"`
	Known         bool           `json:"known"`
	Resolved      bool           `json:"resolved"`
	Overridden    bool           `json:"overridden"`
	MissingFile   bool           `json:"missing_file"`
	Procedural    bool           `json:"procedural"`
	Width         int64          `json:"width"`
	Height        int64          `json:"height"`
	Frames        int64          `json:"frames"`
	TextureFile   string         `json:"texturefile,omitempty"`
}

type getGUISpritesOut struct {
	ProjectID int64              `json:"project_id"`
	Sprites   []guiSpriteSummary `json:"sprites"`
}

func registerGetGUISprites(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_gui_sprites",
		Description: "Inspect exact GFX sprite resources of the project as the game view renders them. " +
			"Use exact sprite names (from get_gui_window or get_gui_overlay); do not guess names. " +
			"Returns compact dimensions, frame count and texture by default. Ordinary portrait/frame/background replacement does not require functional usage or repeated window/entity reads. When subject information is missing, include_usage=true and selected_frames can return evidence for only those exact zero-based indices; this read is not a prerequisite for replacing a known target. Indexed evidence does not establish live runtime triggers; unavailable conditions remain unknown.",
	}, func(ctx context.Context, _ *CallToolRequest, in getGUISpritesIn) (*CallToolResult, getGUISpritesOut, error) {
		if in.ProjectID <= 0 {
			return nil, getGUISpritesOut{}, fmt.Errorf("project_id must be greater than 0")
		}
		if len(in.Names) == 0 || len(in.Names) > 100 {
			return nil, getGUISpritesOut{}, fmt.Errorf("names must contain between 1 and 100 exact GFX sprite names")
		}
		names := make([]string, 0, len(in.Names))
		seen := map[string]bool{}
		for _, raw := range in.Names {
			name := strings.TrimSpace(raw)
			if name == "" || len(name) > 512 || strings.ContainsAny(name, ",\r\n\x00") {
				return nil, getGUISpritesOut{}, fmt.Errorf("invalid GFX sprite name")
			}
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		for name, frames := range in.SelectedFrames {
			if !seen[name] || len(frames) == 0 || len(frames) > 256 {
				return nil, getGUISpritesOut{}, fmt.Errorf("selected_frames must contain 1..256 zero-based indices for exact requested GFX names")
			}
			seenFrames := map[int]bool{}
			for _, frame := range frames {
				if frame < 0 || seenFrames[frame] {
					return nil, getGUISpritesOut{}, fmt.Errorf("selected_frames indices must be distinct and nonnegative")
				}
				seenFrames[frame] = true
			}
		}
		query := url.Values{"names": {strings.Join(names, ",")}}
		if in.IncludeUsage {
			query.Set("include_usage", "true")
		}
		path := fmt.Sprintf("/api/projects/%d/gui-render/sprites?%s", in.ProjectID, query.Encode())
		resp, err := proxyAuthedCall(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, getGUISpritesOut{}, err
		}
		payload, _ := resp["sprites"].(map[string]any)
		usages, _ := resp["usage_evidence"].(map[string]any)
		out := getGUISpritesOut{ProjectID: in.ProjectID, Sprites: make([]guiSpriteSummary, 0, len(names))}
		for _, name := range names {
			meta, _ := payload[name].(map[string]any)
			row := guiSpriteSummary{
				Name: name, Resolved: boolField(meta, "resolved"), Overridden: boolField(meta, "overridden"),
				MissingFile: boolField(meta, "missing_file"), Procedural: boolField(meta, "procedural"),
				Width: int64Field(meta, "width"), Height: int64Field(meta, "height"), Frames: int64Field(meta, "frames"),
				TextureFile: stringField(meta, "texturefile"),
			}
			row.Known = row.Resolved || row.Overridden || row.Procedural || row.TextureFile != "" || row.Width > 0 || row.Height > 0
			for _, frame := range in.SelectedFrames[name] {
				if row.Frames > 0 && int64(frame) >= row.Frames {
					return nil, getGUISpritesOut{}, fmt.Errorf("selected_frames for %s must be within 0..%d", name, row.Frames-1)
				}
			}
			if in.IncludeUsage {
				usage, _ := usages[name].(map[string]any)
				row.UsageEvidence = selectedGUISpriteUsage(usage, in.SelectedFrames[name])
			}
			out.Sprites = append(out.Sprites, row)
		}
		return nil, out, nil
	})
}

func selectedGUISpriteUsage(usage map[string]any, frames []int) map[string]any {
	if usage == nil {
		return nil
	}
	out := make(map[string]any, len(usage))
	for key, value := range usage {
		if key != "frame_usages" {
			out[key] = value
		}
	}
	wanted := map[int64]bool{}
	for _, frame := range frames {
		wanted[int64(frame)] = true
	}
	filtered := []any{}
	rawFrames, _ := usage["frame_usages"].([]any)
	for _, value := range rawFrames {
		row, ok := value.(map[string]any)
		if _, present := row["frame"]; ok && present && wanted[int64Field(row, "frame")] {
			filtered = append(filtered, row)
		}
	}
	out["frame_usages"] = filtered
	return out
}
