package mapgen

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

var UnloadedFolders = []string{
	"map/strategicregions", "map/supplyareas",
	"history/states", "history/countries", "history/units", "history/general",
	"common/bookmarks", "common/national_focus", "events", "common/decisions", "common/decisions/categories",
	"common/ai_areas", "common/ai_strategy", "common/ai_strategy_plans", "common/strategic_locations",

	"common/ai_faction_theaters", "common/country_tags",

	"common/on_actions",
}

func (w *World) Owners(tags []string) (owner map[int]string, capital map[string]int) {
	order := w.statesWestToEast()
	owner = map[int]string{}
	if len(tags) == 0 {
		return owner, map[string]int{}
	}
	for i, s := range order {
		owner[s.id] = tags[min(i*len(tags)/len(order), len(tags)-1)]
	}
	return owner, capitalsOf(order, owner)
}

type stateAt struct {
	id   int
	x, y float64
}

func (w *World) statesWestToEast() []stateAt {
	centre := map[int][2]float64{}
	for _, pr := range w.All.Provinces {
		centre[pr.ID] = [2]float64{pr.X, pr.Y}
	}
	order := make([]stateAt, len(w.All.States))
	for k, st := range w.All.States {
		c := centre[st.Capital]
		order[k] = stateAt{k + 1, c[0], c[1]}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].x < order[j].x })
	return order
}

func capitalsOf(order []stateAt, owner map[int]string) map[string]int {
	held := map[string][]stateAt{}
	for _, s := range order {
		if tag := owner[s.id]; tag != "" {
			held[tag] = append(held[tag], s)
		}
	}
	capital := map[string]int{}
	for tag, states := range held {
		var mx, my float64
		for _, s := range states {
			mx += s.x
			my += s.y
		}
		mx /= float64(len(states))
		my /= float64(len(states))
		best, bestD := 0, 0.0
		for _, s := range states {
			if d := (s.x-mx)*(s.x-mx) + (s.y-my)*(s.y-my); best == 0 || d < bestD {
				best, bestD = s.id, d
			}
		}
		capital[tag] = best
	}
	return capital
}

func (w *World) OwnersFromMask(tags []string, mask []uint8, mw, mh int) (owner map[int]string, capital map[string]int, missing []string) {
	order := w.statesWestToEast()
	owner = map[int]string{}
	if len(tags) > 0 && mw > 0 && mh > 0 && len(mask) >= mw*mh && w.Base != nil {

		maxID := int32(0)
		for _, id := range w.Base.Province {
			maxID = max(maxID, id)
		}
		stateOf := make([]int32, maxID+1)
		for k, st := range w.All.States {
			for _, id := range st.Provinces {
				if id >= 0 && id <= int(maxID) {
					stateOf[id] = int32(k + 1)
				}
			}
		}
		votes := make([]int, (len(w.All.States)+1)*len(tags))
		width, height := w.Base.Width, w.Base.Height
		for y := 0; y < height; y++ {
			row := (y * mh / height) * mw
			for x := 0; x < width; x++ {
				v := int(mask[row+x*mw/width])
				if v == 0 || v > len(tags) {
					continue
				}
				if s := stateOf[w.Base.Province[y*width+x]]; s > 0 {
					votes[int(s)*len(tags)+v-1]++
				}
			}
		}
		var painted []stateAt
		for _, s := range order {
			best, bestVotes := -1, 0
			for t := range tags {
				if n := votes[s.id*len(tags)+t]; n > bestVotes {
					best, bestVotes = t, n
				}
			}
			if best >= 0 {
				owner[s.id] = tags[best]
				painted = append(painted, s)
			}
		}
		if len(painted) > 0 {
			for _, s := range order {
				if owner[s.id] != "" {
					continue
				}
				nearest, nearestD := stateAt{}, 0.0
				for _, p := range painted {
					if d := (p.x-s.x)*(p.x-s.x) + (p.y-s.y)*(p.y-s.y); nearest.id == 0 || d < nearestD {
						nearest, nearestD = p, d
					}
				}
				owner[s.id] = owner[nearest.id]
			}
		} else {
			owner = map[int]string{}
		}
	}
	if len(owner) == 0 {
		owner, capital = w.Owners(tags)
	} else {
		capital = capitalsOf(order, owner)
	}
	for _, tag := range tags {
		if capital[tag] == 0 {
			missing = append(missing, tag)
		}
	}
	return owner, capital, missing
}

type WorldMod struct {
	Tags    []string
	Owner   map[int]string
	Capital map[string]int
	Seed    int64

	LandWeather, SeaWeather string

	Tutorial string

	TagLines []string

	DynamicTags string

	Histories []string

	Languages []string

	StateNames bool

	Date string

	Names map[string]map[string]string
}

