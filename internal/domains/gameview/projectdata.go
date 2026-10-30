package gameview

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"haganeworks/internal/export"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/project"
)

type importedBase uint8

const (
	baseVanilla importedBase = iota
	baseImported
	baseImportedUI
)

type replaceScope uint8

const (
	replaceAllLower replaceScope = iota
	replaceVanillaOnly
)

type previewSpec struct {
	Identity        string
	IncludeImported bool
	Base            importedBase
	ReplaceScope    replaceScope
}

type projectBuild struct {
	Files        map[string][]byte
	Flags        map[string]string
	ShadowPaths  []string
	Winners      map[string]exportengine.NativePreviewWinner
	ReplacePaths []string
	Spec         previewSpec
	ProjectionOK bool

	RawTextures []rawTexture

	OwnedDLCs *[]string
}

func previewSpecFor(snapshot exportengine.Snapshot, legacyImport bool) previewSpec {
	if legacyImport {
		return previewSpec{Identity: "legacy_import", IncludeImported: true, Base: baseImported, ReplaceScope: replaceVanillaOnly}
	}
	var record struct {
		LoadedExisting bool              `json:"is_loaded_existing_mod"`
		Dependencies   []json.RawMessage `json:"dependencies"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	hasDependencies := len(record.Dependencies) > 0
	switch {
	case record.LoadedExisting && hasDependencies:
		return previewSpec{Identity: "ambiguous", IncludeImported: true, Base: baseImportedUI, ReplaceScope: replaceVanillaOnly}
	case record.LoadedExisting:
		return previewSpec{Identity: "loaded_existing", IncludeImported: true, Base: baseImportedUI, ReplaceScope: replaceVanillaOnly}
	case hasDependencies:
		return previewSpec{Identity: "submod", Base: baseImported}
	default:
		return previewSpec{Identity: "standalone"}
	}
}

func buildProjectGameFiles(p *project.Project, belowOf func() *guirender.Index, legacyImport bool) (build projectBuild) {
	started := time.Now()
	snapshot, err := export.Snapshot(p)
	if err != nil {
		log.Printf("[gameview] project %s data layer: snapshot: %v", p.Dir(), err)
		return build
	}
	build.Spec = previewSpecFor(snapshot, legacyImport)
	var dlcChoice struct {
		OwnedDLCs *[]string `json:"owned_dlcs"`
	}
	if json.Unmarshal(snapshot.Project, &dlcChoice) == nil {
		build.OwnedDLCs = dlcChoice.OwnedDLCs
	}

	var below *guirender.Index
	if build.Spec.Base == baseImported && !build.Spec.IncludeImported {
		below = belowOf()
	}
	filterImportedOriginals := below != nil
	if filterImportedOriginals {
		snapshot = keepProjectCountries(snapshot, below.ListCountryTags())
		snapshot = keepProjectDecisionCategories(snapshot, below)
	}
	build.Flags = projectFlagURLs(p.ID(), snapshot)
	projection, err := exportengine.BuildNativePreviewProjection(snapshot, nil, build.Spec.IncludeImported)
	if err != nil {
		log.Printf("[gameview] project %s data layer: export: %v", p.Dir(), err)
		return build
	}
	build.Files = projection.TextFiles
	build.ShadowPaths = append([]string(nil), projection.ShadowPaths...)
	build.Winners = projection.Winners
	build.ReplacePaths = projection.EffectiveReplacePaths
	build.ProjectionOK = true
	if filterImportedOriginals {
		if carriers := decisionCategoryCarriers(snapshot, below); len(carriers) > 0 {
			build.Files = withoutCarrierDefinitions(p, snapshot, build.Files, carriers)
		}
	}
	build.Files = withSuperEventWindows(snapshot, build.Files)
	build.ShadowPaths, build.Winners = reconcileProjection(projection.TextFiles, build.Files, build.ShadowPaths, build.Winners)
	build.RawTextures = rawTexturesOf(snapshot, build.Winners)
	log.Printf("[gameview] project %s %s data layer: %d files in %s (stages reused %d, ran %d)",
		p.Dir(), build.Spec.Identity, len(build.Files), time.Since(started).Round(time.Millisecond),
		projection.StageReuse.Reused, projection.StageReuse.Ran)
	return build
}

func foldPath(name string) string {
	return strings.ToLower(strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/"))
}

func reconcileProjection(original, final map[string][]byte, shadowPaths []string, winners map[string]exportengine.NativePreviewWinner) ([]string, map[string]exportengine.NativePreviewWinner) {
	originalText := make(map[string]bool, len(original))
	for name := range original {
		originalText[foldPath(name)] = true
	}
	finalNames := make(map[string]string, len(final))
	for name := range final {
		finalNames[foldPath(name)] = name
	}
	outWinners := make(map[string]exportengine.NativePreviewWinner, len(winners)+len(final))
	for name, winner := range winners {
		outWinners[name] = winner
	}
	seen := map[string]bool{}
	outPaths := make([]string, 0, len(shadowPaths)+len(final))
	for _, name := range shadowPaths {
		key := foldPath(name)
		if key == "" || seen[key] || originalText[key] && finalNames[key] == "" {
			if originalText[key] && finalNames[key] == "" {
				delete(outWinners, key)
			}
			continue
		}
		seen[key] = true
		outPaths = append(outPaths, name)
	}
	for key, name := range finalNames {
		if !seen[key] {
			seen[key] = true
			outPaths = append(outPaths, name)
		}
		if _, ok := outWinners[key]; !ok {
			outWinners[key] = exportengine.NativePreviewWinner{Origin: "preview", SourceEntity: "gameview_synthetic"}
		}
	}
	return outPaths, outWinners
}

func withSuperEventWindows(snapshot exportengine.Snapshot, files map[string][]byte) map[string][]byte {
	rows := snapshot.Tables["super_events"]
	if len(rows) == 0 {
		return files
	}
	var record struct {
		Tag string `json:"country_tag"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	fallback := strings.ToUpper(strings.TrimSpace(record.Tag))
	if fallback == "" {
		fallback = "UNK"
	}
	if files == nil {
		files = map[string][]byte{}
	}
	for _, raw := range rows {
		var row struct {
			Tag string `json:"country_tag"`
		}
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(row.Tag))
		if tag == "" {
			tag = fallback
		}
		if _, have := files["interface/"+tag+"_superevent.gui"]; have {
			continue
		}
		emitmisc.EmitSuperEventBoilerplate(func(path, text string) {
			if strings.HasPrefix(path, "interface/") {
				files[path] = []byte(text)
			}
		}, "", tag)
	}
	for _, name := range []string{"SE_bg.dds", "SE_frame.dds", "SE_pic.dds", "SE_btn.dds"} {
		if _, have := files["gfx/tvbigevent/"+name]; !have {
			files["gfx/tvbigevent/"+name] = exportengine.SuperEventPlaceholderDDS(name)
		}
	}
	return files
}

