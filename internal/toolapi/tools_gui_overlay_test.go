package toolapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type guiOverlayToolFixture struct {
	t *testing.T

	mu                 sync.Mutex
	exists             bool
	revision           string
	scaffoldHash       string
	edits              []map[string]any
	previewScaffold    string
	previewCalls       int
	writeCalls         int
	validateCalls      int
	lastPreviewBody    map[string]any
	lastWriteBody      map[string]any
	lastValidationBody map[string]any
	windowEditable     bool
}

func newGUIOverlayToolFixture(t *testing.T) *guiOverlayToolFixture {
	t.Helper()
	return &guiOverlayToolFixture{
		t:               t,
		exists:          true,
		revision:        "backend-r1",
		scaffoldHash:    strings.Repeat("a", 64),
		previewScaffold: strings.Repeat("a", 64),
		windowEditable:  true,
		edits: []map[string]any{{
			"id": "set_layout:Root/Label", "target": "Root/Label", "op": "set_layout",
			"payload": map[string]any{"x": float64(5)},
		}},
	}
}

func (fixture *guiOverlayToolFixture) handler(w http.ResponseWriter, r *http.Request) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/gui-render/overlay":
		if r.URL.Query().Get("file") != "demo.gui" || !fixture.exists {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true, "exists": false, "file": r.URL.Query().Get("file"),
				"source": "import", "schema_version": 1, "edits": []any{}, "revision": "0",
				"scaffold_hash": fixture.previewScaffold, "materializable": true,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "exists": true, "id": 91, "file": "demo.gui", "source": "import",
			"schema_version": 1, "scaffold_hash": fixture.scaffoldHash,
			"edits": fixture.edits, "revision": fixture.revision, "materializable": true,
		})
	case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/gui-render/window":
		_ = json.NewEncoder(w).Encode(fixture.windowForEdits(fixture.edits))
	case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/gui-render/overlay/preview":
		fixture.previewCalls++
		fixture.lastPreviewBody = fixture.decodeBody(r)
		edits, err := guiOverlayMapSliceField(fixture.lastPreviewBody, "edits")
		if err != nil {
			fixture.t.Errorf("preview edits: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "valid": true, "write_performed": false,
			"project_id": 7, "file": "demo.gui", "normalized_file": "interface/demo.gui",
			"window_name": "Root", "current_revision": fixture.revision,
			"current_scaffold_hash": fixture.scaffoldHash, "scaffold_hash": fixture.previewScaffold,
			"materializable": true, "edits": edits, "window": fixture.windowForEdits(edits),
			"diagnostics": []map[string]any{},
		})
	case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/gui-render/overlay":
		fixture.writeCalls++
		fixture.lastWriteBody = fixture.decodeBody(r)
		edits, err := guiOverlayMapSliceField(fixture.lastWriteBody, "edits")
		if err != nil {
			fixture.t.Errorf("write edits: %v", err)
		}
		fixture.edits = edits
		fixture.exists = true
		fixture.scaffoldHash = stringField(fixture.lastWriteBody, "scaffold_hash")
		fixture.revision = "backend-r2"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "id": 91, "project_id": 7, "file": "demo.gui", "source": "import",
			"schema_version": 1, "scaffold_hash": fixture.scaffoldHash,
			"edits": fixture.edits, "revision": fixture.revision,
		})
	case r.Method == http.MethodPost && r.URL.Path == "/api/projects/7/gui-render/overlay/validate":
		fixture.validateCalls++
		fixture.lastValidationBody = fixture.decodeBody(r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "valid": true, "saved": true, "write_performed": false,
			"project_id": 7, "file": "demo.gui", "normalized_file": "interface/demo.gui",
			"window_name": "Root", "revision": fixture.revision, "scaffold_hash": fixture.scaffoldHash,
			"materializable": true, "edits": fixture.edits, "window": fixture.windowForEdits(fixture.edits),
			"diagnostics": []map[string]any{},
		})
	default:
		fixture.t.Errorf("unexpected upstream request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		http.Error(w, `{"detail":"unexpected request"}`, http.StatusNotFound)
	}
}

func (fixture *guiOverlayToolFixture) decodeBody(r *http.Request) map[string]any {
	fixture.t.Helper()
	out := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		fixture.t.Errorf("decode request: %v", err)
	}
	return out
}

