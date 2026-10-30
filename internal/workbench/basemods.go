package workbench

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"haganeworks/internal/hoi4/emit/moddesc"
	"haganeworks/internal/project"
)

var ErrNotAMod = ValidationError("this folder is not a Hearts of Iron IV mod: it has no descriptor.mod with a name")

type BaseMod struct {
	Name string `json:"name"`

	Folder string `json:"folder"`

	Found bool `json:"found"`
}

func BaseMods(p *project.Project) []BaseMod {
	folders := p.BaseMods()
	out := []BaseMod{}
	for _, name := range stringList(p.Record()["dependencies"]) {
		mod := BaseMod{Name: name, Folder: folders[name]}
		if mod.Folder != "" {
			info, err := os.Stat(mod.Folder)
			mod.Found = err == nil && info.IsDir()
		}
		out = append(out, mod)
	}
	return out
}

func SetBaseMod(p *project.Project, dependency, folder string) (string, error) {
	name := dependency
	if folder != "" {
		modName, err := ModName(folder)
		if err != nil {
			return "", err
		}
		if name == "" {
			name = modName
		}
	} else if name == "" {
		return "", ValidationError("choose the folder of a base mod")
	}
	if dependencies := stringList(p.Record()["dependencies"]); !slices.Contains(dependencies, name) {
		if dependency != "" {
			return "", ValidationError(fmt.Sprintf("%q is not a dependency of this project", dependency))
		}
		dependencies = append(dependencies, name)
		patch := UpdatePatch{Dependencies: Optional[[]string]{Set: true, Value: &dependencies}}
		if err := validateAgainst(&patch, p.Record()); err != nil {
			return "", ValidationError(err.Error())
		}
	}
	_, err := p.UpdateRecordAndBaseMods(func(record project.Row, baseMods map[string]string) {
		if dependencies := stringList(record["dependencies"]); !slices.Contains(dependencies, name) {
			list := make([]any, 0, len(dependencies)+1)
			for _, dependency := range append(dependencies, name) {
				list = append(list, dependency)
			}
			record["dependencies"] = list
		}
		if folder == "" {
			delete(baseMods, name)
		} else {
			baseMods[name] = folder
		}
		record["updated_at"] = nowTimestamp()
	})
	if err != nil {
		return "", err
	}
	return name, nil
}

func forgetRemovedBaseMods(record project.Row, baseMods map[string]string) {
	dependencies := stringList(record["dependencies"])
	for name := range baseMods {
		if !slices.Contains(dependencies, name) {
			delete(baseMods, name)
		}
	}
}

func ModName(dir string) (string, error) {
	file, err := os.Open(filepath.Join(dir, "descriptor.mod"))
	if err != nil {
		return "", ErrNotAMod
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 64<<10))
	if err != nil {
		return "", ErrNotAMod
	}
	name := strings.TrimSpace(moddesc.ParseDescriptorMod(string(raw)).Scalars["name"])
	if name == "" {
		return "", ErrNotAMod
	}
	return name, nil
}
