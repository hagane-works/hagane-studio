package exportmerge

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/paradox"
)

func gateParityMergeArchive(t *testing.T, files map[string][]byte) Archive {
	t.Helper()
	name := filepath.Join(t.TempDir(), "uploaded.zip")
	file, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry, err := writer.Create(key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write(files[key]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return gateParityMergeReadZIP(t, name)
}

func gateParityMergeReadZIP(t *testing.T, filename string) Archive {
	t.Helper()
	archive, err := ReadArchive(context.Background(), filename, DefaultArchiveBudget())
	if err != nil {
		t.Fatalf("read real ZIP: %v", err)
	}
	if len(archive.Entries) != len(archive.ByPath) {
		t.Fatal("duplicate final ZIP members")
	}
	return archive
}

func gateParityMergeSnapshot() exportengine.Snapshot {
	return exportengine.Snapshot{
		Schema: exportengine.SnapshotSchema, ProjectID: 915,
		Project: json.RawMessage(`{"id":915,"name":"Gate Merge","country_tag":"ROM","export_languages":["english"],"dependencies":[]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees": {json.RawMessage(`{"id":7,"tree_id":"gp_platform","tree_kind":"primary","country_tag":"ROM","scope_kind":"own","source":"created"}`)},
			"focus_nodes": {
				json.RawMessage(`{"id":1,"focus_tree_id":7,"focus_id":"GP_A_child","relative_position_id":"GP_Z_anchor","x":-3,"y":5,"cost":10}`),
				json.RawMessage(`{"id":2,"focus_tree_id":7,"focus_id":"GP_M_independent","x":9,"y":1,"cost":10}`),
				json.RawMessage(`{"id":3,"focus_tree_id":7,"focus_id":"GP_Z_anchor","x":6,"y":2,"cost":10}`),
			},
			"focus_file_policies": {json.RawMessage(`{"id":1,"target_path":"common/national_focus/gate_parity.txt","tree_ids":["gp_platform"]}`)},
		},
	}
}

func gateParityMergeScalar(block *paradox.Block, key string) string {
	for _, entry := range block.Entries {
		if entry.Key != nil && *entry.Key == key && entry.Block == nil {
			return entry.Scalar.StrValue()
		}
	}
	return ""
}

func gateParityMergeIDs(block *paradox.Block, keyword string) []string {
	ids := []string{}
	for _, entry := range block.Entries {
		if entry.Key != nil && *entry.Key == keyword && entry.Block != nil {
			ids = append(ids, gateParityMergeScalar(entry.Block, "id"))
		}
	}
	return ids
}

func gateParityMergeParse(t *testing.T, body []byte) *paradox.Block {
	t.Helper()
	root, recovery, err := paradox.Parse(string(body))
	if err != nil || len(recovery) != 0 {
		t.Fatalf("final script parse: err=%v recovery=%v\n%s", err, recovery, body)
	}
	return root
}

func gateParityMergeTree(t *testing.T, root *paradox.Block, id string) *paradox.Block {
	t.Helper()
	for _, entry := range root.Entries {
		if entry.Key != nil && *entry.Key == "focus_tree" && entry.Block != nil && gateParityMergeScalar(entry.Block, "id") == id {
			return entry.Block
		}
	}
	t.Fatalf("tree %q missing in actual merged script", id)
	return nil
}

func TestGateParityFocusArtifactAppendUsesWholeFinalPlatformFile(t *testing.T) {
	for _, rawPlatform := range []bool{false, true} {
		name := "generated_platform"
		if rawPlatform {
			name = "raw_platform"
		}
		t.Run(name, func(t *testing.T) {
			snapshot := gateParityMergeSnapshot()
			raw := "# untouched raw winner\r\nfocus_tree = { id = gp_raw focus = { id = GP_raw_child relative_position_id = GP_raw_anchor x = -4 y = 1 } focus = { id = GP_raw_anchor x = 8 y = 0 } }\r\n"
			if rawPlatform {
				body, err := json.Marshal(map[string]any{"id": 51, "dest_path": "common/national_focus/gate_parity.txt", "raw_kind": "verbatim_override", "source": "created", "content": raw})
				if err != nil {
					t.Fatal(err)
				}
				snapshot.Tables["project_raw_files"] = []json.RawMessage{body}
			}
			native, err := exportengine.BuildNativeShadow(snapshot, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			platformZIP := filepath.Join(t.TempDir(), "platform.zip")
			if _, err := exportengine.BuildWithMetadata(platformZIP, "gate-platform", native.Entries, exportengine.DefaultArtifactBudget(), native.Timeline); err != nil {
				t.Fatal(err)
			}
			platformArchive := gateParityMergeReadZIP(t, platformZIP)
			platform := ArchiveEntries(platformArchive, true)
			var platformBytes []byte
			for _, entry := range platform {
				if strings.HasSuffix(entry.Path, "/common/national_focus/gate_parity.txt") {
					platformBytes = entry.Bytes
				}
			}
			if len(platformBytes) == 0 {
				t.Fatal("native ZIP focus file missing")
			}
			untouched := []byte("# base-only focus file\r\nfocus_tree = { id = gp_keep focus = { id = GP_keep x = 1 y = 0 } }\r\n")
			base := gateParityMergeArchive(t, map[string][]byte{
				"Base/descriptor.mod":                        []byte("name=\"Base\"\n"),
				"Base/common/national_focus/gate_parity.txt": []byte("focus_tree = { id = gp_old focus = { id = GP_replaced x = 0 y = 0 } }\n"),
				"Base/common/national_focus/keep.txt":        untouched,
			})
			destination := filepath.Join(t.TempDir(), "appended.zip")
			manifest, conflicts, err := MergeAppendWithFindingsAndSnapshot(context.Background(), base, platform, destination, "gate-append", nil, snapshot, native.Timeline, english)
			if err != nil {
				t.Fatalf("append rejected content: %v", err)
			}
			merged := gateParityMergeReadZIP(t, destination)
			actual := merged.ByPath["Base/common/national_focus/gate_parity.txt"]
			if !bytes.Equal(actual, platformBytes) {
				t.Fatal("append must choose the whole platform focus file")
			}
			if !bytes.Equal(merged.ByPath["Base/common/national_focus/keep.txt"], untouched) {
				t.Fatal("base-only focus file changed")
			}
			if bytes.Contains(actual, []byte("GP_replaced")) {
				t.Fatal("append accidentally mixed base focus definitions")
			}
			if !reflect.DeepEqual(conflicts, []string{"Base/common/national_focus/gate_parity.txt"}) {
				t.Errorf("unexpected collision paths: %v", conflicts)
			}
			t.Logf("actual append ZIP bytes=%d sha256=%s focusBytes=%d", manifest.ZIPBytes, manifest.ZIPSHA256, len(actual))
			if rawPlatform {
				if !bytes.Equal(actual, []byte(raw)) {
					t.Fatal("raw platform winner was reformatted")
				}
			} else {
				ids := gateParityMergeIDs(gateParityMergeTree(t, gateParityMergeParse(t, actual), "gp_platform"), "focus")
				if want := []string{"GP_M_independent", "GP_Z_anchor", "GP_A_child"}; !reflect.DeepEqual(ids, want) {
					t.Errorf("actual appended ZIP order=%v want=%v", ids, want)
				}
			}
		})
	}
}

func TestGateParityFocusArtifactDiffSortsTouchedAndPreservesUntouched(t *testing.T) {
	const source = "Base/common/national_focus/edited.txt"
	untouched := []byte("# author trivia\r\nfocus_tree = { id = gp_keep focus = { id = GP_keep x = 2 y = 1 } }\r\n")
	base := gateParityMergeArchive(t, map[string][]byte{
		"Base/descriptor.mod": []byte("name = \"Base\"\n"),
		source: []byte(`focus_tree = {
 id = edited
 country = { factor = 100 modifier = { add = 100 tag = ROM } }
 focus = { id = GP_A_child relative_position_id = GP_Z_anchor x = -3 y = 5 cost = 10 completion_reward = { add_stability = 0.1 } }
 focus = { id = GP_M_independent x = 9 y = 1 cost = 10 }
 focus = { id = GP_Z_anchor x = 6 y = 2 cost = 10 }
}`),
		"Base/common/national_focus/keep.txt": untouched,
		"Base/gfx/keep.bin":                   []byte{0, 255, 3, 13, 10},
	})
	target := ParseTarget(context.Background(), base)
	if target.FocusNodes["GP_A_child"] == nil {
		t.Fatalf("target lost child: warnings=%v", target.Warnings)
	}
	destination := filepath.Join(t.TempDir(), "diff.zip")
	manifest, err := ApplyOpsToArchiveWithFindingsAndSnapshot(context.Background(), base, target, map[string][]DiffOp{
		"focus_node/GP_A_child": {{"op": "replace", "path": "cost", "old": float64(10), "value": float64(11)}},
	}, "ROM", destination, "gate-diff", nil, gateParityMergeSnapshot(), english)
	if err != nil {
		t.Fatalf("diff write failed: %v", err)
	}
	merged := gateParityMergeReadZIP(t, destination)
	actual := merged.ByPath[source]
	ids := gateParityMergeIDs(gateParityMergeTree(t, gateParityMergeParse(t, actual), "edited"), "focus")
	if want := []string{"GP_M_independent", "GP_Z_anchor", "GP_A_child"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("actual diff ZIP order=%v want=%v", ids, want)
	}
	for _, path := range []string{"Base/descriptor.mod", "Base/common/national_focus/keep.txt", "Base/gfx/keep.bin"} {
		if !bytes.Equal(merged.ByPath[path], base.ByPath[path]) {
			t.Errorf("untouched member changed: %s", path)
		}
	}
	for _, field := range []string{"cost = 11", "relative_position_id = GP_Z_anchor", "x = -3", "add_stability = 0.1"} {
		if !bytes.Contains(actual, []byte(field)) {
			t.Errorf("edited script lost %q", field)
		}
	}
	t.Logf("actual diff ZIP bytes=%d sha256=%s", manifest.ZIPBytes, manifest.ZIPSHA256)
}

func TestGateParityFocusArtifactDiffPreservesEverySourceBlock(t *testing.T) {
	tests := []struct {
		name, additional, expectedNode string
		expectedTrees, expectedPool    []string
	}{
		{"two_primary_trees", "focus_tree = { id = gp_beta focus = { id = GP_beta x = 2 y = 1 cost = 10 } }", "GP_beta", []string{"gp_alpha", "gp_beta"}, []string{}},
		{"primary_and_shared_pool", "shared_focus = { id = GP_pool x = 2 y = 1 cost = 10 }", "GP_pool", []string{"gp_alpha"}, []string{"GP_pool"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const source = "Base/common/national_focus/multiple.txt"
			body := "# first tree and all later blocks belong to this source\nfocus_tree = { id = gp_alpha country = { factor = 20 } focus = { id = GP_alpha x = 1 y = 1 cost = 10 } }\n" + tt.additional + "\n"
			base := gateParityMergeArchive(t, map[string][]byte{"Base/descriptor.mod": []byte("name=\"Base\"\n"), source: []byte(body)})
			target := ParseTarget(context.Background(), base)
			if target.FocusNodes[tt.expectedNode] == nil {
				t.Errorf("target parser discarded later block node %s; warnings=%v", tt.expectedNode, target.Warnings)
			}
			if target.FocusNodes["GP_alpha"] == nil {
				t.Fatal("first block unavailable for mutation")
			}
			destination := filepath.Join(t.TempDir(), "multi-diff.zip")
			manifest, err := ApplyOpsToArchiveWithFindingsAndSnapshot(context.Background(), base, target, map[string][]DiffOp{
				"focus_node/GP_alpha": {{"op": "replace", "path": "cost", "old": float64(10), "value": float64(11)}},
			}, "ROM", destination, "gate-multi-diff", nil, gateParityMergeSnapshot(), english)
			if err != nil {
				t.Fatalf("diff rejected multi-block content: %v", err)
			}
			merged := gateParityMergeReadZIP(t, destination)
			actual := merged.ByPath[source]
			root := gateParityMergeParse(t, actual)
			trees, pool := gateParityMergeIDs(root, "focus_tree"), gateParityMergeIDs(root, "shared_focus")
			if !reflect.DeepEqual(trees, tt.expectedTrees) {
				t.Errorf("actual ZIP tree identities=%v want=%v", trees, tt.expectedTrees)
			}
			if !reflect.DeepEqual(pool, tt.expectedPool) {
				t.Errorf("actual ZIP shared definitions=%v want=%v", pool, tt.expectedPool)
			}
			if !bytes.Contains(actual, []byte("id = "+tt.expectedNode)) {
				t.Errorf("actual ZIP deleted later block node %s", tt.expectedNode)
			}
			if !bytes.Contains(actual, []byte("cost = 11")) {
				t.Error("requested first-block edit was lost")
			}
			for _, entry := range root.Entries {
				if entry.Key == nil || *entry.Key != "focus_tree" || entry.Block == nil || gateParityMergeScalar(entry.Block, "id") != "gp_alpha" {
					continue
				}
				factor := ""
				for _, header := range entry.Block.Entries {
					if header.Key != nil && *header.Key == "country" && header.Block != nil {
						factor = gateParityMergeScalar(header.Block, "factor")
					}
				}
				if factor != "20" {
					t.Errorf("source tree country selection changed: factor=%q want=20", factor)
				}
			}
			for _, invented := range []string{"focus = None", "x = None", "y = None"} {
				if bytes.Contains(actual, []byte(invented)) {
					t.Errorf("diff invented invalid header value: %s", invented)
				}
			}
			t.Logf("actual multi-block diff ZIP bytes=%d sha256=%s\n%s", manifest.ZIPBytes, manifest.ZIPSHA256, actual)
		})
	}
}

func gateParityUnparsedFocusBytes() []byte {

	return []byte("# 作者原文，无法映射的坐标也必须保留\r\nfocus_tree={id=author_tree focus={id=GP_old x=inf y=0 cost=10}}\r\n")
}

func gateParityNewFocusOps() map[string][]DiffOp {
	return map[string][]DiffOp{"focus_node/GP_new": {
		{"op": "add", "path": "focus_id", "value": "GP_new"},
		{"op": "add", "path": "x", "value": 1},
		{"op": "add", "path": "y", "value": 2},
		{"op": "add", "path": "cost", "value": 10},
	}}
}

func TestGateParityFocusArtifactUnparsedDefaultPathRejectsWithoutReplacingZIP(t *testing.T) {
	const source = "common/national_focus/hagane_focus.txt"
	dir := t.TempDir()
	sourceZIP := filepath.Join(dir, "uploaded.zip")
	destination := filepath.Join(dir, "destination.zip")
	original := gateParityUnparsedFocusBytes()
	if _, err := exportengine.Build(sourceZIP, "gate-unparsed-base", []exportengine.Entry{
		{Path: source, Bytes: original}, {Path: "descriptor.mod", Bytes: []byte("name=\"Author Base\"\n")},
	}, exportengine.DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}

	if _, err := exportengine.Build(destination, "gate-previous-good", []exportengine.Entry{
		{Path: "previous-good.txt", Bytes: []byte("previous successful artifact\r\n")},
	}, exportengine.DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}
	sourceBefore, err := os.ReadFile(sourceZIP)
	if err != nil {
		t.Fatal(err)
	}
	destinationBefore, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	base := gateParityMergeReadZIP(t, sourceZIP)
	target := ParseTarget(context.Background(), base)
	if len(target.Warnings) == 0 || target.FocusNodes["GP_old"] != nil {
		t.Fatalf("fixture must be unparsed: warnings=%v nodes=%v", target.Warnings, target.FocusNodes)
	}
	_, applyErr := ApplyOpsToArchiveWithFindingsAndSnapshot(context.Background(), base, target, gateParityNewFocusOps(), "ROM", destination, "gate-unparsed-reject", nil, gateParityMergeSnapshot(), english)
	if applyErr == nil || !strings.Contains(applyErr.Error(), "cannot address focus in existing unparsed file") {
		t.Errorf("expected exact unparsed-file address error, got %v", applyErr)
	}
	sourceAfter, err := os.ReadFile(sourceZIP)
	if err != nil {
		t.Fatal(err)
	}
	destinationAfter, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceAfter, sourceBefore) || !bytes.Equal(base.ByPath[source], original) {
		t.Error("failed op mutated original uploaded ZIP or its loaded bytes")
	}
	if !bytes.Equal(destinationAfter, destinationBefore) {
		t.Error("failed op replaced previous valid destination ZIP")
	}
	if got := gateParityMergeReadZIP(t, sourceZIP).ByPath[source]; !bytes.Equal(got, original) {
		t.Error("reopened uploaded ZIP lost author bytes")
	}
	if got := gateParityMergeReadZIP(t, destination).ByPath["previous-good.txt"]; !bytes.Equal(got, []byte("previous successful artifact\r\n")) {
		t.Error("reopened destination ZIP lost previous successful member")
	}
	t.Logf("actual uploaded ZIP sha256 before=%x after=%x; destination before=%x after=%x", sha256.Sum256(sourceBefore), sha256.Sum256(sourceAfter), sha256.Sum256(destinationBefore), sha256.Sum256(destinationAfter))
}

func TestGateParityFocusArtifactUnparsedUntouchedMemberPreserved(t *testing.T) {
	const source = "common/national_focus/hagane_focus.txt"
	const edited = "common/national_focus/other.txt"
	original := gateParityUnparsedFocusBytes()
	base := gateParityMergeArchive(t, map[string][]byte{
		source: original, "descriptor.mod": []byte("name=\"Author Base\"\n"),
		edited: []byte("focus_tree={id=other focus={id=GP_edit x=1 y=2 cost=10}}\r\n"),
	})
	target := ParseTarget(context.Background(), base)
	if len(target.Warnings) == 0 || target.FocusNodes["GP_edit"] == nil {
		t.Fatalf("expected one unparsed file and a unique editable node: %v", target.Warnings)
	}
	destination := filepath.Join(t.TempDir(), "unrelated-op.zip")
	manifest, err := ApplyOpsToArchiveWithFindingsAndSnapshot(context.Background(), base, target, map[string][]DiffOp{
		"focus_node/GP_edit": {{"op": "replace", "path": "cost", "value": 11}},
	}, "ROM", destination, "gate-unparsed-untouched", nil, gateParityMergeSnapshot(), english)
	if err != nil {
		t.Fatalf("unrelated op was blocked by an unparsed file: %v", err)
	}
	actual := gateParityMergeReadZIP(t, destination)
	if !bytes.Equal(actual.ByPath[source], original) || !bytes.Equal(base.ByPath[source], original) {
		t.Error("untouched unparsed author bytes changed")
	}
	if !bytes.Contains(actual.ByPath[edited], []byte("cost = 11")) {
		t.Errorf("unrelated requested edit missing: %s", actual.ByPath[edited])
	}
	t.Logf("actual unrelated-op ZIP bytes=%d sha256=%s unparsed-member-sha256=%x", manifest.ZIPBytes, manifest.ZIPSHA256, sha256.Sum256(actual.ByPath[source]))
}

func TestGateParityFocusArtifactAbsentDefaultPathCreatesNewFile(t *testing.T) {
	const source = "common/national_focus/hagane_focus.txt"
	base := gateParityMergeArchive(t, map[string][]byte{
		"descriptor.mod": []byte("name=\"Author Base\"\r\n"), "gfx/keep.bin": {0, 255, 13, 10},
	})
	target := ParseTarget(context.Background(), base)
	destination := filepath.Join(t.TempDir(), "new-file.zip")
	manifest, err := ApplyOpsToArchiveWithFindingsAndSnapshot(context.Background(), base, target, gateParityNewFocusOps(), "ROM", destination, "gate-absent-create", nil, gateParityMergeSnapshot(), english)
	if err != nil {
		t.Fatalf("genuinely new focus file rejected: %v", err)
	}
	actual := gateParityMergeReadZIP(t, destination)
	root := gateParityMergeParse(t, actual.ByPath[source])
	ids := gateParityMergeIDs(gateParityMergeTree(t, root, "hagane_focus"), "focus")
	if !reflect.DeepEqual(ids, []string{"GP_new"}) {
		t.Errorf("new ZIP focus definitions=%v", ids)
	}
	for _, name := range []string{"descriptor.mod", "gfx/keep.bin"} {
		if !bytes.Equal(actual.ByPath[name], base.ByPath[name]) {
			t.Errorf("new-file op altered existing member %s", name)
		}
	}
	t.Logf("actual new-file ZIP bytes=%d sha256=%s", manifest.ZIPBytes, manifest.ZIPSHA256)
}
