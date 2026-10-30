package characters

import (
	"sort"
	"strings"
)

type bucket struct {
	Key   string
	Chars []*Character
}

func bucketByCountry(items []*Character, primaryTag string) []bucket {
	pt := pyUpper(primaryTag)
	if pt == "" {
		pt = "UNK"
	}
	var order []string
	byKey := map[string]int{}
	var out []bucket
	for _, it := range items {
		key := pyUpper(it.CountryTag)
		if key == "" {
			key = pt
		}
		if idx, ok := byKey[key]; ok {
			out[idx].Chars = append(out[idx].Chars, it)
			continue
		}
		byKey[key] = len(out)
		order = append(order, key)
		out = append(out, bucket{Key: key, Chars: []*Character{it}})
	}
	_ = order
	return out
}

type sourceGroup struct {
	Path string
	Rows []*Character
}

func groupBySourceFile(rows []*Character, defaultPath func(*Character) string, expectedDir string) []sourceGroup {
	prefix := strings.Trim(expectedDir, "/")
	byPath := map[string]int{}
	var out []sourceGroup
	for _, row := range rows {
		raw := pyStrip(strings.ReplaceAll(row.SourceFile, "\\", "/"))
		path := ""
		if raw != "" {
			var segments []string
			hasDotDot := false
			for _, s := range strings.Split(raw, "/") {
				if s == "" || s == "." {
					continue
				}
				if s == ".." {
					hasDotDot = true
				}
				segments = append(segments, s)
			}
			if len(segments) > 0 && !hasDotDot && !strings.Contains(raw, ":") {
				candidate := strings.Join(segments, "/")
				if strings.HasPrefix(candidate, prefix+"/") {
					path = candidate
				} else {
					path = prefix + "/" + segments[len(segments)-1]
				}
			}
		}
		if path == "" {
			path = defaultPath(row)
		}
		if idx, ok := byPath[path]; ok {
			out[idx].Rows = append(out[idx].Rows, row)
			continue
		}
		byPath[path] = len(out)
		out = append(out, sourceGroup{Path: path, Rows: []*Character{row}})
	}
	return out
}

func authorRawOverridePaths(rawFiles []RawFile) map[string]bool {
	out := map[string]bool{}
	for _, row := range rawFiles {
		if row.Source == "imported" {
			continue
		}
		p := strings.Trim(strings.ReplaceAll(row.DestPath, "\\", "/"), "/")
		out[p] = true
	}
	return out
}

