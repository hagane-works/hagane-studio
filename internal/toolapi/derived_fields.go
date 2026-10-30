package toolapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
)

func countList(row map[string]any, field string) (any, bool) {
	switch value := row[field].(type) {
	case []any:
		return len(value), true
	case map[string]any:
		return len(value), true
	}
	return nil, false
}

func hasActionableWarning(row map[string]any) (any, bool) {
	warnings, ok := row["warnings"].([]any)
	if !ok {
		return nil, false
	}
	for _, item := range warnings {
		if text, ok := item.(string); ok && strings.HasPrefix(text, "actionable:") {
			return true, true
		}
	}
	return false, true
}

func rawFileSize(row map[string]any) (any, bool) {
	if binary, _ := row["is_binary"].(bool); binary {
		return nil, false
	}
	text, ok := row["content"].(string)
	if !ok {
		return nil, false
	}
	return len(text), true
}

func rawFileReferences(row map[string]any) (any, bool) {
	if binary, _ := row["is_binary"].(bool); binary {
		return nil, false
	}
	text, ok := row["content"].(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil, false
	}
	seen := map[string]bool{}
	var refs []string
	for _, ref := range exportengine.ExtractTextRefs(text) {
		label := ref.TargetType + ":" + ref.TargetID
		if !seen[label] {
			seen[label] = true
			refs = append(refs, label)
		}
	}
	if refs == nil {
		return nil, false
	}
	sort.Strings(refs)
	return refs, true
}

func prideOfTheFleet(row map[string]any) (any, bool) {
	if value, ok := row["pride_of_the_fleet"]; ok && value != nil {
		return value, true
	}
	entries, ok := row["body_ast"].([]any)
	if !ok {
		return nil, false
	}
	for _, item := range entries {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if key, _ := entry["key"].(string); key != "pride_of_the_fleet" {
			continue
		}
		text, _ := entry["value"].(string)
		return strings.EqualFold(strings.TrimSpace(text), "yes"), true
	}
	return nil, false
}

type childCount struct {
	Field string

	ChildKind string

	ChildLink string

	ParentLink string
}

type childCounter struct {
	ctx       context.Context
	proxy     *RESTProxy
	projectID int64
	tallies   map[string]map[string]int
	names     map[string]map[string]string
	failed    map[string]bool
}

func newChildCounter(ctx context.Context, proxy *RESTProxy, projectID int64) *childCounter {
	return &childCounter{ctx: ctx, proxy: proxy, projectID: projectID,
		tallies: map[string]map[string]int{}, names: map[string]map[string]string{},
		failed: map[string]bool{}}
}

func (c *childCounter) count(spec childCount, row map[string]any) (any, bool) {
	if c == nil {
		return nil, false
	}
	parentKey := scalarKey(row[spec.ParentLink])
	if parentKey == "" {
		return nil, false
	}
	tally, ok := c.tallies[spec.ChildKind]
	if !ok {
		if c.failed[spec.ChildKind] {
			return nil, false
		}
		built, err := c.build(spec)
		if err != nil {

			c.failed[spec.ChildKind] = true
			return nil, false
		}
		c.tallies[spec.ChildKind] = built
		tally = built
	}
	return tally[parentKey], true
}

func (c *childCounter) build(spec childCount) (map[string]int, error) {
	kind, err := lookupEntityKind(spec.ChildKind)
	if err != nil {
		return nil, err
	}
	rows, err := proxyAuthedCallAny(c.ctx, c.proxy, "GET",
		fmt.Sprintf("/api/projects/%d/%s", c.projectID, kind.Path), nil)
	if err != nil {
		return nil, err
	}
	tally := map[string]int{}
	for _, row := range rows {
		if key := scalarKey(row[spec.ChildLink]); key != "" {
			tally[key]++
		}
	}
	return tally, nil
}

func scalarKey(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	}
	return ""
}

type parentLookup struct {
	Field string

	ParentKind string

	ChildLink string

	ParentValue string
}

func (c *childCounter) lookup(spec parentLookup, row map[string]any) (any, bool) {
	if c == nil {
		return nil, false
	}
	linkKey := scalarKey(row[spec.ChildLink])
	if linkKey == "" {
		return nil, false
	}
	cacheKey := "parent:" + spec.ParentKind + ":" + spec.ParentValue
	names, ok := c.names[cacheKey]
	if !ok {
		if c.failed[cacheKey] {
			return nil, false
		}
		built, err := c.buildNames(spec)
		if err != nil {
			c.failed[cacheKey] = true
			return nil, false
		}
		c.names[cacheKey] = built
		names = built
	}

	name, ok := names[linkKey]
	if !ok || name == "" {
		return nil, false
	}
	return name, true
}

func (c *childCounter) buildNames(spec parentLookup) (map[string]string, error) {
	kind, err := lookupEntityKind(spec.ParentKind)
	if err != nil {
		return nil, err
	}
	rows, err := proxyAuthedCallAny(c.ctx, c.proxy, "GET",
		fmt.Sprintf("/api/projects/%d/%s", c.projectID, kind.Path), nil)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, row := range rows {
		id := scalarKey(row["id"])
		if id == "" {
			continue
		}
		for _, field := range []string{spec.ParentValue, "name", kind.KeyField} {
			if field == "" {
				continue
			}
			if text, ok := row[field].(string); ok && strings.TrimSpace(text) != "" {
				names[id] = strings.TrimSpace(text)
				break
			}
		}
	}
	return names, nil
}
