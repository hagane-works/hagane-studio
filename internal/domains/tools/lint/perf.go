package lint

import (
	"regexp"
	"sort"
	"strings"
)

var perfIteratorTokens = []string{
	"every_country", "any_country", "all_country", "random_country",
	"every_state", "any_state", "all_state", "random_state",
	"every_neighbor_country", "any_neighbor_country", "all_neighbor_country",
	"every_neighbor_state", "any_neighbor_state",
	"every_owned_state", "any_owned_state", "random_owned_state",
	"every_controlled_state", "any_controlled_state",
	"every_core_state", "any_core_state",
	"every_other_country", "any_other_country", "random_other_country",
	"every_enemy_country", "any_enemy_country",
	"every_unit_leader", "any_unit_leader", "every_army_leader",
	"every_navy_leader", "every_character", "any_character",
	"every_possible_country", "any_possible_country",
}

var perfFanoutTokens = []string{
	"num_of_controlled_states", "num_of_factories", "num_subjects",
	"any_war", "has_war_with", "is_in_faction_with",
	"controls_state", "num_owned_neighbour_states",
}

func perfAlternation(tokens []string) string {
	sorted := append([]string(nil), tokens...)
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })
	return `\b(` + strings.Join(sorted, "|") + `)\b`
}

var (
	perfIterRe   = regexp.MustCompile(perfAlternation(perfIteratorTokens))
	perfFanoutRe = regexp.MustCompile(perfAlternation(perfFanoutTokens))
	perfDaysRe   = regexp.MustCompile(`(?i)\bdays\s*=\s*\d`)
	perfFactorRe = regexp.MustCompile(`(?i)\bfactor\s*=`)
	perfMYRe     = regexp.MustCompile(`(?i)\b(?:months|years)\s*=\s*\d`)
)

const (
	perfThreshFocusNodesPerTree  = 200
	perfThreshEventsPerNamespace = 300
	perfThreshDecisionsPerCat    = 120
	perfThreshHeavyTriggerFanout = 6
	perfThreshTotalFocusNodes    = 1500
)

func perfIteratorNames(text string) []string {
	if text == "" {
		return nil
	}
	ms := perfIterRe.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

func perfNestedIterator(text string) (outer, inner string, ok bool) {
	if text == "" || len(perfIteratorNames(text)) < 2 {
		return "", "", false
	}
	for _, m := range perfIterRe.FindAllStringSubmatchIndex(text, -1) {
		o := text[m[2]:m[3]]
		brace := strings.Index(text[m[1]:], "{")
		if brace == -1 {
			continue
		}
		brace += m[1]
		depth, end := 0, -1
		for i := brace; i < len(text); i++ {
			switch text[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i
					goto scanned
				}
			}
		}
	scanned:
		if end == -1 {
			end = len(text)
		}
		innerBlock := text[min(brace+1, len(text)):end]
		if im := perfIterRe.FindStringSubmatch(innerBlock); im != nil {
			return o, im[1], true
		}
	}
	return "", "", false
}

func perfSortedUnique(names []string) []any {
	set := map[string]struct{}{}
	for _, n := range names {
		set[n] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	res := make([]any, len(out))
	for i, s := range out {
		res[i] = s
	}
	return res
}

func perfEntityRef(kind string, entityID, name, countryTag *string) map[string]any {
	return map[string]any{
		"kind": kind, "entity_id": ptrToAny(entityID),
		"name": ptrToAny(name), "country_tag": ptrToAny(countryTag),
	}
}

func ptrToAny(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func perfRefName(candidates ...*string) *string {
	for _, c := range candidates {
		if c != nil && strings.TrimSpace(*c) != "" {
			v := strings.TrimSpace(*c)
			return &v
		}
	}
	return nil
}

func perfScanHotField(text string, ref map[string]any, file, contextLabel string, findings *[]LintFinding) {
	if text == "" {
		return
	}
	f := &file
	if outer, inner, ok := perfNestedIterator(text); ok {
		*findings = append(*findings, LintFinding{
			Severity: "error", Code: "perf.nested_scope_loop",
			Title: "Nested scope loop (O(N²)) in a hot context",
			Reason: contextLabel + " nests `" + inner + "` inside `" + outer + "`. " +
				"This iterates the game database once per outer scope — O(N²) — and it runs on every evaluation of this context.",
			Suggestion: "Cache the outer result in a variable, narrow the inner set with a `limit = { ... }`, or move the work behind a one-shot triggered event instead of a polled context.",
			EntityRef:  ref, File: f,
			Detail: map[string]any{"context": contextLabel, "outer": outer, "inner": inner},
		})
		return
	}
	iters := perfIteratorNames(text)
	if len(iters) > 0 {
		uniq := perfSortedUnique(iters)
		names := make([]string, len(uniq))
		for i, u := range uniq {
			names[i] = u.(string)
		}
		*findings = append(*findings, LintFinding{
			Severity: "warning", Code: "perf.hot_scope_loop",
			Title: "Scope loop in a frequently-evaluated context",
			Reason: contextLabel + " uses scope iterator(s) " + pyListRepr(names) + ". " +
				"The engine re-evaluates this context often, so each iterator is an O(N) database scan every time.",
			Suggestion: "Prefer a cheap pre-check before the loop, gate the context with `is_triggered_only` / a flag, or add a `days = N` floor so it polls less often.",
			EntityRef:  ref, File: f,
			Detail: map[string]any{"context": contextLabel, "iterators": uniq},
		})
	}
	if len(iters) >= perfThreshHeavyTriggerFanout {
		*findings = append(*findings, LintFinding{
			Severity: "warning", Code: "perf.heavy_trigger_fanout",
			Title: "Heavy trigger fan-out in a hot context",
			Reason: contextLabel + " chains " + itoaInt(len(iters)) + " scope iterators in " +
				"a single evaluation — a deep, expensive boolean tree.",
			Suggestion: "Split the check, short-circuit with the cheapest condition first, or precompute the result into a flag updated on a slower cadence.",
			EntityRef:  ref, File: f,
			Detail: map[string]any{"context": contextLabel, "iterator_count": len(iters)},
		})
	}
}

func pyListRepr(items []string) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, s := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('\'')
		b.WriteString(s)
		b.WriteByte('\'')
	}
	b.WriteByte(']')
	return b.String()
}

func perfScanNested(text string, ref map[string]any, file, contextLabel string, findings *[]LintFinding) {
	outer, inner, ok := perfNestedIterator(text)
	if !ok {
		return
	}
	f := &file
	*findings = append(*findings, LintFinding{
		Severity: "warning", Code: "perf.nested_scope_loop_coldpath",
		Title: "Nested scope loop (O(N²))",
		Reason: contextLabel + " nests `" + inner + "` inside `" + outer + "` (O(N²) over " +
			"the game database). Even in a one-shot effect this can stall a tick on a large game state.",
		Suggestion: "Narrow the inner set with a `limit = { ... }` or restructure to avoid iterating all scopes twice.",
		EntityRef:  ref, File: f,
		Detail: map[string]any{"context": contextLabel, "outer": outer, "inner": inner},
	})
}
