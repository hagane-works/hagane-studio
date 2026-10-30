package exportengine

import (
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

const LiteralNewlineLintLogName = "_LITERAL_NEWLINE_LINT.log"

type LiteralNewlineHit struct {
	Path    string
	Line    int
	Column  int
	Excerpt string
}

type LiteralNewlineReport struct {
	Hits []LiteralNewlineHit
	msg  exportmsg.Printer
}

func (r LiteralNewlineReport) Count() int { return len(r.Hits) }

func (r LiteralNewlineReport) Headline() string {
	if len(r.Hits) == 0 {
		return ""
	}
	return r.msg.Text("newline.headline", exportmsg.Args{"count": len(r.Hits), "log": LiteralNewlineLintLogName})
}

func ScanLiteralNewlines(entries []Entry, msg exportmsg.Printer) LiteralNewlineReport {
	root := artifactRootFromEntries(entries)
	report := LiteralNewlineReport{msg: msg}
	for _, entry := range entries {
		rel, ok := literalNewlineScriptPath(entry.Path, root)
		if !ok || len(entry.Bytes) == 0 {
			continue
		}
		report.Hits = append(report.Hits, literalNewlineHits(rel, scriptText(entry))...)
	}
	sort.SliceStable(report.Hits, func(i, j int) bool {
		a, b := report.Hits[i], report.Hits[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	return report
}

func literalNewlineScriptPath(entryPath, root string) (string, bool) {
	p := strings.TrimPrefix(strings.ReplaceAll(entryPath, "\\", "/"), "/")
	if root != "" {
		p = strings.TrimPrefix(p, strings.TrimSuffix(root, "/")+"/")
	}
	lower := strings.ToLower(p)
	if !strings.HasSuffix(lower, ".txt") {
		return "", false
	}
	for _, prefix := range []string{"common/", "events/", "history/"} {
		if strings.HasPrefix(lower, prefix) {
			return p, true
		}
	}
	return "", false
}

func literalNewlineHits(path, src string) []LiteralNewlineHit {
	var hits []LiteralNewlineHit
	inComment := false
	line, column, lineStart := 1, 1, 0
	for i := 0; i < len(src)-1; i++ {
		switch src[i] {
		case '\n':
			inComment = false
			line, column, lineStart = line+1, 1, i+1
		case '#':
			if !inComment {
				inComment = true
			}
			column++
		case '\\':
			if !inComment && src[i+1] == 'n' {
				hits = append(hits, LiteralNewlineHit{Path: path, Line: line, Column: column,
					Excerpt: literalNewlineExcerpt(src, lineStart)})
				i++
				column += 2
				continue
			}
			column++
		default:
			column++
		}
	}
	return hits
}

func literalNewlineExcerpt(src string, start int) string {
	if start < 0 || start > len(src) {
		return ""
	}
	end := strings.IndexByte(src[start:], '\n')
	if end < 0 {
		end = len(src) - start
	}
	line := strings.TrimSpace(src[start : start+end])
	if len(line) > 180 {
		line = line[:180] + "…"
	}
	return line
}

func LiteralNewlineLogEntry(modRoot string, report LiteralNewlineReport) Entry {
	if len(report.Hits) == 0 {
		return Entry{}
	}
	var body strings.Builder
	for _, line := range commentLines(report.msg.Text("newline.log.header", nil)) {
		body.WriteString(line + "\n")
	}
	for _, hit := range report.Hits {
		fmt.Fprintf(&body, "%s:%d:%d | %s\n", hit.Path, hit.Line, hit.Column, hit.Excerpt)
	}
	return Entry{Path: strings.TrimSuffix(modRoot, "/") + "/" + LiteralNewlineLintLogName,
		Bytes: []byte(body.String()), Origin: "generated"}
}
