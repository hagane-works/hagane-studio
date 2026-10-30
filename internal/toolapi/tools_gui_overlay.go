package toolapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
)

const (
	guiOverlayBatchLimit = 256
	guiOverlayNodeLimit  = 512
	guiOverlayDepthLimit = 32
)

var guiEngineSwappedWindows = map[string]bool{"frontend_background": true, "load_screen": true}

var guiOverlayOps = map[string]bool{
	"set_text": true, "swap_sprite": true, "set_layout": true,
	"set_orientation": true, "set_origo": true,
	"add_node": true, "delete_node": true, "hide_node": true,
}

var guiOverlayNodeTypes = map[string]bool{
	"containerwindowtype": true, "windowtype": true, "icontype": true,
	"buttontype": true, "guibuttontype": true, "instanttextboxtype": true,
	"textboxtype": true, "editboxtype": true, "checkboxtype": true,
	"listboxtype": true, "smoothlistboxtype": true, "gridboxtype": true,
	"scrollbartype": true, "extendedscrollbartype": true,
	"overlappingelementsboxtype": true, "positiontype": true,
	"shieldtype": true, "progressbartype": true,
	"dropdownboxtype": true, "expandedwindow": true,
}

var guiOverlayAnchors = map[string]bool{
	"UPPER_LEFT": true, "CENTER_UP": true, "UPPER_RIGHT": true,
	"CENTER_LEFT": true, "CENTER": true, "CENTER_RIGHT": true,
	"LOWER_LEFT": true, "CENTER_DOWN": true, "LOWER_RIGHT": true,
}

type guiWindowToolIn struct {
	ProjectID  int64  `json:"project_id"`
	WindowName string `json:"window_name" jsonschema:"exact GUI window name"`
	File       string `json:"file,omitempty" jsonschema:"exact source .gui file; when supplied it must own window_name"`
}

type guiOverlayToolIn struct {
	ProjectID int64  `json:"project_id"`
	File      string `json:"file" jsonschema:"source .gui file (from get_gui_window)"`
}

type guiOverlayEditInput struct {
	ID      string         `json:"id,omitempty" jsonschema:"optional canonical id; when supplied it must match op and NAME-PATH target"`
	Target  string         `json:"target" jsonschema:"stable NAME-PATH beginning with the exact window name"`
	Op      string         `json:"op" jsonschema:"set_text, swap_sprite, set_layout, set_orientation, set_origo, add_node, delete_node, or hide_node (hide a base-game element; delete_node only removes what add_node added)"`
	Payload map[string]any `json:"payload,omitempty"`
}

type guiOverlayBatchIn struct {
	ProjectID            int64                 `json:"project_id"`
	File                 string                `json:"file" jsonschema:"source .gui file (from get_gui_window)"`
	WindowName           string                `json:"window_name" jsonschema:"exact owning window; all NAME-PATHs must begin with it"`
	ExpectedRevision     string                `json:"expected_revision" jsonschema:"revision returned by get_gui_overlay; stale revisions are refused before writing"`
	ExpectedScaffoldHash string                `json:"expected_scaffold_hash" jsonschema:"exact scaffold_hash returned by get_gui_overlay; a scaffold (imported mod or base game) has a real hash even before the first overlay row"`
	Edits                []guiOverlayEditInput `json:"edits,omitempty" jsonschema:"structured edits to upsert; edits plus remove_edit_ids must contain 1 to 256 operations"`
	RemoveEditIDs        []string              `json:"remove_edit_ids,omitempty" jsonschema:"canonical edit ids from get_gui_overlay to remove before upserting edits"`
}

