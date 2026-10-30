package toolapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type createFocusTreeIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"blank -> project primary country"`
	TreeID     string `json:"tree_id,omitempty" jsonschema:"blank -> auto-generated collision-free id"`
}

type createFocusTreeOut struct {
	Replayed   bool   `json:"replayed,omitempty" jsonschema:"true when an exact existing tree was read after a lost or duplicate response; no new tree was created"`
	Code       string `json:"code,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Error      string `json:"error,omitempty"`
	Recovery   string `json:"recovery,omitempty"`

	EditRevision string `json:"edit_revision,omitempty"`
	FocusTreeID  int64  `json:"focus_tree_id"`
	TreeID       string `json:"tree_id"`
	CountryTag   string `json:"country_tag"`
}

func registerCreateFocusTree(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_focus_tree",
		Description: "Create a focus tree for a country. Blank country_tag → project primary; " +
			"blank tree_id → auto-generated collision-free id. Supply a stable explicit tree_id when continuing a task; " +
			"a duplicate never creates a second tree. An exact tree_id + explicit country_tag match is reused with replayed=true; " +
			"otherwise read the existing tree before deciding what remains.",
	}, func(ctx context.Context, _ *CallToolRequest, in createFocusTreeIn) (*CallToolResult, createFocusTreeOut, error) {
		body := map[string]any{"country_tag": in.CountryTag, "tree_id": in.TreeID}
		if strings.TrimSpace(in.TreeID) != "" {
			body["on_conflict"] = "error"
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/focus-trees", in.ProjectID), body)
		if err == nil && (focusInt64Field(resp, "id") <= 0 || stringField(resp, "tree_id") == "") {
			err = fmt.Errorf("create focus tree response omitted its persisted identity; read the existing tree before retrying")
		}
		replayed := false
		if err != nil {
			if recovered, ok := recoverExactFocusTreeCreate(ctx, proxy, in, err); ok {
				resp = recovered
				replayed = true
			} else {
				var upstream *RESTError
				if errors.As(err, &upstream) && upstream.Status == 409 && strings.TrimSpace(in.TreeID) != "" {
					reason := upstream.Code
					if reason == "" {
						reason = "focus_tree_identity_unverified"
					}
					out := createFocusTreeOut{Code: "invalid_request", HTTPStatus: 409, Reason: reason, Error: upstream.Detail,
						Recovery: "Read get_focus_tree and continue with its returned numeric id and edit_revision. An existing tree can be reused only when its explicit tree_id and country_tag both match; do not create a suffixed replacement."}
					encoded, _ := json.Marshal(out)
					return &CallToolResult{IsError: true, Content: []Content{&TextContent{Text: string(encoded)}}}, out, nil
				}
				return nil, createFocusTreeOut{}, err
			}
		}

		out := createFocusTreeOut{
			Replayed:     replayed,
			EditRevision: stringField(resp, "edit_revision"),
			TreeID:       stringField(resp, "tree_id"),
			CountryTag:   stringField(resp, "country_tag"),
		}
		out.FocusTreeID = focusInt64Field(resp, "id")
		return nil, out, nil
	})
}

func recoverExactFocusTreeCreate(ctx context.Context, proxy *RESTProxy, in createFocusTreeIn, createErr error) (map[string]any, bool) {
	if strings.TrimSpace(in.TreeID) == "" || strings.TrimSpace(in.CountryTag) == "" {
		return nil, false
	}
	var upstream *RESTError
	if errors.As(createErr, &upstream) && upstream.Status != 409 {
		return nil, false
	}
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	rows, err := proxyAuthedCallList(recoveryCtx, proxy, "GET", fmt.Sprintf("/api/projects/%d/focus-trees?tree_id=%s", in.ProjectID, url.QueryEscape(in.TreeID)), nil)
	if err != nil {
		return nil, false
	}
	for _, row := range rows {
		if stringField(row, "tree_id") == in.TreeID && strings.EqualFold(strings.TrimSpace(stringField(row, "country_tag")), strings.TrimSpace(in.CountryTag)) && focusInt64Field(row, "id") > 0 && stringField(row, "edit_revision") != "" {
			return row, true
		}
	}
	return nil, false
}

