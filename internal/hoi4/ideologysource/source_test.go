package ideologysource

import (
	"bytes"
	"strings"
	"testing"
)

func vanillaLayer(path string, source []byte) Layer {
	return Layer{SourceKind: SourceVanilla, LoadOrder: 0, Files: []SourceFile{{Path: path, Bytes: source}}}
}

func basicSource(eol string, bom bool) []byte {
	text := strings.Join([]string{
		"ideologies = {",
		"\tdemocratic = {",
		"\t\t# { comment braces are inert }",
		"\t\ttypes = {",
		"\t\t\tconservatism = {",
		"\t\t\t\tcan_be_randomly_selected = yes",
		"\t\t\t\tnote = \"quoted { braces } and # text\"",
		"\t\t\t}",
		"\t\t}",
		"\t\tcolor = { 0 0 255 }",
		"\t}",
		"}",
		"# exact tail",
		"",
	}, eol)
	if bom {
		return append([]byte{0xef, 0xbb, 0xbf}, []byte(text)...)
	}
	return []byte(text)
}

func lockOne(t *testing.T, layers []Layer, subtype Subtype) Manifest {
	t.Helper()
	discovery, err := Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Lock(discovery, []Subtype{subtype})
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func findingCodes(err error) map[string]bool {
	codes := map[string]bool{}
	if typed, ok := err.(FindingsError); ok {
		for _, item := range typed.Findings {
			codes[item.Code] = true
		}
	}
	return codes
}

func validationCodes(findings []Finding) map[string]bool {
	codes := map[string]bool{}
	for _, item := range findings {
		codes[item.Code] = true
	}
	return codes
}

func TestCompilePreservesBOMCRLFCommentsQuotedBracesAndIsIdempotent(t *testing.T) {
	path := `common\ideologies\00_ideologies.txt`
	source := basicSource("\r\n", true)
	layers := []Layer{vanillaLayer(path, source)}
	subtype := Subtype{ScriptID: "civic_coalition", ParentRootID: "democratic", Names: map[string]string{"EN": "Civic Coalition", "zh": "公民联盟"}, Descriptions: map[string]string{"en": "A project subtype."}}
	manifest := lockOne(t, layers, subtype)

	first, err := Compile(layers, manifest, []Subtype{subtype})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Files) != 1 || first.Files[0].Path != "common/ideologies/00_ideologies.txt" {
		t.Fatalf("files=%+v", first.Files)
	}
	if !bytes.HasPrefix(first.Files[0].Bytes, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("UTF-8 BOM was lost")
	}
	if bytes.Contains(bytes.ReplaceAll(first.Files[0].Bytes, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatal("compiler introduced bare LF into CRLF source")
	}
	for _, want := range []string{"civic_coalition = {", "can_be_randomly_selected = no", `note = "quoted { braces } and # text"`, "# exact tail"} {
		if !bytes.Contains(first.Files[0].Bytes, []byte(want)) {
			t.Fatalf("missing %q:\n%s", want, first.Files[0].Bytes)
		}
	}
	stripped, count, err := stripManagedRegions(first.Files[0].Bytes)
	if err != nil || count != 1 || !bytes.Equal(stripped, source) {
		t.Fatalf("strip count=%d err=%v exact=%t", count, err, bytes.Equal(stripped, source))
	}
	if len(first.Localisations) != 3 || first.Localisations[0].Language != "en" || first.Localisations[0].Key != "civic_coalition" {
		t.Fatalf("loc=%+v", first.Localisations)
	}

	secondLayers := []Layer{vanillaLayer(path, first.Files[0].Bytes)}
	second, err := Compile(secondLayers, manifest, []Subtype{subtype})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Files[0].Bytes, second.Files[0].Bytes) {
		t.Fatal("second compile was not byte-identical")
	}
	third, err := Compile(layers, manifest, []Subtype{subtype})
	if err != nil || !bytes.Equal(first.Files[0].Bytes, third.Files[0].Bytes) {
		t.Fatal("same input was not deterministic")
	}
}

func TestCompileMultipleRootsUsesDescendingOffsetsAndSameOwnerShadow(t *testing.T) {
	source := []byte("ideologies = {\n\talpha = {\n\t\ttypes = {\n\t\t\ta_old = {}\n\t\t}\n\t}\n\tbeta = {\n\t\ttypes = {\n\t\t\tb_old = {}\n\t\t}\n\t}\n}\n")
	layers := []Layer{vanillaLayer("common/ideologies/roots.txt", source)}
	subtypes := []Subtype{{ScriptID: "alpha_new", ParentRootID: "alpha"}, {ScriptID: "beta_new", ParentRootID: "beta"}}
	discovery, err := Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Lock(discovery, subtypes)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(layers, manifest, subtypes)
	if err != nil {
		t.Fatal(err)
	}
	text := string(compiled.Files[0].Bytes)
	if strings.Count(text, string(managedBegin)) != 2 || strings.Index(text, "alpha_new") > strings.Index(text, "beta_new") {
		t.Fatalf("wrong insertion order:\n%s", text)
	}
	stripped, count, err := stripManagedRegions(compiled.Files[0].Bytes)
	if err != nil || count != 2 || !bytes.Equal(stripped, source) {
		t.Fatalf("multi-root source not reversible: count=%d err=%v", count, err)
	}
}

