package military

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const subUnitTable = "sub_units"

var (
	errSubUnitMissing     = apiutil.NotFoundError("Sub-unit not found")
	errSubUnitUnavailable = detailError(http.StatusServiceUnavailable, "sub-unit store unavailable")
)

var subUnitInternalTypes = map[string]bool{
	"infantry": true, "support": true, "artillery": true, "anti_tank": true,
	"anti_air": true, "armor": true, "fighter": true, "cas": true,
	"naval_bomber": true, "interceptor": true, "suicide": true,
	"tactical_bomber": true, "strategic_bomber": true, "air_transport": true,
	"missile": true, "submarine": true, "screen_ship": true,
	"capital_ship": true, "carrier": true, "motorized": true,
	"mechanized": true, "flame": true, "amphibious": true,
}

var subUnitBooleanFields = map[string]bool{
	"active": true, "cavalry": true, "special_forces": true, "marines": true,
	"can_exfiltrate_from_coast": true, "mountaineers": true,
	"can_be_parachuted": true, "affects_speed": true,
}

var subUnitCategoryTokenRE = regexp.MustCompile(`^[A-Za-z0-9_@.$:-]+$`)

func validateSubUnitTypes(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.TrimSpace(value)
		if subUnitBooleanFields[token] {
			return nil, fmt.Errorf("unit_types contains boolean field '%s'; set %s=yes instead", token, token)
		}
		if !subUnitInternalTypes[token] {
			return nil, fmt.Errorf("unsupported sub-unit internal type '%s'", token)
		}
		normalized = append(normalized, token)
	}
	return normalized, nil
}

func validateSubUnitCategories(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.TrimSpace(value)
		if token == "" || !subUnitCategoryTokenRE.MatchString(token) {
			return nil, fmt.Errorf("invalid sub-unit category token '%s'", token)
		}
		normalized = append(normalized, token)
	}
	return normalized, nil
}

type SubUnit struct {
	ID                     int64             `json:"id"`
	ProjectID              int64             `json:"project_id"`
	SubUnitID              string            `json:"sub_unit_id"`
	CountryTag             string            `json:"country_tag"`
	SourceFile             string            `json:"source_file"`
	Source                 string            `json:"source"`
	Ordering               int               `json:"ordering"`
	Name                   string            `json:"name"`
	NameEn                 string            `json:"name_en"`
	NameZh                 string            `json:"name_zh"`
	DescEn                 string            `json:"desc_en"`
	DescZh                 string            `json:"desc_zh"`
	DisplayNameLocs        map[string]string `json:"display_name_locs"`
	DescriptionLocs        map[string]string `json:"description_locs"`
	Abbreviation           string            `json:"abbreviation"`
	Sprite                 string            `json:"sprite"`
	MapIconCategory        string            `json:"map_icon_category"`
	Priority               *int              `json:"priority"`
	AiPriority             *int              `json:"ai_priority"`
	Transport              string            `json:"transport"`
	Group                  string            `json:"group"`
	Active                 *bool             `json:"active"`
	Cavalry                *bool             `json:"cavalry"`
	SpecialForces          *bool             `json:"special_forces"`
	Marines                *bool             `json:"marines"`
	CanExfiltrateFromCoast *bool             `json:"can_exfiltrate_from_coast"`
	Mountaineers           *bool             `json:"mountaineers"`
	CanBeParachuted        *bool             `json:"can_be_parachuted"`
	AffectsSpeed           *bool             `json:"affects_speed"`
	UnitTypes              []string          `json:"unit_types"`
	Categories             []string          `json:"categories"`
	Essential              []string          `json:"essential"`
	Need                   map[string]any    `json:"need"`
	Stats                  map[string]any    `json:"stats"`
	TerrainModifiers       map[string]any    `json:"terrain_modifiers"`
	RawExtras              map[string]any    `json:"raw_extras"`
}

