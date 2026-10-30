package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/hoi4/paradox"
)

type pb04Fixture struct {
	Project json.RawMessage              `json:"project"`
	Tables  map[string][]json.RawMessage `json:"tables"`
	Expect  struct {
		RootID         string `json:"root_id"`
		SubtypeID      string `json:"subtype_id"`
		CountryTag     string `json:"country_tag"`
		CharacterTag   string `json:"character_tag"`
		RootSupport    int    `json:"root_support"`
		VanillaSupport int    `json:"vanilla_support"`
		ReplacePath    string `json:"replace_path"`
	} `json:"expect"`
}

func loadPB04Fixture(t *testing.T) pb04Fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/p0b_pb04_minimal_politics_zip.json")
	if err != nil {
		t.Fatalf("read PB-04 fixture: %v", err)
	}
	var fixture pb04Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode PB-04 fixture: %v", err)
	}
	return fixture
}

func pb04Snapshot(t *testing.T, fixture pb04Fixture) Snapshot {
	t.Helper()
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 404, Project: fixture.Project,
		Tables: fixture.Tables,
	}
	fingerprint, err := SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatalf("fingerprint PB-04 snapshot: %v", err)
	}
	snapshot.Fingerprint = fingerprint
	return snapshot
}

func pb04FlagBlob(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 82, 52))
	for y := 0; y < 52; y++ {
		for x := 0; x < 82; x++ {
			pixel := color.NRGBA{R: 35, G: 72, B: 108, A: 255}
			if y >= 20 && y < 32 {
				pixel = color.NRGBA{R: 235, G: 239, B: 232, A: 255}
			}
			if x >= 36 && x < 46 && y >= 21 && y < 31 {
				pixel = color.NRGBA{R: 202, G: 165, B: 65, A: 255}
			}
			img.SetNRGBA(x, y, pixel)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("encode PB-04 flag fixture: %v", err)
	}
	return out.Bytes()
}

func pb04BuildNative(t *testing.T, snapshot Snapshot) NativeArtifact {
	t.Helper()
	flag := pb04FlagBlob(t)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) {
		return flag, nil
	})
	if err != nil {
		t.Fatalf("build PB-04 native artifact: %v", err)
	}
	return artifact
}

func pb04EntryBySuffix(t *testing.T, entries []Entry, suffix string) Entry {
	t.Helper()
	for _, entry := range entries {
		if strings.HasSuffix(entry.Path, suffix) {
			return entry
		}
	}
	var paths []string
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	t.Fatalf("PB-04 entry ending in %q not found; entries=%v", suffix, paths)
	return Entry{}
}

func pb04ParseSource(t *testing.T, label string, body []byte) {
	t.Helper()
	_, recoveries, err := paradox.Parse(string(body))
	if failed, detail := clausewitz.ParseErrorMessage(recoveries, err); failed {
		t.Fatalf("PB-04 %s source readback failed: %s", label, detail)
	}
}

func pb04ZIPEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open PB-04 ZIP: %v", err)
	}
	defer reader.Close()
	out := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open PB-04 ZIP entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read PB-04 ZIP entry %s: %v", file.Name, err)
		}
		out[file.Name] = body
	}
	return out
}

