package emitscripts

import (
	"regexp"
	"strconv"
	"strings"
)

type Character struct {
	ID                 int
	Tag                string
	Traits             []string
	RoleType           string
	HasAdvisorRole     bool
	IsPortraitModified bool
	Source             string
	DisplayNameLocs    *Dict
	NameEn             string
	NameZh             string
	Name               string
	DescriptionLocs    *Dict
	DescEn             string
	DescZh             string
}

func CountryLeaderTraits(customTraits []*CustomTrait) []*CustomTrait {
	var out []*CustomTrait
	for _, t := range customTraits {
		if t.TraitType == "country_leader" {
			out = append(out, t)
		}
	}
	return out
}

func UnitLeaderTraits(customTraits []*CustomTrait) []*CustomTrait {
	var out []*CustomTrait
	for _, t := range customTraits {
		if t.TraitType == "unit_leader" {
			out = append(out, t)
		}
	}
	return out
}

func ScientistTraits(customTraits []*CustomTrait) []*CustomTrait {
	var out []*CustomTrait
	for _, t := range customTraits {
		if t.TraitType == "scientist" {
			out = append(out, t)
		}
	}
	return out
}

var UnitLeaderRoles = map[string]bool{
	"general": true, "field_marshal": true, "admiral": true,
}

type TraitPair struct {
	CharTag  string
	TraitIDs []string
}

func BuildTraitOverlayPairs(importedCountryLeaders []*Character, customCountryLeaderTraitIDs map[string]bool) []TraitPair {
	var pairs []TraitPair
	for _, c := range importedCountryLeaders {
		var attachedCustom []string
		for _, t := range c.Traits {
			if customCountryLeaderTraitIDs[t] {
				attachedCustom = append(attachedCustom, t)
			}
		}
		if len(attachedCustom) > 0 {
			pairs = append(pairs, TraitPair{c.Tag, attachedCustom})
		}
	}
	return pairs
}

func BuildPortraitOverrides(
	replacementChars []*Character,
	ctag string,
	obsHas func(kind string, charID int) bool,
	genExists func(filename string) bool,
) *Dict {
	overrides := NewDict()
	for _, c := range replacementChars {
		if !c.IsPortraitModified {
			continue
		}
		slug := ExtractSlug(c.Tag, ctag)
		portraitPng := PortraitFilename(ctag, slug)
		advisorPng := AdvisorIconFilename(ctag, slug)
		hasPortrait := obsHas("character_portrait", c.ID) || genExists(portraitPng)
		hasAdvisor := obsHas("character_advisor", c.ID) || genExists(advisorPng)
		if !(hasPortrait || hasAdvisor) {
			continue
		}
		portraitSprite := PortraitSpriteName(ctag, slug)
		advisorSprite := AdvisorSpriteName(ctag, slug)
		isMilitary := UnitLeaderRoles[c.RoleType]
		needsArmyLarge := isMilitary || c.RoleType == "scientist"
		needsArmySmall := needsArmyLarge || c.RoleType == "advisor" || c.HasAdvisorRole
		charLayers := NewDict()
		if hasPortrait {
			civ := NewDict()
			civ.Set("large", portraitSprite)
			charLayers.Set("civilian", civ)
			if needsArmyLarge {
				army, ok := charLayers.Get("army").(*Dict)
				if !ok {
					army = NewDict()
					charLayers.Set("army", army)
				}
				army.Set("large", portraitSprite)
			}
		}
		if hasAdvisor && needsArmySmall {
			army, ok := charLayers.Get("army").(*Dict)
			if !ok {
				army = NewDict()
				charLayers.Set("army", army)
			}
			army.Set("small", advisorSprite)
		}
		if charLayers.Truthy() {
			overrides.Set(c.Tag, charLayers)
		}
	}
	return overrides
}

func ResolveTraitOverlayNamespace(ctag string, existingNamespaces map[string]bool) string {
	base := pyLower(ctag) + "_trait_overlay"
	if !existingNamespaces[base] {
		return base
	}
	n := 2
	for existingNamespaces[base+"_"+strconv.Itoa(n)] {
		n++
	}
	return base + "_" + strconv.Itoa(n)
}

