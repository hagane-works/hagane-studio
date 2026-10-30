package lint

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	playtestStartDateRe = regexp.MustCompile(`Start-date:\s*([0-9]{3,4}(?:\.[0-9]+){0,3})`)
	playtestGameDateRe  = regexp.MustCompile(`\]\[([0-9]{4}\.[0-9.]+)\]\[`)
	playtestExcTypeRe   = regexp.MustCompile(`(?i)Unhandled Exception\s+(\S+)\s*\(([^)]+)\)`)
	playtestExcVerRe    = regexp.MustCompile(`(?m)^Version:\s*(.+)$`)
)

func PlaytestLeadingYear(dateStr string) *int {
	s := strings.TrimSpace(dateStr)
	if s == "" {
		return nil
	}
	head, _, _ := strings.Cut(s, ".")
	v, err := strconv.Atoi(head)
	if err != nil {
		return nil
	}
	return &v
}

func PlaytestLoadedStartDate(gameLog string) string {
	if m := playtestStartDateRe.FindStringSubmatch(gameLog); m != nil {
		return m[1]
	}
	var order []string
	counts := map[string]int{}
	for _, m := range playtestGameDateRe.FindAllStringSubmatch(gameLog, -1) {
		s := m[1]
		if strings.HasPrefix(s, "0") {
			continue
		}
		if _, seen := counts[s]; !seen {
			order = append(order, s)
		}
		counts[s]++
	}
	best, bestN := "", -1
	for _, s := range order {
		if counts[s] > bestN {
			best, bestN = s, counts[s]
		}
	}
	return best
}

type RuntimeInventoryEntry struct {
	Name           string
	Path           string
	BuildID        string
	Active         *bool
	InventoryIndex int
}

func (e RuntimeInventoryEntry) identityKey() [2]string {
	return [2]string{runtimeNormaliseName(e.Name), runtimeNormalisePath(e.Path)}
}

func (e RuntimeInventoryEntry) normalisedBuildID() string {
	return strings.ToLower(strings.TrimSpace(e.BuildID))
}

func (e RuntimeInventoryEntry) toMap() map[string]any {
	m := map[string]any{
		"name": e.Name, "path": e.Path,
		"inventory_index": e.InventoryIndex,
	}
	if e.BuildID == "" {
		m["build_id"] = nil
	} else {
		m["build_id"] = e.BuildID
	}
	if e.Active == nil {
		m["active"] = nil
	} else {
		m["active"] = *e.Active
	}
	return m
}

type RuntimeExpectedIdentity struct {
	Name    string
	Path    string
	BuildID string
}

type RuntimeInputError struct{ msg string }

func (e *RuntimeInputError) Error() string { return e.msg }

func runtimeNormaliseName(v string) string {
	return strings.ToLower(strings.Join(strings.Fields(v), " "))
}

func runtimeNormalisePath(v string) string {
	raw := strings.Trim(strings.TrimSpace(v), "\"'")
	raw = strings.ReplaceAll(raw, "\\", "/")
	if raw == "" {
		return "."
	}
	return strings.ToLower(path.Clean(raw))
}

var (
	activeModCountRe = regexp.MustCompile(`(?i)\bActive\s+Mod\s+Count\s*[:=]\s*(\d+)\b`)
	activeDLCCountRe = regexp.MustCompile(`(?i)\bActive\s+DLC\s+Count\s*[:=]\s*(\d+)\b`)
)

func lastIntMatch(re *regexp.Regexp, text string) *int {
	ms := re.FindAllStringSubmatch(text, -1)
	if len(ms) == 0 {
		return nil
	}
	v, err := strconv.Atoi(ms[len(ms)-1][1])
	if err != nil {
		return nil
	}
	return &v
}

var runtimeSentinelRules = []struct {
	marker, stage, confidence string
	re                        *regexp.Regexp
}{
	{"launcher_inventory", "launcher_inventory", "medium", regexp.MustCompile(`(?i)\bActive\s+Mod\s+Count\s*[:=]`)},
	{"engine_boot", "engine_boot", "low", regexp.MustCompile(`(?i)\b(?:initiali[sz]ing application|application version)\b`)},
	{"database_loading", "database_loading", "medium", regexp.MustCompile(`(?i)\b(?:loading database|loading defines|loading common database)\b`)},
	{"map_loading", "map_loading", "medium", regexp.MustCompile(`(?i)\b(?:loading map|map initiali[sz]ation)\b`)},
	{"history_loading", "history_loading", "medium", regexp.MustCompile(`(?i)\b(?:loading history|executing history)\b`)},
	{"scenario_started", "scenario_started", "high", regexp.MustCompile(`(?i)\bStart-date\s*:`)},
	{"in_game", "in_game", "high", regexp.MustCompile(`(?i)\b(?:game setup finished|entered game|game started)\b`)},
}

