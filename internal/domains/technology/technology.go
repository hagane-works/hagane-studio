package technology

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
	"haganeworks/internal/workbench"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps, game: newGameData(deps.Game)}
	const base = "/api/go/projects/{id}/technologies"
	mux.HandleFunc("GET "+base, h.list)
	mux.HandleFunc("POST "+base, h.create)
	mux.HandleFunc("GET "+base+"/{tid}", h.get)
	mux.HandleFunc("PUT "+base+"/{tid}", h.update)
	mux.HandleFunc("DELETE "+base+"/{tid}", h.delete)
	mux.HandleFunc("GET "+base+"/known-folders", h.knownFolders)
	mux.HandleFunc("GET "+base+"/preview/{line}", h.preview)
	mux.HandleFunc("POST "+base+"/{tid}/parse-preview", h.parsePreview)
	mux.HandleFunc("POST "+base+"/{tid}/upload-icon", h.uploadIcon)
	mux.HandleFunc("GET "+base+"/reachability/{tag}", h.reachability)
	mux.HandleFunc("GET "+base+"/reachability/{tag}/{tech}/paths", h.reachabilityPaths)

	mux.HandleFunc("GET "+base+"/surface", h.surfaceStatus)
	mux.HandleFunc("PUT "+base+"/surface/environment", h.surfaceEnvironment)
	mux.HandleFunc("POST "+base+"/surface/source", h.surfaceSource)
	mux.HandleFunc("POST "+base+"/surface/source/certified", h.surfaceCertifiedSource)
	mux.HandleFunc("POST "+base+"/surface/repair", h.surfaceRepair)
	mux.HandleFunc("GET "+base+"/surface/branches", h.listBranches)
	mux.HandleFunc("POST "+base+"/surface/branches", h.createBranch)
	mux.HandleFunc("GET "+base+"/surface/branches/{branch}", h.getBranch)
	mux.HandleFunc("PUT "+base+"/surface/branches/{branch}", h.updateBranch)
	mux.HandleFunc("DELETE "+base+"/surface/branches/{branch}", h.deleteBranch)
	mux.HandleFunc("POST "+base+"/surface/branches/{branch}/technologies", h.createBranchTechnology)

	mux.HandleFunc("GET /api/go/projects/{id}/tech-icons", h.listIcons)
	mux.HandleFunc("GET /api/go/projects/{id}/tech-icons/{tech}", h.serveIcon)
	mux.HandleFunc("GET /api/go/projects/{id}/tech-icons/{tech}/size", h.iconSize)
	mux.HandleFunc("GET /api/go/vanilla/tech-icon/{tech}", h.vanillaIcon)
}

type handlers struct {
	deps domains.Deps
	game *gameData
}

func detail(w http.ResponseWriter, status int, msg string) {
	apiutil.JSON(w, status, map[string]string{"detail": msg})
}

func (h *handlers) open(w http.ResponseWriter, r *http.Request) (*project.Project, bool) {
	p, err := h.deps.Open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return nil, false
	}
	return p, true
}

func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, strict bool, v any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	if strict {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(v)
}

var (
	errTechMissing = errors.New("Technology not found")

	errStore = errors.New("technology store unavailable")
)

type techConflict struct{ Detail string }

func (e techConflict) Error() string { return e.Detail }

type surfaceError struct{ Detail string }

func (e surfaceError) Error() string { return e.Detail }

func writeTechError(w http.ResponseWriter, err error) {
	var conflict techConflict
	var surface surfaceError
	switch {
	case errors.Is(err, errTechMissing):
		detail(w, http.StatusNotFound, "Technology not found")
	case errors.As(err, &conflict):
		detail(w, http.StatusConflict, conflict.Detail)
	case errors.As(err, &surface):
		detail(w, http.StatusUnprocessableEntity, surface.Detail)
	default:
		detail(w, http.StatusServiceUnavailable, "technology store unavailable")
	}
}