func (w *World) WorldFiles(m WorldMod) map[string]string {
	files := map[string]string{}
	positions := w.Positions(m.Seed)
	lines := func(list []string) string { return strings.Join(list, "\n") + "\n" }

	files["map/buildings.txt"] = strings.Join(positions.Buildings, "\n")
	files["map/unitstacks.txt"] = lines(positions.UnitStacks)
	files["map/supply_nodes.txt"] = lines(positions.SupplyNodes)
	files["map/weatherpositions.txt"] = lines(positions.Weather)
	files["map/railways.txt"] = "\n"

	files["map/adjacencies.csv"] = "From;To;Type;Through;start_x;start_y;stop_x;stop_y;adjacency_rule_name;Comment\n-1;-1;;-1;-1;-1;-1;-1;-1\n"
	files["map/adjacency_rules.txt"] = "# A new world: no straits or canals.\n"

	names := worldNames{}
	for _, region := range w.Regions {
		weather := m.LandWeather
		if region.Sea {
			weather = m.SeaWeather
		}
		if region.Weather != "" {
			weather = region.Weather
		}
		files[fmt.Sprintf("map/strategicregions/%d-Region %d.txt", region.ID, region.ID)] = RegionFile(region.ID, region.Provinces, weather)
		names.add(fmt.Sprintf("STRATEGICREGION_%d", region.ID), func(lang string) string {
			if region.Sea {
				return worldWord(lang, "sea_region", region.ID)
			}
			return worldWord(lang, "land_region", region.ID)
		})
	}
	files["map/supplyareas/1-SupplyArea.txt"] = "supply_area={\n\tid=1\n\tname=\"SUPPLYAREA_1\"\n\tvalue=12\n\tstates={\n\t\t1\n\t}\n}\n"
	names.add("SUPPLYAREA_1", func(lang string) string { return worldWord(lang, "supply_area", 0) })
	for k, st := range w.All.States {
		id := k + 1
		if m.StateNames {
			names.add(fmt.Sprintf("STATE_%d", id), func(lang string) string { return worldWord(lang, "state", id) })
		}
		names.add(fmt.Sprintf("VICTORY_POINTS_%d", st.Capital), func(lang string) string { return worldWord(lang, "town", id) })
	}

	if m.Tutorial != "" && len(w.All.States) > 0 && len(m.Tags) > 0 {
		enemy := m.Tags[len(m.Tags)-1]
		enemyCapital := w.All.States[0].Capital
		if id := m.Capital[enemy]; id > 0 && id <= len(w.All.States) {
			enemyCapital = w.All.States[id-1].Capital
		}
		files["tutorial/tutorial.txt"] = WorldTutorial(m.Tutorial, 1, w.All.States[0].Capital, enemy, enemyCapital)
	}

	files["common/country_tags/00_countries.txt"] = lines(m.TagLines)
	if m.DynamicTags != "" {
		files["common/country_tags/zz_dynamic_countries.txt"] = m.DynamicTags
	}
	for _, tag := range m.Histories {
		capital := m.Capital[tag]
		if capital == 0 {
			continue
		}
		files[fmt.Sprintf("history/countries/%s - New World.txt", tag)] = worldHistory(capital)
	}
	date := m.Date
	if date == "" {
		date = "1936.1.1.12"
	}
	var bookmarkCountries strings.Builder
	first := ""
	for _, tag := range m.Tags {
		if m.Capital[tag] == 0 {
			continue
		}
		if first == "" {
			first = tag
		}
		fmt.Fprintf(&bookmarkCountries, "\t\t%q = {\n\t\t\thistory = \"NEW_WORLD_DESC\"\n\t\t\tideology = neutrality\n\t\t\tideas = { }\n\t\t\tfocuses = { }\n\t\t}\n", tag)
	}
	files["common/bookmarks/new_world.txt"] = fmt.Sprintf(`bookmarks = {
	bookmark = {
		name = "NEW_WORLD_NAME"
		desc = "NEW_WORLD_DESC"
		date = %s
		picture = "GFX_select_date_1936"
		default_country = %q
		default = yes
%s		effect = {
			randomize_weather = 22345
		}
	}
}
`, date, first, bookmarkCountries.String())

	files["common/national_focus/new_world_default.txt"] = `focus_tree = {
	id = new_world_default_focus
	country = {
		factor = 0
	}
	default = yes
	focus = {
		id = NEW_WORLD_first_steps
		icon = GFX_goal_generic_construct_civ_factory
		x = 0
		y = 0
		cost = 10
		completion_reward = {
			add_political_power = 50
		}
	}
}
`
	for _, key := range []string{"name", "desc", "first_steps", "first_steps_desc"} {
		names.add(map[string]string{
			"name": "NEW_WORLD_NAME", "desc": "NEW_WORLD_DESC",
			"first_steps": "NEW_WORLD_first_steps", "first_steps_desc": "NEW_WORLD_first_steps_desc",
		}[key], func(lang string) string { return worldWord(lang, key, 0) })
	}
	names.override(m.Names)
	for _, lang := range m.Languages {
		files[fmt.Sprintf("localisation/%s/new_world_l_%s.yml", lang, lang)] = names.yml(lang)
	}
	return files
}

