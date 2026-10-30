package military

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

const equipmentTable = "equipments"

var errEquipmentMissing = apiutil.NotFoundError("Equipment not found")

var errEquipmentUnavailable = detailError(http.StatusServiceUnavailable, "equipment store unavailable")

type Equipment struct {
	ID                              int64             `json:"id"`
	ProjectID                       int64             `json:"project_id"`
	EquipmentID                     string            `json:"equipment_id"`
	CountryTag                      string            `json:"country_tag"`
	Name                            string            `json:"name"`
	EquipmentType                   string            `json:"equipment_type"`
	Year                            *int              `json:"year"`
	Picture                         string            `json:"picture"`
	Archetype                       string            `json:"archetype"`
	Parent                          string            `json:"parent"`
	IsArchetype                     bool              `json:"is_archetype"`
	IsBuildable                     *bool             `json:"is_buildable"`
	SoftAttack                      *float64          `json:"soft_attack"`
	HardAttack                      *float64          `json:"hard_attack"`
	ApAttack                        *float64          `json:"ap_attack"`
	AirAttack                       *float64          `json:"air_attack"`
	Breakthrough                    *float64          `json:"breakthrough"`
	Defense                         *float64          `json:"defense"`
	ArmorValue                      *float64          `json:"armor_value"`
	Hardness                        *float64          `json:"hardness"`
	MaxStrength                     *float64          `json:"max_strength"`
	BuildCostIc                     *float64          `json:"build_cost_ic"`
	Manpower                        *int              `json:"manpower"`
	Reliability                     *float64          `json:"reliability"`
	MaximumSpeed                    *float64          `json:"maximum_speed"`
	CanLicense                      *bool             `json:"can_license"`
	NavalSpeed                      *float64          `json:"naval_speed"`
	FireRange                       *float64          `json:"fire_range"`
	LgArmorPiercing                 *float64          `json:"lg_armor_piercing"`
	LgAttack                        *float64          `json:"lg_attack"`
	HgArmorPiercing                 *float64          `json:"hg_armor_piercing"`
	HgAttack                        *float64          `json:"hg_attack"`
	TorpedoAttack                   *float64          `json:"torpedo_attack"`
	AntiAirAttack                   *float64          `json:"anti_air_attack"`
	ShoreBombardment                *float64          `json:"shore_bombardment"`
	Evasion                         *float64          `json:"evasion"`
	SurfaceDetection                *float64          `json:"surface_detection"`
	SubAttack                       *float64          `json:"sub_attack"`
	SubDetection                    *float64          `json:"sub_detection"`
	SurfaceVisibility               *float64          `json:"surface_visibility"`
	SubVisibility                   *float64          `json:"sub_visibility"`
	NavalRange                      *float64          `json:"naval_range"`
	PortCapacityUsage               *float64          `json:"port_capacity_usage"`
	SearchAndDestroyCoordination    *float64          `json:"search_and_destroy_coordination"`
	ConvoyRaidingCoordination       *float64          `json:"convoy_raiding_coordination"`
	AirDefence                      *float64          `json:"air_defence"`
	AirRange                        *float64          `json:"air_range"`
	AirAgility                      *float64          `json:"air_agility"`
	AirGroundAttack                 *float64          `json:"air_ground_attack"`
	AirBombing                      *float64          `json:"air_bombing"`
	AirSuperiority                  *float64          `json:"air_superiority"`
	NavalStrikeAttack               *float64          `json:"naval_strike_attack"`
	NavalStrikeTargetting           *float64          `json:"naval_strike_targetting"`
	CarrierSize                     *float64          `json:"carrier_size"`
	DefaultCarrierCompositionWeight *float64          `json:"default_carrier_composition_weight"`
	CarrierCapable                  *bool             `json:"carrier_capable"`
	Resources                       map[string]any    `json:"resources"`
	RawExtras                       map[string]any    `json:"raw_extras"`
	NameEn                          string            `json:"name_en"`
	NameZh                          string            `json:"name_zh"`
	DescEn                          string            `json:"desc_en"`
	DescZh                          string            `json:"desc_zh"`
	DisplayNameLocs                 map[string]string `json:"display_name_locs"`
	DescriptionLocs                 map[string]string `json:"description_locs"`
	Source                          string            `json:"source"`
	SourceFile                      string            `json:"source_file"`
	Warnings                        []string          `json:"warnings"`
}

func equipmentFromRow(row project.Row) Equipment {
	var e Equipment
	fromRow(row, nil, &e)
	if e.Resources == nil {
		e.Resources = map[string]any{}
	}
	if e.RawExtras == nil {
		e.RawExtras = map[string]any{}
	}
	if e.DisplayNameLocs == nil {
		e.DisplayNameLocs = map[string]string{}
	}
	if e.DescriptionLocs == nil {
		e.DescriptionLocs = map[string]string{}
	}
	e.Warnings = []string{}
	if e.Source == "" {
		e.Source = "created"
	}
	return e
}