func techRowID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
	return id, err == nil && id > 0
}

func recordManifest(record project.Row) (techsurface.Manifest, error) {
	value, _ := record["tech_surface_manifest"].(map[string]any)
	if value == nil {
		return techsurface.Manifest{}, nil
	}
	return techsurface.Parse(value)
}

func listTechnologies(p *project.Project, techLine, countryTag string) ([]Technology, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return nil, err
	}
	if techLine != "" {
		kept := rows[:0]
		for _, row := range rows {
			if text(row, "tech_line", "") == techLine {
				kept = append(kept, row)
			}
		}
		rows = kept
	}
	sortForList(rows)
	out := []Technology{}
	for _, row := range rows {
		t := techFromRow(row)
		if countryTag != "" && !techVisibleForCountry(t.TechID, countryTag) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	items, err := listTechnologies(p, r.URL.Query().Get("tech_line"), r.URL.Query().Get("country_tag"))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "technology store unavailable")
		return
	}
	apiutil.JSON(w, http.StatusOK, items)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	id, valid := techRowID(r)
	if !valid {
		detail(w, http.StatusNotFound, "Technology not found")
		return
	}
	item, err := getTechnology(p, id)
	if err != nil {
		writeTechError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, item)
}

func getTechnology(p *project.Project, id int64) (Technology, error) {
	rows, err := p.Rows(table)
	if err != nil {
		return Technology{}, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return techFromRow(row), nil
		}
	}
	return Technology{}, errTechMissing
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	var in CreateInput
	if err := decodeBody(w, r, apiutil.MaxJSONBody, false, &in); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology payload")
		return
	}
	in.Normalize()
	vanilla := h.game.techIDs()
	var item Technology
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		manifest, err := recordManifest(tx.Record())
		if err != nil {
			return err
		}
		if protectedSurfaceFolder(manifest, in) {
			return surfaceError{Detail: "This certified research area is managed by its extension branch. Create the root or next technology through the research branch workflow."}
		}
		item, err = createTechnology(tx, in, vanilla)
		return err
	})
	if err != nil {
		writeTechError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusCreated, item)
}

