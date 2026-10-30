package guirender

import (
	"path"
	"strconv"
	"strings"
)

func (idx *Index) indexContinentFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "continents") {
			continue
		}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Block != nil {
				continue
			}
			name := ""
			if item.Key != nil {
				name = *item.Key
			} else {
				name = toString(scalarValue(item.Scalar))
			}
			if name != "" {
				idx.continents = append(idx.continents, name)
			}
		}
	}
}

func (idx *Index) indexProvinceDefinitions(body []byte) {
	idx.provinceDefs = idx.provinceDefs[:0]
	for _, line := range strings.Split(string(body), "\n") {
		if def, ok := parseProvinceDef(line); ok {
			idx.provinceDefs = append(idx.provinceDefs, def)
		}
		fields := strings.Split(strings.TrimSpace(line), ";")
		if len(fields) < 8 {
			continue
		}
		province, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		continent, err := strconv.Atoi(strings.TrimSpace(fields[7]))
		if err != nil || continent <= 0 {
			continue
		}
		idx.provinceContinent[province] = continent
	}
}

func (idx *Index) indexStateFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "state") {
			continue
		}
		id, ok := toFloat(ciScalarValue(entry.Block.Entries, "id"))
		if !ok {
			continue
		}

		idx.states[int(id)] = parseStateRecord(entry.Block.Entries)
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "provinces") {
				continue
			}
			for _, province := range field.Block.Entries {
				if province == nil || province.Block != nil {
					continue
				}
				if value, ok := toFloat(scalarValue(province.Scalar)); ok {
					idx.stateProvince[int(id)] = int(value)
					break
				}
			}
			break
		}
	}
}

func (idx *Index) continentList() []string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.continents) > 0 {
			return layer.continents
		}
	}
	return nil
}

func (idx *Index) provinceOfState(state int) (int, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if province, ok := layer.stateProvince[state]; ok {
			return province, true
		}
	}
	return 0, false
}

func (idx *Index) continentOfProvince(province int) (int, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if continent, ok := layer.provinceContinent[province]; ok {
			return continent, true
		}
	}
	return 0, false
}

func (idx *Index) ContinentOf(tag string) string {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil || politics.Capital == 0 {
		return ""
	}
	province, ok := idx.provinceOfState(politics.Capital)
	if !ok {
		return ""
	}
	continent, ok := idx.continentOfProvince(province)
	if !ok {
		return ""
	}
	names := idx.continentList()
	if continent < 1 || continent > len(names) {
		return ""
	}
	return names[continent-1]
}

func stateFileID(name string) int {
	base := path.Base(name)
	cut := strings.IndexByte(base, '-')
	if cut <= 0 {
		return 0
	}
	id, err := strconv.Atoi(base[:cut])
	if err != nil {
		return 0
	}
	return id
}
