package exportmsg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func placeholders(template string) map[string]bool {
	out := map[string]bool{}
	for _, match := range placeholder.FindAllStringSubmatch(template, -1) {
		out[match[1]] = true
	}
	return out
}

func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestEveryLocaleHasEveryKeyWithTheSamePlaceholders(t *testing.T) {
	english := catalogs["en"]
	if len(english) == 0 {
		t.Fatal("the English catalog is empty")
	}
	for _, locale := range Locales {
		catalog := catalogs[locale]
		for key, template := range english {
			text, ok := catalog[key]
			if !ok || strings.TrimSpace(text) == "" {
				t.Errorf("%s: %q is missing", locale, key)
				continue
			}
			want, got := sortedNames(placeholders(template)), sortedNames(placeholders(text))
			if strings.Join(want, ",") != strings.Join(got, ",") {
				t.Errorf("%s: %q has placeholders %v, English has %v", locale, key, got, want)
			}
		}
		for key := range catalog {
			if _, ok := english[key]; !ok {
				t.Errorf("%s: %q is not in the English catalog", locale, key)
			}
		}
	}
}

func TestUnknownLocaleWritesEnglish(t *testing.T) {
	for _, locale := range []string{"", "xx", "en", " EN "} {
		if got := For(locale); got != (Printer{}) || got.Locale() != "en" {
			t.Errorf("For(%q) = %+v, want English", locale, got)
		}
	}
	if got := For(" ZH ").Locale(); got != "zh" {
		t.Errorf("For(\" ZH \").Locale() = %q", got)
	}
}

func TestTextFillsPlaceholders(t *testing.T) {
	p := For("en")
	if got := p.Text("list.more", Args{"items": "A, B", "count": 3}); got != "A, B and 3 more" {
		t.Errorf("list.more = %q", got)
	}

	reason := Message{Key: "oob.reason.and", Args: Args{"first": Message{Key: "oob.reason.army"}, "second": Message{Key: "oob.reason.air"}}}
	if got := p.Render(reason); got != "the owner of the states its divisions spawn in and the owner of the state its air wings are based in" {
		t.Errorf("nested message = %q", got)
	}
	zh := For("zh")
	if got, want := zh.Render(reason), zh.Text("oob.reason.and", Args{"first": zh.Text("oob.reason.army", nil), "second": zh.Text("oob.reason.air", nil)}); got != want {
		t.Errorf("nested message in Chinese = %q, want %q", got, want)
	}

	if got := p.Text("no.such.key", nil); got != "no.such.key" {
		t.Errorf("unknown key = %q", got)
	}
	if got := p.Text("list.more", Args{"items": "A"}); got != "A and {count} more" {
		t.Errorf("missing argument = %q", got)
	}
}

func TestCodeAndCatalogAgree(t *testing.T) {
	english := catalogs["en"]
	areas := map[string]bool{}
	for key := range english {
		areas[key[:strings.IndexByte(key, '.')]] = true
	}
	used := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		imports := false
		for _, spec := range file.Imports {
			imports = imports || spec.Path.Value == `"haganeworks/internal/hoi4/exportmsg"`
		}
		if !imports {
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if dot := strings.IndexByte(value, '.'); err == nil && dot > 0 && areas[value[:dot]] && !strings.ContainsAny(value, " /") && !strings.HasSuffix(value, ".") {
				used[value] = true
				if _, ok := english[value]; !ok {
					t.Errorf("%s names %q, which the catalog does not have", fset.Position(literal.Pos()), value)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key := range english {
		if !used[key] {
			t.Errorf("%q is in the catalog but no code writes it", key)
		}
	}
}