func TestPB04MinimalPoliticsZIPSourceReadback(t *testing.T) {
	fixture := loadPB04Fixture(t)
	snapshot := pb04Snapshot(t, fixture)
	artifact := pb04BuildNative(t, snapshot)
	root := artifactRoot(artifact)

	outer := pb04EntryBySuffix(t, artifact.Entries, root+".mod")
	inner := pb04EntryBySuffix(t, artifact.Entries, "/descriptor.mod")
	for label, body := range map[string][]byte{"outer descriptor": outer.Bytes, "inner descriptor": inner.Bytes} {
		if got := strings.Count(string(body), `replace_path="`+fixture.Expect.ReplacePath+`"`); got != 1 {
			t.Fatalf("PB-04 %s replace_path count=%d, want 1:\n%s", label, got, body)
		}
		if strings.Contains(string(body), "dependencies={") {
			t.Fatalf("PB-04 must be standalone/no-DLC, but %s contains dependencies:\n%s", label, body)
		}
	}

	ideology := pb04EntryBySuffix(t, artifact.Entries, "/common/ideologies/00_ideologies.txt")
	pb04ParseSource(t, "ideologies", ideology.Bytes)
	for _, rootID := range []string{"democratic", "communism", "fascism", "neutrality", fixture.Expect.RootID} {
		if got := strings.Count(string(ideology.Bytes), "\t"+rootID+" = {"); got != 1 {
			t.Fatalf("PB-04 ROOT %s definitions=%d, want 1", rootID, got)
		}
	}
	if got := strings.Count(string(ideology.Bytes), "\t\t"+fixture.Expect.SubtypeID+" = {"); got != 1 {
		t.Fatalf("PB-04 subtype definitions=%d, want 1", got)
	}
	if strings.Contains(string(ideology.Bytes), "\t\t\t"+fixture.Expect.SubtypeID+" = {\n\t\t\t\tcan_be_randomly_selected = no") {
		t.Fatalf("PB-04 random-enabled subtype was exported as disabled:\n%s", ideology.Bytes)
	}

	for _, suffix := range []string{
		"/gfx/flags/GER_" + fixture.Expect.RootID + ".tga",
		"/gfx/flags/medium/GER_" + fixture.Expect.RootID + ".tga",
		"/gfx/flags/small/GER_" + fixture.Expect.RootID + ".tga",
	} {
		flag := pb04EntryBySuffix(t, artifact.Entries, suffix)
		if len(flag.Bytes) < 18 || flag.Bytes[2] != 2 || flag.Bytes[16] != 32 {
			t.Fatalf("PB-04 flag %s is not an uncompressed 32-bit TGA", flag.Path)
		}
	}

	character := pb04EntryBySuffix(t, artifact.Entries, "/common/characters/GER_GER.txt")
	pb04ParseSource(t, "character", character.Bytes)
	characterText := string(character.Bytes)
	for _, want := range []string{fixture.Expect.CharacterTag, "country_leader = {", "ideology = " + fixture.Expect.SubtypeID} {
		if !strings.Contains(characterText, want) {
			t.Fatalf("PB-04 character source missing %q:\n%s", want, characterText)
		}
	}

	history := pb04EntryBySuffix(t, artifact.Entries, "/history/countries/GER - z"+root+".txt")
	pb04ParseSource(t, "country history", history.Bytes)
	historyText := string(history.Bytes)
	recruit := "recruit_character = " + fixture.Expect.CharacterTag
	politics := "set_politics = {"
	popularity := "set_popularities = {"
	if strings.Count(historyText, recruit) != 1 || strings.Count(historyText, politics) != 1 ||
		strings.Count(historyText, popularity) != 1 {
		t.Fatalf("PB-04 history blocks are not unique:\n%s", historyText)
	}
	if !(strings.Index(historyText, recruit) < strings.Index(historyText, politics) &&
		strings.Index(historyText, politics) < strings.Index(historyText, popularity)) {
		t.Fatalf("PB-04 history order must be recruit -> set_politics -> set_popularities:\n%s", historyText)
	}
	for _, want := range []string{
		"ruling_party = " + fixture.Expect.RootID,
		fixture.Expect.RootID + " = " + strconv.Itoa(fixture.Expect.RootSupport),
		"democratic = " + strconv.Itoa(fixture.Expect.VanillaSupport),
	} {
		if !strings.Contains(historyText, want) {
			t.Fatalf("PB-04 history missing %q:\n%s", want, historyText)
		}
	}
	if last := finalPhysicalLine(historyText); strings.HasPrefix(last, "recruit_character") {
		t.Fatalf("PB-04 recruit_character is the final physical line: %q", last)
	}

	for _, suffix := range []string{
		"/localisation/english/replace/GER_characters_l_english.yml",
		"/localisation/simp_chinese/replace/GER_characters_l_simp_chinese.yml",
		"_ideology_l_english.yml",
		"_ideology_l_simp_chinese.yml",
	} {
		entry := pb04EntryBySuffix(t, artifact.Entries, suffix)
		if !bytes.HasPrefix(entry.Bytes, []byte("\xef\xbb\xbf")) {
			t.Fatalf("PB-04 localisation %s is not UTF-8 BOM", entry.Path)
		}
	}
	if body := string(pb04EntryBySuffix(t, artifact.Entries, "_ideology_l_english.yml").Bytes); !strings.Contains(body, "Civic Reformism") ||
		!strings.Contains(body, fixture.Expect.RootID+`_noun:0 "Civic Coalition"`) ||
		!strings.Contains(body, fixture.Expect.RootID+`_desc:0 "PB-04 project-managed political group."`) {
		t.Fatalf("PB-04 English ROOT/subtype localisation missing:\n%s", body)
	}
	if body := string(pb04EntryBySuffix(t, artifact.Entries, "_ideology_l_simp_chinese.yml").Bytes); !strings.Contains(body, "公民改革主义") ||
		!strings.Contains(body, fixture.Expect.RootID+`_noun:0 "公民联合阵营"`) ||
		!strings.Contains(body, fixture.Expect.RootID+`_desc:0 "PB-04 项目管理政治阵营。"`) {
		t.Fatalf("PB-04 Chinese ROOT/subtype localisation missing:\n%s", body)
	}

	for _, entry := range artifact.Entries {
		normalized := strings.ToLower(entry.Path)
		if strings.Contains(normalized, "/common/technologies/") ||
			strings.Contains(normalized, "/interface/countrytechtreeview.gui") {
			t.Fatalf("technology content leaked into politics-only ZIP: %s", entry.Path)
		}
	}

	destination := os.Getenv("P0B_PB04_ZIP_OUT")
	if destination == "" {
		destination = filepath.Join(t.TempDir(), "P0B_PB04_MINIMAL_POLITICS.zip")
	}
	manifest, err := Build(destination, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write PB-04 ZIP: %v", err)
	}
	zipEntries := pb04ZIPEntries(t, destination)
	if len(zipEntries) != len(artifact.Entries)+1 {
		t.Fatalf("PB-04 ZIP entry count=%d, want %d", len(zipEntries), len(artifact.Entries)+1)
	}
	for _, entry := range artifact.Entries {
		if !bytes.Equal(zipEntries[entry.Path], entry.Bytes) {
			t.Fatalf("PB-04 ZIP readback differs at %s", entry.Path)
		}
	}
	embedded := zipEntries[manifest.ManifestPath]
	var manifestDoc struct {
		Schema      string `json:"schema"`
		Fingerprint string `json:"fingerprint"`
		EntryCount  int    `json:"entry_count"`
	}
	if err := json.Unmarshal(embedded, &manifestDoc); err != nil {
		t.Fatalf("read PB-04 embedded manifest: %v", err)
	}
	if manifestDoc.Schema != ArtifactManifestSchema || manifestDoc.Fingerprint != snapshot.Fingerprint ||
		manifestDoc.EntryCount != len(artifact.Entries)+1 {
		t.Fatalf("PB-04 embedded manifest drift: %+v", manifestDoc)
	}

	duplicate := filepath.Join(t.TempDir(), "determinism.zip")
	second, err := Build(duplicate, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write deterministic PB-04 comparison ZIP: %v", err)
	}
	if second.ZIPSHA256 != manifest.ZIPSHA256 || second.ZIPBytes != manifest.ZIPBytes {
		t.Fatalf("PB-04 ZIP is not deterministic: first=%+v second=%+v", manifest, second)
	}
	rawZIP, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read final PB-04 ZIP: %v", err)
	}
	digest := sha256.Sum256(rawZIP)
	if got := hex.EncodeToString(digest[:]); got != manifest.ZIPSHA256 {
		t.Fatalf("PB-04 returned ZIP hash=%s, actual=%s", manifest.ZIPSHA256, got)
	}
	t.Logf("PB04_ZIP path=%s bytes=%d sha256=%s fingerprint=%s entries=%d",
		destination, manifest.ZIPBytes, manifest.ZIPSHA256, snapshot.Fingerprint, len(zipEntries))
}

func TestPB04IdeologyTakeoverIsFailClosed(t *testing.T) {
	fixture := loadPB04Fixture(t)
	var project map[string]any
	if err := json.Unmarshal(fixture.Project, &project); err != nil {
		t.Fatal(err)
	}
	project["dependencies"] = []any{"A Base Mod"}
	projectRaw, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Project = projectRaw
	artifact := pb04BuildNative(t, pb04Snapshot(t, fixture))
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, "/common/ideologies/00_ideologies.txt") {
			t.Fatalf("PB-04 submod negative control emitted a ROOT takeover: %s", entry.Path)
		}
		if strings.HasSuffix(entry.Path, "/descriptor.mod") &&
			strings.Contains(string(entry.Bytes), `replace_path="common/ideologies"`) {
			t.Fatalf("PB-04 submod negative control emitted destructive replace_path:\n%s", entry.Bytes)
		}
	}
}
