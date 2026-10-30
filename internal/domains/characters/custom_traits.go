package characters

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

const traitTable = "custom_traits"

var validTraitTypes = map[string]bool{"country_leader": true, "unit_leader": true, "scientist": true}

const traitTypesDetail = "Invalid trait_type. Must be one of: ['country_leader', 'scientist', 'unit_leader']"

var (
	errTraitMissing     = detail(http.StatusNotFound, "Custom trait not found")
	errTraitUnavailable = detail(http.StatusServiceUnavailable, "trait store unavailable")
)

type CustomTrait struct {
	ID               int64             `json:"id"`
	ProjectID        int64             `json:"project_id"`
	TraitType        string            `json:"trait_type"`
	TraitID          string            `json:"trait_id"`
	CountryTag       string            `json:"country_tag"`
	NameEn           string            `json:"name_en"`
	NameZh           string            `json:"name_zh"`
	DescEn           string            `json:"desc_en"`
	DescZh           string            `json:"desc_zh"`
	DisplayNameLocs  map[string]string `json:"display_name_locs"`
	DescriptionLocs  map[string]string `json:"description_locs"`
	Modifier         map[string]any    `json:"modifier"`
	AIWillDo         string            `json:"ai_will_do"`
	Random           bool              `json:"random"`
	ExtraFields      map[string]any    `json:"extra_fields"`
	RawExtras        map[string]any    `json:"raw_extras"`
	TargetedModifier []map[string]any  `json:"targeted_modifier"`
}

func traitFromRow(row project.Row) CustomTrait {
	t := CustomTrait{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		TraitType: str(row["trait_type"]), TraitID: str(row["trait_id"]), CountryTag: str(row["country_tag"]),
		NameEn: str(row["name_en"]), NameZh: str(row["name_zh"]), DescEn: str(row["desc_en"]), DescZh: str(row["desc_zh"]),
		AIWillDo: str(row["ai_will_do"]), Random: boolOr(row["random"], true),
	}
	decodeColumn(row["display_name_locs"], &t.DisplayNameLocs)
	decodeColumn(row["description_locs"], &t.DescriptionLocs)
	decodeColumn(row["modifier"], &t.Modifier)
	decodeColumn(row["extra_fields"], &t.ExtraFields)
	decodeColumn(row["raw_extras"], &t.RawExtras)
	decodeColumn(row["targeted_modifier"], &t.TargetedModifier)
	if t.DisplayNameLocs == nil {
		t.DisplayNameLocs = map[string]string{}
	}
	if t.DescriptionLocs == nil {
		t.DescriptionLocs = map[string]string{}
	}
	if t.Modifier == nil {
		t.Modifier = map[string]any{}
	}
	if t.ExtraFields == nil {
		t.ExtraFields = map[string]any{}
	}
	if t.RawExtras == nil {
		t.RawExtras = map[string]any{}
	}
	if t.TargetedModifier == nil {
		t.TargetedModifier = []map[string]any{}
	}
	return t
}

type CustomTraitCreateInput struct {
	TraitType        *string           `json:"trait_type"`
	TraitID          *string           `json:"trait_id"`
	CountryTag       string            `json:"country_tag"`
	NameEn           string            `json:"name_en"`
	NameZh           string            `json:"name_zh"`
	DescEn           string            `json:"desc_en"`
	DescZh           string            `json:"desc_zh"`
	DisplayNameLocs  map[string]string `json:"display_name_locs"`
	DescriptionLocs  map[string]string `json:"description_locs"`
	Modifier         map[string]any    `json:"modifier"`
	AIWillDo         string            `json:"ai_will_do"`
	Random           *bool             `json:"random"`
	ExtraFields      map[string]any    `json:"extra_fields"`
	RawExtras        map[string]any    `json:"raw_extras"`
	TargetedModifier []map[string]any  `json:"targeted_modifier"`
}

func (in *CustomTraitCreateInput) Normalize() error {
	if in.TraitID == nil {
		return fmt.Errorf("trait_id is required")
	}
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	if in.Modifier == nil {
		in.Modifier = map[string]any{}
	}
	if in.ExtraFields == nil {
		in.ExtraFields = map[string]any{}
	}
	if in.RawExtras == nil {
		in.RawExtras = map[string]any{}
	}
	if in.TargetedModifier == nil {
		in.TargetedModifier = []map[string]any{}
	}
	return nil
}

func listCustomTraits(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rows, err := p.Rows(traitTable)
	if err != nil {
		return err
	}
	query := r.URL.Query()
	traitType, countryTag := query.Get("trait_type"), query.Get("country_tag")
	out := []CustomTrait{}
	for _, row := range rows {
		if traitType != "" && str(row["trait_type"]) != traitType {
			continue
		}
		if countryTag != "" && str(row["country_tag"]) != countryTag {
			continue
		}
		out = append(out, traitFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TraitType != out[j].TraitType {
			return out[i].TraitType < out[j].TraitType
		}
		return out[i].TraitID < out[j].TraitID
	})
	apiutil.JSON(w, http.StatusOK, out)
	return nil
}

func getCustomTrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tid, ok := pathID(r, "tid")
	if !ok {
		return errTraitMissing
	}
	row, err := findRow(p, traitTable, tid)
	if err != nil {
		return err
	}
	if row == nil {
		return errTraitMissing
	}
	apiutil.JSON(w, http.StatusOK, traitFromRow(row))
	return nil
}

func createCustomTrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in CustomTraitCreateInput
	if err := decodeBody(w, r, 2<<20, &in); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid trait payload")
	}
	if err := in.Normalize(); err != nil {
		return detail(http.StatusUnprocessableEntity, err.Error())
	}
	traitType := strDefault(in.TraitType, "country_leader")
	if !validTraitTypes[traitType] {
		return detail(http.StatusBadRequest, traitTypesDetail)
	}
	cleaned := strings.TrimSpace(*in.TraitID)
	if cleaned == "" {
		return detail(http.StatusBadRequest, "trait_id cannot be empty")
	}
	countryTag := strings.ToUpper(firstNonBlank(in.CountryTag, projectCountryTag(p)))
	var created project.Row
	err := p.Write([]string{traitTable}, func(tx *project.Tx) error {
		for _, row := range tx.Rows(traitTable) {
			if str(row["trait_type"]) == traitType && str(row["trait_id"]) == cleaned {
				return detail(http.StatusConflict, fmt.Sprintf("Custom trait '%s' already exists for type '%s'", cleaned, traitType))
			}
		}

		created = tx.Insert(traitTable, project.Row{
			"country_tag": countryTag, "trait_type": traitType, "trait_id": cleaned,
			"name_en": firstNonBlank(in.NameEn, cleaned), "name_zh": in.NameZh,
			"desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": jsonValue(in.DisplayNameLocs), "description_locs": jsonValue(in.DescriptionLocs),
			"modifier": jsonValue(in.Modifier), "ai_will_do": in.AIWillDo,
			"random": boolDefault(in.Random, true), "extra_fields": jsonValue(in.ExtraFields),
			"targeted_modifier": jsonValue(in.TargetedModifier), "raw_extras": jsonValue(in.RawExtras),
		})
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, traitFromRow(created))
	return nil
}

type CustomTraitPatch struct {
	TraitType        workbench.Optional[string]            `json:"trait_type"`
	TraitID          workbench.Optional[string]            `json:"trait_id"`
	CountryTag       workbench.Optional[string]            `json:"country_tag"`
	NameEn           workbench.Optional[string]            `json:"name_en"`
	NameZh           workbench.Optional[string]            `json:"name_zh"`
	DescEn           workbench.Optional[string]            `json:"desc_en"`
	DescZh           workbench.Optional[string]            `json:"desc_zh"`
	DisplayNameLocs  workbench.Optional[map[string]string] `json:"display_name_locs"`
	DescriptionLocs  workbench.Optional[map[string]string] `json:"description_locs"`
	Modifier         workbench.Optional[map[string]any]    `json:"modifier"`
	AIWillDo         workbench.Optional[string]            `json:"ai_will_do"`
	Random           workbench.Optional[bool]              `json:"random"`
	ExtraFields      workbench.Optional[map[string]any]    `json:"extra_fields"`
	RawExtras        workbench.Optional[map[string]any]    `json:"raw_extras"`
	TargetedModifier workbench.Optional[[]map[string]any]  `json:"targeted_modifier"`
}

func updateCustomTrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tid, ok := pathID(r, "tid")
	if !ok {
		return errTraitMissing
	}
	var patch CustomTraitPatch
	if err := decodeBody(w, r, 2<<20, &patch); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid trait payload")
	}
	var updated project.Row
	err := p.Write([]string{traitTable}, func(tx *project.Tx) error {
		current := tx.Find(traitTable, tid)
		if current == nil {
			return errTraitMissing
		}
		if patch.TraitType.Set && (patch.TraitType.Value == nil || !validTraitTypes[*patch.TraitType.Value]) {
			return detail(http.StatusBadRequest, traitTypesDetail)
		}
		if patch.TraitID.Set {
			cleaned := ""
			if patch.TraitID.Value != nil {
				cleaned = strings.TrimSpace(*patch.TraitID.Value)
			}
			if cleaned == "" {
				return detail(http.StatusBadRequest, "trait_id cannot be empty")
			}
			patch.TraitID.Value = &cleaned
		}
		if patch.CountryTag.Set && patch.CountryTag.Value != nil {
			v := upperTrim(*patch.CountryTag.Value)
			patch.CountryTag.Value = &v
		}
		changes := map[string]any{}
		set(changes, "trait_type", patch.TraitType)
		set(changes, "trait_id", patch.TraitID)
		set(changes, "country_tag", patch.CountryTag)
		set(changes, "name_en", patch.NameEn)
		set(changes, "name_zh", patch.NameZh)
		set(changes, "desc_en", patch.DescEn)
		set(changes, "desc_zh", patch.DescZh)
		set(changes, "display_name_locs", patch.DisplayNameLocs)
		set(changes, "description_locs", patch.DescriptionLocs)
		set(changes, "modifier", patch.Modifier)
		set(changes, "ai_will_do", patch.AIWillDo)
		set(changes, "random", patch.Random)
		set(changes, "extra_fields", patch.ExtraFields)
		set(changes, "raw_extras", patch.RawExtras)
		set(changes, "targeted_modifier", patch.TargetedModifier)
		if len(changes) == 0 {
			updated = current
			return nil
		}
		if notNullViolation(traitTable, changes) {
			return errTraitUnavailable
		}
		traitType, traitID := str(current["trait_type"]), str(current["trait_id"])
		if v, ok := changes["trait_type"].(string); ok {
			traitType = v
		}
		if v, ok := changes["trait_id"].(string); ok {
			traitID = v
		}
		for _, row := range tx.Rows(traitTable) {
			if project.Int64(row["id"]) != tid && str(row["trait_type"]) == traitType && str(row["trait_id"]) == traitID {
				return errTraitUnavailable
			}
		}
		var err error
		updated, err = tx.Update(traitTable, tid, func(row project.Row) {
			for column, value := range changes {
				row[column] = value
			}
		})
		return err
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, traitFromRow(updated))
	return nil
}

func deleteCustomTrait(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tid, ok := pathID(r, "tid")
	if !ok {
		return errTraitMissing
	}
	err := p.Write([]string{traitTable}, func(tx *project.Tx) error {
		if tx.Delete(traitTable, func(row project.Row) bool { return project.Int64(row["id"]) == tid }) == 0 {
			return errTraitMissing
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func customTraitIconFilename(traitType, traitID string) string {
	return "customtrait_" + traitType + "_" + emitmisc.Slugify(traitID) + ".png"
}

func customTraitSpriteName(traitType, traitID string) string {
	switch traitType {
	case "scientist":
		return "GFX_" + traitID
	case "unit_leader":
		return "GFX_trait_" + traitID
	default:
		return ""
	}
}

func uploadTraitIcon(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tid, ok := pathID(r, "tid")
	if !ok {
		return errAssetTargetMissing
	}
	row, err := findRow(p, traitTable, tid)
	if err != nil {
		return err
	}
	if row == nil {
		return errAssetTargetMissing
	}
	traitType, traitID := str(row["trait_type"]), str(row["trait_id"])
	if traitType != "unit_leader" && traitType != "scientist" {
		return detail(http.StatusBadRequest, "Icon upload is only supported for unit_leader and scientist traits (country_leader uses a numeric advisor sprite index).")
	}
	if strings.TrimSpace(p.Blobs().ProjectUUID()) == "" {
		return detail(http.StatusServiceUnavailable, "asset metadata unavailable")
	}
	raw, filename, crop, err := readUpload(w, r)
	if err != nil {
		return err
	}
	spec := assetSpec{"custom_trait_icon", "trait", customTraitIconFilename(traitType, traitID), customTraitSpriteName(traitType, traitID)}
	assets, err := storeRenditions(p, tid, []assetSpec{spec}, raw, filename, crop)
	if err != nil {
		return err
	}
	err = p.Write([]string{"entity_obs_paths", traitTable}, func(tx *project.Tx) error {
		current := tx.Find(traitTable, tid)
		if current == nil || (str(current["trait_type"]) != "unit_leader" && str(current["trait_type"]) != "scientist") {
			return detail(http.StatusServiceUnavailable, "asset metadata commit failed; retry is safe")
		}
		upsertEntityObsPath(tx, spec.entityType, tid, assets[0].key)
		_, err := tx.Update(traitTable, tid, func(project.Row) {})
		return err
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"filename": spec.filename, "warnings": []string{}, "sprite": spec.gfxName,
		"preview_url": "/api/go/projects/" + strconv.FormatInt(p.ID(), 10) + "/custom-traits/" + strconv.FormatInt(tid, 10) + "/icon",
	})
	return nil
}

func traitIcon(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tid, ok := pathID(r, "tid")
	if !ok {
		return errAssetTargetMissing
	}
	row, err := findRow(p, traitTable, tid)
	if err != nil {
		return err
	}
	if row == nil {
		return errAssetNotUploaded
	}
	paths, err := p.Rows("entity_obs_paths")
	if err != nil {
		return err
	}
	key := entityObsKey(paths, "custom_trait_icon", tid)
	if key == "" {
		return errAssetNotUploaded
	}
	return serveBlob(p, w, key)
}
