package importjob

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const maxImportedFlagBytes = 4 << 20

var importedFlagName = regexp.MustCompile(`(?i)^([a-z0-9]{3})(?:_([a-z0-9_]+))?\.(tga|dds|png)$`)

var vanillaFlagIdeologies = []string{"democratic", "fascism", "neutrality", "communism"}

func stagedCountryFlags(root string) (map[string]map[string]string, error) {
	dir := filepath.Join(root, "gfx", "flags")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rank := func(name string) int {
		return strings.Index(".tga.dds.png", strings.ToLower(filepath.Ext(name)))
	}
	out := map[string]map[string]string{}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		match := importedFlagName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		tag, ideology := strings.ToUpper(match[1]), strings.ToLower(match[2])
		if out[tag] == nil {
			out[tag] = map[string]string{}
		}
		if previous, ok := out[tag][ideology]; ok && rank(previous) <= rank(entry.Name()) {
			continue
		}
		out[tag][ideology] = filepath.Join(dir, entry.Name())
	}
	return out, nil
}

func flagIdeologies(registryRaw []byte) map[string]bool {
	known := map[string]bool{}
	for _, ideology := range vanillaFlagIdeologies {
		known[ideology] = true
	}
	var registry struct {
		Groups          []string `json:"groups"`
		RootDefinitions []struct {
			ID string `json:"id"`
		} `json:"root_definitions"`
	}
	if json.Unmarshal(registryRaw, &registry) == nil {
		names := append([]string(nil), registry.Groups...)
		for _, root := range registry.RootDefinitions {
			names = append(names, root.ID)
		}
		for _, name := range names {
			if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
				known[name] = true
			}
		}
	}
	return known
}

type importedFlag struct {
	countryID int64
	tag       string
	ideology  string
	file      string
	key       string
}

func storeImportedFlag(store GUISourceStore, projectUUID string, flag *importedFlag) (string, error) {
	file, err := os.Open(flag.file)
	if err != nil {
		return "", err
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxImportedFlagBytes+1))
	_ = file.Close()
	if err != nil {
		return "", err
	}
	if len(raw) > maxImportedFlagBytes {
		return "", errors.New("the file is too large for a flag")
	}
	png, err := importassets.NormalizePNG(raw, filepath.Base(flag.file), "country_flag")
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(png)
	sha := hex.EncodeToString(sum[:])
	var key string
	if flag.ideology == "" {
		key, err = importassets.EntityAssetKey(store.EnvPrefix(), projectUUID, "flag", flag.countryID, "country_flag", sha)
	} else {
		key, err = importassets.NamedAssetKey(store.EnvPrefix(), projectUUID, "flag", countryFlagVariantName(flag.tag, flag.ideology), sha)
	}
	if err != nil {
		return "", err
	}
	if err := store.Put(key, "image/png", int64(len(png)), bytes.NewReader(png)); err != nil {
		return "", err
	}
	return key, nil
}

func (s *session) persistImportedCountryFlags(tx *project.Tx, files GUISourceStore) (int, []string, error) {
	staged, err := stagedCountryFlags(s.root)
	if err != nil || len(staged) == 0 {
		return 0, nil, err
	}
	if files == nil {
		return 0, nil, errors.New("country flags were imported but private object storage is unavailable")
	}

	record := tx.Record()
	projectUUID, _ := record["uuid"].(string)
	registryRaw := []byte("{}")
	if registry := record["ideology_registry"]; registry != nil {
		if raw, err := json.Marshal(registry); err == nil {
			registryRaw = raw
		}
	}
	known := flagIdeologies(registryRaw)

	uploaded := map[int64]bool{}
	for _, row := range tx.Rows("entity_obs_paths") {
		if row["entity_type"] == "country_flag" {
			uploaded[project.Int64(row["entity_id"])] = true
		}
	}
	countries := append([]project.Row(nil), tx.Rows("countries")...)
	sort.SliceStable(countries, func(i, j int) bool {
		return project.Int64(countries[i]["id"]) < project.Int64(countries[j]["id"])
	})
	var flags []importedFlag
	for _, row := range countries {
		countryID := project.Int64(row["id"])
		rawTag, _ := row["tag"].(string)
		tag := strings.ToUpper(strings.TrimSpace(rawTag))
		byTag := staged[tag]
		if len(byTag) == 0 {
			continue
		}
		path, _ := row["flag_image_path"].(string)
		hasBase := path != "" || uploaded[countryID]
		existing := map[string]string{}
		if byIdeology, ok := row["flag_paths_by_ideology"].(map[string]any); ok {
			for ideology, value := range byIdeology {
				if text, ok := value.(string); ok {
					existing[ideology] = text
				}
			}
		}
		ideologies := make([]string, 0, len(byTag))
		for ideology := range byTag {
			ideologies = append(ideologies, ideology)
		}
		sort.Strings(ideologies)
		for _, ideology := range ideologies {
			switch {
			case ideology == "" && hasBase:
			case ideology != "" && (!known[ideology] || existing[ideology] != ""):
			default:
				flags = append(flags, importedFlag{countryID: countryID, tag: tag, ideology: ideology, file: byTag[ideology]})
			}
		}
	}

	problems := make([]string, len(flags))
	var wg sync.WaitGroup
	slots := make(chan struct{}, min(4, runtime.GOMAXPROCS(0)))
	for i := range flags {
		wg.Add(1)
		slots <- struct{}{}
		go func(flag *importedFlag, problem *string) {
			defer wg.Done()
			defer func() { <-slots }()
			key, err := storeImportedFlag(files, projectUUID, flag)
			if err != nil {
				*problem = fmt.Sprintf("Country flag gfx/flags/%s was not kept: %v", filepath.Base(flag.file), err)
				return
			}
			flag.key = key
		}(&flags[i], &problems[i])
	}
	wg.Wait()

	baseFlags := newKeyedTable(tx, "entity_obs_paths", "entity_type", "entity_id")
	ideologyFlags := newKeyedTable(tx, "project_obs_assets", "asset_type", "gfx_name")
	kept := 0
	var notes []string
	for i, flag := range flags {
		if problems[i] != "" {
			notes = append(notes, problems[i])
			continue
		}
		if flag.ideology == "" {

			baseFlags.insertNew(project.Row{"entity_type": "country_flag", "entity_id": flag.countryID, "obs_key": flag.key})

			country := tx.Find("countries", flag.countryID)
			if current, _ := country["flag_image_path"].(string); country != nil && current == "" {
				if _, err := tx.Update("countries", flag.countryID, func(row project.Row) {
					row["flag_image_path"], row["has_obs"] = "flags/"+flag.tag+".tga", true
				}); err != nil {
					return 0, nil, err
				}
			}
		} else {

			ideologyFlags.insertNew(project.Row{
				"asset_type": "country_flag_ideology", "gfx_name": countryFlagVariantName(flag.tag, flag.ideology), "obs_key": flag.key,
			})

			country := tx.Find("countries", flag.countryID)
			current, _ := country["flag_paths_by_ideology"].(map[string]any)
			if _, present := current[flag.ideology]; country != nil && !present {
				if _, err := tx.Update("countries", flag.countryID, func(row project.Row) {
					next := map[string]any{}
					if byIdeology, ok := row["flag_paths_by_ideology"].(map[string]any); ok {
						for ideology, value := range byIdeology {
							next[ideology] = value
						}
					}
					next[flag.ideology] = "flags/" + flag.tag + "_" + flag.ideology + ".tga"
					row["flag_paths_by_ideology"] = next
				}); err != nil {
					return 0, nil, err
				}
			}
		}
		kept++
	}
	return kept, notes, nil
}
