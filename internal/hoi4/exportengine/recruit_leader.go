package exportengine

import (
	"encoding/json"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/emit/worldout"
)

func BuildRecruitAndLeaderMaps(snapshot Snapshot, includeImported bool) (recruitByCtag, clByCtag *worldout.OMap, err error) {
	recruitByCtag = worldout.NewOMap()

	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, nil, err
	}
	primaryTag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if primaryTag == "" {
		primaryTag = "UNK"
	}

	for _, raw := range snapshot.Tables["characters"] {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, nil, err
		}

		if (!includeImported && entity.Str("source") == "imported") || entity.Bool("replaces_vanilla_id", false) {
			continue
		}
		token := strings.TrimSpace(entity.Str("tag"))
		if token == "" {
			continue
		}
		ctag := strings.ToUpper(strings.TrimSpace(entity.Str("country_tag")))
		if ctag == "" {
			ctag = primaryTag
		}
		existing, _ := recruitByCtag.Get(ctag)
		list, _ := existing.([]worldout.Value)
		recruitByCtag.Set(ctag, append(list, worldout.Value(token)))
	}

	clByCtag, err = buildClReplacementsByCtag(snapshot, primaryTag, includeImported)
	if err != nil {
		return nil, nil, err
	}
	return recruitByCtag, clByCtag, nil
}

func buildClReplacementsByCtag(snapshot Snapshot, primaryTag string, includeImported bool) (*worldout.OMap, error) {
	clByCtag := worldout.NewOMap()
	for _, raw := range snapshot.Tables["characters"] {
		row := worldout.NewOMap()
		if err := json.Unmarshal(raw, row); err != nil {
			return nil, err
		}
		if !omBool(row, "replaces_vanilla_id") {
			continue
		}
		if omStr(row, "role_type") != "country_leader" {
			continue
		}

		if v, ok := row.Get("source_snapshot"); ok && v != nil {
			continue
		}
		if omStr(row, "source") == "imported" {
			continue
		}

		if extras, _ := omMap(row, "raw_extras"); extras != nil {
			if v, ok := extras.Get("emit_trait_overlay_event"); ok {
				if b, isBool := v.(bool); isBool && !b {
					continue
				}
			}
		}
		charTag := strings.TrimSpace(omStr(row, "tag"))
		if charTag == "" {

			continue
		}
		ctag := strings.ToUpper(strings.TrimSpace(omStr(row, "country_tag")))
		if ctag == "" {
			ctag = primaryTag
		}

		ideology := omStr(row, "ideology")
		if ideology == "" {
			ideology = "neutrality"
		}
		expire := omStr(row, "expire")
		if expire == "" {
			expire = "1965.1.1"
		}

		cl := worldout.NewOMap()

		cl.Set("char_tag", charTag)

		cl.Set("char", row)
		cl.Set("loc_emitted", nativeCharacterLocEmitted(row, includeImported))
		cl.Set("ideology", ideology)
		cl.Set("traits", clTraits(row))
		cl.Set("expire", expire)
		cl.Set("picture_sprite", clPictureSprite())

		existing, _ := clByCtag.Get(ctag)
		list, _ := existing.([]worldout.Value)
		clByCtag.Set(ctag, append(list, worldout.Value(cl)))
	}
	return clByCtag, nil
}

func clTraits(row *worldout.OMap) []worldout.Value {
	v, ok := row.Get("traits")
	if !ok {
		return nil
	}
	list, _ := v.([]worldout.Value)
	seen := map[string]bool{}
	out := []worldout.Value{}
	for _, item := range list {
		s, isStr := item.(string)
		if !isStr || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, worldout.Value(s))
	}
	return out
}

func nativeCharacterLocEmitted(row *worldout.OMap, includeImported bool) bool {
	return includeImported || omStr(row, "source") != "imported"
}

func clPictureSprite() string { return "" }

func omStr(o *worldout.OMap, key string) string {
	v, ok := o.Get(key)
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func omBool(o *worldout.OMap, key string) bool {
	v, _ := o.Get(key)
	b, _ := v.(bool)
	return b
}

func omMap(o *worldout.OMap, key string) (*worldout.OMap, bool) {
	v, ok := o.Get(key)
	if !ok || v == nil {
		return nil, false
	}
	m, isMap := v.(*worldout.OMap)
	return m, isMap
}
