package vanilladata

import (
	"context"
	"encoding/json"
	"sort"

	"haganeworks/internal/gameindex"
)

func idSet(blob []byte, key string) (map[string]struct{}, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(blob, &top); err != nil {
		return nil, err
	}
	var ids []string
	_ = json.Unmarshal(top[key], &ids)
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

type tokenSetKey string

func tokenSet(dataset, key string) map[string]struct{} {
	set, err := gameindex.Load(context.Background(), tokenSetKey(dataset), dataset, func(raw []byte) (map[string]struct{}, error) {
		return idSet(raw, key)
	})
	if err != nil {
		return map[string]struct{}{}
	}
	return set
}

func vanillaEffectsOnce() map[string]struct{} { return tokenSet(gameindex.VanillaEffects, "ids") }
func vanillaTriggersOnce() map[string]struct{} {
	return tokenSet(gameindex.VanillaTriggers, "ids")
}
func structuralFieldsOnce() map[string]struct{} {
	return tokenSet(gameindex.StructuralFields, "fields")
}
func yesNoKeysOnce() map[string]struct{} { return tokenSet(gameindex.YesNoKeys, "ids") }

func IsVanillaEffect(name string) bool {
	_, ok := vanillaEffectsOnce()[name]
	return ok
}

func IsVanillaTrigger(name string) bool {
	_, ok := vanillaTriggersOnce()[name]
	return ok
}

func IsStructuralField(name string) bool {
	_, ok := structuralFieldsOnce()[name]
	return ok
}

func IsVanillaYesNoKey(name string) bool {
	_, ok := yesNoKeysOnce()[name]
	return ok
}

func VanillaYesNoKeys() []string {
	set := yesNoKeysOnce()
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func VanillaEffectCount() int   { return len(vanillaEffectsOnce()) }
func VanillaTriggerCount() int  { return len(vanillaTriggersOnce()) }
func StructuralFieldCount() int { return len(structuralFieldsOnce()) }
func VanillaYesNoKeyCount() int { return len(yesNoKeysOnce()) }
