package techsurface

import "testing"

func certifiedShellManifest(t *testing.T) (Manifest, Profile) {
	t.Helper()
	profile, ok := ProfileByID("p0_prod_nsb0_bba0_mtg0_fec641fb")
	if !ok {
		t.Fatal("certified P0 profile missing from the catalogue")
	}
	return Manifest{
		SchemaVersion:     SchemaVersion,
		Profile:           profile.ID,
		BaselineSHA256:    profile.BaselineSHA256,
		CompatibilityMode: "all_players",
		Branches:          []Branch{},
	}, profile
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestEmptyBranchManifestWithNoProjectTechnologiesIsNotBlocked(t *testing.T) {
	manifest, _ := certifiedShellManifest(t)

	findings := Validate(manifest, nil, false)
	if hasFinding(findings, "tech_surface_branch_missing") {
		t.Fatalf("an author with zero project-owned technologies must not be blocked: %#v", findings)
	}
	if len(findings) != 0 {
		t.Fatalf("nothing to validate should mean no findings at all: %#v", findings)
	}

	imported := []Technology{{ID: "infantry_weapons", Source: "imported", Folder: "infantry_folder"}}
	if findings := Validate(manifest, imported, false); hasFinding(findings, "tech_surface_branch_missing") {
		t.Fatalf("imported technologies are not project-owned roots: %#v", findings)
	}
}

func TestEmptyBranchManifestStillBlocksAProjectOwnedRoot(t *testing.T) {
	manifest, profile := certifiedShellManifest(t)
	folder := ""
	for name := range profile.Folders {
		if folder == "" || name < folder {
			folder = name
		}
	}
	if folder == "" {
		t.Fatal("certified profile exposes no folders")
	}

	techs := []Technology{{ID: "MOD_rifle", Source: "created", Folder: folder}}
	findings := Validate(manifest, techs, false)
	if !hasFinding(findings, "tech_surface_branch_missing") {
		t.Fatalf("a project-owned root in a certified folder must still be blocked: %#v", findings)
	}
	if !hasFinding(findings, "tech_surface_branch_member_missing") {
		t.Fatalf("the author still needs the per-technology guidance: %#v", findings)
	}
}

func TestEmptyBranchManifestIgnoresTechnologiesOutsideCertifiedFolders(t *testing.T) {
	manifest, profile := certifiedShellManifest(t)
	outside := "a_folder_no_certified_profile_owns"
	if _, clash := profile.Folders[outside]; clash {
		t.Fatalf("fixture folder %q unexpectedly belongs to the profile", outside)
	}

	techs := []Technology{{ID: "MOD_rifle", Source: "created", Folder: outside}}
	if findings := Validate(manifest, techs, false); hasFinding(findings, "tech_surface_branch_missing") {
		t.Fatalf("technologies outside certified folders are none of the surface's business: %#v", findings)
	}
}
