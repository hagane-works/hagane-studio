package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func ideaArtifact(t *testing.T, project string, ideaRow string) NativeArtifact {
	t.Helper()
	snapshot := Snapshot{ProjectID: 42, Project: []byte(project), Tables: map[string][]json.RawMessage{
		"ideas": {json.RawMessage(ideaRow)},
	}}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a dropped picture must never fail the export: %v", err)
	}
	return artifact
}

func ideaPictureFindings(t *testing.T, artifact NativeArtifact) []ExportFinding {
	t.Helper()
	var out []ExportFinding
	seen := 0
	for _, entry := range artifact.Entries {
		if !isExportFindingsPath(entry.Path) {
			continue
		}
		seen++
		doc, err := DecodeExportFindingsDocument(entry.Bytes)
		if err != nil {
			t.Fatalf("export findings record does not decode: %v\n%s", err, entry.Bytes)
		}
		for _, finding := range doc.Findings {
			if finding.Code == IdeaPictureDroppedCode {
				out = append(out, finding)
			}
		}
	}
	if seen > 1 {
		t.Fatalf("more than one export findings record in the artifact")
	}
	return out
}

func ideaDropWarnings(artifact NativeArtifact) []string {
	var out []string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "Idea picture left out") {
			out = append(out, w)
		}
	}
	return out
}

func TestDroppedIdeaPictureShipsFindingAndWarning(t *testing.T) {
	artifact := ideaArtifact(t, ideaTestProject,
		`{"id":7,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_my_custom_art","source":"created"}`)

	var script string
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/common/ideas/ger_ger.txt" {
			script = string(entry.Bytes)
		}
	}
	if script == "" {
		t.Fatalf("idea script missing: %#v", artifact.Entries)
	}
	if strings.Contains(script, "picture =") {
		t.Fatalf("an unshippable picture must stay out of the artifact:\n%s", script)
	}

	findings := ideaPictureFindings(t, artifact)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one %s finding in the shipped record, got %d", IdeaPictureDroppedCode, len(findings))
	}
	finding := findings[0]
	if finding.Severity != "warning" {
		t.Fatalf("a value the author typed that provably did not reach the game is a warning, got %q", finding.Severity)
	}
	if finding.Path != nil || finding.Line != nil {
		t.Fatalf("entity-level finding must say null rather than invent a file/line: path=%v line=%v", finding.Path, finding.Line)
	}
	if !strings.Contains(finding.Message, "GER_demo") || !strings.Contains(finding.Message, "GFX_idea_my_custom_art") {
		t.Fatalf("message must name the idea and the typed picture: %q", finding.Message)
	}
	if strings.TrimSpace(finding.PlayerImpact) == "" || !strings.Contains(finding.PlayerImpact, "?") {
		t.Fatalf("player impact must say what the player sees: %q", finding.PlayerImpact)
	}
	if strings.TrimSpace(finding.EditRoute) == "" {
		t.Fatalf("edit route must not be empty")
	}
	if finding.SourceEvidence["idea_id"] != "GER_demo" || finding.SourceEvidence["picture"] != "GFX_idea_my_custom_art" {
		t.Fatalf("source evidence must carry idea id and picture as fields: %#v", finding.SourceEvidence)
	}
	if !strings.Contains(string(finding.Source), "idea_picture_resolve") {
		t.Fatalf("source must be traceable to this producer: %s", finding.Source)
	}

	if warnings := ideaDropWarnings(artifact); len(warnings) != 1 {
		t.Fatalf("native_warnings must still carry the drop sentence, got %#v", artifact.Warnings)
	}
}

func TestDroppedIdeaPictureAdviceIsNotCircular(t *testing.T) {
	artifact := ideaArtifact(t, ideaTestProject,
		`{"id":7,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_my_custom_art","source":"created"}`)
	findings := ideaPictureFindings(t, artifact)
	if len(findings) != 1 {
		t.Fatalf("expected one drop finding, got %d", len(findings))
	}
	banned := []string{"use a vanilla picture name", "vanilla picture name"}
	texts := map[string]string{
		"message":       findings[0].Message,
		"player_impact": findings[0].PlayerImpact,
		"edit_route":    findings[0].EditRoute,
	}
	for _, w := range ideaDropWarnings(artifact) {
		texts["native_warning"] = w
	}
	for field, text := range texts {
		for _, phrase := range banned {
			if strings.Contains(strings.ToLower(text), phrase) {
				t.Fatalf("%s still carries the circular advice %q: %s", field, phrase, text)
			}
		}
	}

	route := strings.ToLower(findings[0].EditRoute)
	for _, must := range []string{"upload", "base-game sprite", "interface/*.gfx"} {
		if !strings.Contains(route, must) {
			t.Fatalf("edit route is missing the %q escape hatch: %s", must, findings[0].EditRoute)
		}
	}
}