type runtimeSentinel struct {
	source     string
	lineNumber int
	marker     string
	stage      string
	confidence string
	excerpt    string
}

func (s runtimeSentinel) toMap() map[string]any {
	return map[string]any{
		"source": s.source, "line_number": s.lineNumber, "marker": s.marker,
		"stage": s.stage, "confidence": s.confidence, "excerpt": s.excerpt,
	}
}

func diagnoseCrashStage(systemLog, gameLog string) map[string]any {
	var last *runtimeSentinel
	lastObserved := ""
	haveObserved := false
	for _, src := range []struct{ name, text string }{{"system.log", systemLog}, {"game.log", gameLog}} {
		if src.text == "" {
			continue
		}
		for i, line := range strings.Split(src.text, "\n") {

			stripped := strings.TrimSpace(line)
			if stripped != "" {
				lastObserved, haveObserved = stripped, true
			}
			for _, rule := range runtimeSentinelRules {
				if rule.re.MatchString(line) {
					last = &runtimeSentinel{
						source: src.name, lineNumber: i + 1, marker: rule.marker,
						stage: rule.stage, confidence: rule.confidence, excerpt: stripped,
					}
				}
			}
		}
	}
	var observed any
	if haveObserved {
		observed = lastObserved
	}
	out := map[string]any{
		"last_observed_line":               observed,
		"last_observed_line_is_root_cause": false,
		"root_cause":                       nil,
		"caveat":                           "The last observed line is chronology evidence, not proof of the crash root cause.",
	}
	if last == nil {
		out["stage"], out["confidence"], out["last_sentinel"] = "unknown", "low", nil
		return out
	}
	out["stage"], out["confidence"], out["last_sentinel"] = last.stage, last.confidence, last.toMap()
	return out
}

func parseDescriptorInventory(text string) ([]RuntimeInventoryEntry, error) {
	var decoded any
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return nil, &RuntimeInputError{"descriptor inventory is not valid JSON: " + jsonDecodeMsg(err)}
	}
	if obj, ok := decoded.(map[string]any); ok {
		found := false
		for _, key := range []string{"descriptors", "mods", "inventory"} {
			if v, ok := obj[key]; ok {
				decoded, found = v, true
				break
			}
		}
		if !found {
			return nil, &RuntimeInputError{
				"descriptor inventory object must contain descriptors, mods, or inventory"}
		}
	}
	if _, isStr := decoded.(string); isStr {
		return nil, &RuntimeInputError{"descriptor inventory must be a JSON array"}
	}
	arr, ok := decoded.([]any)
	if !ok {
		return nil, &RuntimeInputError{"descriptor inventory must be a JSON array"}
	}
	entries := make([]RuntimeInventoryEntry, 0, len(arr))
	for i, raw := range arr {
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil, &RuntimeInputError{fmt.Sprintf("descriptor inventory entry %d must be an object", i)}
		}
		name := runtimeFirstText(obj, "name", "display_name", "mod_name")
		p := runtimeFirstText(obj, "path", "descriptor_path", "mod_path")
		if name == "" || p == "" {
			return nil, &RuntimeInputError{fmt.Sprintf("descriptor inventory entry %d requires name and path", i)}
		}
		active, hasActive := obj["active"]
		if !hasActive {
			active = obj["enabled"]
		}
		entries = append(entries, RuntimeInventoryEntry{
			Name: name, Path: p,
			BuildID:        runtimeFirstText(obj, "build_id", "hagane_build_id"),
			Active:         runtimeOptionalBool(active),
			InventoryIndex: i,
		})
	}
	return entries, nil
}

func jsonDecodeMsg(err error) string { return err.Error() }

func runtimeFirstText(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if s := strings.TrimSpace(pyStr(v)); s != "" {
			return s
		}
	}
	return ""
}

