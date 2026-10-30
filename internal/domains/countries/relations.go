package countries

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/paradox"
	"haganeworks/internal/project"
)

type rowReader interface {
	rows(table string) []project.Row
	record() project.Row
}

type projectReader struct {
	p   *project.Project
	err error
}

func (r *projectReader) rows(table string) []project.Row {
	rows, err := r.p.Rows(table)
	if err != nil && r.err == nil {
		r.err = err
	}
	return rows
}

func (r *projectReader) record() project.Row { return r.p.Record() }

type txReader struct{ tx *project.Tx }

func (r txReader) rows(table string) []project.Row { return r.tx.Rows(table) }
func (r txReader) record() project.Row             { return r.tx.Record() }

type refEdge struct {
	SourceKind string   `json:"source_kind"`
	SourceID   int64    `json:"source_id"`
	SourceName string   `json:"source_name"`
	SourceTag  *string  `json:"source_tag"`
	SourceTags []string `json:"source_tags"`
	EffectName string   `json:"effect_name"`
	TargetType string   `json:"target_type"`
	TargetID   string   `json:"target_id"`
	FieldName  string   `json:"field_name"`
	Resolved   bool     `json:"resolved"`
	TargetTag  *string  `json:"target_tag"`
}

var vanillaAutonomyStates = []string{
	"autonomy_aat_defense_council_member", "autonomy_austro_hungarian_subject",
	"autonomy_collaboration_government", "autonomy_colony", "autonomy_dominion",
	"autonomy_eu_member", "autonomy_free", "autonomy_integrated_puppet",
	"autonomy_personal_union", "autonomy_puppet", "autonomy_reichskommissariat",
	"autonomy_reichsprotectorate", "autonomy_satellite",
	"autonomy_sea_integrated_warlord_subject", "autonomy_sea_warlord_subject",
	"autonomy_supervised_state", "autonomy_volkskommissariat",
	"autonomy_wtt_imperial_associate", "autonomy_wtt_imperial_protectorate",
	"autonomy_wtt_imperial_subject",
}

type refExtractor func(e *paradox.Entry) []string

func scalarStr(e *paradox.Entry) string { return e.Scalar.StrValue() }

func extScalar(e *paradox.Entry) []string {
	if e.Block == nil {
		return []string{scalarStr(e)}
	}
	return nil
}

func extScalarOrList(e *paradox.Entry) []string {
	if e.Block == nil {
		return []string{scalarStr(e)}
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key == nil && en.Block == nil {
			out = append(out, scalarStr(en))
		}
	}
	return out
}

func extBlockWithAnyField(fields ...string) refExtractor {
	accepted := map[string]bool{}
	for _, f := range fields {
		accepted[f] = true
	}
	return func(e *paradox.Entry) []string {
		if e.Block == nil {
			return nil
		}
		var out []string
		for _, en := range e.Block.Entries {
			if en.Key != nil && accepted[*en.Key] && en.Block == nil {
				out = append(out, scalarStr(en))
			}
		}
		return out
	}
}

func extBlockWithField(field string) refExtractor { return extBlockWithAnyField(field) }

func extScalarOrBlockField(field string) refExtractor {
	blockExt := extBlockWithField(field)
	return func(e *paradox.Entry) []string {
		if e.Block == nil {
			return []string{scalarStr(e)}
		}
		return blockExt(e)
	}
}

func extSwapPair(e *paradox.Entry) []string {
	return extBlockWithAnyField("add_idea", "remove_idea", "add", "remove")(e)
}

func extBlockKeys(e *paradox.Entry) []string {
	if e.Block == nil {
		return nil
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key == nil || *en.Key == "popup" {
			continue
		}
		out = append(out, *en.Key)
	}
	return out
}

func extCreateCountryLeaderTraits(e *paradox.Entry) []string {
	if e.Block == nil {
		return nil
	}
	var out []string
	for _, en := range e.Block.Entries {
		if en.Key != nil && *en.Key == "traits" && en.Block != nil {
			for _, sub := range en.Block.Entries {
				if sub.Key == nil && sub.Block == nil {
					out = append(out, scalarStr(sub))
				}
			}
		}
	}
	return out
}