func (fixture *guiOverlayToolFixture) windowForEdits(edits []map[string]any) map[string]any {
	text := "OLD"
	x := any(float64(1))
	sprite := ""
	texts := map[string]any{}
	children := []any{}
	for _, edit := range edits {
		payload, _ := edit["payload"].(map[string]any)
		if stringField(edit, "op") == "set_text" {
			projected := projectedGUIText(payload)
			_, hasText := payload["text"].(string)
			langTexts, hasLangTexts := payload["lang_texts"].(map[string]any)
			if key := strings.TrimSpace(stringField(payload, "key")); key != "" && (hasText || (hasLangTexts && len(langTexts) > 0)) {
				texts[projected] = map[string]any{"display": "generated", "kind": "loc"}
			}
		}
		if stringField(edit, "target") != "Root/Label" {
			continue
		}
		switch stringField(edit, "op") {
		case "set_text":
			text = projectedGUIText(payload)
		case "set_layout":
			if value, exists := payload["x"]; exists {
				x = value
			}
		case "swap_sprite":
			sprite = stringField(payload, "gfx_ref")
		}
	}
	children = append(children, map[string]any{
		"type": "instantTextBoxType", "name": "Label", "text": text, "sprite": sprite,
		"position": map[string]any{"x": x, "y": float64(2)},
		"size":     map[string]any{"width": float64(120), "height": float64(24)},
	})
	for _, edit := range edits {
		if stringField(edit, "target") != "Root" || stringField(edit, "op") != "add_node" {
			continue
		}
		payload, _ := edit["payload"].(map[string]any)
		if node, ok := payload["node"].(map[string]any); ok {
			children = append(children, cloneMap(node))
		}
	}
	root := map[string]any{
		"type": "containerWindowType", "name": "Root", "children": children,
	}

	for _, edit := range edits {
		if stringField(edit, "op") != "set_text" || stringField(edit, "target") == "Root/Label" {
			continue
		}
		if ref := findGUINode(root, strings.Split(stringField(edit, "target"), "/")); ref != nil {
			payload, _ := edit["payload"].(map[string]any)
			ref.Node["text"] = projectedGUIText(payload)
		}
	}
	var collectTexts func(map[string]any)
	collectTexts = func(node map[string]any) {
		if raw, ok := node["text"].(string); ok && raw != "" {
			if _, exists := texts[raw]; !exists {
				kind := "key"
				switch {
				case raw == "EXISTING_KEY":
					kind = "loc"
				case strings.Contains(raw, "[") && strings.Contains(raw, "]"):
					kind = "runtime"
				case raw != strings.ToUpper(raw) || strings.ContainsAny(raw, " \t"):
					kind = "literal"
				}
				texts[raw] = map[string]any{"display": raw, "kind": kind}
			}
		}
		for _, child := range guiNodeChildren(node) {
			collectTexts(child)
		}
	}
	collectTexts(root)
	sprites := map[string]any{
		"GFX_changed": map[string]any{"resolved": true, "width": 64, "height": 64, "frames": 1},
	}
	window := map[string]any{
		"ok": true, "name": "Root", "file": "interface/demo.gui", "source": "import",
		"sprites": sprites, "texts": texts,
		"node": root,
	}
	if fixture.windowEditable {
		window["revision"] = fixture.revision
		window["scaffold_hash"] = firstNonBlank(fixture.scaffoldHash, fixture.previewScaffold)
	}
	return window
}

func guiOverlayToolSession(t *testing.T, fixture *guiOverlayToolFixture) *testSession {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(fixture.handler))
	t.Cleanup(upstream.Close)
	proxy := NewRESTProxy(upstream.URL)
	server := newProxyOnlyTestServer(t, func(instance *Server) {
		registerGUIOverlayTools(instance, proxy)
	})
	return connectClient(t, server.URL)
}

func guiOverlayBatchArgs() map[string]any {
	return map[string]any{
		"project_id": 7, "file": "interface/demo.gui", "window_name": "Root",
		"expected_revision": "backend-r1", "expected_scaffold_hash": strings.Repeat("a", 64),
		"edits": []any{map[string]any{
			"target": "Root/Label", "op": "set_text", "payload": map[string]any{"text": "NEW"},
		}},
	}
}