func projectFlagURLs(projectID int64, snapshot exportengine.Snapshot) map[string]string {
	out := map[string]string{}
	for _, raw := range snapshot.Tables["countries"] {
		var row struct {
			ID         json.Number     `json:"id"`
			Tag        string          `json:"tag"`
			Flag       string          `json:"flag_image_path"`
			ByIdeology json.RawMessage `json:"flag_paths_by_ideology"`
			UpdatedAt  string          `json:"updated_at"`
		}
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		id, err := row.ID.Int64()
		if err != nil || id <= 0 {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(row.Tag))
		if len(tag) != 3 {
			continue
		}
		base := fmt.Sprintf("/api/go/projects/%d/countries/%d/flag", projectID, id)
		version := url.QueryEscape(row.UpdatedAt)
		if strings.TrimSpace(row.Flag) != "" {
			out[tag] = base + "?v=" + version
		}
		var byIdeology map[string]string
		if len(row.ByIdeology) > 0 && json.Unmarshal(row.ByIdeology, &byIdeology) == nil {
			for ideology, file := range byIdeology {
				if strings.TrimSpace(file) != "" && strings.TrimSpace(ideology) != "" {
					out[tag+"_"+strings.ToLower(ideology)] = base + "?ideology=" + url.QueryEscape(ideology) + "&v=" + version
				}
			}
		}
	}
	return out
}