func subUnitFromRow(row project.Row) SubUnit {
	var u SubUnit
	fromRow(row, map[string]any{"source": "created"}, &u)
	if u.DisplayNameLocs == nil {
		u.DisplayNameLocs = map[string]string{}
	}
	if u.DescriptionLocs == nil {
		u.DescriptionLocs = map[string]string{}
	}
	if u.UnitTypes == nil {
		u.UnitTypes = []string{}
	}
	if u.Categories == nil {
		u.Categories = []string{}
	}
	if u.Essential == nil {
		u.Essential = []string{}
	}
	if u.Need == nil {
		u.Need = map[string]any{}
	}
	if u.Stats == nil {
		u.Stats = map[string]any{}
	}
	if u.TerrainModifiers == nil {
		u.TerrainModifiers = map[string]any{}
	}
	if u.RawExtras == nil {
		u.RawExtras = map[string]any{}
	}
	return u
}

func listSubUnits(p *project.Project, countryTag, sourceFile *string) ([]SubUnit, error) {
	rows, err := p.Rows(subUnitTable)
	if err != nil {
		return nil, err
	}
	out := []SubUnit{}
	for _, row := range rows {
		if countryTag != nil && !stringEquals(row["country_tag"], upper(*countryTag)) {
			continue
		}
		if sourceFile != nil && !stringEquals(row["source_file"], strings.ReplaceAll(*sourceFile, "\\", "/")) {
			continue
		}
		out = append(out, subUnitFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
		if a.Ordering != b.Ordering {
			return a.Ordering < b.Ordering
		}
		return a.SubUnitID < b.SubUnitID
	})
	return out, nil
}

func getSubUnit(p *project.Project, id int64) (SubUnit, error) {
	rows, err := p.Rows(subUnitTable)
	if err != nil {
		return SubUnit{}, err
	}
	if row := findRow(rows, id); row != nil {
		return subUnitFromRow(row), nil
	}
	return SubUnit{}, errSubUnitMissing
}

type subUnitCreateInput struct {
	SubUnitID              string            `json:"sub_unit_id"`
	CountryTag             string            `json:"country_tag"`
	Ordering               int               `json:"ordering"`
	Name                   string            `json:"name"`
	NameEn                 string            `json:"name_en"`
	NameZh                 string            `json:"name_zh"`
	DescEn                 string            `json:"desc_en"`
	DescZh                 string            `json:"desc_zh"`
	DisplayNameLocs        map[string]string `json:"display_name_locs"`
	DescriptionLocs        map[string]string `json:"description_locs"`
	Abbreviation           string            `json:"abbreviation"`
	Sprite                 string            `json:"sprite"`
	MapIconCategory        string            `json:"map_icon_category"`
	Priority               *int              `json:"priority"`
	AiPriority             *int              `json:"ai_priority"`
	Transport              string            `json:"transport"`
	Group                  string            `json:"group"`
	Active                 *bool             `json:"active"`
	Cavalry                *bool             `json:"cavalry"`
	SpecialForces          *bool             `json:"special_forces"`
	Marines                *bool             `json:"marines"`
	CanExfiltrateFromCoast *bool             `json:"can_exfiltrate_from_coast"`
	Mountaineers           *bool             `json:"mountaineers"`
	CanBeParachuted        *bool             `json:"can_be_parachuted"`
	AffectsSpeed           *bool             `json:"affects_speed"`
	UnitTypes              []string          `json:"unit_types"`
	Categories             []string          `json:"categories"`
	Essential              []string          `json:"essential"`
	Need                   map[string]any    `json:"need"`
	Stats                  map[string]any    `json:"stats"`
	TerrainModifiers       map[string]any    `json:"terrain_modifiers"`
	RawExtras              map[string]any    `json:"raw_extras"`
}

