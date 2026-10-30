package build

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/gameindex"
)

const docsSource = "HOI4 game-shipped documentation/*.md (via app.kb.hoi4_docs_parser)"

func splitH2(text string) [][2]string {
	var out [][2]string
	var cur *string
	var buf []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			if cur != nil {
				out = append(out, [2]string{*cur, strings.Join(buf, "\n")})
			}
			title := strings.TrimSpace(line[3:])
			cur = &title
			buf = nil
			continue
		}
		buf = append(buf, line)
	}
	if cur != nil {
		out = append(out, [2]string{*cur, strings.Join(buf, "\n")})
	}
	return out
}

var (
	scopeSectionRe = regexp.MustCompile(`(?i)^(Effects|Triggers) for scope `)
	supScopeRe     = regexp.MustCompile(`(?m)^\* Supported Scopes:(.*)$`)
	supTargetRe    = regexp.MustCompile(`(?m)^\* Supported Targets:(.*)$`)
)

func scopedEntryNames(text string) map[string]bool {
	out := map[string]bool{}
	for _, section := range splitH2(text) {
		title, body := section[0], section[1]
		if title == "Table of Content" || scopeSectionRe.MatchString(title) {
			continue
		}
		if !supScopeRe.MatchString(body) && !supTargetRe.MatchString(body) {
			continue
		}
		out[title] = true
	}
	return out
}

func readDoc(in *gameindex.Input, name string) (string, error) {
	text, ok := readBase(in, "documentation/"+name)
	if !ok {
		return "", missingFile("documentation/" + name)
	}
	return text, nil
}

func scriptNamesJSON(in *gameindex.Input, file string) ([]byte, error) {
	text, err := readDoc(in, file)
	if err != nil {
		return nil, err
	}
	ids := sortedKeys(scopedEntryNames(text))
	return marshal(map[string]any{
		"source": docsSource, "game_version": in.GameVersion,
		"extracted_at": time.Now().Format("2006-01-02"), "total": len(ids), "ids": ids,
	})
}

func buildVanillaEffects(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	return scriptNamesJSON(in, "effects_documentation.md")
}

func buildVanillaTriggers(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	return scriptNamesJSON(in, "triggers_documentation.md")
}

var (
	modSpanRe     = regexp.MustCompile(`^\s*<span id="[^"]*"></span>(.*)$`)
	placeholderRe = regexp.MustCompile(`<[A-Za-z_]+>`)
)

func buildVanillaModifiers(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	text, err := readDoc(in, "modifiers_documentation.md")
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, section := range splitH2(text) {
		title := section[0]
		if title == "Table of Content" || strings.HasPrefix(title, "Modifiers for scope ") {
			continue
		}
		name := title
		if m := modSpanRe.FindStringSubmatch(title); m != nil {
			name = m[1]
		}
		name = strings.TrimSpace(name)
		if name == "" || placeholderRe.MatchString(name) {
			continue
		}
		names[name] = true
	}
	list := sortedKeys(names)
	return marshal(map[string]any{
		"source": docsSource, "game_version": in.GameVersion,
		"extracted_at": time.Now().Format("2006-01-02"), "total": len(list), "modifiers": list,
	})
}

var mathStructuralSubKeys = []string{"alpha", "else", "limit", "named_collection", "to", "value"}

func buildMathFunctions(ctx context.Context, in *gameindex.Input) ([]byte, error) {
	text, err := readDoc(in, "script_math_functions.md")
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, section := range splitH2(text) {
		title := section[0]
		switch title {
		case "Table of Content", "":
			continue
		case "<":
			title = "less_than"
		case ">":
			title = "greater_than"
		}
		ids[title] = true
	}
	for _, key := range mathStructuralSubKeys {
		ids[key] = true
	}
	list := sortedKeys(ids)
	return marshal(map[string]any{
		"source":              "documentation/script_math_functions.md",
		"game_version":        shortVersion(in.GameVersion),
		"total":               len(list),
		"structural_sub_keys": mathStructuralSubKeys,
		"ids":                 list,
	})
}

func mathFunctions(raw []byte) ([]string, []string) {
	var f struct {
		IDs []string `json:"ids"`
		Sub []string `json:"structural_sub_keys"`
	}
	_ = json.Unmarshal(raw, &f)
	sort.Strings(f.IDs)
	return f.IDs, f.Sub
}
