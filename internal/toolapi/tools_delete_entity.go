package toolapi

import (
	"context"
	"fmt"
	"strings"
)

type InboundReference struct {
	SourceKind string `json:"source_kind"`
	SourceID   int64  `json:"source_id"`
	SourceName string `json:"source_name"`
	FieldName  string `json:"field_name"`
	EffectName string `json:"effect_name"`
}

type previewDeleteEntityIn struct {
	ProjectID int64  `json:"project_id"`
	Kind      string `json:"kind"`
	ID        int64  `json:"id" jsonschema:"row id from list_entities"`
	ParentID  int64  `json:"parent_id,omitempty"`
	Locale    string `json:"locale,omitempty"`
}

type previewDeleteEntityOut struct {
	ProjectID    int64              `json:"project_id"`
	Kind         string             `json:"kind"`
	Target       EntitySummary      `json:"target"`
	ReferencedBy []InboundReference `json:"referenced_by"`

	ReferencedByRawFiles []RawFileReference `json:"referenced_by_raw_files"`
	ReferenceCount       int                `json:"reference_count"`
	SafeToDelete         bool               `json:"safe_to_delete"`
	Coverage             string             `json:"coverage"`
	Recommendation       string             `json:"recommendation"`
	ConfirmToken         string             `json:"confirm_token"`
}

func registerPreviewDeleteEntity(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "preview_delete_entity",
		Description: "Show what deleting a piece of content would break, WITHOUT deleting it. " +
			"Returns the thing itself plus every place that references it. Call this before " +
			"delete_entity — deleting something other content points at does not fail, it leaves " +
			"references to nothing, and those fail silently in-game.",
	}, func(ctx context.Context, _ *CallToolRequest, in previewDeleteEntityIn) (*CallToolResult, previewDeleteEntityOut, error) {
		kind, target, err := loadDeleteTarget(ctx, proxy, in.ProjectID, in.Kind, in.ID, in.ParentID)
		if err != nil {
			return nil, previewDeleteEntityOut{}, err
		}
		inbound, err := inboundReferences(ctx, proxy, in.ProjectID, kind, target)
		if err != nil {
			return nil, previewDeleteEntityOut{}, err
		}

		rawInbound, err := rawFileReferencesTo(ctx, proxy, in.ProjectID, target.Key)
		if err != nil {
			return nil, previewDeleteEntityOut{}, err
		}
		out := previewDeleteEntityOut{
			ProjectID: in.ProjectID, Kind: kind.Kind, Target: target,
			ReferencedBy: inbound, ReferenceCount: len(inbound) + len(rawInbound),
			ReferencedByRawFiles: rawInbound,
			SafeToDelete:         len(inbound) == 0 && len(rawInbound) == 0,
			Coverage:             "Searched both the project's structured content and its raw script files.",
			ConfirmToken:         deleteConfirmToken(kind, in.ID),
		}
		if out.SafeToDelete {
			out.Recommendation = "Nothing references this, in structured content or in raw files. " +
				"Call delete_entity with confirm set to the confirm_token above."
		} else {
			out.Recommendation = fmt.Sprintf(
				"%d place(s) reference this. Repoint or remove them FIRST — deleting now leaves "+
					"references to nothing, which the game ignores silently rather than reporting.",
				out.ReferenceCount)
		}
		return nil, out, nil
	})
}

type deleteEntityIn struct {
	ProjectID int64  `json:"project_id"`
	Kind      string `json:"kind"`
	ID        int64  `json:"id"`
	ParentID  int64  `json:"parent_id,omitempty"`
	Confirm   string `json:"confirm" jsonschema:"the confirm_token from preview_delete_entity — a plain true is not accepted"`
}

type deleteEntityOut struct {
	ProjectID      int64  `json:"project_id"`
	Kind           string `json:"kind"`
	ID             int64  `json:"id"`
	Key            string `json:"key"`
	Deleted        bool   `json:"deleted"`
	ReferencesLeft int    `json:"references_left"`
	Note           string `json:"note,omitempty"`
}

