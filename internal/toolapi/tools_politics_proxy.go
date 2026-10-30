package toolapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type getPoliticsReferencesIn struct {
	ProjectID int64 `json:"project_id"`
}

type discardIdeologyConflictIn struct {
	ProjectID              int64  `json:"project_id"`
	ScriptID               string `json:"script_id" jsonschema:"script id from get_politics_references.conflicts"`
	AcknowledgedUsageCount int    `json:"acknowledged_usage_count" jsonschema:"exact current usage_count explicitly acknowledged by the user"`
}

type politicsReferenceConflicts struct {
	Conflicts []struct {
		ScriptID   string `json:"script_id"`
		Reason     string `json:"reason"`
		UsageCount int    `json:"usage_count"`
	} `json:"conflicts"`
}

type previewDeleteIdeologyIn struct {
	ProjectID int64  `json:"project_id"`
	Kind      string `json:"kind" jsonschema:"ideology target kind: root or subtype"`
	ScriptID  string `json:"script_id" jsonschema:"stable HOI4 ideology script id"`
}

type applyDeleteIdeologyIn struct {
	ProjectID    int64  `json:"project_id"`
	PreviewToken string `json:"preview_token" jsonschema:"single-use token returned by preview_delete_ideology"`
}

type createIdeologyRootIn struct {
	ProjectID int64             `json:"project_id"`
	ScriptID  string            `json:"script_id" jsonschema:"stable project-managed ROOT script id; immutable after creation"`
	Body      string            `json:"body" jsonschema:"ROOT definition body, including types and color"`
	Name      map[string]string `json:"name,omitempty" jsonschema:"visible names keyed by content locale; at least one is required"`
	Desc      map[string]string `json:"desc,omitempty" jsonschema:"optional descriptions keyed by content locale"`
}

type createIdeologySubtypeIn struct {
	ProjectID      int64             `json:"project_id"`
	ScriptID       string            `json:"script_id" jsonschema:"stable subtype script id; immutable after creation"`
	RootID         string            `json:"root_id" jsonschema:"existing ROOT script id"`
	Name           map[string]string `json:"name,omitempty" jsonschema:"visible names keyed by content locale; at least one is required"`
	Desc           map[string]string `json:"desc,omitempty" jsonschema:"optional descriptions keyed by content locale"`
	RandomDisabled bool              `json:"random_disabled,omitempty" jsonschema:"exclude this subtype from random ideology selection"`
}

type updateIdeologyPatchIn struct {
	Name           *map[string]string `json:"name,omitempty"`
	Desc           *map[string]string `json:"desc,omitempty"`
	RandomDisabled *bool              `json:"random_disabled,omitempty" jsonschema:"subtype only"`
}

type updateIdeologyIn struct {
	ProjectID int64                 `json:"project_id"`
	Kind      string                `json:"kind" jsonschema:"definition kind: root or subtype"`
	ScriptID  string                `json:"script_id" jsonschema:"stable existing script id; cannot be renamed"`
	Patch     updateIdeologyPatchIn `json:"patch"`
}

