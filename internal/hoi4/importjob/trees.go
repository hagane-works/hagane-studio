package importjob

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	treeparser "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type TreeEntityProgress struct {
	FilesDone, FilesTotal int
	Rows                  int64
}

type TreeEntityCounts struct {
	FocusTrees   int64 `json:"focus_trees"`
	FocusNodes   int64 `json:"focus_nodes"`
	Technologies int64 `json:"technologies"`
	YearColumns  int64 `json:"tech_year_columns"`
}

type TreeStageOptions struct {
	ProjectID   int64
	CountryTag  string
	ImportFocus bool
	ImportTech  bool
}
type TreeMergeOptions struct {
	ConflictPolicy string
}

type treeStage struct {
	staged bool
	trees  []treeStagedTree
	techs  []treeStagedTech
	years  []treeYearColumn
}

type treeStagedTree struct {
	treeID string
	data   payload
}

type treeStagedTech struct {
	techID string
	data   payload
}

type treeYearColumn struct {
	name string
	row  int64
}

func (s *session) stageTrees(options TreeStageOptions, progress func(TreeEntityProgress)) (TreeEntityCounts, error) {
	var counts TreeEntityCounts
	st := &s.trees
	*st = treeStage{staged: true}
	files, err := treeEntityFiles(s.root)
	if err != nil {
		return counts, err
	}

	st.years = treeProjectYearColumns(s.record["tech_year_columns"])

	countryTag, _ := s.record["country_tag"].(string)
	if explicit := strings.ToUpper(strings.TrimSpace(options.CountryTag)); explicit != "" {
		countryTag = explicit
	}
	allowed := map[string]bool{}
	for _, tag := range append(treeStrings(s.record["country_tags"]), countryTag) {
		if tag = strings.ToUpper(strings.TrimSpace(tag)); tag != "" {
			allowed[tag] = true
		}
	}

	type treeFile struct {
		rel     string
		trees   []treeStagedTree
		techs   []treeStagedTech
		years   []treeYearColumn
		invalid error

		notLoaded bool
	}
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (treeFile, error) {
		path := files[i]
		rel, _ := filepath.Rel(s.root, path)
		rel = filepath.ToSlash(rel)
		file := treeFile{rel: rel}
		lower := strings.ToLower(rel)
		if options.ImportFocus && !focusFileLoaded(lower) {
			file.notLoaded = true
			return file, nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return treeFile{}, fmt.Errorf("read %s: %w", rel, err)
		}
		switch {
		case options.ImportFocus && strings.HasPrefix(lower, "common/national_focus/"):
			bundles, _, e := treeparser.ImportFocusFile(decodeScriptBytes(body), rel)
			if e != nil {
				file.invalid = e
				return file, nil
			}
			for _, bundle := range bundles {
				raw, keep, e := focusBundlePayload(bundle, rel, countryTag, allowed)
				if e != nil {
					return treeFile{}, fmt.Errorf("encode %s: %w", rel, e)
				}
				if !keep {
					continue
				}
				id, _ := bundle.TreeData["tree_id"].(string)
				file.trees = append(file.trees, treeStagedTree{treeID: id, data: decodePayload(raw)})
			}
		case options.ImportTech && strings.HasPrefix(lower, "common/technologies/"):
			if strings.EqualFold(filepath.Base(path), "special_projects_tech.txt") {
				break
			}
			line, folder := guessTechLine(filepath.Base(path))
			techs, years, _, e := treeparser.ImportTechnologies(decodeScriptBytes(body), line, folder)
			if e != nil {
				file.invalid = e
				return file, nil
			}
			for _, tech := range techs {
				id, _ := tech["tech_id"].(string)
				if id == "" {
					continue
				}
				tech["source_file"] = rel
				raw, e := marshalPGJSON(jsonValue(tech))
				if e != nil {
					return treeFile{}, e
				}
				file.techs = append(file.techs, treeStagedTech{techID: id, data: decodePayload(raw)})
			}
			for _, y := range years {
				if y.Row.IsInt64() {
					file.years = append(file.years, treeYearColumn{name: y.Name, row: y.Row.Int64()})
				}
			}
		}
		return file, nil
	}, func(i int, file treeFile) error {
		if file.invalid != nil {
			s.warnParse(file.rel, file.invalid)
			return nil
		}
		if file.notLoaded {
			s.warn(importsvc.CodedWarning("not_loaded_by_game",
				[]importsvc.WarnParam{{Key: "file", Value: file.rel}, {Key: "folder", Value: "common/national_focus"}},
				file.rel+" was not imported: the game reads only the files directly in common/national_focus/, not those in its subfolders."))
		}
		st.trees = append(st.trees, file.trees...)
		st.techs = append(st.techs, file.techs...)
		st.years = append(st.years, file.years...)
		total += int64(len(file.trees) + len(file.techs))
		if progress != nil {
			progress(TreeEntityProgress{i + 1, len(files), total})
		}
		return nil
	})
	if err != nil {
		return counts, err
	}

	st.trees = lastWins(st.trees, func(t treeStagedTree) string { return t.treeID })
	st.techs = lastWins(st.techs, func(t treeStagedTech) string { return t.techID })
	st.years = lastWins(st.years, func(y treeYearColumn) string { return y.name })

	counts.FocusTrees = int64(len(st.trees))
	for _, tree := range st.trees {
		counts.FocusNodes += int64(len(treeNodes(tree.data)))
	}
	counts.Technologies = int64(len(st.techs))
	counts.YearColumns = int64(len(st.years))

	for _, tree := range st.trees {
		for _, node := range treeNodes(tree.data) {
			if icon := node.str("icon"); icon != "" {
				s.needImage(s.gfx[icon])
			}
		}
	}
	for _, tech := range st.techs {
		s.needImage(s.gfx["GFX_"+tech.techID+"_medium"])
	}
	return counts, nil
}

