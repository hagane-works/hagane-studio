package exportengine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/hoi4/vanilladata"
)

type ScriptedRef struct {
	Token string

	Count int

	First string

	Files []string
}

var scriptWrapKeys = map[string]bool{
	"if": true, "else": true, "else_if": true, "limit": true,
	"AND": true, "OR": true, "NOT": true, "and": true, "or": true, "not": true,
	"any": true, "all": true, "all_owned_state": true, "any_owned_state": true,
	"any_country": true, "all_country": true, "any_other_country": true,
	"FROM": true, "ROOT": true, "THIS": true, "PREV": true, "OWNER": true, "CONTROLLER": true,
	"var": true, "value": true,
	"option": true, "ai_chance": true, "base": true, "modifier": true, "factor": true, "add": true,
	"trigger": true, "immediate": true, "after": true, "fire_only_once": true,
	"is_triggered_only": true, "hidden": true, "major": true,
	"id": true, "title": true, "desc": true, "picture": true, "name": true,
	"scope": true, "tag": true, "original_tag": true, "owner": true, "controller": true,
	"random_list": true, "random_country": true, "random_state": true,
	"global_event_target": true, "event_target": true,

	"is_border_war": true, "show_ideas_tooltip": true, "war_with_on_create": true,
	"instant": true, "popup": true, "reversed": true,
}

func pyIsUpper(s string) bool {
	cased := false
	for _, r := range s {
		if unicode.IsLower(r) || unicode.IsTitle(r) {
			return false
		}
		if unicode.IsUpper(r) {
			cased = true
		}
	}
	return cased
}

func isScriptedCall(key, scalar string, defined map[string]bool) bool {
	if key == "" {
		return false
	}
	if lower := strings.ToLower(scalar); lower != "yes" && lower != "no" {
		return false
	}
	if vanilladata.IsVanillaEffect(key) || vanilladata.IsVanillaTrigger(key) {
		return false
	}
	if vanilladata.IsStructuralField(key) {
		return false
	}
	if scriptWrapKeys[key] {
		return false
	}
	if _, isCrossRef := crossRefSpecs[key]; isCrossRef {
		return false
	}
	if strings.Contains(key, ".") {
		return false
	}
	if pyIsUpper(key) {
		return false
	}

	if vanilladata.IsVanillaYesNoKey(key) {
		return false
	}
	if defined[key] {
		return false
	}
	return true
}

var scriptedDefinitionDirs = []string{"common/scripted_effects/", "common/scripted_triggers/"}

func declaredScriptedNames(entries []Entry, modRoot string) map[string]bool {
	out := map[string]bool{}
	for _, entry := range entries {
		if !isScriptEntry(entry) {
			continue
		}
		rel := modRelative(entry.Path, modRoot)
		inDefinitionDir := false
		for _, dir := range scriptedDefinitionDirs {
			if strings.HasPrefix(rel, dir) {
				inDefinitionDir = true
				break
			}
		}
		if !inDefinitionDir {
			continue
		}
		block, _, err := paradox.Parse(scriptText(entry))
		if err != nil || block == nil {
			continue
		}
		for _, e := range block.Entries {
			if e.Key != nil && e.Block != nil && *e.Key != "" {
				out[*e.Key] = true
			}
		}
	}
	return out
}

func hasUnclosedBlock(recoveries []paradox.Recovery) bool {
	for _, r := range recoveries {
		if r.Code == paradox.RecoveryUnclosedBlock {
			return true
		}
	}
	return false
}

func modRelative(path, modRoot string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, modRoot), "/")
}

func ScanScriptedRefs(entries []Entry, modRoot string) []ScriptedRef {
	if len(entries) == 0 {
		return nil
	}
	defined := declaredScriptedNames(entries, modRoot)
	byToken := map[string]*ScriptedRef{}
	filesSeen := map[string]map[string]bool{}

	for _, entry := range entries {
		if !isScriptEntry(entry) {
			continue
		}
		rel := modRelative(entry.Path, modRoot)

		text := scriptText(entry)
		block, recoveries, err := paradox.Parse(text)
		if err != nil || block == nil || hasUnclosedBlock(recoveries) {

			continue
		}
		locator := newTokenLocator(text)
		walkBlock(block, func(e *paradox.Entry) {
			if e.Key == nil || e.Block != nil {
				return
			}
			key := *e.Key
			if !isScriptedCall(key, e.Scalar.StrValue(), defined) {
				return
			}
			ref := byToken[key]
			if ref == nil {
				ref = &ScriptedRef{Token: key}
				byToken[key] = ref
				filesSeen[key] = map[string]bool{}
				if line, column, ok := locator.locate(key, e.Line); ok {
					ref.First = fmt.Sprintf("%s:%d:%d", rel, line, column)
				} else {
					ref.First = rel
				}
			}
			ref.Count++
			filesSeen[key][rel] = true
		})
	}

	out := make([]ScriptedRef, 0, len(byToken))
	for token, ref := range byToken {
		files := make([]string, 0, len(filesSeen[token]))
		for f := range filesSeen[token] {
			files = append(files, f)
		}
		sort.Strings(files)
		ref.Files = files
		out = append(out, *ref)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Token < out[j].Token })
	return out
}

func scriptedSectionHeader(msg exportmsg.Printer) []string {
	return append([]string{"# informational · scripted_effect_trigger"}, commentLines(msg.Text("scripted.log.header", nil))...)
}

func scriptedLines(refs []ScriptedRef, msg exportmsg.Printer) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		line := fmt.Sprintf("informational · scripted_effect_trigger · %s: %d", ref.Token, ref.Count)
		if ref.First != "" {
			line += " · " + msg.Text("scripted.log.first", exportmsg.Args{"where": ref.First})
		}
		out = append(out, line)
	}
	return out
}

func scriptedHeadline(refs []ScriptedRef, msg exportmsg.Printer) string {
	if len(refs) == 0 {
		return ""
	}
	tokens := make([]string, 0, len(refs))
	for _, ref := range refs {
		tokens = append(tokens, ref.Token)
	}
	return msg.Text("scripted.headline", exportmsg.Args{
		"count": len(refs), "names": joinCapped(tokens, 5, msg), "log": crossRefLogName,
	})
}
