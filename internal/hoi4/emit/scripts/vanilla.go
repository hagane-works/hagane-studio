package emitscripts

import (
	"encoding/json"
	"os"
)

func LoadVanillaTraits(path string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return out
	}
	for _, id := range payload.IDs {
		out[id] = true
	}
	return out
}
