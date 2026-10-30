package toolapi

import (
	"strings"

	"haganeworks/internal/domains/tools/lint"
)

func uiPanelWindowName(panel map[string]any) string {
	if derived := nodeField(panel, "derived"); derived != nil {
		if window := strings.TrimSpace(stringField(derived, "window_name")); window != "" {
			return window
		}
	}
	if window := strings.TrimSpace(stringField(panel, "window_name")); window != "" {
		return window
	}
	token := strings.TrimSpace(stringField(panel, "token"))
	if token == "" {
		return ""
	}
	return lint.DerivePanelNames(token, "", false)["window_name"]
}
