package worldmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func TestModMapImportKeepsTheModsStatesRegionsAndNames(t *testing.T) {
	_, base, _ := provinceEditWorld(t)
	req := worldRequest{source: "mod", languages: []string{"en", "zh"}, date: "1936.1.1", tagMap: map[string]string{}, modFiles: map[string][]byte{
		"history/states/1-West.txt": []byte(`state={ id=1 name="STATE_1" manpower=1000 state_category=city
	resources={ steel=5 }
	history={ owner=AAA add_core_of=AAA add_core_of=ZZZ victory_points={ 1 5 } buildings={ infrastructure=3 arms_factory=2 1={ naval_base=1 } } }
	provinces={ 1 } }`),
		"history/states/2-East.txt": []byte(`state={ id=2 name="STATE_2" history={ owner=BBB } provinces={ 5 } }`),
		"map/strategicregions/1-West.txt": []byte(`strategic_region={ id=1 name="STRATEGICREGION_1" provinces={ 1 }
	weather={ period={ between={ 0.0 30.11 } temperature={ -5.0 5.0 } } } }`),
		"map/strategicregions/2-Sea.txt":                      []byte(`strategic_region={ id=2 name="STRATEGICREGION_2" provinces={ 2 3 4 } }`),
		"map/strategicregions/3-East.txt":                     []byte(`strategic_region={ id=3 name="STRATEGICREGION_3" provinces={ 5 } }`),
		"localisation/english/states_l_english.yml":           []byte("\uFEFFl_english:\n STATE_1:0 \"Westland\"\n STRATEGICREGION_1:0 \"West Region\"\n VICTORY_POINTS_1:0 \"Capital City\"\n"),
		"localisation/simp_chinese/states_l_simp_chinese.yml": []byte("\uFEFFl_simp_chinese:\n STATE_1:0 \"西境\"\n"),
	}}
	world, mod, problem, err := buildModWorld(base, req)
	if err != nil || problem != nil {
		t.Fatalf("build: %q %v", problem, err)
	}
	if len(world.All.States) != 2 || len(world.Regions) != 3 || !strings.Contains(world.Regions[0].Weather, "temperature") {
		t.Fatalf("world: %d states, regions %+v", len(world.All.States), world.Regions)
	}
	if got := world.LoadRules(); len(got) != 0 {
		t.Fatalf("load rules: %+v", got)
	}
	if mod.stateNames["en"][1] != "Westland" || mod.stateNames["zh"][1] != "西境" || mod.stateNames["en"][2] != "" {
		t.Fatalf("state names %+v", mod.stateNames)
	}
	if mod.names["english"]["STRATEGICREGION_1"] != "West Region" || mod.names["english"]["VICTORY_POINTS_1"] != "Capital City" {
		t.Fatalf("names %+v", mod.names)
	}

	if owner, _, _, unknown := modOwners(world, mod, req, []string{"AAA"}); owner != nil || !reflect.DeepEqual(unknown, []modUnknownTag{{Tag: "BBB", States: 1}}) {
		t.Fatalf("unknown %+v owner %+v", unknown, owner)
	}
	req.tagMap["BBB"] = "AAA"
	owner, capital, tags, unknown := modOwners(world, mod, req, []string{"AAA"})
	if unknown != nil || !reflect.DeepEqual(owner, map[int]string{1: "AAA", 2: "AAA"}) || capital["AAA"] != 1 || !reflect.DeepEqual(tags, []string{"AAA"}) {
		t.Fatalf("owner %+v capital %+v tags %v unknown %v", owner, capital, tags, unknown)
	}

	rows := modStateRows(world, mod, req, owner, []string{"AAA"})
	first := rows[0]
	if first.Category != "city" || first.Manpower != 1000 || first.Locs["en"] != "Westland" || first.Locs["zh"] != "西境" {
		t.Fatalf("row 1 %+v", first)
	}
	if first.Buildings["infrastructure"] != 3 || first.Buildings["arms_factory"] != 2 || first.Buildings["naval_base"] != nil {
		t.Fatalf("row 1 buildings %+v (province ones left out)", first.Buildings)
	}
	if first.Resources["steel"] != 5 || len(first.Cores) != 0 || first.VictoryPoints[1] != 5 {
		t.Fatalf("row 1 resources %+v cores %v (ZZZ is not the project's) points %v", first.Resources, first.Cores, first.VictoryPoints)
	}
	if rows[1].Locs["en"] == "" || rows[1].Owner != "AAA" || rows[1].Buildings["infrastructure"] != 2 {
		t.Fatalf("row 2 %+v", rows[1])
	}
	if got := modNamedStates(mod, req, 2); got != 1 {
		t.Fatalf("named states %d", got)
	}
}

