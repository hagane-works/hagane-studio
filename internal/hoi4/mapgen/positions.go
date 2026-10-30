package mapgen

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

var stateBuildings = []struct {
	kind  string
	count int
}{
	{"arms_factory", 6}, {"industrial_complex", 6}, {"anti_air_building", 3}, {"air_base", 1},
	{"synthetic_refinery", 1}, {"fuel_silo", 1}, {"radar_station", 1}, {"rocket_site_spawn", 1},
	{"nuclear_reactor_spawn", 1}, {"stronghold_network", 1},
}

var provinceBuildings = []string{"supply_node", "bunker", "special_project_facility_spawn"}
var coastalBuildings = []string{"naval_base_spawn", "naval_headquarters", "naval_supply_hub", "coastal_bunker", "floating_harbor"}

var unitStackTypes = []int{0, 1, 9, 10, 21, 22, 38}

type Positions struct {
	Buildings   []string
	UnitStacks  []string
	SupplyNodes []string
	Weather     []string
}

func (b *Base) at(p int) (x, y, z float64) {
	px, py := p%b.Width, p/b.Width
	return float64(px), math.Max(float64(b.Heights[p])/10, 9.5), float64(b.Height - 1 - py)
}

func (b *Base) line(state int, kind string, p int, rotation float64, sea int) string {
	x, y, z := b.at(p)
	return fmt.Sprintf("%d;%s;%.2f;%.2f;%.2f;%.2f;%d", state, kind, x, y, z, rotation, sea)
}

func (b *Base) middle(px []int32) int {
	var sx, sy float64
	for _, p := range px {
		sx += float64(int(p) % b.Width)
		sy += float64(int(p) / b.Width)
	}
	sx, sy = sx/float64(len(px)), sy/float64(len(px))
	best, bestDistance := int(px[0]), math.MaxFloat64
	for _, p := range px {
		dx, dy := float64(int(p)%b.Width)-sx, float64(int(p)/b.Width)-sy
		if d := dx*dx + dy*dy; d < bestDistance {
			best, bestDistance = int(p), d
		}
	}
	return best
}

func (b *Base) shore(px []int32, kinds []byte, rng *rand.Rand) (land, water, sea int, ok bool) {
	var found [][2]int
	for _, p := range px {
		for _, q := range [4]int{int(p) - 1, int(p) + 1, int(p) - b.Width, int(p) + b.Width} {
			if q < 0 || q >= len(b.Province) {
				continue
			}
			if id := b.Province[q]; int(id) < len(kinds) && kinds[id] == kindSea {
				found = append(found, [2]int{int(p), q})
			}
		}
	}
	if len(found) == 0 {
		return 0, 0, 0, false
	}
	pick := found[rng.Intn(len(found))]
	return pick[0], pick[1], int(b.Province[pick[1]]), true
}

