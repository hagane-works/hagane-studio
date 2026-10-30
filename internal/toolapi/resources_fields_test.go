package toolapi

import (
	"strings"
	"testing"
)

func TestFieldDocs(t *testing.T) {
	t.Run("fields index", func(t *testing.T) {
		index := buildFieldsIndex()
		if !strings.Contains(index, "## Field references (`hw fields <name>`)") {
			t.Errorf("index missing the field references section")
		}
		for _, want := range []string{"- `focus`", "- `characters`", "- `tokens`"} {
			if !strings.Contains(index, want) {
				t.Errorf("index missing %q", want)
			}
		}
		_, listing, _ := strings.Cut(index, "## Field references")
		if strings.Contains(listing, "- `README`") || strings.Contains(listing, "- `_") {
			t.Errorf("index must not list README or _* meta docs")
		}
		if strings.Contains(index, "\r\n") {
			t.Errorf("index contains CRLF")
		}
	})

	t.Run("tokens", func(t *testing.T) {
		doc, err := readFieldDoc("tokens")
		if err != nil || !strings.HasPrefix(doc, "# Token vocabulary") {
			t.Errorf("tokens.md not served: %v %.60q", err, doc)
		}
	})

	t.Run("underscore meta doc readable by name", func(t *testing.T) {

		for _, name := range []string{"_raw-script-fields", "_editing-existing-mods", "_timeline-and-replace-path", "_validation-tools"} {
			if doc, err := readFieldDoc(name); err != nil || doc == "" {
				t.Errorf("%s must be readable by name: %v", name, err)
			}
		}
	})

	t.Run("unknown name errors with the available list", func(t *testing.T) {
		_, err := readFieldDoc("nope")
		if err == nil {
			t.Fatalf("expected an error for an unknown name")
		}
		msg := err.Error()
		if !strings.Contains(msg, "Unknown field reference 'nope'") || !strings.Contains(msg, "focus") {
			t.Errorf("error = %q, want the available list", msg)
		}
		if strings.Contains(msg, "_raw-script-fields") {
			t.Errorf("error = %q, must not advertise _* meta docs", msg)
		}
	})

	t.Run("no web-version channels", func(t *testing.T) {
		for name, doc := range fieldDocs() {
			for _, word := range []string{"hoi4pat", "MCP", "hagane.works", "the platform", "hoi4://"} {
				if strings.Contains(doc, word) {
					t.Errorf("%s mentions %q", name, word)
				}
			}
		}
	})
}
