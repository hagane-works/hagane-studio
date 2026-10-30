package techlifecycle

import (
	"fmt"
	"sort"
	"strings"
)

const (
	BlockerChildTechnology   = "child_technology"
	BlockerEquipmentUnlock   = "equipment_unlock"
	BlockerExternalReference = "external_reference"
)

type DeleteBlocker struct {
	Kind       string `json:"kind"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
	Field      string `json:"field"`
	TargetID   string `json:"target_id"`
	Detail     string `json:"detail"`
}

type DeleteBlockedError struct {
	Code     string          `json:"code"`
	BranchID string          `json:"branch_id"`
	Blockers []DeleteBlocker `json:"blockers"`
}

func (e DeleteBlockedError) Error() string {
	return fmt.Sprintf("%s: technology branch %q has %d blocking reference(s)", e.Code, e.BranchID, len(e.Blockers))
}

type DeletePlan struct {
	ProjectID            int64  `json:"project_id"`
	BranchID             string `json:"branch_id"`
	TechnologyDatabaseID int64  `json:"technology_db_id"`
	RootTechnologyID     string `json:"root_technology_id"`
}

func blockerRank(kind string) int {
	switch kind {
	case BlockerChildTechnology:
		return 0
	case BlockerEquipmentUnlock:
		return 1
	case BlockerExternalReference:
		return 2
	default:
		return 3
	}
}

func sortBlockers(blockers []DeleteBlocker) {
	sort.Slice(blockers, func(i, j int) bool {
		left, right := blockers[i], blockers[j]
		if blockerRank(left.Kind) != blockerRank(right.Kind) {
			return blockerRank(left.Kind) < blockerRank(right.Kind)
		}
		if left.SourceKind != right.SourceKind {
			return left.SourceKind < right.SourceKind
		}
		if left.SourceID != right.SourceID {
			return left.SourceID < right.SourceID
		}
		if left.Field != right.Field {
			return left.Field < right.Field
		}
		return left.TargetID < right.TargetID
	})
}

func deleteBlockers(state State, branch Branch) ([]DeleteBlocker, error) {
	if err := validateBranchMembers(state, branch); err != nil {
		return nil, err
	}
	members := map[string]bool{}
	for _, memberID := range branch.TechnologyIDs {
		members[strings.TrimSpace(memberID)] = true
	}
	blockers := []DeleteBlocker{}
	for _, childID := range branch.TechnologyIDs[1:] {
		childID = strings.TrimSpace(childID)
		blockers = append(blockers, DeleteBlocker{
			Kind: BlockerChildTechnology, SourceKind: "technology", SourceID: childID,
			Field: "technology_ids", TargetID: branch.RootTechnologyID,
			Detail: "Remove child technologies explicitly before deleting their controlled branch.",
		})
	}
	for _, memberID := range branch.TechnologyIDs {
		technology, err := technologyByID(state, branch.ProjectID, memberID)
		if err != nil {
			return nil, err
		}
		for _, equipmentID := range technology.EnableEquipments {
			equipmentID = strings.TrimSpace(equipmentID)
			if equipmentID == "" {
				continue
			}
			blockers = append(blockers, DeleteBlocker{
				Kind: BlockerEquipmentUnlock, SourceKind: "technology", SourceID: technology.TechID,
				Field: "enable_equipments", TargetID: equipmentID,
				Detail: "Clear the technology-to-equipment unlock explicitly before deleting the branch.",
			})
		}
	}
	for _, reference := range state.ExternalReferences {
		if reference.ProjectID != branch.ProjectID || !members[strings.TrimSpace(reference.TargetTechnologyID)] {
			continue
		}
		blockers = append(blockers, DeleteBlocker{
			Kind: BlockerExternalReference, SourceKind: strings.TrimSpace(reference.SourceKind),
			SourceID: strings.TrimSpace(reference.SourceID), Field: strings.TrimSpace(reference.Field),
			TargetID: strings.TrimSpace(reference.TargetTechnologyID),
			Detail:   "Remove or retarget the external technology reference explicitly before deleting the branch.",
		})
	}
	sortBlockers(blockers)
	return blockers, nil
}

func Delete(state State, projectID int64, branchID string) (DeletePlan, error) {
	branch, err := findBranch(state, projectID, branchID)
	if err != nil {
		return DeletePlan{}, err
	}
	blockers, err := deleteBlockers(state, branch)
	if err != nil {
		return DeletePlan{}, err
	}
	if len(blockers) > 0 {
		return DeletePlan{}, DeleteBlockedError{Code: CodeDeleteBlocked, BranchID: strings.TrimSpace(branch.ID), Blockers: blockers}
	}
	root, err := technologyByID(state, projectID, branch.RootTechnologyID)
	if err != nil {
		return DeletePlan{}, err
	}
	return DeletePlan{
		ProjectID: projectID, BranchID: strings.TrimSpace(branch.ID),
		TechnologyDatabaseID: root.DatabaseID, RootTechnologyID: root.TechID,
	}, nil
}