type guiOverlayValidateIn struct {
	ProjectID        int64  `json:"project_id"`
	File             string `json:"file"`
	WindowName       string `json:"window_name"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
}

type guiOverlayState struct {
	Payload             map[string]any
	File                string
	Normalized          string
	Source              string
	ScaffoldHash        string
	CurrentScaffoldHash string
	ScaffoldHashPinned  bool
	ScaffoldDrift       bool
	SchemaVersion       int
	Exists              bool
	Materializable      bool
	Edits               []map[string]any
	Revision            string
}

func registerGUIOverlayTools(s *Server, proxy *RESTProxy) {
	registerGetGUIWindow(s, proxy)
	registerGetGUIOverlay(s, proxy)
	registerPreviewGUIOverlay(s, proxy)
	registerApplyGUIOverlayEdits(s, proxy)
	registerValidateGUIOverlay(s, proxy)
}

func registerGetGUIWindow(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_gui_window",
		Description: "Read one exact effective GUI window, as the game view renders it, and its source file, node tree, sprites, texts and fonts. " +
			"Use its NAME-PATH node names and overlay_revision before editing; never guess a file or fall back to raw_file.",
	}, func(ctx context.Context, _ *CallToolRequest, in guiWindowToolIn) (*CallToolResult, map[string]any, error) {
		if err := validateGUIProjectContext(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		window, normalized, err := readGUIWindow(ctx, proxy, in.ProjectID, in.WindowName, in.File)
		if err != nil {
			return nil, nil, err
		}
		state, err := readGUIOverlay(ctx, proxy, in.ProjectID, normalized)
		if err != nil {
			return nil, nil, err
		}
		out := cloneMap(window)
		out["project_id"] = in.ProjectID
		out["normalized_file"] = normalized
		out["overlay_revision"] = state.Revision
		out["overlay_exists"] = state.Exists
		out["scaffold_hash"] = state.ScaffoldHash
		out["current_scaffold_hash"] = state.CurrentScaffoldHash
		out["scaffold_hash_pinned"] = state.ScaffoldHashPinned
		out["scaffold_drift"] = state.ScaffoldDrift
		windowRevision := strings.TrimSpace(stringField(window, "revision"))
		windowScaffold := strings.TrimSpace(stringField(window, "scaffold_hash"))
		effectiveEditable := state.Materializable && !state.ScaffoldDrift &&
			windowRevision == state.Revision && windowScaffold == state.ScaffoldHash
		out["materializable"] = effectiveEditable
		out["effective_overlay_editable"] = effectiveEditable
		if !state.Materializable {
			out["diagnostics"] = []map[string]any{guiOverlayDiagnostic("error", "scaffold_unavailable", "Neither the imported mod nor the base game has this GUI file, so it can be viewed but not edited as a structured overlay.", "", "")}
		} else if !effectiveEditable {
			out["diagnostics"] = []map[string]any{guiOverlayDiagnostic("error", "effective_window_not_editable", "The effective window is shadowed, drifted, or lacks the exact overlay revision/scaffold envelope; refusing a structural write.", "", "")}
		}
		return nil, out, nil
	})
}

func registerGetGUIOverlay(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_gui_overlay",
		Description: "Read the complete structured overlay for one safe interface/*.gui path. Returns a deterministic revision and scaffold_hash for optimistic editing. " +
			"This never exposes or edits raw source files.",
	}, func(ctx context.Context, _ *CallToolRequest, in guiOverlayToolIn) (*CallToolResult, map[string]any, error) {
		if err := validateGUIProjectContext(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		state, err := readGUIOverlay(ctx, proxy, in.ProjectID, in.File)
		if err != nil {
			return nil, nil, err
		}
		out := guiOverlayStateResult(state, in.ProjectID)
		return nil, out, nil
	})
}

func registerPreviewGUIOverlay(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "preview_gui_overlay",
		Description: "Validate and project a batch of structured GUI overlay edit upserts/removals without writing. Targets are exact NAME-PATHs in window_name; remove_edit_ids restores selected existing edits. " +
			"The project service renders the complete candidate against the exact scaffold — the imported mod's file, or the base game's for a project that did not import it — and returns the authoritative window and scaffold_hash; no overlay is persisted.",
	}, func(ctx context.Context, _ *CallToolRequest, in guiOverlayBatchIn) (*CallToolResult, map[string]any, error) {
		state, _, normalized, canonical, changes, err := prepareGUIOverlayBatch(ctx, proxy, in)
		if err != nil {
			return nil, nil, err
		}
		merged := mergeGUIOverlayEdits(state.Edits, canonical)
		candidateEdits, err := decodeAndValidateGUIOverlayEdits(merged, in.WindowName)
		if err != nil {
			return nil, nil, err
		}
		preview, err := previewGUIOverlayCandidate(ctx, proxy, in.ProjectID, state, in.WindowName, merged)
		if err != nil {
			return nil, nil, err
		}
		materialized, materializedDiagnostics := verifyGUIEditsReadback(nodeField(nodeField(preview, "window"), "node"), candidateEdits)
		if !materialized {
			return nil, nil, fmt.Errorf("authoritative GUI overlay preview did not materialize every edit: %v", materializedDiagnostics)
		}
		previewWindow := nodeField(preview, "window")
		spriteReferencesVerified, spriteDiagnostics := validateGUIOverlaySpriteReferences(previewWindow, candidateEdits)
		localisationVerified, localisationDiagnostics := validateGUIOverlayLocalisationReferences(previewWindow, candidateEdits)
		if !spriteReferencesVerified || !localisationVerified {
			diagnostics := append(spriteDiagnostics, localisationDiagnostics...)
			return nil, nil, fmt.Errorf("authoritative GUI overlay preview has unresolved references: %v", diagnostics)
		}
		out := cloneMap(preview)
		out["project_id"] = in.ProjectID
		out["file"] = guiOverlayMutationFile(state, normalized)
		out["normalized_file"] = normalized
		out["window_name"] = in.WindowName
		out["current_revision"] = state.Revision
		out["current_scaffold_hash"] = state.ScaffoldHash
		out["write_performed"] = false
		out["changes"] = changes
		out["proposed_edits"] = merged
		out["removed_edit_ids"] = in.RemoveEditIDs
		out["sprite_references_verified"] = true
		out["localisation_verified"] = true
		out["references_verified"] = true
		return nil, out, nil
	})
}

func registerApplyGUIOverlayEdits(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "apply_gui_overlay_edits",
		Description: "Authoritatively preview, then atomically apply a validated batch of NAME-PATH GUI edit upserts/removals into one overlay document using revision and scaffold compare-and-swap; remove_edit_ids restores selected existing edits. Finally re-read and validate the saved window as the game view renders it. " +
			"Supports set_text, swap_sprite, set_layout, set_orientation, set_origo, add_node and delete_node. " +
			"Pass the exact revision and scaffold_hash from get_gui_overlay. completed=false means it was persisted but is not materialized in the game view/export and must not be reported as finished. Never use raw_file as a fallback.",
	}, func(ctx context.Context, _ *CallToolRequest, in guiOverlayBatchIn) (*CallToolResult, map[string]any, error) {
		state, window, normalized, canonical, _, err := prepareGUIOverlayBatch(ctx, proxy, in)
		if err != nil {
			return nil, nil, err
		}
		merged := mergeGUIOverlayEdits(state.Edits, canonical)
		candidateEdits, err := decodeAndValidateGUIOverlayEdits(merged, in.WindowName)
		if err != nil {
			return nil, nil, err
		}
		preview, err := previewGUIOverlayCandidate(ctx, proxy, in.ProjectID, state, in.WindowName, merged)
		if err != nil {
			return nil, nil, err
		}
		candidateScaffoldHash := strings.TrimSpace(stringField(preview, "scaffold_hash"))
		if candidateScaffoldHash == "" {
			return nil, nil, fmt.Errorf("authoritative GUI overlay preview returned no scaffold_hash")
		}
		previewVerified, previewDiagnostics := verifyGUIEditsReadback(nodeField(nodeField(preview, "window"), "node"), candidateEdits)
		if !previewVerified {
			return nil, nil, fmt.Errorf("authoritative GUI overlay preview did not materialize every edit: %v", previewDiagnostics)
		}
		previewWindow := nodeField(preview, "window")
		previewSpritesVerified, previewSpriteDiagnostics := validateGUIOverlaySpriteReferences(previewWindow, candidateEdits)
		previewLocalisationVerified, previewLocalisationDiagnostics := validateGUIOverlayLocalisationReferences(previewWindow, candidateEdits)
		if !previewSpritesVerified || !previewLocalisationVerified {
			diagnostics := append(previewSpriteDiagnostics, previewLocalisationDiagnostics...)
			return nil, nil, fmt.Errorf("authoritative GUI overlay preview has unresolved references: %v", diagnostics)
		}
		source := state.Source
		if !state.Exists || strings.TrimSpace(source) == "" {
			source = stringField(window, "source")
			if source == "mod" {
				source = "import"
			}
		}
		mutationFile := guiOverlayMutationFile(state, normalized)
		body := map[string]any{
			"file": mutationFile, "edits": merged, "scaffold_hash": candidateScaffoldHash,
			"expected_revision": state.Revision, "expected_scaffold_hash": state.ScaffoldHash,
			"source": source, "schema_version": maxInt(state.SchemaVersion, 1),
		}
		saved, err := proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/gui-render/overlay", in.ProjectID), body)
		if err != nil {
			return nil, nil, err
		}
		out := map[string]any{
			"ok": true, "project_id": in.ProjectID, "file": mutationFile,
			"normalized_file": normalized, "window_name": in.WindowName,
			"requested": len(canonical) + len(in.RemoveEditIDs), "applied": len(canonical) + len(in.RemoveEditIDs), "failed": 0,
			"persisted": true, "persistence_verified": false,
			"authoritative_validation": false, "readback_verified": false,
			"sprite_references_verified": false, "localisation_verified": false,
			"references_verified": false, "completed": false,
			"scaffold_hash":    candidateScaffoldHash,
			"removed_edit_ids": in.RemoveEditIDs,
		}
		if id := int64Field(saved, "id"); id > 0 {
			out["overlay_id"] = id
		}

		reread, readErr := readGUIOverlay(ctx, proxy, in.ProjectID, mutationFile)
		if readErr != nil {
			out["diagnostics"] = []map[string]any{{"severity": "error", "code": "overlay_readback_unavailable", "message": "The overlay was saved but could not be read back."}}
			return nil, out, nil
		}
		persistenceVerified := equalJSON(reread.Edits, merged) && reread.ScaffoldHash == candidateScaffoldHash
		out["persistence_verified"] = persistenceVerified
		out["revision"] = reread.Revision
		out["saved_edits"] = reread.Edits

		diagnostics := []map[string]any{}
		authoritative, validationErr := validateSavedGUIOverlay(ctx, proxy, in.ProjectID, reread, in.WindowName)
		authoritativeVerified := false
		var authoritativeRoot map[string]any
		if validationErr != nil {
			diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "authoritative_validation_unavailable", validationErr.Error(), "", ""))
		} else {
			authoritativeVerified = boolField(authoritative, "valid") && boolField(authoritative, "saved") && boolField(authoritative, "materializable")
			if backendDiagnostics, decodeErr := guiOverlayMapSliceField(authoritative, "diagnostics"); decodeErr == nil {
				diagnostics = append(diagnostics, backendDiagnostics...)
			}
			if authoritativeWindow := nodeField(authoritative, "window"); authoritativeWindow != nil {
				authoritativeRoot = nodeField(authoritativeWindow, "node")
				materialized, materializedDiagnostics := verifyGUIEditsReadback(authoritativeRoot, candidateEdits)
				authoritativeVerified = authoritativeVerified && materialized
				diagnostics = append(diagnostics, materializedDiagnostics...)
			} else {
				authoritativeVerified = false
				diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "authoritative_window_missing", "The saved overlay validation returned no rendered window.", "", ""))
			}
		}
		out["authoritative_validation"] = authoritativeVerified

		visible, _, visibleErr := readGUIWindow(ctx, proxy, in.ProjectID, in.WindowName, mutationFile)
		readbackVerified := false
		spriteReferencesVerified := false
		localisationVerified := false
		if visibleErr != nil {
			diagnostics = append(diagnostics, map[string]any{"severity": "error", "code": "window_readback_unavailable", "message": "The overlay was saved but the effective window could not be read back."})
		} else {
			var readbackDiagnostics []map[string]any
			readbackVerified, readbackDiagnostics = verifyGUIEditsReadback(nodeField(visible, "node"), candidateEdits)
			diagnostics = append(diagnostics, readbackDiagnostics...)
			var spriteDiagnostics []map[string]any
			spriteReferencesVerified, spriteDiagnostics = validateGUIOverlaySpriteReferences(visible, candidateEdits)
			diagnostics = append(diagnostics, spriteDiagnostics...)
			var localisationDiagnostics []map[string]any
			localisationVerified, localisationDiagnostics = validateGUIOverlayLocalisationReferences(visible, candidateEdits)
			diagnostics = append(diagnostics, localisationDiagnostics...)
			readbackVerified = readbackVerified && spriteReferencesVerified && localisationVerified
			if authoritativeRoot != nil && !equalJSON(nodeField(visible, "node"), authoritativeRoot) {
				readbackVerified = false
				diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "effective_window_mismatch", "The effective window differs from the authoritative saved-overlay render.", "", ""))
			}
		}
		out["readback_verified"] = readbackVerified
		out["sprite_references_verified"] = spriteReferencesVerified
		out["localisation_verified"] = localisationVerified
		out["references_verified"] = spriteReferencesVerified && localisationVerified
		out["completed"] = persistenceVerified && authoritativeVerified && readbackVerified && spriteReferencesVerified && localisationVerified
		out["diagnostics"] = diagnostics
		return nil, out, nil
	})
}

func registerValidateGUIOverlay(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "validate_gui_overlay",
		Description: "Validate the persisted overlay against its exact scaffold (imported mod or base game), then independently prove every edit against the effective window as the game view renders it. " +
			"valid/readback_verified=false means scaffold drift, a shadowing project file, or an invalid edit; do not claim the GUI task is complete.",
	}, func(ctx context.Context, _ *CallToolRequest, in guiOverlayValidateIn) (*CallToolResult, map[string]any, error) {
		if err := validateGUIProjectContext(ctx, in.ProjectID); err != nil {
			return nil, nil, err
		}
		windowName, err := validateGUIWindowName(in.WindowName)
		if err != nil {
			return nil, nil, err
		}
		state, err := readGUIOverlay(ctx, proxy, in.ProjectID, in.File)
		if err != nil {
			return nil, nil, err
		}
		if strings.TrimSpace(in.ExpectedRevision) != "" && in.ExpectedRevision != state.Revision {
			return nil, nil, fmt.Errorf("stale GUI overlay revision: expected %s, current %s", in.ExpectedRevision, state.Revision)
		}
		diagnostics := []map[string]any{}
		canonical := make([]guiOverlayEditInput, 0, len(state.Edits))
		for index, raw := range state.Edits {
			edit, parseErr := decodeStoredGUIEdit(raw)
			if parseErr == nil {
				parseErr = validateGUIEditSchema(&edit, guiEditWindowOr(edit.Target, windowName))
			}
			if parseErr != nil {
				diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "invalid_edit", fmt.Sprintf("edit %d: %v", index, parseErr), edit.Target, edit.Op))
				continue
			}
			canonical = append(canonical, edit)
		}

		windowEdits := guiEditsInWindow(canonical, windowName)

		authoritative, authoritativeErr := validateSavedGUIOverlay(ctx, proxy, in.ProjectID, state, windowName)
		authoritativeVerified := false
		materializable := false
		var authoritativeRoot map[string]any
		if authoritativeErr != nil {
			diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "authoritative_validation_unavailable", authoritativeErr.Error(), "", ""))
		} else {
			materializable = boolField(authoritative, "materializable")
			authoritativeVerified = boolField(authoritative, "valid") && boolField(authoritative, "saved") && materializable
			if backendDiagnostics, decodeErr := guiOverlayMapSliceField(authoritative, "diagnostics"); decodeErr == nil {
				diagnostics = append(diagnostics, backendDiagnostics...)
			}
			if authoritativeWindow := nodeField(authoritative, "window"); authoritativeWindow != nil {
				authoritativeRoot = nodeField(authoritativeWindow, "node")
				materialized, materializedDiagnostics := verifyGUIEditsReadback(authoritativeRoot, windowEdits)
				authoritativeVerified = authoritativeVerified && materialized
				diagnostics = append(diagnostics, materializedDiagnostics...)
			} else {
				authoritativeVerified = false
				diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "authoritative_window_missing", "The overlay validation returned no rendered window.", "", ""))
			}
		}

		window, normalized, windowErr := readGUIWindow(ctx, proxy, in.ProjectID, windowName, guiOverlayMutationFile(state, state.Normalized))
		readbackVerified := false
		spriteReferencesVerified := false
		localisationVerified := false
		if windowErr != nil {
			diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "window_readback_unavailable", windowErr.Error(), "", ""))
		} else {
			var readback []map[string]any
			readbackVerified, readback = verifyGUIEditsReadback(nodeField(window, "node"), windowEdits)
			diagnostics = append(diagnostics, readback...)
			var spriteDiagnostics []map[string]any
			spriteReferencesVerified, spriteDiagnostics = validateGUIOverlaySpriteReferences(window, windowEdits)
			diagnostics = append(diagnostics, spriteDiagnostics...)
			var localisationDiagnostics []map[string]any
			localisationVerified, localisationDiagnostics = validateGUIOverlayLocalisationReferences(window, windowEdits)
			diagnostics = append(diagnostics, localisationDiagnostics...)
			readbackVerified = readbackVerified && spriteReferencesVerified && localisationVerified
			if authoritativeRoot != nil && !equalJSON(nodeField(window, "node"), authoritativeRoot) {
				readbackVerified = false
				diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "effective_window_mismatch", "The effective window differs from the authoritative saved-overlay render.", "", ""))
			}
		}
		valid := len(canonical) == len(state.Edits) && authoritativeVerified && readbackVerified && spriteReferencesVerified && localisationVerified
		out := guiOverlayStateResult(state, in.ProjectID)
		out["window_name"] = windowName
		out["normalized_file"] = normalized
		out["materializable"] = materializable
		out["authoritative_validation"] = authoritativeVerified
		out["valid"] = valid
		out["readback_verified"] = readbackVerified
		out["sprite_references_verified"] = spriteReferencesVerified
		out["localisation_verified"] = localisationVerified
		out["references_verified"] = spriteReferencesVerified && localisationVerified
		out["completed"] = valid
		out["diagnostics"] = diagnostics
		return nil, out, nil
	})
}

func validateGUIProjectContext(_ context.Context, projectID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("project_id must be greater than 0")
	}
	return nil
}

func validateGUIWindowName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || len(name) > 512 || strings.ContainsAny(name, "/\\\r\n\x00") {
		return "", fmt.Errorf("window_name must be one exact GUI window name")
	}
	return name, nil
}

func normalizeGUIOverlayFile(raw string) (string, error) {
	normalized, err := exportengine.NormalizeGUISourcePath(raw)
	if err != nil {
		return "", err
	}
	return normalized, nil
}

func readGUIWindow(ctx context.Context, proxy *RESTProxy, projectID int64, rawName, rawFile string) (map[string]any, string, error) {
	name, err := validateGUIWindowName(rawName)
	if err != nil {
		return nil, "", err
	}
	query := url.Values{"name": {name}}
	requested := ""
	if strings.TrimSpace(rawFile) != "" {
		requested, err = normalizeGUIOverlayFile(rawFile)
		if err != nil {
			return nil, "", err
		}
		query.Set("file", requested)
	}
	window, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/gui-render/window?%s", projectID, query.Encode()), nil)
	if err != nil {
		return nil, "", err
	}
	if stringField(window, "name") != name {
		return nil, "", fmt.Errorf("project service returned a different GUI window")
	}
	actual, err := normalizeGUIOverlayFile(stringField(window, "file"))
	if err != nil {
		return nil, "", fmt.Errorf("GUI window has no safe source file")
	}
	if requested != "" && !strings.EqualFold(requested, actual) {
		return nil, "", fmt.Errorf("window %s belongs to %s, not %s", name, actual, requested)
	}
	if nodeField(window, "node") == nil {
		return nil, "", fmt.Errorf("GUI window has no node tree")
	}
	return window, actual, nil
}

func readGUIOverlay(ctx context.Context, proxy *RESTProxy, projectID int64, rawFile string) (guiOverlayState, error) {
	normalized, err := normalizeGUIOverlayFile(rawFile)
	if err != nil {
		return guiOverlayState{}, err
	}
	candidates := []string{normalized, strings.TrimPrefix(normalized, "interface/")}
	exact := strings.Trim(strings.ReplaceAll(rawFile, "\\", "/"), "/")
	if exact != "" {
		candidates = append(candidates, exact)
	}
	seen := map[string]bool{}
	var absent map[string]any
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		query := url.Values{"file": {candidate}}
		resp, callErr := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/gui-render/overlay?%s", projectID, query.Encode()), nil)
		if callErr != nil {
			return guiOverlayState{}, callErr
		}
		if boolField(resp, "exists") {
			return decodeGUIOverlayState(resp, normalized, true)
		}
		if absent == nil || candidate == normalized {
			absent = resp
		}
	}
	if absent == nil {
		absent = map[string]any{"file": normalized, "source": "vanilla", "schema_version": 1, "edits": []any{}, "exists": false}
	}
	absent["file"] = normalized
	return decodeGUIOverlayState(absent, normalized, false)
}

func decodeGUIOverlayState(resp map[string]any, normalized string, exists bool) (guiOverlayState, error) {
	edits, err := guiOverlayMapSliceField(resp, "edits")
	if err != nil {
		return guiOverlayState{}, err
	}
	file := strings.TrimSpace(stringField(resp, "file"))
	if file == "" || !exists {
		file = normalized
	}
	if _, err := normalizeGUIOverlayFile(file); err != nil {
		return guiOverlayState{}, fmt.Errorf("project service returned an unsafe GUI overlay file")
	}
	state := guiOverlayState{
		Payload: cloneMap(resp), File: file, Normalized: normalized,
		Source: stringField(resp, "source"), ScaffoldHash: stringField(resp, "scaffold_hash"),
		CurrentScaffoldHash: stringField(resp, "current_scaffold_hash"),
		ScaffoldHashPinned:  boolField(resp, "scaffold_hash_pinned"), ScaffoldDrift: boolField(resp, "scaffold_drift"),
		SchemaVersion: int(int64Field(resp, "schema_version")), Exists: exists,
		Materializable: boolField(resp, "materializable"), Edits: edits,
	}
	state.Edits = normalizeLegacyGUIOverlayEdits(state.Edits, normalized)
	if state.Source == "" {
		state.Source = "vanilla"
	}
	if state.SchemaVersion <= 0 {
		state.SchemaVersion = 1
	}
	if state.CurrentScaffoldHash == "" {
		state.CurrentScaffoldHash = state.ScaffoldHash
	}
	if _, supplied := resp["scaffold_hash_pinned"]; !supplied {
		state.ScaffoldHashPinned = state.Exists && state.ScaffoldHash != ""
	}
	state.Revision = strings.TrimSpace(stringField(resp, "revision"))
	if state.Revision == "" {

		state.Revision = revisionForGUIOverlay(state)
	}
	return state, nil
}

func revisionForGUIOverlay(state guiOverlayState) string {
	payload := map[string]any{
		"exists": state.Exists, "file": state.File, "source": state.Source,
		"scaffold_hash": state.ScaffoldHash, "schema_version": state.SchemaVersion,
		"edits": state.Edits,
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func guiOverlayStateResult(state guiOverlayState, projectID int64) map[string]any {
	return map[string]any{
		"ok": true, "project_id": projectID, "file": state.File,
		"normalized_file": state.Normalized, "source": state.Source,
		"scaffold_hash": state.ScaffoldHash, "current_scaffold_hash": state.CurrentScaffoldHash,
		"scaffold_hash_pinned": state.ScaffoldHashPinned, "scaffold_drift": state.ScaffoldDrift,
		"materializable": state.Materializable,
		"schema_version": state.SchemaVersion, "edits": state.Edits,
		"exists": state.Exists, "revision": state.Revision,
	}
}

func guiOverlayMutationFile(state guiOverlayState, normalized string) string {
	if state.Exists && strings.TrimSpace(state.File) != "" {
		return state.File
	}
	return normalized
}

func previewGUIOverlayCandidate(
	ctx context.Context,
	proxy *RESTProxy,
	projectID int64,
	state guiOverlayState,
	windowName string,
	edits []map[string]any,
) (map[string]any, error) {
	body := map[string]any{
		"file":                   guiOverlayMutationFile(state, state.Normalized),
		"window_name":            windowName,
		"expected_revision":      state.Revision,
		"expected_scaffold_hash": state.ScaffoldHash,
		"edits":                  edits,
	}
	return proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/gui-render/overlay/preview", projectID), body)
}

func validateSavedGUIOverlay(
	ctx context.Context,
	proxy *RESTProxy,
	projectID int64,
	state guiOverlayState,
	windowName string,
) (map[string]any, error) {
	body := map[string]any{
		"file":              guiOverlayMutationFile(state, state.Normalized),
		"window_name":       windowName,
		"expected_revision": state.Revision,
	}
	return proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/gui-render/overlay/validate", projectID), body)
}

func prepareGUIOverlayBatch(ctx context.Context, proxy *RESTProxy, in guiOverlayBatchIn) (guiOverlayState, map[string]any, string, []guiOverlayEditInput, []map[string]any, error) {
	if err := validateGUIProjectContext(ctx, in.ProjectID); err != nil {
		return guiOverlayState{}, nil, "", nil, nil, err
	}
	windowName, err := validateGUIWindowName(in.WindowName)
	if err != nil {
		return guiOverlayState{}, nil, "", nil, nil, err
	}
	operationCount := len(in.Edits) + len(in.RemoveEditIDs)
	if operationCount == 0 || operationCount > guiOverlayBatchLimit {
		return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("edits plus remove_edit_ids must contain between 1 and %d operations", guiOverlayBatchLimit)
	}
	if strings.TrimSpace(in.ExpectedRevision) == "" {
		return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("expected_revision is required; call get_gui_overlay first")
	}
	state, err := readGUIOverlay(ctx, proxy, in.ProjectID, in.File)
	if err != nil {
		return guiOverlayState{}, nil, "", nil, nil, err
	}
	if in.ExpectedRevision != state.Revision {
		return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("stale GUI overlay revision: expected %s, current %s", in.ExpectedRevision, state.Revision)
	}
	if in.ExpectedScaffoldHash != state.ScaffoldHash {
		return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("stale GUI scaffold hash: expected %q, current %q", in.ExpectedScaffoldHash, state.ScaffoldHash)
	}
	remaining, removed, err := removeGUIOverlayEdits(state.Edits, in.RemoveEditIDs, windowName)
	if err != nil {
		return guiOverlayState{}, nil, "", nil, nil, err
	}
	state.Edits = remaining
	window, normalized, err := readGUIWindow(ctx, proxy, in.ProjectID, windowName, state.Normalized)
	if err != nil {
		return guiOverlayState{}, nil, "", nil, nil, err
	}
	windowRevision := strings.TrimSpace(stringField(window, "revision"))
	windowScaffold := strings.TrimSpace(stringField(window, "scaffold_hash"))
	if !state.Materializable || state.ScaffoldDrift || windowRevision != state.Revision || windowScaffold != state.ScaffoldHash {
		return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("effective GUI window is not safely editable: it is shadowed, drifted, or missing the current overlay envelope")
	}
	root := cloneMap(nodeField(window, "node"))
	canonical := make([]guiOverlayEditInput, 0, len(in.Edits))
	changes := make([]map[string]any, 0, operationCount)
	for _, removedEdit := range removed {
		changes = append(changes, map[string]any{
			"id": storedGUIEditKey(removedEdit), "op": "remove_edit", "before": removedEdit, "after": nil,
		})
	}
	seen := map[string]bool{}
	removedSet := map[string]bool{}
	existingByID := make(map[string]map[string]any, len(state.Edits))
	for _, stored := range state.Edits {
		if id := storedGUIEditKey(stored); id != "" {
			existingByID[id] = stored
		}
	}
	for _, id := range in.RemoveEditIDs {
		removedSet[id] = true
	}
	for index := range in.Edits {
		edit := in.Edits[index]
		if err := validateGUIEditSchema(&edit, windowName); err != nil {
			return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("edit %d: %w", index, err)
		}
		if seen[edit.ID] {
			return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("edit %d duplicates %s", index, edit.ID)
		}
		if removedSet[edit.ID] {
			return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("edit %d both removes and upserts %s", index, edit.ID)
		}
		seen[edit.ID] = true

		if edit.Op == "add_node" {
			if stored, exists := existingByID[edit.ID]; exists {
				oldEdit, decodeErr := decodeStoredGUIEdit(stored)
				if decodeErr != nil {
					return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("stored edit %s is invalid: %w", edit.ID, decodeErr)
				}
				oldNode, _ := oldEdit.Payload["node"].(map[string]any)
				oldName := strings.TrimSpace(stringField(oldNode, "name"))
				oldRef := findGUINode(root, strings.Split(edit.Target+"/"+oldName, "/"))
				if oldName == "" || oldRef == nil || oldRef.Parent == nil {
					return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("stored add_node %s is not materialized in the effective window", edit.ID)
				}
				children := guiNodeChildren(oldRef.Parent)
				children = append(children[:oldRef.Index], children[oldRef.Index+1:]...)
				setGUINodeChildren(oldRef.Parent, children)
			}
		}
		change, err := projectGUIEdit(root, edit)
		if err != nil {
			return guiOverlayState{}, nil, "", nil, nil, fmt.Errorf("edit %d: %w", index, err)
		}
		canonical = append(canonical, edit)
		changes = append(changes, change)
	}
	return state, window, normalized, canonical, changes, nil
}

func validateGUIEditSchema(edit *guiOverlayEditInput, windowName string) error {
	edit.Op = strings.TrimSpace(strings.ToLower(edit.Op))
	if !guiOverlayOps[edit.Op] {
		return fmt.Errorf("unsupported GUI overlay op %q", edit.Op)
	}
	target, err := validateGUINamePath(edit.Target, windowName)
	if err != nil {
		return err
	}
	edit.Target = target
	if edit.Payload == nil {
		edit.Payload = map[string]any{}
	}
	canonicalID, err := canonicalGUIEditID(*edit)
	if err != nil {
		return err
	}
	if supplied := strings.TrimSpace(edit.ID); supplied != "" && supplied != canonicalID {
		return fmt.Errorf("id must be %q", canonicalID)
	}
	edit.ID = canonicalID

	switch edit.Op {
	case "set_text":
		if err := rejectUnknownGUIPayloadFields(edit.Payload, "text", "lang_texts", "key", "is_new"); err != nil {
			return err
		}
		text, hasText := edit.Payload["text"]
		langs, hasLangs := edit.Payload["lang_texts"]
		key, hasKey := edit.Payload["key"]
		if !hasText && !hasLangs && !hasKey {
			return fmt.Errorf("set_text payload requires text, lang_texts, or key")
		}
		if hasText {
			if value, ok := text.(string); !ok || len(value) > 100_000 || strings.ContainsRune(value, '\x00') {
				return fmt.Errorf("set_text text must be a string of at most 100000 bytes")
			}
		}
		if hasLangs {
			if err := validateGUILanguageTexts(langs); err != nil {
				return err
			}
		}
		if hasLangs && !hasKey {
			return fmt.Errorf("set_text lang_texts requires key so every language is exported")
		}
		if hasKey {

			value, ok := key.(string)
			if !ok || len(value) > 512 || !guiLocalisationKeyPattern.MatchString(value) {
				return fmt.Errorf("set_text key must be a localisation key of letters, digits, _ . - (at most 512 bytes)")
			}
		}
		if isNew, present := edit.Payload["is_new"]; present {
			if _, ok := isNew.(bool); !ok {
				return fmt.Errorf("set_text is_new must be a boolean")
			}
		}
		if hasKey && !hasText && !hasLangs {
			if isNew, supplied := edit.Payload["is_new"].(bool); !supplied || isNew {
				return fmt.Errorf("set_text with a new key requires text or lang_texts")
			}
		}
	case "swap_sprite":
		if root, _, _ := strings.Cut(target, "/"); guiEngineSwappedWindows[root] {
			return fmt.Errorf("swap_sprite cannot point %s at another sprite: the game swaps that picture itself and stops drawing (the GPU stops on the main menu). Remove such an edit with remove_edit_ids and replace the background picture itself instead (the picture get_gui_window shows there)", root)
		}
		if err := rejectUnknownGUIPayloadFields(edit.Payload, "gfx_ref", "override_file"); err != nil {
			return err
		}
		ref, _ := edit.Payload["gfx_ref"].(string)
		if strings.TrimSpace(ref) == "" || len(ref) > 512 || strings.ContainsAny(ref, "\r\n\x00") {
			return fmt.Errorf("swap_sprite payload requires a valid gfx_ref")
		}

		if raw, present := edit.Payload["override_file"]; present {
			if override, ok := raw.(string); !ok || override != "" {
				return fmt.Errorf("swap_sprite override_file is not supported by the Agent overlay contract; apply artwork to the verified GFX sprite first, then use gfx_ref only")
			}
		}
	case "set_layout":
		if err := rejectUnknownGUIPayloadFields(edit.Payload, "x", "y", "width", "height"); err != nil {
			return err
		}
		found := false
		for _, key := range []string{"x", "y", "width", "height"} {
			if value, ok := edit.Payload[key]; ok {
				found = true
				if !validGUILayoutValue(value) {
					return fmt.Errorf("set_layout %s must be a finite number or short GUI token", key)
				}
			}
		}
		if !found {
			return fmt.Errorf("set_layout requires at least one of x, y, width or height")
		}
	case "set_orientation", "set_origo":
		if err := rejectUnknownGUIPayloadFields(edit.Payload, "value"); err != nil {
			return err
		}
		value, ok := edit.Payload["value"].(string)
		value = strings.ToUpper(strings.TrimSpace(value))
		if !ok || (value == "" && edit.Op != "set_origo") || (value != "" && !guiOverlayAnchors[value]) {
			return fmt.Errorf("%s value must be a canonical 3x3 anchor%s", edit.Op, map[bool]string{true: " or empty to clear", false: ""}[edit.Op == "set_origo"])
		}
		edit.Payload["value"] = value
	case "add_node":
		if err := rejectUnknownGUIPayloadFields(edit.Payload, "node", "index"); err != nil {
			return err
		}
		node, ok := edit.Payload["node"].(map[string]any)
		if !ok {
			return fmt.Errorf("add_node payload requires node")
		}
		count := 0
		if err := validateGUIAddedNode(node, 0, &count); err != nil {
			return err
		}
		if index, exists := edit.Payload["index"]; exists {
			value, ok := wholeNumber(index)
			if !ok || value < 0 || value > guiOverlayNodeLimit {
				return fmt.Errorf("add_node index must be a non-negative integer")
			}
			edit.Payload["index"] = value
		}
	case "delete_node", "hide_node":
		if err := rejectUnknownGUIPayloadFields(edit.Payload); err != nil {
			return err
		}
		if target == windowName {
			return fmt.Errorf("%s cannot target the root window", edit.Op)
		}
	}
	encoded, _ := json.Marshal(edit.Payload)
	if len(encoded) > 1<<20 {
		return fmt.Errorf("edit payload exceeds 1 MiB")
	}
	return nil
}

func rejectUnknownGUIPayloadFields(payload map[string]any, allowed ...string) error {
	known := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		known[key] = true
	}
	for key := range payload {
		if !known[key] {
			return fmt.Errorf("payload field %q is not supported by this GUI overlay operation", key)
		}
	}
	return nil
}

func validateGUINamePath(raw, windowName string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" || len(target) > 2048 || strings.ContainsAny(target, "\\\r\n\x00") {
		return "", fmt.Errorf("target must be a safe NAME-PATH")
	}
	parts := strings.Split(target, "/")
	if len(parts) == 0 || parts[0] != windowName {
		return "", fmt.Errorf("target must begin with window_name %q", windowName)
	}
	for _, part := range parts {
		if strings.TrimSpace(part) != part || part == "" || part == "." || part == ".." || len(part) > 512 {
			return "", fmt.Errorf("target contains an invalid NAME-PATH segment")
		}
	}
	return target, nil
}

func canonicalGUIEditID(edit guiOverlayEditInput) (string, error) {
	if edit.Op != "add_node" {
		return edit.Op + ":" + edit.Target, nil
	}
	node, _ := edit.Payload["node"].(map[string]any)
	name, _ := node["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("add_node node.name is required so later edits have a stable NAME-PATH")
	}
	if strings.ContainsAny(name, "/\\\r\n\x00") || name == "." || name == ".." || len(name) > 512 {
		return "", fmt.Errorf("add_node node.name is not a safe NAME-PATH segment")
	}
	return "add_node:" + edit.Target + "/" + name, nil
}

func validateGUILanguageTexts(value any) error {
	texts, ok := value.(map[string]any)
	if !ok || len(texts) == 0 || len(texts) > 64 {
		return fmt.Errorf("lang_texts must be a non-empty language-to-string object")
	}
	for lang, raw := range texts {
		text, ok := raw.(string)
		if !ok || strings.TrimSpace(lang) == "" || len(lang) > 32 || len(text) > 100_000 || strings.ContainsRune(text, '\x00') {
			return fmt.Errorf("lang_texts contains an invalid language or value")
		}
	}
	return nil
}

func validGUILayoutValue(value any) bool {
	switch typed := value.(type) {
	case float64:
		return !math.IsNaN(typed) && !math.IsInf(typed, 0) && math.Abs(typed) <= 10_000_000
	case float32:
		return !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0) && math.Abs(float64(typed)) <= 10_000_000
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		return strings.TrimSpace(typed) == typed && typed != "" && len(typed) <= 64 && !strings.ContainsAny(typed, "\r\n\x00{}")
	default:
		return false
	}
}

func validateGUIAddedNode(node map[string]any, depth int, count *int) error {

	if depth >= guiOverlayDepthLimit {
		return fmt.Errorf("add_node subtree exceeds depth %d", guiOverlayDepthLimit)
	}
	*count++
	if *count > guiOverlayNodeLimit {
		return fmt.Errorf("add_node subtree exceeds %d nodes", guiOverlayNodeLimit)
	}
	allowed := map[string]bool{
		"type": true, "name": true, "text": true, "sprite": true,
		"orientation": true, "origo": true, "position": true,
		"size": true, "children": true,
	}
	for key := range node {
		if !allowed[key] {
			return fmt.Errorf("add_node field %q is not materialized by the GUI overlay exporter; use a structured UI panel for bindings, actions, tooltips, fonts, frames, formats, or raw GUI properties", key)
		}
	}
	typeName, _ := node["type"].(string)
	if !guiOverlayNodeTypes[strings.ToLower(strings.TrimSpace(typeName))] {
		return fmt.Errorf("add_node has unsupported node.type %q", typeName)
	}

	name, _ := node["name"].(string)
	if strings.TrimSpace(name) == "" || strings.TrimSpace(name) != name || strings.ContainsAny(name, "/\\\"\r\n\x00") || len(name) > 512 {
		return fmt.Errorf("every added node requires a safe non-empty name")
	}
	for _, key := range []string{"text", "sprite", "orientation", "origo"} {
		if raw, present := node[key]; present {
			value, ok := raw.(string)
			if !ok || len(value) > 100_000 || strings.ContainsAny(value, "\"\r\n\x00") {
				return fmt.Errorf("added node %s must be a single-line string without double quotes", key)
			}
		}
	}
	for _, key := range []string{"position", "size"} {
		if raw, present := node[key]; present && raw != nil {
			point, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("added node %s must be an object", key)
			}
			allowedCoordinates := map[string]bool{"x": true, "y": true}
			if key == "size" {
				allowedCoordinates = map[string]bool{"width": true, "height": true}
			}
			for coordinate, value := range point {
				if !allowedCoordinates[coordinate] {
					return fmt.Errorf("added node %s contains unsupported field %q", key, coordinate)
				}
				if !validGUILayoutValue(value) {
					return fmt.Errorf("added node %s has an invalid coordinate", key)
				}
			}
		}
	}
	children := []map[string]any{}
	if raw, present := node["children"]; present && raw != nil {
		switch typed := raw.(type) {
		case []map[string]any:
			children = typed
		case []any:
			for _, rawChild := range typed {
				child, ok := rawChild.(map[string]any)
				if !ok {
					return fmt.Errorf("added node children must contain only objects")
				}
				children = append(children, child)
			}
		default:
			return fmt.Errorf("added node children must be an array")
		}
	}
	seen := map[string]bool{}
	for _, child := range children {
		childName, _ := child["name"].(string)
		if seen[childName] {
			return fmt.Errorf("added node has duplicate child name %q", childName)
		}
		seen[childName] = true
		if err := validateGUIAddedNode(child, depth+1, count); err != nil {
			return err
		}
	}
	return nil
}

func projectGUIEdit(root map[string]any, edit guiOverlayEditInput) (map[string]any, error) {
	ref := findGUINode(root, strings.Split(edit.Target, "/"))
	if ref == nil {
		return nil, fmt.Errorf("NAME-PATH target not found: %s", edit.Target)
	}
	before := guiNodeSummary(ref.Node)
	change := map[string]any{"id": edit.ID, "op": edit.Op, "target": edit.Target, "before": before}
	switch edit.Op {
	case "add_node":
		node := cloneMap(edit.Payload["node"].(map[string]any))
		name := stringField(node, "name")
		childPath := edit.Target + "/" + name
		if findGUINode(root, strings.Split(childPath, "/")) != nil {
			return nil, fmt.Errorf("add_node would duplicate NAME-PATH %s", childPath)
		}
		children := guiNodeChildren(ref.Node)
		index := len(children)
		if raw, ok := edit.Payload["index"]; ok {
			index, _ = wholeNumber(raw)
			if index > len(children) {
				index = len(children)
			}
		}
		children = append(children, nil)
		copy(children[index+1:], children[index:])
		children[index] = node
		setGUINodeChildren(ref.Node, children)
		change["created_path"] = childPath
		change["after"] = guiNodeSummary(node)
		return change, nil
	case "delete_node":
		if ref.Parent == nil {
			return nil, fmt.Errorf("delete_node cannot delete the root window")
		}
		children := guiNodeChildren(ref.Parent)
		children = append(children[:ref.Index], children[ref.Index+1:]...)
		setGUINodeChildren(ref.Parent, children)
		change["after"] = nil
		return change, nil
	case "set_text":
		ref.Node["text"] = projectedGUIText(edit.Payload)
	case "swap_sprite":
		ref.Node["sprite"] = stringField(edit.Payload, "gfx_ref")
	case "set_layout":
		position := mapFieldCopy(ref.Node, "position")
		size := mapFieldCopy(ref.Node, "size")
		for _, key := range []string{"x", "y"} {
			if value, ok := edit.Payload[key]; ok {
				position[key] = value
			}
		}
		for _, key := range []string{"width", "height"} {
			if value, ok := edit.Payload[key]; ok {
				size[key] = value
			}
		}
		ref.Node["position"], ref.Node["size"] = position, size
	case "set_orientation":
		ref.Node["orientation"] = stringField(edit.Payload, "value")
	case "set_origo":
		value := stringField(edit.Payload, "value")
		if value == "" {
			ref.Node["origo"] = nil
		} else {
			ref.Node["origo"] = value
		}
	}
	change["after"] = guiNodeSummary(ref.Node)
	return change, nil
}

type guiNodeReference struct {
	Node   map[string]any
	Parent map[string]any
	Index  int
}

func findGUINode(root map[string]any, parts []string) *guiNodeReference {
	if root == nil || len(parts) == 0 || stringField(root, "name") != parts[0] {
		return nil
	}
	current := root
	var parent map[string]any
	index := -1
	for _, part := range parts[1:] {
		children := guiNodeChildren(current)
		found := -1
		for i, child := range children {
			if stringField(child, "name") == part {
				if found >= 0 {

					return nil
				}
				found = i
			}
		}
		if found < 0 {
			return nil
		}
		parent, index, current = current, found, children[found]
	}
	return &guiNodeReference{Node: current, Parent: parent, Index: index}
}

func guiNodeChildren(node map[string]any) []map[string]any {
	if node == nil {
		return nil
	}
	switch typed := node["children"].(type) {
	case []map[string]any:
		return append([]map[string]any(nil), typed...)
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, raw := range typed {
			if child, ok := raw.(map[string]any); ok {
				out = append(out, child)
			}
		}
		return out
	default:
		return []map[string]any{}
	}
}

func setGUINodeChildren(node map[string]any, children []map[string]any) {
	out := make([]any, 0, len(children))
	for _, child := range children {
		out = append(out, child)
	}
	node["children"] = out
}

func guiNodeSummary(node map[string]any) map[string]any {
	if node == nil {
		return nil
	}
	out := map[string]any{}
	for _, key := range []string{"type", "name", "position", "size", "orientation", "origo", "text", "sprite"} {
		if value, ok := node[key]; ok {
			out[key] = value
		}
	}
	out["children"] = len(guiNodeChildren(node))
	return out
}

func projectedGUIText(payload map[string]any) string {
	if key, ok := payload["key"].(string); ok && strings.TrimSpace(key) != "" {
		key = strings.TrimSpace(key)
		if isNew, supplied := payload["is_new"].(bool); !supplied || isNew {
			return "GUISUB_" + key
		}
		return key
	}
	if text, ok := payload["text"].(string); ok {
		return text
	}
	texts, _ := payload["lang_texts"].(map[string]any)
	if english, ok := texts["en"].(string); ok {
		return english
	}
	keys := make([]string, 0, len(texts))
	for key := range texts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if text, ok := texts[key].(string); ok {
			return text
		}
	}
	return ""
}

func mergeGUIOverlayEdits(existing []map[string]any, incoming []guiOverlayEditInput) []map[string]any {
	out := make([]map[string]any, len(existing))
	positions := map[string]int{}
	for index, edit := range existing {
		out[index] = cloneMap(edit)
		if key := storedGUIEditKey(edit); key != "" {
			positions[key] = index
		}
	}
	for _, edit := range incoming {
		row := map[string]any{"id": edit.ID, "target": edit.Target, "op": edit.Op, "payload": cloneMap(edit.Payload)}
		if index, found := positions[edit.ID]; found {
			out[index] = row
		} else {
			positions[edit.ID] = len(out)
			out = append(out, row)
		}
	}
	return out
}

func removeGUIOverlayEdits(existing []map[string]any, requested []string, windowName string) ([]map[string]any, []map[string]any, error) {
	if len(requested) == 0 {
		return existing, nil, nil
	}
	wanted := make(map[string]bool, len(requested))
	for index, rawID := range requested {
		id := strings.TrimSpace(rawID)
		separator := strings.IndexByte(id, ':')
		if id == "" || id != rawID || len(id) > 4096 || strings.ContainsAny(id, "\r\n\x00") || separator <= 0 || !guiOverlayOps[id[:separator]] {
			return nil, nil, fmt.Errorf("remove_edit_ids[%d] is not a canonical GUI overlay edit id", index)
		}
		if _, err := validateGUINamePath(id[separator+1:], windowName); err != nil {
			return nil, nil, fmt.Errorf("remove_edit_ids[%d]: %w", index, err)
		}
		if wanted[id] {
			return nil, nil, fmt.Errorf("remove_edit_ids contains duplicate %q", id)
		}
		wanted[id] = true
	}

	remaining := make([]map[string]any, 0, len(existing))
	removed := make([]map[string]any, 0, len(requested))
	found := map[string]bool{}
	for _, edit := range existing {
		id := storedGUIEditKey(edit)
		if wanted[id] {
			if found[id] {
				return nil, nil, fmt.Errorf("stored GUI overlay contains duplicate canonical id %q", id)
			}
			found[id] = true
			removed = append(removed, cloneMap(edit))
			continue
		}
		remaining = append(remaining, cloneMap(edit))
	}
	for _, id := range requested {
		if !found[id] {
			return nil, nil, fmt.Errorf("remove_edit_id %q does not exist in the current overlay", id)
		}
	}
	return remaining, removed, nil
}

func storedGUIEditKey(raw map[string]any) string {
	if id := strings.TrimSpace(stringField(raw, "id")); id != "" {
		return id
	}
	edit, err := decodeStoredGUIEdit(raw)
	if err != nil {
		return ""
	}
	key, _ := canonicalGUIEditID(edit)
	return key
}

func decodeStoredGUIEdit(raw map[string]any) (guiOverlayEditInput, error) {
	edit := guiOverlayEditInput{ID: stringField(raw, "id"), Target: stringField(raw, "target"), Op: stringField(raw, "op")}
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return edit, fmt.Errorf("payload is not an object")
	}
	edit.Payload = cloneMap(payload)
	return edit, nil
}

func normalizeLegacyGUIOverlayEdits(rawEdits []map[string]any, normalizedFile string) []map[string]any {
	out := cloneJSONMaps(rawEdits)
	for _, raw := range out {
		payload, ok := raw["payload"].(map[string]any)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(stringField(raw, "op"))) {
		case "swap_sprite":
			if strings.TrimSpace(stringField(payload, "gfx_ref")) != "" {
				delete(payload, "override_file")
			}
		case "set_text":
			if strings.TrimSpace(stringField(payload, "key")) != "" {
				continue
			}
			texts, ok := payload["lang_texts"].(map[string]any)
			if !ok || len(texts) == 0 {
				continue
			}
			payload["key"] = legacyGUIOverlayLocalisationKey(normalizedFile, stringField(raw, "target"))
			payload["is_new"] = true
		}
	}
	return out
}

func legacyGUIOverlayLocalisationKey(normalizedFile, target string) string {
	leaf := target
	if slash := strings.LastIndexByte(leaf, '/'); slash >= 0 {
		leaf = leaf[slash+1:]
	}
	var slug strings.Builder
	for _, char := range strings.ToUpper(leaf) {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
			slug.WriteRune(char)
		} else if slug.Len() == 0 || !strings.HasSuffix(slug.String(), "_") {
			slug.WriteByte('_')
		}
		if slug.Len() >= 48 {
			break
		}
	}
	name := strings.Trim(slug.String(), "_")
	if name == "" {
		name = "TEXT"
	}
	sum := sha256.Sum256([]byte(normalizedFile + "\x00" + target))
	return "GAMEVIEW_" + name + "_" + hex.EncodeToString(sum[:6])
}

func decodeAndValidateGUIOverlayEdits(rawEdits []map[string]any, windowName string) ([]guiOverlayEditInput, error) {
	edits := make([]guiOverlayEditInput, 0, len(rawEdits))
	seen := map[string]bool{}
	for index, raw := range rawEdits {
		edit, err := decodeStoredGUIEdit(raw)
		if err == nil {
			err = validateGUIEditSchema(&edit, guiEditWindowOr(edit.Target, windowName))
		}
		if err != nil {
			return nil, fmt.Errorf("candidate edit %d: %w", index, err)
		}
		if seen[edit.ID] {
			return nil, fmt.Errorf("candidate overlay contains duplicate edit id %q", edit.ID)
		}
		seen[edit.ID] = true
		edits = append(edits, edit)
	}
	return guiEditsInWindow(edits, windowName), nil
}

func guiEditWindowOr(target, fallback string) string {
	if window, _, _ := strings.Cut(strings.TrimSpace(target), "/"); window != "" {
		return window
	}
	return fallback
}

func guiEditsInWindow(edits []guiOverlayEditInput, windowName string) []guiOverlayEditInput {
	out := make([]guiOverlayEditInput, 0, len(edits))
	for _, edit := range edits {
		if guiEditWindowOr(edit.Target, "") == windowName {
			out = append(out, edit)
		}
	}
	return out
}

func verifyGUIEditsReadback(root map[string]any, edits []guiOverlayEditInput) (bool, []map[string]any) {
	diagnostics := []map[string]any{}
	for _, edit := range edits {
		ref := findGUINode(root, strings.Split(edit.Target, "/"))
		ok := false
		message := ""
		switch edit.Op {
		case "add_node":
			name := stringField(edit.Payload["node"].(map[string]any), "name")
			ok = findGUINode(root, strings.Split(edit.Target+"/"+name, "/")) != nil
			message = "added node was not found in the effective window"
		case "delete_node":
			ok = ref == nil
			message = "deleted node is still present in the effective window"
		case "hide_node":
			if ref != nil {
				raw, _ := ref.Node["raw"].(map[string]any)
				ok = strings.EqualFold(stringField(raw, "gameview_hidden"), "yes")
			}
			message = "hidden node is not marked hidden in the effective window"
		case "set_text":
			ok = ref != nil && stringField(ref.Node, "text") == projectedGUIText(edit.Payload)
			message = "text did not materialize in the effective window"
		case "swap_sprite":
			ok = ref != nil && stringField(ref.Node, "sprite") == stringField(edit.Payload, "gfx_ref")
			message = "sprite reference did not materialize in the effective window"
		case "set_layout":
			ok = ref != nil && guiLayoutMatches(ref.Node, edit.Payload)
			message = "layout did not materialize in the effective window"
		case "set_orientation":
			ok = ref != nil && strings.EqualFold(stringField(ref.Node, "orientation"), stringField(edit.Payload, "value"))
			message = "orientation did not materialize in the effective window"
		case "set_origo":
			expected := stringField(edit.Payload, "value")
			actual := stringField(ref.Node, "origo")
			ok = ref != nil && strings.EqualFold(actual, expected)
			message = "origo did not materialize in the effective window"
		}
		if !ok {
			diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "overlay_not_materialized", message, edit.Target, edit.Op))
		}
	}
	return len(diagnostics) == 0, diagnostics
}

func validateGUIOverlaySpriteReferences(window map[string]any, edits []guiOverlayEditInput) (bool, []map[string]any) {
	type use struct{ target, op string }
	refs := map[string]use{}
	var collectNode func(map[string]any, string)
	collectNode = func(node map[string]any, target string) {
		if sprite := strings.TrimSpace(stringField(node, "sprite")); sprite != "" {
			refs[sprite] = use{target: target, op: "add_node"}
		}
		if background := nodeField(node, "background"); background != nil {
			if sprite := strings.TrimSpace(stringField(background, "sprite")); sprite != "" {
				refs[sprite] = use{target: target, op: "add_node"}
			}
		}
		for _, child := range guiNodeChildren(node) {
			collectNode(child, target)
		}
	}
	for _, edit := range edits {
		switch edit.Op {
		case "swap_sprite":
			if sprite := strings.TrimSpace(stringField(edit.Payload, "gfx_ref")); sprite != "" {
				refs[sprite] = use{target: edit.Target, op: edit.Op}
			}
		case "add_node":
			if node := nodeField(edit.Payload, "node"); node != nil {
				collectNode(node, edit.Target)
			}
		}
	}
	if len(refs) == 0 {
		return true, nil
	}
	sprites, _ := window["sprites"].(map[string]any)
	diagnostics := []map[string]any{}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		meta, _ := sprites[name].(map[string]any)
		known := boolField(meta, "resolved") || boolField(meta, "overridden") || boolField(meta, "procedural") ||
			stringField(meta, "texturefile") != "" || int64Field(meta, "width") > 0 || int64Field(meta, "height") > 0
		if !known || boolField(meta, "missing_file") {
			at := refs[name]
			diagnostics = append(diagnostics, guiOverlayDiagnostic("error", "sprite_unresolved", fmt.Sprintf("sprite %q is not resolvable in the effective project GUI index", name), at.target, at.op))
		}
	}
	return len(diagnostics) == 0, diagnostics
}

func validateGUIOverlayLocalisationReferences(window map[string]any, edits []guiOverlayEditInput) (bool, []map[string]any) {
	type use struct {
		target              string
		op                  string
		requireLocalisation bool
	}
	refs := map[string][]use{}
	addReference := func(text string, at use) {
		if text != "" {
			refs[text] = append(refs[text], at)
		}
	}
	var collectAddedNodeTexts func(map[string]any, string)
	collectAddedNodeTexts = func(node map[string]any, target string) {
		name := strings.TrimSpace(stringField(node, "name"))
		if name != "" {
			target += "/" + name
		}
		if text, ok := node["text"].(string); ok {

			addReference(text, use{target: target, op: "add_node"})
		}
		for _, child := range guiNodeChildren(node) {
			collectAddedNodeTexts(child, target)
		}
	}
	for _, edit := range edits {
		switch edit.Op {
		case "set_text":
			if strings.TrimSpace(stringField(edit.Payload, "key")) != "" {
				addReference(projectedGUIText(edit.Payload), use{target: edit.Target, op: edit.Op, requireLocalisation: true})
			}
		case "add_node":
			if added := nodeField(edit.Payload, "node"); added != nil {

				name := strings.TrimSpace(stringField(added, "name"))
				if ref := findGUINode(nodeField(window, "node"), strings.Split(edit.Target+"/"+name, "/")); ref != nil {
					collectAddedNodeTexts(ref.Node, edit.Target)
				} else {
					collectAddedNodeTexts(added, edit.Target)
				}
			}
		}
	}
	if len(refs) == 0 {
		return true, nil
	}
	texts, _ := window["texts"].(map[string]any)
	diagnostics := []map[string]any{}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		meta, _ := texts[key].(map[string]any)
		kind := strings.ToLower(strings.TrimSpace(stringField(meta, "kind")))
		for _, at := range refs[key] {
			resolved := kind == "loc"
			if !at.requireLocalisation {

				resolved = resolved || kind == "literal" || kind == "runtime"
			}
			if resolved {
				continue
			}
			diagnostics = append(diagnostics, guiOverlayDiagnostic(
				"error",
				"localisation_unresolved",
				fmt.Sprintf("GUI text %q is not a literal, runtime expression, or resolvable localisation key in the effective project language index", key),
				at.target,
				at.op,
			))
		}
	}
	return len(diagnostics) == 0, diagnostics
}

func guiLayoutMatches(node, payload map[string]any) bool {
	position, _ := node["position"].(map[string]any)
	size, _ := node["size"].(map[string]any)
	for _, key := range []string{"x", "y"} {
		if expected, exists := payload[key]; exists && !scalarJSONEqual(position[key], expected) {
			return false
		}
	}
	for _, key := range []string{"width", "height"} {
		if expected, exists := payload[key]; exists && !scalarJSONEqual(size[key], expected) {
			return false
		}
	}
	return true
}

func scalarJSONEqual(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if string(leftJSON) == string(rightJSON) {
		return true
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func guiOverlayDiagnostic(severity, code, message, target, op string) map[string]any {
	out := map[string]any{"severity": severity, "code": code, "message": message}
	if target != "" {
		out["target"] = target
	}
	if op != "" {
		out["op"] = op
	}
	return out
}

func guiOverlayMapSliceField(in map[string]any, key string) ([]map[string]any, error) {
	raw, exists := in[key]
	if !exists || raw == nil {
		return []map[string]any{}, nil
	}
	switch typed := raw.(type) {
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for index := range typed {
			out[index] = cloneMap(typed[index])
		}
		return out, nil
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, value := range typed {
			row, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("project service returned an invalid GUI edit list")
			}
			out = append(out, cloneMap(row))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("project service returned an invalid GUI edit list")
	}
}

func nodeField(in map[string]any, key string) map[string]any {
	value, _ := in[key].(map[string]any)
	return value
}

func mapFieldCopy(in map[string]any, key string) map[string]any {
	value, _ := in[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return cloneMap(value)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	encoded, _ := json.Marshal(in)
	out := map[string]any{}
	_ = json.Unmarshal(encoded, &out)
	return out
}

func equalJSON(left, right any) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}

func wholeNumber(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if math.Trunc(typed) == typed && typed >= math.MinInt32 && typed <= math.MaxInt32 {
			return int(typed), true
		}
	}
	return 0, false
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

var guiLocalisationKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)