func TestGetGUIOverlayPreservesStoredAliasAndBackendRevision(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	result := callTool(t, guiOverlayToolSession(t, fixture), "get_gui_overlay", map[string]any{
		"project_id": 7, "file": "interface/demo.gui",
	})
	if result.IsError {
		t.Fatalf("get_gui_overlay: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["file"] != "demo.gui" || out["normalized_file"] != "interface/demo.gui" {
		t.Fatalf("stored/normalized file lost: %#v", out)
	}
	if out["revision"] != "backend-r1" {
		t.Fatalf("revision = %v, want backend-r1", out["revision"])
	}
}

func TestGetGUIWindowReturnsImportedScaffoldHashWithoutOverlayRow(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	fixture.exists = false
	fixture.revision = "0"
	result := callTool(t, guiOverlayToolSession(t, fixture), "get_gui_window", map[string]any{
		"project_id": 7, "file": "interface/demo.gui", "window_name": "Root",
	})
	if result.IsError {
		t.Fatalf("get_gui_window: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["overlay_exists"] != false || out["overlay_revision"] != "0" {
		t.Fatalf("absent overlay envelope is wrong: %#v", out)
	}
	if out["scaffold_hash"] != fixture.previewScaffold || out["materializable"] != true {
		t.Fatalf("imported scaffold fingerprint missing: %#v", out)
	}
}

func TestGUIOverlayRefusesShadowedEffectiveWindowBeforePreviewOrWrite(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	fixture.windowEditable = false
	windowResult := callTool(t, guiOverlayToolSession(t, fixture), "get_gui_window", map[string]any{
		"project_id": 7, "file": "interface/demo.gui", "window_name": "Root",
	})
	if windowResult.IsError {
		t.Fatalf("get shadowed window: %s", contentText(windowResult))
	}
	window := windowResult.StructuredContent.(map[string]any)
	if window["materializable"] != false || window["effective_overlay_editable"] != false {
		t.Fatalf("shadowed window was advertised as editable: %#v", window)
	}
	applyResult := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", guiOverlayBatchArgs())
	if !applyResult.IsError || !strings.Contains(contentText(applyResult), "not safely editable") {
		t.Fatalf("shadowed window was not refused: %s", contentText(applyResult))
	}
	if fixture.previewCalls != 0 || fixture.writeCalls != 0 {
		t.Fatalf("shadowed window reached preview/write: %d/%d", fixture.previewCalls, fixture.writeCalls)
	}
}

func TestPreviewGUIOverlayUsesAuthoritativeUnsavedEndpoint(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", guiOverlayBatchArgs())
	if result.IsError {
		t.Fatalf("preview_gui_overlay: %s", contentText(result))
	}
	if fixture.previewCalls != 1 || fixture.writeCalls != 0 {
		t.Fatalf("preview calls=%d writes=%d", fixture.previewCalls, fixture.writeCalls)
	}
	if fixture.lastPreviewBody["file"] != "demo.gui" || fixture.lastPreviewBody["expected_revision"] != "backend-r1" {
		t.Fatalf("preview did not preserve alias/CAS: %#v", fixture.lastPreviewBody)
	}
	edits, _ := guiOverlayMapSliceField(fixture.lastPreviewBody, "edits")
	if len(edits) != 2 {
		t.Fatalf("preview must receive complete merged candidate, got %#v", edits)
	}
	out := result.StructuredContent.(map[string]any)
	if out["write_performed"] != false || out["scaffold_hash"] != fixture.previewScaffold {
		t.Fatalf("preview output is not authoritative: %#v", out)
	}
}

func TestApplyGUIOverlayPreviewsThenConditionallyWritesAndValidates(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", guiOverlayBatchArgs())
	if result.IsError {
		t.Fatalf("apply_gui_overlay_edits: %s", contentText(result))
	}
	if fixture.previewCalls != 1 || fixture.writeCalls != 1 || fixture.validateCalls != 1 {
		t.Fatalf("preview/write/validate = %d/%d/%d", fixture.previewCalls, fixture.writeCalls, fixture.validateCalls)
	}
	if fixture.lastWriteBody["file"] != "demo.gui" || fixture.lastWriteBody["expected_revision"] != "backend-r1" {
		t.Fatalf("write lost stored alias or revision: %#v", fixture.lastWriteBody)
	}
	if fixture.lastWriteBody["expected_scaffold_hash"] != fixture.previewScaffold || fixture.lastWriteBody["scaffold_hash"] != fixture.previewScaffold {
		t.Fatalf("write did not forward preview scaffold hash: %#v", fixture.lastWriteBody)
	}
	if fixture.lastValidationBody["expected_revision"] != "backend-r2" || fixture.lastValidationBody["file"] != "demo.gui" {
		t.Fatalf("saved revision/alias not validated: %#v", fixture.lastValidationBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["persistence_verified"] != true || out["authoritative_validation"] != true || out["readback_verified"] != true {
		t.Fatalf("apply was not fully verified: %#v", out)
	}
}

func TestApplyGUIOverlayAcceptsResolvableExistingSpriteReference(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root/Label", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_changed"},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("apply existing sprite reference: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["references_verified"] != true {
		t.Fatalf("resolvable sprite reference was not verified: %#v", out)
	}
}

func TestApplyGUIOverlayDoesNotCompleteForUnresolvedSpriteReference(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root/Label", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_missing"},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if !result.IsError || !strings.Contains(contentText(result), "sprite_unresolved") {
		t.Fatalf("unresolved sprite reference was not refused before write: %s", contentText(result))
	}
	if fixture.previewCalls != 1 || fixture.writeCalls != 0 {
		t.Fatalf("unresolved sprite reached write: preview=%d write=%d", fixture.previewCalls, fixture.writeCalls)
	}
}

func TestPreviewGUIOverlayRefusesUnresolvedExistingLocalisationKey(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root/Label", "op": "set_text",
		"payload": map[string]any{"key": "DOES_NOT_EXIST", "is_new": false},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", args)
	if !result.IsError || !strings.Contains(contentText(result), "localisation_unresolved") {
		t.Fatalf("unresolved localisation key was not refused: %s", contentText(result))
	}
	if fixture.previewCalls != 1 || fixture.writeCalls != 0 {
		t.Fatalf("unresolved localisation reached write: preview=%d write=%d", fixture.previewCalls, fixture.writeCalls)
	}
}

func TestPreviewGUIOverlayRefusesUnresolvedLocalisationInAddedSubtree(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root", "op": "add_node",
		"payload": map[string]any{"node": map[string]any{
			"type": "containerWindowType", "name": "StatusPanel",
			"children": []any{map[string]any{
				"type": "instantTextBoxType", "name": "Caption", "text": "MISSING_STATUS_TEXT",
			}},
		}},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", args)
	if !result.IsError || !strings.Contains(contentText(result), "localisation_unresolved") || !strings.Contains(contentText(result), "Root/StatusPanel/Caption") {
		t.Fatalf("unresolved nested add_node localisation was not refused: %s", contentText(result))
	}
	if fixture.previewCalls != 1 || fixture.writeCalls != 0 {
		t.Fatalf("unresolved added-node localisation reached write: preview=%d write=%d", fixture.previewCalls, fixture.writeCalls)
	}
}

func TestApplyGUIOverlayAcceptsResolvedAndLiteralTextInAddedSubtree(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root", "op": "add_node",
		"payload": map[string]any{"node": map[string]any{
			"type": "containerWindowType", "name": "StatusPanel",
			"children": []any{
				map[string]any{"type": "instantTextBoxType", "name": "Title", "text": "EXISTING_KEY"},
				map[string]any{"type": "instantTextBoxType", "name": "Value", "text": "Inline status"},
			},
		}},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("resolved/literal add_node text: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["localisation_verified"] != true {
		t.Fatalf("resolved/literal add_node text was not verified: %#v", out)
	}
}

func TestApplyGUIOverlayAcceptsAddedNodeWithSameBatchGeneratedLocalisation(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{
		map[string]any{
			"target": "Root", "op": "add_node",
			"payload": map[string]any{"node": map[string]any{
				"type": "instantTextBoxType", "name": "Status", "text": "PENDING_STATUS_TEXT",
			}},
		},
		map[string]any{
			"target": "Root/Status", "op": "set_text",
			"payload": map[string]any{
				"key": "STATUS_TEXT", "is_new": true,
				"lang_texts": map[string]any{"en": "Ready", "zh": "就绪"},
			},
		},
	}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("add_node plus generated localisation: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["localisation_verified"] != true || out["references_verified"] != true {
		t.Fatalf("same-batch generated localisation was not verified: %#v", out)
	}
}

func TestApplyGUIOverlayVerifiesGeneratedMultilingualLocalisation(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root/Label", "op": "set_text",
		"payload": map[string]any{
			"key": "NEW_LABEL", "is_new": true,
			"lang_texts": map[string]any{"en": "Hello", "zh": "你好"},
		},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("generated multilingual localisation: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["localisation_verified"] != true || out["references_verified"] != true {
		t.Fatalf("generated localisation was not verified: %#v", out)
	}
}

func TestApplyGUIOverlayNormalizesLegacyBrowserPayloads(t *testing.T) {
	t.Run("sprite override file", func(t *testing.T) {
		fixture := newGUIOverlayToolFixture(t)
		fixture.edits = []map[string]any{{
			"id": "swap_sprite:Root/Label", "target": "Root/Label", "op": "swap_sprite",
			"payload": map[string]any{"gfx_ref": "GFX_changed", "override_file": "changed.png"},
		}}
		result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", guiOverlayBatchArgs())
		if result.IsError {
			t.Fatalf("legacy sprite payload: %s", contentText(result))
		}
		edits, _ := guiOverlayMapSliceField(fixture.lastWriteBody, "edits")
		payload, _ := edits[0]["payload"].(map[string]any)
		if _, exists := payload["override_file"]; exists {
			t.Fatalf("legacy override_file was carried forward: %#v", edits)
		}
	})

	t.Run("multilingual text without key", func(t *testing.T) {
		fixture := newGUIOverlayToolFixture(t)
		fixture.edits = []map[string]any{{
			"target": "Root/Label", "op": "set_text",
			"payload": map[string]any{"lang_texts": map[string]any{"en": "Old", "zh": "旧"}},
		}}
		args := guiOverlayBatchArgs()
		args["edits"] = []any{map[string]any{
			"target": "Root/Label", "op": "set_layout", "payload": map[string]any{"x": 9},
		}}
		result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
		if result.IsError {
			t.Fatalf("legacy multilingual payload: %s", contentText(result))
		}
		edits, _ := guiOverlayMapSliceField(fixture.lastWriteBody, "edits")
		payload, _ := edits[0]["payload"].(map[string]any)
		if strings.TrimSpace(stringField(payload, "key")) == "" || payload["is_new"] != true {
			t.Fatalf("legacy multilingual payload was not assigned a stable key: %#v", edits)
		}
	})
}

func TestApplyGUIOverlayAtomicallyReplacesExistingAddedNode(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	fixture.edits = []map[string]any{{
		"id": "add_node:Root/Badge", "target": "Root", "op": "add_node",
		"payload": map[string]any{"node": map[string]any{"type": "instantTextBoxType", "name": "Badge", "text": "old"}},
	}}
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root", "op": "add_node",
		"payload": map[string]any{"node": map[string]any{"type": "instantTextBoxType", "name": "Badge", "text": "new"}},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("replace existing add_node: %s", contentText(result))
	}
	if len(fixture.edits) != 1 || stringField(nodeField(fixture.edits[0]["payload"].(map[string]any), "node"), "text") != "new" {
		t.Fatalf("add_node was not replaced atomically: %#v", fixture.edits)
	}
}

func TestApplyGUIOverlayFirstWriteUsesAbsentOverlayBaseHash(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	fixture.exists = false
	fixture.revision = "0"
	fixture.edits = nil
	args := guiOverlayBatchArgs()
	args["expected_revision"] = "0"
	args["expected_scaffold_hash"] = fixture.previewScaffold
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("first overlay write: %s", contentText(result))
	}
	if fixture.lastPreviewBody["expected_scaffold_hash"] != fixture.previewScaffold {
		t.Fatalf("preview did not CAS the absent-row base hash: %#v", fixture.lastPreviewBody)
	}
	if fixture.lastWriteBody["file"] != "interface/demo.gui" || fixture.lastWriteBody["expected_revision"] != "0" || fixture.lastWriteBody["expected_scaffold_hash"] != fixture.previewScaffold {
		t.Fatalf("first write lost normalized file/base CAS: %#v", fixture.lastWriteBody)
	}
}