var equipmentDataColumns = []string{
	"equipment_id", "country_tag", "name", "equipment_type", "year", "picture",
	"archetype", "parent", "is_archetype", "is_buildable",
	"soft_attack", "hard_attack", "ap_attack", "air_attack", "breakthrough",
	"defense", "armor_value", "hardness", "max_strength",
	"build_cost_ic", "manpower", "reliability", "maximum_speed", "can_license",
	"naval_speed", "fire_range", "lg_armor_piercing", "lg_attack",
	"hg_armor_piercing", "hg_attack", "torpedo_attack", "anti_air_attack",
	"shore_bombardment", "evasion", "surface_detection", "sub_attack",
	"sub_detection", "surface_visibility", "sub_visibility", "naval_range",
	"port_capacity_usage", "search_and_destroy_coordination",
	"convoy_raiding_coordination", "air_defence", "air_range", "air_agility",
	"air_ground_attack", "air_bombing", "air_superiority", "naval_strike_attack",
	"naval_strike_targetting", "carrier_size", "default_carrier_composition_weight",
	"carrier_capable", "resources", "raw_extras", "name_en", "name_zh",
	"desc_en", "desc_zh", "display_name_locs", "description_locs",
	"source", "source_file",
}

func columnValue(table, column string, value any) (any, bool) {
	if value == nil {
		return nil, true
	}
	columnType := ""
	for _, c := range schema.Columns(table) {
		if c.Name == column {
			columnType = c.Type
			break
		}
	}
	switch columnType {
	case "json", "jsonb":
		return jsonValue(value), true
	case "integer", "bigint", "smallint":
		f, ok := value.(float64)
		if !ok || f != math.Trunc(f) || math.IsInf(f, 0) {
			return nil, false
		}
		return jsonValue(int64(f)), true
	case "double precision", "real", "numeric":
		f, ok := value.(float64)
		if !ok {
			return nil, false
		}
		return jsonValue(f), true
	case "boolean":
		b, ok := value.(bool)
		return b, ok
	default:
		s, ok := value.(string)
		return s, ok
	}
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "True"
		}
		return "False"
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func listEquipment(p *project.Project, countryTag, archetype *string, isArchetype *bool) ([]Equipment, error) {
	rows, err := p.Rows(equipmentTable)
	if err != nil {
		return nil, err
	}
	out := []Equipment{}
	for _, row := range rows {
		if countryTag != nil && !stringEquals(row["country_tag"], *countryTag) {
			continue
		}
		if archetype != nil && !stringEquals(row["archetype"], *archetype) {
			continue
		}
		if isArchetype != nil {
			if b, ok := row["is_archetype"].(bool); !ok || b != *isArchetype {
				continue
			}
		}
		out = append(out, equipmentFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].EquipmentID < out[j].EquipmentID })
	return out, nil
}

func stringEquals(v any, want string) bool {
	s, ok := v.(string)
	return ok && s == want
}

func getEquipment(p *project.Project, id int64) (Equipment, error) {
	rows, err := p.Rows(equipmentTable)
	if err != nil {
		return Equipment{}, err
	}
	if row := findRow(rows, id); row != nil {
		return equipmentFromRow(row), nil
	}
	return Equipment{}, errEquipmentMissing
}

func createEquipment(p *project.Project, in map[string]any) (Equipment, error) {
	eqID := strings.TrimSpace(anyToString(in["equipment_id"]))
	if eqID == "" {
		eqID = "new_equipment"
	}
	if tag, ok := in["country_tag"].(string); ok {
		in["country_tag"] = upper(tag)
	}
	defaults := map[string]any{
		"country_tag": "", "name": "", "equipment_type": "", "picture": "",
		"archetype": "", "parent": "", "is_archetype": false,
		"resources": map[string]any{}, "raw_extras": map[string]any{},
		"name_en": "", "name_zh": "", "desc_en": "", "desc_zh": "",
		"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
		"source": "created", "source_file": "",
	}
	var created Equipment
	err := p.Write([]string{equipmentTable}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(equipmentTable) {
			taken[text(row, "equipment_id")] = true
		}
		base, suffix := eqID, 0
		for taken[eqID] {
			suffix++
			eqID = fmt.Sprintf("%s_%d", base, suffix)
			if suffix > 1000 {
				return detailError(http.StatusConflict, "equipment_id collision (cap 1000)")
			}
		}
		in["equipment_id"] = eqID
		row := project.Row{}
		for _, column := range equipmentDataColumns {
			value, present := in[column]
			if !present {
				value = defaults[column]
			}
			converted, ok := columnValue(equipmentTable, column, value)
			if !ok {
				return errEquipmentUnavailable
			}
			row[column] = converted
		}
		if nullViolation(equipmentTable, row) {
			return errEquipmentUnavailable
		}
		created = equipmentFromRow(tx.Insert(equipmentTable, row))
		return nil
	})
	return created, err
}