func (b *Base) Positions(add *Addition, stateIDs []int, regionID int, seed int64) Positions {
	rng := rand.New(rand.NewSource(seed + 101))
	kinds := b.kinds()
	var out Positions
	rotation := func() float64 { return math.Round(rng.Float64()*2*math.Pi*100) / 100 }
	stateOf := map[int]int{}
	for k, st := range add.States {
		for _, id := range st.Provinces {
			stateOf[id] = stateIDs[k]
		}
	}
	for k, st := range add.States {
		state := stateIDs[k]
		capital := add.Pixels(st.Capital)
		for _, building := range stateBuildings {
			for i := 0; i < building.count; i++ {
				out.Buildings = append(out.Buildings, b.line(state, building.kind, int(capital[rng.Intn(len(capital))]), rotation(), 0))
			}
		}
		for _, id := range st.Provinces {
			if land, _, _, ok := b.shore(add.Pixels(id), kinds, rng); ok {
				out.Buildings = append(out.Buildings, b.line(state, "dockyard", land, rotation(), 0))
				break
			}
		}
		out.SupplyNodes = append(out.SupplyNodes, fmt.Sprintf("1 %d", st.Capital))
	}
	for _, pr := range add.Provinces {
		px := add.Pixels(pr.ID)
		if pr.Kind == "land" {
			state := stateOf[pr.ID]
			for _, kind := range provinceBuildings {
				out.Buildings = append(out.Buildings, b.line(state, kind, int(px[rng.Intn(len(px))]), rotation(), 0))
			}
			if land, water, sea, ok := b.shore(px, kinds, rng); ok {
				for _, kind := range coastalBuildings {
					switch kind {
					case "naval_base_spawn":
						out.Buildings = append(out.Buildings, b.line(state, kind, land, rotation(), sea))
					case "floating_harbor":
						out.Buildings = append(out.Buildings, b.line(state, kind, water, rotation(), sea))
					default:
						out.Buildings = append(out.Buildings, b.line(state, kind, land, rotation(), 0))
					}
				}
			}
		}
		middle := b.middle(px)
		for _, kind := range unitStackTypes {
			x, y, z := b.at(middle)
			out.UnitStacks = append(out.UnitStacks, fmt.Sprintf("%d;%d;%.2f;%.2f;%.2f;0.00;0.30", pr.ID, kind, x, y, z))
		}
	}
	var land []int32
	for _, pr := range add.Provinces {
		if pr.Kind == "land" {
			land = append(land, add.Pixels(pr.ID)...)
		}
	}
	if len(land) > 0 {
		for _, size := range []string{"big", "small"} {
			x, y, z := b.at(int(land[rng.Intn(len(land))]))
			out.Weather = append(out.Weather, fmt.Sprintf("%d;%.2f;%.2f;%.2f;%s", regionID, x, y, z, size))
		}
	}
	return out
}

func (b *Base) MoveCovered(text string, add *Addition, unitStacks bool) (string, int) {
	newIDs := map[int32]bool{}
	for _, pr := range add.Provinces {
		newIDs[int32(pr.ID)] = true
	}
	lines := strings.Split(text, "\n")
	moved := 0
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		fields := strings.Split(trimmed, ";")
		if len(fields) < 7 {
			continue
		}
		xi, zi, seaField := 2, 4, 6
		if unitStacks {
			seaField = 0
		}
		x, errX := strconv.ParseFloat(fields[xi], 64)
		z, errZ := strconv.ParseFloat(fields[zi], 64)
		sea, errS := strconv.Atoi(fields[seaField])
		if errX != nil || errZ != nil || errS != nil || sea <= 0 {
			continue
		}
		px, py := int(x), b.Height-1-int(z)
		if px < 0 || py < 0 || px >= b.Width || py >= b.Height {
			continue
		}
		p := py*b.Width + px
		if unitStacks && int(b.Province[p]) == sea {
			continue
		}
		if !unitStacks && !newIDs[b.Province[p]] {
			continue
		}
		kept := add.Pixels(sea)
		if _, touched := add.Covered[sea]; !touched || len(kept) == 0 {
			continue
		}
		best, bestDistance := 0, math.MaxFloat64
		for _, q := range kept {
			dx, dy := float64(int(q)%b.Width-px), float64(int(q)/b.Width-py)
			if d := dx*dx + dy*dy; d < bestDistance {
				best, bestDistance = int(q), d
			}
		}
		nx, ny, nz := b.at(best)
		fields[xi], fields[xi+1], fields[zi] = fmt.Sprintf("%.2f", nx), fmt.Sprintf("%.2f", ny), fmt.Sprintf("%.2f", nz)
		lines[i] = strings.Join(fields, ";") + line[len(trimmed):]
		moved++
	}
	return strings.Join(lines, "\n"), moved
}

func AppendLines(text string, lines []string) string {
	if len(lines) == 0 {
		return text
	}
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}

	ended := text == "" || strings.HasSuffix(text, "\n")
	if !ended {
		text += newline
	}
	text += strings.Join(lines, newline)
	if ended {
		text += newline
	}
	return text
}

