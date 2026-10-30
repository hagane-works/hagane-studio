package worldout

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var startTransferDate = [3]int{1936, 1, 1}

var historyDateRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

func ownerOverridesStartTransfers(st *OMap, vanillaOwner string) bool {
	owner := pyStrip(pyStrOrEmpty(eGet(st, "owner")))
	if owner == "" || vanillaOwner == "" || strings.EqualFold(owner, vanillaOwner) {
		return false
	}
	return pyStrip(pyStrOrEmpty(eGet(st, "source"))) != "imported"
}

func StartTransfersLeftOut(rawExtras map[string]any) (set map[string]any, drop []string) {
	elseFollows := false
	for key := range rawExtras {
		if k := toLower(key); k == "history.else" || k == "history.else_if" {
			elseFollows = true
		}
	}
	set = map[string]any{}
	for key, value := range rawExtras {
		if !strings.HasPrefix(key, "history.") {
			continue
		}
		out, targets, keep := historyWithoutStartTransfers(key[len("history."):], value, elseFollows)
		if len(targets) == 0 {
			continue
		}
		if keep {
			set[key] = out
		} else {
			drop = append(drop, key)
		}
	}
	sort.Strings(drop)
	return set, drop
}

func historyWithoutStartTransfers(key string, value Value, elseFollows bool) (Value, []string, bool) {
	if list, ok := value.([]Value); ok {
		var kept []Value
		var targets []string
		for _, item := range list {
			out, t, keep := historyWithoutStartTransfers(key, item, elseFollows)
			targets = append(targets, t...)
			if keep {
				kept = append(kept, out)
			}
		}
		if len(kept) == 0 {
			return nil, targets, false
		}
		return kept, targets, true
	}
	text, ok := value.(string)
	if !ok {
		return value, nil, true
	}
	if m := historyDateRe.FindStringSubmatch(key); m != nil {
		var date [3]int
		for i := range date {
			date[i], _ = strconv.Atoi(m[i+1])
		}
		if dateAfter(date, startTransferDate) {
			return value, nil, true
		}
		out, targets, ok := stripStartTransfers(text)
		if !ok || len(targets) == 0 {
			return value, nil, true
		}
		if strings.TrimSpace(out) == "" {
			return nil, targets, false
		}
		return out, targets, true
	}
	switch {
	case strings.EqualFold(key, "if"):
		out, targets, ok := stripStartTransfers(text)
		if !ok || len(targets) == 0 {
			return value, nil, true
		}
		if elseFollows {
			return out, targets, true
		}
		return nil, targets, false
	case strings.EqualFold(key, "transfer_state_to"):
		return nil, []string{pyStrip(text)}, false
	}
	if hasTransferState(text) {
		return nil, []string{key}, false
	}
	return value, nil, true
}