func runtimeOptionalBool(v any) *bool {
	if v == nil {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
	}
	if n, ok := v.(json.Number); ok {
		if n.String() == "0" || n.String() == "1" {
			b := n.String() == "1"
			return &b
		}
	}
	token := strings.ToLower(strings.TrimSpace(pyStr(v)))
	switch token {
	case "true", "yes", "1", "active", "enabled":
		b := true
		return &b
	case "false", "no", "0", "inactive", "disabled":
		b := false
		return &b
	}
	return nil
}

func pyStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	case json.Number:
		s := t.String()
		if !strings.ContainsAny(s, ".eE") {
			return s
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return s
		}
		if f == float64(int64(f)) {
			return strconv.FormatFloat(f, 'f', 1, 64)
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func diagnoseExpectedMod(expected *RuntimeExpectedIdentity, inventory []RuntimeInventoryEntry,
	activeModCount *int, inventorySupplied bool) map[string]any {

	mk := func(status, reason string, matched, collision []RuntimeInventoryEntry) map[string]any {
		var exp any
		if expected != nil {
			var bid any
			if expected.BuildID != "" {
				bid = expected.BuildID
			}
			exp = map[string]any{"name": expected.Name, "path": expected.Path, "build_id": bid}
		}
		toList := func(es []RuntimeInventoryEntry) []map[string]any {
			out := make([]map[string]any, 0, len(es))
			for _, e := range es {
				out = append(out, e.toMap())
			}
			return out
		}
		return map[string]any{
			"status": status, "reason": reason,
			"collision_proven": status == "identity_collision",
			"expected":         exp,
			"matched_entries":  toList(matched), "collision_entries": toList(collision),
		}
	}
	if expected == nil {
		return mk("not_requested", "No expected mod identity was supplied.", nil, nil)
	}
	key := [2]string{runtimeNormaliseName(expected.Name), runtimeNormalisePath(expected.Path)}
	var sameNamePath []RuntimeInventoryEntry
	for _, e := range inventory {
		if e.identityKey() == key {
			sameNamePath = append(sameNamePath, e)
		}
	}
	if len(sameNamePath) > 1 {
		return mk("identity_collision",
			"Launcher inventory proves multiple descriptors with the same normalised mod name and descriptor path.",
			sameNamePath, sameNamePath)
	}
	expectedBuild := strings.ToLower(strings.TrimSpace(expected.BuildID))
	var exact []RuntimeInventoryEntry
	for _, e := range sameNamePath {
		if expectedBuild == "" || e.normalisedBuildID() == expectedBuild {
			exact = append(exact, e)
		}
	}
	for _, e := range exact {
		if e.Active != nil && *e.Active {
			return mk("activated", "Launcher inventory marks the expected exact identity active.", exact, nil)
		}
	}
	if activeModCount != nil && *activeModCount == 0 {
		return mk("not_activated",
			"system.log reports Active Mod Count 0; the expected mod was not activated. This is not evidence of a duplicate name.",
			exact, nil)
	}
	if len(exact) > 0 {
		allFalse := true
		for _, e := range exact {
			if e.Active == nil || *e.Active {
				allFalse = false
				break
			}
		}
		if allFalse {
			return mk("not_activated",
				"Launcher inventory explicitly marks the expected identity inactive.", exact, nil)
		}
	}
	if inventorySupplied && len(exact) == 0 {
		return mk("not_activated",
			"The supplied launcher inventory does not contain the expected exact name, path, and build identity.",
			sameNamePath, nil)
	}
	return mk("indeterminate",
		"The available evidence does not prove whether the expected exact identity was active.", exact, nil)
}

func AnalyzeRuntimeDiagnostics(systemLog, gameLog string,
	expected *RuntimeExpectedIdentity, inventoryText *string) (map[string]any, error) {

	modCount := lastIntMatch(activeModCountRe, systemLog)
	dlcCount := lastIntMatch(activeDLCCountRe, systemLog)
	var inventory []RuntimeInventoryEntry
	if inventoryText != nil {
		var err error
		inventory, err = parseDescriptorInventory(*inventoryText)
		if err != nil {
			return nil, err
		}
	}

	if expected != nil {
		expected = &RuntimeExpectedIdentity{
			Name:    strings.TrimSpace(expected.Name),
			Path:    strings.TrimSpace(expected.Path),
			BuildID: strings.TrimSpace(expected.BuildID),
		}
		if expected.Name == "" || expected.Path == "" {
			return nil, &RuntimeInputError{"expected mod identity requires name and path"}
		}
	}
	var mc, dc any
	if modCount != nil {
		mc = *modCount
	}
	if dlcCount != nil {
		dc = *dlcCount
	}
	invList := make([]map[string]any, 0, len(inventory))
	for _, e := range inventory {
		invList = append(invList, e.toMap())
	}
	return map[string]any{
		"system": map[string]any{
			"active_mod_count": mc, "active_dlc_count": dc,
			"active_dlc_count_interpretation": "environment_only",
		},
		"expected_mod":         diagnoseExpectedMod(expected, inventory, modCount, inventoryText != nil),
		"crash":                diagnoseCrashStage(systemLog, gameLog),
		"descriptor_inventory": invList,
	}, nil
}

func strPtr(s string) *string { return &s }

func analyzeGameLog(text, builtDate, bookmarkID string) []LintFinding {
	loaded := PlaytestLoadedStartDate(text)
	if loaded == "" {
		return nil
	}
	bl, ll := PlaytestLeadingYear(builtDate), PlaytestLeadingYear(loaded)
	if bl != nil && ll != nil && *bl != *ll {
		bm := bookmarkID
		if bm == "" {
			bm = "?"
		}
		var built any
		if builtDate != "" {
			built = builtDate
		}
		var bid any
		if bookmarkID != "" {
			bid = bookmarkID
		}
		return []LintFinding{{
			Severity: "error", Code: "playtest.start_date_mismatch",
			Title:  "Game loaded a different start date",
			Reason: fmt.Sprintf("You built for %s (bookmark '%s') but the game launched at %s.", builtDate, bm, loaded),
			Suggestion: `A custom start date only takes effect if your bookmark REPLACES ` +
				`vanilla. Add replace_path="common/bookmarks" to the OUTER ` +
				`<mod>.mod descriptor (not just the in-zip descriptor.mod), then ` +
				`re-export. Until then HOI4 falls back to the vanilla 1936 start.`,
			File:   strPtr("descriptor (<mod>.mod)"),
			Detail: map[string]any{"built_date": built, "loaded_date": loaded, "bookmark_id": bid},
		}}
	}
	if bl == nil {
		return []LintFinding{{
			Severity: "info", Code: "playtest.loaded_start_date",
			Title:  "Loaded start date",
			Reason: fmt.Sprintf("The game launched at %s.", loaded),
			Suggestion: "This project defines no bookmark, so the program can't verify " +
				"the intended start date. If you meant a custom start, add a " +
				"bookmark and emit it under replace_path=\"common/bookmarks\".",
			File:   strPtr("descriptor (<mod>.mod)"),
			Detail: map[string]any{"loaded_date": loaded},
		}}
	}
	var built any
	if builtDate != "" {
		built = builtDate
	}
	return []LintFinding{{
		Severity: "info", Code: "playtest.start_date_ok",
		Title:      "Start date matches",
		Reason:     fmt.Sprintf("The game launched at %s, matching your bookmark.", loaded),
		Suggestion: "",
		File:       nil,
		Detail:     map[string]any{"built_date": built, "loaded_date": loaded},
	}}
}

func analyzeDLCLoad(text string) []LintFinding {
	var decoded any
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return nil
	}
	obj, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	enabled, ok := obj["enabled_mods"].([]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(enabled))
	for _, m := range enabled {
		names = append(names, pyStr(m))
	}
	sev, orderNote := "info", ""
	if len(names) > 1 {
		sev = "warning"
		orderNote = " When multiple mods are enabled, a mod LOWER in this list overrides " +
			"files from mods above it — a load-order conflict can silently replace " +
			"your content."
	}
	joined := strings.Join(names, ", ")
	if joined == "" {
		joined = "(none)"
	}
	disabled, ok := obj["disabled_dlcs"]
	if !ok {
		disabled = []any{}
	}
	return []LintFinding{{
		Severity: sev, Code: "playtest.enabled_mods",
		Title:      fmt.Sprintf("%d mod(s) enabled", len(names)),
		Reason:     "Active mod load order: " + joined,
		Suggestion: "Confirm your mod is present and not overridden." + orderNote,
		File:       strPtr("dlc_load.json"),
		Detail:     map[string]any{"enabled_mods": names, "disabled_dlcs": disabled},
	}}
}