func (s *session) mergeTrees(tx *project.Tx, options TreeMergeOptions) error {
	policy := strings.ToLower(strings.TrimSpace(options.ConflictPolicy))
	if policy == "" {
		policy = "overwrite"
	}
	if policy != "overwrite" && policy != "skip" {
		return fmt.Errorf("unsupported tree conflict policy %q", options.ConflictPolicy)
	}
	st := &s.trees
	if !st.staged {

		return nil
	}
	trees := newKeyedTable(tx, "focus_trees", "tree_id")
	techs := newKeyedTable(tx, "technologies", "tech_id")
	if policy == "skip" {

		keptTrees := st.trees[:0]
		for _, tree := range st.trees {
			if _, exists := trees.find(tree.treeID); !exists {
				keptTrees = append(keptTrees, tree)
			}
		}
		st.trees = keptTrees
		keptTechs := st.techs[:0]
		for _, tech := range st.techs {
			if _, exists := techs.find(tech.techID); !exists {
				keptTechs = append(keptTechs, tech)
			}
		}
		st.techs = keptTechs
	} else {

		replaced := map[int64]bool{}
		for _, tree := range st.trees {
			if id, ok := trees.find(tree.treeID); ok {
				replaced[id] = true
			}
		}
		if len(replaced) > 0 {
			tx.Delete("focus_nodes", func(row project.Row) bool { return replaced[project.Int64(row["focus_tree_id"])] })
		}
	}

	treeUpdate := []string{"country_tag", "is_default", "tree_kind", "shared_focus_refs", "source_file", "continuous_focus_position_x", "continuous_focus_position_y", "initial_show_position", "initial_show_focus", "scope_kind", "shared_country_tags", "source", "raw_extras"}
	treePKs := make(map[string]int64, len(st.trees))
	for _, tree := range st.trees {
		stored, _ := trees.upsert(treeFocusTreeRow(tree), treeUpdate)
		treePKs[tree.treeID] = project.Int64(stored["id"])
	}

	for _, tree := range st.trees {
		seen := map[string]bool{}
		for _, node := range treeNodes(tree.data) {
			if node == nil {
				continue
			}
			if key, err := json.Marshal(node); err == nil {
				if seen[string(key)] {
					continue
				}
				seen[string(key)] = true
			}
			tx.Insert("focus_nodes", s.treeFocusNodeRow(treePKs[tree.treeID], node))
		}
	}

	treePruneSharedPools(tx, st.trees)

	if policy == "overwrite" {

		staged := make(map[string]bool, len(st.techs))
		for _, tech := range st.techs {
			staged[tech.techID] = true
		}
		if len(staged) > 0 {
			tx.Delete("technologies", func(row project.Row) bool {
				id, _ := row["tech_id"].(string)
				return staged[id]
			})
		}
	}

	for _, tech := range st.techs {
		tx.Insert("technologies", s.treeTechnologyRow(tech))
	}

	years := append([]treeYearColumn(nil), st.years...)
	sort.SliceStable(years, func(i, j int) bool { return years[i].row < years[j].row })
	columns := make([]any, 0, len(years))
	for _, y := range years {
		columns = append(columns, []any{y.name, y.row})
	}
	tx.SetRecord(func(record project.Row) { record["tech_year_columns"] = columns })

	return s.persistImportedFocusFilePolicies(tx)
}

