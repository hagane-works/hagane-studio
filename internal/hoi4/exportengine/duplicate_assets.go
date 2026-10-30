package exportengine

import (
	"crypto/sha256"
	"encoding/hex"

	"haganeworks/internal/hoi4/exportmsg"
)

func DuplicatePrivateAssets(entries []Entry, msg exportmsg.Printer) []string {
	type seen struct {
		path   string
		source string
	}
	first := map[string]seen{}
	var warnings []string
	for _, entry := range entries {
		if entry.Origin != "private_asset" || len(entry.Bytes) == 0 {
			continue
		}

		if entry.SourceEntity == "" {
			continue
		}
		sum := sha256.Sum256(entry.Bytes)
		digest := hex.EncodeToString(sum[:])
		prev, ok := first[digest]
		if !ok {
			first[digest] = seen{path: entry.Path, source: entry.SourceEntity}
			continue
		}
		if prev.source == entry.SourceEntity {

			continue
		}
		warnings = append(warnings, msg.Text("duplicate.image", exportmsg.Args{
			"first": prev.path, "first_source": prev.source, "second": entry.Path, "second_source": entry.SourceEntity,
		}))
	}
	return warnings
}