func createSubUnit(p *project.Project, in subUnitCreateInput) (SubUnit, error) {
	unitTypes, err := validateSubUnitTypes(in.UnitTypes)
	if err != nil {
		return SubUnit{}, detailError(http.StatusUnprocessableEntity, err.Error())
	}
	categories, err := validateSubUnitCategories(in.Categories)
	if err != nil {
		return SubUnit{}, detailError(http.StatusUnprocessableEntity, err.Error())
	}
	base := strings.TrimSpace(in.SubUnitID)
	if base == "" {
		base = "new_sub_unit"
	}
	if in.Essential == nil {
		in.Essential = []string{}
	}
	if in.Need == nil {
		in.Need = map[string]any{}
	}
	if in.Stats == nil {
		in.Stats = map[string]any{}
	}
	if in.TerrainModifiers == nil {
		in.TerrainModifiers = map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	var created SubUnit
	err = p.Write([]string{subUnitTable}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(subUnitTable) {
			taken[text(row, "sub_unit_id")] = true
		}
		candidate, suffix := base, 0
		for taken[candidate] {
			suffix++
			candidate = fmt.Sprintf("%s_%d", base, suffix)
			if suffix > 1000 {
				return detailError(http.StatusConflict, "sub_unit_id collision (cap 1000)")
			}
		}
		row := project.Row{
			"sub_unit_id": candidate, "country_tag": upper(in.CountryTag),
			"source_file": "", "source": "created", "ordering": jsonValue(in.Ordering),
			"name": in.Name, "name_en": in.NameEn, "name_zh": in.NameZh,
			"desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
			"abbreviation": in.Abbreviation, "sprite": in.Sprite, "map_icon_category": in.MapIconCategory,
			"priority": jsonValue(in.Priority), "ai_priority": jsonValue(in.AiPriority),
			"transport": in.Transport, "group": in.Group,
			"active": jsonValue(in.Active), "cavalry": jsonValue(in.Cavalry),
			"special_forces": jsonValue(in.SpecialForces), "marines": jsonValue(in.Marines),
			"can_exfiltrate_from_coast": jsonValue(in.CanExfiltrateFromCoast),
			"mountaineers":              jsonValue(in.Mountaineers),
			"can_be_parachuted":         jsonValue(in.CanBeParachuted),
			"affects_speed":             jsonValue(in.AffectsSpeed),
			"unit_types":                jsonValue(unitTypes), "categories": jsonValue(categories),
			"essential": jsonValue(in.Essential), "need": jsonValue(in.Need),
			"stats": jsonValue(in.Stats), "terrain_modifiers": jsonValue(in.TerrainModifiers),
			"raw_extras": jsonValue(in.RawExtras),
		}
		created = subUnitFromRow(tx.Insert(subUnitTable, row))
		return nil
	})
	return created, err
}

type subUnitPatch struct {
	SubUnitID              workbench.Optional[string]            `json:"sub_unit_id"`
	CountryTag             workbench.Optional[string]            `json:"country_tag"`
	SourceFile             workbench.Optional[string]            `json:"source_file"`
	Source                 workbench.Optional[string]            `json:"source"`
	Ordering               workbench.Optional[int]               `json:"ordering"`
	Name                   workbench.Optional[string]            `json:"name"`
	NameEn                 workbench.Optional[string]            `json:"name_en"`
	NameZh                 workbench.Optional[string]            `json:"name_zh"`
	DescEn                 workbench.Optional[string]            `json:"desc_en"`
	DescZh                 workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs        workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs        workbench.Optional[map[string]string] `json:"description_locs"`
	Abbreviation           workbench.Optional[string]            `json:"abbreviation"`
	Sprite                 workbench.Optional[string]            `json:"sprite"`
	MapIconCategory        workbench.Optional[string]            `json:"map_icon_category"`
	Priority               workbench.Optional[int]               `json:"priority"`
	AiPriority             workbench.Optional[int]               `json:"ai_priority"`
	Transport              workbench.Optional[string]            `json:"transport"`
	Group                  workbench.Optional[string]            `json:"group"`
	Active                 workbench.Optional[bool]              `json:"active"`
	Cavalry                workbench.Optional[bool]              `json:"cavalry"`
	SpecialForces          workbench.Optional[bool]              `json:"special_forces"`
	Marines                workbench.Optional[bool]              `json:"marines"`
	CanExfiltrateFromCoast workbench.Optional[bool]              `json:"can_exfiltrate_from_coast"`
	Mountaineers           workbench.Optional[bool]              `json:"mountaineers"`
	CanBeParachuted        workbench.Optional[bool]              `json:"can_be_parachuted"`
	AffectsSpeed           workbench.Optional[bool]              `json:"affects_speed"`
	UnitTypes              workbench.Optional[[]string]          `json:"unit_types"`
	Categories             workbench.Optional[[]string]          `json:"categories"`
	Essential              workbench.Optional[[]string]          `json:"essential"`
	Need                   workbench.Optional[map[string]any]    `json:"need"`
	Stats                  workbench.Optional[map[string]any]    `json:"stats"`
	TerrainModifiers       workbench.Optional[map[string]any]    `json:"terrain_modifiers"`
	RawExtras              workbench.Optional[map[string]any]    `json:"raw_extras"`
}