func RegionFile(id int, provinces []int, weather string) string {
	ids := append([]int(nil), provinces...)
	sort.Ints(ids)
	var list strings.Builder
	for _, p := range ids {
		list.WriteString(strconv.Itoa(p))
		list.WriteByte(' ')
	}
	out := fmt.Sprintf("strategic_region={\n\tid=%d\n\tname=\"STRATEGICREGION_%d\"\n\tprovinces={\n\t\t%s\n\t}\n", id, id, list.String())
	if weather != "" {
		out += "\t" + weather + "\n"
	}
	return out + "}\n"
}

func block(text, key string) (start, end int) {
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], key)
		if i < 0 {
			return -1, -1
		}
		i += from
		j := i + len(key)
		for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r' || text[j] == '\n') {
			j++
		}
		before := i == 0 || !isWordByte(text[i-1])
		if before && j < len(text) && text[j] == '=' {
			j++
			for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r' || text[j] == '\n') {
				j++
			}
			if j < len(text) && text[j] == '{' {
				depth := 0
				for k := j; k < len(text); k++ {
					switch text[k] {
					case '#':
						for k < len(text) && text[k] != '\n' {
							k++
						}
					case '{':
						depth++
					case '}':
						depth--
						if depth == 0 {
							return i, k + 1
						}
					}
				}
				return -1, -1
			}
		}
		from = i + len(key)
	}
	return -1, -1
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func WeatherBlock(regionText string) string {
	start, end := block(regionText, "weather")
	if start < 0 {
		return ""
	}
	return regionText[start:end]
}

func numbers(text string) [][2]int {
	var out [][2]int
	for i := 0; i < len(text); {
		switch c := text[i]; {
		case c == '#':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case c >= '0' && c <= '9' && (i == 0 || !isWordByte(text[i-1])):
			j := i
			for j < len(text) && text[j] >= '0' && text[j] <= '9' {
				j++
			}
			if j == len(text) || !isWordByte(text[j]) {
				out = append(out, [2]int{i, j})
			}
			i = j
		default:
			i++
		}
	}
	return out
}

func provincesList(regionText string) (inner, closing int, ok bool) {
	start, end := block(regionText, "provinces")
	if start < 0 {
		return 0, 0, false
	}
	return strings.IndexByte(regionText[start:end], '{') + start + 1, end - 1, true
}

func RegionProvinces(regionText string) []int {
	inner, closing, ok := provincesList(regionText)
	if !ok {
		return nil
	}
	var out []int
	for _, span := range numbers(regionText[inner:closing]) {
		if n, err := strconv.Atoi(regionText[inner+span[0] : inner+span[1]]); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func RemoveFromRegion(regionText string, id int) string {
	inner, closing, ok := provincesList(regionText)
	if !ok {
		return regionText
	}
	want := strconv.Itoa(id)
	list := regionText[inner:closing]
	for _, span := range numbers(list) {
		if list[span[0]:span[1]] != want {
			continue
		}

		from, to := span[0], span[1]
		if to < len(list) && (list[to] == ' ' || list[to] == '\t') {
			to++
		} else if from > 0 && (list[from-1] == ' ' || list[from-1] == '\t') {
			from--
		}
		return regionText[:inner+from] + regionText[inner+to:]
	}
	return regionText
}

func AddToRegion(regionText string, ids []int) string {
	inner, closing, ok := provincesList(regionText)
	if !ok || len(ids) == 0 {
		return regionText
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	insert := closing
	spans := numbers(regionText[inner:closing])
	if len(spans) > 0 {
		last := inner + spans[len(spans)-1][1]

		lineEnd := strings.IndexByte(regionText[last:closing], '\n')
		rest := regionText[last:closing]
		if lineEnd >= 0 {
			rest = rest[:lineEnd]
		}
		if strings.TrimSpace(rest) == "" {
			insert = last
		}
	}
	return regionText[:insert] + " " + strings.Join(parts, " ") + " " + regionText[insert:]
}
