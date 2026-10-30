package characters

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	emitcharacters "haganeworks/internal/hoi4/emit/characters"
	"haganeworks/internal/hoi4/emit/locout"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	previewemit "haganeworks/internal/hoi4/emit/preview"
	"haganeworks/internal/project"
)

func previewAssetKeys(p *project.Project, characterID int64) (map[string]string, error) {
	rows, err := p.Rows("entity_obs_paths")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, row := range rows {
		kind := str(row["entity_type"])
		if project.Int64(row["entity_id"]) != characterID || (kind != "character_portrait" && kind != "character_advisor") {
			continue
		}
		out[kind] = str(row["obs_key"])
	}
	return out, nil
}

func assetResolves(p *project.Project, key string) bool {
	return strings.TrimSpace(key) != "" && p.Blobs().Exists(key)
}

func imageGates(p *project.Project, char Character) (portraitOK, advisorOK bool, err error) {
	keys, err := previewAssetKeys(p, char.ID)
	if err != nil {
		return false, false, err
	}
	portraitOK = assetResolves(p, keys["character_portrait"])
	needsAdvisor := char.RoleType == "advisor" || char.HasAdvisorRole
	advisorOK = needsAdvisor && assetResolves(p, keys["character_advisor"])
	return portraitOK, advisorOK, nil
}

func loadCharacter(p *project.Project, r *http.Request) (Character, error) {
	cid, ok := pathID(r, "cid")
	if !ok {
		return Character{}, errCharacterMissing
	}
	row, err := findRow(p, table, cid)
	if err != nil {
		return Character{}, err
	}
	if row == nil {
		return Character{}, errCharacterMissing
	}
	return characterFromRow(row), nil
}

func characterPreview(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	char, err := loadCharacter(p, r)
	if err != nil {
		return err
	}
	portraitOK, advisorOK, err := imageGates(p, char)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"char": char, "country_tag": projectCountryTag(p),
		"has_portrait_image": portraitOK, "has_advisor_image": advisorOK,
		"portrait_path_override": "", "advisor_path_override": "",
	})
	if err != nil {
		return err
	}
	script, err := previewemit.EmitCharacterJSON(body)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]string{"script": script})
	return nil
}

type previewFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func buildPreviewFiles(char Character, projectTag string, exportLanguages []string, portraitOK, advisorOK bool) (map[string]previewFile, error) {
	ctag := char.CountryTag
	if ctag == "" {
		ctag = projectTag
	}
	ctag = strings.ToUpper(ctag)
	defaultScriptPath := "common/characters/" + strings.ToUpper(projectTag) + "_" + ctag + ".txt"

	charJSON, err := json.Marshal(char)
	if err != nil {
		return nil, err
	}
	miscChar, err := emitmisc.DecodeEntityJSON(charJSON)
	if err != nil {
		return nil, err
	}
	groups := emitmisc.GroupBySourceFile(
		[]*emitmisc.NS{miscChar},
		func(*emitmisc.NS) string { return defaultScriptPath },
		"common/characters",
	)
	if len(groups.Keys) != 1 {
		return nil, errors.New("character source path did not resolve")
	}

	var script string
	if char.IsReplacement {
		script = "# " + char.Tag + " is a vanilla replacement character.\n" +
			"# Trait edits emit via on_actions trait_overlay event,\n" +
			"# not as a `common/characters/` block. The advisor / loc /\n" +
			"# gfx files below still ship in the zip.\n"
	} else {
		body, err := json.Marshal(map[string]any{
			"char": char, "country_tag": ctag,
			"has_portrait_image": portraitOK, "has_advisor_image": advisorOK,
			"portrait_path_override": "", "advisor_path_override": "",
		})
		if err != nil {
			return nil, err
		}
		script, err = previewemit.EmitCharacterJSON(body)
		if err != nil {
			return nil, err
		}
	}

	locValue, err := locout.ParseJSON(charJSON)
	if err != nil || locValue.Kind != locout.KObj {
		if err == nil {
			err = errors.New("character localisation row is not an object")
		}
		return nil, err
	}
	files := map[string]previewFile{
		"script": {Path: groups.Keys[0], Content: script},
	}
	for _, lang := range locout.EnsureEnFirstStrings(exportLanguages) {
		hoi4Lang := locout.LangCodeToHOI4[lang]
		files["loc_"+lang] = previewFile{
			Path: "localisation/" + hoi4Lang + "/replace/" +
				ctag + "_characters_l_" + hoi4Lang + ".yml",
			Content: locout.GenerateCharacterLocalisation([]*locout.Obj{locValue.Obj}, ctag, lang),
		}
	}
	gfxChar := &emitcharacters.Character{ID: int(char.ID), Tag: char.Tag}
	files["gfx"] = previewFile{
		Path: "interface/" + ctag + "_characters.gfx",
		Content: emitcharacters.GenerateGFXSprites(
			[]*emitcharacters.Character{gfxChar}, ctag,
			map[int][2]bool{gfxChar.ID: {portraitOK, advisorOK}},
		),
	}
	return files, nil
}

func characterPreviewFiles(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	char, err := loadCharacter(p, r)
	if err != nil {
		return err
	}
	portraitOK, advisorOK, err := imageGates(p, char)
	if err != nil {
		return err
	}
	languages := stringList(p.Record()["export_languages"])
	files, err := buildPreviewFiles(char, projectCountryTag(p), languages, portraitOK, advisorOK)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"files": files})
	return nil
}
