package gamedir

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const steamFolderName = "Hearts of Iron IV"

var ErrNotFound = errors.New("hearts of iron iv installation not found")

type Install struct {
	Dir     string `json:"dir"`
	Version string `json:"version"`
}

func Valid(dir string) bool {
	if dir == "" {
		return false
	}
	for _, name := range []string{"common", "interface", "gfx", "localisation"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func Inspect(dir string) (Install, error) {
	dir = filepath.Clean(dir)
	if !Valid(dir) {
		return Install{}, ErrNotFound
	}
	return Install{Dir: dir, Version: readVersion(dir)}, nil
}

func Find() (Install, error) {
	for _, library := range steamLibraries() {
		if install, err := Inspect(filepath.Join(library, "steamapps", "common", steamFolderName)); err == nil {
			return install, nil
		}
	}
	return Install{}, ErrNotFound
}

func readVersion(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "launcher-settings.json"))
	if err != nil {
		return ""
	}
	var settings struct {
		RawVersion string `json:"rawVersion"`
	}
	if json.Unmarshal(raw, &settings) != nil {
		return ""
	}
	return settings.RawVersion
}

func steamLibraries() []string {
	seen := map[string]bool{}
	var libraries []string
	add := func(dir string) {
		dir = filepath.Clean(dir)
		key := strings.ToLower(dir)
		if dir == "." || seen[key] {
			return
		}
		seen[key] = true
		libraries = append(libraries, dir)
	}
	for _, root := range steamRoots() {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		add(root)
		for _, path := range libraryFolderPaths(filepath.Join(root, "steamapps", "libraryfolders.vdf")) {
			add(path)
		}
	}
	return libraries
}

func steamRoots() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		roots := []string{}
		for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
			if dir := os.Getenv(env); dir != "" {
				roots = append(roots, filepath.Join(dir, "Steam"))
			}
		}
		return append(roots, `C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`)
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Steam")}
	default:
		return []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		}
	}
}

func libraryFolderPaths(vdf string) []string {
	file, err := os.Open(vdf)
	if err != nil {
		return nil
	}
	defer file.Close()
	var paths []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := quotedFields(scanner.Text())
		if len(fields) == 2 && strings.EqualFold(fields[0], "path") {
			paths = append(paths, strings.ReplaceAll(fields[1], `\\`, `\`))
		}
	}
	return paths
}

func quotedFields(line string) []string {
	var fields []string
	for {
		start := strings.IndexByte(line, '"')
		if start < 0 {
			return fields
		}
		line = line[start+1:]
		end := 0
		for end < len(line) && line[end] != '"' {
			if line[end] == '\\' {
				end++
			}
			end++
		}
		if end >= len(line) {
			return fields
		}
		fields = append(fields, line[:end])
		line = line[end+1:]
	}
}

func ModDir() string {
	candidates := userDataCandidates()
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return filepath.Join(dir, "mod")
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	return filepath.Join(candidates[0], "mod")
}

func userDataCandidates() []string {
	home, _ := os.UserHomeDir()
	suffix := filepath.Join("Paradox Interactive", "Hearts of Iron IV")
	switch runtime.GOOS {
	case "windows":
		var candidates []string
		for _, documents := range []string{
			filepath.Join(home, "Documents"),
			filepath.Join(os.Getenv("OneDrive"), "Documents"),
			filepath.Join(home, "OneDrive", "Documents"),
		} {
			if documents != "Documents" {
				candidates = append(candidates, filepath.Join(documents, suffix))
			}
		}
		return candidates
	case "darwin":
		return []string{filepath.Join(home, "Documents", suffix)}
	default:
		return []string{filepath.Join(home, ".local", "share", suffix)}
	}
}
