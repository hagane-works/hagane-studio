package guirender

import (
	"bufio"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type bitmapFontDef struct {
	Stem  string
	Color string

	Opacity float64
}

type fntData struct {
	Line, Base     int
	ScaleW, ScaleH int
	Page           string
	Tint           string
	Chars          map[int][7]int
	Kerning        map[string]int
}

func (idx *Index) indexBitmapFont(body []*paradox.Entry) {
	name := strings.ToLower(ciScalar(body, "name"))
	if name == "" {
		return
	}
	for _, entry := range body {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "languages") {
			continue
		}
		english := false
		for _, lang := range entry.Block.Entries {
			if lang != nil && lang.Key == nil && strings.EqualFold(lang.Scalar.StrValue(), "l_english") {
				english = true
			}
		}
		if !english {
			return
		}
	}
	stem := ciScalar(body, "path")
	if stem == "" {
		for _, entry := range body {
			if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "fontfiles") {
				continue
			}
			for _, file := range entry.Block.Entries {
				if file != nil && file.Key == nil && file.Scalar.StrValue() != "" {
					stem = file.Scalar.StrValue()
					break
				}
			}
		}
	}
	if stem == "" {
		return
	}
	idx.bitmapFonts[name] = bitmapFontDef{
		Stem:    foldTexturePath(strings.TrimSuffix(stem, path.Ext(stem))),
		Color:   fontColor(ciScalar(body, "color")),
		Opacity: fontOpacity(ciScalar(body, "color")),
	}
}

func fontColor(raw string) string {
	value := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "0x")
	if len(value) != 8 {
		return ""
	}
	if _, err := strconv.ParseUint(value, 16, 32); err != nil {
		return ""
	}
	rgb := value[2:]
	if rgb == "ffffff" {
		return ""
	}
	return "#" + rgb
}

func fontOpacity(raw string) float64 {
	value := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "0x")
	if len(value) != 8 {
		return 0
	}
	alpha, err := strconv.ParseUint(value[:2], 16, 8)
	if err != nil || alpha == 0xff {
		return 0
	}
	return float64(alpha) / 255
}

var (
	fntCommonRE = regexp.MustCompile(`lineHeight=(\d+)\s+base=(\d+)\s+scaleW=(\d+)\s+scaleH=(\d+)`)
	fntPageRE   = regexp.MustCompile(`^page\s+id=0\s+file="([^"]+)"`)
	fntChnlRE   = regexp.MustCompile(`alphaChnl=(\d+)\s+redChnl=(\d+)`)
	fntFieldRE  = regexp.MustCompile(`(\w+)=(-?\d+)`)
)

func parseFnt(body []byte) (*fntData, bool) {
	data := &fntData{Chars: map[int][7]int{}, Kerning: map[string]int{}, Tint: "luminance"}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "common "):
			if m := fntCommonRE.FindStringSubmatch(line); len(m) == 5 {
				data.Line, _ = strconv.Atoi(m[1])
				data.Base, _ = strconv.Atoi(m[2])
				data.ScaleW, _ = strconv.Atoi(m[3])
				data.ScaleH, _ = strconv.Atoi(m[4])
			}

			if m := fntChnlRE.FindStringSubmatch(line); len(m) == 3 && m[2] == "4" {
				data.Tint = "alpha"
			}
		case strings.HasPrefix(line, "page "):
			if m := fntPageRE.FindStringSubmatch(line); len(m) == 2 {
				data.Page = m[1]
			}
		case strings.HasPrefix(line, "char "):
			fields := map[string]int{}
			for _, m := range fntFieldRE.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[2])
				fields[m[1]] = n
			}
			id, ok := fields["id"]
			if !ok {
				continue
			}
			data.Chars[id] = [7]int{fields["x"], fields["y"], fields["width"], fields["height"], fields["xoffset"], fields["yoffset"], fields["xadvance"]}
		case strings.HasPrefix(line, "kerning "):
			fields := map[string]int{}
			for _, m := range fntFieldRE.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[2])
				fields[m[1]] = n
			}
			if amount := fields["amount"]; amount != 0 {
				data.Kerning[fmt.Sprintf("%d,%d", fields["first"], fields["second"])] = amount
			}
		}
	}
	if data.Line == 0 || len(data.Chars) == 0 {
		return nil, false
	}
	return data, true
}

func (idx *Index) bitmapFontDefFor(name string) (bitmapFontDef, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if def, ok := layer.bitmapFonts[name]; ok {
			return def, true
		}
	}
	return bitmapFontDef{}, false
}

func (idx *Index) fntFor(stem string) (*fntData, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if data, ok := layer.fnts[stem]; ok {
			return data, true
		}
	}
	return nil, false
}

func (idx *Index) bitmapFontPayload(name string) map[string]any {
	key := strings.ToLower(name)
	def, ok := idx.bitmapFontDefFor(key)
	if !ok {

		def = bitmapFontDef{Stem: "gfx/fonts/" + key}
	}
	data, ok := idx.fntFor(def.Stem)
	if !ok {
		return nil
	}
	if idx.vanillaBase() == "" {
		return nil
	}
	atlasURL := ""
	for _, candidate := range atlasCandidates(def.Stem, data.Page) {
		if _, _, rel, found := idx.textureSize(candidate); found && rel != "" {
			atlasURL = idx.vanillaTextureURL(rel)
			break
		}
	}
	if atlasURL == "" {
		return nil
	}
	chars := make(map[string][7]int, len(data.Chars))
	for id, glyph := range data.Chars {
		chars[strconv.Itoa(id)] = glyph
	}
	payload := map[string]any{
		"line": data.Line, "base": data.Base,
		"atlas": atlasURL, "atlas_w": data.ScaleW, "atlas_h": data.ScaleH,
		"tint": data.Tint, "chars": chars,
	}
	if def.Color != "" {
		payload["color"] = def.Color
	}
	if def.Opacity > 0 {
		payload["opacity"] = def.Opacity
	}
	if len(data.Kerning) > 0 {
		payload["kerning"] = data.Kerning
	}
	return payload
}

func atlasCandidates(stem, page string) []string {
	out := []string{}
	if page != "" {
		out = append(out, path.Dir(stem)+"/"+page)
	}
	return append(out, stem+".dds")
}