func createTechnology(tx *project.Tx, in CreateInput, vanilla map[string]bool) (Technology, error) {
	techLine := strings.TrimSpace(workbench.StrDefault(in.TechLine, "infantry"))
	if techLine == "" {
		techLine = "infantry"
	}
	rawName := strings.TrimSpace(workbench.StrDefault(in.Name, "New Technology"))
	if rawName == "" {
		rawName = "New Technology"
	}
	derived := strings.Trim(strings.ReplaceAll(strings.ToLower(rawName), " ", "_"), "_")
	if derived == "" {
		derived = "new_technology"
	}
	techID := strings.TrimSpace(in.TechID)
	if techID == "" {
		techID = techLine + "_" + derived
	}
	folder := strings.TrimSpace(workbench.StrDefault(in.Folder, ""))
	if folder == "" {
		if f, ok := techLineFolders[techLine]; ok {
			folder = f
		} else {
			folder = techLine + "_folder"
		}
	}
	placements := normalizedFolderPlacements(in.FolderPlacements, folder)
	if placementFolder, placementX, placementY, ok := firstFolderPlacement(placements); ok {
		folder = placementFolder
		in.PositionX = placementX
		in.PositionY = placementY
	}
	taken := map[string]bool{}
	for _, row := range tx.Rows(table) {
		taken[text(row, "tech_id", "")] = true
	}
	baseTechID := techID
	for suffix := 1; taken[techID]; suffix++ {
		if suffix > 1000 {
			return Technology{}, techConflict{Detail: fmt.Sprintf(
				"无法为 `%s` 找到唯一 tech_id (已尝试 %s_1 至 %s_1000)· 请在 TechEditor 内显式指定 tech_id。",
				baseTechID, baseTechID, baseTechID)}
		}
		techID = fmt.Sprintf("%s_%d", baseTechID, suffix)
	}
	source := workbench.StrDefault(in.Source, "created")
	researchCost := 1.0
	if in.ResearchCost != nil {
		researchCost = *in.ResearchCost
	}
	startYear := 1936
	if in.StartYear != nil {
		startYear = *in.StartYear
	}
	stored := tx.Insert(table, project.Row{
		"tech_id": techID, "country_tag": in.CountryTag, "name": rawName, "tech_line": techLine,
		"research_cost": researchCost, "start_year": int64(startYear), "folder": folder,
		"position_x": in.PositionX, "position_y": in.PositionY, "position_y_is_explicit": in.PositionYIsExplicit,
		"prerequisites": jsonValue(in.Prerequisites), "categories": jsonValue(in.Categories),
		"modifiers": jsonValue(in.Modifiers), "ai_will_do": workbench.StrDefault(in.AIWillDo, "factor = 1"),
		"name_en": workbench.FirstNonBlank(in.NameEn, rawName), "name_zh": in.NameZh,
		"desc_en": in.DescEn, "desc_zh": in.DescZh,
		"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
		"is_banner": in.IsBanner, "force_use_small_tech_layout": in.ForceUseSmallTechLayout,
		"enable_equipments": jsonValue(in.EnableEquipments), "enable_subunits": jsonValue(in.EnableSubunits),
		"enable_equipment_modules": jsonValue(in.EnableEquipmentModules),
		"enable_building":          in.EnableBuilding, "enable_tactic": in.EnableTactic,
		"xor": jsonValue(in.Xor), "dependencies": jsonValue(in.Dependencies),
		"sub_technologies": jsonValue(in.SubTechnologies), "doctrine": in.Doctrine,
		"allow": in.Allow, "allow_branch": in.AllowBranch, "on_research_complete": in.OnResearchComplete,
		"on_research_complete_limit": in.OnResearchCompleteLimit, "path": jsonValue(in.Path),
		"xp_research_type": in.XpResearchType, "xp_boost_cost": int64(in.XpBoostCost),
		"xp_research_bonus": jsonValue(in.XpResearchBonus),
		"source":            source, "raw_extras": map[string]any{}, "folder_placements": jsonValue(placements),
		"source_file": "", "has_obs": false,
	})
	t := techFromRow(stored)
	if warning := vanillaTechCollisionWarning(vanilla, techID, in.CountryTag, source); warning != nil {
		t.Warnings = []string{*warning}
	}
	return t, nil
}

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	id, valid := techRowID(r)
	if !valid {
		detail(w, http.StatusNotFound, "Technology not found")
		return
	}
	var patch Patch
	if err := decodeBody(w, r, apiutil.MaxJSONBody, false, &patch); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology payload")
		return
	}
	var item Technology
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		var err error
		item, err = updateTechnology(tx, id, patch)
		return err
	})
	if err != nil {
		writeTechError(w, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, item)
}

func optionalValue[T any](o workbench.Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	return *o.Value
}

func optionalJSON[T any](o workbench.Optional[T]) any {
	if o.Value == nil {
		return nil
	}
	return jsonValue(*o.Value)
}

