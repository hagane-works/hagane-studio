package guirender

import (
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type namesGroup struct {
	Key       string
	Name      string
	Countries []string
	CanUse    *paradox.Block
	Fallback  string

	Ordered []numberedName
	Order   int
}

type numberedName struct {
	Number int
	Name   string
}

func (idx *Index) indexDivisionNamesFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		group := idx.namesGroups[key]
		if group == nil {
			idx.namesGroupSeq++
			group = &namesGroup{Key: *entry.Key, Order: idx.namesGroupSeq}
			idx.namesGroups[key] = group
		}

		name := strings.Trim(ciScalar(entry.Block.Entries, "name"), "\"")
		if name != "" {
			idx.divisionNames[key] = name
			group.Name = name
		}
		if fallback := strings.Trim(ciScalar(entry.Block.Entries, "fallback_name"), "\""); fallback != "" {
			group.Fallback = fallback
		}
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil || field.Block == nil {
				continue
			}
			switch strings.ToLower(*field.Key) {
			case "for_countries":
				group.Countries = nil
				for _, item := range field.Block.Entries {
					if item != nil && item.Key == nil {
						group.Countries = append(group.Countries, strings.ToUpper(strings.Trim(item.Scalar.StrValue(), "\"")))
					}
				}
			case "can_use":
				group.CanUse = field.Block
			case "ordered":
				for _, item := range field.Block.Entries {
					if item == nil || item.Key == nil || item.Block == nil {
						continue
					}
					number, err := strconv.Atoi(strings.TrimSpace(*item.Key))
					if err != nil {
						continue
					}
					for _, value := range item.Block.Entries {
						if value != nil && value.Key == nil {
							group.Ordered = append(group.Ordered, numberedName{number, strings.Trim(value.Scalar.StrValue(), "\"")})
							break
						}
					}
				}
			}
		}
	}
}

const namesGroupExamples = 10

func (idx *Index) NamesGroupsFor(tag, date string) []map[string]any {
	tag = strings.ToUpper(tag)
	state := idx.decisionStartOf(tag, date)
	seen := map[string]bool{}
	groups := []*namesGroup{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, group := range layer.namesGroups {
			if seen[key] || group.Name == "" {
				continue
			}
			seen[key] = true
			if len(group.Countries) > 0 && !containsFold(group.Countries, tag) {
				continue
			}
			if holds, decided := state.verdict(group.CanUse); !holds || !decided {
				continue
			}
			groups = append(groups, group)
		}
	}
	sort.SliceStable(groups, func(a, b int) bool { return groups[a].Order < groups[b].Order })
	out := []map[string]any{}
	for _, group := range groups {
		examples := []string{}
		for _, item := range group.Ordered {
			if len(examples) == namesGroupExamples {
				break
			}
			examples = append(examples, divisionName(item.Name, item.Number))
		}
		item := map[string]any{
			"key":      group.Key,
			"name":     idx.divisionNamesGroup(group.Key),
			"examples": examples,
			"more":     len(group.Ordered) > namesGroupExamples,
		}

		if group.Fallback != "" {
			item["fallback"] = divisionName(group.Fallback, 123)
		}
		out = append(out, item)
	}
	return out
}

func divisionName(pattern string, number int) string {
	out := strings.ReplaceAll(pattern, "%d", strconv.Itoa(number))
	return strings.ReplaceAll(out, "%s", romanNumeral(number))
}

func romanNumeral(number int) string {
	if number <= 0 {
		return strconv.Itoa(number)
	}
	values := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	symbols := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var out strings.Builder
	for i, value := range values {
		for number >= value {
			out.WriteString(symbols[i])
			number -= value
		}
	}
	return out.String()
}

func containsFold(list []string, want string) bool {
	for _, item := range list {
		if strings.EqualFold(item, want) {
			return true
		}
	}
	return false
}

func (idx *Index) divisionNamesGroup(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return ""
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		name, ok := layer.divisionNames[key]
		if !ok {
			continue
		}
		if text, ok := idx.Localise(name); ok && text != "" {
			return text
		}
		return name
	}
	return ""
}