func treeFocusTreeRow(tree treeStagedTree) project.Row {
	p := tree.data
	var countryTag any
	if tag := p.str("country_tag"); tag != "" {
		countryTag = tag
	}
	return project.Row{
		"tree_id": tree.treeID, "country_tag": countryTag,
		"is_default": p.boolOr("is_default", false), "tree_kind": p.strOr("tree_kind", "primary"),
		"shared_focus_refs": p.jsonOr("shared_focus_refs", []any{}), "source_file": p.str("source_file"),
		"continuous_focus_position_x": p.intOr("continuous_focus_position_x", 50),
		"continuous_focus_position_y": p.intOr("continuous_focus_position_y", 1000),
		"initial_show_position":       p.jsonOr("initial_show_position", map[string]any{}),
		"initial_show_focus":          p.str("initial_show_focus"), "scope_kind": p.strOr("scope_kind", "own"),
		"shared_country_tags": p["shared_country_tags"], "source": "imported",
		"raw_extras": p.jsonOr("raw_extras", map[string]any{}),
	}
}

func (s *session) treeFocusNodeRow(treePK int64, n payload) project.Row {
	focusID := n.str("focus_id")
	names := s.locNamesOrEmpty(focusID)
	descriptions := s.locNamesOrEmpty(focusID + "_desc")

	nameEn := n.nullableText("focus_id")
	if text, ok := s.locText(focusID, "en"); ok {
		nameEn = text
	}
	nameZh := s.locStr(focusID, "zh")
	descEn := s.locStr(focusID+"_desc", "en")
	descZh := s.locStr(focusID+"_desc", "zh")
	snapshot := make(map[string]any, len(n)+4)
	for key, value := range n {
		snapshot[key] = value
	}
	snapshot["name_en"], snapshot["name_zh"], snapshot["desc_en"], snapshot["desc_zh"] = nameEn, nameZh, descEn, descZh
	return project.Row{
		"focus_tree_id": treePK, "focus_id": focusID, "name": n.strOr("name", focusID),
		"icon": n.strOr("icon", "GFX_goal_unknown"), "cost": treeFloatOr(n, "cost", 10),
		"x": n.intOr("x", 0), "y": n.intOr("y", 0), "relative_position_id": n.str("relative_position_id"),
		"prerequisites": n.jsonOr("prerequisites", []any{}), "mutually_exclusive": n.jsonOr("mutually_exclusive", []any{}),
		"completion_reward": n.str("completion_reward"), "available": n.str("available"), "bypass": n.str("bypass"),
		"cancel": n.str("cancel"), "select_effect": n.str("select_effect"), "ai_will_do": n.strOr("ai_will_do", "factor = 1"),
		"search_filters":    n.jsonOr("search_filters", []any{}),
		"cancel_if_invalid": n.boolOr("cancel_if_invalid", true), "continue_if_invalid": n.boolOr("continue_if_invalid", false),
		"available_if_capitulated": n.boolOr("available_if_capitulated", false),
		"will_lead_to_war_with":    n.jsonOr("will_lead_to_war_with", []any{}),
		"allow_branch_ast":         n.jsonOr("allow_branch_ast", map[string]any{}),
		"load_focus_tree_rules":    n.jsonOr("load_focus_tree_rules", []any{}),
		"raw_extras":               n.jsonOr("raw_extras", map[string]any{}),
		"name_en":                  nameEn, "name_zh": nameZh, "desc_en": descEn, "desc_zh": descZh,
		"display_name_locs": names, "description_locs": descriptions, "effect_tooltip_locs": map[string]any{},
		"source": "imported", "source_snapshot": snapshot,
		"offset": n.jsonOr("offset", []any{}), "dynamic_icon": n.jsonOr("dynamic_icon", []any{}),
		"text_icon": n.str("text_icon"),
	}
}