func TestPreviewGUIOverlayCanRemoveAnExistingEditWithoutWriting(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{}
	args["remove_edit_ids"] = []any{"set_layout:Root/Label"}
	result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", args)
	if result.IsError {
		t.Fatalf("preview removal: %s", contentText(result))
	}
	if fixture.writeCalls != 0 || fixture.previewCalls != 1 {
		t.Fatalf("removal preview calls/writes = %d/%d", fixture.previewCalls, fixture.writeCalls)
	}
	edits, _ := guiOverlayMapSliceField(fixture.lastPreviewBody, "edits")
	if len(edits) != 0 {
		t.Fatalf("removed edit remained in candidate: %#v", edits)
	}
}

func TestApplyGUIOverlayCanRestoreOneExistingEdit(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{}
	args["remove_edit_ids"] = []any{"set_layout:Root/Label"}
	result := callTool(t, guiOverlayToolSession(t, fixture), "apply_gui_overlay_edits", args)
	if result.IsError {
		t.Fatalf("apply removal: %s", contentText(result))
	}
	if len(fixture.edits) != 0 {
		t.Fatalf("removed edit was persisted: %#v", fixture.edits)
	}
	out := result.StructuredContent.(map[string]any)
	if out["completed"] != true || out["requested"] != float64(1) {
		t.Fatalf("restore was not fully verified: %#v", out)
	}
}