type createFocusIn struct {
	ProjectID         int64             `json:"project_id"`
	FocusTreeID       int64             `json:"focus_tree_id" jsonschema:"numeric id from create_focus_tree"`
	Name              string            `json:"name"`
	FocusID           string            `json:"focus_id,omitempty" jsonschema:"optional script id; omitted → derived from country tag + name. An explicit duplicate is rejected unless it is an exact retry of an already-committed ambiguous request"`
	X                 int               `json:"x,omitempty"`
	Y                 int               `json:"y,omitempty"`
	Icon              string            `json:"icon,omitempty" jsonschema:"default GFX_goal_unknown"`
	Cost              *float64          `json:"cost,omitempty" jsonschema:"default 10.0"`
	Prerequisites     [][]string        `json:"prerequisites,omitempty" jsonschema:"list of AND-groups of focus_ids"`
	MutuallyExclusive []string          `json:"mutually_exclusive,omitempty"`
	CompletionReward  string            `json:"completion_reward,omitempty"`
	NameEn            string            `json:"name_en,omitempty"`
	NameZh            string            `json:"name_zh,omitempty"`
	DescEn            string            `json:"desc_en,omitempty"`
	DescZh            string            `json:"desc_zh,omitempty"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string `json:"description_locs,omitempty"`
	EffectTooltipLocs map[string]string `json:"effect_tooltip_locs,omitempty"`
}

type createFocusOut struct {
	TreeEditRevision  string              `json:"tree_edit_revision,omitempty"`
	Diagnostics       *[]saveDiagnostic   `json:"diagnostics,omitempty"`
	TreeDiagnostics   *[]saveDiagnostic   `json:"tree_diagnostics,omitempty"`
	FocusNodeID       int64               `json:"focus_node_id"`
	FocusID           string              `json:"focus_id" jsonschema:"the id actually persisted — may differ from the requested one when a duplicate was auto-suffixed"`
	Name              string              `json:"name"`
	Icon              string              `json:"icon"`
	Cost              float64             `json:"cost"`
	X                 int64               `json:"x"`
	Y                 int64               `json:"y"`
	Prerequisites     [][]string          `json:"prerequisites"`
	PrerequisiteExpr  string              `json:"prerequisite_expression" jsonschema:"unambiguous saved logic: AND joins outer groups; OR joins ids inside one group"`
	MutuallyExclusive []string            `json:"mutually_exclusive"`
	CompletionReward  string              `json:"completion_reward"`
	NameEn            string              `json:"name_en"`
	NameZh            string              `json:"name_zh"`
	DescEn            string              `json:"desc_en"`
	DescZh            string              `json:"desc_zh"`
	DisplayNameLocs   map[string]string   `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string   `json:"description_locs,omitempty"`
	EffectTooltipLocs map[string]string   `json:"effect_tooltip_locs,omitempty"`
	ReferenceCheck    string              `json:"reference_check" jsonschema:"not_applicable | passed | failed | unavailable"`
	UnresolvedRefs    []DanglingReference `json:"unresolved_references"`

	Warnings []string `json:"warnings,omitempty" jsonschema:"upstream warnings, e.g. 'focus_id X collided · auto-suffixed to X_1.'"`
}