func updateEquipment(p *project.Project, id int64, patch map[string]any) (Equipment, error) {
	var updated Equipment
	err := p.Write([]string{equipmentTable}, func(tx *project.Tx) error {
		current := tx.Find(equipmentTable, id)
		if current == nil {
			return errEquipmentMissing
		}
		changes := map[string]any{}
		for _, column := range equipmentDataColumns {
			if column == "source" || column == "source_file" {
				continue
			}
			value, present := patch[column]
			if !present {
				continue
			}
			if s, ok := value.(string); ok {
				switch column {
				case "equipment_id":
					value = strings.TrimSpace(s)
				case "country_tag":
					value = upper(s)
				}
			}
			converted, ok := columnValue(equipmentTable, column, value)
			if !ok {
				return errEquipmentUnavailable
			}
			changes[column] = converted
		}
		if textOr(current, "source", "created") == "imported" {
			changes["source"] = "created"
		}
		if len(changes) == 0 {
			updated = equipmentFromRow(current)
			return nil
		}
		if nullViolation(equipmentTable, changes) {
			return errEquipmentUnavailable
		}
		if newID, ok := changes["equipment_id"].(string); ok {
			for _, row := range tx.Rows(equipmentTable) {
				if project.Int64(row["id"]) != id && text(row, "equipment_id") == newID {
					return detailError(http.StatusConflict, "equipment_id collision")
				}
			}
		}
		row, err := tx.Update(equipmentTable, id, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		if err != nil {
			return err
		}
		updated = equipmentFromRow(row)
		return nil
	})
	return updated, err
}

type EquipmentEnabler struct {
	TechID   string `json:"tech_id"`
	Name     string `json:"name"`
	TechLine string `json:"tech_line"`
}

func equipmentEnabledBy(p *project.Project, equipmentID string) (map[string]any, error) {
	rows, err := p.Rows("technologies")
	if err != nil {
		return nil, err
	}
	techs := []EquipmentEnabler{}
	for _, row := range rows {
		if !jsonContains(row["enable_equipments"], equipmentID) {
			continue
		}
		techs = append(techs, EquipmentEnabler{
			TechID: text(row, "tech_id"), Name: text(row, "name"), TechLine: text(row, "tech_line"),
		})
	}
	return map[string]any{"equipment_id": equipmentID, "techs": techs}, nil
}

func jsonContains(v any, want string) bool {
	switch value := v.(type) {
	case []any:
		for _, item := range value {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	case map[string]any:
		_, ok := value[want]
		return ok
	case string:
		return value == want
	}
	return false
}

func (h *handlers) equipment(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	const missing = "Equipment not found"
	failEquipment := func(err error, op string) { fail(w, err, op, "equipment store unavailable") }
	if len(rest) == 1 && rest[0] == "import" {
		h.equipmentImport(w, r, p)
		return
	}
	if len(rest) == 2 && rest[0] == "enabled-by" {
		if r.Method != http.MethodGet {
			methodError(w, "GET only")
			return
		}
		result, err := equipmentEnabledBy(p, rest[1])
		if err != nil {
			failEquipment(err, "equipment enabled-by")
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if len(rest) == 2 && rest[0] == "preview" {
		equipmentPreview(w, r, p, rest[1])
		return
	}
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			q := r.URL.Query()
			countryTag, archetype := queryPtr(r, "country_tag"), queryPtr(r, "archetype")
			var isArchetype *bool
			if v, ok := q["is_archetype"]; ok && len(v) > 0 {
				b := v[0] == "true" || v[0] == "True" || v[0] == "1"
				isArchetype = &b
			}
			items, err := listEquipment(p, countryTag, archetype, isArchetype)
			if err != nil {
				failEquipment(err, "list equipment")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var in map[string]any
			if err := decodeBody(r, 4<<20, &in); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid equipment payload")
				return
			}
			if in == nil {
				in = map[string]any{}
			}
			item, err := createEquipment(p, in)
			if err != nil {
				failEquipment(err, "create equipment")
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
		item, err := getEquipment(p, id)
		if err != nil {
			failEquipment(err, "get equipment")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var patch map[string]any
		if err := decodeBody(r, 4<<20, &patch); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid equipment payload")
			return
		}
		item, err := updateEquipment(p, id, patch)
		if err != nil {
			failEquipment(err, "update equipment")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := deleteRow(p, equipmentTable, id, errEquipmentMissing); err != nil {
			failEquipment(err, "delete equipment")
			return
		}
		apiutil.NoContent(w)
	default:
		methodError(w, "GET, PUT or DELETE only")
	}
}

func equipmentPreview(w http.ResponseWriter, r *http.Request, p *project.Project, rawID string) {
	if r.Method != http.MethodGet {
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	id, ok := previewID(w, rawID, "equipment_id", "Equipment not found")
	if !ok {
		return
	}
	item, err := getEquipment(p, id)
	if err != nil {
		previewFail(w, err, "equipment preview")
		return
	}
	writePreview(w, item, emitmisc.GenerateSingleEquipmentScript, "equipment preview emit")
}
