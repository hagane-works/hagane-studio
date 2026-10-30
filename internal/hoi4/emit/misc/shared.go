package emitmisc

import "strings"

func illegalPathChar(r rune) bool {
	switch r {
	case ':', '\\', '/', '*', '?', '"', '<', '>', '|':
		return true
	}
	return false
}

func subIllegalPathChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if illegalPathChar(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type Buckets struct {
	Keys []string
	m    map[string][]*NS
}

func newBuckets() *Buckets { return &Buckets{m: map[string][]*NS{}} }

func (b *Buckets) append(k string, v *NS) {
	if _, ok := b.m[k]; !ok {
		b.Keys = append(b.Keys, k)
		b.m[k] = nil
	}
	b.m[k] = append(b.m[k], v)
}

func (b *Buckets) ensure(k string) {
	if _, ok := b.m[k]; !ok {
		b.Keys = append(b.Keys, k)
		b.m[k] = nil
	}
}

func (b *Buckets) Get(k string) []*NS { return b.m[k] }

func (b *Buckets) Has(k string) bool {
	_, ok := b.m[k]
	return ok
}

func BucketByCountry(items []*NS, primaryTag string) *Buckets {
	buckets := newBuckets()
	pt := pyUpper(primaryTag)
	if pt == "" {
		pt = "UNK"
	}
	for _, it := range items {
		raw := pyUpper(it.Str("country_tag"))
		key := raw
		if key == "" {
			key = pt
		}
		buckets.append(key, it)
	}
	return buckets
}

type FileGroups struct {
	Keys []string
	m    map[string][]*NS
}

func newFileGroups() *FileGroups { return &FileGroups{m: map[string][]*NS{}} }

func (g *FileGroups) append(k string, v *NS) {
	if _, ok := g.m[k]; !ok {
		g.Keys = append(g.Keys, k)
		g.m[k] = nil
	}
	g.m[k] = append(g.m[k], v)
}

func (g *FileGroups) Get(k string) []*NS { return g.m[k] }

func FocusTreeFilenames(trees []*NS) *FileGroups {
	out := newFileGroups()
	for _, tree := range trees {
		rawSrc := tree.StrOr("source_file", tree.Str("tree_id")+".txt")
		replaced := strings.ReplaceAll(rawSrc, "\\", "/")
		parts := strings.Split(replaced, "/")
		filename := parts[len(parts)-1]
		if filename == "" {
			filename = tree.Str("tree_id") + ".txt"
		}
		out.append(filename, tree)
	}
	return out
}

func GroupBySourceFile(rows []*NS, defaultPath func(*NS) string, expectedDir string) *FileGroups {
	grouped := newFileGroups()
	prefix := pyStripChars(expectedDir, "/")
	for _, row := range rows {
		raw := pyStrip(strings.ReplaceAll(row.Str("source_file"), "\\", "/"))
		path := ""
		if raw != "" {
			var segments []string
			for _, s := range strings.Split(raw, "/") {
				if s != "" && s != "." {
					segments = append(segments, s)
				}
			}
			hasDotDot := false
			for _, s := range segments {
				if s == ".." {
					hasDotDot = true
				}
			}
			if len(segments) > 0 && !hasDotDot && !strings.Contains(raw, ":") {
				candidate := strings.Join(segments, "/")
				if strings.HasPrefix(candidate, prefix+"/") {
					path = candidate
				} else {
					path = prefix + "/" + segments[len(segments)-1]
				}
			}
		}
		if path == "" {
			path = defaultPath(row)
		}
		grouped.append(path, row)
	}
	return grouped
}

func AuthorRawOverridePaths(rawFiles []*NS) map[string]bool {
	out := map[string]bool{}
	for _, row := range rawFiles {
		src := row.Attr("source", nil)
		srcStr, _ := src.(string)
		if src != nil && srcStr == "imported" {
			continue
		}
		p := pyStripChars(strings.ReplaceAll(row.Str("dest_path"), "\\", "/"), "/")
		out[p] = true
	}
	return out
}

func OOBDestination(row *NS) string {
	sourceFile := pyStripChars(strings.ReplaceAll(row.Str("source_file"), "\\", "/"), "/")
	if strings.HasPrefix(sourceFile, "history/units/") && strings.HasSuffix(sourceFile, ".txt") {
		relative := sourceFile[len("history/units/"):]
		var parts []string
		for _, part := range strings.Split(relative, "/") {
			if part != "" && part != "." && part != ".." {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			safeParts := make([]string, 0, len(parts))
			allOK := true
			for _, part := range parts {
				sp := pyStrip(strings.ReplaceAll(subIllegalPathChars(part), "..", "_"))
				safeParts = append(safeParts, sp)
				if sp == "" {
					allOK = false
				}
			}
			if allOK {
				return "history/units/" + strings.Join(safeParts, "/")
			}
		}
	}
	oob := row.Str("oob_file")
	if oob == "" {
		oob = "untitled"
	}
	stem := pyStripChars(strings.ReplaceAll(subIllegalPathChars(oob), "..", "_"), "/\\ ")
	if stem == "" {
		stem = "untitled"
	}
	return "history/units/" + stem + ".txt"
}