func registerCreateFocus(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_focus",
		Description: "Create a focus node in a tree. focus_tree_id = numeric id from " +
			"create_focus_tree. prerequisites = list of AND-groups of focus_ids. " +
			"For a playable merge after either of two mutually-exclusive branches use one OR group, " +
			"for example [[\"left\",\"right\"]]; [[\"left\"],[\"right\"]] requires both and is normally impossible. " +
			"An explicit duplicate focus_id returns a conflict. Omit focus_id to derive one from the name. " +
			"Create every entity named by completion_reward before this focus. Treat reference_check=failed or unavailable as unfinished, not success. " +
			"Content diagnostics are advisory; use the returned focus_id and prerequisite_expression when reporting saved state.",
	}, func(ctx context.Context, _ *CallToolRequest, in createFocusIn) (*CallToolResult, createFocusOut, error) {
		icon := in.Icon
		if icon == "" {
			icon = "GFX_goal_unknown"
		}
		cost := focusFloatOr(in.Cost, 10.0)
		prereqs := in.Prerequisites
		if prereqs == nil {
			prereqs = [][]string{}

		}
		mutex := in.MutuallyExclusive
		if mutex == nil {
			mutex = []string{}
		}
		body := map[string]any{
			"name":               in.Name,
			"on_conflict":        "error",
			"x":                  in.X,
			"y":                  in.Y,
			"icon":               icon,
			"cost":               cost,
			"prerequisites":      prereqs,
			"mutually_exclusive": mutex,
			"completion_reward":  in.CompletionReward,
			"name_en":            in.NameEn,
			"name_zh":            in.NameZh,
			"desc_en":            in.DescEn,
			"desc_zh":            in.DescZh,
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		if in.EffectTooltipLocs != nil {
			body["effect_tooltip_locs"] = in.EffectTooltipLocs
		}
		if in.FocusID != "" {
			body["focus_id"] = in.FocusID
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes", in.ProjectID, in.FocusTreeID), body)
		recoveryWarnings := []string{}
		if err != nil {
			if recovered, ok := recoverExactFocusCreate(ctx, proxy, in, icon, cost, prereqs, mutex, err); ok {
				resp = recovered
				recoveryWarnings = append(recoveryWarnings,
					"The create response was ambiguous, but an exact persisted focus_id match was found. The existing committed node was returned; do not retry this create.")
			} else {
				return nil, createFocusOut{}, err
			}
		}

		nodeID := focusInt64Field(resp, "id")
		savedName := stringField(resp, "name")
		if _, present := resp["name"]; !present {
			savedName = in.Name
		}
		savedIcon := stringField(resp, "icon")
		if _, present := resp["icon"]; !present {
			savedIcon = icon
		}
		savedCost := focusFloat64MapField(resp, "cost")
		if _, present := resp["cost"]; !present {
			savedCost = cost
		}
		savedX, savedY := focusInt64Field(resp, "x"), focusInt64Field(resp, "y")
		if _, present := resp["x"]; !present {
			savedX = int64(in.X)
		}
		if _, present := resp["y"]; !present {
			savedY = int64(in.Y)
		}
		savedPrerequisites := focusStringMatrixField(resp, "prerequisites")
		if _, present := resp["prerequisites"]; !present {
			savedPrerequisites = prereqs
		}
		savedMutex := focusStringSliceField(resp, "mutually_exclusive")
		if _, present := resp["mutually_exclusive"]; !present {
			savedMutex = mutex
		}
		savedReward := stringField(resp, "completion_reward")
		if _, present := resp["completion_reward"]; !present {
			savedReward = in.CompletionReward
		}
		referenceCheck, unresolved, referenceWarnings := checkFocusReferences(
			ctx, proxy, in.ProjectID, nodeID, savedReward,
		)
		out := createFocusOut{
			TreeEditRevision:  stringField(resp, "tree_edit_revision"),
			Diagnostics:       saveDiagnosticsField(resp),
			TreeDiagnostics:   namedSaveDiagnosticsField(resp, "tree_diagnostics"),
			FocusID:           stringField(resp, "focus_id"),
			Name:              savedName,
			Icon:              savedIcon,
			Cost:              savedCost,
			X:                 savedX,
			Y:                 savedY,
			Prerequisites:     savedPrerequisites,
			PrerequisiteExpr:  focusPrerequisiteExpression(savedPrerequisites),
			MutuallyExclusive: savedMutex,
			CompletionReward:  savedReward,
			NameEn:            focusStringOr(resp, "name_en", in.NameEn),
			NameZh:            focusStringOr(resp, "name_zh", in.NameZh),
			DescEn:            focusStringOr(resp, "desc_en", in.DescEn),
			DescZh:            focusStringOr(resp, "desc_zh", in.DescZh),
			DisplayNameLocs:   focusStringMapOr(resp, "display_name_locs", in.DisplayNameLocs),
			DescriptionLocs:   focusStringMapOr(resp, "description_locs", in.DescriptionLocs),
			EffectTooltipLocs: focusStringMapOr(resp, "effect_tooltip_locs", in.EffectTooltipLocs),
			ReferenceCheck:    referenceCheck,
			UnresolvedRefs:    unresolved,
			Warnings:          append(append(stringSliceField(resp, "warnings"), recoveryWarnings...), referenceWarnings...),
		}
		out.FocusNodeID = nodeID
		return nil, out, nil
	})
}

