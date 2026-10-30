package toolapi

import (
	"context"
	"fmt"
	"strings"
)

type editStateIn struct {
	ProjectID               int64             `json:"project_id"`
	StateDBID               int64             `json:"state_db_id" jsonschema:"the SURROGATE PK (StateModel.id) from get_states -- NOT the HOI4 state_id"`
	Owner                   *string           `json:"owner,omitempty"`
	Controller              *string           `json:"controller,omitempty"`
	StateCategory           *string           `json:"state_category,omitempty"`
	Manpower                *int              `json:"manpower,omitempty"`
	Resources               map[string]any    `json:"resources,omitempty"`
	AddCoreOf               []string          `json:"add_core_of,omitempty" jsonschema:"REPLACES the whole list -- send the full set"`
	AddClaimBy              []string          `json:"add_claim_by,omitempty" jsonschema:"REPLACES the whole list -- send the full set"`
	RemoveCoreOf            []string          `json:"remove_core_of,omitempty" jsonschema:"REPLACES the whole list -- send the full set"`
	VictoryPoints           []map[string]any  `json:"victory_points,omitempty" jsonschema:"REPLACES the whole list -- send the full set"`
	SetProvinceController   []map[string]any  `json:"set_province_controller,omitempty" jsonschema:"REPLACES the whole list -- send the full set"`
	Buildings               map[string]any    `json:"buildings,omitempty"`
	ProvinceBuildings       map[string]any    `json:"province_buildings,omitempty" jsonschema:"{provinceId: {building: level}} for the SIMPLE per-province blocks"`
	LocalSupplies           *float64          `json:"local_supplies,omitempty"`
	BuildingsMaxLevelFactor *float64          `json:"buildings_max_level_factor,omitempty"`
	Impassable              *bool             `json:"impassable,omitempty"`
	ImpassableIgnoredLinks  *string           `json:"impassable_ignored_links,omitempty" jsonschema:"raw inner list of neighbor state ids, e.g. '1024' or '1024 1025'"`
	ForceLinkOwnershipTo    *int64            `json:"force_link_ownership_to,omitempty" jsonschema:"state_id of the NEIGHBOUR state this impassable (wasteland) state follows for ownership; all 21 vanilla impassable states carry it -- set it whenever you set impassable=true"`
	DisplayNameEn           *string           `json:"display_name_en,omitempty" jsonschema:"rename override; merges into the existing display_name_locs (every other language keeps its value); pass '' to clear"`
	DisplayNameZh           *string           `json:"display_name_zh,omitempty" jsonschema:"rename override; merges into the existing display_name_locs (every other language keeps its value); pass '' to clear"`
	DisplayNameLocs         map[string]string `json:"display_name_locs,omitempty" jsonschema:"rename override for ANY language, {lang: name} -- e.g. {\"ru\":\"Корсика\"}. Merges per language into the existing map; languages you omit keep their value; pass '' for one to clear it. display_name_en/zh are the two-language shorthand for this same field."`
}

type editStateOut struct {
	StateDBID   int64             `json:"state_db_id"`
	StateID     int64             `json:"state_id"`
	Name        string            `json:"name"`
	DisplayName map[string]string `json:"display_name"`
	Owner       string            `json:"owner"`
	Controller  string            `json:"controller"`
	Source      string            `json:"source"`
}

