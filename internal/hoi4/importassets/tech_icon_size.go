package importassets

import (
	"context"
	"encoding/json"
	"strings"

	"haganeworks/internal/gameindex"
)

type techIconSizesKey struct{}

func vanillaTechIconSizes() map[string]Size {
	sizes, err := gameindex.Load(context.Background(), techIconSizesKey{}, gameindex.TechIconSizes, func(raw []byte) (map[string]Size, error) {
		var table struct {
			Sizes map[string][2]int `json:"sizes"`
		}
		if err := json.Unmarshal(raw, &table); err != nil {
			return nil, err
		}
		out := make(map[string]Size, len(table.Sizes))
		for id, wh := range table.Sizes {
			out[strings.ToLower(id)] = Size{Width: wh[0], Height: wh[1]}
		}
		return out, nil
	})
	if err != nil {
		return map[string]Size{}
	}
	return sizes
}

func TechIconSize(techID string) (size Size, base bool) {
	if size, ok := vanillaTechIconSizes()[strings.ToLower(strings.TrimSpace(techID))]; ok {
		return size, true
	}
	return CanonicalSizes["tech_icon"], false
}