func TestStripManagedRegionsReadsTheLegacyMarker(t *testing.T) {
	source := []byte("ideologies = {\n\tdemocratic = {\n\t\ttypes = {\n\t\t\t# HOI4 Hagane managed subtypes BEGIN\n\t\t\tlegacy = {}\n\t\t\t# HOI4 Hagane managed subtypes END\n\t\t}\n\t}\n}\n")
	want := []byte("ideologies = {\n\tdemocratic = {\n\t\ttypes = {}\n\t}\n}\n")
	got, count, err := stripManagedRegions(source)
	if err != nil || count != 1 || !bytes.Equal(got, want) {
		t.Fatalf("legacy marker strip count=%d err=%v output=%q", count, err, got)
	}
}

func TestLockRejectsMissingAmbiguousReplacedAndInvalidTypesOwners(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		d, err := Discover([]Layer{vanillaLayer("common/ideologies/a.txt", basicSource("\n", false))})
		if err != nil {
			t.Fatal(err)
		}
		_, err = Lock(d, []Subtype{{ScriptID: "x", ParentRootID: "missing"}})
		if !findingCodes(err)["ideology_source_owner_missing"] {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("two owners", func(t *testing.T) {
		source := []byte("ideologies={ democratic={ types={} } }")
		_, err := Discover([]Layer{{SourceKind: SourceVanilla, LoadOrder: 0, Files: []SourceFile{{Path: "common/ideologies/a.txt", Bytes: source}, {Path: "common/ideologies/b.txt", Bytes: source}}}})
		if !findingCodes(err)["ideology_source_owner_ambiguous"] {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("replace path hides vanilla", func(t *testing.T) {
		layers := []Layer{vanillaLayer("common/ideologies/a.txt", basicSource("\n", false)), {SourceKind: SourceDependency, DependencyKey: "base", LoadOrder: 1, ReplacePaths: []string{"./common\\ideologies/"}}}
		d, err := Discover(layers)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Lock(d, []Subtype{{ScriptID: "x", ParentRootID: "democratic"}})
		if !findingCodes(err)["ideology_source_owner_replaced"] {
			t.Fatalf("err=%v", err)
		}
	})
	for name, source := range map[string][]byte{
		"zero types": []byte("ideologies={ democratic={ color={1 2 3} } }"),
		"two types":  []byte("ideologies={ democratic={ types={} types={} } }"),
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Discover([]Layer{vanillaLayer("common/ideologies/a.txt", source)})
			if err != nil {
				t.Fatal(err)
			}
			_, err = Lock(d, []Subtype{{ScriptID: "x", ParentRootID: "democratic"}})
			if !findingCodes(err)["ideology_source_types_ambiguous"] {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestDiscoverRejectsMalformedLayerAndLoadOrderAmbiguity(t *testing.T) {
	_, err := Discover([]Layer{vanillaLayer("common/ideologies/a.txt", []byte(`ideologies={ democratic={ types={ note="unterminated } } }`))})
	if !findingCodes(err)["ideology_source_malformed"] {
		t.Fatalf("malformed err=%v", err)
	}
	base := vanillaLayer("common/ideologies/a.txt", basicSource("\n", false))
	for name, layers := range map[string][]Layer{
		"two dependencies": {base, {SourceKind: SourceDependency, DependencyKey: "a", LoadOrder: 1}, {SourceKind: SourceDependency, DependencyKey: "b", LoadOrder: 2}},
		"same order":       {base, {SourceKind: SourceDependency, DependencyKey: "a", LoadOrder: 0}},
		"wrong order":      {{SourceKind: SourceVanilla, LoadOrder: 2}, {SourceKind: SourceDependency, DependencyKey: "a", LoadOrder: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Discover(layers)
			if !findingCodes(err)[CodeLayerAmbiguous] {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestDependencyOwnerCanBeLockedButSameOutputPathFailsClosed(t *testing.T) {
	vanilla := vanillaLayer("common/ideologies/vanilla.txt", basicSource("\n", false))
	dependencySource := []byte("ideologies={ custom_root={ types={ inherited_type={} } } }")
	dependency := Layer{SourceKind: SourceDependency, DependencyKey: "base_mod", LoadOrder: 1, Files: []SourceFile{{Path: "common/ideologies/base.txt", Bytes: dependencySource}}}
	d, err := Discover([]Layer{vanilla, dependency})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Lock(d, []Subtype{{ScriptID: "project_type", ParentRootID: "custom_root"}})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Entries[0].SourceKind != SourceDependency || manifest.Entries[0].DependencyKey != "base_mod" || manifest.Entries[0].OwnerPath != "common/ideologies/base.txt" {
		t.Fatalf("entry=%+v", manifest.Entries[0])
	}
	dependency.Files[0].Path = "common/ideologies/vanilla.txt"
	_, err = Discover([]Layer{vanilla, dependency})
	if !findingCodes(err)["ideology_source_owner_path_ambiguous"] {
		t.Fatalf("err=%v", err)
	}
}

func TestDuplicateAndExistingScriptIDsAreBlocked(t *testing.T) {
	layers := []Layer{vanillaLayer("common/ideologies/a.txt", basicSource("\n", false))}
	d, err := Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Lock(d, []Subtype{{ScriptID: "conservatism", ParentRootID: "democratic"}, {ScriptID: "duplicate", ParentRootID: "democratic"}, {ScriptID: "duplicate", ParentRootID: "democratic"}})
	codes := findingCodes(err)
	if !codes["ideology_source_script_id_conflict"] || !codes["ideology_source_script_id_duplicate"] {
		t.Fatalf("codes=%v err=%v", codes, err)
	}
}

func TestRelockAgainstAReimportedExportCompilesTheSubtypeOnce(t *testing.T) {
	path := "common/ideologies/00_ideologies.txt"
	source := basicSource("\n", false)
	subtype := Subtype{ScriptID: "civic_coalition", ParentRootID: "democratic"}
	first, err := Compile([]Layer{vanillaLayer(path, source)}, lockOne(t, []Layer{vanillaLayer(path, source)}, subtype), []Subtype{subtype})
	if err != nil {
		t.Fatal(err)
	}

	reimported := append(append([]byte(nil), first.Files[0].Bytes...), []byte("# the mod's own edit\n")...)
	layers := []Layer{vanillaLayer(path, reimported)}
	stale := lockOne(t, []Layer{vanillaLayer(path, source)}, subtype)
	if _, err := Compile(layers, stale, []Subtype{subtype}); err == nil {
		t.Fatal("the old lock must not compile against the changed source")
	}

	relocked, err := Relock(layers, []Subtype{subtype})
	if err != nil {
		t.Fatalf("relock against the re-imported source: %v", err)
	}
	out, err := Compile(layers, relocked, []Subtype{subtype})
	if err != nil {
		t.Fatalf("compile after relock: %v", err)
	}
	body := string(out.Files[0].Bytes)
	if strings.Count(body, "civic_coalition = {") != 1 || !strings.Contains(body, "# the mod's own edit") {
		t.Fatalf("want the subtype once on top of the Mod's own file:\n%s", body)
	}
}

func TestRelockFailsWhenTheRootIsGone(t *testing.T) {
	path := "common/ideologies/00_ideologies.txt"
	noDemocratic := []byte("ideologies = {\n\tfascism = {\n\t\ttypes = { nazism = { } }\n\t}\n}\n")
	_, err := Relock([]Layer{vanillaLayer(path, noDemocratic)}, []Subtype{{ScriptID: "civic_coalition", ParentRootID: "democratic"}})
	if !findingCodes(err)["ideology_source_owner_missing"] {
		t.Fatalf("err=%v", err)
	}
}

func TestSourceAndRootDriftFailClosedAndCertifiedIDsReturnNothing(t *testing.T) {
	path := "common/ideologies/a.txt"
	source := basicSource("\n", false)
	layers := []Layer{vanillaLayer(path, source)}
	subtype := Subtype{ScriptID: "project_type", ParentRootID: "democratic"}
	manifest := lockOne(t, layers, subtype)
	ids, err := CertifiedIDs(layers, manifest)
	if err != nil || len(ids) != 1 || ids[0] != "project_type" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}

	tailDrift := append([]byte(nil), source...)
	tailDrift = append(tailDrift, []byte("# changed\n")...)
	driftLayers := []Layer{vanillaLayer(path, tailDrift)}
	if ids, err = CertifiedIDs(driftLayers, manifest); err == nil || len(ids) != 0 || !findingCodes(err)["ideology_source_file_drift"] {
		t.Fatalf("ids=%v err=%v", ids, err)
	}

	rootDrift := bytes.Replace(source, []byte("color = { 0 0 255 }"), []byte("color = { 1 2 3 }"), 1)
	codes := validationCodes(Validate([]Layer{vanillaLayer(path, rootDrift)}, manifest, []Subtype{subtype}))
	if !codes["ideology_source_file_drift"] || !codes["ideology_source_root_drift"] {
		t.Fatalf("codes=%v", codes)
	}
}

func TestManagedRegionTamperAndManifestMismatchFailClosed(t *testing.T) {
	path := "common/ideologies/a.txt"
	layers := []Layer{vanillaLayer(path, basicSource("\n", false))}
	subtype := Subtype{ScriptID: "project_type", ParentRootID: "democratic"}
	manifest := lockOne(t, layers, subtype)
	compiled, err := Compile(layers, manifest, []Subtype{subtype})
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(compiled.Files[0].Bytes, []byte("can_be_randomly_selected = no"), []byte("can_be_randomly_selected = yes"), 1)
	_, err = Compile([]Layer{vanillaLayer(path, tampered)}, manifest, []Subtype{subtype})
	if !findingCodes(err)["ideology_source_managed_region_drift"] {
		t.Fatalf("err=%v", err)
	}
	codes := validationCodes(Validate(layers, manifest, nil))
	if !codes["ideology_source_subtype_missing"] {
		t.Fatalf("codes=%v", codes)
	}
}