func registerEditState(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "edit_state",
		Description: "Edit an EXISTING state's territorial content (owner, cores, manpower, ...). " +
			"EDIT-ONLY: cannot create or delete a state. state_db_id is the SURROGATE PK from " +
			"get_states, NOT the HOI4 state_id. Only the fields you supply are changed (patch " +
			"semantics); LIST fields REPLACE the whole list. RENAME = display_name_locs (any " +
			"language) or the display_name_en/zh shorthand -- a pure localisation override (the " +
			"state file's name loc key and provinces are never touched). " +
			"Editing flips source imported->created so the change reaches the default export.",
	}, func(ctx context.Context, _ *CallToolRequest, in editStateIn) (*CallToolResult, editStateOut, error) {
		body := map[string]any{}
		if in.Owner != nil {
			body["owner"] = *in.Owner
		}
		if in.Controller != nil {
			body["controller"] = *in.Controller
		}
		if in.StateCategory != nil {
			body["state_category"] = *in.StateCategory
		}
		if in.Manpower != nil {
			body["manpower"] = *in.Manpower
		}
		if in.Resources != nil {
			body["resources"] = in.Resources
		}
		if in.AddCoreOf != nil {
			body["add_core_of"] = in.AddCoreOf
		}
		if in.AddClaimBy != nil {
			body["add_claim_by"] = in.AddClaimBy
		}
		if in.RemoveCoreOf != nil {
			body["remove_core_of"] = in.RemoveCoreOf
		}
		if in.VictoryPoints != nil {
			body["victory_points"] = in.VictoryPoints
		}
		if in.SetProvinceController != nil {
			body["set_province_controller"] = in.SetProvinceController
		}
		if in.Buildings != nil {
			body["buildings"] = in.Buildings
		}
		if in.ProvinceBuildings != nil {
			body["province_buildings"] = in.ProvinceBuildings
		}
		if in.LocalSupplies != nil {
			body["local_supplies"] = *in.LocalSupplies
		}
		if in.BuildingsMaxLevelFactor != nil {
			body["buildings_max_level_factor"] = *in.BuildingsMaxLevelFactor
		}
		if in.Impassable != nil {
			body["impassable"] = *in.Impassable
		}
		if in.ImpassableIgnoredLinks != nil {
			body["impassable_ignored_links"] = *in.ImpassableIgnoredLinks
		}

		if in.ForceLinkOwnershipTo != nil {
			body["force_link_ownership_to"] = *in.ForceLinkOwnershipTo
		}

		statePath := fmt.Sprintf("/api/projects/%d/states/%d", in.ProjectID, in.StateDBID)

		if in.DisplayNameEn != nil || in.DisplayNameZh != nil || len(in.DisplayNameLocs) > 0 {
			current, err := proxyAuthedCall(ctx, proxy, "GET", statePath, nil)
			if err != nil {
				return nil, editStateOut{}, err
			}
			locs := map[string]any{}

			if existing, ok := current["display_name_locs"].(map[string]any); ok {
				for k, v := range existing {
					locs[k] = v
				}
			}

			for lang, text := range in.DisplayNameLocs {
				if lang = strings.ToLower(strings.TrimSpace(lang)); lang != "" {
					locs[lang] = text
				}
			}
			if in.DisplayNameEn != nil {
				locs["en"] = *in.DisplayNameEn
			}
			if in.DisplayNameZh != nil {
				locs["zh"] = *in.DisplayNameZh
			}
			body["display_name_locs"] = locs
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT", statePath, body)
		if err != nil {
			return nil, editStateOut{}, err
		}

		return nil, editStateOut{
			StateDBID:   int64Field(resp, "id"),
			StateID:     int64Field(resp, "state_id"),
			Name:        stringField(resp, "name"),
			DisplayName: orEmptyMap(stringMapField(resp, "display_name")),
			Owner:       stringField(resp, "owner"),
			Controller:  stringField(resp, "controller"),
			Source:      stringField(resp, "source"),
		}, nil
	})
}

type seedStatesFromVanillaIn struct {
	ProjectID int64   `json:"project_id"`
	AllStates bool    `json:"all_states,omitempty" jsonschema:"seed every vanilla state; wins over state_ids when both given"`
	StateIDs  []int64 `json:"state_ids,omitempty" jsonschema:"HOI4 state ids to seed (matched by the state-file id prefix)"`
}

type seedStatesFromVanillaOut struct {
	Seeded   int      `json:"seeded"`
	Skipped  int      `json:"skipped"`
	Missing  []int64  `json:"missing"`
	Warnings []string `json:"warnings"`
}

func registerSeedStatesFromVanilla(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "seed_states_from_vanilla",
		Description: "Bootstrap vanilla states into a fresh standalone project (idempotent). A " +
			"brand-new standalone project starts with ZERO states, so edit_state has nothing to " +
			"target. Pass all_states=true to seed every vanilla state, OR state_ids=[...] to seed " +
			"only those. all_states wins when both are given. Idempotent -- a state already present " +
			"is SKIPPED, never overwritten. Without the game folder set this surfaces as a clean " +
			"error.",
	}, func(ctx context.Context, _ *CallToolRequest, in seedStatesFromVanillaIn) (*CallToolResult, seedStatesFromVanillaOut, error) {
		body := map[string]any{
			"all":       in.AllStates,
			"state_ids": in.StateIDs,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/states/seed-from-vanilla", in.ProjectID), body)
		if err != nil {
			return nil, seedStatesFromVanillaOut{}, err
		}

		out := seedStatesFromVanillaOut{
			Seeded:   int(int64Field(resp, "seeded")),
			Skipped:  int(int64Field(resp, "skipped")),
			Warnings: stringSliceField(resp, "warnings"),
		}
		if missing, ok := resp["missing"].([]any); ok {
			out.Missing = make([]int64, 0, len(missing))
			for _, m := range missing {
				if f, ok := m.(float64); ok {
					out.Missing = append(out.Missing, int64(f))
				}
			}
		}
		return nil, out, nil
	})
}
