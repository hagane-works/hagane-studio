package countries

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/project"
)

var (
	errCosmeticMissing  = detail(http.StatusNotFound, "cosmetic tag not found in this project")
	errCosmeticConflict = detail(http.StatusConflict, "cosmetic tag changed; reload before saving")
)

func cosmeticOwnerTables(ctx context.Context) (*worldout.Tables, error) {
	return worldout.LoadVanillaTables(ctx)
}

func cosmeticValidation(err error) error {
	return detail(http.StatusUnprocessableEntity, err.Error())
}

type cosmeticTagView struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Source    string `json:"source"`
	Revision  int64  `json:"revision"`
	cosmetic.Definition
	ReferencedBy []string `json:"referenced_by"`
}

type cosmeticRawOverride struct {
	FileID int64  `json:"file_id"`
	Path   string `json:"path"`
	Locale string `json:"locale"`
	Suffix string `json:"suffix"`
	Value  string `json:"value"`
}

type cosmeticNameOp struct {
	Locale string  `json:"locale"`
	Suffix string  `json:"suffix"`
	Value  *string `json:"value"`
}

func (op *cosmeticNameOp) UnmarshalJSON(raw []byte) error {
	type plain cosmeticNameOp
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for key := range fields {
		if key != "locale" && key != "suffix" && key != "value" {
			return errors.New("unknown name operation field: " + key)
		}
	}
	for _, key := range []string{"locale", "suffix", "value"} {
		if _, ok := fields[key]; !ok {
			return errors.New("name operation requires locale, suffix and value")
		}
	}
	for _, key := range []string{"locale", "suffix"} {
		if bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
			return errors.New("name operation locale and suffix must be strings")
		}
	}
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*op = cosmeticNameOp(value)
	return nil
}

type cosmeticTagPatch struct {
	ExpectedRevision int64            `json:"expected_revision"`
	NameOps          []cosmeticNameOp `json:"name_ops"`

	Color   json.RawMessage `json:"color"`
	ColorUI json.RawMessage `json:"color_ui"`
}

func decodeRGB(raw json.RawMessage) (*cosmetic.RGB, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var channels []json.RawMessage
	if err := json.Unmarshal(raw, &channels); err != nil || len(channels) != 3 {
		return nil, detail(http.StatusUnprocessableEntity, "color must be an RGB array of three integers")
	}
	var rgb cosmetic.RGB
	for i, c := range channels {
		if bytes.Equal(bytes.TrimSpace(c), []byte("null")) {
			return nil, detail(http.StatusUnprocessableEntity, "RGB channels cannot be null")
		}
		if err := json.Unmarshal(c, &rgb[i]); err != nil || rgb[i] < 0 || rgb[i] > 255 {
			return nil, detail(http.StatusUnprocessableEntity, "RGB channels must be integers between 0 and 255")
		}
	}
	return &rgb, nil
}

func (in cosmeticTagPatch) validate() error {
	if in.ExpectedRevision < 1 {
		return detail(http.StatusUnprocessableEntity, "expected_revision must be positive")
	}
	if len(in.NameOps) > 150 {
		return detail(http.StatusUnprocessableEntity, "at most 150 name operations are allowed")
	}
	seen := map[string]bool{}
	for _, op := range in.NameOps {
		value := ""
		if op.Value != nil {
			value = *op.Value
		}
		if err := cosmetic.ValidateName(op.Locale, op.Suffix, value); err != nil {
			return cosmeticValidation(err)
		}
		key := op.Locale + "\x00" + op.Suffix
		if seen[key] {
			return detail(http.StatusUnprocessableEntity, "duplicate name operation")
		}
		seen[key] = true
	}
	if _, err := decodeRGB(in.Color); err != nil {
		return err
	}
	_, err := decodeRGB(in.ColorUI)
	return err
}

func decodeCosmeticRequest(r *http.Request, out any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return detail(http.StatusUnprocessableEntity, "Invalid cosmetic tag payload")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return detail(http.StatusUnprocessableEntity, "Expected one JSON payload")
	}
	return nil
}

func rgbOf(value any) *cosmetic.RGB {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var rgb cosmetic.RGB
	if json.Unmarshal(raw, &rgb) != nil {
		return nil
	}
	return &rgb
}

func rgbValue(rgb *cosmetic.RGB) any {
	if rgb == nil {
		return nil
	}
	return []any{int64(rgb[0]), int64(rgb[1]), int64(rgb[2])}
}

