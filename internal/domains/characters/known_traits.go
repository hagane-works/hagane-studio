package characters

import (
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type knownTraitMetadata struct {
	Types      []string `json:"types"`
	TraitTypes []string `json:"trait_types"`
}

type knownCustomTrait struct {
	TraitID  string             `json:"trait_id"`
	NameEn   string             `json:"name_en"`
	NameZh   string             `json:"name_zh"`
	Metadata knownTraitMetadata `json:"metadata"`
}

type knownTraitsResponse struct {
	Vanilla         []string                      `json:"vanilla"`
	VanillaMetadata map[string]knownTraitMetadata `json:"vanilla_metadata"`
	Custom          []knownCustomTrait            `json:"custom"`
}

const knownTraitTypesDetail = "trait_type must be one of ['country_leader', 'scientist', 'unit_leader']"

func knownTraitsHandler(deps domains.Deps) http.HandlerFunc {
	return apiutil.Handler(deps.Open, func(p *project.Project, w http.ResponseWriter, r *http.Request) error {
		traitType := "country_leader"
		if values, present := r.URL.Query()["trait_type"]; present {
			traitType = ""
			if len(values) > 0 {

				traitType = values[len(values)-1]
			}
		}
		if !validTraitTypes[traitType] {
			return detail(http.StatusBadRequest, knownTraitTypesDetail)
		}
		rows, err := p.Rows(traitTable)
		if err != nil {
			return err
		}
		var custom []project.Row
		for _, row := range rows {
			if str(row["trait_type"]) == traitType {
				custom = append(custom, row)
			}
		}
		sort.SliceStable(custom, func(i, j int) bool { return str(custom[i]["trait_id"]) < str(custom[j]["trait_id"]) })
		catalog := gameTraitCatalog(deps.Game)
		apiutil.JSON(w, http.StatusOK, buildKnownTraits(traitType, catalog, custom))
		return nil
	})
}

func buildKnownTraits(traitType string, catalog *traitCatalog, custom []project.Row) knownTraitsResponse {
	response := knownTraitsResponse{
		Vanilla:         catalog.idsByType[traitType],
		VanillaMetadata: catalog.metadataByType[traitType],
		Custom:          make([]knownCustomTrait, 0, len(custom)),
	}
	if response.Vanilla == nil {
		response.Vanilla = []string{}
	}
	if response.VanillaMetadata == nil {
		response.VanillaMetadata = map[string]knownTraitMetadata{}
	}
	for _, row := range custom {
		var extras map[string]any
		decodeColumn(row["extra_fields"], &extras)
		response.Custom = append(response.Custom, knownCustomTrait{
			TraitID: str(row["trait_id"]), NameEn: str(row["name_en"]), NameZh: str(row["name_zh"]),
			Metadata: knownCustomTraitMetadata(traitType, extras),
		})
	}
	return response
}

func knownCustomTraitMetadata(traitType string, extras map[string]any) knownTraitMetadata {
	metadata := knownTraitMetadata{Types: []string{}, TraitTypes: []string{}}
	if traitType != "unit_leader" {
		return metadata
	}
	metadata.Types = []string{"land"}
	if rawType, ok := extras["type"].(string); ok {
		if value := strings.TrimSpace(rawType); value != "" {
			metadata.Types = []string{value}
		}
	}
	if rawTraitType, ok := extras["trait_type"].(string); ok {
		if value := strings.TrimSpace(rawTraitType); value != "" {
			metadata.TraitTypes = []string{value}
		}
	}
	return metadata
}

type traitCatalog struct {
	idsByType      map[string][]string
	metadataByType map[string]map[string]knownTraitMetadata
}

type traitCatalogKey struct{}

func gameTraitCatalog(game func() (fs.FS, error)) *traitCatalog {
	g, err := gamefiles.Open(game)
	if err != nil {
		return &traitCatalog{}
	}
	return gamefiles.Derived(g, traitCatalogKey{}, extractGameTraits)
}

func extractGameTraits(game *gamefiles.Game) *traitCatalog {
	catalog := &traitCatalog{
		idsByType:      map[string][]string{},
		metadataByType: map[string]map[string]knownTraitMetadata{},
	}
	sources := []struct{ dir, traitType string }{
		{"common/country_leader", "country_leader"},
		{"common/unit_leader", "unit_leader"},
		{"common/scientist_traits", "scientist"},
	}
	unitMeta := map[string]*traitMetaSets{}
	for _, source := range sources {
		ids := map[string]bool{}
		for _, name := range game.Files(source.dir, false, ".txt") {
			if source.traitType == "unit_leader" && strings.Contains(path.Base(name), "_skills") {
				continue
			}
			data, err := game.ReadFile(name)
			if err != nil {
				continue
			}
			collectTraits(string(data), source.traitType, ids, unitMeta)
		}
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		sort.Strings(list)
		catalog.idsByType[source.traitType] = list
	}
	metadata := make(map[string]knownTraitMetadata, len(unitMeta))
	for id, sets := range unitMeta {
		metadata[id] = knownTraitMetadata{Types: sortedSet(sets.types), TraitTypes: sortedSet(sets.traitTypes)}
	}
	catalog.metadataByType["unit_leader"] = metadata
	return catalog
}

type traitMetaSets struct{ types, traitTypes map[string]bool }

func collectTraits(text, traitType string, ids map[string]bool, unitMeta map[string]*traitMetaSets) {
	ast, _, err := paradox.Parse(text)
	if err != nil || ast == nil {
		return
	}
	collect := func(entries []*paradox.Entry) {
		for _, e := range entries {
			if e.Key == nil || e.Block == nil {
				continue
			}
			key := *e.Key
			ids[key] = true
			if traitType != "unit_leader" {
				continue
			}
			meta := unitMeta[key]
			if meta == nil {
				meta = &traitMetaSets{types: map[string]bool{}, traitTypes: map[string]bool{}}
				unitMeta[key] = meta
			}
			for _, child := range e.Block.Entries {
				if child.Key == nil || (*child.Key != "type" && *child.Key != "trait_type") {
					continue
				}
				target := meta.types
				if *child.Key == "trait_type" {
					target = meta.traitTypes
				}
				var values []string
				if child.Block != nil {
					for _, item := range child.Block.Entries {
						if item.Key == nil && item.Block == nil {
							values = append(values, item.Scalar.StrValue())
						}
					}
				} else {
					values = []string{child.Scalar.StrValue()}
				}
				for _, value := range values {
					if value = strings.TrimSpace(value); value != "" {
						target[value] = true
					}
				}
			}
		}
	}
	wrapped := false
	for _, e := range ast.Entries {
		if e.Key == nil || e.Block == nil {
			continue
		}
		switch *e.Key {
		case "leader_traits", "scientist_traits", "unit_leader_traits":
			collect(e.Block.Entries)
			wrapped = true
		}
	}
	if !wrapped {
		collect(ast.Entries)
	}
}

func sortedSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