func UnloadMarkers(written []string) map[string]string {
	has := map[string]bool{}
	for _, file := range written {
		has[path.Dir(file)] = true
	}
	out := map[string]string{}
	for _, folder := range UnloadedFolders {
		if !has[folder] {
			out[folder+"/zz_new_world_unloaded.txt"] = "# A new world: the base game's files in this folder are unloaded.\n"
		}
	}
	return out
}

func Unloads(file string) bool {
	dir := path.Dir(file)
	for _, folder := range UnloadedFolders {
		if dir == folder {
			return true
		}
	}
	return false
}

func worldHistory(capital int) string {
	return fmt.Sprintf(`capital = %d
set_research_slots = 3
set_stability = 0.6
set_war_support = 0.4
set_politics = {
	ruling_party = neutrality
	last_election = "1932.1.1"
	election_frequency = 48
	elections_allowed = no
}
set_popularities = {
	democratic = 20
	fascism = 20
	communism = 20
	neutrality = 40
}
`, capital)
}

type worldNames struct {
	keys  []string
	names map[string]func(lang string) string
}

func (n *worldNames) add(key string, name func(lang string) string) {
	if n.names == nil {
		n.names = map[string]func(string) string{}
	}
	if _, ok := n.names[key]; !ok {
		n.keys = append(n.keys, key)
	}
	n.names[key] = name
}

func (n *worldNames) override(own map[string]map[string]string) {
	keys := map[string]bool{}
	for _, byKey := range own {
		for key := range byKey {
			keys[key] = true
		}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	for _, key := range sorted {
		generated := n.names[key]
		n.add(key, func(lang string) string {
			if text := own[lang][key]; text != "" {
				return text
			}
			if text := own["english"][key]; text != "" {
				return text
			}
			if generated != nil {
				return generated(lang)
			}
			return key
		})
	}
}

func (n *worldNames) yml(lang string) string {
	var out strings.Builder
	out.WriteString("\ufeffl_" + lang + ":\n")
	for _, key := range n.keys {
		fmt.Fprintf(&out, " %s:0 %q\n", key, n.names[key](lang))
	}
	return out.String()
}

func worldWord(lang, kind string, n int) string {
	zh := lang == "simp_chinese"
	switch kind {
	case "state":
		if zh {
			return fmt.Sprintf("第 %d 州", n)
		}
		return fmt.Sprintf("State %d", n)
	case "town":
		if zh {
			return fmt.Sprintf("城镇 %d", n)
		}
		return fmt.Sprintf("Town %d", n)
	case "land_region":
		if zh {
			return fmt.Sprintf("陆地战略区 %d", n)
		}
		return fmt.Sprintf("Land Region %d", n)
	case "sea_region":
		if zh {
			return fmt.Sprintf("海域 %d", n)
		}
		return fmt.Sprintf("Sea Region %d", n)
	case "supply_area":
		if zh {
			return "补给区"
		}
		return "Supply Area"
	case "name":
		if zh {
			return "新世界"
		}
		return "A New World"
	case "desc":
		if zh {
			return "一个从草图开始的世界。"
		}
		return "A world of its own."
	case "first_steps":
		if zh {
			return "第一步"
		}
		return "First Steps"
	case "first_steps_desc":
		if zh {
			return "国家迈出第一步。"
		}
		return "The country takes its first steps."
	}
	return ""
}

var (
	tutorialState     = regexp.MustCompile(`\bstate\s*=\s*\d+`)
	tutorialStates    = regexp.MustCompile(`\bhighlight_states_trigger\s*=\s*\{[^}]*\}`)
	tutorialPointer   = regexp.MustCompile(`\bdirection_pointer\s*=\s*\d+`)
	tutorialProvinces = regexp.MustCompile(`\bhighlight_provinces\s*=\s*\{[^}]*\}`)
	tutorialTarget    = regexp.MustCompile(`\btarget\s*=\s*"?[A-Z]{3}"?`)
	tutorialAt        = regexp.MustCompile(`\btarget\s*=\s*\d+`)
)

func WorldTutorial(text string, state, province int, enemy string, enemyCapital int) string {
	text = tutorialAt.ReplaceAllString(text, fmt.Sprintf("target = %d", enemyCapital))
	text = tutorialState.ReplaceAllString(text, fmt.Sprintf("state = %d", state))
	text = tutorialStates.ReplaceAllString(text, fmt.Sprintf("highlight_states_trigger = { %d }", state))
	text = tutorialPointer.ReplaceAllString(text, fmt.Sprintf("direction_pointer = %d", province))
	text = tutorialProvinces.ReplaceAllString(text, fmt.Sprintf("highlight_provinces = { %d }", province))
	return tutorialTarget.ReplaceAllString(text, fmt.Sprintf("target = %q", enemy))
}