func TestModMapImportNeedsItsStatesAndRegions(t *testing.T) {
	_, base, _ := provinceEditWorld(t)
	req := worldRequest{source: "mod", languages: []string{"en"}, date: "1936.1.1", modFiles: map[string][]byte{
		"history/states/1-West.txt":       []byte(`state={ id=1 history={ owner=AAA } provinces={ 1 } }`),
		"history/states/3-East.txt":       []byte(`state={ id=3 history={ owner=AAA } provinces={ 5 } }`),
		"map/strategicregions/1-West.txt": []byte(`strategic_region={ id=1 provinces={ 1 2 3 4 5 } }`),
	}}
	if _, _, problem, err := buildModWorld(base, req); err != nil || !strings.Contains(problemDetail(problem), "缺了 #2") {
		t.Fatalf("a gap in the state ids: %q %v", problem, err)
	}
}

func TestParseWorldModReadsFilesByPath(t *testing.T) {
	request := func(fields map[string]string, files map[string]string) *http.Request {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for name, value := range fields {
			_ = form.WriteField(name, value)
		}
		writeModBundle(form, files)
		_ = form.Close()
		r := httptest.NewRequest(http.MethodPost, "/", &body)
		r.Header.Set("Content-Type", form.FormDataContentType())
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		return r
	}
	files := map[string]string{
		"history/states/1-West.txt":       "state={ id=1 }",
		"map/strategicregions/1-West.txt": "strategic_region={ id=1 }",
	}
	req := worldRequest{upload: mapgenUploadWithDefinitions()}
	if problem := parseWorldMod(request(map[string]string{"tag_map": "kor=SHU, BBB=AAA"}, files), &req); problem != nil {
		t.Fatal(problem)
	}
	if string(req.modFiles["history/states/1-West.txt"]) != "state={ id=1 }" || string(req.modFiles["map/strategicregions/1-West.txt"]) != "strategic_region={ id=1 }" ||
		req.tagMap["KOR"] != "SHU" || req.tagMap["BBB"] != "AAA" || req.date != "1936.1.1" {
		t.Fatalf("files %v tag map %v date %q", req.modFiles, req.tagMap, req.date)
	}
	files["common/scripted_effects/x.txt"] = "x"
	if problem := parseWorldMod(request(nil, files), &worldRequest{upload: mapgenUploadWithDefinitions()}); !strings.Contains(problemDetail(problem), "common/scripted_effects") {
		t.Fatalf("a file outside the map folders: %q", problem)
	}
	delete(files, "common/scripted_effects/x.txt")
	if problem := parseWorldMod(request(nil, files), &worldRequest{}); !strings.Contains(problemDetail(problem), "definition.csv") {
		t.Fatalf("no definition.csv: %q", problem)
	}
}