type updateFocusIn struct {
	ProjectID          int64              `json:"project_id"`
	FocusTreeID        int64              `json:"focus_tree_id"`
	NodeID             int64              `json:"node_id" jsonschema:"numeric node_id"`
	Name               *string            `json:"name,omitempty"`
	Cost               *float64           `json:"cost,omitempty"`
	X                  *int               `json:"x,omitempty"`
	Y                  *int               `json:"y,omitempty"`
	Icon               *string            `json:"icon,omitempty"`
	Prerequisites      *[][]string        `json:"prerequisites,omitempty"`
	MutuallyExclusive  *[]string          `json:"mutually_exclusive,omitempty"`
	CompletionReward   *string            `json:"completion_reward,omitempty"`
	Available          *string            `json:"available,omitempty"`
	RelativePositionID *string            `json:"relative_position_id,omitempty"`
	NameEn             *string            `json:"name_en,omitempty"`
	NameZh             *string            `json:"name_zh,omitempty"`
	DescEn             *string            `json:"desc_en,omitempty"`
	DescZh             *string            `json:"desc_zh,omitempty"`
	DisplayNameLocs    *map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs    *map[string]string `json:"description_locs,omitempty"`
	EffectTooltipLocs  *map[string]string `json:"effect_tooltip_locs,omitempty"`
}

type updateFocusOut struct {
	TreeEditRevision  string              `json:"tree_edit_revision,omitempty"`
	Diagnostics       *[]saveDiagnostic   `json:"diagnostics,omitempty"`
	TreeDiagnostics   *[]saveDiagnostic   `json:"tree_diagnostics,omitempty"`
	FocusNodeID       int64               `json:"focus_node_id"`
	FocusID           string              `json:"focus_id"`
	Name              *string             `json:"name,omitempty"`
	Icon              *string             `json:"icon,omitempty"`
	Cost              *float64            `json:"cost,omitempty"`
	X                 *int64              `json:"x,omitempty"`
	Y                 *int64              `json:"y,omitempty"`
	Prerequisites     *[][]string         `json:"prerequisites,omitempty"`
	PrerequisiteExpr  string              `json:"prerequisite_expression,omitempty" jsonschema:"unambiguous saved logic: AND joins outer groups; OR joins ids inside one group"`
	MutuallyExclusive *[]string           `json:"mutually_exclusive,omitempty"`
	CompletionReward  *string             `json:"completion_reward,omitempty"`
	Available         *string             `json:"available,omitempty"`
	NameEn            *string             `json:"name_en,omitempty"`
	NameZh            *string             `json:"name_zh,omitempty"`
	DescEn            *string             `json:"desc_en,omitempty"`
	DescZh            *string             `json:"desc_zh,omitempty"`
	DisplayNameLocs   *map[string]string  `json:"display_name_locs,omitempty"`
	DescriptionLocs   *map[string]string  `json:"description_locs,omitempty"`
	EffectTooltipLocs *map[string]string  `json:"effect_tooltip_locs,omitempty"`
	ReferenceCheck    string              `json:"reference_check" jsonschema:"not_applicable | passed | failed | unavailable"`
	UnresolvedRefs    []DanglingReference `json:"unresolved_references"`
	Warnings          []string            `json:"warnings,omitempty"`
}

