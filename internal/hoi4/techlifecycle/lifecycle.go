package techlifecycle

import (
	"sort"
	"strings"
)

func projectExists(state State, projectID int64) bool {
	for _, candidate := range state.Projects {
		if candidate == projectID {
			return true
		}
	}
	return false
}

func validateProject(state State, projectID int64) error {
	if projectID <= 0 {
		return contractError(CodeProjectInvalid, "project_id", "Technology branch project_id must be a positive integer.")
	}
	if !projectExists(state, projectID) {
		return contractError(CodeProjectNotFound, "project_id", "Technology branch project does not exist.")
	}
	return nil
}

func findBranch(state State, projectID int64, branchID string) (Branch, error) {
	if err := validateProject(state, projectID); err != nil {
		return Branch{}, err
	}
	branchID = strings.TrimSpace(branchID)
	if branchID == "" {
		return Branch{}, contractError(CodeBranchIdentityInvalid, "id", "Technology branch id is required.")
	}
	matches := []Branch{}
	foreign := false
	for _, branch := range state.Branches {
		if strings.TrimSpace(branch.ID) != branchID {
			continue
		}
		if branch.ProjectID == projectID {
			matches = append(matches, branch)
		} else {
			foreign = true
		}
	}
	if len(matches) > 1 {
		return Branch{}, contractError(CodeBranchIdentityAmbiguous, "id", "Technology branch identity is duplicated inside the project.")
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if foreign {
		return Branch{}, contractError(CodeBranchProjectMismatch, "id", "Technology branch belongs to a different project.")
	}
	return Branch{}, contractError(CodeBranchNotFound, "id", "Technology branch does not exist.")
}

func technologyByID(state State, projectID int64, technologyID string) (Technology, error) {
	technologyID = strings.TrimSpace(technologyID)
	matches := []Technology{}
	foreign := false
	for _, technology := range state.Technologies {
		if strings.TrimSpace(technology.TechID) != technologyID {
			continue
		}
		if technology.ProjectID == projectID {
			matches = append(matches, technology)
		} else {
			foreign = true
		}
	}
	if len(matches) > 1 {
		return Technology{}, contractError(CodeBranchIdentityAmbiguous, "root_technology_id", "Technology identity is duplicated inside the project.")
	}
	if len(matches) == 1 {
		return normalizedTechnology(matches[0]), nil
	}
	if foreign {
		return Technology{}, contractError(CodeBranchMemberProject, "root_technology_id", "Branch technology belongs to a different project.")
	}
	return Technology{}, contractError(CodeBranchMemberMissing, "root_technology_id", "Branch technology does not exist.")
}

func validateBranchMembers(state State, branch Branch) error {
	branch.ID = strings.TrimSpace(branch.ID)
	branch.RootTechnologyID = strings.TrimSpace(branch.RootTechnologyID)
	if branch.ProjectID <= 0 || branch.ID == "" || branch.RootTechnologyID == "" {
		return contractError(CodeBranchIdentityInvalid, "id", "Technology branch has an incomplete project, branch, or root identity.")
	}
	if len(branch.TechnologyIDs) == 0 || strings.TrimSpace(branch.TechnologyIDs[0]) != branch.RootTechnologyID {
		return contractError(CodeBranchIdentityInvalid, "technology_ids", "Technology branch root must be its first ordered member.")
	}
	seen := map[string]bool{}
	for _, memberID := range branch.TechnologyIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" || seen[memberID] {
			return contractError(CodeBranchIdentityInvalid, "technology_ids", "Technology branch members must be non-empty and unique.")
		}
		seen[memberID] = true
		if _, err := technologyByID(state, branch.ProjectID, memberID); err != nil {
			return err
		}
	}
	return nil
}

func viewForBranch(state State, branch Branch) (BranchView, error) {
	if err := validateBranchMembers(state, branch); err != nil {
		return BranchView{}, err
	}
	root, err := technologyByID(state, branch.ProjectID, branch.RootTechnologyID)
	if err != nil {
		return BranchView{}, err
	}
	gridboxID := strings.TrimSpace(branch.GridboxID)
	if gridboxID == "" {
		gridboxID = root.TechID + "_tree"
	}
	return BranchView{
		ProjectID: branch.ProjectID, ID: strings.TrimSpace(branch.ID),
		ProfileID: strings.TrimSpace(branch.ProfileID), BaselineSHA256: strings.TrimSpace(branch.BaselineSHA256),
		SchemaVersion: branch.SchemaVersion, Surface: strings.TrimSpace(branch.Surface),
		Folder: strings.TrimSpace(branch.Folder), Slot: strings.TrimSpace(branch.Slot), GridboxID: gridboxID,
		RootTechnologyID: root.TechID, TechnologyIDs: cloneStrings(branch.TechnologyIDs),
		TechnologyDatabaseID: root.DatabaseID, Source: root.Source,
		Name: root.Name, NameEn: root.NameEn, NameZh: root.NameZh,
		DescEn: root.DescEn, DescZh: root.DescZh,
		DisplayNameLocs: cloneMap(root.DisplayNameLocs), DescriptionLocs: cloneMap(root.DescriptionLocs),
		EnableEquipments: cloneStrings(root.EnableEquipments),
		TechLine:         root.TechLine, StartYear: root.StartYear,
		PositionX: root.PositionX, PositionY: root.PositionY,
		FolderPlacements: clonePlacements(root.FolderPlacements),
		Path:             clonePaths(root.Path),
	}, nil
}