func updateTechnology(tx *project.Tx, id int64, p Patch) (Technology, error) {
	current := tx.Find(table, id)
	if current == nil {
		return Technology{}, errTechMissing
	}
	curTechID, curSource, curFolder := text(current, "tech_id", ""), text(current, "source", ""), text(current, "folder", "")
	manifest, err := recordManifest(tx.Record())
	if err != nil {
		return Technology{}, err
	}
	if surfaceManagedTechnology(manifest, curTechID) && (p.Folder.Set || p.FolderPlacements.Set || p.PositionX.Set || p.PositionY.Set || p.PositionYIsExplicit.Set || p.Path.Set || p.TechLine.Set || p.StartYear.Set) {
		return Technology{}, surfaceError{Detail: "A certified branch owns this technology's research area, visual path, and grid position. Edit its descriptive fields here, but add branch technologies through the controlled research workflow."}
	}
	if p.TechID.Set && p.TechID.Value != nil {
		v := strings.TrimSpace(*p.TechID.Value)
		p.TechID.Value = &v
	}
	var currentPlacements []map[string]any
	decodeColumn(current["folder_placements"], &currentPlacements)
	if p.FolderPlacements.Set && p.FolderPlacements.Value != nil {
		fallback := curFolder
		if p.Folder.Set && p.Folder.Value != nil && strings.TrimSpace(*p.Folder.Value) != "" {
			fallback = strings.TrimSpace(*p.Folder.Value)
		}
		normalized := normalizedFolderPlacements(*p.FolderPlacements.Value, fallback)
		p.FolderPlacements.Value = &normalized
		if folder, x, y, ok := firstFolderPlacement(normalized); ok {
			p.Folder = workbench.Optional[string]{Set: true, Value: &folder}
			p.PositionX = workbench.Optional[float64]{Set: true, Value: &x}
			p.PositionY = workbench.Optional[float64]{Set: true, Value: &y}
		}
	} else if len(currentPlacements) > 0 &&
		((p.Folder.Set && p.Folder.Value != nil) || (p.PositionX.Set && p.PositionX.Value != nil) || (p.PositionY.Set && p.PositionY.Value != nil)) {
		normalized := normalizedFolderPlacements(currentPlacements, curFolder)
		first := normalized[0]
		if p.Folder.Set && p.Folder.Value != nil {
			first["folder"] = strings.TrimSpace(*p.Folder.Value)
		}
		if p.PositionX.Set && p.PositionX.Value != nil {
			first["x"] = *p.PositionX.Value
		}
		if p.PositionY.Set && p.PositionY.Value != nil {
			first["y"] = *p.PositionY.Value
		}
		p.FolderPlacements = workbench.Optional[[]map[string]any]{Set: true, Value: &normalized}
	}
	sets := map[string]any{}
	add := func(column string, set bool, value any) {
		if set {
			sets[column] = value
		}
	}
	add("tech_id", p.TechID.Set, optionalValue(p.TechID))
	add("country_tag", p.CountryTag.Set, optionalValue(p.CountryTag))
	add("name", p.Name.Set, optionalValue(p.Name))
	add("tech_line", p.TechLine.Set, optionalValue(p.TechLine))
	add("research_cost", p.ResearchCost.Set, optionalValue(p.ResearchCost))
	add("start_year", p.StartYear.Set, optionalJSON(p.StartYear))
	add("folder", p.Folder.Set, optionalValue(p.Folder))
	add("position_x", p.PositionX.Set, optionalValue(p.PositionX))
	add("position_y", p.PositionY.Set, optionalValue(p.PositionY))
	add("folder_placements", p.FolderPlacements.Set, optionalJSON(p.FolderPlacements))
	add("position_y_is_explicit", p.PositionYIsExplicit.Set, optionalValue(p.PositionYIsExplicit))
	add("prerequisites", p.Prerequisites.Set, optionalJSON(p.Prerequisites))
	add("categories", p.Categories.Set, optionalJSON(p.Categories))
	add("modifiers", p.Modifiers.Set, optionalJSON(p.Modifiers))
	add("ai_will_do", p.AIWillDo.Set, optionalValue(p.AIWillDo))
	add("raw_extras", p.RawExtras.Set, optionalJSON(p.RawExtras))
	add("name_en", p.NameEn.Set, optionalValue(p.NameEn))
	add("name_zh", p.NameZh.Set, optionalValue(p.NameZh))
	add("desc_en", p.DescEn.Set, optionalValue(p.DescEn))
	add("desc_zh", p.DescZh.Set, optionalValue(p.DescZh))
	add("display_name_locs", p.DisplayNameLocs.Set, optionalJSON(p.DisplayNameLocs))
	add("description_locs", p.DescriptionLocs.Set, optionalJSON(p.DescriptionLocs))

	flipToCreated := curSource == "imported"
	if p.Source.Set && p.Source.Value != nil {
		flipToCreated = *p.Source.Value == "imported"
	}
	add("source", p.Source.Set && !flipToCreated, optionalValue(p.Source))
	add("is_banner", p.IsBanner.Set, optionalValue(p.IsBanner))
	add("force_use_small_tech_layout", p.ForceUseSmallTechLayout.Set, optionalValue(p.ForceUseSmallTechLayout))
	add("enable_equipments", p.EnableEquipments.Set, optionalJSON(p.EnableEquipments))
	add("enable_subunits", p.EnableSubunits.Set, optionalJSON(p.EnableSubunits))
	add("enable_equipment_modules", p.EnableEquipmentModules.Set, optionalJSON(p.EnableEquipmentModules))
	add("enable_building", p.EnableBuilding.Set, optionalValue(p.EnableBuilding))
	add("enable_tactic", p.EnableTactic.Set, optionalValue(p.EnableTactic))
	add("xor", p.Xor.Set, optionalJSON(p.Xor))
	add("dependencies", p.Dependencies.Set, optionalJSON(p.Dependencies))
	add("sub_technologies", p.SubTechnologies.Set, optionalJSON(p.SubTechnologies))
	add("doctrine", p.Doctrine.Set, optionalValue(p.Doctrine))
	add("allow", p.Allow.Set, optionalValue(p.Allow))
	add("allow_branch", p.AllowBranch.Set, optionalValue(p.AllowBranch))
	add("on_research_complete", p.OnResearchComplete.Set, optionalValue(p.OnResearchComplete))
	add("on_research_complete_limit", p.OnResearchCompleteLimit.Set, optionalValue(p.OnResearchCompleteLimit))
	add("path", p.Path.Set, optionalJSON(p.Path))
	add("xp_research_type", p.XpResearchType.Set, optionalValue(p.XpResearchType))
	add("xp_boost_cost", p.XpBoostCost.Set, optionalJSON(p.XpBoostCost))
	add("xp_research_bonus", p.XpResearchBonus.Set, optionalJSON(p.XpResearchBonus))
	if flipToCreated {
		sets["source"] = "created"
	}
	if len(sets) > 0 {
		if err := checkColumns(tx, id, sets); err != nil {
			return Technology{}, err
		}
		if _, err := tx.Update(table, id, func(row project.Row) {
			for column, value := range sets {
				row[column] = value
			}
		}); err != nil {
			return Technology{}, err
		}
	}
	return techFromRow(tx.Find(table, id)), nil
}

func checkColumns(tx *project.Tx, id int64, sets map[string]any) error {
	for _, column := range schema.Columns(table) {
		if value, ok := sets[column.Name]; ok && column.NotNull && value == nil {
			return errStore
		}
	}
	if techID, ok := sets["tech_id"].(string); ok {
		for _, row := range tx.Rows(table) {
			if project.Int64(row["id"]) != id && text(row, "tech_id", "") == techID {
				return errStore
			}
		}
	}
	return nil
}

func (h *handlers) delete(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	id, valid := techRowID(r)
	if !valid {
		detail(w, http.StatusNotFound, "Technology not found")
		return
	}
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		row := tx.Find(table, id)
		if row == nil {
			return errTechMissing
		}
		manifest, err := recordManifest(tx.Record())
		if err != nil {
			return err
		}
		if surfaceManagedTechnology(manifest, text(row, "tech_id", "")) {
			return surfaceError{Detail: "A certified branch owns this technology. P0 does not support deleting an individual branch node because it would leave a broken research path."}
		}
		tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == id })
		return nil
	})
	if err != nil {
		writeTechError(w, err)
		return
	}
	apiutil.NoContent(w)
}