func registerUpdateFocus(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_focus",
		Description: "Edit a focus node in place (by numeric node_id). Only supplied fields change. " +
			"prerequisites/mutually_exclusive REPLACE — use connect_focus for wiring-only edits. " +
			"One inner group is OR; separate outer groups are AND. A merge after mutually-exclusive branches normally needs [[\"left\",\"right\"]], not [[\"left\"],[\"right\"]]. " +
			"Create completion_reward targets first and treat reference_check=failed or unavailable as unfinished.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateFocusIn) (*CallToolResult, updateFocusOut, error) {
		body := map[string]any{}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.Cost != nil {
			body["cost"] = *in.Cost
		}
		if in.X != nil {
			body["x"] = *in.X
		}
		if in.Y != nil {
			body["y"] = *in.Y
		}
		if in.Icon != nil {
			body["icon"] = *in.Icon
		}
		if in.Prerequisites != nil {
			body["prerequisites"] = *in.Prerequisites
		}
		if in.MutuallyExclusive != nil {
			body["mutually_exclusive"] = *in.MutuallyExclusive
		}
		if in.CompletionReward != nil {
			body["completion_reward"] = *in.CompletionReward
		}
		if in.Available != nil {
			body["available"] = *in.Available
		}
		if in.RelativePositionID != nil {
			body["relative_position_id"] = *in.RelativePositionID
		}
		if in.NameEn != nil {
			body["name_en"] = *in.NameEn
		}
		if in.NameZh != nil {
			body["name_zh"] = *in.NameZh
		}
		if in.DescEn != nil {
			body["desc_en"] = *in.DescEn
		}
		if in.DescZh != nil {
			body["desc_zh"] = *in.DescZh
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = *in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = *in.DescriptionLocs
		}
		if in.EffectTooltipLocs != nil {
			body["effect_tooltip_locs"] = *in.EffectTooltipLocs
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes/%d", in.ProjectID, in.FocusTreeID, in.NodeID), body)
		if err != nil {
			return nil, updateFocusOut{}, err
		}

		nodeID := focusInt64Field(resp, "id")
		if nodeID <= 0 {
			nodeID = in.NodeID
		}
		savedPrerequisites := focusSavedStringMatrix(resp, "prerequisites", in.Prerequisites)
		savedCompletionReward := focusSavedString(resp, "completion_reward", in.CompletionReward)
		referenceCheck, unresolved, referenceWarnings := "not_applicable", []DanglingReference{}, []string{}
		if in.CompletionReward != nil && savedCompletionReward != nil {
			referenceCheck, unresolved, referenceWarnings = checkFocusReferences(
				ctx, proxy, in.ProjectID, nodeID, *savedCompletionReward,
			)
		}
		out := updateFocusOut{
			TreeEditRevision:  stringField(resp, "tree_edit_revision"),
			TreeDiagnostics:   namedSaveDiagnosticsField(resp, "tree_diagnostics"),
			Diagnostics:       saveDiagnosticsField(resp),
			FocusID:           stringField(resp, "focus_id"),
			Name:              focusSavedString(resp, "name", in.Name),
			Icon:              focusSavedString(resp, "icon", in.Icon),
			Cost:              focusSavedFloat64(resp, "cost", in.Cost),
			X:                 focusSavedInt64(resp, "x", in.X),
			Y:                 focusSavedInt64(resp, "y", in.Y),
			Prerequisites:     savedPrerequisites,
			MutuallyExclusive: focusSavedStringSlice(resp, "mutually_exclusive", in.MutuallyExclusive),
			CompletionReward:  savedCompletionReward,
			Available:         focusSavedString(resp, "available", in.Available),
			NameEn:            focusSavedString(resp, "name_en", in.NameEn),
			NameZh:            focusSavedString(resp, "name_zh", in.NameZh),
			DescEn:            focusSavedString(resp, "desc_en", in.DescEn),
			DescZh:            focusSavedString(resp, "desc_zh", in.DescZh),
			DisplayNameLocs:   focusSavedStringMap(resp, "display_name_locs", in.DisplayNameLocs),
			DescriptionLocs:   focusSavedStringMap(resp, "description_locs", in.DescriptionLocs),
			EffectTooltipLocs: focusSavedStringMap(resp, "effect_tooltip_locs", in.EffectTooltipLocs),
			ReferenceCheck:    referenceCheck,
			UnresolvedRefs:    unresolved,
			Warnings:          referenceWarnings,
		}
		if savedPrerequisites != nil {
			out.PrerequisiteExpr = focusPrerequisiteExpression(*savedPrerequisites)
		}
		out.FocusNodeID = nodeID
		return nil, out, nil
	})
}

type connectFocusIn struct {
	ProjectID         int64       `json:"project_id"`
	FocusTreeID       int64       `json:"focus_tree_id"`
	NodeID            int64       `json:"node_id"`
	Prerequisites     *[][]string `json:"prerequisites,omitempty" jsonschema:"AND-groups of focus_id strings (each inner list = OR alternatives)"`
	MutuallyExclusive *[]string   `json:"mutually_exclusive,omitempty" jsonschema:"focus_id strings"`
}