func List(state State, projectID int64) ([]BranchView, error) {
	if err := validateProject(state, projectID); err != nil {
		return nil, err
	}
	branches := make([]Branch, 0)
	seenIDs := map[string]bool{}
	for _, branch := range state.Branches {
		if branch.ProjectID == projectID {
			branchID := strings.TrimSpace(branch.ID)
			if branchID == "" {
				return nil, contractError(CodeBranchIdentityInvalid, "id", "Technology branch id is required.")
			}
			if seenIDs[branchID] {
				return nil, contractError(CodeBranchIdentityAmbiguous, "id", "Technology branch identity is duplicated inside the project.")
			}
			seenIDs[branchID] = true
			branches = append(branches, branch)
		}
	}
	sort.Slice(branches, func(i, j int) bool {
		return strings.TrimSpace(branches[i].ID) < strings.TrimSpace(branches[j].ID)
	})
	result := make([]BranchView, 0, len(branches))
	for _, branch := range branches {
		view, err := viewForBranch(state, branch)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func Get(state State, projectID int64, branchID string) (BranchView, error) {
	branch, err := findBranch(state, projectID, branchID)
	if err != nil {
		return BranchView{}, err
	}
	return viewForBranch(state, branch)
}

func validateUpdate(request UpdateRequest) error {
	if len(request.ReadOnlyFields) > 0 {
		fields := append([]string(nil), request.ReadOnlyFields...)
		sort.Strings(fields)
		return contractError(CodeImmutableField, fields[0], "Technology branch profile-owned fields cannot be changed through partial update.")
	}
	if len(request.UnknownFields) > 0 {
		fields := append([]string(nil), request.UnknownFields...)
		sort.Strings(fields)
		return contractError(CodeUnknownField, fields[0], "Technology branch update contains an unsupported field.")
	}
	return nil
}

func stringField(field Field[string], current string) string {
	if !field.Set {
		return current
	}
	if field.Value == nil {
		return ""
	}
	return *field.Value
}

func mapField(field Field[map[string]string], current map[string]string) map[string]string {
	if !field.Set {
		return cloneMap(current)
	}
	if field.Value == nil {
		return map[string]string{}
	}
	return cloneMap(*field.Value)
}

func controlledEquipments(state State, projectID int64, field Field[[]string], current []string) ([]string, error) {
	if !field.Set {
		return cloneStrings(current), nil
	}
	if field.Value == nil {
		return []string{}, nil
	}
	result := make([]string, 0, len(*field.Value))
	seen := map[string]bool{}
	for _, rawID := range *field.Value {
		equipmentID := strings.TrimSpace(rawID)
		if equipmentID == "" || seen[equipmentID] {
			return nil, contractError(CodeEquipmentInvalid, "enable_equipments", "Equipment unlock ids must be non-empty and unique.")
		}
		seen[equipmentID] = true
		sameProject := false
		foreign := false
		for _, equipment := range state.Equipments {
			if strings.TrimSpace(equipment.EquipmentID) != equipmentID {
				continue
			}
			if equipment.ProjectID == projectID {
				sameProject = true
			} else {
				foreign = true
			}
		}
		if !sameProject && foreign {
			return nil, contractError(CodeEquipmentProjectMismatch, "enable_equipments", "Equipment unlock belongs to a different project.")
		}
		if !sameProject {
			return nil, contractError(CodeEquipmentMissing, "enable_equipments", "Equipment unlock does not exist in this project.")
		}
		result = append(result, equipmentID)
	}
	return result, nil
}

func mutableFromView(view BranchView) MutableTechnology {
	return MutableTechnology{
		Name: view.Name, NameEn: view.NameEn, NameZh: view.NameZh,
		DescEn: view.DescEn, DescZh: view.DescZh,
		DisplayNameLocs: cloneMap(view.DisplayNameLocs), DescriptionLocs: cloneMap(view.DescriptionLocs),
		EnableEquipments: cloneStrings(view.EnableEquipments),
	}
}

func changedFields(request UpdateRequest) []string {
	fields := []string{}
	for name, set := range map[string]bool{
		"name": request.Name.Set, "name_en": request.NameEn.Set, "name_zh": request.NameZh.Set,
		"desc_en": request.DescEn.Set, "desc_zh": request.DescZh.Set,
		"display_name_locs": request.DisplayNameLocs.Set, "description_locs": request.DescriptionLocs.Set,
		"enable_equipments": request.EnableEquipments.Set,
	} {
		if set {
			fields = append(fields, name)
		}
	}
	sort.Strings(fields)
	return fields
}

func Update(state State, projectID int64, branchID string, request UpdateRequest) (UpdatePlan, error) {
	if err := validateUpdate(request); err != nil {
		return UpdatePlan{}, err
	}
	view, err := Get(state, projectID, branchID)
	if err != nil {
		return UpdatePlan{}, err
	}
	equipments, err := controlledEquipments(state, projectID, request.EnableEquipments, view.EnableEquipments)
	if err != nil {
		return UpdatePlan{}, err
	}
	view.Name = stringField(request.Name, view.Name)
	view.NameEn = stringField(request.NameEn, view.NameEn)
	view.NameZh = stringField(request.NameZh, view.NameZh)
	view.DescEn = stringField(request.DescEn, view.DescEn)
	view.DescZh = stringField(request.DescZh, view.DescZh)
	view.DisplayNameLocs = mapField(request.DisplayNameLocs, view.DisplayNameLocs)
	view.DescriptionLocs = mapField(request.DescriptionLocs, view.DescriptionLocs)
	view.EnableEquipments = equipments
	return UpdatePlan{
		ProjectID: projectID, BranchID: view.ID,
		TechnologyDatabaseID: view.TechnologyDatabaseID, RootTechnologyID: view.RootTechnologyID,
		ChangedFields: changedFields(request), Mutable: mutableFromView(view), View: view,
	}, nil
}
