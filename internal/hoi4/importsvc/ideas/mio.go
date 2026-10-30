package ideas

import (
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var mioScalarKeys = map[string]bool{"include": true, "icon": true}
var mioScriptKeys = map[string]bool{"allowed": true, "available": true, "visible": true}

func mapTrait(kind string, body []*paradox.Entry) *Dict {
	trait := NewDict()
	trait.Set("kind", kind)
	trait.Set("raw_extras", NewDict())
	for _, e := range body {
		ek := keyOrEmpty(e)
		if ek == "" {
			continue
		}
		if e.Block != nil {

			if reAny, ok := trait.Get("raw_extras"); ok {
				if re, dok := reAny.(*Dict); dok {
					accumulateRawExtra(re, ek, blockText(e.Block))
				}
			}
		} else {
			trait.Set(ek, e.Scalar)
		}
	}
	return trait
}

func ImportMIO(text string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("mio")
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
		rawExtras := NewDict()
		rec := NewDict()
		rec.Set("mio_id", *entry.Key)
		rec.Set("include", "")
		rec.Set("icon", "")
		rec.Set("equipment_type", []any{})
		rec.Set("research_categories", []any{})
		rec.Set("allowed", "")
		rec.Set("available", "")
		rec.Set("visible", "")
		rec.Set("traits", []any{})
		rec.Set("raw_extras", rawExtras)

		appendTrait := func(kind string, body []*paradox.Entry) {
			listAny, _ := rec.Get("traits")
			rec.Set("traits", append(listAny.([]any), mapTrait(kind, body)))
		}

		for _, child := range entry.Block.Entries {
			ck := keyOrEmpty(child)
			if ck == "" {
				continue
			}
			isBlock := child.Block != nil
			switch {
			case mioScalarKeys[ck] && !isBlock:
				rec.Set(ck, importsvc.PyScalarStr(child.Scalar))
			case mioScriptKeys[ck] && isBlock:
				rec.Set(ck, blockText(child.Block))

				if ck == "allowed" {
					rec.Set("_scope_verdict", importsvc.ClassifyScope(
						[][]*paradox.Entry{child.Block.Entries}))
				}
			case (ck == "equipment_type" || ck == "research_categories") && isBlock:

				vals := []any{}
				for _, c := range child.Block.Entries {
					if c.Key == nil && c.Block == nil {
						vals = append(vals, importsvc.PyScalarStr(c.Scalar))
					}
				}
				rec.Set(ck, vals)
			case ck == "initial_trait" && isBlock:
				appendTrait("initial", child.Block.Entries)
			case ck == "override_trait" && isBlock:
				appendTrait("override", child.Block.Entries)
			case (ck == "add_trait" || ck == "trait") && isBlock:
				appendTrait("add", child.Block.Entries)
			default:
				if isBlock {
					accumulateRawExtra(rawExtras, ck, blockText(child.Block))
				} else {

					accumulateRawExtra(rawExtras, ck, child.Scalar)
				}
				summary.AddPreserved(ck)
			}
		}
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}
