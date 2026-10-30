package toolapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var worldImageWorkingTools = map[string]bool{
	"create_world_image": true, "create_terrain_sketch": true, "create_country_split": true,
}

var (
	worldImageStepTimeout  = 90 * time.Second
	worldImageWorldTimeout = 5 * time.Minute
)

type worldImagePoint struct {
	X int `json:"x" jsonschema:"pixels from the picture's left edge"`
	Y int `json:"y" jsonschema:"pixels from the picture's top edge"`
}

type worldImageRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type importWorldImageIn struct {
	ProjectID  int64           `json:"project_id"`
	DataBase64 string          `json:"data_base64,omitempty" jsonschema:"the picture (PNG, JPEG, WebP, GIF or BMP) as base64, or as a data: URL"`
	FilePath   string          `json:"file_path,omitempty" jsonschema:"or: a picture file on this computer (PNG, JPEG, WebP, GIF or BMP)"`
	ImageID    string          `json:"image_id,omitempty" jsonschema:"a world image already imported, to cut a part of it (with crop)"`
	Crop       *worldImageRect `json:"crop,omitempty" jsonschema:"with image_id: the part to keep, in that image's pixels"`
}

type inspectWorldImageIn struct {
	ProjectID  int64                 `json:"project_id"`
	ImageID    string                `json:"image_id"`
	MaxColours int                   `json:"max_colours,omitempty" jsonschema:"how many main colours to list (default 16, at most 40)"`
	Marks      []worldImageMarkInput `json:"marks,omitempty" jsonschema:"points to draw on the preview, to check where planned seeds or rules land"`
	Zoom       *worldImageRect       `json:"zoom,omitempty" jsonschema:"look closer at this part of the picture; the grid stays in whole-picture pixels"`
}

type worldImageMarkInput struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Label string `json:"label,omitempty"`
}

type terrainRuleInput struct {
	Colour    string           `json:"colour,omitempty" jsonschema:"#rrggbb, e.g. from get_world_image colours"`
	At        *worldImagePoint `json:"at,omitempty" jsonschema:"or: take the colour around this pixel"`
	Terrain   string           `json:"terrain" jsonschema:"ocean, plains, forest, hills, mountain, desert, marsh, jungle, lakes, or ignore (fill from around it)"`
	Tolerance float64          `json:"tolerance,omitempty" jsonschema:"how far a colour may be from this one and still match (default 12 for political sources, 40 for terrain/legacy); explicit values are retained, inspect nearby country colours and returned effective tolerance"`
}

type terrainAreaInput struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Terrain string `json:"terrain" jsonschema:"what everything in this box is, whatever its colours (a compass rose or sea monster on the sea → ocean)"`
}

type makeTerrainSketchIn struct {
	ProjectID  int64              `json:"project_id"`
	ImageID    string             `json:"image_id" jsonschema:"the picture to read terrain from (often a redrawn, flat-colour terrain map)"`
	SourceKind string             `json:"source_kind,omitempty" jsonschema:"political for a flat country-colour map: requires explicit ocean rule(s), permits only ocean/ignore rules and areas, and all unmatched land becomes plains; terrain for actual terrain colours; omitted keeps the legacy interpretation"`
	Rules      []terrainRuleInput `json:"rules,omitempty" jsonschema:"which colours are which terrain; the nearest matching rule wins"`
	Areas      []terrainAreaInput `json:"areas,omitempty" jsonschema:"boxes forced to a terrain, over the rules"`
	Unmatched  string             `json:"unmatched,omitempty" jsonschema:"what colours no rule matches become: nearest (the closest legend colour, default), a terrain, or ignore"`
	Dark       float64            `json:"dark,omitempty" jsonschema:"brightness 0-255 below which an unmatched pixel is a line or lettering, filled from around it (default 45; -1 off)"`
	Smooth     int                `json:"smooth,omitempty" jsonschema:"radius of the filter that removes thin rivers, roads and borders (default 3; -1 off)"`
	MinPatch   float64            `json:"min_patch,omitempty" jsonschema:"smallest patch kept, as a share of the drawing (default 0.0002)"`
	Crop       *worldImageRect    `json:"crop,omitempty" jsonschema:"the part of the picture that is the map, leaving out frames, titles and legends"`
	Margin     float64            `json:"margin,omitempty" jsonschema:"sea left around the drawing, as a share of the map (default 0.03)"`
}