func registerDeleteEntity(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "delete_entity",
		Description: "Permanently delete a piece of content. Requires `confirm` set to the " +
			"confirm_token that preview_delete_entity returned for this exact kind and id — so a " +
			"deletion cannot happen without having looked at its consequences first. There is no " +
			"undo.",
	}, func(ctx context.Context, _ *CallToolRequest, in deleteEntityIn) (*CallToolResult, deleteEntityOut, error) {
		kind, target, err := loadDeleteTarget(ctx, proxy, in.ProjectID, in.Kind, in.ID, in.ParentID)
		if err != nil {
			return nil, deleteEntityOut{}, err
		}
		want := deleteConfirmToken(kind, in.ID)
		if strings.TrimSpace(in.Confirm) != want {

			return nil, deleteEntityOut{}, fmt.Errorf(
				"delete_entity refused: `confirm` must be the confirm_token from "+
					"preview_delete_entity for %s id %d. Run the preview first and read what "+
					"references it", kind.Kind, in.ID)
		}
		inbound, err := inboundReferences(ctx, proxy, in.ProjectID, kind, target)
		if err != nil {
			return nil, deleteEntityOut{}, err
		}
		base, err := entityListPath(in.ProjectID, kind, in.ParentID)
		if err != nil {
			return nil, deleteEntityOut{}, err
		}
		if _, err := proxyAuthedCall(ctx, proxy, "DELETE", fmt.Sprintf("%s/%d", base, in.ID), nil); err != nil {
			return nil, deleteEntityOut{}, err
		}
		out := deleteEntityOut{
			ProjectID: in.ProjectID, Kind: kind.Kind, ID: in.ID, Key: target.Key,
			Deleted: true, ReferencesLeft: len(inbound),
		}
		if len(inbound) > 0 {

			out.Note = fmt.Sprintf(
				"%d reference(s) now point at content that no longer exists. Fix them, or those "+
					"effects will silently do nothing in-game. validate_project finds them.",
				len(inbound))
		}
		return nil, out, nil
	})
}

func deleteConfirmToken(kind entityKind, id int64) string {
	return fmt.Sprintf("delete-%s-%d", kind.Kind, id)
}

func loadDeleteTarget(ctx context.Context, proxy *RESTProxy, projectID int64, kindName string,
	id, parentID int64) (entityKind, EntitySummary, error) {

	kind, err := lookupEntityKind(kindName)
	if err != nil {
		return entityKind{}, EntitySummary{}, err
	}
	if id <= 0 {
		return entityKind{}, EntitySummary{}, fmt.Errorf(
			"needs the row id (the `id` field from list_entities), not the game identifier")
	}
	base, err := entityListPath(projectID, kind, parentID)
	if err != nil {
		return entityKind{}, EntitySummary{}, err
	}
	rows, err := proxyAuthedCallAny(ctx, proxy, "GET", fmt.Sprintf("%s/%d", base, id), nil)
	if err != nil {
		return entityKind{}, EntitySummary{}, err
	}
	if len(rows) == 0 {
		return entityKind{}, EntitySummary{}, fmt.Errorf(
			"no %s with id %d in project %d — nothing to delete", kind.Kind, id, projectID)
	}
	return kind, summarize(kind, rows[0], newChildCounter(ctx, proxy, projectID)), nil
}

func inboundReferences(ctx context.Context, proxy *RESTProxy, projectID int64,
	kind entityKind, target EntitySummary) ([]InboundReference, error) {

	if target.Key == "" {
		return []InboundReference{}, nil
	}
	resp, err := proxyAuthedCall(ctx, proxy, "GET",
		fmt.Sprintf("/api/projects/%d/relations/graph", projectID), nil)
	if err != nil {
		return nil, err
	}
	edges, _ := resp["edges"].([]any)
	inbound := []InboundReference{}
	for _, raw := range edges {
		edge, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if !strings.EqualFold(stringField(edge, "target_id"), target.Key) {
			continue
		}
		inbound = append(inbound, InboundReference{
			SourceKind: stringField(edge, "source_kind"),
			SourceID:   intField(edge, "source_id"),
			SourceName: stringField(edge, "source_name"),
			FieldName:  stringField(edge, "field_name"),
			EffectName: stringField(edge, "effect_name"),
		})
	}
	return inbound, nil
}