func updateSubUnit(p *project.Project, id int64, patch subUnitPatch) (SubUnit, error) {
	var updated SubUnit
	err := p.Write([]string{subUnitTable}, func(tx *project.Tx) error {
		row := tx.Find(subUnitTable, id)
		if row == nil {
			return errSubUnitMissing
		}
		cur := subUnitFromRow(row)
		if patch.Source.Set && optStr(patch.Source) != cur.Source {
			return detailError(http.StatusUnprocessableEntity, "source is managed by the import/create workflow")
		}
		if patch.SourceFile.Set {
			requested := strings.ReplaceAll(optStr(patch.SourceFile), "\\", "/")
			if requested != strings.ReplaceAll(cur.SourceFile, "\\", "/") {
				return detailError(http.StatusUnprocessableEntity,
					"source_file is managed; create a new sub-unit for a project-owned file or import the complete original file")
			}
		}
		effectiveTypes := cur.UnitTypes
		if patch.UnitTypes.Set && patch.UnitTypes.Value != nil {
			effectiveTypes = *patch.UnitTypes.Value
		}
		effectiveCategories := cur.Categories
		if patch.Categories.Set && patch.Categories.Value != nil {
			effectiveCategories = *patch.Categories.Value
		}
		validTypes, err := validateSubUnitTypes(effectiveTypes)
		if err != nil {
			return detailError(http.StatusUnprocessableEntity, err.Error())
		}
		validCategories, err := validateSubUnitCategories(effectiveCategories)
		if err != nil {
			return detailError(http.StatusUnprocessableEntity, err.Error())
		}

		changes := map[string]any{}
		set := func(column string, isSet bool, value any) {
			if isSet {
				changes[column] = value
			}
		}
		if patch.SubUnitID.Set {
			changes["sub_unit_id"] = strings.TrimSpace(optStr(patch.SubUnitID))
		}
		if patch.CountryTag.Set {
			changes["country_tag"] = upper(optStr(patch.CountryTag))
		}
		set("ordering", patch.Ordering.Set, opt(patch.Ordering))
		set("name", patch.Name.Set, opt(patch.Name))
		set("name_en", patch.NameEn.Set, opt(patch.NameEn))
		set("name_zh", patch.NameZh.Set, opt(patch.NameZh))
		set("desc_en", patch.DescEn.Set, opt(patch.DescEn))
		set("desc_zh", patch.DescZh.Set, opt(patch.DescZh))
		set("display_name_locs", patch.DisplayNameLocs.Set, opt(patch.DisplayNameLocs))
		set("description_locs", patch.DescriptionLocs.Set, opt(patch.DescriptionLocs))
		set("abbreviation", patch.Abbreviation.Set, opt(patch.Abbreviation))
		set("sprite", patch.Sprite.Set, opt(patch.Sprite))
		set("map_icon_category", patch.MapIconCategory.Set, opt(patch.MapIconCategory))
		set("priority", patch.Priority.Set, opt(patch.Priority))
		set("ai_priority", patch.AiPriority.Set, opt(patch.AiPriority))
		set("transport", patch.Transport.Set, opt(patch.Transport))
		set("group", patch.Group.Set, opt(patch.Group))
		set("active", patch.Active.Set, opt(patch.Active))
		set("cavalry", patch.Cavalry.Set, opt(patch.Cavalry))
		set("special_forces", patch.SpecialForces.Set, opt(patch.SpecialForces))
		set("marines", patch.Marines.Set, opt(patch.Marines))
		set("can_exfiltrate_from_coast", patch.CanExfiltrateFromCoast.Set, opt(patch.CanExfiltrateFromCoast))
		set("mountaineers", patch.Mountaineers.Set, opt(patch.Mountaineers))
		set("can_be_parachuted", patch.CanBeParachuted.Set, opt(patch.CanBeParachuted))
		set("affects_speed", patch.AffectsSpeed.Set, opt(patch.AffectsSpeed))

		set("unit_types", patch.UnitTypes.Set, jsonValue(validTypes))
		set("categories", patch.Categories.Set, jsonValue(validCategories))
		set("essential", patch.Essential.Set, opt(patch.Essential))
		set("need", patch.Need.Set, opt(patch.Need))
		set("stats", patch.Stats.Set, opt(patch.Stats))
		set("terrain_modifiers", patch.TerrainModifiers.Set, opt(patch.TerrainModifiers))
		set("raw_extras", patch.RawExtras.Set, opt(patch.RawExtras))
		if cur.Source == "imported" {
			changes["source"] = "created"
		}
		if len(changes) == 0 {
			updated = cur
			return nil
		}
		if nullViolation(subUnitTable, changes) {
			return errSubUnitUnavailable
		}
		if newID, ok := changes["sub_unit_id"].(string); ok {
			for _, other := range tx.Rows(subUnitTable) {
				if project.Int64(other["id"]) != id && text(other, "sub_unit_id") == newID {
					return detailError(http.StatusConflict, "sub_unit_id collision")
				}
			}
		}
		next, err := tx.Update(subUnitTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = subUnitFromRow(next)
		return nil
	})
	return updated, err
}

func (h *handlers) subUnits(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	failSubUnit := func(err error, op string) { fail(w, err, op, "sub-unit store unavailable") }
	if len(rest) == 1 && rest[0] == "import" {
		h.subUnitImport(w, r, p)
		return
	}
	if len(rest) == 2 && rest[0] == "preview" {
		subUnitPreview(w, r, p, rest[1])
		return
	}
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := listSubUnits(p, queryPtr(r, "country_tag"), queryPtr(r, "source_file"))
			if err != nil {
				failSubUnit(err, "list sub-units")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var in subUnitCreateInput
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid sub-unit payload")
				return
			}
			item, err := createSubUnit(p, in)
			if err != nil {
				failSubUnit(err, "create sub-unit")
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
		detail(w, http.StatusNotFound, "Sub-unit not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := getSubUnit(p, id)
		if err != nil {
			failSubUnit(err, "get sub-unit")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch subUnitPatch
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid sub-unit payload")
			return
		}
		item, err := updateSubUnit(p, id, patch)
		if err != nil {
			failSubUnit(err, "update sub-unit")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, subUnitTable, id, errSubUnitMissing); err != nil {
			failSubUnit(err, "delete sub-unit")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}

func subUnitPreview(w http.ResponseWriter, r *http.Request, p *project.Project, rawID string) {
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, ok := previewID(w, rawID, "sub_unit_pk", "Sub-unit not found")
	if !ok {
		return
	}
	item, err := getSubUnit(p, id)
	if err != nil {
		previewFail(w, err, "sub-unit preview")
		return
	}
	if _, err := validateSubUnitTypes(item.UnitTypes); err != nil {
		detail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := validateSubUnitCategories(item.Categories); err != nil {
		detail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writePreview(w, item, emitmisc.GenerateSingleSubUnitScript, "sub-unit preview emit")
}