func analyzeException(text string) []LintFinding {
	m := playtestExcTypeRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	codeHex, kind := m[1], strings.TrimSpace(m[2])
	version, verSuffix := "", ""
	if vm := playtestExcVerRe.FindStringSubmatch(text); vm != nil {
		version = strings.TrimSpace(vm[1])
	}
	if version != "" {
		verSuffix = " (" + version + ")"
	}
	var ver any
	if version != "" {
		ver = version
	}
	return []LintFinding{{
		Severity: "error", Code: "playtest.crash",
		Title:  fmt.Sprintf("Game crashed (%s)", kind),
		Reason: fmt.Sprintf("HOI4%s hit an unhandled exception %s (%s).", verSuffix, codeHex, kind),
		Suggestion: "The exception header confirms a crash but does not identify its root " +
			"cause. Use the last recognised lifecycle sentinel to narrow the stage, " +
			"then cross-check error.log and nearby runtime evidence. The PHYSFS_* " +
			"stack offsets are not symbolized and are not actionable by themselves.",
		File:   strPtr("exception.txt"),
		Detail: map[string]any{"exception": codeHex, "kind": kind, "version": ver},
	}}
}

type PlaytestBundleInput struct {
	GameLog          *string
	ExceptionTxt     *string
	DLCLoadJSON      *string
	SystemLog        *string
	ModInventoryJSON *string
	ExpectedName     string
	ExpectedPath     string
	ExpectedBuildID  string
}