func cosmeticView(row project.Row, names, history []project.Row) cosmeticTagView {
	id := project.Int64(row["id"])
	v := cosmeticTagView{
		ID: id, ProjectID: project.Int64(row["project_id"]),
		Source: str(row["source"]), Revision: project.Int64(row["revision"]),
	}
	v.Tag = str(row["tag"])
	v.Color, v.ColorUI = rgbOf(row["color"]), rgbOf(row["color_ui"])
	v.ImportedColor, v.ImportedColorUI = rgbOf(row["imported_color"]), rgbOf(row["imported_color_ui"])
	v.NameLocs, v.ImportedNameLocs = cosmetic.Names{}, cosmetic.Names{}
	for _, name := range names {
		if project.Int64(name["cosmetic_tag_id"]) != id {
			continue
		}
		target := v.NameLocs
		switch str(name["source"]) {
		case "created":
		case "imported":
			target = v.ImportedNameLocs
		default:
			continue
		}
		locale := str(name["locale"])
		if target[locale] == nil {
			target[locale] = map[string]string{}
		}
		target[locale][str(name["suffix"])] = str(name["value"])
	}
	seen := map[string]bool{}
	v.ReferencedBy = []string{}
	for _, h := range history {
		tag := str(h["country_tag"])
		if str(h["set_cosmetic_tag"]) == v.Tag && !seen[tag] {
			seen[tag] = true
			v.ReferencedBy = append(v.ReferencedBy, tag)
		}
	}
	sort.Strings(v.ReferencedBy)
	return v
}

func cosmeticData(p *project.Project) (tags, names, history []project.Row, err error) {
	if tags, err = p.Rows("project_cosmetic_tags"); err != nil {
		return
	}
	if names, err = p.Rows("project_cosmetic_tag_names"); err != nil {
		return
	}
	history, err = p.Rows("history_files")
	return
}

func findCosmeticView(p *project.Project, tag string) (cosmeticTagView, error) {
	tags, names, history, err := cosmeticData(p)
	if err != nil {
		return cosmeticTagView{}, err
	}
	for _, row := range tags {
		if str(row["tag"]) == tag {
			return cosmeticView(row, names, history), nil
		}
	}
	return cosmeticTagView{}, errCosmeticMissing
}

func cosmeticRawOverrides(p *project.Project, tag string) ([]cosmeticRawOverride, error) {
	files, err := p.Rows("project_raw_files")
	if err != nil {
		return nil, err
	}
	sort.SliceStable(files, func(i, j int) bool { return str(files[i]["dest_path"]) < str(files[j]["dest_path"]) })
	out := []cosmeticRawOverride{}
	for _, file := range files {
		dest := str(file["dest_path"])
		path := strings.ToLower(strings.ReplaceAll(dest, "\\", "/"))
		if file["is_binary"] == true || str(file["raw_kind"]) == "tombstone" || !strings.HasPrefix(path, "localisation/") || !strings.HasSuffix(path, ".yml") {
			continue
		}
		locale, entries, _ := loc.ParseLocText(str(file["content"]))
		if cosmetic.ValidateLocale(locale) != nil {
			continue
		}
		for _, suffix := range cosmetic.Suffixes() {
			key := tag
			if suffix != "" {
				key += "_" + suffix
			}
			if value, ok := entries[key]; ok {
				out = append(out, cosmeticRawOverride{project.Int64(file["id"]), dest, locale, suffix, value})
			}
		}
	}
	return out, nil
}

func writeCosmeticTag(p *project.Project, w http.ResponseWriter, status int, item cosmeticTagView) error {
	overrides, err := cosmeticRawOverrides(p, item.Tag)
	if err != nil {
		return err
	}
	apiutil.JSON(w, status, struct {
		cosmeticTagView
		RawOverrides []cosmeticRawOverride `json:"raw_overrides"`
	}{item, overrides})
	return nil
}

func listCosmeticHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tags, names, history, err := cosmeticData(p)
	if err != nil {
		return err
	}
	items := make([]cosmeticTagView, 0, len(tags))
	for _, row := range tags {
		items = append(items, cosmeticView(row, names, history))
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Tag < items[j].Tag })
	apiutil.JSON(w, http.StatusOK, map[string]any{"cosmetic_tags": items, "total": len(items)})
	return nil
}

func getCosmeticHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tag := r.PathValue("tag")
	if err := cosmetic.ValidateTag(tag); err != nil {
		return cosmeticValidation(err)
	}
	item, err := findCosmeticView(p, tag)
	if err != nil {
		return err
	}
	return writeCosmeticTag(p, w, http.StatusOK, item)
}

func createCosmeticHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Tag string `json:"tag"`
	}
	if err := decodeCosmeticRequest(r, &in); err != nil {
		return err
	}
	if err := cosmetic.ValidateTag(in.Tag); err != nil {
		return cosmeticValidation(err)
	}

	err := p.Write([]string{"project_cosmetic_tags", "project_cosmetic_tag_names", "project_loc_entries"}, func(tx *project.Tx) error {
		for _, row := range tx.Rows("project_cosmetic_tags") {
			if str(row["tag"]) == in.Tag {
				return nil
			}
		}
		created := tx.Insert("project_cosmetic_tags", project.Row{"tag": in.Tag, "source": "created", "revision": int64(1)})
		refreshCosmeticImportedNames(tx, project.Int64(created["id"]), in.Tag)
		return nil
	})
	if err != nil {
		return err
	}
	item, err := findCosmeticView(p, in.Tag)
	if err != nil {
		return err
	}
	return writeCosmeticTag(p, w, http.StatusCreated, item)
}

func patchCosmeticHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tag := r.PathValue("tag")
	if err := cosmetic.ValidateTag(tag); err != nil {
		return cosmeticValidation(err)
	}
	var in cosmeticTagPatch
	if err := decodeCosmeticRequest(r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	err := p.Write([]string{"project_cosmetic_tags", "project_cosmetic_tag_names"}, func(tx *project.Tx) error {
		var row project.Row
		for _, candidate := range tx.Rows("project_cosmetic_tags") {
			if str(candidate["tag"]) == tag {
				row = candidate
			}
		}
		if row == nil {
			return errCosmeticMissing
		}
		id := project.Int64(row["id"])
		if project.Int64(row["revision"]) != in.ExpectedRevision {
			return errCosmeticConflict
		}
		for _, op := range in.NameOps {
			matches := func(name project.Row) bool {
				return project.Int64(name["cosmetic_tag_id"]) == id && str(name["locale"]) == op.Locale &&
					str(name["suffix"]) == op.Suffix && str(name["source"]) == "created"
			}
			if op.Value == nil || strings.TrimSpace(*op.Value) == "" {
				tx.Delete("project_cosmetic_tag_names", matches)
				continue
			}
			found := false
			for _, name := range tx.Rows("project_cosmetic_tag_names") {
				if matches(name) {
					found = true
					value := *op.Value
					if _, err := tx.Update("project_cosmetic_tag_names", project.Int64(name["id"]), func(next project.Row) { next["value"] = value }); err != nil {
						return err
					}
					break
				}
			}
			if !found {
				tx.Insert("project_cosmetic_tag_names", project.Row{"cosmetic_tag_id": id, "locale": op.Locale, "suffix": op.Suffix, "value": *op.Value, "source": "created"})
			}
		}
		changed := len(in.NameOps) > 0 || len(in.Color) > 0 || len(in.ColorUI) > 0
		if !changed {
			return nil
		}
		color, _ := decodeRGB(in.Color)
		colorUI, _ := decodeRGB(in.ColorUI)
		_, err := tx.Update("project_cosmetic_tags", id, func(next project.Row) {
			if len(in.Color) > 0 {
				next["color"] = rgbValue(color)
			}
			if len(in.ColorUI) > 0 {
				next["color_ui"] = rgbValue(colorUI)
			}
			next["revision"] = project.Int64(next["revision"]) + 1
		})
		return err
	})
	if err != nil {
		return err
	}
	item, err := findCosmeticView(p, tag)
	if err != nil {
		return err
	}
	return writeCosmeticTag(p, w, http.StatusOK, item)
}

func resolveCosmeticHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	tag := r.PathValue("tag")
	if err := cosmetic.ValidateTag(tag); err != nil {
		return cosmeticValidation(err)
	}
	var in struct {
		Locale   string          `json:"locale"`
		NameLocs cosmetic.Names  `json:"name_locs"`
		Color    json.RawMessage `json:"color"`
		ColorUI  json.RawMessage `json:"color_ui"`
	}
	if err := decodeCosmeticRequest(r, &in); err != nil {
		return err
	}
	if err := cosmetic.ValidateLocale(in.Locale); err != nil {
		return cosmeticValidation(err)
	}
	for locale, names := range in.NameLocs {
		for suffix, value := range names {
			if err := cosmetic.ValidateName(locale, suffix, value); err != nil {
				return cosmeticValidation(err)
			}
		}
	}
	color, err := decodeRGB(in.Color)
	if err != nil {
		return err
	}
	ui, err := decodeRGB(in.ColorUI)
	if err != nil {
		return err
	}
	v, err := findCosmeticView(p, tag)
	if err != nil {
		return err
	}
	v.NameLocs, v.Color, v.ColorUI = in.NameLocs, color, ui
	owner, err := cosmeticOwner(p, tag)
	if err != nil {
		return err
	}
	tables, err := cosmeticOwnerTables(r.Context())
	if err != nil {
		return err
	}
	ownerColor, ownerUI := worldout.CosmeticOwnerColors(owner, tables.CountryCultures)
	result := struct {
		Names   map[string]cosmetic.ResolvedName `json:"names"`
		Color   cosmetic.RGB                     `json:"color"`
		ColorUI cosmetic.RGB                     `json:"color_ui"`
	}{Names: cosmetic.ResolveNames(v.Definition, in.Locale)}
	result.Color, result.ColorUI = cosmetic.ResolveColors(v.Definition, ownerColor, ownerUI)
	apiutil.JSON(w, http.StatusOK, result)
	return nil
}