func TestGUIOverlayRemovalRefusesMissingAndConflictingIDs(t *testing.T) {
	for _, test := range []struct {
		name  string
		id    string
		edits []any
		want  string
	}{
		{name: "missing", id: "set_text:Root/Label", edits: []any{}, want: "does not exist"},
		{name: "remove and upsert", id: "set_layout:Root/Label", edits: []any{map[string]any{
			"target": "Root/Label", "op": "set_layout", "payload": map[string]any{"x": 7},
		}}, want: "both removes and upserts"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newGUIOverlayToolFixture(t)
			args := guiOverlayBatchArgs()
			args["edits"] = test.edits
			args["remove_edit_ids"] = []any{test.id}
			result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", args)
			if !result.IsError || !strings.Contains(contentText(result), test.want) {
				t.Fatalf("removal refusal = %s, want %q", contentText(result), test.want)
			}
			if fixture.previewCalls != 0 || fixture.writeCalls != 0 {
				t.Fatalf("invalid removal reached preview/write: %d/%d", fixture.previewCalls, fixture.writeCalls)
			}
		})
	}
}

func TestGUIOverlayEditsOneWindowBesideAnotherWindowsSavedEdits(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	fixture.edits = append(fixture.edits, map[string]any{
		"id": "set_layout:Other/Title", "target": "Other/Title", "op": "set_layout",
		"payload": map[string]any{"y": float64(9)},
	})
	session := guiOverlayToolSession(t, fixture)
	result := callTool(t, session, "apply_gui_overlay_edits", guiOverlayBatchArgs())
	if result.IsError {
		t.Fatalf("apply_gui_overlay_edits: %s", contentText(result))
	}
	if out := result.StructuredContent.(map[string]any); out["completed"] != true {
		t.Fatalf("apply beside another window's edit was not completed: %#v", out)
	}
	kept := false
	for _, edit := range fixture.edits {
		kept = kept || stringField(edit, "target") == "Other/Title"
	}
	if !kept {
		t.Fatalf("the other window's edit was dropped: %#v", fixture.edits)
	}
	result = callTool(t, session, "validate_gui_overlay", map[string]any{
		"project_id": 7, "file": "interface/demo.gui", "window_name": "Root",
	})
	if out := result.StructuredContent.(map[string]any); result.IsError || out["valid"] != true {
		t.Fatalf("validate beside another window's edit: %s %#v", contentText(result), result.StructuredContent)
	}
}