type refSpec struct {
	targetType string
	extract    refExtractor
}

var activeCrossRefs = map[string][]refSpec{
	"add_ideas":                     {{"idea", extScalarOrList}},
	"remove_ideas":                  {{"idea", extScalarOrList}},
	"swap_ideas":                    {{"idea", extSwapPair}},
	"add_timed_idea":                {{"idea", extBlockWithField("idea")}},
	"has_idea":                      {{"idea", extScalar}},
	"add_dynamic_modifier":          {{"dynamic_modifier", extBlockWithField("modifier")}},
	"remove_dynamic_modifier":       {{"dynamic_modifier", extBlockWithField("modifier")}},
	"has_dynamic_modifier":          {{"dynamic_modifier", extScalarOrBlockField("modifier")}},
	"set_autonomy":                  {{"autonomy_state", extBlockWithAnyField("autonomy_state", "autonomous_state")}},
	"has_autonomy_state":            {{"autonomy_state", extScalar}},
	"country_event":                 {{"event", extScalarOrBlockField("id")}},
	"state_event":                   {{"event", extScalarOrBlockField("id")}},
	"news_event":                    {{"event", extScalarOrBlockField("id")}},
	"unit_leader_event":             {{"event", extScalarOrBlockField("id")}},
	"complete_national_focus":       {{"focus", extScalarOrBlockField("focus")}},
	"has_completed_focus":           {{"focus", extScalar}},
	"focus_progress":                {{"focus", extBlockWithField("focus")}},
	"activate_decision":             {{"decision", extScalar}},
	"unlock_decision_tooltip":       {{"decision", extScalarOrBlockField("decision")}},
	"has_trait":                     {{"trait", extScalar}},
	"has_country_leader_with_trait": {{"trait", extScalar}},
	"add_country_leader_trait":      {{"trait", extScalarOrBlockField("trait")}},
	"remove_country_leader_trait":   {{"trait", extScalarOrBlockField("trait")}},
	"add_unit_leader_trait":         {{"trait", extScalar}},
	"remove_unit_leader_trait":      {{"trait", extScalar}},
	"swap_ruler_traits":             {{"trait", extBlockWithAnyField("add", "remove")}},
	"create_country_leader":         {{"trait", extCreateCountryLeaderTraits}},
	"add_opinion_modifier":          {{"opinion_modifier", extBlockWithField("modifier")}},
	"remove_opinion_modifier":       {{"opinion_modifier", extBlockWithField("modifier")}},
	"reverse_add_opinion_modifier":  {{"opinion_modifier", extBlockWithField("modifier")}},
	"has_opinion_modifier":          {{"opinion_modifier", extScalarOrBlockField("modifier")}},
	"has_tech":                      {{"technology", extScalar}},
	"can_research":                  {{"technology", extScalar}},
	"set_technology":                {{"technology", extBlockKeys}},
	"has_character":                 {{"character", extScalar}},
	"is_character":                  {{"character", extScalar}},
	"retire_character":              {{"character", extScalar}},
	"promote_character":             {{"character", extScalarOrBlockField("character")}},
	"remove_country_leader_role":    {{"character", extScalarOrBlockField("character")}},
	"remove_unit_leader_role":       {{"character", extScalarOrBlockField("character")}},
	"add_country_leader_role":       {{"character", extBlockWithField("character")}},
	"add_corps_commander_role":      {{"character", extBlockWithField("character")}},
	"add_trait": {
		{"character", extBlockWithField("character")},
		{"trait", extBlockWithField("trait")},
	},
	"add_mio_funds":                               {{"mio", extBlockWithField("mio")}},
	"add_mio_size_up":                             {{"mio", extBlockWithField("mio")}},
	"add_mio_design_team_assign_cost":             {{"mio", extBlockWithField("mio")}},
	"add_mio_industrial_manufacturer_assign_cost": {{"mio", extBlockWithField("mio")}},
}

