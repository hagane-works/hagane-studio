package toolapi

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed all:fieldsdata
var fieldsDataFS embed.FS

var (
	fieldDocsOnce sync.Once
	fieldDocsMap  map[string]string
)

func fieldDocs() map[string]string {
	fieldDocsOnce.Do(func() {
		entries, err := fieldsDataFS.ReadDir("fieldsdata")
		if err != nil {
			panic(fmt.Sprintf("toolapi: embedded fieldsdata unreadable: %v", err))
		}
		docs := make(map[string]string, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".md") {
				continue
			}
			raw, err := fieldsDataFS.ReadFile("fieldsdata/" + name)
			if err != nil {
				panic(fmt.Sprintf("toolapi: embedded fieldsdata/%s unreadable: %v", name, err))
			}
			stem := strings.TrimSuffix(name, ".md")
			docs[stem] = strings.ReplaceAll(string(raw), "\r\n", "\n")
		}
		fieldDocsMap = docs
	})
	return fieldDocsMap
}

func readFieldDoc(workbench string) (string, error) {
	docs := fieldDocs()
	if doc, ok := docs[workbench]; ok {
		return doc, nil
	}
	avail := make([]string, 0, len(docs))
	for stem := range docs {
		if !strings.HasPrefix(stem, "_") {
			avail = append(avail, stem)
		}
	}
	sort.Strings(avail)
	return "", fmt.Errorf("Unknown field reference '%s'. Available: %s",
		workbench, strings.Join(avail, ", "))
}

func listFieldWorkbenches() []string {
	docs := fieldDocs()
	out := make([]string, 0, len(docs))
	for stem := range docs {
		if !strings.HasPrefix(stem, "_") && stem != "README" {
			out = append(out, stem)
		}
	}
	sort.Strings(out)
	return out
}

func buildFieldsIndex() string {
	head, err := readFieldDoc("README")
	if err != nil {

		head = err.Error()
	}
	lines := make([]string, 0, len(fieldDocs()))
	for _, w := range listFieldWorkbenches() {
		lines = append(lines, "- `"+w+"`")
	}
	return head + "\n\n## Field references (`hw fields <name>`)\n" + strings.Join(lines, "\n") + "\n- `tokens`\n"
}
