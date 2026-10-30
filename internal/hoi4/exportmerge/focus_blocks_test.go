package exportmerge

import (
	"reflect"
	"strings"
	"testing"
)

func focusBlocksParse(t *testing.T, text string) *FocusSourceFile {
	t.Helper()
	file, err := ParseFocusSourceFile(text, "Base/common/national_focus/multiple.txt")
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func focusBlocksPatch(t *testing.T, file *FocusSourceFile, id string, ops ...DiffOp) map[string]map[string]any {
	t.Helper()
	for _, row := range file.Rows() {
		if row["focus_id"] != id {
			continue
		}
		patched, err := ApplyDiff(row, ops)
		if err != nil {
			t.Fatal(err)
		}
		return map[string]map[string]any{"focus_node/" + id: patched}
	}
	t.Fatalf("missing fixture ID %s", id)
	return nil
}

func focusBlocksRender(t *testing.T, file *FocusSourceFile, overrides map[string]map[string]any) string {
	t.Helper()
	text, err := file.Render(overrides)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func focusBlocksIDs(file *FocusSourceFile) []string {
	var ids []string
	for _, row := range file.Rows() {
		ids = append(ids, stringField(row, "focus_id"))
	}
	return ids
}

func TestFocusSourceBlocksCostPatchPreservesAuthorBytes(t *testing.T) {

	text := "\ufeff# 中文作者 { }\r\n@x = 12\r\n" +
		"focus_tree = {\r\n id = gp_alpha\r\n country = { factor = 20 }\r\n" +
		" custom_header = { text = \"双引号 \\\" # } {\" }\r\n" +
		" focus = { id = GP_alpha x = @x y = 1 cost = 10 # 花括号 }\r\n" +
		"  available = { log = \"literal { # }\" }\r\n  custom_key = { test = yes }\r\n }\r\n}\r\n" +
		"# 第二棵树保持字节\r\nfocus_tree={id=gp_beta country={factor=31} focus={id=GP_beta cost=7}}\r\n" +
		"shared_focus = { id = GP_pool icon = GFX_pool cost = 3 } # 原样\r\n"
	file := focusBlocksParse(t, text)
	if got := focusBlocksIDs(file); !reflect.DeepEqual(got, []string{"GP_alpha", "GP_beta", "GP_pool"}) {
		t.Fatalf("nodes=%v", got)
	}
	got := focusBlocksRender(t, file, focusBlocksPatch(t, file, "GP_alpha", DiffOp{"op": "replace", "path": "cost", "value": 11}))
	want := strings.Replace(text, "cost = 10", "cost = 11", 1)
	if got != want {
		t.Errorf("only cost bytes may change\ngot: %q\nwant: %q", got, want)
	}
	if again := focusBlocksRender(t, file, nil); again != text {
		t.Fatal("Render mutated immutable author text")
	}
	rows := file.Rows()
	rows[0]["cost"] = -1
	rows[0]["raw_extras"].(map[string]any)["custom_key"] = "lost"
	if reflect.DeepEqual(rows[0], file.Rows()[0]) {
		t.Fatal("Rows leaked mutable internal maps")
	}
}

func TestFocusSourceBlocksSortOnlyTouchedPrimary(t *testing.T) {
	first := "focus_tree = { id = a country={factor=20}\n focus={id=A relative_position_id=Z cost=10 x=1 y=2}\n focus={id=M x=2 y=3}\n focus={id=Z x=4 y=5}\n}\n"
	second := "focus_tree={id=b focus={id=B_child relative_position_id=B_root} focus={id=B_root}}\r\n"
	file := focusBlocksParse(t, first+second)
	overrides := focusBlocksPatch(t, file, "A", DiffOp{"op": "replace", "path": "cost", "value": 12})
	got := focusBlocksRender(t, file, overrides)
	if !strings.HasSuffix(got, second) || !strings.HasPrefix(got, "focus_tree = { id = a country={factor=20}\n ") {
		t.Fatalf("untouched tree or original header changed: %s", got)
	}
	parsed := focusBlocksParse(t, got)
	want := []string{"M", "Z", "A", "B_child", "B_root"}
	if ids := focusBlocksIDs(parsed); !reflect.DeepEqual(ids, want) {
		t.Fatalf("order=%v want=%v", ids, want)
	}
	if repeated := focusBlocksRender(t, file, overrides); repeated != got {
		t.Fatal("render is not deterministic")
	}
	noChange := focusBlocksPatch(t, parsed, "A")
	if repeated := focusBlocksRender(t, parsed, noChange); repeated != got {
		t.Fatal("second parse/render changed already ordered source")
	}
}

func TestFocusSourceBlocksSharedJointAndPrimaryBoundaries(t *testing.T) {
	pool := "shared_focus={id=S_child relative_position_id=S_root cost=10}\n" +
		"joint_focus={id=S_root cost=10}\n"
	primary := "focus_tree={id=middle shared_focus=S_child focus={id=P relative_position_id=S_root}}\n"
	last := "shared_focus={id=Last relative_position_id=S_child cost=10}\n"
	file := focusBlocksParse(t, pool+primary+last)
	got := focusBlocksRender(t, file, focusBlocksPatch(t, file, "S_child", DiffOp{"op": "replace", "path": "cost", "value": 11}))
	if !strings.HasPrefix(got, "joint_focus={id=S_root cost=10}\nshared_focus={id=S_child") || !strings.HasSuffix(got, primary+last) {
		t.Fatalf("pool keyword/order or raw boundary lost: %s", got)
	}
	if focusBlocksParse(t, got).Count("S_root") != 1 {
		t.Fatal("shared node copied")
	}
	if len(file.blocks) != 3 || file.blocks[0].kind != "shared_pool" || file.blocks[1].kind != "primary" {
		t.Fatal("source block ownership lost")
	}
}

func TestFocusSourceBlocksAmbiguousOpsAreLocal(t *testing.T) {
	text := "focus_tree={id=a focus={id=D cost=10} focus={id=U cost=10}}\nfocus_tree={id=b focus={id=D cost=7}}\n"
	file := focusBlocksParse(t, text)
	if file.Count("D") != 2 {
		t.Fatal("duplicate ID overwritten")
	}
	if _, err := file.Render(focusBlocksPatch(t, file, "D", DiffOp{"op": "replace", "path": "cost", "value": 1})); err == nil || !strings.Contains(err.Error(), "2 definitions") {
		t.Fatalf("ambiguous override err=%v", err)
	}
	got := focusBlocksRender(t, file, focusBlocksPatch(t, file, "U", DiffOp{"op": "replace", "path": "cost", "value": 11}))
	if strings.Count(got, "id=D") != 2 || !strings.Contains(got, "id=U cost = 11") {
		t.Fatalf("unrelated unique op failed or discarded duplicates: %s", got)
	}
	files := map[string]*FocusSourceFile{"one": file, "two": focusBlocksParse(t, "shared_focus={id=U}")}
	if err := ValidateFocusTarget(files, "U"); err == nil {
		t.Fatal("cross-source duplicate accepted")
	}
	if err := ValidateFocusTarget(files, "new"); err != nil {
		t.Fatalf("blocked caller new-file path: %v", err)
	}
	if _, err := file.Render(map[string]map[string]any{"focus_node/new": {"focus_id": "new"}}); err == nil {
		t.Fatal("new node silently placed in an arbitrary existing block")
	}
}

func TestFocusSourceBlocksInvalidComponentsStillRender(t *testing.T) {
	for _, tc := range []struct{ name, a, b, ids string }{
		{"cycle", "id=A relative_position_id=B", "id=B relative_position_id=A", "A,B,R,L"},
		{"missing_anchor", "id=A relative_position_id=gone", "id=B relative_position_id=A", "A,B,R,L"},
		{"duplicate", "id=A", "id=A", "A,A,R,L"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := "focus_tree={id=t\nfocus={" + tc.a + "}\nfocus={id=L relative_position_id=R cost=10}\nfocus={" + tc.b + "}\nfocus={id=R}\n}"
			file := focusBlocksParse(t, text)
			got := focusBlocksRender(t, file, focusBlocksPatch(t, file, "L", DiffOp{"op": "replace", "path": "cost", "value": 11}))
			if ids := strings.Join(focusBlocksIDs(focusBlocksParse(t, got)), ","); ids != tc.ids {
				t.Fatalf("fallback order %s want %s", ids, tc.ids)
			}
			if !strings.Contains(got, "focus={"+tc.a+"}") || !strings.Contains(got, "focus={"+tc.b+"}") {
				t.Fatal("invalid author content changed")
			}
		})
	}
}

func TestFocusSourceBlocksFieldRemovalAndRepeatedFields(t *testing.T) {
	text := "focus_tree={id=t focus={id=A x=@missing y=0 cost=10 relative_position_id=gone prerequisite={focus=X} prerequisite={focus=Y} custom = { x = yes }}}"
	file := focusBlocksParse(t, text)
	overrides := focusBlocksPatch(t, file, "A",
		DiffOp{"op": "remove", "path": "relative_position_id"},
		DiffOp{"op": "remove", "path": "x"},
		DiffOp{"op": "replace", "path": "prerequisites", "value": []any{[]any{"Z"}}},
		DiffOp{"op": "add", "path": "available", "value": "has_war = yes"},
	)
	got := focusBlocksRender(t, file, overrides)
	if strings.Contains(got, "relative_position_id") || strings.Contains(got, "x=@missing") || strings.Contains(got, "None") {
		t.Fatalf("remove became invalid renderer default: %s", got)
	}
	if strings.Count(got, "prerequisite") != 1 || !strings.Contains(got, "focus = Z") || !strings.Contains(got, "has_war = yes") || !strings.Contains(got, "custom = { x = yes }") {
		t.Fatalf("repeat/add/raw patch failure: %s", got)
	}
}

func TestFocusSourceBlocksEmptyAndOpaqueSource(t *testing.T) {
	for _, text := range []string{"", "# comments { }\r\n", "@constant=4\nunknown={focus={id=hidden}}\nfocus_tree={id=empty custom=yes}"} {
		file := focusBlocksParse(t, text)
		if got := focusBlocksRender(t, file, nil); got != text || len(file.Rows()) != 0 {
			t.Fatalf("opaque source changed: %q", got)
		}
	}
}

func TestFocusSourceBlocksRejectMalformedSpansWithoutMutation(t *testing.T) {
	for _, text := range []string{"focus_tree={", "focus_tree={}}", "focus_tree={ id=\"unfinished }"} {
		if _, err := ParseFocusSourceFile(text, "broken.txt"); err == nil {
			t.Errorf("accepted unsafe replacement spans in %q", text)
		}
	}
}

func TestFocusSourceBlocksExplicitRenameCollisions(t *testing.T) {
	file := focusBlocksParse(t, "focus_tree={id=t focus={id=A} focus={id=B}}")
	files := map[string]*FocusSourceFile{"one": file, "two": focusBlocksParse(t, "shared_focus={id=C}")}
	if err := ValidateFocusOverride(files, "A", map[string]any{"focus_id": "C"}); err == nil {
		t.Fatal("cross-file ID collision accepted")
	}
	if err := ValidateFocusOverride(files, "A", map[string]any{"focus_id": "New"}); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Render(focusBlocksPatch(t, file, "A", DiffOp{"op": "replace", "path": "focus_id", "value": "B"})); err == nil {
		t.Fatal("same-file ID collision accepted")
	}
	overrides := focusBlocksPatch(t, file, "A", DiffOp{"op": "replace", "path": "focus_id", "value": "New"})
	for key, row := range focusBlocksPatch(t, file, "B", DiffOp{"op": "replace", "path": "focus_id", "value": "New"}) {
		overrides[key] = row
	}
	if _, err := file.Render(overrides); err == nil {
		t.Fatal("two patches created the same new ID")
	}
	if err := ValidateFocusOverrides(files, map[string]map[string]any{
		"focus_node/A": {"focus_id": "New"}, "focus_node/C": {"focus_id": "New"},
	}); err == nil {
		t.Fatal("two source files created the same new ID")
	}
	if err := ValidateFocusOverrides(files, map[string]map[string]any{
		"focus_node/A": {"focus_id": "New"}, "event/ignored": {"focus_id": "New"},
	}); err != nil {
		t.Fatalf("batch includes irrelevant non-focus entity: %v", err)
	}
}

func TestFocusSourceBlocksAddedFieldUsesFileCRLFAndMetadataOnlyIsRaw(t *testing.T) {
	text := "# CRLF 文本\r\nfocus_tree={id=t focus={id=A}}\r\n"
	file := focusBlocksParse(t, text)
	got := focusBlocksRender(t, file, focusBlocksPatch(t, file, "A", DiffOp{"op": "add", "path": "available", "value": "has_war = yes"}))
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Fatalf("introduced LF into CRLF source: %q", got)
	}
	got = focusBlocksRender(t, file, focusBlocksPatch(t, file, "A", DiffOp{"op": "replace", "path": "name", "value": "本地化名字"}))
	if got != text {
		t.Fatal("localisation-only patch changed author script")
	}
}