func TestModMapImportTakesThousandsOfStateFiles(t *testing.T) {
	files := map[string]string{"map/strategicregions/1-All.txt": "strategic_region={ id=1 }"}
	for id := 1; id <= 1500; id++ {
		files[fmt.Sprintf("history/states/%d-State.txt", id)] = fmt.Sprintf("state={ id=%d }", id)
	}
	send := func(write func(form *multipart.Writer)) *http.Request {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		_ = form.WriteField("source", "mod")
		for _, field := range []string{"provinces", "heightmap", "terrain"} {
			part, _ := form.CreateFormFile(field, field+".png")
			_ = png.Encode(part, image.NewGray(image.Rect(0, 0, 1, 1)))
		}
		part, _ := form.CreateFormFile("definition", "definition.csv")
		_, _ = part.Write(mapgenUploadWithDefinitions().Definitions)
		write(form)
		_ = form.Close()
		r := httptest.NewRequest(http.MethodPost, "/", &body)
		r.Header.Set("Content-Type", form.FormDataContentType())
		return r
	}
	onePartEach := send(func(form *multipart.Writer) {
		for name, content := range files {
			part, _ := form.CreateFormFile("file:"+name, path.Base(name))
			_, _ = part.Write([]byte(content))
		}
	})
	if err := onePartEach.ParseMultipartForm(landSketchMaxBytes); err == nil {
		t.Fatal("a form of 1500 parts was read: the bundle is not needed")
	}
	req, problem := parseWorldRequest(httptest.NewRecorder(), send(func(form *multipart.Writer) { writeModBundle(form, files) }))
	if problem != nil {
		t.Fatalf("bundle refused: %+v", problem)
	}
	if len(req.modFiles) != len(files) || string(req.modFiles["history/states/1500-State.txt"]) != "state={ id=1500 }" {
		t.Fatalf("%d files read of %d", len(req.modFiles), len(files))
	}
}

func modTextUploadRequest(t *testing.T, write func(*multipart.Writer)) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	write(form)
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	if err := r.ParseMultipartForm(landSketchMaxBytes); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.MultipartForm.RemoveAll() })
	return r
}

func writeLegacyModFiles(form *multipart.Writer, files map[string]string) {
	for name, content := range files {
		part, _ := form.CreateFormFile("file:"+name, path.Base(name))
		_, _ = part.Write([]byte(content))
	}
}

func TestModTextUploadPreservesOldPagesAndRejectsMixedProtocols(t *testing.T) {
	files := map[string]string{
		"history/states/1-West.txt":         "state={ id=1 }\r\n",
		"map/strategicregions/1-West.txt":   "strategic_region={ id=1 }\n",
		"localisation/states_l_english.yml": "\uFEFFl_english:\r\n STATE_1:0 \"West\"\r\n",
	}
	legacy := modTextUploadRequest(t, func(form *multipart.Writer) {
		_ = form.WriteField("tag_map", "KOR=SHU")
		writeLegacyModFiles(form, files)
	})
	req := worldRequest{upload: mapgenUploadWithDefinitions()}
	if problem := parseWorldMod(legacy, &req); problem != nil {
		t.Fatalf("page opened before update can no longer import: %+v", problem)
	}
	if req.date != "1936.1.1" || req.tagMap["KOR"] != "SHU" || len(req.modFiles) != len(files) {
		t.Fatalf("legacy metadata changed: %+v", req)
	}
	for name, content := range files {
		if string(req.modFiles[name]) != content {
			t.Fatalf("legacy bytes changed for %s", name)
		}
	}
	for _, format := range []string{"bundle", "malformed_index", "empty_index", "file_index", "bundle_without_index"} {
		t.Run(format, func(t *testing.T) {
			r := modTextUploadRequest(t, func(form *multipart.Writer) {
				writeLegacyModFiles(form, files)
				switch format {
				case "bundle":
					writeModBundle(form, files)
				case "malformed_index":
					_ = form.WriteField(modIndexField, "{")
				case "empty_index":
					_ = form.WriteField(modIndexField, "")
				case "file_index":
					part, _ := form.CreateFormFile(modIndexField, "mod_index.json")
					_, _ = part.Write([]byte("[]"))
				case "bundle_without_index":
					part, _ := form.CreateFormFile(modBundleField, "mod_bundle.bin")
					_, _ = part.Write([]byte("invalid"))
				}
			})
			got, problem := readModBundle(r)
			if got != nil || problem == nil || problem.Code != problemBadRequest {
				t.Fatalf("new/malformed protocol fell back to old files: files=%d problem=%+v", len(got), problem)
			}
		})
	}
}