type refTriple struct{ effect, targetType, targetID string }

func walkRefs(entries []*paradox.Entry, out *[]refTriple) {
	for _, e := range entries {
		if e.Key != nil {
			for _, spec := range activeCrossRefs[*e.Key] {
				for _, rid := range spec.extract(e) {
					*out = append(*out, refTriple{*e.Key, spec.targetType, rid})
				}
			}
		}
		if e.Block != nil {
			walkRefs(e.Block.Entries, out)
		}
	}
}

func HasRelationRefs(text string) bool { return len(extractRelationRefs(text)) > 0 }

func extractRelationRefs(text string) []refTriple {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	block, recoveries, err := paradox.Parse(text)
	if err != nil || block == nil {
		return nil
	}
	for _, rec := range recoveries {
		if rec.Code == paradox.RecoveryUnclosedBlock {
			return nil
		}
	}
	var out []refTriple
	walkRefs(block.Entries, &out)
	return out
}

type relProject struct {
	countryTag  string
	countryTags []string
	isSubmod    bool
}

func relLoadProject(record project.Row) relProject {
	proj := relProject{countryTag: str(record["country_tag"]), countryTags: jsonStrings(record["country_tags"])}

	switch deps := record["dependencies"].(type) {
	case []any:
		proj.isSubmod = len(deps) > 0
	case map[string]any:
		proj.isSubmod = len(deps) > 0
	case string:
		proj.isSubmod = deps != ""
	case nil:
	default:
		proj.isSubmod = true
	}
	return proj
}