func keepProjectCountries(snapshot exportengine.Snapshot, belowTags []string) exportengine.Snapshot {
	rows := snapshot.Tables["countries"]
	if len(rows) == 0 || len(belowTags) == 0 {
		return snapshot
	}
	known := make(map[string]bool, len(belowTags))
	for _, tag := range belowTags {
		known[strings.ToUpper(tag)] = true
	}
	kept := make([]json.RawMessage, 0, len(rows))
	for _, raw := range rows {
		var row struct {
			Tag       string `json:"tag"`
			AutoAdded *bool  `json:"auto_added"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		if err := json.Unmarshal(raw, &row); err == nil {
			knownTag := known[strings.ToUpper(strings.TrimSpace(row.Tag))]
			untouchedImport := row.CreatedAt == row.UpdatedAt
			if row.AutoAdded != nil {
				untouchedImport = *row.AutoAdded && row.CreatedAt == row.UpdatedAt
			}
			if knownTag && untouchedImport {
				continue
			}
		}
		if known[strings.ToUpper(strings.TrimSpace(row.Tag))] {

			raw = withoutRowKey(raw, "ideology")
		}
		kept = append(kept, raw)
	}
	return withTable(snapshot, "countries", kept)
}

func withoutRowKey(raw json.RawMessage, key string) json.RawMessage {
	var row map[string]json.RawMessage
	if json.Unmarshal(raw, &row) != nil {
		return raw
	}
	if _, ok := row[key]; !ok {
		return raw
	}
	delete(row, key)
	out, err := json.Marshal(row)
	if err != nil {
		return raw
	}
	return out
}

func decisionsHolding(snapshot exportengine.Snapshot) map[int64]bool {
	holding := map[int64]bool{}
	for _, raw := range snapshot.Tables["decisions"] {
		var decision struct {
			CategoryID json.Number `json:"category_id"`
			Source     string      `json:"source"`
		}
		if json.Unmarshal(raw, &decision) == nil && decision.Source != "imported" {
			if id, err := decision.CategoryID.Int64(); err == nil {
				holding[id] = true
			}
		}
	}
	return holding
}

type categoryRow struct {
	ID         json.Number `json:"id"`
	CategoryID string      `json:"category_id"`
	CreatedAt  string      `json:"created_at"`
	UpdatedAt  string      `json:"updated_at"`
}

func knownCategories(below *guirender.Index) map[string]bool {
	known := map[string]bool{}
	for _, category := range below.DecisionCategories() {
		known[strings.ToLower(category.Name)] = true
	}
	return known
}

func keepProjectDecisionCategories(snapshot exportengine.Snapshot, below *guirender.Index) exportengine.Snapshot {
	rows := snapshot.Tables["decision_categories"]
	if len(rows) == 0 {
		return snapshot
	}
	holding := decisionsHolding(snapshot)
	known := knownCategories(below)
	kept := make([]json.RawMessage, 0, len(rows))
	for _, raw := range rows {
		var row categoryRow
		if err := json.Unmarshal(raw, &row); err == nil {
			id, _ := row.ID.Int64()
			if !holding[id] && known[strings.ToLower(strings.TrimSpace(row.CategoryID))] && row.CreatedAt == row.UpdatedAt {
				continue
			}
		}
		kept = append(kept, raw)
	}
	return withTable(snapshot, "decision_categories", kept)
}

func decisionCategoryCarriers(snapshot exportengine.Snapshot, below *guirender.Index) map[int64]bool {
	holding := decisionsHolding(snapshot)
	known := knownCategories(below)
	carriers := map[int64]bool{}
	for _, raw := range snapshot.Tables["decision_categories"] {
		var row categoryRow
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		id, _ := row.ID.Int64()
		if holding[id] && known[strings.ToLower(strings.TrimSpace(row.CategoryID))] && row.CreatedAt == row.UpdatedAt {
			carriers[id] = true
		}
	}
	return carriers
}

func isCategoryDefinitionFile(name string) bool {
	return strings.Contains(strings.ToLower(name), "common/decisions/categories/")
}

func withoutCarrierDefinitions(p *project.Project, snapshot exportengine.Snapshot, files map[string][]byte, carriers map[int64]bool) map[string][]byte {
	rows := make([]json.RawMessage, 0, len(snapshot.Tables["decision_categories"]))
	for _, raw := range snapshot.Tables["decision_categories"] {
		var row struct {
			ID json.Number `json:"id"`
		}
		if json.Unmarshal(raw, &row) == nil {
			if id, _ := row.ID.Int64(); carriers[id] {
				continue
			}
		}
		rows = append(rows, raw)
	}
	decisions := make([]json.RawMessage, 0, len(snapshot.Tables["decisions"]))
	for _, raw := range snapshot.Tables["decisions"] {
		var decision struct {
			CategoryID json.Number `json:"category_id"`
		}
		if json.Unmarshal(raw, &decision) == nil {
			if id, _ := decision.CategoryID.Int64(); carriers[id] {
				continue
			}
		}
		decisions = append(decisions, raw)
	}
	definitions, err := exportengine.BuildNativeTextTree(exportengine.Snapshot{
		Schema: snapshot.Schema, ProjectID: snapshot.ProjectID, Project: snapshot.Project,
		Tables: map[string][]json.RawMessage{"decision_categories": rows, "decisions": decisions},
	}, false)
	if err != nil {
		log.Printf("[gameview] project %s data layer: tab definitions: %v", p.Dir(), err)
		return files
	}
	out := make(map[string][]byte, len(files))
	for name, body := range files {
		if !isCategoryDefinitionFile(name) {
			out[name] = body
		}
	}
	for name, body := range definitions {
		if isCategoryDefinitionFile(name) {
			out[name] = body
		}
	}
	return out
}

func withTable(snapshot exportengine.Snapshot, name string, rows []json.RawMessage) exportengine.Snapshot {
	tables := make(map[string][]json.RawMessage, len(snapshot.Tables))
	for table, existing := range snapshot.Tables {
		tables[table] = existing
	}
	tables[name] = rows
	snapshot.Tables = tables
	return snapshot
}

type rawTexture struct {
	Path      string
	ObjectKey string
	SHA       string
}

func rawTexturesOf(snapshot exportengine.Snapshot, winners map[string]exportengine.NativePreviewWinner) []rawTexture {
	out := []rawTexture{}
	for _, raw := range snapshot.Tables["project_raw_files"] {
		var row struct {
			ID            json.Number `json:"id"`
			DestPath      string      `json:"dest_path"`
			IsBinary      bool        `json:"is_binary"`
			RawKind       string      `json:"raw_kind"`
			ContentOBSKey *string     `json:"content_obs_key"`
			ContentSHA256 *string     `json:"content_sha256"`
		}
		if json.Unmarshal(raw, &row) != nil || !row.IsBinary || row.RawKind == "empty_shadow" ||
			row.ContentOBSKey == nil || strings.TrimSpace(*row.ContentOBSKey) == "" {
			continue
		}
		switch strings.ToLower(path.Ext(row.DestPath)) {
		case ".dds", ".tga", ".png":
		default:
			continue
		}
		if winner, ok := winners[foldPath(row.DestPath)]; !ok || winner.SourceEntity != "project_raw_files:"+row.ID.String() {
			continue
		}
		texture := rawTexture{Path: row.DestPath, ObjectKey: strings.TrimSpace(*row.ContentOBSKey)}
		if row.ContentSHA256 != nil {
			texture.SHA = strings.TrimSpace(*row.ContentSHA256)
		}
		out = append(out, texture)
	}
	return out
}