func TestVanillaIdeaPictureProducesNoFinding(t *testing.T) {
	artifact := ideaArtifact(t, ideaTestProject,
		`{"id":7,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"generic_agrarian_society","source":"created"}`)
	var script string
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/common/ideas/ger_ger.txt" {
			script = string(entry.Bytes)
		}
	}
	if !strings.Contains(script, "picture = generic_agrarian_society") {
		t.Fatalf("a stock base-game sprite must still be referenced:\n%s", script)
	}
	if findings := ideaPictureFindings(t, artifact); len(findings) != 0 {
		t.Fatalf("no finding may be raised for a working picture: %#v", findings)
	}
	if warnings := ideaDropWarnings(artifact); len(warnings) != 0 {
		t.Fatalf("no drop warning may be raised for a working picture: %#v", warnings)
	}
}

func TestSubmodIdeaPictureProducesNoFinding(t *testing.T) {
	artifact := ideaArtifact(t, ideaTestSubmod,
		`{"id":7,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_GER_to_banish_want","source":"created"}`)
	var script string
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/common/ideas/ger_ger.txt" {
			script = string(entry.Bytes)
		}
	}
	if !strings.Contains(script, "picture = GER_to_banish_want") {
		t.Fatalf("submod base-mod sprite reference was dropped:\n%s", script)
	}
	if findings := ideaPictureFindings(t, artifact); len(findings) != 0 {
		t.Fatalf("submod must not report a drop: %#v", findings)
	}
	if warnings := ideaDropWarnings(artifact); len(warnings) != 0 {
		t.Fatalf("submod must not warn: %#v", warnings)
	}
}

func TestDroppedIdeaPictureCarriesSuggestions(t *testing.T) {
	artifact := ideaArtifact(t, ideaTestProject,
		`{"id":7,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"war_ecnomy","source":"created"}`)
	findings := ideaPictureFindings(t, artifact)
	if len(findings) != 1 {
		t.Fatalf("expected one drop finding for a typo, got %d", len(findings))
	}
	if !strings.Contains(findings[0].EditRoute, "war_economy") {
		t.Fatalf("edit route must name the sprite the author probably meant: %s", findings[0].EditRoute)
	}
	suggestions, _ := findings[0].SourceEvidence["suggestions"].([]any)
	if len(suggestions) == 0 || suggestions[0] != "war_economy" {
		t.Fatalf("source evidence must carry the ranked suggestions as data: %#v", findings[0].SourceEvidence)
	}
	var warned bool
	for _, w := range ideaDropWarnings(artifact) {
		if strings.Contains(w, "Did you mean: war_economy") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("native_warnings must carry the same hint: %#v", artifact.Warnings)
	}
}

func TestIdeaPictureDropTextsWithoutSuggestions(t *testing.T) {
	drop := IdeaPictureDrop{IdeaID: "GER_demo", Picture: "GFX_idea_zzz", Stripped: "zzz"}
	for _, text := range []string{ideaPictureDropWarning(drop, english), drop.Message(english), drop.EditRoute(english)} {
		if strings.Contains(text, "Did you mean: ?") || strings.Contains(text, "closest existing names: )") {
			t.Fatalf("empty suggestion list rendered: %s", text)
		}
	}
	withSuggestions := drop
	withSuggestions.Suggestions = []string{"aaa", "bbb"}
	if !strings.Contains(ideaPictureDropWarning(withSuggestions, english), "Did you mean: aaa, bbb?") {
		t.Fatalf("suggestions missing from the warning: %s", ideaPictureDropWarning(withSuggestions, english))
	}
	if !strings.Contains(withSuggestions.EditRoute(english), "closest existing names: aaa, bbb") {
		t.Fatalf("suggestions missing from the edit route: %s", withSuggestions.EditRoute(english))
	}
}
