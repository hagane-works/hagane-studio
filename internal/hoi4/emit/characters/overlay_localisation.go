package characters

import (
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
)

const bomUTF8 = "\ufeff"

func overlayYMLLine(key, text string) string {
	safeKey := key
	if !isLegalLocKey(key) {
		safeKey = sanitizeLocKey(key)
	}
	return " " + safeKey + `: "` + locout.YmlEscapeValue(text) + `"`
}

func generateTraitOverlayLocalisation(namespace string, lang LocLanguage) string {
	title := "Trait Overlay (hidden)"
	desc := "Internally fired at game start to apply custom country leader, unit leader, and scientist traits to existing vanilla characters without replacing the character file."
	if lang.Code == "zh" {
		title = "特质覆盖(隐藏)"
		desc = "游戏启动时内部触发,用于将自定义领袖、将领、科学家特质赋予现有原版角色,无需替换角色文件。"
	}
	return strings.Join([]string{
		lang.Suffix + ":",
		overlayYMLLine(namespace+".1.t", title),
		overlayYMLLine(namespace+".1.d", desc),
	}, "\n")
}

func recordTraitOverlay(res *Result, ctag string, update func(*TraitOverlayArtifacts)) {
	artifacts := res.TraitOverlays[ctag]
	update(&artifacts)
	res.TraitOverlays[ctag] = artifacts
}

func emitTraitOverlayLocalisation(res *Result, ctag, namespace string, langs []LocLanguage) {
	for _, lang := range langs {
		if strings.TrimSpace(lang.Code) == "" || strings.TrimSpace(lang.Subdir) == "" || strings.TrimSpace(lang.Suffix) == "" {
			continue
		}
		path := "localisation/" + strings.Trim(lang.Subdir, "/") + "/" + ctag + "_trait_overlay_" + lang.Suffix + ".yml"
		res.Texts[path] = bomUTF8 + generateTraitOverlayLocalisation(namespace, lang)
		recordTraitOverlay(res, ctag, func(artifacts *TraitOverlayArtifacts) {
			artifacts.EventLocalisationPaths = append(artifacts.EventLocalisationPaths, path)
		})
	}
}

func emitReplacementLocalisation(in *Input, res *Result, primaryUpper string) {
	type group struct {
		ctag  string
		chars []*Character
	}
	var groups []group
	byCountry := map[string]int{}
	add := func(c *Character) {
		ctag := ctagOf(c, primaryUpper)
		if idx, ok := byCountry[ctag]; ok {
			groups[idx].chars = append(groups[idx].chars, c)
			return
		}
		byCountry[ctag] = len(groups)
		groups = append(groups, group{ctag: ctag, chars: []*Character{c}})
	}
	for _, chars := range [][]*Character{
		in.ReplacementCountryLeaders,
		in.ReplacementUnitLeaders,
		in.ReplacementScientists,
	} {
		for _, c := range chars {
			add(c)
		}
	}

	for _, group := range groups {
		for _, lang := range in.ExportLanguages {
			if strings.TrimSpace(lang.Code) == "" || strings.TrimSpace(lang.Subdir) == "" || strings.TrimSpace(lang.Suffix) == "" {
				continue
			}
			lines := []string{lang.Suffix + ":"}
			for _, c := range group.chars {
				name := resolveDisplayName(c, lang.Code)
				if name == "" || isPlaceholderDisplayName(name, c.Tag) {
					continue
				}
				lines = append(lines, overlayYMLLine(characterLocKey(c), name))
			}
			if len(lines) == 1 {
				continue
			}
			path := "localisation/" + strings.Trim(lang.Subdir, "/") + "/" + group.ctag + "_characters_replacement_overrides_" + lang.Suffix + ".yml"
			res.Texts[path] = bomUTF8 + strings.Join(lines, "\n")
			recordTraitOverlay(res, group.ctag, func(artifacts *TraitOverlayArtifacts) {
				artifacts.ReplacementLocalisationPaths = append(artifacts.ReplacementLocalisationPaths, path)
			})
		}
	}
}
