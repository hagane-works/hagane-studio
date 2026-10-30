package miscstructured

import (
	"fmt"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var idRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

func scalar(e *paradox.Entry) string {
	if e == nil || e.Block != nil {
		return ""
	}
	return importsvc.ScalarToText(e)
}
func block(e *paradox.Entry) string {
	if e == nil || e.Block == nil {
		return ""
	}
	return importsvc.BlockToText(e.Block, 0)
}

func braceBlock(e *paradox.Entry) string {
	if e == nil || e.Block == nil {
		return ""
	}
	body := importsvc.ToScript(e.Block, 1)
	if body == "" {
		return "{\n}"
	}
	return "{\n" + body + "\n}"
}
func parse(text string) (*paradox.Block, error) {
	b, rec, err := paradox.Parse(text)
	if err != nil {
		return nil, err
	}
	if len(rec) > 0 {
		return nil, fmt.Errorf("parser recovered malformed script")
	}
	return b, nil
}

func ModifierDefinitions(text, source string) ([]map[string]any, error) {
	b, e := parse(text)
	if e != nil {
		return nil, e
	}
	var out []map[string]any
	for _, x := range b.Entries {
		if x.Key == nil || x.Block == nil {
			return nil, fmt.Errorf("expected modifier block")
		}
		id := *x.Key
		if !idRE.MatchString(id) {
			return nil, fmt.Errorf("invalid modifier id %q", id)
		}
		r := map[string]any{"modifier_id": id, "color_type": "bad", "value_type": "number", "precision": int64(2), "postfix": "none", "categories": []string{}, "explicit_fields": []string{}, "source_file": source}
		seen := map[string]bool{}
		for _, c := range x.Block.Entries {
			if c.Key == nil || c.Block != nil {
				return nil, fmt.Errorf("%s: nested/empty field", id)
			}
			k := *c.Key
			if k != "category" && seen[k] {
				return nil, fmt.Errorf("%s: duplicate %s", id, k)
			}
			seen[k] = true
			switch k {
			case "color_type", "value_type", "postfix":
				r[k] = scalar(c)
			case "precision":
				var n int64
				if _, err := fmt.Sscan(scalar(c), &n); err != nil || n < 0 || n > 3 {
					return nil, fmt.Errorf("%s: invalid precision", id)
				}
				r[k] = n
			case "category":
				r["categories"] = append(r["categories"].([]string), scalar(c))
			default:
				return nil, fmt.Errorf("%s: unsupported %s", id, k)
			}
			if k != "category" || len(r["categories"].([]string)) == 1 {
				r["explicit_fields"] = append(r["explicit_fields"].([]string), k)
			}
		}
		if len(r["categories"].([]string)) == 0 {
			r["categories"] = []string{"all"}
		}
		out = append(out, r)
	}
	return out, nil
}

func aiTemplateField(k string) bool {
	switch k {
	case "role", "front_role_override", "blocked_for", "available_for", "upgrade_prio", "enable":
		return true
	}
	return false
}

func AIPlans(text, source string) ([]map[string]any, error)     { return ai(text, source, false) }
func AITemplates(text, source string) ([]map[string]any, error) { return ai(text, source, true) }
func ai(text, source string, template bool) ([]map[string]any, error) {
	b, e := parse(text)
	if e != nil {
		return nil, e
	}
	var out []map[string]any
	for _, x := range b.Entries {
		if x.Key == nil || x.Block == nil {
			return nil, fmt.Errorf("top-level definition must be block")
		}
		r := map[string]any{"source_file": source, "raw_block": importsvc.ToScript(&paradox.Block{Entries: []*paradox.Entry{x}}, 0), "field_order": []string{}}
		if template {
			r["template_id"] = *x.Key
			r["role"] = ""
			r["blocked_for"] = ""
			r["available_for"] = ""
			r["upgrade_prio"] = ""
			r["enable"] = ""
			r["front_role_override"] = ""
			r["target_templates"] = []map[string]string{}
		} else {
			r["plan_id"] = *x.Key
			r["name"] = ""
			r["desc"] = ""
			for _, k := range []string{"allowed", "enable", "abort", "ai_national_focuses", "focus_factors", "research", "ideas", "traits", "ai_strategy", "weight"} {
				r[k] = []string{}
			}
		}
		for _, c := range x.Block.Entries {
			if c.Key == nil {
				return nil, fmt.Errorf("bare value")
			}
			k := *c.Key
			marker := k
			if template && !aiTemplateField(k) {
				marker = "target:" + k
			}
			r["field_order"] = append(r["field_order"].([]string), marker)
			if template {
				switch k {
				case "role", "front_role_override":
					if c.Block != nil {
						return nil, fmt.Errorf("%s scalar", k)
					}
					r[k] = scalar(c)
				case "blocked_for", "available_for", "upgrade_prio", "enable":
					if c.Block == nil {
						return nil, fmt.Errorf("%s block", k)
					}
					r[k] = braceBlock(c)
				default:
					if c.Block == nil {
						return nil, fmt.Errorf("unsupported %s", k)
					}
					r["target_templates"] = append(r["target_templates"].([]map[string]string), map[string]string{"id": k, "body": braceBlock(c)})
				}
			} else {
				switch k {
				case "name", "desc":
					r[k] = scalar(c)
				case "allowed", "enable", "abort", "ai_national_focuses", "focus_factors", "research", "ideas", "traits", "ai_strategy", "weight":
					if c.Block == nil {
						return nil, fmt.Errorf("%s block", k)
					}
					r[k] = append(r[k].([]string), braceBlock(c))
				default:
					return nil, fmt.Errorf("unsupported plan field %s", k)
				}
			}
		}
		if template && (r["role"] == "" || len(r["target_templates"].([]map[string]string)) == 0) {
			return nil, fmt.Errorf("template missing role/targets")
		}
		out = append(out, r)
	}
	return out, nil
}

func Wargoals(text, source string) ([]map[string]any, error) {
	b, e := parse(text)
	if e != nil {
		return nil, e
	}
	if len(b.Entries) != 1 || b.Entries[0].Key == nil || *b.Entries[0].Key != "wargoal_types" || b.Entries[0].Block == nil {
		return nil, fmt.Errorf("missing wargoal_types wrapper")
	}
	var out []map[string]any
	for _, d := range b.Entries[0].Block.Entries {
		if d.Key == nil || d.Block == nil {
			return nil, fmt.Errorf("invalid definition")
		}
		r := map[string]any{"wargoal_key": *d.Key, "war_name": "", "safe_scalars": map[string]string{}, "allowed": "", "available": "", "take_states": "", "puppet": "", "liberate": "", "ordered_entries": []map[string]string{}, "definition_leading_comments": []any{}, "definition_trailing_comments": []any{}, "file_trivia": map[string]any{}, "source_file": source}
		seen := map[string]bool{}
		for _, x := range d.Block.Entries {
			if x.Key == nil {
				return nil, fmt.Errorf("bare value")
			}
			k := *x.Key
			modeled := k == "war_name" || isSafeWargoalScalar(k) || k == "allowed" || k == "available" || k == "take_states" || k == "puppet" || k == "liberate"
			if modeled && seen[k] {
				return nil, fmt.Errorf("duplicate %s", k)
			}
			if modeled {
				seen[k] = true
			}
			script := importsvc.ToScript(&paradox.Block{Entries: []*paradox.Entry{x}}, 0)
			r["ordered_entries"] = append(r["ordered_entries"].([]map[string]string), map[string]string{"key": k, "script": strings.TrimSpace(script)})
			switch k {
			case "war_name":
				r[k] = scalar(x)
			case "allowed", "available", "take_states", "puppet", "liberate":
				if x.Block == nil {
					return nil, fmt.Errorf("%s block", k)
				}
				r[k] = block(x)
			default:
				if isSafeWargoalScalar(k) {
					if x.Block != nil {
						return nil, fmt.Errorf("%s scalar", k)
					}
					r["safe_scalars"].(map[string]string)[k] = scalar(x)
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func isSafeWargoalScalar(k string) bool {
	switch k {
	case "generate_base_cost", "generate_per_state_cost", "take_states_limit", "take_states_cost", "puppet_cost", "force_government_cost", "expire", "threat", "take_states_threat_factor":
		return true
	}
	return false
}