type connectFocusOut struct {
	TreeEditRevision  string            `json:"tree_edit_revision,omitempty"`
	Diagnostics       *[]saveDiagnostic `json:"diagnostics,omitempty"`
	TreeDiagnostics   *[]saveDiagnostic `json:"tree_diagnostics,omitempty"`
	FocusNodeID       int64             `json:"focus_node_id"`
	FocusID           string            `json:"focus_id"`
	Prerequisites     *[][]string       `json:"prerequisites,omitempty"`
	MutuallyExclusive *[]string         `json:"mutually_exclusive,omitempty"`
	PrerequisiteExpr  string            `json:"prerequisite_expression,omitempty" jsonschema:"unambiguous saved logic: AND joins outer groups; OR joins ids inside one group"`
}

func registerConnectFocus(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "connect_focus",
		Description: "Wire a focus node's graph edges. prerequisites = AND-groups of focus_id strings " +
			"(each inner list = OR alternatives); mutually_exclusive = focus_id strings. " +
			"Both REPLACE the existing lists — send the full set. Refs are focus_id STRINGS, not node ids. " +
			"For a two-sided mutual exclusion use set_focus_mutual_exclusion instead of calling this twice. " +
			"For a playable merge after either mutually-exclusive branch send one OR group [[\"left\",\"right\"]]; " +
			"separate groups [[\"left\"],[\"right\"]] mean AND and are normally impossible. Report the returned prerequisite_expression verbatim.",
	}, func(ctx context.Context, _ *CallToolRequest, in connectFocusIn) (*CallToolResult, connectFocusOut, error) {
		body := map[string]any{}
		if in.Prerequisites != nil {
			body["prerequisites"] = *in.Prerequisites
		}
		if in.MutuallyExclusive != nil {
			body["mutually_exclusive"] = *in.MutuallyExclusive
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes/%d", in.ProjectID, in.FocusTreeID, in.NodeID), body)
		if err != nil {
			return nil, connectFocusOut{}, err
		}

		nodeID := focusInt64Field(resp, "id")
		if nodeID <= 0 {
			nodeID = in.NodeID
		}
		out := connectFocusOut{
			TreeEditRevision:  stringField(resp, "tree_edit_revision"),
			TreeDiagnostics:   namedSaveDiagnosticsField(resp, "tree_diagnostics"),
			Diagnostics:       saveDiagnosticsField(resp),
			FocusID:           stringField(resp, "focus_id"),
			Prerequisites:     focusSavedStringMatrix(resp, "prerequisites", in.Prerequisites),
			MutuallyExclusive: focusSavedStringSlice(resp, "mutually_exclusive", in.MutuallyExclusive),
		}
		if out.Prerequisites != nil {
			out.PrerequisiteExpr = focusPrerequisiteExpression(*out.Prerequisites)
		}
		out.FocusNodeID = nodeID
		return nil, out, nil
	})
}

type setFocusMutualExclusionIn struct {
	ProjectID   int64 `json:"project_id"`
	FocusTreeID int64 `json:"focus_tree_id"`
	NodeID      int64 `json:"node_id" jsonschema:"numeric node id for one branch entry"`
	OtherNodeID int64 `json:"other_node_id" jsonschema:"numeric node id for the other branch entry"`
	Enabled     bool  `json:"enabled" jsonschema:"true adds the relation; false removes it"`
}

type focusMutualExclusionNode struct {
	FocusNodeID       int64    `json:"focus_node_id"`
	FocusID           string   `json:"focus_id"`
	MutuallyExclusive []string `json:"mutually_exclusive"`
}

type setFocusMutualExclusionOut struct {
	TreeEditRevision string                     `json:"tree_edit_revision,omitempty"`
	Enabled          bool                       `json:"enabled"`
	Relation         string                     `json:"relation"`
	Focuses          []focusMutualExclusionNode `json:"focuses"`
	Warnings         []string                   `json:"warnings,omitempty"`
}