func TestValidateGUIOverlayUsesAuthoritativeEndpointAndEffectiveReadback(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	result := callTool(t, guiOverlayToolSession(t, fixture), "validate_gui_overlay", map[string]any{
		"project_id": 7, "file": "interface/demo.gui", "window_name": "Root", "expected_revision": "backend-r1",
	})
	if result.IsError {
		t.Fatalf("validate_gui_overlay: %s", contentText(result))
	}
	if fixture.validateCalls != 1 || fixture.previewCalls != 0 || fixture.writeCalls != 0 {
		t.Fatalf("validate/preview/write = %d/%d/%d", fixture.validateCalls, fixture.previewCalls, fixture.writeCalls)
	}
	out := result.StructuredContent.(map[string]any)
	if out["valid"] != true || out["completed"] != true || out["readback_verified"] != true {
		t.Fatalf("overlay validation failed: %#v", out)
	}
}

func TestGUIOverlayRejectsUnmaterializableAddedFieldsBeforePreviewOrWrite(t *testing.T) {
	fixture := newGUIOverlayToolFixture(t)
	args := guiOverlayBatchArgs()
	args["edits"] = []any{map[string]any{
		"target": "Root", "op": "add_node", "payload": map[string]any{"node": map[string]any{
			"type": "iconType", "name": "Badge", "sprite": "GFX_badge", "frame_var": "unsupported",
		}},
	}}
	result := callTool(t, guiOverlayToolSession(t, fixture), "preview_gui_overlay", args)
	if !result.IsError || !strings.Contains(contentText(result), "not materialized") {
		t.Fatalf("unexportable add_node field was not refused: %s", contentText(result))
	}
	if fixture.previewCalls != 0 || fixture.writeCalls != 0 {
		t.Fatalf("invalid edit reached mutation endpoints: preview=%d write=%d", fixture.previewCalls, fixture.writeCalls)
	}
}