type CLReplacement struct {
	CharTag       string
	PictureSprite string
	Expire        string
	Ideology      string
	Traits        []string
	Char          *Character
	LocEmitted    bool
}

func CLDescLocKeyWillBeEmitted(cl *CLReplacement) bool {
	char := cl.Char
	if char == nil {
		return false
	}
	if !cl.LocEmitted {
		return false
	}
	tag := char.Tag
	if ResolveDescriptionAnyLang(char) != "" {
		return true
	}
	_, truthy := ResolveMeaningfulDisplayName(char, "en", tag)
	return truthy
}

type EffectGroup struct {
	Name  string
	Pairs []TraitPair
}

func GenerateTraitOverlayEvent(
	namespace string,
	effectGroups []EffectGroup,
	portraitOverrides *Dict,
	countryLeaderReplacements []*CLReplacement,
	countryTag string,
) string {
	type effectTID struct{ Effect, TID string }
	perChar := map[string][]effectTID{}
	var charOrder []string
	for _, group := range effectGroups {
		for _, pair := range group.Pairs {
			if len(pair.TraitIDs) == 0 {
				continue
			}
			if _, ok := perChar[pair.CharTag]; !ok {
				perChar[pair.CharTag] = []effectTID{}
				charOrder = append(charOrder, pair.CharTag)
			}
			for _, tid := range pair.TraitIDs {
				perChar[pair.CharTag] = append(perChar[pair.CharTag], effectTID{group.Name, tid})
			}
		}
	}

	if portraitOverrides == nil {
		portraitOverrides = NewDict()
	}
	for _, charTag := range portraitOverrides.Keys() {
		if _, ok := perChar[charTag]; !ok {
			perChar[charTag] = []effectTID{}
			charOrder = append(charOrder, charTag)
		}
	}

	if len(perChar) == 0 && len(countryLeaderReplacements) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "add_namespace = "+namespace)
	lines = append(lines, "")
	lines = append(lines, "country_event = {")
	lines = append(lines, "\tid = "+namespace+".1")
	lines = append(lines, "\ttitle = "+namespace+".1.t")
	lines = append(lines, "\tdesc = "+namespace+".1.d")
	lines = append(lines, "\thidden = yes")
	lines = append(lines, "\tis_triggered_only = yes")
	lines = append(lines, "")
	lines = append(lines, "\timmediate = {")

	if len(countryLeaderReplacements) > 0 {
		lines = append(lines, "\t\t"+countryTag+" = {")
		for _, cl := range countryLeaderReplacements {
			lines = append(lines, "\t\t\tretire_country_leader = yes")
			lines = append(lines, "\t\t\tcreate_country_leader = {")
			lines = append(lines, "\t\t\t\tname = "+cl.CharTag)
			if CLDescLocKeyWillBeEmitted(cl) {
				lines = append(lines, "\t\t\t\tdesc = "+cl.CharTag+"_desc")
			}
			if cl.PictureSprite != "" {
				lines = append(lines, "\t\t\t\tpicture = \""+cl.PictureSprite+"\"")
			}
			expire := cl.Expire
			if expire == "" {
				expire = "1965.1.1"
			}
			lines = append(lines, "\t\t\t\texpire = \""+expire+"\"")
			ideology := cl.Ideology
			if ideology == "" {
				ideology = "neutrality"
			}
			lines = append(lines, "\t\t\t\tideology = "+ideology)
			if len(cl.Traits) > 0 {
				lines = append(lines, "\t\t\t\ttraits = { "+strings.Join(cl.Traits, " ")+" }")
			}
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
	}

	for _, charTag := range charOrder {
		lines = append(lines, "\t\t"+charTag+" = {")
		for _, et := range perChar[charTag] {
			lines = append(lines, "\t\t\t"+et.Effect+" = "+et.TID)
		}
		charPortraits, _ := portraitOverrides.Get(charTag).(*Dict)
		if charPortraits.Truthy() {
			lines = append(lines, "\t\t\tset_portraits = {")
			for _, layer := range []string{"civilian", "army"} {
				slots, _ := charPortraits.Get(layer).(*Dict)
				if !slots.Truthy() {
					continue
				}
				lines = append(lines, "\t\t\t\t"+layer+" = {")
				for _, slot := range []string{"large", "small"} {
					if sprite := slots.Get(slot); pyTruthy(sprite) {
						lines = append(lines, "\t\t\t\t\t"+slot+" = \""+PyStr(sprite)+"\"")
					}
				}
				lines = append(lines, "\t\t\t\t}")
			}
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
	}
	lines = append(lines, "\t}")
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func EmitHistoryFileLeaderReplacementBlock(countryTag string, clReplacements []*CLReplacement) []string {
	if len(clReplacements) == 0 {
		return nil
	}
	lines := []string{
		"# Family 15 · pre-day-0 leader replacement for " + countryTag,
		"# Mirrors the day-1 trait_overlay event so country-selection",
		"# screen / bookmark UI shows the mod's leader, not vanilla.",
		"retire_country_leader = yes",
	}
	for _, cl := range clReplacements {
		lines = append(lines, "create_country_leader = {")
		lines = append(lines, "\tname = "+cl.CharTag)
		if CLDescLocKeyWillBeEmitted(cl) {
			lines = append(lines, "\tdesc = "+cl.CharTag+"_desc")
		}
		if cl.PictureSprite != "" {
			lines = append(lines, "\tpicture = \""+cl.PictureSprite+"\"")
		}
		expire := cl.Expire
		if expire == "" {
			expire = "1965.1.1"
		}
		lines = append(lines, "\texpire = \""+expire+"\"")
		ideology := cl.Ideology
		if ideology == "" {
			ideology = "neutrality"
		}
		lines = append(lines, "\tideology = "+ideology)
		if len(cl.Traits) > 0 {
			lines = append(lines, "\ttraits = { "+strings.Join(cl.Traits, " ")+" }")
		}
		lines = append(lines, "}")
	}
	lines = append(lines, "")
	return lines
}

func ScanForEventRef(text, eventID string) bool {
	if text == "" || eventID == "" {
		return false
	}
	eid := regexp.QuoteMeta(eventID)
	bare := `(?:country_event|news_event|event)\s*=\s*` + eid + `\b`
	braced := `(?:country_event|news_event|event)\s*=\s*\{[^}]*\bid\s*=\s*` + eid + `\b`
	pattern, err := regexp.Compile(`(?:` + bare + `)|(?:` + braced + `)`)
	if err != nil {
		return false
	}
	return pattern.MatchString(text)
}

func EmitSuperEventBoilerplate(modDir, ctag string) []FileEntry {
	upper := pyUpper(ctag)
	r := func(template string) string { return strings.ReplaceAll(template, "«U»", upper) }

	gfxText := r("spriteTypes = {\n" +
		"\n" +
		"\t### Superevent frame assets (binary DDS shipped with `gfx/tvbigevent/`).\n" +
		"\t### These 4 sprite names are referenced by `interface/«U»_superevent.gui`.\n" +
		"\n" +
		"\tspriteType = {\n" +
		"\t\tname = \"GFX_«U»_SE_bg\"\n" +
		"\t\ttexturefile = \"gfx//tvbigevent//SE_bg.dds\"\n" +
		"\t\tlegacy_lazy_load = yes\n" +
		"\t}\n" +
		"\tspriteType = {\n" +
		"\t\tname = \"GFX_«U»_SE_exit_btn\"\n" +
		"\t\ttexturefile = \"gfx//tvbigevent//SE_btn.dds\"\n" +
		"\t\tlegacy_lazy_load = yes\n" +
		"\t}\n" +
		"\tspriteType = {\n" +
		"\t\tname = \"GFX_«U»_SE_frame_overlay\"\n" +
		"\t\ttexturefile = \"gfx//tvbigevent//SE_frame.dds\"\n" +
		"\t\tlegacy_lazy_load = yes\n" +
		"\t}\n" +
		"\t# Fallback picture · used when no SuperEvent row uploaded a custom image.\n" +
		"\tspriteType = {\n" +
		"\t\tname = \"GFX_«U»_SE_pic_entry\"\n" +
		"\t\ttexturefile = \"gfx//tvbigevent//SE_pic.dds\"\n" +
		"\t\tlegacy_lazy_load = yes\n" +
		"\t}\n" +
		"\n" +
		"\t### Per-event picture sprites are registered separately in\n" +
		"\t### interface/«U»_se_pictures.gfx (auto-emitted at export time).\n" +
		"}\n")

	guiText := r("guiTypes = {\n" +
		"\n" +
		"\tcontainerWindowType = {\n" +
		"\t\tname = \"«U»_SE_main_container\"\n" +
		"\t\torientation = center\n" +
		"\t\tOrigo = center\n" +
		"\t\tsize = { width = 500 height = 700 }\n" +
		"\t\tposition = { x = 0  y = 0 }\n" +
		"\t\tmoveable = yes\n" +
		"\n" +
		"\t\tbackground = {\n" +
		"\t\t\tname = \"Background\"\n" +
		"\t\t\tquadTextureSprite = \"GFX_tiled_window_transparent\"\n" +
		"\t\t}\n" +
		"\n" +
		"\t\ticonType = {\n" +
		"\t\t\tname = \"«U»_SE_bg\"\n" +
		"\t\t\tspriteType = \"GFX_«U»_SE_bg\"\n" +
		"\t\t\tposition = { x = 0 y = 0 }\n" +
		"\t\t\tOrientation = upper_left\n" +
		"\t\t\talwaystransparent = yes\n" +
		"\t\t}\n" +
		"\n" +
		"\t\ticonType = {\n" +
		"\t\t\tname = \"«U»_SE_picture_positioning_entry\"\n" +
		"\t\t\tspriteType = \"GFX_«U»_SE_pic_entry\"\n" +
		"\t\t\tposition = { x = 20  y = 21 }\n" +
		"\t\t\tOrientation = upper_left\n" +
		"\t\t\talwaystransparent = yes\n" +
		"\t\t}\n" +
		"\n" +
		"\t\ticonType = {\n" +
		"\t\t\tname = \"«U»_SE_frame_overlay\"\n" +
		"\t\t\tspriteType = \"GFX_«U»_SE_frame_overlay\"\n" +
		"\t\t\tposition = { x = 0 y = 0 }\n" +
		"\t\t\tOrientation = upper_left\n" +
		"\t\t\talwaystransparent = yes\n" +
		"\t\t}\n" +
		"\n" +
		"\t\t# 关闭按钮 · clickable hitbox 必须来自 quadTextureSprite · buttonType 用裸\n" +
		"\t\t# spriteType(iconType 的非交互属性)会渲染贴图但拿不到点击 hitbox → 整个\n" +
		"\t\t# superevent「点哪里都没反应」。三角验证:TFR(GFX_spacebar)+ CCCW\n" +
		"\t\t# (GFX_Super_event_bg)两个 working mod 的 close 按钮全用 quadTextureSprite。\n" +
		"\t\t# 按钮名 + \"_click\" = scripted_gui effect 名(«U»_SE_exit_btn_click)·\n" +
		"\t\t# 这是 HOI4 scripted_gui 的绑定约定(TFR Super_Event_Close→_click 印证)。\n" +
		"\t\tbuttonType = {\n" +
		"\t\t\tname = \"«U»_SE_exit_btn\"\n" +
		"\t\t\tposition = { x = 184  y = 638 }\n" +
		"\t\t\tquadTextureSprite = \"GFX_«U»_SE_exit_btn\"\n" +
		"\t\t\torientation = upper_left\n" +
		"\t\t\tscale = 1.1\n" +
		"\t\t\tclicksound = click_ok\n" +
		"\t\t}\n" +
		"\n" +
		"\t\t# 5 段化文案 · text 字段必须用 [NAME] 方括号触发 scripted_localisation dispatch.\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_btn\"\n" +
		"\t\t\tposition = { x = 0 y = 654  }\n" +
		"\t\t\tfont = \"hoi_20bs\"\n" +
		"\t\t\ttext = \"[«U»_SE_btn]\"\n" +
		"\t\t\tmaxWidth = 500\n" +
		"\t\t\tmaxHeight = 30\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t\tformat = center\n" +
		"\t\t}\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_title\"\n" +
		"\t\t\tposition = { x = 0 y = 278 }\n" +
		"\t\t\tfont = \"hoi_30header\"\n" +
		"\t\t\ttext = \"[«U»_SE_title]\"\n" +
		"\t\t\tmaxWidth = 500\n" +
		"\t\t\tmaxHeight = 80\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t\tformat = center\n" +
		"\t\t}\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_desc_left\"\n" +
		"\t\t\tposition = { x = 30 y = 342 }\n" +
		"\t\t\tfont = \"hoi_18mbs\"\n" +
		"\t\t\ttext = \"[«U»_SE_desc]\"\n" +
		"\t\t\tmaxWidth = 440\n" +
		"\t\t\tmaxHeight = 200\n" +
		"\t\t\tformat = left\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t}\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_desc_center\"\n" +
		"\t\t\tposition = { x = 30 y = 342 }\n" +
		"\t\t\tfont = \"hoi_18mbs\"\n" +
		"\t\t\ttext = \"[«U»_SE_desc]\"\n" +
		"\t\t\tmaxWidth = 440\n" +
		"\t\t\tmaxHeight = 200\n" +
		"\t\t\tformat = center\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t}\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_quote\"\n" +
		"\t\t\tposition = { x = 50  y = 560 }\n" +
		"\t\t\tfont = \"hoi_16mbs\"\n" +
		"\t\t\ttext = \"[«U»_SE_quote]\"\n" +
		"\t\t\tmaxWidth = 400\n" +
		"\t\t\tmaxHeight = 50\n" +
		"\t\t\tformat = center\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t}\n" +
		"\t\tinstantTextboxType = {\n" +
		"\t\t\tname = \"«U»_SE_quote_extra\"\n" +
		"\t\t\tposition = { x = 50  y = 560 }\n" +
		"\t\t\tfont = \"hoi_16mbs\"\n" +
		"\t\t\ttext = \"[«U»_SE_quote]\"\n" +
		"\t\t\tmaxWidth = 400\n" +
		"\t\t\tmaxHeight = 50\n" +
		"\t\t\tformat = center\n" +
		"\t\t\tfixedsize = yes\n" +
		"\t\t}\n" +
		"\t}\n" +
		"\n" +
		"}\n")

	seEffects := r("SE_center = {\n" +
		"\tset_global_flag = «U»_SE_center\n" +
		"}\n" +
		"# Toggle the SuperEvent layout to centered text mode (default is left-aligned).\n")

	sguiText := r("scripted_gui = {\n" +
		"\t«U»_SE_main_container = {\n" +
		"\t\twindow_name = «U»_SE_main_container\n" +
		"\t\tcontext_type = player_context\n" +
		"\n" +
		"\t\tvisible = {\n" +
		"\t\t\thas_variable = global.«U»_se_display\n" +
		"\t\t\tis_ai = no\n" +
		"\t\t}\n" +
		"\t\ttriggers = {\n" +
		"\t\t\t# Toggle alignment by hiding one of the two desc textboxes\n" +
		"\t\t\t«U»_SE_desc_left_visible = { NOT = { has_global_flag = «U»_SE_center } }\n" +
		"\t\t\t«U»_SE_desc_center_visible = { has_global_flag = «U»_SE_center }\n" +
		"\t\t}\n" +
		"\t\teffects = {\n" +
		"\t\t\t«U»_SE_exit_btn_click = {\n" +
		"\t\t\t\tclr_global_flag = «U»_SE_center\n" +
		"\t\t\t\tclear_variable = global.«U»_se_display\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t\tproperties = {\n" +
		"\t\t\t# scripted_localisation [«U»_SE_getpicture] returns the GFX sprite\n" +
		"\t\t\t# matching global.«U»_se_display value. Routed in\n" +
		"\t\t\t# common/scripted_localisation/«U»_SE_getpicture.txt (auto-emitted).\n" +
		"\t\t\t«U»_SE_picture_positioning_entry = { image = \"[«U»_SE_getpicture]\" }\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n")

	return []FileEntry{
		{modDir + "interface/" + upper + "_superevent.gfx", gfxText},
		{modDir + "interface/" + upper + "_superevent.gui", guiText},
		{modDir + "common/scripted_effects/" + upper + "_big_event_scripted_effects.txt", seEffects},
		{modDir + "common/scripted_guis/" + upper + "_superevent_gui.txt", sguiText},
	}
}
