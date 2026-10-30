package worldout

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

func emitRawExtrasBlockOrScalar(lines *[]string, key string, valRaw Value, indent string) {
	if list, ok := valRaw.([]Value); ok {
		for _, item := range list {
			emitRawExtrasBlockOrScalar(lines, key, item, indent)
		}
		return
	}
	var raw string
	switch t := valRaw.(type) {
	case nil:
		raw = ""
	case json.Number:
		if numIsFloatLiteral(t) {
			f, _ := t.Float64()
			raw = fmtHoi4Float(f)
		} else {
			raw = t.String()
		}
	default:
		raw = pyStr(valRaw)
	}
	isBlock := strings.ContainsAny(raw, "\n{=><")
	if !isBlock {
		val := pyStrip(raw)
		if val == "" {
			return
		}
		*lines = append(*lines, indent+key+" = "+val)
		return
	}
	var body []string
	for _, vl := range pySplitlines(raw) {
		if s := pyStrip(vl); s != "" {
			body = append(body, s)
		}
	}
	if len(body) == 0 {
		*lines = append(*lines, indent+key+" = { }")
		return
	}
	*lines = append(*lines, indent+key+" = {")
	for _, vl := range body {
		*lines = append(*lines, indent+"\t"+vl)
	}
	*lines = append(*lines, indent+"}")
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

func bookmarkWeatherSeed(bmID string) int64 {
	if bmID == "" {
		bmID = "bookmark"
	}
	h := uint64(2166136261)
	for _, ch := range bmID {
		h = (h*16777619 + uint64(ch)) & 0xFFFFFFFF
	}
	return 10000 + int64(h%90000)
}

func bookmarkHasWeather(rawExtras *OMap) bool {
	if rawExtras == nil {
		return false
	}
	for _, k := range rawExtras.Keys() {
		v, _ := rawExtras.Get(k)
		if toLower(pyStrip(k)) == "effect" {
			return true
		}
		if v == nil {
			continue
		}

		if list, ok := v.([]Value); ok {
			for _, item := range list {
				if item != nil && strings.Contains(pyStr(item), "randomize_weather") {
					return true
				}
			}
			continue
		}
		if strings.Contains(pyStr(v), "randomize_weather") {
			return true
		}
	}
	return false
}

const defaultBookmarkPicture = "GFX_select_date_1936"

var vanillaBookmarkPictures = map[string]bool{
	"GFX_select_date_1936": true,
	"GFX_select_date_1939": true,
}

func resolveBookmarkPicture(pictureValue, customSprite string, resolvedBackdropSprites map[string]bool) (string, bool) {
	pictureRef := pyStrip(pictureValue)
	if resolvedBackdropSprites[customSprite] {
		return customSprite, false
	}
	if pictureRef == "" {
		return defaultBookmarkPicture, false
	}
	if vanillaBookmarkPictures[pictureRef] {
		return pictureRef, false
	}
	return defaultBookmarkPicture, true
}

var hoi4DateRe = regexp.MustCompile(
	`^([0-9]{1,4})\.([0-9]{1,2})\.([0-9]{1,2})(?:\.([0-9]{1,2}))?$`)

type historyDateError string

func (e historyDateError) Error() string { return exportmsg.Printer{}.Text(string(e), nil) }

func validateHistoryDate(value string) (string, error) {
	raw := pyStrip(value)
	m := hoi4DateRe.FindStringSubmatch(raw)
	if m == nil {
		if raw == "" {
			return "", historyDateError("history_date.empty")
		}
		return "", historyDateError("history_date.format")
	}
	year := atoiSafe(m[1])
	month := atoiSafe(m[2])
	day := atoiSafe(m[3])
	hour := int64(12)
	if m[4] != "" {
		hour = atoiSafe(m[4])
	}
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return "", historyDateError("history_date.range")
	}
	if hour < 0 || hour > 23 {
		return "", historyDateError("history_date.hour")
	}
	return raw, nil
}

func atoiSafe(s string) int64 {
	var n int64
	for i := 0; i < len(s); i++ {
		n = n*10 + int64(s[i]-'0')
	}
	return n
}

func renderHistoryDateBlocksTolerant(blocks []Value, ctag string) (string, []exportmsg.Message, error) {
	where := "history file"
	if c := pyStrip(ctag); c != "" {
		where = c + " history file"
	}
	var rendered []string
	var notices []exportmsg.Message
	for i, b := range blocks {
		bm, ok := b.(*OMap)
		if !ok {
			return "", nil, fmt.Errorf(
				"%s: dated block #%d is not a mapping, so the exporter cannot read it. "+
					"This is a decoding fault in the exporter, not something in your "+
					"project — please report it with the project id; the export was "+
					"stopped rather than ship a history file with dated blocks missing",
				where, i+1)
		}
		raw := eStr(bm, "date")
		date, err := validateHistoryDate(raw)
		if err != nil {
			reason, _ := err.(historyDateError)
			notices = append(notices, exportmsg.Message{Key: "politics.date_block_skipped", Args: exportmsg.Args{
				"subject": pyStrip(ctag), "number": i + 1, "reason": exportmsg.Message{Key: string(reason)},
			}})
			continue
		}
		script := eStr(bm, "script")
		rendered = append(rendered, date+" = {"+script+"}")
	}
	return strings.Join(rendered, "\n\n"), notices, nil
}

func renderHistoryDateBlocks(blocks []Value) string {
	out, _, err := renderHistoryDateBlocksTolerant(blocks, "")
	if err != nil {
		panic(err.Error())
	}
	return out
}

func clDescLocKeyWillBeEmitted(cl *OMap) bool {
	charV, _ := cl.Get("char")
	char, ok := charV.(*OMap)
	if !ok || char == nil {
		return false
	}
	if !pyTruthy(eGet(cl, "loc_emitted")) {
		return false
	}
	tag := eStrOrEmpty(char, "tag")
	if resolveDescriptionAnyLang(char) != "" {
		return true
	}
	name, gate := resolveMeaningfulDisplayName(char, "en", tag, "name_en", "name_zh", "name")
	return name != "" || gate
}

func emitHistoryFileLeaderReplacementBlock(countryTag string, clReplacements []*OMap) []string {
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
		lines = append(lines, "\tname = "+eStr(cl, "char_tag"))
		if clDescLocKeyWillBeEmitted(cl) {
			lines = append(lines, "\tdesc = "+eStr(cl, "char_tag")+"_desc")
		}
		picture := pyStrOrEmpty(eGet(cl, "picture_sprite"))
		if picture != "" {
			lines = append(lines, "\tpicture = \""+picture+"\"")
		}
		expire := pyStrOrEmpty(eGet(cl, "expire"))
		if expire == "" {
			expire = "1965.1.1"
		}
		lines = append(lines, "\texpire = \""+expire+"\"")
		ideology := pyStrOrEmpty(eGet(cl, "ideology"))
		if ideology == "" {
			ideology = "neutrality"
		}
		lines = append(lines, "\tideology = "+ideology)
		var traits []string
		for _, t := range eList(cl, "traits") {
			traits = append(traits, pyStr(t))
		}
		if len(traits) > 0 {
			lines = append(lines, "\ttraits = { "+strings.Join(traits, " ")+" }")
		}
		lines = append(lines, "}")
	}
	lines = append(lines, "")
	return lines
}
