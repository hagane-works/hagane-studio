package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

const MIOSourceKey = "__source_file"

const mioOrganizationsDir = "common/military_industrial_organization/organizations/"

func MIOSourceFile(row *emitmisc.NS) string {
	extras := row.Dict("raw_extras")
	if extras == nil {
		return ""
	}
	value, ok := extras.Get(MIOSourceKey)
	if !ok {
		return ""
	}
	file, _ := value.(string)
	file = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(file), "\\", "/"), "/")
	lower := strings.ToLower(file)
	if !strings.HasPrefix(lower, mioOrganizationsDir) || !strings.HasSuffix(lower, ".txt") {
		return ""
	}
	if rest := file[len(mioOrganizationsDir):]; rest == ".txt" || strings.Contains(rest, "/") || strings.Contains(rest, "..") {
		return ""
	}
	return file
}

func withoutMIOTraits(entity *locout.Obj) *locout.Obj {
	if entity == nil {
		return nil
	}
	out := &locout.Obj{M: map[string]locout.Value{}}
	for _, key := range entity.Keys {
		if key == "traits" {
			continue
		}
		out.Keys = append(out.Keys, key)
		out.M[key] = entity.M[key]
	}
	return out
}