func registerPoliticsProxyTools(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "create_ideology_root",
		Description: "Create one project-managed ideology ROOT with a stable script ID, engine definition body, and visible localisation. Source ownership and takeover are program-controlled. If this returns politics_registry_invalid, first read get_politics_references.conflicts, then use discard_ideology_conflict only after the required explicit user confirmation.",
	}, func(ctx context.Context, _ *CallToolRequest, in createIdeologyRootIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{"script_id": in.ScriptID, "body": in.Body, "name": in.Name}
		if in.Desc != nil {
			body["desc"] = in.Desc
		}
		out, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/politics/roots", in.ProjectID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "create_ideology_subtype",
		Description: "Create one project-managed ideology subtype under an existing ROOT. The stable script ID cannot be renamed after creation and inherited source ownership is enforced by the project service. A project with its own file under common/ideologies (or replace_path=common/ideologies) owns its ideologies: the game loads that file, so this returns politics_source_raw_owned and the subtype must be added to that raw file with update_raw_file instead. If this returns politics_registry_invalid, first read get_politics_references.conflicts, then use discard_ideology_conflict only after the required explicit user confirmation.",
	}, func(ctx context.Context, _ *CallToolRequest, in createIdeologySubtypeIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{
			"script_id": in.ScriptID, "root_id": in.RootID, "name": in.Name,
			"random_disabled": in.RandomDisabled,
		}
		if in.Desc != nil {
			body["desc"] = in.Desc
		}
		out, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/politics/subtypes", in.ProjectID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "update_ideology",
		Description: "Update author-owned localisation or subtype random-selection settings. kind + script_id address the stable definition; patch cannot rename it, move it to another ROOT, or change source ownership. If this returns politics_registry_invalid, first read get_politics_references.conflicts, then use discard_ideology_conflict only after the required explicit user confirmation.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateIdeologyIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{}
		if in.Patch.Name != nil {
			body["name"] = *in.Patch.Name
		}
		if in.Patch.Desc != nil {
			body["desc"] = *in.Patch.Desc
		}
		if in.Patch.RandomDisabled != nil {
			body["random_disabled"] = *in.Patch.RandomDisabled
		}
		out, err := proxyAuthedCall(ctx, proxy, "PATCH",
			fmt.Sprintf("/api/projects/%d/politics/definitions/%s/%s", in.ProjectID,
				url.PathEscape(in.Kind), url.PathEscape(in.ScriptID)), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "get_politics_references",
		Description: "Read the project's canonical ideology ownership, reference impacts, custom-ROOT compatibility findings, and conflicts. conflicts contains invalid registry definitions that can block every ideology write and supplies the script_id, reason, and usage_count required for an explicitly confirmed discard_ideology_conflict call.",
	}, func(ctx context.Context, _ *CallToolRequest, in getPoliticsReferencesIn) (*CallToolResult, map[string]any, error) {
		out, err := proxyAuthedCall(ctx, proxy, "GET",
			fmt.Sprintf("/api/projects/%d/politics/references", in.ProjectID), nil)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "discard_ideology_conflict",
		Description: "Discard exactly one invalid ideology definition listed in get_politics_references.conflicts. While such a conflict exists, every ideology write in the project can fail validation. Discarding removes the definition from the registry, but the usage_count characters keep this script id in their ideology fields and must be repaired separately. Before calling, tell the user the exact script_id, reason, and usage_count and obtain explicit consent; acknowledged_usage_count must equal the freshly read count.",
	}, func(ctx context.Context, _ *CallToolRequest, in discardIdeologyConflictIn) (*CallToolResult, map[string]any, error) {
		scriptID := strings.TrimSpace(in.ScriptID)
		references, err := proxyAuthedCall(ctx, proxy, "GET",
			fmt.Sprintf("/api/projects/%d/politics/references", in.ProjectID), nil)
		if err != nil {
			return nil, nil, err
		}

		encoded, err := json.Marshal(references)
		if err != nil {
			return nil, nil, errors.New("The politics references response could not be inspected. Call get_politics_references and retry.")
		}
		var snapshot politicsReferenceConflicts
		if err := json.Unmarshal(encoded, &snapshot); err != nil {
			return nil, nil, errors.New("The politics references response could not be inspected. Call get_politics_references and retry.")
		}
		for _, conflict := range snapshot.Conflicts {
			if scriptID == "" || conflict.ScriptID != scriptID {
				continue
			}
			if conflict.UsageCount != in.AcknowledgedUsageCount {
				return nil, nil, fmt.Errorf("Ideology conflict %q currently has usage_count %d, not the acknowledged %d. Call get_politics_references again, tell the user the current script_id, reason, and usage_count, and obtain explicit confirmation before retrying.", scriptID, conflict.UsageCount, in.AcknowledgedUsageCount)
			}
			out, err := proxyAuthedCall(ctx, proxy, "DELETE",
				fmt.Sprintf("/api/projects/%d/politics/conflicts/%s", in.ProjectID, url.PathEscape(scriptID)), nil)
			return nil, out, err
		}
		return nil, nil, fmt.Errorf("Ideology conflict %q is not in the current conflicts list. Call get_politics_references before asking the user to confirm a discard.", scriptID)
	})

	AddTool(s, &Tool{
		Name:        "preview_delete_ideology",
		Description: "Preview deletion of one project-managed ideology ROOT or subtype. Returns exact reference impacts and a single-use preview token; never rewrites references automatically.",
	}, func(ctx context.Context, _ *CallToolRequest, in previewDeleteIdeologyIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{"kind": in.Kind, "script_id": in.ScriptID}
		out, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/politics/delete-preview", in.ProjectID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "apply_delete_ideology",
		Description: "Apply an ideology deletion authorized by preview_delete_ideology. Accepts only its single-use preview_token; blocked references remain blocked and there is no force/cascade/rewrite/takeover mode.",
	}, func(ctx context.Context, _ *CallToolRequest, in applyDeleteIdeologyIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{"preview_token": in.PreviewToken}
		out, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/politics/delete-apply", in.ProjectID), body)
		return nil, out, err
	})
}
