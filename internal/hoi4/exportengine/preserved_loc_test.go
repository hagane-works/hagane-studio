package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func locRow(id int64, locale, key, value, sourceFile, source string,
	version *int, sourceOrder, entryOrder int) json.RawMessage {
	b, err := json.Marshal(map[string]any{
		"id": id, "locale": locale, "key": key, "value": value,
		"source_file": sourceFile, "source": source, "version": version,
		"source_order": sourceOrder, "entry_order": entryOrder,
	})
	if err != nil {
		panic(err)
	}
	return b
}

func snapWith(rows ...json.RawMessage) Snapshot {
	return Snapshot{Tables: map[string][]json.RawMessage{"project_loc_entries": rows}}
}

func onlyEntry(t *testing.T, entries []Entry) Entry {
	t.Helper()
	if len(entries) != 1 {
		t.Fatalf("want exactly 1 entry, got %d", len(entries))
	}
	return entries[0]
}

func TestPreservedLoc_StaticGameViewGUIOverrideExportsReplaceFile(t *testing.T) {
	const sourceFile = "localisation/english/replace/zz_game_view_l_english.yml"
	const value = "§RPolitical Power§!"
	snap := snapWith(locRow(
		1, "l_english", "POLITICAL_POWER", value,
		sourceFile, "created", nil, 0, 0,
	))

	entries, warns, err := NativePreservedLoc(snap, "Demo", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	e := onlyEntry(t, entries)
	if want := "Demo/" + sourceFile; e.Path != want {
		t.Fatalf("path = %q, want %q", e.Path, want)
	}
	if e.Origin != "preserved_localisation" || e.SourceEntity != "project_loc_file" {
		t.Fatalf("entry provenance = (%q, %q)", e.Origin, e.SourceEntity)
	}
	if want := bomUTF8 + "l_english:\n POLITICAL_POWER: \"" + value + "\"\n"; string(e.Bytes) != want {
		t.Fatalf("exported static GUI localisation differs\n--- want ---\n%q\n--- got ---\n%q", want, string(e.Bytes))
	}
}

func TestPreservedLoc_ReplaysImportedKeysAtOriginalPath(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_english", "PROBE_FREEFORM_A", "freeform-alpha",
			"localisation/english/zzz_probe_l_english.yml", "imported", nil, 0, 0),
		locRow(2, "l_english", "GER_VICTORY_POINTS_11666", "Wien-probe",
			"localisation/english/zzz_probe_l_english.yml", "imported", nil, 0, 1),
	)
	entries, warns, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	e := onlyEntry(t, entries)
	if e.Path != "mod/localisation/english/zzz_probe_l_english.yml" {
		t.Fatalf("path = %q", e.Path)
	}
	body := string(e.Bytes)
	if !strings.HasPrefix(body, bomUTF8) {
		t.Fatal("HOI4 不加载没有 BOM 的非 ASCII loc 文件 —— BOM 缺失")
	}
	for _, want := range []string{
		"l_english:\n",
		" PROBE_FREEFORM_A: \"freeform-alpha\"\n",
		" GER_VICTORY_POINTS_11666: \"Wien-probe\"\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q\n--- got ---\n%s", want, body)
		}
	}
	if e.Origin != "preserved_localisation" {
		t.Fatalf("origin = %q", e.Origin)
	}
}

func TestPreservedLoc_ExcludesKeysAlreadyWritten(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_english", "GER_focus_x", "OLD imported name",
			"localisation/english/old_l_english.yml", "imported", nil, 0, 0),
		locRow(2, "l_english", "UNTOUCHED_KEY", "keep me",
			"localisation/english/old_l_english.yml", "imported", nil, 0, 1),
	)
	existing := []Entry{{
		Path:  "mod/localisation/english/GER_focus_l_english.yml",
		Bytes: []byte(bomUTF8 + "l_english:\n GER_focus_x: \"NEW workbench name\"\n"),
	}}
	entries, _, err := NativePreservedLoc(snap, "mod", true, existing)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Contains(body, "OLD imported name") {
		t.Fatal("排除集失效:陈旧导入值盖过了工作台的编辑")
	}
	if !strings.Contains(body, "UNTOUCHED_KEY") {
		t.Fatal("排除集过宽:没被结构化 emitter 写过的 key 也被吃掉了")
	}
}

