package exportengine

import (
	"encoding/json"
	"strconv"
	"strings"

	emitchars "haganeworks/internal/hoi4/emit/characters"
)

func nativeTraitOverlays(snapshot Snapshot, modDir string, includeImported bool, characterEntries []Entry) ([]Entry, map[string]string, error) {
	availableIDs := map[string]bool{}
	for _, entry := range characterEntries {
		if strings.HasPrefix(entry.SourceEntity, "character_portrait:") ||
			strings.HasPrefix(entry.SourceEntity, "character_advisor:") {
			availableIDs[entry.SourceEntity] = true
		}
	}
	availableKeys := map[string]bool{}
	for _, raw := range snapshot.Tables["entity_obs_paths"] {
		var row struct {
			EntityType string `json:"entity_type"`
			EntityID   int64  `json:"entity_id"`
			OBSKey     string `json:"obs_key"`
		}
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		prefix := ""
		switch row.EntityType {
		case "character_portrait":
			prefix = "character_portrait:"
		case "character_advisor":
			prefix = "character_advisor:"
		}
		if prefix != "" && availableIDs[prefix+strconv.FormatInt(row.EntityID, 10)] {
			availableKeys[row.OBSKey] = true
		}
	}
	result, err := emitchars.EmitTraitOverlayRows(emitchars.TraitOverlayRows{
		Project: snapshot.Project, Characters: snapshot.Tables["characters"],
		CustomTraits: snapshot.Tables["custom_traits"], EventNamespaces: snapshot.Tables["event_namespaces"],
		EntityOBSPaths: snapshot.Tables["entity_obs_paths"], IncludeImported: includeImported,
		OBSAvailable: func(key string) (bool, error) {
			return availableKeys[key], nil
		},
	})
	if err != nil {
		return nil, nil, err
	}
	outputs := result.TextOutputs()
	entries := make([]Entry, 0, len(outputs))
	for _, output := range outputs {
		entries = append(entries, Entry{
			Path: modDir + "/" + strings.TrimLeft(output.Path, "/"), Bytes: []byte(output.Text),
			Origin: "generated", SourceEntity: "character_trait_overlays",
		})
	}
	return entries, result.TraitOverlayNamespaces(), nil
}