func dateAfter(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func stripStartTransfers(text string) (string, []string, bool) {
	stmts, ok := scriptStatements(text)
	if !ok {
		return text, nil, false
	}
	var b strings.Builder
	var targets []string
	last := 0
	for n, s := range stmts {
		if tag, ok := transferTarget(text, s); ok {
			b.WriteString(text[last:s.start])
			last = s.end
			targets = append(targets, tag)
			continue
		}
		if !s.block || !strings.EqualFold(s.key, "if") {
			continue
		}
		inner, innerTargets, ok := stripStartTransfers(text[s.bodyStart:s.bodyEnd])
		if !ok || len(innerTargets) == 0 {
			continue
		}
		targets = append(targets, innerTargets...)
		b.WriteString(text[last:s.start])
		if n+1 < len(stmts) && (strings.EqualFold(stmts[n+1].key, "else") || strings.EqualFold(stmts[n+1].key, "else_if")) {
			b.WriteString(text[s.start:s.bodyStart])
			b.WriteString(inner)
			b.WriteString(text[s.bodyEnd:s.end])
		}
		last = s.end
	}
	if len(targets) == 0 {
		return text, nil, true
	}
	b.WriteString(text[last:])
	return b.String(), targets, true
}

func transferTarget(text string, s scriptStatement) (string, bool) {
	if !s.block {
		if strings.EqualFold(s.key, "transfer_state_to") && s.value != "" {
			return s.value, true
		}
		return "", false
	}
	if s.key == "" || strings.EqualFold(s.key, "if") || strings.EqualFold(s.key, "limit") {
		return "", false
	}
	if hasTransferState(text[s.bodyStart:s.bodyEnd]) {
		return s.key, true
	}
	return "", false
}

func hasTransferState(body string) bool {
	stmts, ok := scriptStatements(body)
	if !ok {
		return false
	}
	for _, s := range stmts {
		if strings.EqualFold(s.key, "transfer_state") {
			return true
		}
	}
	return false
}

func startTransferNote(targets []string, owner string) string {
	seen := map[string]bool{}
	var tags []string
	for _, t := range targets {
		t = strings.ToUpper(pyStrip(t))
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	sort.Strings(tags)
	return "# Left out: a DLC hands this state to " + strings.Join(tags, ", ") +
		" when the game starts; its owner is set to " + owner + " instead."
}

type scriptStatement struct {
	key                string
	value              string
	start, end         int
	block              bool
	bodyStart, bodyEnd int
}

type scriptToken struct {
	kind       byte
	text       string
	start, end int
}

func scriptTokens(text string) ([]scriptToken, bool) {
	var out []scriptToken
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '#':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case c == '{' || c == '}':
			out = append(out, scriptToken{kind: c, start: i, end: i + 1})
			i++
		case c == '=' || c == '<' || c == '>' || c == '!':
			j := i + 1
			if j < len(text) && text[j] == '=' {
				j++
			}
			out = append(out, scriptToken{kind: '=', text: text[i:j], start: i, end: j})
			i = j
		case c == '"':
			j := strings.IndexByte(text[i+1:], '"')
			if j < 0 {
				return nil, false
			}
			out = append(out, scriptToken{kind: 's', text: text[i+1 : i+1+j], start: i, end: i + 2 + j})
			i += 2 + j
		default:
			j := i
			for j < len(text) && !strings.ContainsRune(" \t\r\n{}=<>!\"#", rune(text[j])) {
				j++
			}
			out = append(out, scriptToken{kind: 'w', text: text[i:j], start: i, end: j})
			i = j
		}
	}
	return out, true
}

func scriptStatements(text string) ([]scriptStatement, bool) {
	toks, ok := scriptTokens(text)
	if !ok {
		return nil, false
	}
	closing := func(open int) int {
		depth := 0
		for j := open; j < len(toks); j++ {
			switch toks[j].kind {
			case '{':
				depth++
			case '}':
				if depth--; depth == 0 {
					return j
				}
			}
		}
		return -1
	}
	var out []scriptStatement
	for i := 0; i < len(toks); {
		t := toks[i]
		switch t.kind {
		case '}', '=':
			return nil, false
		case '{':
			j := closing(i)
			if j < 0 {
				return nil, false
			}
			out = append(out, scriptStatement{start: t.start, end: toks[j].end, block: true, bodyStart: t.end, bodyEnd: toks[j].start})
			i = j + 1
			continue
		}
		s := scriptStatement{key: t.text, start: t.start, end: t.end}
		k := i + 1
		hasOp := k < len(toks) && toks[k].kind == '='
		if hasOp {
			k++
		}
		switch {
		case k < len(toks) && toks[k].kind == '{':
			j := closing(k)
			if j < 0 {
				return nil, false
			}
			s.block, s.bodyStart, s.bodyEnd, s.end = true, toks[k].end, toks[j].start, toks[j].end
			i = j + 1
		case hasOp && k < len(toks) && (toks[k].kind == 'w' || toks[k].kind == 's'):
			s.value, s.end = toks[k].text, toks[k].end
			i = k + 1
		case hasOp:
			return nil, false
		default:
			i++
		}
		out = append(out, s)
	}
	return out, true
}