func (s *session) treeTechnologyRow(tech treeStagedTech) project.Row {
	p, id := tech.data, tech.techID
	names := s.locNamesOrEmpty(id)
	descriptions := s.locNamesOrEmpty(id + "_desc")

	name := id
	if text, ok := s.locText(id, "en"); ok {
		name = text
	}
	return project.Row{
		"country_tag": "", "tech_id": id, "name": name, "tech_line": p.str("tech_line"),
		"research_cost": treeFloatOr(p, "research_cost", 1), "start_year": p.intOr("start_year", 1936),
		"folder": p.str("folder"), "position_x": treeFloatOr(p, "position_x", 0), "position_y": treeFloatOr(p, "position_y", 0),
		"position_y_is_explicit": treeNullableBool(p, "position_y_is_explicit"),
		"folder_placements":      p["folder_placements"],
		"prerequisites":          p.jsonOr("prerequisites", []any{}), "categories": p.jsonOr("categories", []any{}),
		"modifiers": p.jsonOr("modifiers", map[string]any{}), "ai_will_do": p.strOr("ai_will_do", "factor = 1"),
		"raw_extras": p.jsonOr("raw_extras", map[string]any{}), "source": "imported", "is_banner": false,
		"force_use_small_tech_layout": p.boolOr("force_use_small_tech_layout", false),
		"name_en":                     name, "name_zh": s.locStr(id, "zh"),
		"desc_en": s.locStr(id+"_desc", "en"), "desc_zh": s.locStr(id+"_desc", "zh"),
		"display_name_locs": names, "description_locs": descriptions,
		"enable_equipments": p.jsonOr("enable_equipments", []any{}), "enable_subunits": p.jsonOr("enable_subunits", []any{}),
		"enable_equipment_modules": p.jsonOr("enable_equipment_modules", []any{}),
		"enable_building":          p.str("enable_building"), "enable_tactic": p.str("enable_tactic"),
		"xor": p.jsonOr("xor", []any{}), "dependencies": p.jsonOr("dependencies", map[string]any{}),
		"sub_technologies": p.jsonOr("sub_technologies", []any{}), "doctrine": p.boolOr("doctrine", false),
		"allow": p.str("allow"), "allow_branch": p.str("allow_branch"),
		"on_research_complete": p.str("on_research_complete"), "on_research_complete_limit": p.str("on_research_complete_limit"),
		"path": p.jsonOr("path", []any{}), "xp_research_type": p.str("xp_research_type"),
		"xp_boost_cost": p.intOr("xp_boost_cost", 0), "xp_research_bonus": p.jsonOr("xp_research_bonus", map[string]any{}),
		"has_obs": false, "source_file": p.str("source_file"),
	}
}