func TestPreservedLoc_ExclusionIsPerLocale(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_simp_chinese", "GER_focus_x", "中文名",
			"localisation/simp_chinese/old_l_simp_chinese.yml", "imported", nil, 0, 0),
	)
	existing := []Entry{{
		Path:  "mod/localisation/english/GER_focus_l_english.yml",
		Bytes: []byte(bomUTF8 + "l_english:\n GER_focus_x: \"English only\"\n"),
	}}
	entries, _, err := NativePreservedLoc(snap, "mod", true, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatal("同 key 不同 locale 被误排除")
	}
}

func TestPreservedLoc_SourceFilter(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_english", "IMPORTED_K", "i", "localisation/english/a_l_english.yml", "imported", nil, 0, 0),
		locRow(2, "l_english", "CREATED_K", "c", "localisation/english/a_l_english.yml", "created", nil, 0, 1),
	)
	entries, _, err := NativePreservedLoc(snap, "mod", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Contains(body, "IMPORTED_K") {
		t.Fatal("include_imported=false 时不该发 imported 行")
	}
	if !strings.Contains(body, "CREATED_K") {
		t.Fatal("created 行任何时候都该发")
	}
}

func TestPreservedLoc_VersionRoundTrip(t *testing.T) {
	zero := 0
	three := 3
	snap := snapWith(
		locRow(1, "l_english", "NO_VERSION", "a", "localisation/english/v_l_english.yml", "imported", nil, 0, 0),
		locRow(2, "l_english", "ZERO_VERSION", "b", "localisation/english/v_l_english.yml", "imported", &zero, 0, 1),
		locRow(3, "l_english", "THREE_VERSION", "c", "localisation/english/v_l_english.yml", "imported", &three, 0, 2),
	)
	entries, _, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	for _, want := range []string{
		" NO_VERSION: \"a\"\n",
		" ZERO_VERSION:0 \"b\"\n",
		" THREE_VERSION:3 \"c\"\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("version 未原样保留,missing %q\n--- got ---\n%s", want, body)
		}
	}
}

func TestPreservedLoc_LocaleHeaderRepeatsOnChange(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_english", "A", "a", "localisation/mixed.yml", "imported", nil, 0, 0),
		locRow(2, "l_simp_chinese", "B", "b", "localisation/mixed.yml", "imported", nil, 0, 1),
		locRow(3, "l_english", "C", "c", "localisation/mixed.yml", "imported", nil, 0, 2),
	)
	entries, _, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Count(body, "l_english:") != 2 || strings.Count(body, "l_simp_chinese:") != 1 {
		t.Fatalf("locale 头没有按行切换\n--- got ---\n%s", body)
	}
}

func TestPreservedLoc_KeepsDuplicateOccurrences(t *testing.T) {
	snap := snapWith(
		locRow(1, "l_english", "DUP", "first", "localisation/english/d_l_english.yml", "imported", nil, 0, 0),
		locRow(2, "l_english", "DUP", "second", "localisation/english/d_l_english.yml", "imported", nil, 1, 0),
	)
	entries, _, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Count(body, " DUP:") != 2 {
		t.Fatalf("重复行被吞了\n--- got ---\n%s", body)
	}
	if strings.Index(body, "first") > strings.Index(body, "second") {
		t.Fatal("行序未按 (source_order, entry_order, id)")
	}
}

func TestPreservedLoc_RejectsPathEscape(t *testing.T) {
	for _, bad := range []string{
		"../../etc/x.yml",
		"/absolute/localisation/x.yml",
		"C:/localisation/x.yml",
		"localisation/../../x.yml",
		"common/not_localisation.txt",
	} {
		snap := snapWith(locRow(1, "l_english", "K", "v", bad, "imported", nil, 0, 0))
		entries, warns, err := NativePreservedLoc(snap, "mod", true, nil)
		if err != nil {
			t.Fatalf("%s: %v", bad, err)
		}
		if len(entries) != 0 {
			t.Fatalf("%s: 逃逸路径被写进了产物: %q", bad, entries[0].Path)
		}
		if len(warns) == 0 {
			t.Fatalf("%s: 被拒但没有告警 —— 静默丢正是这条要修的病", bad)
		}
	}
}

func TestPreservedLoc_RenamesOnCollision(t *testing.T) {
	rel := "localisation/english/GER_focus_l_english.yml"
	snap := snapWith(locRow(1, "l_english", "FREEFORM", "keep", rel, "imported", nil, 0, 0))
	existing := []Entry{{
		Path:  "mod/" + rel,
		Bytes: []byte(bomUTF8 + "l_english:\n GER_other: \"structured\"\n"),
	}}
	entries, _, err := NativePreservedLoc(snap, "mod", true, existing)
	if err != nil {
		t.Fatal(err)
	}
	e := onlyEntry(t, entries)
	if e.Path != "mod/localisation/english/GER_focus_l_english_hagane_preserved.yml" {
		t.Fatalf("撞名未让路,path = %q", e.Path)
	}
	if !strings.Contains(string(e.Bytes), "FREEFORM") {
		t.Fatal("让路后内容丢了")
	}
}