func cosmeticOwner(p *project.Project, tag string) (*worldout.OMap, error) {
	history, err := p.Rows("history_files")
	if err != nil {
		return nil, err
	}
	for _, h := range history {
		if str(h["set_cosmetic_tag"]) != tag {
			continue
		}
		countries, err := p.Rows("countries")
		if err != nil {
			return nil, err
		}
		for _, c := range countries {
			if str(c["tag"]) == str(h["country_tag"]) {
				raw, err := json.Marshal(c)
				if err != nil {
					return nil, err
				}
				owner := worldout.NewOMap()
				if err := owner.UnmarshalJSON(raw); err != nil {
					return nil, err
				}
				return owner, nil
			}
		}
		return nil, nil
	}
	return nil, nil
}

type cosmeticLocCandidate struct {
	value                                        string
	replacePriority, sourceOrder, entryOrder, id int64
}

func (a cosmeticLocCandidate) ranksAbove(b cosmeticLocCandidate) bool {
	switch {
	case a.replacePriority != b.replacePriority:
		return a.replacePriority > b.replacePriority
	case a.sourceOrder != b.sourceOrder:
		return a.sourceOrder > b.sourceOrder
	case a.entryOrder != b.entryOrder:
		return a.entryOrder > b.entryOrder
	}
	return a.id > b.id
}

func refreshCosmeticImportedNames(tx *project.Tx, id int64, tag string) {
	locales := map[string]bool{}
	for _, locale := range cosmetic.Locales() {
		locales[locale] = true
	}
	keys := map[string]string{}
	for _, suffix := range cosmetic.Suffixes() {
		key := tag
		if suffix != "" {
			key += "_" + suffix
		}
		keys[key] = suffix
	}
	type nameKey struct{ locale, suffix string }
	best := map[nameKey]cosmeticLocCandidate{}
	for _, row := range tx.Rows("project_loc_entries") {
		locale := str(row["locale"])
		suffix, known := keys[str(row["key"])]
		if !known || !locales[locale] {
			continue
		}
		if _, ok := exportengine.SafePreservedLocSourceFile(str(row["source_file"])); !ok {
			continue
		}
		candidate := cosmeticLocCandidate{
			value: str(row["value"]), replacePriority: project.Int64(row["replace_priority"]),
			sourceOrder: project.Int64(row["source_order"]), entryOrder: project.Int64(row["entry_order"]),
			id: project.Int64(row["id"]),
		}
		k := nameKey{locale, suffix}
		if current, ok := best[k]; !ok || candidate.ranksAbove(current) {
			best[k] = candidate
		}
	}
	old := map[nameKey]string{}
	for _, row := range tx.Rows("project_cosmetic_tag_names") {
		if project.Int64(row["cosmetic_tag_id"]) == id && str(row["source"]) == "imported" {
			old[nameKey{str(row["locale"]), str(row["suffix"])}] = str(row["value"])
		}
	}
	same := len(old) == len(best)
	for k, candidate := range best {
		if value, ok := old[k]; !ok || value != candidate.value {
			same = false
		}
	}
	if same {
		return
	}
	_, _ = tx.Update("project_cosmetic_tags", id, func(next project.Row) {
		next["revision"] = project.Int64(next["revision"]) + 1
	})
	tx.Delete("project_cosmetic_tag_names", func(row project.Row) bool {
		return project.Int64(row["cosmetic_tag_id"]) == id && str(row["source"]) == "imported"
	})
	ordered := make([]nameKey, 0, len(best))
	for k := range best {
		ordered = append(ordered, k)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].locale != ordered[j].locale {
			return ordered[i].locale < ordered[j].locale
		}
		return ordered[i].suffix < ordered[j].suffix
	})
	for _, k := range ordered {
		tx.Insert("project_cosmetic_tag_names", project.Row{
			"cosmetic_tag_id": id, "locale": k.locale, "suffix": k.suffix, "value": best[k].value, "source": "imported",
		})
	}
}
