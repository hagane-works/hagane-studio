package guirender

import (
	"path"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

type maskPlan struct {
	once   sync.Once
	source *Index
	groups map[string][]string
	writes map[string][]int
}

var maskFieldsIndex = func() []int {
	skip := map[string]bool{"files": true, "filesFolded": true, "source": true, "masking": true}
	t := reflect.TypeOf(Index{})
	out := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if !skip[t.Field(i).Name] {
			out = append(out, i)
		}
	}
	return out
}()

func maskGroup(name string) string {
	lower := strings.ToLower(name)
	return path.Dir(lower) + "|" + path.Ext(lower)
}

func (idx *Index) PrepareMasking() {
	if idx != nil && idx.masking != nil {
		idx.masking.once.Do(idx.masking.build)
	}
}

func (plan *maskPlan) build() {
	layer := plan.source
	plan.groups = map[string][]string{}
	for name := range layer.files {
		group := maskGroup(name)
		plan.groups[group] = append(plan.groups[group], name)
	}
	empty, _ := newIndex(map[string][]byte{}, layer.source)
	emptyValue := reflect.ValueOf(empty).Elem()
	plan.writes = make(map[string][]int, len(plan.groups))
	for group, names := range plan.groups {
		files := make(map[string][]byte, len(names))
		for _, name := range names {
			files[name] = layer.files[name]
		}
		read, err := newIndex(files, layer.source)
		if err != nil {
			plan.writes[group] = maskFieldsIndex
			continue
		}
		readValue := reflect.ValueOf(read).Elem()
		for _, field := range maskFieldsIndex {
			if !reflect.DeepEqual(indexField(readValue, field).Interface(), indexField(emptyValue, field).Interface()) {
				plan.writes[group] = append(plan.writes[group], field)
			}
		}
	}
}

func (idx *Index) withoutOwnFiles(mask lowerFileMask) (out *Index, partial bool, err error) {
	plan := idx.masking
	if plan == nil {
		return nil, false, nil
	}
	plan.once.Do(plan.build)
	affected := map[string]bool{}
	for name := range idx.files {
		if mask.removes(name) {
			affected[maskGroup(name)] = true
		}
	}

	fields := map[int]bool{}
	for changed := true; changed; {
		changed = false
		for group := range affected {
			for _, field := range plan.writes[group] {
				if !fields[field] {
					fields[field], changed = true, true
				}
			}
		}
		for group, written := range plan.writes {
			if affected[group] {
				continue
			}
			for _, field := range written {
				if fields[field] {
					affected[group], changed = true, true
					break
				}
			}
		}
	}
	kept := map[string][]byte{}
	for group := range affected {
		for _, name := range plan.groups[group] {
			if body, ok := idx.files[name]; ok && !mask.removes(name) {
				kept[name] = body
			}
		}
	}
	read, err := newIndex(kept, idx.source)
	if err != nil {
		return nil, true, err
	}
	masked := *idx
	masked.files = make(map[string][]byte, len(idx.files))
	masked.filesFolded = make(map[string]string, len(idx.files))
	for name, body := range idx.files {
		if mask.removes(name) {
			continue
		}
		masked.files[name] = body
		masked.filesFolded[strings.ToLower(name)] = name
	}
	maskedValue, readValue := reflect.ValueOf(&masked).Elem(), reflect.ValueOf(read).Elem()
	for field := range fields {
		indexField(maskedValue, field).Set(indexField(readValue, field))
	}

	if len(masked.equipmentDuplicates) > 0 && sameMap(masked.equipment, idx.equipment) {
		masked.equipment = make(map[string]*equipmentRecord, len(idx.equipment))
		for name, record := range idx.equipment {
			masked.equipment[name] = record
		}
	}
	return &masked, true, nil
}

func indexField(value reflect.Value, field int) reflect.Value {
	f := value.Field(field)
	return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
}

func sameMap(a, b map[string]*equipmentRecord) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