func TestPreservedLoc_EmptyTableIsNoOp(t *testing.T) {
	entries, warns, err := NativePreservedLoc(Snapshot{}, "mod", true, nil)
	if err != nil || len(entries) != 0 || len(warns) != 0 {
		t.Fatalf("空表应当完全 no-op: entries=%d warns=%v err=%v", len(entries), warns, err)
	}
}

func TestPreservedLoc_SkipsIllegalKeysAndWarns(t *testing.T) {
	rel := "localisation/english/x_l_english.yml"
	snap := snapWith(
		locRow(1, "l_english", "GOOD_BEFORE", "before", rel, "imported", nil, 0, 0),
		locRow(2, "l_english", "BAD KEY WITH SPACES", "boom", rel, "imported", nil, 0, 1),
		locRow(3, "l_english", "BAD_DIACRITIC_ö", "boom", rel, "imported", nil, 0, 2),
		locRow(4, "l_english", "GOOD_AFTER", "after", rel, "imported", nil, 0, 3),
	)
	entries, warns, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Contains(body, "BAD KEY") || strings.Contains(body, "ö") {
		t.Fatalf("非法 key 进了产物,会废掉它后面的所有行\n--- got ---\n%s", body)
	}

	if !strings.Contains(body, "GOOD_BEFORE") || !strings.Contains(body, "GOOD_AFTER") {
		t.Fatalf("合法行被连带丢掉了\n--- got ---\n%s", body)
	}
	if len(warns) == 0 {
		t.Fatal("跳过了却不告警 —— 静默丢正是这条要修的病")
	}
	if warning := english.Render(warns[0]); !strings.Contains(warning, "keys left out (2)") {
		t.Fatalf("告警没说清跳了几条: %q", warning)
	}
}

func TestPreservedLoc_EscapingGoesThroughSingleSource(t *testing.T) {
	snap := snapWith(locRow(1, "l_english", "K", "line1\\nline2",
		"localisation/english/e_l_english.yml", "imported", nil, 0, 0))
	entries, _, err := NativePreservedLoc(snap, "mod", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(onlyEntry(t, entries).Bytes)
	if strings.Contains(body, "\\\\n") {
		t.Fatalf("作者写的 \\n 被翻倍成 \\\\n\n--- got ---\n%s", body)
	}
}

func TestPreservedLoc_WorldMapNamesInEachExportLanguage(t *testing.T) {
	const enFile = "localisation/english/replace/zz_game_view_l_english.yml"
	const zhFile = "localisation/simp_chinese/replace/zz_game_view_l_simp_chinese.yml"
	snap := snapWith(
		locRow(1, "l_english", "STATE_64", "Mark Brandenburg", enFile, "created", nil, 0, 0),
		locRow(2, "l_english", "VICTORY_POINTS_6521", "Neu-Berlin", enFile, "created", nil, 0, 0),
		locRow(3, "l_english", "STRATEGICREGION_8", "Ostmark", enFile, "created", nil, 0, 0),
		locRow(4, "l_simp_chinese", "STATE_64", "勃兰登堡边区", zhFile, "created", nil, 0, 0),
	)
	entries, warns, err := NativePreservedLoc(snap, "mod", false, nil)
	if err != nil || len(warns) != 0 {
		t.Fatalf("err %v warns %v", err, warns)
	}
	byPath := map[string]string{}
	for _, e := range entries {
		byPath[e.Path] = string(e.Bytes)
	}
	en, zh := byPath["mod/"+enFile], byPath["mod/"+zhFile]
	if len(byPath) != 2 || en == "" || zh == "" {
		t.Fatalf("files: %v", byPath)
	}
	for _, want := range []string{bomUTF8 + "l_english:\n", " STATE_64: \"Mark Brandenburg\"\n", " VICTORY_POINTS_6521: \"Neu-Berlin\"\n", " STRATEGICREGION_8: \"Ostmark\"\n"} {
		if !strings.Contains(en, want) {
			t.Errorf("English file missing %q:\n%s", want, en)
		}
	}
	if !strings.HasPrefix(zh, bomUTF8+"l_simp_chinese:\n") || !strings.Contains(zh, " STATE_64: \"勃兰登堡边区\"\n") || strings.Contains(zh, "Brandenburg") {
		t.Errorf("Chinese file:\n%s", zh)
	}
}
