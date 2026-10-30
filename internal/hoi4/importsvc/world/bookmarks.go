package world

import (
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var bookmarkKnownKeys = map[string]bool{
	"name": true, "desc": true, "date": true, "picture": true,
	"default_country": true, "default": true,
}

var bookmarkCountryKnownKeys = map[string]bool{
	"history": true, "minor": true, "ideology": true, "ideas": true,
	"focuses": true,
}

func recoverBookmarkIDFromLocKey(locKey string) (bookmarkID, display string) {
	raw := pyStripChar(pyStrip(locKey), '"')
	if raw == "" {
		return "", ""
	}
	if strings.HasSuffix(pyUpper(raw), "_NAME") && len([]rune(raw)) > 5 {
		rs := []rune(raw)
		return pyLower(string(rs[:len(rs)-5])), ""
	}
	return pyLower(raw), raw
}

func stripLocSuffix(locKey, suffix string) string {
	raw := pyStripChar(pyStrip(locKey), '"')
	if raw == "" {
		return ""
	}
	if strings.HasSuffix(pyUpper(raw), pyUpper(suffix)) {
		return ""
	}
	return raw
}

func strsToAny(ss []string) []any {
	out := make([]any, 0, len(ss))
	for _, s := range ss {
		out = append(out, s)
	}
	return out
}

func mapBookmarkCountryBlock(ctag string, body *paradox.Block, summary *importsvc.ImportSummary) *PyDict {
	sc := NewPyDict()
	sc.Set("tag", ctag)
	sc.Set("history_en", "")
	sc.Set("history_zh", "")
	sc.Set("minor", false)
	sc.Set("raw_extras", NewPyDict())
	for _, ent := range body.Entries {
		switch {
		case keyIs(ent, "history"):
			sc.Set("history_en", stripLocSuffix(pyStr(entVal(ent)), "_DESC"))
		case keyIs(ent, "minor"):
			sc.Set("minor", pyLower(pyStrip(pyStr(entVal(ent)))) == "yes")
		case keyIs(ent, "ideology"):
			sc.Set("ideology", pyStrip(pyStr(entVal(ent))))
		case keyIs(ent, "ideas") && ent.Block != nil:
			sc.Set("ideas", strsToAny(ppBareValues(ent.Block.Entries)))
		case keyIs(ent, "focuses") && ent.Block != nil:
			sc.Set("focuses", strsToAny(ppBareValues(ent.Block.Entries)))
		case keyIs(ent, "label") && ent.Block != nil:
			sc.Set("label", strsToAny(ppBareValues(ent.Block.Entries)))
		case keyIs(ent, "available"):
			if ent.Block != nil {
				sc.Set("available", importsvc.BlockToText(ent.Block, 0))
			} else {
				sc.Set("available", pyStrip(pyStr(entVal(ent))))
			}
			summary.AddPreserved("bookmark_country.available")
		case ent.Key != nil && *ent.Key != "" && !bookmarkCountryKnownKeys[*ent.Key]:
			re := sc.MustGet("raw_extras").(*PyDict)
			if ent.Block != nil {
				re.Accumulate(*ent.Key, importsvc.BlockToText(ent.Block, 0))
			} else {
				re.Accumulate(*ent.Key, scalarAny(ent.Scalar))
			}
			summary.AddPreserved("bookmark_country." + *ent.Key)
		}
	}
	return sc
}

func mapOneBookmark(body *paradox.Block, summary *importsvc.ImportSummary) *PyDict {
	data := NewPyDict()
	data.Set("bookmark_id", "")
	data.Set("name_en", "")
	data.Set("name_zh", "")
	data.Set("desc_en", "")
	data.Set("desc_zh", "")
	data.Set("date", "1936.1.1.12")
	data.Set("picture", "GFX_select_date_1936")
	data.Set("default_country", "")
	data.Set("default_flag", false)
	data.Set("selectable_countries", []any{})
	data.Set("filters", []any{})
	data.Set("raw_extras", NewPyDict())
	descLocKey := ""

	for _, ent := range body.Entries {
		switch {
		case keyIs(ent, "name"):
			nameLocKey := pyStripChar(pyStrip(pyStr(entVal(ent))), '"')
			bmID, display := recoverBookmarkIDFromLocKey(nameLocKey)
			if bmID != "" {
				data.Set("bookmark_id", bmID)
			}
			if display != "" {
				data.Set("name_en", display)
			}
		case keyIs(ent, "desc"):
			descLocKey = pyStripChar(pyStrip(pyStr(entVal(ent))), '"')
			data.Set("desc_en", stripLocSuffix(descLocKey, "_DESC"))
		case keyIs(ent, "date"):
			data.Set("date", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "picture"):
			data.Set("picture", pyStripChar(pyStrip(pyStr(entVal(ent))), '"'))
		case keyIs(ent, "default_country"):
			data.Set("default_country", pyUpper(pyStripChar(pyStrip(pyStr(entVal(ent))), '"')))
		case keyIs(ent, "default"):
			data.Set("default_flag", pyLower(pyStrip(pyStr(entVal(ent)))) == "yes")
		case keyIs(ent, "filters") && ent.Block != nil:
			data.Set("filters", strsToAny(ppBareValues(ent.Block.Entries)))
		case ent.Key != nil && *ent.Key != "" && ent.Block != nil:
			ctag := pyUpper(pyStripChar(pyStrip(*ent.Key), '"'))
			if !IsTagShape(ctag) {
				data.MustGet("raw_extras").(*PyDict).Accumulate(
					*ent.Key, importsvc.BlockToText(ent.Block, 0))
				summary.AddPreserved("bookmark." + *ent.Key)
				continue
			}
			sc := mapBookmarkCountryBlock(ctag, ent.Block, summary)
			if sc != nil {
				data.appendTo("selectable_countries", sc)
			}
		case ent.Key != nil && *ent.Key != "" && !bookmarkKnownKeys[*ent.Key]:
			data.MustGet("raw_extras").(*PyDict).Accumulate(*ent.Key, scalarAny(ent.Scalar))
			summary.AddPreserved("bookmark." + *ent.Key)
		}
	}

	if data.MustGet("bookmark_id").(string) == "" && descLocKey != "" {
		arg := descLocKey
		if strings.HasSuffix(pyUpper(descLocKey), "_DESC") {
			rs := []rune(descLocKey)
			arg = string(rs[:len(rs)-5]) + "_NAME"
		}
		bmID, _ := recoverBookmarkIDFromLocKey(arg)
		if bmID != "" {
			data.Set("bookmark_id", bmID)
		}
	}

	if data.MustGet("bookmark_id").(string) == "" {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"bookmark_no_name",
			nil,
			"Skipped a bookmark that had no usable name — nothing else was affected.",
		))
		return nil
	}
	return data
}

func ImportBookmarks(text string) ([]*PyDict, *importsvc.ImportSummary) {
	summary := importsvc.NewImportSummary("bookmark")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		summary.Warnings = append(summary.Warnings, importsvc.ParseSkipWarning(summary, ""))
		return []*PyDict{}, summary
	}
	expandAtConstants(ast)

	results := []*PyDict{}
	for _, entry := range ast.Entries {
		if !keyIs(entry, "bookmarks") || entry.Block == nil {
			continue
		}
		for _, bmEntry := range ppFindAll(entry.Block.Entries, "bookmark") {
			if bmEntry.Block == nil {
				continue
			}
			if data := mapOneBookmark(bmEntry.Block, summary); data != nil {
				results = append(results, data)
			}
		}
	}

	summary.Total = len(results)
	return results, summary
}