type countrySeedInput struct {
	X               *int             `json:"x,omitempty" jsonschema:"legacy point x; give both x and y, or prefer colour for a political map"`
	Y               *int             `json:"y,omitempty" jsonschema:"legacy point y"`
	Tag             string           `json:"tag" jsonschema:"the country's tag (3 letters/digits, a letter first); several seeds may share a tag"`
	Colour          string           `json:"colour,omitempty" jsonschema:"exact source country colour #rrggbb: prefer get_world_image colours[].sample_source_colour with that same verified sample as at; hex is display-only clustered colour and may not exist in the source. Backend selects a verified deep interior pixel"`
	ColourTolerance float64          `json:"colour_tolerance,omitempty" jsonschema:"source colour matching tolerance 0..100 (0 means default 20), separate from border-growth tolerance; inspect source_colour and match_share in seed_matches"`
	Box             *worldImageRect  `json:"box,omitempty" jsonschema:"source-pixel search box selecting one same-colour country or island; required with colour when disconnected regions are ambiguous"`
	At              *worldImagePoint `json:"at,omitempty" jsonschema:"with colour, a verified source location selecting one disconnected region: use the same swatch.sample after matching its source region to the intended country; backend chooses its deep interior. Without colour, an explicit legacy point"`
}

type splitCountriesIn struct {
	ProjectID      int64              `json:"project_id"`
	ImageID        string             `json:"image_id" jsonschema:"the picture whose colours show the countries (the original political map, same size as the terrain sketch's source)"`
	TerrainImageID string             `json:"terrain_image_id" jsonschema:"the create_terrain_sketch result: which pixels are land, and where the map sits"`
	Seeds          []countrySeedInput `json:"seeds" jsonschema:"a point inside each country (more for islands or differently coloured parts)"`
	Tolerance      float64            `json:"tolerance,omitempty" jsonschema:"how far a colour may be from a seed's and still be that country's (default 28; more for shaded paintings)"`
	Dark           float64            `json:"dark,omitempty" jsonschema:"brightness below which a pixel is a line or lettering (default 45; -1 off)"`
	SeaCost        float64            `json:"sea_cost,omitempty" jsonschema:"cost of crossing water to an island (default 6)"`
}

type worldFromImagesIn struct {
	ProjectID      int64    `json:"project_id"`
	TerrainImageID string   `json:"terrain_image_id"`
	OwnersImageID  string   `json:"owners_image_id,omitempty" jsonschema:"the create_country_split result; without it, give tags and the states go to them west to east"`
	Tags           []string `json:"tags,omitempty"`
	Land           int      `json:"land,omitempty" jsonschema:"pixels per land province on the 5632×2048 map, 400-60000 (default 2000; the base game is about 400)"`
	Sea            int      `json:"sea,omitempty" jsonschema:"pixels per sea province (default 5× land)"`
	Seed           int64    `json:"seed,omitempty" jsonschema:"random seed; the same pictures and seed give the same map"`
	PerState       int      `json:"per_state,omitempty" jsonschema:"provinces per state, 1-30 (default: the generator's)"`
	Languages      []string `json:"languages,omitempty" jsonschema:"languages to write state names in, e.g. en, zh (default en)"`
}

func worldImageResult(resp map[string]any) (*CallToolResult, map[string]any) {
	images := map[string]string{}
	if preview, ok := resp["preview"].(string); ok {
		images["preview"] = preview
		delete(resp, "preview")
	}
	if previews, ok := resp["previews"].(map[string]any); ok {
		for name, value := range previews {
			if url, ok := value.(string); ok {
				images[name] = url
			}
		}
		delete(resp, "previews")
	}
	names := make([]string, 0, len(images))
	for name := range images {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		resp["previews_attached"] = names
	}

	delete(resp, "preview_url")
	delete(resp, "preview_urls")
	text, _ := json.Marshal(resp)
	call := &CallToolResult{Content: []Content{&TextContent{Text: string(text)}}}
	for _, name := range names {
		if content := dataURLImage(images[name]); content != nil {
			call.Content = append(call.Content, content)
		}
	}
	return call, resp
}

