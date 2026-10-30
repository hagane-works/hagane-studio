package ideas

import (
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func ImportScriptedSnippets(text, kind, sourceFile string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("scripted_snippet")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	results := []*Dict{}
	for _, entry := range ast.Entries {
		if !hasKey(entry) || entry.Block == nil {
			continue
		}
		name := *entry.Key
		body := blockText(entry.Block)
		rec := NewDict()
		rec.Set("kind", kind)
		rec.Set("name", name)
		rec.Set("body", body)
		rec.Set("source_file", sourceFile)
		rec.Set("source", "imported")
		rec.Set("raw_extras", nil)
		results = append(results, rec)
		summary.AddPreserved(name)
	}

	summary.Total = len(results)
	return results, summary, nil
}

func ImportScriptedLocalisations(text, sourceFile string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("scripted_localisation")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	results := []*Dict{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key != "defined_text" || entry.Block == nil {
			continue
		}

		nameScalar := ppScalar(entry.Block.Entries, "name")
		name := ""
		if scalarTruthy(nameScalar) {
			name = pyStrip(importsvc.PyScalarStr(nameScalar))
		}
		if name == "" {

			continue
		}
		body := blockText(entry.Block)
		rec := NewDict()
		rec.Set("name", name)
		rec.Set("body", body)
		rec.Set("source_file", sourceFile)
		rec.Set("source", "imported")
		rec.Set("raw_extras", nil)
		results = append(results, rec)
		summary.AddPreserved(name)
	}

	summary.Total = len(results)
	return results, summary, nil
}

func agencySlug(s string) string {
	var b strings.Builder
	inRun := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			inRun = false
		} else if !inRun {
			b.WriteByte('_')
			inRun = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func ImportIntelligenceAgencies(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("intelligence_agency")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, nil, err
	}
	expandAtConstants(ast.Entries)

	agencyScalarKeys := map[string]bool{"picture": true}

	results := []*Dict{}
	usedIDs := map[string]bool{}

	synthAgencyID := func(names []any, picture string, idx int) string {
		seed := ""
		if len(names) > 0 {
			if s, ok := names[0].(string); ok {
				seed = agencySlug(pyLower(s))
			}
		}
		if seed == "" && picture != "" {
			s := strings.TrimPrefix(pyLower(picture), "gfx_intelligence_agency_logo_")
			seed = agencySlug(s)
		}
		if seed == "" {
			seed = "agency_" + strconv.Itoa(idx)
		}
		candidate := seed
		n := 2
		for usedIDs[candidate] {
			candidate = seed + "_" + strconv.Itoa(n)
			n++
		}
		usedIDs[candidate] = true
		return candidate
	}

	idx := 0
	for _, entry := range ast.Entries {
		if keyStr(entry) != "intelligence_agency" || entry.Block == nil {
			continue
		}
		idx++
		rawExtras := NewDict()
		rec := NewDict()
		rec.Set("agency_id", "")
		rec.Set("picture", "")
		rec.Set("names", []any{})
		rec.Set("default_trigger", "")
		rec.Set("available", "")
		rec.Set("upgrade_branches", []any{})
		rec.Set("raw_extras", rawExtras)

		for _, child := range entry.Block.Entries {
			ck := keyOrEmpty(child)
			if ck == "" {
				continue
			}
			isBlock := child.Block != nil
			switch {
			case agencyScalarKeys[ck] && !isBlock:
				rec.Set(ck, importsvc.PyScalarStr(child.Scalar))
			case ck == "names" && isBlock:
				rec.Set("names", ppBareValues(child.Block.Entries))
			case ck == "default" && isBlock:
				cur, _ := rec.Get("default_trigger")
				if s, _ := cur.(string); s != "" {
					accumulateRawExtra(rawExtras, ck, blockText(child.Block))
					summary.AddPreserved(ck)
				} else {
					rec.Set("default_trigger", blockText(child.Block))
				}
			case ck == "available" && isBlock:
				cur, _ := rec.Get("available")
				if s, _ := cur.(string); s != "" {

					accumulateRawExtra(rawExtras, ck, blockText(child.Block))
					summary.AddPreserved(ck)
				} else {
					rec.Set("available", blockText(child.Block))
				}
			default:
				if isBlock {
					accumulateRawExtra(rawExtras, ck, blockText(child.Block))
				} else {

					accumulateRawExtra(rawExtras, ck, child.Scalar)
				}
				summary.AddPreserved(ck)
			}
		}
		namesAny, _ := rec.Get("names")
		pictureAny, _ := rec.Get("picture")
		picture, _ := pictureAny.(string)
		rec.Set("agency_id", synthAgencyID(namesAny.([]any), picture, idx))
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}
