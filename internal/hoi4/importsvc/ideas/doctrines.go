package ideas

import (
	"math/big"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

func doctrineStripQuote(s string) string {
	r := []rune(s)
	if len(r) >= 2 && r[0] == '"' && r[len(r)-1] == '"' {
		return string(r[1 : len(r)-1])
	}
	return s
}

func mapDoctrineBlock(entityType, key string, body []*paradox.Entry,
	sourceFile string, summary *importsvc.ImportSummary) (*Dict, error) {

	rawExtras := NewDict()
	rec := NewDict()
	rec.Set("entity_type", entityType)
	rec.Set("entity_key", key)
	rec.Set("folder_key", "")
	rec.Set("track_key", "")
	rec.Set("name", "")
	rec.Set("description", "")
	rec.Set("icon", "")
	rec.Set("xp_cost", big.NewInt(0))
	rec.Set("xp_type", "")
	rec.Set("xor", nil)
	rec.Set("allow_in_multiple_tracks", nil)
	rec.Set("raw_extras", rawExtras)
	rec.Set("source_file", sourceFile)
	rec.Set("source", "imported")
	rec.Set("display_name_locs", NewDict())
	rec.Set("description_locs", NewDict())

	for _, e := range body {
		ek := keyOrEmpty(e)
		if ek == "" {

			if e.Block == nil {
				accumulateRawExtra(rawExtras, "__bare__", importsvc.PyScalarStr(e.Scalar))
			}
			continue
		}
		isBlock := e.Block != nil

		if ek == "folder" && !isBlock {
			rec.Set("folder_key", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}
		if ek == "track" && !isBlock {
			rec.Set("track_key", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}

		if ek == "name" && !isBlock {
			rec.Set("name", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}
		if ek == "description" && !isBlock {
			rec.Set("description", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}
		if ek == "icon" && !isBlock {
			rec.Set("icon", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}
		if ek == "xp_type" && !isBlock {
			rec.Set("xp_type", doctrineStripQuote(importsvc.PyScalarStr(e.Scalar)))
			continue
		}
		if ek == "xp_cost" && !isBlock {
			sv := importsvc.PyScalarStr(e.Scalar)
			z, ck := intFloatStr(sv)
			switch ck {
			case coerceOverflowError:
				return nil, ErrOverflow
			case coerceOK:
				rec.Set("xp_cost", z)
			default:
				accumulateRawExtra(rawExtras, ek, sv)
			}
			continue
		}

		isSubdoctrine := entityType == "subdoctrine"
		if ek == "xor" && isBlock && isSubdoctrine {
			rec.Set("xor", blockText(e.Block))
			continue
		}
		if ek == "allow_in_multiple_tracks" && !isBlock && isSubdoctrine {
			rec.Set("allow_in_multiple_tracks",
				pyLower(pyStrip(importsvc.PyScalarStr(e.Scalar))) == "yes")
			continue
		}

		if isBlock {
			accumulateRawExtra(rawExtras, ek, blockText(e.Block))
		} else {
			accumulateRawExtra(rawExtras, ek, importsvc.ScalarToText(e))
		}
		summary.AddPreserved(ek)
	}
	return rec, nil
}

func ImportDoctrines(text, entityType, sourceFile string) ([]*Dict, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("doctrine")
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
		rec, mErr := mapDoctrineBlock(entityType, *entry.Key, entry.Block.Entries,
			sourceFile, summary)
		if mErr != nil {
			return nil, nil, mErr
		}
		results = append(results, rec)
	}

	summary.Total = len(results)
	return results, summary, nil
}
