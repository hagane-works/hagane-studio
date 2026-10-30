package importjob

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/gfx"
)

func normalizeImagePath(value string) string {
	v := strings.Trim(strings.ReplaceAll(value, "\\", "/"), " \t\r\n\"")
	for strings.Contains(v, "//") {
		v = strings.ReplaceAll(v, "//", "/")
	}
	v = strings.TrimPrefix(v, "/")
	return v
}

func (s *session) stageGFX() (int64, error) {
	var files []string
	base := filepath.Join(s.root, "interface")
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".gfx") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	sort.Strings(files)
	var total int64
	err = inFileOrder(s.ctx, len(files), func(i int) (map[string]string, error) {
		body, err := os.ReadFile(files[i])
		if err != nil {
			return nil, err
		}
		return gfx.ParseGfxMappings(decodeScriptBytes(body)), nil
	}, func(_ int, mappings map[string]string) error {
		for name, texture := range mappings {
			s.gfx[name] = normalizeImagePath(texture)
			total++
		}
		return nil
	})
	return total, err
}

func (s *session) neededImageList() []string {
	for _, row := range s.entities {
		if icon := row.data().str("icon"); icon != "" {
			if texture := s.gfx[icon]; texture != "" {
				s.neededImages[texture] = true
			}
		}
	}
	out := make([]string, 0, len(s.neededImages))
	for path := range s.neededImages {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