func TestGUIOverlayRejectsStringsThatWouldBreakTheExportedFiles(t *testing.T) {
	for name, edit := range map[string]guiOverlayEditInput{
		"quote in added text": {Target: "Root", Op: "add_node", Payload: map[string]any{"node": map[string]any{
			"type": "instantTextBoxType", "name": "Added", "text": `Hi" } buttonType = { name = "evil`,
		}}},
		"line break in added sprite": {Target: "Root", Op: "add_node", Payload: map[string]any{"node": map[string]any{
			"type": "iconType", "name": "Added", "sprite": "GFX_a\nGFX_b",
		}}},
		"quote in added name": {Target: "Root", Op: "add_node", Payload: map[string]any{"node": map[string]any{
			"type": "iconType", "name": `Added"x`,
		}}},
		"space in key":             {Target: "Root/Label", Op: "set_text", Payload: map[string]any{"key": "A B", "is_new": false}},
		"blank override_file":      {Target: "Root/Label", Op: "swap_sprite", Payload: map[string]any{"gfx_ref": "GFX_changed", "override_file": " "}},
		"non-string override_file": {Target: "Root/Label", Op: "swap_sprite", Payload: map[string]any{"gfx_ref": "GFX_changed", "override_file": 5}},
		"colon in key":             {Target: "Root/Label", Op: "set_text", Payload: map[string]any{"key": "A:0", "is_new": false}},
	} {
		edit := edit
		if err := validateGUIEditSchema(&edit, "Root"); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	ok := guiOverlayEditInput{Target: "Root/Label", Op: "set_text", Payload: map[string]any{"key": "GER_focus.1-t", "is_new": false}}
	if err := validateGUIEditSchema(&ok, "Root"); err != nil {
		t.Errorf("an ordinary key was refused: %v", err)
	}
}

func TestGUIOverlaySchemasCoverEverySupportedOperation(t *testing.T) {
	tests := []guiOverlayEditInput{
		{Target: "Root/Label", Op: "set_text", Payload: map[string]any{"text": "Changed"}},
		{Target: "Root/Label", Op: "set_text", Payload: map[string]any{"key": "EXISTING_KEY", "is_new": false}},
		{Target: "Root/Label", Op: "set_text", Payload: map[string]any{"key": "NEW_KEY", "is_new": true, "lang_texts": map[string]any{"en": "Changed"}}},
		{Target: "Root/Label", Op: "swap_sprite", Payload: map[string]any{"gfx_ref": "GFX_existing"}},
		{Target: "Root/Label", Op: "set_layout", Payload: map[string]any{"x": 1, "height": 22}},
		{Target: "Root/Label", Op: "set_orientation", Payload: map[string]any{"value": "CENTER"}},
		{Target: "Root/Label", Op: "set_origo", Payload: map[string]any{"value": "UPPER_LEFT"}},
		{Target: "Root", Op: "add_node", Payload: map[string]any{"node": map[string]any{"type": "iconType", "name": "Badge", "sprite": "GFX_badge"}}},
		{Target: "Root/Label", Op: "delete_node", Payload: map[string]any{}},
		{Target: "Root/Label", Op: "hide_node", Payload: map[string]any{}},
	}
	for _, edit := range tests {
		edit := edit
		t.Run(edit.Op, func(t *testing.T) {
			if err := validateGUIEditSchema(&edit, "Root"); err != nil {
				t.Fatalf("%s schema: %v", edit.Op, err)
			}
			if edit.ID == "" {
				t.Fatal("canonical edit id was not assigned")
			}
		})
	}
}

func TestGUIOverlayRejectsNewLocalisationKeyWithoutText(t *testing.T) {
	edit := guiOverlayEditInput{
		Target: "Root/Label",
		Op:     "set_text",
		Payload: map[string]any{
			"key": "NEW_BUT_EMPTY",
		},
	}
	err := validateGUIEditSchema(&edit, "Root")
	if err == nil || !strings.Contains(err.Error(), "new key requires") {
		t.Fatalf("new key without localisation text was accepted: %v", err)
	}
}

func TestGUIOverlayRejectsMultilingualTextWithoutStableKey(t *testing.T) {
	edit := guiOverlayEditInput{
		Target: "Root/Label",
		Op:     "set_text",
		Payload: map[string]any{
			"lang_texts": map[string]any{"en": "Hello", "zh": "你好"},
		},
	}
	err := validateGUIEditSchema(&edit, "Root")
	if err == nil || !strings.Contains(err.Error(), "requires key") {
		t.Fatalf("multilingual text without stable key was accepted: %v", err)
	}
}

func TestGUIOverlayRejectsLegacyInlineSpriteOverride(t *testing.T) {
	edit := guiOverlayEditInput{
		Target:  "Root/Label",
		Op:      "swap_sprite",
		Payload: map[string]any{"gfx_ref": "GFX_changed", "override_file": "changed.png"},
	}
	err := validateGUIEditSchema(&edit, "Root")
	if err == nil || !strings.Contains(err.Error(), "apply artwork") {
		t.Fatalf("inline sprite override was accepted without export/readback parity: %v", err)
	}
}

func TestFindGUINodeRejectsAmbiguousSiblingNamePath(t *testing.T) {
	root := map[string]any{
		"name": "Root",
		"children": []any{
			map[string]any{"name": "Duplicate"},
			map[string]any{"name": "Duplicate"},
		},
	}
	if ref := findGUINode(root, []string{"Root", "Duplicate"}); ref != nil {
		t.Fatalf("ambiguous NAME-PATH resolved to %#v", ref)
	}
}

func TestGUIOverlayHideNodeIsCheckedAndReadBack(t *testing.T) {
	for _, bad := range []guiOverlayEditInput{
		{Target: "Root", Op: "hide_node", Payload: map[string]any{}},
		{Target: "Root/Label", Op: "hide_node", Payload: map[string]any{"x": 1}},
	} {
		bad := bad
		if err := validateGUIEditSchema(&bad, "Root"); err == nil {
			t.Errorf("%+v was accepted", bad)
		}
	}
	window := func(raw map[string]any) map[string]any {
		return map[string]any{"name": "Root", "children": []any{map[string]any{"name": "Label", "raw": raw}}}
	}
	edits := []guiOverlayEditInput{{Target: "Root/Label", Op: "hide_node", Payload: map[string]any{}}}
	if ok, diagnostics := verifyGUIEditsReadback(window(map[string]any{"gameview_hidden": "yes"}), edits); !ok {
		t.Errorf("a hidden element was not read back: %v", diagnostics)
	}
	if ok, _ := verifyGUIEditsReadback(window(map[string]any{}), edits); ok {
		t.Error("an element shown as before passed as hidden")
	}
	if ok, _ := verifyGUIEditsReadback(map[string]any{"name": "Root"}, edits); ok {
		t.Error("a missing element passed as hidden")
	}
}

func TestGUIOverlayRefusesSwappingTheGameSwappedBackgrounds(t *testing.T) {
	for _, target := range []string{"frontend_background", "load_screen", "frontend_background/inner"} {
		window, _, _ := strings.Cut(target, "/")
		edit := guiOverlayEditInput{Target: target, Op: "swap_sprite", Payload: map[string]any{"gfx_ref": "GFX_gv_menu"}}
		err := validateGUIEditSchema(&edit, window)
		if err == nil || !strings.Contains(err.Error(), "remove_edit_ids") {
			t.Errorf("%s: %v", target, err)
		}
	}
	ok := guiOverlayEditInput{Target: "Root/Label", Op: "swap_sprite", Payload: map[string]any{"gfx_ref": "GFX_gv_label"}}
	if err := validateGUIEditSchema(&ok, "Root"); err != nil {
		t.Errorf("an ordinary swap was refused: %v", err)
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