func registerSetFocusMutualExclusion(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "set_focus_mutual_exclusion",
		Description: "Atomically add or remove a mutual-exclusion relation between two focus nodes. " +
			"Use this instead of two connect_focus calls: the program updates both nodes in one transaction " +
			"and returns both saved mutually_exclusive lists. enabled=true adds the relation; false removes it.",
	}, func(ctx context.Context, _ *CallToolRequest, in setFocusMutualExclusionIn) (*CallToolResult, setFocusMutualExclusionOut, error) {
		rows, err := proxyAuthedCallList(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes/%d/mutual-exclusion",
				in.ProjectID, in.FocusTreeID, in.NodeID),
			map[string]any{"other_node_id": in.OtherNodeID, "enabled": in.Enabled})
		if err != nil {
			return nil, setFocusMutualExclusionOut{}, err
		}

		out := setFocusMutualExclusionOut{
			Enabled: in.Enabled,
			Focuses: make([]focusMutualExclusionNode, 0, len(rows)),
		}
		for _, row := range rows {
			nodeID := focusInt64Field(row, "id")
			out.Focuses = append(out.Focuses, focusMutualExclusionNode{
				FocusNodeID:       nodeID,
				FocusID:           stringField(row, "focus_id"),
				MutuallyExclusive: focusStringSliceField(row, "mutually_exclusive"),
			})
			if revision := stringField(row, "tree_edit_revision"); revision != "" {
				out.TreeEditRevision = revision
			}
		}
		if len(out.Focuses) == 2 {
			verb := "NOT MUTUALLY EXCLUSIVE WITH"
			if in.Enabled {
				verb = "MUTUALLY EXCLUSIVE WITH"
			}
			out.Relation = out.Focuses[0].FocusID + " " + verb + " " + out.Focuses[1].FocusID
		} else {
			out.Warnings = append(out.Warnings,
				"The write returned an unexpected node count; re-read the focus tree before reporting the relation.")
		}
		return nil, out, nil
	})
}

type deleteFocusIn struct {
	ProjectID   int64 `json:"project_id"`
	FocusTreeID int64 `json:"focus_tree_id"`
	NodeID      int64 `json:"node_id" jsonschema:"numeric node_id"`
}

type deleteFocusOut struct {
	Deleted     bool  `json:"deleted"`
	NodeID      int64 `json:"node_id"`
	FocusTreeID int64 `json:"focus_tree_id"`
}

func registerDeleteFocus(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "delete_focus",
		Description: "Delete a focus node by numeric node_id (cascade-strips sibling references to it).",
	}, func(ctx context.Context, _ *CallToolRequest, in deleteFocusIn) (*CallToolResult, deleteFocusOut, error) {
		_, err := proxyAuthedCall(ctx, proxy, "DELETE",
			fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes/%d", in.ProjectID, in.FocusTreeID, in.NodeID), nil)
		if err != nil {
			return nil, deleteFocusOut{}, err
		}

		out := deleteFocusOut{
			Deleted:     true,
			NodeID:      in.NodeID,
			FocusTreeID: in.FocusTreeID,
		}
		return nil, out, nil
	})
}

func focusFloatOr(p *float64, fallback float64) float64 {
	if p == nil {
		return fallback
	}
	return *p
}

func focusFloat64MapField(m map[string]any, key string) float64 {
	value, _ := m[key].(float64)
	return value
}

func focusStringOr(m map[string]any, key, fallback string) string {
	if _, present := m[key]; present {
		return stringField(m, key)
	}
	return fallback
}

func focusStringMapOr(m map[string]any, key string, fallback map[string]string) map[string]string {
	if _, present := m[key]; present {
		value := stringMapField(m, key)
		if value == nil {
			return map[string]string{}
		}
		return value
	}
	if fallback == nil {
		return map[string]string{}
	}
	return fallback
}