func dataURLImage(url string) *ImageContent {
	header, payload, ok := strings.Cut(url, ",")
	if !ok || !strings.HasPrefix(header, "data:image/") || !strings.HasSuffix(header, ";base64") {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil
	}
	return &ImageContent{Data: data, MIMEType: strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")}
}

func worldImagePath(projectID int64, name string) string {
	return fmt.Sprintf("/api/projects/%d/gui-render/world-map/%s", projectID, name)
}

func worldImageTool[In any](s *Server, proxy *RESTProxy, name, description, route string, timeout time.Duration,
	project func(In) int64, body func(In) (any, error)) {
	worldImageToolThen(s, proxy, name, description, route, timeout, project, body, nil)
}

func worldImageToolThen[In any](s *Server, proxy *RESTProxy, name, description, route string, timeout time.Duration,
	project func(In) int64, body func(In) (any, error), then func(projectID int64, resp map[string]any)) {
	AddTool(s, &Tool{Name: name, Description: description},
		func(ctx context.Context, _ *CallToolRequest, in In) (*CallToolResult, map[string]any, error) {
			projectID := project(in)
			if projectID <= 0 {
				return nil, nil, fmt.Errorf("project_id must be greater than 0")
			}
			payload, err := body(in)
			if err != nil {
				return nil, nil, err
			}
			resp, err := proxyAuthedCall(ctx, proxy.withTimeout(timeout), "POST", worldImagePath(projectID, route), payload)
			if err != nil {
				return nil, nil, err
			}
			if then != nil {
				then(projectID, resp)
			}
			if route == "image-world-preview" || route == "image-world" {
				resp = compactWorldResult(resp)
			}
			call, out := worldImageResult(resp)
			return call, out, nil
		})
}

func registerWorldImageTools(s *Server, proxy *RESTProxy) {
	worldImageTool(s, proxy, "create_world_image",
		"Bring a map picture into the project's world-image workspace (not the mod): a picture as base64 (data_base64), "+
			"a picture file on this computer (file_path), or part of an imported picture (image_id + crop). "+
			"Returns its image id and a preview with a pixel grid. First step of building a world from a picture.",
		"image-import", worldImageStepTimeout,
		func(in importWorldImageIn) int64 { return in.ProjectID },
		func(in importWorldImageIn) (any, error) {
			body := map[string]any{}
			switch {
			case strings.TrimSpace(in.DataBase64) != "":
				body["data_base64"] = strings.TrimSpace(in.DataBase64)
			case strings.TrimSpace(in.FilePath) != "":
				data, err := readWorldImageFile(strings.TrimSpace(in.FilePath))
				if err != nil {
					return nil, err
				}
				body["data_base64"] = base64.StdEncoding.EncodeToString(data)
			case strings.TrimSpace(in.ImageID) != "":
				body["image_id"] = strings.TrimSpace(in.ImageID)
				if in.Crop != nil {
					body["crop"] = in.Crop
				}
			default:
				return nil, fmt.Errorf("give data_base64, file_path, or image_id with crop")
			}
			return body, nil
		})

	worldImageTool(s, proxy, "get_world_image",
		"Look at a world image: its size, merged palette colours, terrain legend, and a preview with a source-pixel grid. "+
			"colours[].hex is a display-only cluster representative and may not occur in the source. sample_source_colour is the exact original colour at sample; use that colour with the same verified sample as at for terrain rules or country seeds. "+
			"nearest_terrain is only colour distance, not semantic terrain evidence; a blue political ocean must be explicitly ocean. "+
			"Palette may merge nearby country colours and its sample may belong to another country. Prefer create_country_split "+
			"colour plus box/at for verified source-pixel seeds. Use marks/zoom only for a genuinely unclear part; reuse unchanged source inspections.",
		"image-inspect", worldImageStepTimeout,
		func(in inspectWorldImageIn) int64 { return in.ProjectID },
		func(in inspectWorldImageIn) (any, error) {
			body := map[string]any{"image_id": in.ImageID}
			if in.MaxColours > 0 {
				body["max_colours"] = in.MaxColours
			}
			if len(in.Marks) > 0 {
				body["marks"] = in.Marks
			}
			if in.Zoom != nil {
				body["zoom"] = in.Zoom
			}
			return body, nil
		})

	worldImageTool(s, proxy, "create_terrain_sketch",
		"Turn a world image into a terrain sketch for the world generator: rules say which colours are which terrain, "+
			"For a political map set source_kind=political, give explicit ocean colour rules, and keep other land plains; political country colours do not imply terrain. "+
			"areas force boxes, crop leaves out frames. The drawing is placed, centred with a sea margin, on the game map's "+
			"shape. Returns the sketch's image id, each terrain's share and a preview over the picture's own grid — compare "+
			"it with the picture and fix with more rules or areas.",
		"image-terrain", worldImageStepTimeout,
		func(in makeTerrainSketchIn) int64 { return in.ProjectID },
		func(in makeTerrainSketchIn) (any, error) {
			body := map[string]any{"image_id": in.ImageID}
			if in.SourceKind != "" {
				body["source_kind"] = in.SourceKind
			}
			if len(in.Rules) > 0 {
				body["rules"] = in.Rules
			}
			if len(in.Areas) > 0 {
				body["areas"] = in.Areas
			}
			if in.Unmatched != "" {
				body["unmatched"] = in.Unmatched
			}
			if in.Dark != 0 {
				body["dark"] = in.Dark
			}
			if in.Smooth != 0 {
				body["smooth"] = in.Smooth
			}
			if in.MinPatch != 0 {
				body["min_patch"] = in.MinPatch
			}
			if in.Crop != nil {
				body["crop"] = in.Crop
			}
			if in.Margin != 0 {
				body["margin"] = in.Margin
			}
			return body, nil
		})

	worldImageTool(s, proxy, "create_country_split",
		"Split the land of a terrain sketch between countries, as a picture shows them: use each verified country's sample_source_colour as colour and the same sample as at, with its tag; never use display-only palette hex as an exact source colour. "+
			"First match the source sample's position and label to the intended country; a merged swatch does not identify every similar-coloured country. The backend picks a verified source-pixel deep interior seed. Use verified box or at for same-colour regions/islands; never guess a box or choose the largest region silently. "+
			"legacy x/y points remain supported. Returns seed_matches with source_colour, actual_seed, match_share and warnings. Borders follow the picture's colours, and fall halfway between seeds where two countries share "+
			"a colour. Returns the owners image id, each country's share, parts and box, warnings and a preview.",
		"image-countries", worldImageStepTimeout,
		func(in splitCountriesIn) int64 { return in.ProjectID },
		func(in splitCountriesIn) (any, error) {
			body := map[string]any{"image_id": in.ImageID, "terrain_image_id": in.TerrainImageID, "seeds": in.Seeds}
			if in.Tolerance != 0 {
				body["tolerance"] = in.Tolerance
			}
			if in.Dark != 0 {
				body["dark"] = in.Dark
			}
			if in.SeaCost != 0 {
				body["sea_cost"] = in.SeaCost
			}
			return body, nil
		})

	worldBody := func(in worldFromImagesIn) (any, error) {
		body := map[string]any{"terrain_image_id": in.TerrainImageID}
		if in.OwnersImageID != "" {
			body["owners_image_id"] = in.OwnersImageID
		}
		if len(in.Tags) > 0 {
			body["tags"] = in.Tags
		}
		if in.Land != 0 {
			body["land"] = in.Land
		}
		if in.Sea != 0 {
			body["sea"] = in.Sea
		}
		if in.Seed != 0 {
			body["seed"] = in.Seed
		}
		if in.PerState != 0 {
			body["per_state"] = in.PerState
		}
		if len(in.Languages) > 0 {
			body["languages"] = in.Languages
		}
		return body, nil
	}
	worldProject := func(in worldFromImagesIn) int64 { return in.ProjectID }
	worldImageToolThen(s, proxy, "preview_world_from_images",
		"Preview the world a terrain sketch and its countries make — provinces, states, regions, each country's state "+
			"count, the map rule check's findings and political/terrain pictures — without saving. The countries' tags must already exist in the project "+
			"(create_country). When the project's map already has changes (new land, province edits, an earlier new world) "+
			"the answer carries map_changes: the records a new world cannot be saved over. Never take them away on your own: "+
			"the player does, in the game view's map studio (its map records); when the player asks you to restore the map, "+
			"call restore_world_map. Takes up to a few minutes.",
		"image-world-preview", worldImageWorldTimeout, worldProject, worldBody, nil)
	worldImageTool(s, proxy, "create_world_from_images",
		"Replace the project's world map with the world preview_world_from_images showed: map files, states (numbered "+
			"from 1, replacing the project's), strategic regions and the countries' capitals. Returns state_layout with each "+
			"new state_id, owner, capital and source-image position; next call name_world_map once with the whole batch. "+
			"Countries that need a generated "+
			"start receive one canonical structured country leader instead of a parallel raw-history leader, so character "+
			"tools, exported MOD data and the game view share the same entity; re-read it with get_characters "+
			"before customising it. Refused while the map has other changes (the preview's map_changes), which only the "+
			"player can restore. Undo with the world map's restore. Call only after the user confirms the preview.",
		"image-world", worldImageWorldTimeout, worldProject, worldBody)

	registerNameWorldMapTool(s, proxy)
	registerWorldStateLayoutTool(s, proxy)
}

const worldImageFileMaxBytes = 16 << 20

func readWorldImageFile(path string) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
	default:
		return nil, fmt.Errorf("file_path must be a PNG, JPEG, WebP, GIF or BMP picture")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file_path: %w", err)
	}
	if info.IsDir() || info.Size() > worldImageFileMaxBytes {
		return nil, fmt.Errorf("file_path must be a picture file of at most %d MB", worldImageFileMaxBytes>>20)
	}
	return os.ReadFile(path)
}
