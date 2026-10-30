package exportengine

import (
	"sort"
	"strconv"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
)

func eventStartupHook(event *emitmisc.NS) string {
	if attach := strings.TrimSpace(event.Str("on_action_attach")); attach != "" {
		return attach
	}
	if event.Bool("auto_fire_on_startup", false) {
		return "on_startup"
	}
	if event.Int("auto_fire_after_days", 0) > 0 {
		return "on_startup"
	}
	return ""
}

type onActionsCountry struct {
	ideaIDs  []string
	autoFire []emitscripts.AutoFire
	treeIDs  []string
}

func NativeOnActions(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	return NativeOnActionsWithTraitOverlays(snapshot, modDir, includeImported, nil)
}

func NativeOnActionsWithTraitOverlays(snapshot Snapshot, modDir string, includeImported bool, traitOverlayNamespaces map[string]string) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	primaryTag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if primaryTag == "" {
		primaryTag = "UNK"
	}
	countryOf := func(entity *emitmisc.NS) string {
		ctag := strings.ToUpper(strings.TrimSpace(entity.Str("country_tag")))
		if ctag == "" {
			return primaryTag
		}
		return ctag
	}

	kept := func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") != "imported"
	}

	countries := map[string]*onActionsCountry{}
	var tags []string
	touch := func(ctag string) *onActionsCountry {
		if state, ok := countries[ctag]; ok {
			return state
		}
		state := &onActionsCountry{}
		countries[ctag] = state
		tags = append(tags, ctag)
		return state
	}
	decode := func(table string, visit func(*emitmisc.NS) error) error {
		for _, raw := range snapshot.Tables[table] {
			entity, err := emitmisc.DecodeEntityJSON(raw)
			if err != nil {
				return err
			}
			if err := visit(entity); err != nil {
				return err
			}
		}
		return nil
	}

	if err := decode("characters", func(character *emitmisc.NS) error {
		if !kept(character) && !character.Bool("replaces_vanilla_id", false) {
			return nil
		}
		touch(countryOf(character))
		return nil
	}); err != nil {
		return nil, err
	}

	if err := decode("ideas", func(idea *emitmisc.NS) error {
		if !kept(idea) || idea.Str("idea_type") != "country" {
			return nil
		}
		state := touch(countryOf(idea))
		if !idea.Bool("auto_attach_on_start", true) {
			return nil
		}
		if ideaID := strings.TrimSpace(idea.Str("idea_id")); ideaID != "" {
			state.ideaIDs = append(state.ideaIDs, ideaID)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	namespaces := map[int64]string{}
	if err := decode("event_namespaces", func(namespace *emitmisc.NS) error {
		namespaces[namespace.Int("id", 0)] = namespace.Str("namespace")
		return nil
	}); err != nil {
		return nil, err
	}

	if err := decode("events", func(event *emitmisc.NS) error {
		hook := eventStartupHook(event)
		if hook == "" || !kept(event) {
			return nil
		}
		state := touch(countryOf(event))
		namespace, ok := namespaces[event.Int("namespace_id", 0)]
		if !ok {
			return nil
		}
		state.autoFire = append(state.autoFire, emitscripts.AutoFire{
			EventID: namespace + "." + strconv.FormatInt(event.Int("event_number", 0), 10),
			Days:    int(event.Int("auto_fire_after_days", 0)),
			Hook:    hook,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := decode("super_events", func(superEvent *emitmisc.NS) error {
		if !kept(superEvent) || !emitmisc.SeIsAutoFire(superEvent) {
			return nil
		}
		ctag := countryOf(superEvent)
		state := touch(ctag)
		state.autoFire = append(state.autoFire, emitscripts.AutoFire{
			EventID: strings.ToLower(ctag) + "_se." + strconv.FormatInt(superEvent.Int("number", 0), 10),
			Days:    int(superEvent.Int("auto_fire_after_days", 0)),
			Hook:    "on_startup",
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if len(project.List("dependencies")) > 0 {
		if err := decode("focus_trees", func(tree *emitmisc.NS) error {
			if !kept(tree) || tree.StrOr("tree_kind", "primary") != "primary" {
				return nil
			}
			treeID := strings.TrimSpace(tree.Str("tree_id"))
			if treeID == "" {
				return nil
			}
			state := touch(countryOf(tree))
			state.treeIDs = append(state.treeIDs, treeID)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	sort.Strings(tags)
	var entries []Entry
	for _, ctag := range tags {
		state := countries[ctag]
		overlayNamespace := strings.TrimSpace(traitOverlayNamespaces[ctag])
		body := emitscripts.GenerateOnActions(
			ctag,
			overlayNamespace != "",
			overlayNamespace,
			state.ideaIDs,
			state.autoFire,
			state.treeIDs,
		)

		if strings.TrimSpace(body) == "" {
			continue
		}
		entries = append(entries, Entry{
			Path:         modDir + "/common/on_actions/" + ctag + "_on_actions.txt",
			Bytes:        []byte(body),
			Origin:       "generated",
			SourceEntity: "on_actions",
		})
	}
	return entries, nil
}