func focusSavedString(m map[string]any, key string, fallback *string) *string {
	if _, present := m[key]; present {
		value := stringField(m, key)
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := *fallback
	return &value
}

func focusSavedFloat64(m map[string]any, key string, fallback *float64) *float64 {
	if _, present := m[key]; present {
		value := focusFloat64MapField(m, key)
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := *fallback
	return &value
}

func focusSavedInt64(m map[string]any, key string, fallback *int) *int64 {
	if _, present := m[key]; present {
		value := focusInt64Field(m, key)
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := int64(*fallback)
	return &value
}

func focusSavedStringMatrix(m map[string]any, key string, fallback *[][]string) *[][]string {
	if _, present := m[key]; present {
		value := focusStringMatrixField(m, key)
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := *fallback
	return &value
}

func focusSavedStringSlice(m map[string]any, key string, fallback *[]string) *[]string {
	if _, present := m[key]; present {
		value := focusStringSliceField(m, key)
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := *fallback
	return &value
}

func focusSavedStringMap(m map[string]any, key string, fallback *map[string]string) *map[string]string {
	if _, present := m[key]; present {
		value := stringMapField(m, key)
		if value == nil {
			value = map[string]string{}
		}
		return &value
	}
	if fallback == nil {
		return nil
	}
	value := *fallback
	return &value
}

func recoverExactFocusCreate(
	ctx context.Context,
	proxy *RESTProxy,
	in createFocusIn,
	icon string,
	cost float64,
	prerequisites [][]string,
	mutex []string,
	createErr error,
) (map[string]any, bool) {
	if strings.TrimSpace(in.FocusID) == "" {
		return nil, false
	}
	var restErr *RESTError
	if errors.As(createErr, &restErr) && restErr.Status != 409 {
		return nil, false
	}
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	rows, err := proxyAuthedCallList(recoveryCtx, proxy, "GET",
		fmt.Sprintf("/api/projects/%d/focus-trees/%d/nodes", in.ProjectID, in.FocusTreeID), nil)
	if err != nil {
		return nil, false
	}
	for _, row := range rows {
		if stringField(row, "focus_id") != strings.TrimSpace(in.FocusID) ||
			stringField(row, "name") != in.Name ||
			stringField(row, "icon") != icon ||
			focusFloat64MapField(row, "cost") != cost ||
			focusInt64Field(row, "x") != int64(in.X) ||
			focusInt64Field(row, "y") != int64(in.Y) ||
			!reflect.DeepEqual(focusStringMatrixField(row, "prerequisites"), prerequisites) ||
			!reflect.DeepEqual(focusStringSliceField(row, "mutually_exclusive"), mutex) ||
			stringField(row, "completion_reward") != in.CompletionReward {
			continue
		}
		if in.NameEn != "" && stringField(row, "name_en") != in.NameEn ||
			in.NameZh != "" && stringField(row, "name_zh") != in.NameZh ||
			in.DescEn != "" && stringField(row, "desc_en") != in.DescEn ||
			in.DescZh != "" && stringField(row, "desc_zh") != in.DescZh ||
			in.DisplayNameLocs != nil && !reflect.DeepEqual(stringMapField(row, "display_name_locs"), in.DisplayNameLocs) ||
			in.DescriptionLocs != nil && !reflect.DeepEqual(stringMapField(row, "description_locs"), in.DescriptionLocs) ||
			in.EffectTooltipLocs != nil && !reflect.DeepEqual(stringMapField(row, "effect_tooltip_locs"), in.EffectTooltipLocs) {
			continue
		}
		return row, true
	}
	return nil, false
}

func focusPrerequisiteExpression(groups [][]string) string {
	if len(groups) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		if len(group) == 0 {
			parts = append(parts, "UNSATISFIABLE(empty group)")
			continue
		}
		part := strings.Join(group, " OR ")
		if len(group) > 1 {
			part = "(" + part + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " AND ")
}

func checkFocusReferences(ctx context.Context, proxy *RESTProxy, projectID, nodeID int64, reward string) (string, []DanglingReference, []string) {
	return checkEntityReferences(ctx, proxy, projectID, "focus", nodeID, containsIndexedReference(strings.TrimSpace(reward)))
}

func focusInt64Field(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func focusStringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func focusStringMatrixField(m map[string]any, key string) [][]string {
	raw, ok := m[key].([]any)
	if !ok {
		return [][]string{}
	}
	out := make([][]string, 0, len(raw))
	for _, v := range raw {
		if inner, ok := v.([]any); ok {
			group := make([]string, 0, len(inner))
			for _, e := range inner {
				if s, ok := e.(string); ok {
					group = append(group, s)
				}
			}
			out = append(out, group)
		}
	}
	return out
}