func treePruneSharedPools(tx *project.Tx, staged []treeStagedTree) {
	touched := make(map[string]bool, len(staged))
	for _, tree := range staged {
		touched[tree.treeID] = true
	}
	nodesByTree := map[int64][]project.Row{}
	for _, node := range tx.Rows("focus_nodes") {
		id := project.Int64(node["focus_tree_id"])
		nodesByTree[id] = append(nodesByTree[id], node)
	}
	reachable := map[string]bool{}
	for _, tree := range tx.Rows("focus_trees") {
		nodes := nodesByTree[project.Int64(tree["id"])]
		if tree["tree_kind"] != "primary" || len(nodes) == 0 {
			continue
		}
		for _, ref := range treeList(tree["shared_focus_refs"]) {
			if object, ok := ref.(map[string]any); ok {
				if id, ok := object["focus_id"].(string); ok {
					reachable[id] = true
				}
			}
		}
		for _, node := range nodes {
			for _, group := range treeList(node["prerequisites"]) {
				for _, id := range treeList(group) {
					if id, ok := id.(string); ok {
						reachable[id] = true
					}
				}
			}
		}
	}
	pruned := map[int64]bool{}
	for _, tree := range tx.Rows("focus_trees") {
		treeID, _ := tree["tree_id"].(string)
		if tree["tree_kind"] != "shared_pool" || !touched[treeID] {
			continue
		}
		pk := project.Int64(tree["id"])
		keep := false
		for _, node := range nodesByTree[pk] {
			if id, ok := node["focus_id"].(string); ok && reachable[id] {
				keep = true
				break
			}
		}
		if !keep {
			pruned[pk] = true
		}
	}
	if len(pruned) == 0 {
		return
	}
	tx.Delete("focus_trees", func(row project.Row) bool { return pruned[project.Int64(row["id"])] })
	tx.Delete("focus_nodes", func(row project.Row) bool { return pruned[project.Int64(row["focus_tree_id"])] })
}

func focusFileLoaded(lower string) bool {
	rest, ok := strings.CutPrefix(lower, "common/national_focus/")
	return !ok || !strings.Contains(rest, "/")
}

func treeEntityFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		n := strings.ToLower(filepath.ToSlash(rel))
		if strings.HasSuffix(n, ".txt") && (strings.HasPrefix(n, "common/national_focus/") || strings.HasPrefix(n, "common/technologies/")) {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func focusBundlePayload(b *importsvc.TreeBundle, rel, projectTag string, allowed map[string]bool) ([]byte, bool, error) {
	td := b.TreeData
	id, _ := td["tree_id"].(string)
	if id == "" {
		return nil, false, nil
	}
	tag, scope := "", "own"
	var shared any
	if b.TreeKind == "primary" {
		verdict, _ := td["_scope_verdict"].(importsvc.ScopeVerdict)
		tags := verdict.TagList()
		switch string(verdict.Kind) {
		case "own":
			if len(tags) > 0 {
				tag = strings.ToUpper(tags[0])
				if len(allowed) > 0 && !allowed[tag] {
					return nil, false, nil
				}
			}
		case "explicit":
			scope = "explicit"
			shared = tags
			if len(allowed) > 0 {
				hit := false
				for _, t := range tags {
					if allowed[strings.ToUpper(t)] {
						hit = true
					}
				}
				if !hit {
					return nil, false, nil
				}
			}
		case "global_except":
			scope = "global_except"
			shared = tags
			for _, t := range tags {
				if strings.EqualFold(t, projectTag) {
					return nil, false, nil
				}
			}
		default:
			scope = "global"
		}
	}
	if tag == "" {
		p := strings.SplitN(id, "_", 2)[0]
		if len(p) >= 2 && len(p) <= 3 && p == strings.ToUpper(p) && (len(allowed) == 0 || allowed[p]) {
			tag = p
		}
	}
	nodes := make([]any, 0, len(b.NodesData))
	for _, n := range b.NodesData {
		nodes = append(nodes, jsonValue(n))
	}
	p := map[string]any{"tree_id": id, "country_tag": tag, "scope_kind": scope, "shared_country_tags": shared, "is_default": td["is_default"], "tree_kind": b.TreeKind, "shared_focus_refs": jsonValue(b.SharedFocusRefs), "source_file": rel, "continuous_focus_position_x": jsonValue(td["continuous_focus_position_x"]), "continuous_focus_position_y": jsonValue(td["continuous_focus_position_y"]), "initial_show_position": jsonValue(td["initial_show_position"]), "initial_show_focus": td["initial_show_focus"], "raw_extras": jsonValue(td["raw_extras"]), "nodes": nodes}
	raw, e := marshalPGJSON(p)
	return raw, true, e
}

func jsonValue(v any) any {
	switch x := v.(type) {
	case *treeparser.RawDict:
		m := map[string]any{}
		for _, k := range x.Keys {
			z, _ := x.Get(k)
			m[k] = jsonValue(z)
		}
		return m
	case paradox.Scalar:
		switch x.Kind {
		case paradox.ScalarInt:
			return jsonValue(x.Int)
		case paradox.ScalarFloat:
			return x.Float
		default:
			return x.Str
		}
	case *big.Int:
		if x.IsInt64() {
			return x.Int64()
		}
		return x.String()
	case map[string]any:
		m := map[string]any{}
		for k, z := range x {
			m[k] = jsonValue(z)
		}
		return m
	case []map[string]any:
		a := make([]any, len(x))
		for i, z := range x {
			a[i] = jsonValue(z)
		}
		return a
	case []any:
		a := make([]any, len(x))
		for i, z := range x {
			a[i] = jsonValue(z)
		}
		return a
	default:
		return v
	}
}

func guessTechLine(filename string) (string, string) {
	stem := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	m := map[string][2]string{"infantry": {"infantry", "infantry_folder"}, "support": {"support", "support_folder"}, "armor": {"armor", "armour_folder"}, "nsb_armor": {"nsb_armor", "nsb_armour_folder"}, "air_techs": {"air_techs", "air_techs_folder"}, "bba_air_techs": {"bba_air_techs", "bba_air_techs_folder"}, "naval": {"naval", "naval_folder"}, "mtg_naval": {"mtg_naval", "mtgnavalfolder"}, "mtg_naval_support": {"naval_support", "mtgnavalsupportfolder"}, "electronic_mechanical_engineering": {"electronic_mechanical_engineering", "electronic_mechanical_engineering_folder"}, "industry": {"industry", "industry_folder"}, "artillery": {"artillery", "artillery_folder"}, "special_forces_doctrine": {"special_forces_doctrine", "special_forces_doctrine_folder"}}
	if v, ok := m[stem]; ok {
		return v[0], v[1]
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		if strings.Contains(stem, k) {
			return m[k][0], m[k][1]
		}
	}
	return stem, stem + "_folder"
}

func treeNodes(p payload) []payload {
	list := treeList(p["nodes"])
	out := make([]payload, len(list))
	for i, item := range list {
		if node, ok := item.(map[string]any); ok {
			out[i] = node
		}
	}
	return out
}

func treeList(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case nil, string, bool, json.Number, float64, int, int64, map[string]any:
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out []any
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func treeStrings(value any) []string {
	var out []string
	for _, item := range treeList(value) {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func treeProjectYearColumns(value any) []treeYearColumn {
	var out []treeYearColumn
	for _, entry := range treeList(value) {
		pair := treeList(entry)
		if len(pair) != 2 {
			continue
		}
		p := payload{"name": pair[0], "row": pair[1]}
		name, ok := p.text("name")
		row, rowOK := p.int("row")
		if ok && rowOK {
			out = append(out, treeYearColumn{name: name, row: row})
		}
	}
	return out
}

func treeFloatOr(p payload, key string, fallback float64) float64 {
	if f, ok := p.float(key); ok {
		return f
	}
	return fallback
}

func treeNullableBool(p payload, key string) any {
	text, ok := p.text(key)
	if !ok || strings.TrimSpace(text) == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "t", "yes", "y", "on", "1":
		return true
	case "false", "f", "no", "n", "off", "0":
		return false
	}
	return nil
}