func blankOrNil(p *string) bool { return p == nil || strings.TrimSpace(*p) == "" }

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func BuildPlaytestReport(in PlaytestBundleInput, builtDate, bookmarkID string) map[string]any {
	var findings []LintFinding
	if !blankOrNil(in.GameLog) {
		findings = append(findings, analyzeGameLog(*in.GameLog, builtDate, bookmarkID)...)
	}
	if !blankOrNil(in.DLCLoadJSON) {
		findings = append(findings, analyzeDLCLoad(*in.DLCLoadJSON)...)
	}
	if !blankOrNil(in.ExceptionTxt) {
		findings = append(findings, analyzeException(*in.ExceptionTxt)...)
	}

	var expected *RuntimeExpectedIdentity
	if in.ExpectedName != "" || in.ExpectedPath != "" || in.ExpectedBuildID != "" {
		expected = &RuntimeExpectedIdentity{
			Name: in.ExpectedName, Path: in.ExpectedPath, BuildID: in.ExpectedBuildID,
		}
	}
	var inventoryText *string
	if !blankOrNil(in.ModInventoryJSON) {
		inventoryText = in.ModInventoryJSON
	}
	var diagnostics map[string]any
	var diagErr any
	if d, err := AnalyzeRuntimeDiagnostics(derefStr(in.SystemLog), derefStr(in.GameLog),
		expected, inventoryText); err != nil {
		diagErr = err.Error()
	} else {
		diagnostics = d
	}
	if diagErr != nil {
		findings = append(findings, LintFinding{
			Severity: "warning", Code: "playtest.runtime_diagnostics_input_invalid",
			Title:  "Runtime identity evidence could not be read",
			Reason: diagErr.(string),
			Suggestion: "Provide both the expected mod name and descriptor path, and use " +
				"a JSON descriptor inventory with name/path fields.",
			File:   strPtr("mod inventory / expected identity"),
			Detail: map[string]any{"input_error": diagErr},
		})
	}

	loaded := ""
	if in.GameLog != nil {
		loaded = PlaytestLoadedStartDate(*in.GameLog)
	}
	var builtOut, loadedOut, diagOut any
	if builtDate != "" {
		builtOut = builtDate
	}
	if loaded != "" {
		loadedOut = loaded
	}
	if diagnostics != nil {
		diagOut = diagnostics
	}
	return LintFindingsReport(findings, map[string]any{
		"files_present": map[string]any{
			"game_log":           !blankOrNil(in.GameLog),
			"exception_txt":      !blankOrNil(in.ExceptionTxt),
			"dlc_load_json":      !blankOrNil(in.DLCLoadJSON),
			"system_log":         !blankOrNil(in.SystemLog),
			"mod_inventory_json": !blankOrNil(in.ModInventoryJSON),
		},
		"built_date":                builtOut,
		"loaded_date":               loadedOut,
		"runtime_diagnostics":       diagOut,
		"runtime_diagnostics_error": diagErr,
	})
}
