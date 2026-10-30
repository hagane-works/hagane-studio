package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	previewemit "haganeworks/internal/hoi4/emit/preview"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

const gateParityFocusPath = "common/national_focus/gate_parity.txt"

func gateParityArtifactJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func gateParityArtifactTree(id int, key, kind string) map[string]any {
	return map[string]any{"id": id, "tree_id": key, "tree_kind": kind, "country_tag": "ROM", "scope_kind": "own", "source": "created"}
}

func gateParityArtifactNode(id, tree int, key, anchor string) map[string]any {
	return map[string]any{"id": id, "focus_tree_id": tree, "focus_id": key, "relative_position_id": anchor, "x": id * 3, "y": id, "cost": 10, "name_en": key, "icon": "GFX_goal_unknown"}
}

func gateParityArtifactSnapshot(t *testing.T, trees, nodes []map[string]any, grouped bool) Snapshot {
	t.Helper()
	snapshot := Snapshot{Schema: SnapshotSchema, ProjectID: 914, Project: json.RawMessage(`{"id":914,"name":"Gate Parity","country_tag":"ROM","export_languages":["english"],"dependencies":[]}`), Tables: map[string][]json.RawMessage{}}
	ids := []string{}
	for _, tree := range trees {
		snapshot.Tables["focus_trees"] = append(snapshot.Tables["focus_trees"], gateParityArtifactJSON(t, tree))
		ids = append(ids, tree["tree_id"].(string))
	}
	for _, node := range nodes {
		snapshot.Tables["focus_nodes"] = append(snapshot.Tables["focus_nodes"], gateParityArtifactJSON(t, node))
	}
	if grouped {
		snapshot.Tables["focus_file_policies"] = []json.RawMessage{gateParityArtifactJSON(t, map[string]any{"id": 1, "target_path": gateParityFocusPath, "tree_ids": ids})}
	}
	return snapshot
}