func TestModTextUploadFormatsSharePathAndByteLimits(t *testing.T) {
	for _, format := range []string{"legacy", "bundle"} {
		for _, check := range []string{"path", "file_size", "total_size"} {
			t.Run(format+"/"+check, func(t *testing.T) {
				files := map[string]string{}
				want := problemModFileOutside
				switch check {
				case "path":
					files["../../escape.txt"] = "not a map file"
				case "file_size":
					want = problemModFileUnreadable
					files["history/states/1.txt"] = strings.Repeat("x", modFileMaxBytes+1)
				case "total_size":
					want = problemModTextTooLarge
					body := strings.Repeat("x", modFileMaxBytes)
					for i := 0; i < modTextMaxBytes/modFileMaxBytes+1; i++ {
						files[fmt.Sprintf("history/states/%d.txt", i)] = body
					}
				}
				r := modTextUploadRequest(t, func(form *multipart.Writer) {
					if format == "bundle" {
						writeModBundle(form, files)
					} else {
						writeLegacyModFiles(form, files)
					}
				})
				got, problem := readModBundle(r)
				if got != nil || problem == nil || problem.Code != want {
					t.Fatalf("%s bypassed limit: files=%d problem=%+v want=%s", format, len(got), problem, want)
				}
			})
		}
	}
}

func writeModBundle(form *multipart.Writer, files map[string]string) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	index := make([]modBundleEntry, len(names))
	var bundle bytes.Buffer
	for i, name := range names {
		index[i] = modBundleEntry{Path: name, Size: int64(len(files[name]))}
		bundle.WriteString(files[name])
	}
	raw, _ := json.Marshal(index)
	_ = form.WriteField(modIndexField, string(raw))
	part, _ := form.CreateFormFile(modBundleField, "mod_bundle.bin")
	_, _ = part.Write(bundle.Bytes())
}

func mapgenUploadWithDefinitions() mapgen.UploadInput {
	return mapgen.UploadInput{Definitions: []byte("0;0;0;0;land;false;unknown;0\n")}
}

func TestModMapImportOfTheBaseGame(t *testing.T) {
	root := os.Getenv("HOI4_GAME_DIR")
	if root == "" {
		t.Skip("set HOI4_GAME_DIR to run against a real installation")
	}
	read := func(rel string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	picture := func(rel string) image.Image {
		t.Helper()
		img, _, err := image.Decode(bytes.NewReader(read(rel)))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		return img
	}
	game, err := mapgen.LoadBase(read("map/provinces.bmp"), read("map/heightmap.bmp"), read("map/terrain.bmp"), read("map/rivers.bmp"), read("map/definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	up, err := mapgen.ReadUpload(mapgen.UploadInput{
		Provinces: picture("map/provinces.bmp"), Heightmap: picture("map/heightmap.bmp"),
		Terrain: picture("map/terrain.bmp"), Rivers: picture("map/rivers.bmp"), Definitions: read("map/definition.csv"),
	}, game)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, dir := range []string{"history/states", "map/strategicregions"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			files[dir+"/"+e.Name()] = read(dir + "/" + e.Name())
		}
	}
	files["localisation/english/state_names_l_english.yml"] = read("localisation/english/state_names_l_english.yml")
	req := worldRequest{source: "mod", languages: []string{"en"}, date: "1936.1.1", modFiles: files, tagMap: map[string]string{}}
	world, mod, problem, err := buildModWorld(up.Base, req)
	if err != nil || problem != nil {
		t.Fatalf("build: %q %v", problem, err)
	}
	if len(world.All.States) != len(mod.facts) || len(world.All.States) < 1000 || len(world.Regions) < 300 {
		t.Fatalf("%d states (%d facts), %d regions", len(world.All.States), len(mod.facts), len(world.Regions))
	}
	if got := world.LoadRules(); len(got) != 0 {
		t.Fatalf("load rules: %+v", got)
	}

	if mod.facts[64].Owner != "GER" || mod.stateNames["en"][64] == "" || strings.HasPrefix(mod.stateNames["en"][64], "STATE_") {
		t.Fatalf("state 64: owner %q name %q", mod.facts[64].Owner, mod.stateNames["en"][64])
	}
	owner, capital, tags, unknown := modOwners(world, mod, req, append(guirenderVanillaTags(mod), "GER"))
	if len(unknown) != 0 || owner[64] != "GER" || capital["GER"] == 0 || len(tags) < 50 {
		t.Fatalf("owners: unknown %v, 64 %q, GER capital %d, %d tags", unknown, owner[64], capital["GER"], len(tags))
	}
}