func generateGfxSprites(chars []*Character, countryTag string, imageGate map[int][2]bool) string {
	lines := []string{"spriteTypes = {"}
	for _, char := range chars {
		slug := extractSlug(char.Tag, countryTag)
		gate := imageGate[char.ID]
		portraitOK, advisorOK := gate[0], gate[1]
		if portraitOK {
			lines = append(lines, "\tspriteType = {")
			lines = append(lines, "\t\tname = "+portraitSpriteName(countryTag, slug))
			lines = append(lines, "\t\ttexturefile = \""+exportPortraitPath(countryTag, slug)+"\"")
			lines = append(lines, "\t}")
		}
		if advisorOK {
			lines = append(lines, "\tspriteType = {")
			lines = append(lines, "\t\tname = "+advisorSpriteName(countryTag, slug))
			lines = append(lines, "\t\ttexturefile = \""+exportAdvisorIconPath(countryTag, slug)+"\"")
			lines = append(lines, "\t}")
		}
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func GenerateGFXSprites(chars []*Character, countryTag string, imageGate map[int][2]bool) string {
	return generateGfxSprites(chars, countryTag, imageGate)
}

func emitSpriteBlock(entries [][2]string) string {
	if len(entries) == 0 {
		return ""
	}
	lines := []string{"spriteTypes = {"}
	for _, e := range entries {
		lines = append(lines, "\tspriteType = {")
		lines = append(lines, "\t\tname = "+e[0])
		lines = append(lines, "\t\ttexturefile = \""+e[1]+"\"")
		lines = append(lines, "\t}")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

type overlayPair struct {
	CharTag string
	Traits  []string
}

func buildTraitOverlayPairs(imported []*Character, customIDs map[string]bool) []overlayPair {
	var pairs []overlayPair
	for _, c := range imported {
		var attached []string
		for _, t := range c.Traits {
			if customIDs[t] {
				attached = append(attached, t)
			}
		}
		if len(attached) > 0 {
			pairs = append(pairs, overlayPair{CharTag: c.Tag, Traits: attached})
		}
	}
	return pairs
}

var unitLeaderRoles = map[string]bool{"general": true, "field_marshal": true, "admiral": true}

type portraitOverrides struct {
	Order  []string
	Layers map[string]map[string]map[string]string
}

func (p *portraitOverrides) has(tag string) bool {
	_, ok := p.Layers[tag]
	return ok
}

func buildPortraitOverrides(in *Input, replacementChars []*Character, ctag string) *portraitOverrides {
	out := &portraitOverrides{Layers: map[string]map[string]map[string]string{}}
	for _, c := range replacementChars {
		if !c.IsPortraitModified {
			continue
		}
		slug := extractSlug(c.Tag, ctag)
		hasPortrait := in.PortraitObs(c.ID) != ""
		hasAdvisor := in.AdvisorObs(c.ID) != ""
		if !hasPortrait && !hasAdvisor {
			continue
		}
		portraitSprite := portraitSpriteName(ctag, slug)
		advisorSprite := advisorSpriteName(ctag, slug)
		isMilitary := unitLeaderRoles[c.RoleType]
		needsArmyLarge := isMilitary || c.RoleType == "scientist"
		needsArmySmall := needsArmyLarge || c.RoleType == "advisor" || c.HasAdvisorRole
		charLayers := map[string]map[string]string{}
		if hasPortrait {
			charLayers["civilian"] = map[string]string{"large": portraitSprite}
			if needsArmyLarge {
				if charLayers["army"] == nil {
					charLayers["army"] = map[string]string{}
				}
				charLayers["army"]["large"] = portraitSprite
			}
		}
		if hasAdvisor && needsArmySmall {
			if charLayers["army"] == nil {
				charLayers["army"] = map[string]string{}
			}
			charLayers["army"]["small"] = advisorSprite
		}
		if len(charLayers) > 0 {
			out.Layers[c.Tag] = charLayers
			out.Order = append(out.Order, c.Tag)
		}
	}
	return out
}

func resolveTraitOverlayNamespace(ctag string, existing map[string]bool) string {
	base := pyLower(ctag) + "_trait_overlay"
	if !existing[base] {
		return base
	}
	n := 2
	for {
		cand := base + "_" + itoa(n)
		if !existing[cand] {
			return cand
		}
		n++
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

type clRepl struct {
	CharTag       string
	Char          *Character
	LocEmitted    bool
	Ideology      string
	Traits        []string
	Expire        string
	PictureSprite string
}

func clDescLocKeyWillBeEmitted(cl *clRepl) bool {
	if cl.Char == nil {
		return false
	}
	if !cl.LocEmitted {
		return false
	}
	tag := cl.Char.Tag
	return resolveDescriptionAnyLang(cl.Char) != "" ||
		resolveMeaningfulDisplayNameENTruthy(cl.Char, tag)
}

type effectGroup struct {
	Name  string
	Pairs []overlayPair
}

func generateTraitOverlayEvent(
	namespace string,
	effectGroups []effectGroup,
	overrides *portraitOverrides,
	clReplacements []*clRepl,
	countryTag string,
) string {
	perChar := map[string][][2]string{}
	var charOrder []string
	for _, g := range effectGroups {
		for _, pair := range g.Pairs {
			if len(pair.Traits) == 0 {
				continue
			}
			if _, ok := perChar[pair.CharTag]; !ok {
				perChar[pair.CharTag] = nil
				charOrder = append(charOrder, pair.CharTag)
			}
			for _, tid := range pair.Traits {
				perChar[pair.CharTag] = append(perChar[pair.CharTag], [2]string{g.Name, tid})
			}
		}
	}
	if overrides != nil {
		for _, charTag := range overrides.Order {
			if _, ok := perChar[charTag]; !ok {
				perChar[charTag] = nil
				charOrder = append(charOrder, charTag)
			}
		}
	}
	if len(perChar) == 0 && len(clReplacements) == 0 {
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

	if len(clReplacements) > 0 {
		lines = append(lines, "\t\t"+countryTag+" = {")
		for _, cl := range clReplacements {
			lines = append(lines, "\t\t\tretire_country_leader = yes")
			lines = append(lines, "\t\t\tcreate_country_leader = {")
			lines = append(lines, "\t\t\t\tname = "+cl.CharTag)
			if clDescLocKeyWillBeEmitted(cl) {
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
		for _, eff := range perChar[charTag] {
			lines = append(lines, "\t\t\t"+eff[0]+" = "+eff[1])
		}
		var charPortraits map[string]map[string]string
		if overrides != nil {
			charPortraits = overrides.Layers[charTag]
		}
		if len(charPortraits) > 0 {
			lines = append(lines, "\t\t\tset_portraits = {")
			for _, layer := range []string{"civilian", "army"} {
				slots := charPortraits[layer]
				if len(slots) == 0 {
					continue
				}
				lines = append(lines, "\t\t\t\t"+layer+" = {")
				for _, slot := range []string{"large", "small"} {
					if sprite := slots[slot]; sprite != "" {
						lines = append(lines, "\t\t\t\t\t"+slot+" = \""+sprite+"\"")
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

func ctagOf(c *Character, primaryUpper string) string {
	t := pyUpper(c.CountryTag)
	if t == "" {
		return primaryUpper
	}
	return t
}

func vanillaPortraitRef(c *Character, ctag string, isSubmod, hasBytes bool) string {
	if hasBytes {
		return ""
	}
	ref := pyStrip(c.LargePortrait)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "GFX_") || strings.HasPrefix(ref, "gfx/") {
		return ref
	}
	if isSubmod && ref != portraitFilename(ctag, extractSlug(c.Tag, ctag)) {
		return ref
	}
	return ""
}

func vanillaAdvisorRef(c *Character, ctag string, isSubmod, hasBytes bool) string {
	if hasBytes {
		return ""
	}
	ref := pyStrip(c.SmallPortrait)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "GFX_") || strings.HasPrefix(ref, "gfx/") {
		return ref
	}
	if isSubmod && ref != advisorIconFilename(ctag, extractSlug(c.Tag, ctag)) {
		return ref
	}
	return ""
}

func wantsOverlayEmit(c *Character) bool {
	v, ok := kvLookup(c.RawExtras, "emit_trait_overlay_event")
	if !ok {
		return true
	}
	if b, isBool := v.(bool); isBool && !b {
		return false
	}
	return true
}

func EmitCharacters(in *Input) *Result {
	res := &Result{
		Texts:           map[string]string{},
		Recruit:         map[string][]string{},
		CL:              map[string][]CLOut{},
		Skips:           map[string]int{},
		StructuredPaths: []string{},
		TraitOverlays:   map[string]TraitOverlayArtifacts{},
	}
	writtenAssetPaths := map[string]bool{}
	locEmittedCharIDs := map[int]bool{}
	charScriptsByID := map[int]string{}
	structuredSet := map[string]bool{}

	if len(in.Characters) > 0 {
		for _, b := range bucketByCountry(in.Characters, in.Tag) {
			ctag, chars := b.Key, b.Chars

			type resolvedRow struct {
				c      *Character
				pb, ab bool
			}
			var resolved []resolvedRow
			imageGate := map[int][2]bool{}
			for _, char := range chars {
				pb := in.PortraitObs(char.ID) != ""
				needsAdvisorAsset := char.RoleType == "advisor" || char.HasAdvisorRole
				ab := false
				if needsAdvisorAsset {
					ab = in.AdvisorObs(char.ID) != ""
				}
				resolved = append(resolved, resolvedRow{char, pb, ab})
				imageGate[char.ID] = [2]bool{pb, ab}
			}

			for _, r := range resolved {
				charScriptsByID[r.c.ID] = GenerateCharacterScript(r.c, ctag, ScriptOpts{
					HasPortraitImage:     r.pb,
					HasAdvisorImage:      r.ab,
					PortraitPathOverride: vanillaPortraitRef(r.c, ctag, in.IsSubmod, r.pb),
					AdvisorPathOverride:  vanillaAdvisorRef(r.c, ctag, in.IsSubmod, r.ab),
				})
			}

			res.Texts["interface/"+ctag+"_characters.gfx"] = generateGfxSprites(chars, ctag, imageGate)

			for _, c := range chars {
				locEmittedCharIDs[c.ID] = true
			}

			for _, r := range resolved {
				slug := extractSlug(r.c.Tag, ctag)
				if r.pb {
					writtenAssetPaths[exportPortraitPath(ctag, slug)] = true
				}
				if r.ab {
					writtenAssetPaths[exportAdvisorIconPath(ctag, slug)] = true
				}
			}
		}

		defaultPath := func(c *Character) string {
			ctag := pyUpper(c.CountryTag)
			if ctag == "" {
				ctag = pyUpper(in.Tag)
			}
			if ctag == "" {
				ctag = "UNK"
			}
			return "common/characters/" + in.Tag + "_" + ctag + ".txt"
		}
		overrides := authorRawOverridePaths(in.RawFiles)
		for _, g := range groupBySourceFile(in.Characters, defaultPath, "common/characters") {
			var scripts []string
			for _, c := range g.Rows {
				if s, ok := charScriptsByID[c.ID]; ok {
					scripts = append(scripts, s)
				}
			}
			if len(scripts) > 0 && !overrides[g.Path] {
				res.Texts[g.Path] = strings.Join(scripts, "\n\n")
				structuredSet[g.Path] = true
			}
		}
	}

	primaryUpper := pyUpper(in.Tag)
	if primaryUpper == "" {
		primaryUpper = "UNK"
	}

	var ctagOrder []string
	ctagSeen := map[string]bool{}
	addCtag := func(t string) {
		if !ctagSeen[t] {
			ctagSeen[t] = true
			ctagOrder = append(ctagOrder, t)
		}
	}
	for _, c := range in.Characters {
		addCtag(ctagOf(c, primaryUpper))
	}
	for _, c := range in.ReplacementCountryLeaders {
		addCtag(ctagOf(c, primaryUpper))
	}
	for _, c := range in.ReplacementUnitLeaders {
		addCtag(ctagOf(c, primaryUpper))
	}
	for _, c := range in.ReplacementScientists {
		addCtag(ctagOf(c, primaryUpper))
	}

	for _, c := range in.Characters {
		if c.ReplacesVanillaID {
			continue
		}
		cctag := ctagOf(c, primaryUpper)
		if c.Tag != "" {
			res.Recruit[cctag] = append(res.Recruit[cctag], c.Tag)
		}
	}

	existingNsNames := map[string]bool{}
	for _, ns := range in.EventNamespaces {
		existingNsNames[pyLower(ns.Namespace)] = true
	}

	for _, ctag := range ctagOrder {
		customCLTraitIDs := map[string]bool{}
		customULTraitIDs := map[string]bool{}
		customSCTraitIDs := map[string]bool{}
		for _, t := range in.CustomTraits {
			tctag := pyUpper(t.CountryTag)
			if tctag == "" {
				tctag = primaryUpper
			}
			if tctag != ctag {
				continue
			}
			switch t.TraitType {
			case "country_leader":
				customCLTraitIDs[t.TraitID] = true
			case "unit_leader":
				customULTraitIDs[t.TraitID] = true
			case "scientist":
				customSCTraitIDs[t.TraitID] = true
			}
		}

		filterInCountry := func(chars []*Character) []*Character {
			var out []*Character
			for _, c := range chars {
				if ctagOf(c, primaryUpper) == ctag && wantsOverlayEmit(c) {
					out = append(out, c)
				}
			}
			return out
		}
		replacementCLs := filterInCountry(in.ReplacementCountryLeaders)
		replacementULs := filterInCountry(in.ReplacementUnitLeaders)
		replacementSCs := filterInCountry(in.ReplacementScientists)

		validCLTraitSet := map[string]bool{}
		for t := range customCLTraitIDs {
			validCLTraitSet[t] = true
		}
		for t := range in.VanillaTraits {
			validCLTraitSet[t] = true
		}

		var clReplacements []*clRepl
		for _, c := range replacementCLs {
			vanillaTraits := c.Traits
			var customAttached []string
			for _, t := range vanillaTraits {
				if customCLTraitIDs[t] {
					customAttached = append(customAttached, t)
				}
			}
			seen := map[string]bool{}
			var combined []string
			for _, t := range append(append([]string{}, vanillaTraits...), customAttached...) {
				if !seen[t] {
					seen[t] = true
					combined = append(combined, t)
				}
			}

			if in.VanillaTraits != nil {
				var gatedCombined []string
				for _, t := range combined {
					if validCLTraitSet[t] {
						gatedCombined = append(gatedCombined, t)
					} else {
						res.Skips["create_country_leader.traits["+t+"]"]++
					}
				}
				combined = gatedCombined
			}

			pictureSprite := ""
			if c.IsPortraitModified {
				slug := extractSlug(c.Tag, ctag)
				if in.PortraitObs(c.ID) != "" {
					pictureSprite = portraitSpriteName(ctag, slug)
				}
			}
			ideology := c.Ideology
			if ideology == "" {
				ideology = "neutrality"
			}
			expire := c.Expire
			if expire == "" {
				expire = "1965.1.1"
			}
			clReplacements = append(clReplacements, &clRepl{
				CharTag:       c.Tag,
				Char:          c,
				LocEmitted:    locEmittedCharIDs[c.ID],
				Ideology:      ideology,
				Traits:        combined,
				Expire:        expire,
				PictureSprite: pictureSprite,
			})
		}

		if len(clReplacements) > 0 {
			var outs []CLOut
			for _, cl := range clReplacements {
				traits := cl.Traits
				if traits == nil {
					traits = []string{}
				}
				outs = append(outs, CLOut{
					CharTag:       cl.CharTag,
					Ideology:      cl.Ideology,
					Traits:        traits,
					Expire:        cl.Expire,
					PictureSprite: cl.PictureSprite,
					LocEmitted:    cl.LocEmitted,
				})
			}
			res.CL[ctag] = outs
		}

		ulPairs := buildTraitOverlayPairs(replacementULs, customULTraitIDs)
		scPairs := buildTraitOverlayPairs(replacementSCs, customSCTraitIDs)

		var effectGroups []effectGroup
		if len(ulPairs) > 0 {
			effectGroups = append(effectGroups, effectGroup{"add_unit_leader_trait", ulPairs})
		}
		if len(scPairs) > 0 {
			effectGroups = append(effectGroups, effectGroup{"add_scientist_trait", scPairs})
		}

		portraitReplacementChars := append(append([]*Character{}, replacementULs...), replacementSCs...)
		overrides := buildPortraitOverrides(in, portraitReplacementChars, ctag)

		hasOverlayPayload := len(effectGroups) > 0 || len(overrides.Order) > 0 || len(clReplacements) > 0
		overlayNs := ""
		if hasOverlayPayload {
			overlayNs = resolveTraitOverlayNamespace(ctag, existingNsNames)
			existingNsNames[overlayNs] = true
		}

		if hasOverlayPayload && overlayNs != "" {
			eventText := generateTraitOverlayEvent(overlayNs, effectGroups, overrides, clReplacements, ctag)
			eventPath := "events/" + ctag + "_trait_overlay.txt"
			res.Texts[eventPath] = eventText
			recordTraitOverlay(res, ctag, func(artifacts *TraitOverlayArtifacts) {
				artifacts.Namespace = overlayNs
				artifacts.EventPath = eventPath
			})
			emitTraitOverlayLocalisation(res, ctag, overlayNs, in.ExportLanguages)
		}

		clCharsWithPicture := map[string]bool{}
		for _, cl := range clReplacements {
			if cl.PictureSprite != "" {
				clCharsWithPicture[cl.CharTag] = true
			}
		}
		charsToEmitPortraits := append(append(append([]*Character{},
			replacementCLs...), replacementULs...), replacementSCs...)
		if len(overrides.Order) > 0 || len(clCharsWithPicture) > 0 {
			var gfxEntries [][2]string
			seenSpriteNames := map[string]bool{}
			for _, char := range charsToEmitPortraits {
				if !overrides.has(char.Tag) && !clCharsWithPicture[char.Tag] {
					continue
				}
				slug := extractSlug(char.Tag, ctag)
				if in.PortraitObs(char.ID) != "" {
					portraitZipPath := exportPortraitPath(ctag, slug)
					spriteName := portraitSpriteName(ctag, slug)
					if !writtenAssetPaths[portraitZipPath] {
						writtenAssetPaths[portraitZipPath] = true
					}
					if !seenSpriteNames[spriteName] {
						gfxEntries = append(gfxEntries, [2]string{spriteName, portraitZipPath})
						seenSpriteNames[spriteName] = true
					}
				}
				if in.AdvisorObs(char.ID) != "" {
					advisorZipPath := exportAdvisorIconPath(ctag, slug)
					spriteName := advisorSpriteName(ctag, slug)
					if !writtenAssetPaths[advisorZipPath] {
						writtenAssetPaths[advisorZipPath] = true
					}
					if !seenSpriteNames[spriteName] {
						gfxEntries = append(gfxEntries, [2]string{spriteName, advisorZipPath})
						seenSpriteNames[spriteName] = true
					}
				}
			}
			if len(gfxEntries) > 0 {
				gfxPath := "interface/zz_" + ctag + "_replacement_portraits.gfx"
				res.Texts[gfxPath] = emitSpriteBlock(gfxEntries)
				recordTraitOverlay(res, ctag, func(artifacts *TraitOverlayArtifacts) {
					artifacts.ReplacementPortraitsPath = gfxPath
				})
			}
		}
	}

	emitReplacementLocalisation(in, res, primaryUpper)

	for p := range structuredSet {
		res.StructuredPaths = append(res.StructuredPaths, p)
	}
	sort.Strings(res.StructuredPaths)
	return res
}

func EmitTraitOverlayArtifacts(in *Input) *Result {
	full := EmitCharacters(in)
	out := &Result{
		Texts:         map[string]string{},
		Recruit:       full.Recruit,
		CL:            full.CL,
		Skips:         full.Skips,
		TraitOverlays: full.TraitOverlays,
	}
	for _, artifacts := range full.TraitOverlays {
		paths := append([]string{artifacts.EventPath, artifacts.ReplacementPortraitsPath}, artifacts.EventLocalisationPaths...)
		paths = append(paths, artifacts.ReplacementLocalisationPaths...)
		for _, path := range paths {
			if text, ok := full.Texts[path]; ok {
				out.Texts[path] = text
			}
		}
	}
	return out
}