func gateParityArtifactZIP(t *testing.T, snapshot Snapshot) (map[string][]byte, Manifest) {
	t.Helper()
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("native builder rejected content: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "gate-parity.zip")
	manifest, err := BuildWithMetadata(destination, "gate-parity-fixture", artifact.Entries, DefaultArtifactBudget(), artifact.Timeline)
	if err != nil {
		t.Fatalf("ZIP writer rejected content: %v", err)
	}
	reader, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	files := map[string][]byte{}
	for _, file := range reader.File {
		if _, exists := files[file.Name]; exists {
			t.Fatalf("duplicate ZIP member %q", file.Name)
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		files[file.Name] = body
	}
	if manifest.ZIPBytes == 0 || len(files[manifest.ManifestPath]) == 0 {
		t.Fatal("actual ZIP lacks artifact manifest")
	}
	t.Logf("real ZIP bytes=%d sha256=%s members=%d", manifest.ZIPBytes, manifest.ZIPSHA256, len(files))
	return files, manifest
}

func gateParityArtifactMember(t *testing.T, files map[string][]byte, relative string) (string, []byte) {
	t.Helper()
	var name string
	for current := range files {
		if current != relative && !strings.HasSuffix(current, "/"+relative) {
			continue
		}
		if name != "" {
			t.Fatalf("multiple ZIP winners for %q", relative)
		}
		name = current
	}
	if name == "" {
		t.Fatalf("ZIP member %q missing", relative)
	}
	body := files[name]
	t.Logf("focus member=%s sha256=%x", name, sha256.Sum256(body))
	return name, body
}

func gateParityArtifactParse(t *testing.T, body []byte) *paradox.Block {
	t.Helper()
	block, recovery, err := paradox.Parse(string(body))
	if err != nil || len(recovery) != 0 {
		t.Fatalf("artifact parse err=%v recovery=%v\n%s", err, recovery, body)
	}
	return block
}

func gateParityArtifactScalar(block *paradox.Block, key string) string {
	for _, entry := range block.Entries {
		if entry.Key != nil && *entry.Key == key && entry.Block == nil {
			return entry.Scalar.StrValue()
		}
	}
	return ""
}

func gateParityArtifactTreeBlock(t *testing.T, root *paradox.Block, treeID string) *paradox.Block {
	t.Helper()
	var found *paradox.Block
	for _, entry := range root.Entries {
		if entry.Key != nil && *entry.Key == "focus_tree" && entry.Block != nil && gateParityArtifactScalar(entry.Block, "id") == treeID {
			if found != nil {
				t.Fatalf("tree %s defined twice", treeID)
			}
			found = entry.Block
		}
	}
	if found == nil {
		t.Fatalf("tree %s missing from actual ZIP script", treeID)
	}
	return found
}

func gateParityArtifactIDs(block *paradox.Block, keyword string) []string {
	ids := []string{}
	for _, entry := range block.Entries {
		if entry.Key != nil && *entry.Key == keyword && entry.Block != nil {
			ids = append(ids, gateParityArtifactScalar(entry.Block, "id"))
		}
	}
	return ids
}

func TestGateParityFocusArtifactForwardAnchorAndPreview(t *testing.T) {
	tree := gateParityArtifactTree(7, "gp_main", "primary")
	child := gateParityArtifactNode(1, 7, "GP_child", "GP_anchor")
	child["x"], child["y"], child["completion_reward"] = -3, 5, "add_stability = 0.1"
	nodes := []map[string]any{child, gateParityArtifactNode(2, 7, "GP_independent", ""), gateParityArtifactNode(3, 7, "GP_anchor", "")}
	files, _ := gateParityArtifactZIP(t, gateParityArtifactSnapshot(t, []map[string]any{tree}, nodes, true))
	_, body := gateParityArtifactMember(t, files, gateParityFocusPath)
	actual := gateParityArtifactIDs(gateParityArtifactTreeBlock(t, gateParityArtifactParse(t, body), "gp_main"), "focus")
	preview := previewemit.GenerateFocusTreeScript(previewemit.BuildFocusTree(&importsvc.TreeBundle{TreeData: tree, TreeKind: "primary", NodesData: nodes}), nil)
	previewIDs := gateParityArtifactIDs(gateParityArtifactTreeBlock(t, gateParityArtifactParse(t, []byte(preview)), "gp_main"), "focus")
	want := []string{"GP_independent", "GP_anchor", "GP_child"}
	if !reflect.DeepEqual(actual, want) {
		t.Errorf("real ZIP definition order=%v want=%v", actual, want)
	}
	if !reflect.DeepEqual(previewIDs, want) {
		t.Errorf("real preview definition order=%v want=%v", previewIDs, want)
	}
	if !reflect.DeepEqual(actual, previewIDs) {
		t.Errorf("preview/ZIP disagree: %v / %v", previewIDs, actual)
	}
	for _, token := range []string{"relative_position_id = GP_anchor", "x = -3", "y = 5", "add_stability = 0.1"} {
		if !bytes.Contains(body, []byte(token)) {
			t.Errorf("author field lost in real ZIP: %s", token)
		}
	}
}

func TestGateParityFocusArtifactInvalidRegionsStillGenerate(t *testing.T) {
	tests := []struct {
		name                   string
		a, b, anchorA, anchorB string
	}{
		{"cycle", "GP_a", "GP_b", "GP_b", "GP_a"},
		{"missing_anchor", "GP_a", "GP_b", "GP_external", "GP_a"},
		{"duplicate_id", "GP_duplicate", "GP_duplicate", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := []map[string]any{gateParityArtifactNode(1, 7, tt.a, tt.anchorA), gateParityArtifactNode(2, 7, "GP_leaf", "GP_root"), gateParityArtifactNode(3, 7, tt.b, tt.anchorB), gateParityArtifactNode(4, 7, "GP_root", "")}
			files, _ := gateParityArtifactZIP(t, gateParityArtifactSnapshot(t, []map[string]any{gateParityArtifactTree(7, "gp_main", "primary")}, nodes, true))
			_, body := gateParityArtifactMember(t, files, gateParityFocusPath)
			ids := gateParityArtifactIDs(gateParityArtifactTreeBlock(t, gateParityArtifactParse(t, body), "gp_main"), "focus")
			want := []string{tt.a, tt.b, "GP_root", "GP_leaf"}
			if !reflect.DeepEqual(ids, want) {
				t.Errorf("actual ZIP fallback order=%v want=%v", ids, want)
			}
			for _, anchor := range []string{tt.anchorA, tt.anchorB, "GP_root"} {
				if anchor != "" && !bytes.Contains(body, []byte("relative_position_id = "+anchor)) {
					t.Errorf("author anchor %s dropped", anchor)
				}
			}
		})
	}
}

func TestGateParityFocusArtifactRawFinalWinnerIsVerbatim(t *testing.T) {
	snapshot := gateParityArtifactSnapshot(t, []map[string]any{gateParityArtifactTree(7, "gp_main", "primary")}, []map[string]any{gateParityArtifactNode(1, 7, "GP_generated", "")}, true)
	raw := "# author spacing and CRLF must survive\r\nfocus_tree = {\r\n id = gp_raw\r\n focus = { id = GP_raw_child x = -2 y = 1 relative_position_id = GP_raw_anchor }\r\n focus = { id = GP_raw_anchor x = 6 y = 0 }\r\n}\r\n"
	snapshot.Tables["project_raw_files"] = []json.RawMessage{gateParityArtifactJSON(t, map[string]any{"id": 51, "dest_path": gateParityFocusPath, "raw_kind": "verbatim_override", "source": "created", "content": raw, "is_binary": false})}
	files, manifest := gateParityArtifactZIP(t, snapshot)
	name, actual := gateParityArtifactMember(t, files, gateParityFocusPath)
	if !bytes.Equal(actual, []byte(raw)) {
		t.Fatalf("raw final winner was rewritten:\n%s", actual)
	}
	for _, entry := range manifest.Entries {
		if entry.Path == name {
			if entry.Origin != "raw_override" || entry.SourceEntity != "project_raw_files:51" {
				t.Fatalf("manifest misidentifies raw winner: %+v", entry)
			}
			return
		}
	}
	t.Fatal("raw winner missing in manifest")
}

func TestGateParityFocusArtifactSharedAndBlockBoundaries(t *testing.T) {
	for _, grouped := range []bool{true, false} {
		t.Run(fmt.Sprintf("same_output_file_%t", grouped), func(t *testing.T) {
			main := gateParityArtifactTree(7, "gp_main", "primary")
			main["shared_focus_refs"] = []map[string]any{{"focus_id": "GP_pool_root"}}
			trees := []map[string]any{main, gateParityArtifactTree(8, "gp_other", "primary"), gateParityArtifactTree(9, "gp_pool", "shared_pool")}
			nodes := []map[string]any{
				gateParityArtifactNode(1, 7, "GP_local_child", "GP_local_root"), gateParityArtifactNode(2, 7, "GP_local_root", ""),
				gateParityArtifactNode(3, 7, "GP_cross_block", "GP_pool_root"), gateParityArtifactNode(4, 8, "GP_other", ""),
				gateParityArtifactNode(5, 9, "GP_pool_child", "GP_pool_root"), gateParityArtifactNode(6, 9, "GP_pool_root", ""),
			}
			files, _ := gateParityArtifactZIP(t, gateParityArtifactSnapshot(t, trees, nodes, grouped))
			combined := []byte{}
			focusFileCount := 0
			for name, body := range files {
				if strings.Contains(name, "/common/national_focus/") {
					combined = append(combined, body...)
					combined = append(combined, '\n')
					focusFileCount++
				}
			}
			wantFileCount := 3
			if grouped {
				wantFileCount = 1
			}
			if focusFileCount != wantFileCount {
				t.Errorf("file policy changed: focus files=%d want=%d", focusFileCount, wantFileCount)
			}
			root := gateParityArtifactParse(t, combined)
			treeIDs := gateParityArtifactIDs(root, "focus_tree")
			sort.Strings(treeIDs)
			if !reflect.DeepEqual(treeIDs, []string{"gp_main", "gp_other"}) {
				t.Errorf("primary blocks duplicated or lost: %v", treeIDs)
			}
			mainIDs := gateParityArtifactIDs(gateParityArtifactTreeBlock(t, root, "gp_main"), "focus")
			if want := []string{"GP_local_root", "GP_local_child", "GP_cross_block"}; !reflect.DeepEqual(mainIDs, want) {
				t.Errorf("main block ids=%v want=%v", mainIDs, want)
			}
			otherIDs := gateParityArtifactIDs(gateParityArtifactTreeBlock(t, root, "gp_other"), "focus")
			if !reflect.DeepEqual(otherIDs, []string{"GP_other"}) {
				t.Errorf("other primary block changed: %v", otherIDs)
			}
			poolIDs := gateParityArtifactIDs(root, "shared_focus")
			if !reflect.DeepEqual(poolIDs, []string{"GP_pool_root", "GP_pool_child"}) {
				t.Errorf("pool definitions=%v", poolIDs)
			}
			mainBlock := gateParityArtifactTreeBlock(t, root, "gp_main")
			if gateParityArtifactScalar(mainBlock, "shared_focus") != "GP_pool_root" {
				t.Error("shared include was dropped")
			}
			crossAnchor := ""
			for _, entry := range mainBlock.Entries {
				if entry.Key != nil && *entry.Key == "focus" && entry.Block != nil && gateParityArtifactScalar(entry.Block, "id") == "GP_cross_block" {
					crossAnchor = gateParityArtifactScalar(entry.Block, "relative_position_id")
				}
			}
			if crossAnchor != "GP_pool_root" {
				t.Errorf("cross-block author's own reference changed: %q", crossAnchor)
			}
		})
	}
}