func guirenderVanillaTags(mod *modWorld) []string {
	var out []string
	for _, fact := range mod.facts {
		if fact.Owner != "" {
			out = append(out, fact.Owner)
		}
	}
	return out
}

func TestModMapImportFixesAStateOfOnlyWater(t *testing.T) {
	_, base, _ := provinceEditWorld(t)
	req := worldRequest{source: "mod", languages: []string{"en"}, date: "1936.1.1", tagMap: map[string]string{}, modFiles: map[string][]byte{
		"history/states/1-Land.txt":       []byte(`state={ id=1 history={ owner=AAA } provinces={ 1 5 } }`),
		"history/states/2-Lake.txt":       []byte(`state={ id=2 history={ owner=AAA } provinces={ 2 } }`),
		"map/strategicregions/1-West.txt": []byte(`strategic_region={ id=1 provinces={ 1 2 5 } }`),
		"map/strategicregions/2-Sea.txt":  []byte(`strategic_region={ id=2 provinces={ 3 4 } }`),
	}}
	if _, _, problem, err := buildModWorld(base, req); err != nil || problem == nil || problem.Code != "mod_state_no_land" {
		t.Fatalf("without the fix: %+v %v", problem, err)
	}
	req.fixes = map[string]bool{"state_no_land": true}
	world, mod, problem, err := buildModWorld(base, req)
	if err != nil || problem != nil {
		t.Fatalf("with the fix: %+v %v", problem, err)
	}
	if mod.landedLakeStates != 1 || len(world.All.States) != 2 {
		t.Fatalf("landed %d, %d states", mod.landedLakeStates, len(world.All.States))
	}
	for _, st := range world.All.States {
		if len(st.Provinces) == 0 {
			t.Fatalf("a state with no land kept: %+v", world.All.States)
		}
	}
}

func TestParseWorldRequestReadsTheFixes(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("source", "upload")
	_ = form.WriteField("fix", "black_pixel, state_no_land,rm_rf")
	_ = form.WriteField("tags", "AAA")
	for _, field := range []string{"provinces", "heightmap", "terrain"} {
		part, _ := form.CreateFormFile(field, field+".png")
		_ = png.Encode(part, image.NewGray(image.Rect(0, 0, 1, 1)))
	}
	_ = form.Close()
	r := httptest.NewRequest(http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	req, problem := parseWorldRequest(httptest.NewRecorder(), r)
	if problem != nil {
		t.Fatal(problem)
	}
	if !reflect.DeepEqual(req.fixes, map[string]bool{"black_pixel": true, "state_no_land": true}) || !req.upload.FillStrayPixels {
		t.Fatalf("fixes %v, fill %v", req.fixes, req.upload.FillStrayPixels)
	}
}
