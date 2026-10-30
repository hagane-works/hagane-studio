package history

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

const countryFixture = `capital = 1
# 1964 must see the first two blocks, but not its own timestamp.
1936.1.1 = { set_politics = { ruling_party = democratic } }
1940.1.1.0 = { add_ideas = { test_idea = 1 } }
1964.5.1.12 = { set_country_flag = at_start }
name = "a date 1938.1.1 = { is only text }"
`

func hashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func baseCountry(tag, content string) CountryHistory {
	hash := hashText(content)
	return CountryHistory{
		Tag: tag, RelativePath: "history/countries/" + tag + " - Test.txt", Content: content,
		CorpusSHA256: hash, ObservedSHA256: hash, Origin: BaseCountry, OwnedStateCount: 1,
	}
}

func standaloneInput(start string, country CountryHistory) AnalysisInput {
	return AnalysisInput{
		WorldMode: UseMyScenarioWorld, EffectiveStartDate: start,
		Provider:  ProviderEvidence{Kind: ProviderVanilla, Standalone: true, GameVersion: "1.19.2", CorpusVersion: "1.19.2", ParserVersion: "history-scan-v1"},
		Countries: []CountryHistory{country},
	}
}

func codes(findings []Finding) map[string]bool {
	result := make(map[string]bool, len(findings))
	for _, item := range findings {
		result[item.Code] = true
	}
	return result
}

func TestScenarioHistoryInherit1939DoesNotProduceRepair(t *testing.T) {
	in := standaloneInput("1939.1.1.12", baseCountry("GER", countryFixture))
	in.WorldMode = InheritBaseWorld
	plan := BuildScenarioHistoryPlan(in)
	if plan.Preview.SHA256 != "" {
		t.Fatalf("inherit preview = %#v, want empty", plan.Preview)
	}
}

func TestScenarioHistoryStandalone1964ProducesReadOnlyRisk(t *testing.T) {
	in := standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture))
	plan := BuildScenarioHistoryPlan(in)
	if plan.Preview.SHA256 == "" || plan.Preview.ExecutableDatedBlockCount != 2 {
		t.Fatalf("preview = %#v, want 2-block read-only preview", plan.Preview)
	}
	if !codes(plan.Findings)["scenario.base_history_will_execute"] {
		t.Fatalf("findings = %#v, want player-readable preview finding", plan.Findings)
	}
	if len(plan.Corpus) != 1 || !plan.Corpus[0].HashVerified || !plan.Corpus[0].ParserSucceeded {
		t.Fatalf("corpus = %#v, want verified parsed provenance", plan.Corpus)
	}
	if plan.Corpus[0].ProviderKind != ProviderVanilla || plan.Corpus[0].GameVersion != "1.19.2" || plan.Corpus[0].RelativePath == "" || plan.Corpus[0].ContentSHA256 == "" || plan.Corpus[0].ParserVersion != "history-scan-v1" {
		t.Fatalf("corpus = %#v, want version-locked relative provenance", plan.Corpus[0])
	}
	if len(plan.Preview.Risks) != 1 || len(plan.Preview.Risks[0].ExecutableDatedBlocks) != 2 {
		t.Fatalf("risks = %#v, want dated-block evidence without source bytes", plan.Preview.Risks)
	}
}

func TestScenarioHistoryFailsClosedForSubmod(t *testing.T) {
	in := standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture))
	in.Provider = ProviderEvidence{Kind: ProviderSubmod, Standalone: false, GameVersion: "1.19.2", CorpusVersion: "1.19.2"}
	plan := BuildScenarioHistoryPlan(in)
	if !codes(plan.Findings)["scenario.history_source_unregistered"] {
		t.Fatalf("plan = %#v, want unregistered-source finding", plan)
	}
}

func TestScenarioHistoryFailsClosedForHashMismatch(t *testing.T) {
	country := baseCountry("GER", countryFixture)
	country.ObservedSHA256 = hashText("different source")
	plan := BuildScenarioHistoryPlan(standaloneInput("1964.5.1.12", country))
	if !codes(plan.Findings)["scenario.base_version_mismatch"] {
		t.Fatalf("plan = %#v, want hash mismatch finding", plan)
	}
}

func TestScenarioHistoryFailsClosedForGameVersionMismatch(t *testing.T) {
	in := standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture))
	in.Provider.CorpusVersion = "1.19.1"
	plan := BuildScenarioHistoryPlan(in)
	if !codes(plan.Findings)["scenario.base_version_mismatch"] {
		t.Fatalf("plan = %#v, want version mismatch finding", plan)
	}
}

func TestScenarioHistoryFailsClosedWithoutParserVersion(t *testing.T) {
	in := standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture))
	in.Provider.ParserVersion = ""
	plan := BuildScenarioHistoryPlan(in)
	if !codes(plan.Findings)["scenario.history_corpus_parser_unverified"] {
		t.Fatalf("plan = %#v, want parser-version finding", plan)
	}
}

func TestScenarioHistoryReadOnlyPlanHasNoConfirmationFinding(t *testing.T) {
	plan := BuildScenarioHistoryPlan(standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture)))
	if codes(plan.Findings)["scenario.history_repair_not_confirmed"] || codes(plan.Findings)["scenario.history_confirmation_stale"] {
		t.Fatalf("read-only plan exposed repair confirmation findings: %#v", plan.Findings)
	}
}

func TestScenarioHistoryFailsClosedForParseFailure(t *testing.T) {
	country := baseCountry("GER", `1936.1.1 = { set_politics = { ruling_party = democratic }`)
	plan := BuildScenarioHistoryPlan(standaloneInput("1964.5.1.12", country))
	if !codes(plan.Findings)["scenario.history_parse_failed"] {
		t.Fatalf("plan = %#v, want parse failure finding", plan)
	}
}

func TestTopLevelDatedBlocksUseStrictComparisonAndIgnoreQuotedText(t *testing.T) {
	blocks, err := ExtractTopLevelDatedBlocks(countryFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 3 {
		t.Fatalf("blocks = %#v, want three top-level blocks", blocks)
	}
	in := standaloneInput("1964.5.1.12", baseCountry("GER", countryFixture))
	preview := BuildScenarioHistoryPlan(in)
	if preview.Preview.ExecutableDatedBlockCount != 2 {
		t.Fatalf("strict compare count = %d, want 2", preview.Preview.ExecutableDatedBlockCount)
	}
}

func TestZeroStateFindings(t *testing.T) {
	base := baseCountry("GER", countryFixture)
	base.OwnedStateCount = 0
	custom := CountryHistory{Tag: "ZZZ", Origin: CustomCountry, OwnedStateCount: 0}
	in := standaloneInput("1939.1.1.12", base)
	in.Countries = append(in.Countries, custom)
	plan := BuildScenarioHistoryPlan(in)
	got := codes(plan.Findings)
	if !got["scenario.zero_state_base_country"] || !got["scenario.zero_state_custom_country_missing_capital"] {
		t.Fatalf("findings = %#v, want both zero-state findings", plan.Findings)
	}
}

func TestTopLevelDatedBlocksRejectUnbalancedOrQuotedInput(t *testing.T) {
	if _, err := ExtractTopLevelDatedBlocks(`1936.1.1 = { flag = "unterminated }`); err == nil {
		t.Fatal("unterminated quote unexpectedly parsed")
	}
	if _, err := ExtractTopLevelDatedBlocks(`1936.1.1 = { flag = yes`); err == nil {
		t.Fatal("unbalanced brace unexpectedly parsed")
	}
}