func jsonStrings(value any) []string {
	list, _ := value.([]any)
	var out []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func ownTag(value any) *string {
	s, ok := value.(string)
	if !ok {
		return nil
	}
	return &s
}

func relBestTag(own *string, shared []string, proj relProject) *string {
	if own != nil && *own != "" {
		u := strings.ToUpper(*own)
		return &u
	}
	for _, t := range shared {
		if t != "" {
			u := strings.ToUpper(t)
			return &u
		}
	}
	if proj.countryTag != "" {
		u := strings.ToUpper(proj.countryTag)
		return &u
	}
	var fallback []string
	for _, t := range proj.countryTags {
		if t != "" {
			fallback = append(fallback, strings.ToUpper(t))
		}
	}
	if len(fallback) == 0 {
		return nil
	}
	sort.Strings(fallback)
	return &fallback[0]
}

func relAllTags(own *string, shared []string, proj relProject) []string {
	set := map[string]bool{}
	if own != nil && *own != "" {
		set[strings.ToUpper(*own)] = true
	}
	for _, t := range shared {
		if t != "" {
			set[strings.ToUpper(t)] = true
		}
	}
	if len(set) == 0 && proj.countryTag != "" {
		set[strings.ToUpper(proj.countryTag)] = true
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

type relScriptSource struct {
	kind       string
	sourceID   int64
	sourceName string
	fieldName  string
	text       string
	tag        *string
	tags       []string
}

func relSources(rr rowReader, proj relProject) []relScriptSource {
	var out []relScriptSource
	emit := func(kind string, row project.Row, name string, own *string, shared []string, fields []string) {
		tag := relBestTag(own, shared, proj)
		tags := relAllTags(own, shared, proj)
		for _, field := range fields {
			if text := str(row[field]); text != "" {
				out = append(out, relScriptSource{kind, project.Int64(row["id"]), name, field, text, tag, tags})
			}
		}
	}
	nodesByTree := map[int64][]project.Row{}
	for _, node := range rr.rows("focus_nodes") {
		tree := project.Int64(node["focus_tree_id"])
		nodesByTree[tree] = append(nodesByTree[tree], node)
	}
	for _, tree := range rr.rows("focus_trees") {
		own, shared := ownTag(tree["country_tag"]), jsonStrings(tree["shared_country_tags"])
		for _, node := range nodesByTree[project.Int64(tree["id"])] {
			emit("focus", node, str(node["focus_id"]), own, shared,
				[]string{"completion_reward", "select_effect", "available", "bypass", "cancel"})
		}
	}
	for _, event := range rr.rows("events") {
		own, shared := ownTag(event["country_tag"]), jsonStrings(event["shared_country_tags"])
		eventID := str(event["event_id"])
		emit("event", event, eventID, own, shared, []string{"trigger", "immediate", "after", "mean_time_to_happen"})
		tag, tags := relBestTag(own, shared, proj), relAllTags(own, shared, proj)
		options, _ := event["options"].([]any)
		for idx, item := range options {
			option, _ := item.(map[string]any)
			for _, field := range []string{"trigger", "effects", "hidden_effects"} {
				if text, ok := option[field].(string); ok && text != "" {
					out = append(out, relScriptSource{"event", project.Int64(event["id"]), eventID,
						"option[" + strconv.Itoa(idx) + "]." + field, text, tag, tags})
				}
			}
		}
	}
	for _, decision := range rr.rows("decisions") {
		emit("decision", decision, str(decision["decision_id"]), ownTag(decision["country_tag"]), jsonStrings(decision["shared_country_tags"]),
			[]string{"complete_effect", "remove_effect", "timeout_effect", "cancel_effect",
				"allowed", "available", "visible", "cancel_trigger", "activation", "ai_will_do"})
	}
	for _, idea := range rr.rows("ideas") {
		emit("idea", idea, str(idea["idea_id"]), ownTag(idea["country_tag"]), jsonStrings(idea["shared_country_tags"]),
			[]string{"on_add", "on_remove", "do_effect", "allowed", "available", "visible",
				"allowed_civil_war", "allowed_to_remove", "cancel"})
	}
	for _, character := range rr.rows("characters") {
		name := str(character["tag"])
		if name == "" {
			name = str(character["name"])
		}
		emit("character", character, name, ownTag(character["country_tag"]), nil,
			[]string{"advisor_on_add", "advisor_on_remove", "advisor_available", "advisor_visible", "advisor_modifier"})
	}
	return out
}

type relExistence struct {
	sets     map[string]map[string]bool
	trustAll bool
}

var relVanillaTypes = map[string]bool{
	"idea": true, "event": true, "trait": true, "technology": true,
	"character": true, "opinion_modifier": true, "dynamic_modifier": true,
}

func (e relExistence) resolved(targetType, targetID string) bool {
	set, ok := e.sets[targetType]
	if !ok {
		return false
	}
	if e.trustAll || set[targetID] {
		return true
	}
	return relVanillaTypes[targetType] && exportengine.IsVanillaRef(targetType, targetID)
}

func relStringSet(rows []project.Row, column string) map[string]bool {
	out := map[string]bool{}
	for _, row := range rows {
		if v, ok := row[column].(string); ok {
			out[v] = true
		}
	}
	return out
}

func relBuildExistence(rr rowReader, proj relProject) relExistence {
	ex := relExistence{sets: map[string]map[string]bool{}, trustAll: proj.isSubmod}
	ex.sets["idea"] = relStringSet(rr.rows("ideas"), "idea_id")
	events := relStringSet(rr.rows("events"), "event_id")

	primaryLower := strings.ToLower(proj.countryTag)
	for _, se := range rr.rows("super_events") {
		tag := strings.ToLower(str(se["country_tag"]))
		if tag == "" {
			tag = primaryLower
		}
		if tag == "" {
			continue
		}
		events[tag+"_se."+strconv.FormatInt(project.Int64(se["number"]), 10)] = true
	}
	ex.sets["event"] = events
	ex.sets["focus"] = relStringSet(rr.rows("focus_nodes"), "focus_id")
	ex.sets["decision"] = relStringSet(rr.rows("decisions"), "decision_id")
	ex.sets["trait"] = relStringSet(rr.rows("custom_traits"), "trait_id")
	ex.sets["technology"] = relStringSet(rr.rows("technologies"), "tech_id")
	ex.sets["character"] = relStringSet(rr.rows("characters"), "tag")
	ex.sets["opinion_modifier"] = relStringSet(rr.rows("opinion_modifiers"), "modifier_id")
	ex.sets["dynamic_modifier"] = relStringSet(rr.rows("dynamic_modifiers"), "dm_id")
	autos := relStringSet(rr.rows("autonomy_states"), "autonomy_id")
	for _, a := range vanillaAutonomyStates {
		autos[a] = true
	}
	ex.sets["autonomy_state"] = autos
	ex.sets["mio"] = relStringSet(rr.rows("military_industrial_organizations"), "mio_id")
	return ex
}

func relTargetOwners(rr rowReader, proj relProject) map[[2]string][]string {
	owners := map[[2]string]map[string]bool{}
	add := func(targetType, targetID string, own *string, shared []string) {
		if targetID == "" {
			return
		}
		tags := relAllTags(own, shared, proj)
		if len(tags) == 0 {
			return
		}
		key := [2]string{targetType, targetID}
		if owners[key] == nil {
			owners[key] = map[string]bool{}
		}
		for _, t := range tags {
			owners[key][t] = true
		}
	}
	for _, spec := range []struct {
		targetType, table, column string
		shared                    bool
	}{
		{"idea", "ideas", "idea_id", true},
		{"event", "events", "event_id", true},
		{"decision", "decisions", "decision_id", true},
		{"trait", "custom_traits", "trait_id", false},
		{"technology", "technologies", "tech_id", false},
		{"character", "characters", "tag", false},
		{"opinion_modifier", "opinion_modifiers", "modifier_id", false},
		{"mio", "military_industrial_organizations", "mio_id", true},
	} {
		for _, row := range rr.rows(spec.table) {
			var shared []string
			if spec.shared {
				shared = jsonStrings(row["shared_country_tags"])
			}
			add(spec.targetType, str(row[spec.column]), ownTag(row["country_tag"]), shared)
		}
	}
	trees := map[int64]project.Row{}
	for _, tree := range rr.rows("focus_trees") {
		trees[project.Int64(tree["id"])] = tree
	}
	for _, node := range rr.rows("focus_nodes") {
		if tree, ok := trees[project.Int64(node["focus_tree_id"])]; ok {
			add("focus", str(node["focus_id"]), ownTag(tree["country_tag"]), jsonStrings(tree["shared_country_tags"]))
		}
	}
	out := make(map[[2]string][]string, len(owners))
	for k, v := range owners {
		tags := make([]string, 0, len(v))
		for t := range v {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		out[k] = tags
	}
	return out
}

func relationGraph(p *project.Project) ([]refEdge, error) {
	rr := &projectReader{p: p}
	proj := relLoadProject(rr.record())
	existence := relBuildExistence(rr, proj)
	targetOwners := relTargetOwners(rr, proj)
	sources := relSources(rr, proj)
	if rr.err != nil {
		return nil, rr.err
	}
	edges := []refEdge{}
	for _, src := range sources {
		for _, ref := range extractRelationRefs(src.text) {
			var targetTag *string
			if tags := targetOwners[[2]string{ref.targetType, ref.targetID}]; len(tags) == 1 {
				t := tags[0]
				targetTag = &t
			}
			edges = append(edges, refEdge{
				SourceKind: src.kind, SourceID: src.sourceID, SourceName: src.sourceName,
				SourceTag: src.tag, SourceTags: src.tags, EffectName: ref.effect,
				TargetType: ref.targetType, TargetID: ref.targetID, FieldName: src.fieldName,
				Resolved: existence.resolved(ref.targetType, ref.targetID), TargetTag: targetTag,
			})
		}
	}
	return edges, nil
}

func relationEntityIdentity(p *project.Project, kind string, sourceID int64) (string, bool, error) {
	if kind == "focus" {
		trees, err := p.Rows("focus_trees")
		if err != nil {
			return "", false, err
		}
		owned := idsOf(trees, func(project.Row) bool { return true })
		nodes, err := p.Rows("focus_nodes")
		if err != nil {
			return "", false, err
		}
		for _, node := range nodes {
			if project.Int64(node["id"]) == sourceID && owned[project.Int64(node["focus_tree_id"])] {
				return identityValue(node["focus_id"]), true, nil
			}
		}
		return "", false, nil
	}
	ident, ok := map[string][2]string{
		"event": {"events", "event_id"}, "decision": {"decisions", "decision_id"},
		"idea": {"ideas", "idea_id"}, "character": {"characters", "tag"},
		"technology": {"technologies", "tech_id"},
	}[kind]
	if !ok {
		return "", false, nil
	}
	rows, err := p.Rows(ident[0])
	if err != nil {
		return "", false, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == sourceID {
			return identityValue(row[ident[1]]), true, nil
		}
	}
	return "", false, nil
}

func identityValue(v any) string {
	if v == nil {
		return "None"
	}
	if s, ok := v.(string); ok {
		return s
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

func relationOutgoingFor(edges []refEdge, kind string, sourceID int64) []refEdge {
	out := []refEdge{}
	for _, e := range edges {
		if e.SourceKind == kind && e.SourceID == sourceID {
			out = append(out, e)
		}
	}
	return out
}

func relationIncomingFor(edges []refEdge, targetType, targetID string) []refEdge {
	out := []refEdge{}
	for _, e := range edges {
		if e.TargetType == targetType && e.TargetID == targetID {
			out = append(out, e)
		}
	}
	return out
}

func relationEdgesForCountry(edges []refEdge, tag string) (outgoing, incoming []refEdge) {
	up := strings.ToUpper(strings.TrimSpace(tag))
	prefix := up + "_"
	outgoing, incoming = []refEdge{}, []refEdge{}
	for _, e := range edges {
		match := false
		for _, t := range e.SourceTags {
			if t == up {
				match = true
				break
			}
		}
		if !match && len(e.SourceTags) == 0 && e.SourceTag != nil && strings.ToUpper(*e.SourceTag) == up {
			match = true
		}
		if match {
			outgoing = append(outgoing, e)
		}
		if strings.HasPrefix(strings.ToUpper(e.TargetID), prefix) {
			incoming = append(incoming, e)
		}
	}
	return outgoing, incoming
}

func writeEdgePair(w http.ResponseWriter, outgoing, incoming []refEdge) {
	apiutil.JSON(w, http.StatusOK, map[string]any{
		"outgoing": outgoing, "incoming": incoming,
		"outgoing_total": len(outgoing), "incoming_total": len(incoming),
	})
}

func relationGraphHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	edges, err := relationGraph(p)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"project_id": p.ID(), "edges": edges, "total": len(edges)})
	return nil
}

var relationKindTargetType = map[string]string{
	"focus": "focus", "event": "event", "decision": "decision",
	"idea": "idea", "character": "character", "technology": "technology",
}

func relationEntityHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	kind := r.PathValue("kind")
	sourceID, err := strconv.ParseInt(r.PathValue("sid"), 10, 64)
	if err != nil {
		return detail(http.StatusUnprocessableEntity, "source_id must be an integer")
	}
	edges, err := relationGraph(p)
	if err != nil {
		return err
	}
	outgoing := relationOutgoingFor(edges, kind, sourceID)
	incoming := []refEdge{}
	if targetType, known := relationKindTargetType[kind]; known {
		targetID := ""
		if len(outgoing) > 0 {
			targetID = outgoing[0].SourceName
		} else if id, ok, err := relationEntityIdentity(p, kind, sourceID); err != nil {
			return err
		} else if ok {
			targetID = id
		}
		if targetID != "" {
			incoming = relationIncomingFor(edges, targetType, targetID)
		}
	}
	writeEdgePair(w, outgoing, incoming)
	return nil
}

func relationCountryHandler(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	edges, err := relationGraph(p)
	if err != nil {
		return err
	}
	outgoing, incoming := relationEdgesForCountry(edges, r.PathValue("tag"))
	writeEdgePair(w, outgoing, incoming)
	return nil
}
