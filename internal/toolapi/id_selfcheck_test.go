package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func backendReturningEventID(t *testing.T, storedID string, storedNumber int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 501, "event_id": storedID, "event_number": storedNumber, "warnings": []any{},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func callCreateEvent(t *testing.T, backendURL string, requestedID string, requestedNumber int) *CallToolResult {
	t.Helper()
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(backendURL),
		}
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_event",
		Arguments: map[string]any{
			"project_id":   7,
			"namespace_id": 3,
			"event_id":     requestedID,
			"event_number": requestedNumber,
			"title":        "A Decision at Xuchang",
			"description":  "The court debates.",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	return result
}

func resultText(result *CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(*TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	if fields, ok := result.StructuredContent.(map[string]any); ok {
		if notice, ok := fields["id_reassigned"].(string); ok {
			parts = append(parts, notice)
		}
	}
	return strings.Join(parts, "\n")
}

func TestIDReassignmentIsReportedToTheCaller(t *testing.T) {
	backend := backendReturningEventID(t, "sanguo_shu.34", 34)
	result := callCreateEvent(t, backend.URL, "sanguo_shu.25", 25)

	text := resultText(result)
	if !strings.Contains(text, "ID REASSIGNED") {
		t.Fatalf("a silent renumber produced no notice:\n%s", text)
	}
	if !strings.Contains(text, "sanguo_shu.25") || !strings.Contains(text, "sanguo_shu.34") {
		t.Fatalf("notice must name BOTH the requested and the stored id:\n%s", text)
	}

	if !strings.Contains(text, "do NOT retry") {
		t.Fatalf("notice must tell the caller not to retry:\n%s", text)
	}

	out, ok := result.StructuredContent.(map[string]any)
	if !ok || out["event_id"] != "sanguo_shu.34" {
		t.Fatalf("structured output altered: %#v", result.StructuredContent)
	}
}

func TestNoNoticeWhenTheStoredIDMatches(t *testing.T) {
	backend := backendReturningEventID(t, "sanguo_shu.25", 25)
	result := callCreateEvent(t, backend.URL, "sanguo_shu.25", 25)

	if text := resultText(result); strings.Contains(text, "ID REASSIGNED") {
		t.Fatalf("matching ids must produce no notice:\n%s", text)
	}
}

func TestNormalisationIsNotReportedAsReassignment(t *testing.T) {
	cases := []struct{ asked, stored string }{
		{"ger", "GER"},
		{" sanguo_shu.25 ", "sanguo_shu.25"},
	}
	for _, tc := range cases {
		notice := reassignmentNotice(
			map[string]any{"country_tag": tc.asked},
			map[string]any{"country_tag": tc.stored})
		if notice != "" {
			t.Errorf("%q -> %q reported as a reassignment: %s", tc.asked, tc.stored, notice)
		}
	}
}

func TestOnlyBusinessIDFieldsAreCompared(t *testing.T) {
	notice := reassignmentNotice(
		map[string]any{
			"project_id":  "7",
			"id":          "11",
			"event_db_id": "501",
			"description": "The court debates.",
		},
		map[string]any{
			"project_id":  "9",
			"id":          "501",
			"event_db_id": "999",
			"description": "Something else entirely.",
		})
	if notice != "" {
		t.Fatalf("non-business fields must not be compared:\n%s", notice)
	}
}
