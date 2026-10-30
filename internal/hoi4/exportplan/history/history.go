package history

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

type ScenarioWorldMode string

const (
	InheritBaseWorld ScenarioWorldMode = "inherit_base_world"

	UseMyScenarioWorld ScenarioWorldMode = "use_my_scenario_world"
)

type ProviderKind string

const (
	ProviderVanilla ProviderKind = "vanilla"
	ProviderBaseMod ProviderKind = "base_mod"
	ProviderSubmod  ProviderKind = "submod"
)

type CountryOrigin string

const (
	BaseCountry   CountryOrigin = "base_country"
	CustomCountry CountryOrigin = "custom_country"
)

type ProviderEvidence struct {
	Kind          ProviderKind
	Standalone    bool
	GameVersion   string
	CorpusVersion string

	ParserVersion string
}

type CountryHistory struct {
	Tag                    string
	RelativePath           string
	Content                string
	CorpusSHA256           string
	ObservedSHA256         string
	Origin                 CountryOrigin
	OwnedStateCount        int
	FallbackCapitalStateID *int
}

type AnalysisInput struct {
	WorldMode          ScenarioWorldMode
	EffectiveStartDate string
	Provider           ProviderEvidence
	Countries          []CountryHistory
}

type Finding struct {
	Code       string
	Severity   string
	Title      string
	Reason     string
	Suggestion string
	Tag        string
	Path       string
}

type DatedBlock struct {
	Date        string
	StartOffset int
	EndOffset   int
	SHA256      string
}

type CorpusProvenance struct {
	ProviderKind    ProviderKind
	GameVersion     string
	RelativePath    string
	CorpusSHA256    string
	ObservedSHA256  string
	ContentSHA256   string
	ParserVersion   string
	HashVerified    bool
	ParserSucceeded bool
}

type CountryHistoryRisk struct {
	Tag                   string
	ExecutableDatedBlocks []DatedBlock
}

type Preview struct {
	SHA256                    string
	AffectedCountries         []string
	ExecutableDatedBlockCount int
	PlayerSummary             string
	Risks                     []CountryHistoryRisk
}

type ScenarioHistoryPlan struct {
	EffectiveStartDate string
	WorldMode          ScenarioWorldMode
	Preview            Preview
	Corpus             []CorpusProvenance
	Findings           []Finding
}

type hoi4Date [4]int

func parseHOI4Date(value string) (hoi4Date, bool) {
	v := strings.TrimSpace(value)
	parts := strings.Split(v, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return hoi4Date{}, false
	}
	values := [4]int{0, 0, 0, 12}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return hoi4Date{}, false
		}
		values[i] = n
	}
	if values[0] <= 0 || values[1] < 1 || values[1] > 12 || values[2] < 1 || values[2] > 31 || values[3] < 0 || values[3] > 23 {
		return hoi4Date{}, false
	}
	return hoi4Date(values), true
}

func normaliseDate(value hoi4Date) string {
	return strconv.Itoa(value[0]) + "." + strconv.Itoa(value[1]) + "." + strconv.Itoa(value[2]) + "." + strconv.Itoa(value[3])
}

func dateLess(left, right hoi4Date) bool {
	for i := range left {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return false
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameSHA256(left, right string) bool {
	return validSHA256(left) && validSHA256(right) && strings.EqualFold(left, right)
}

func finding(code, severity, title, reason, suggestion, tag, path string) Finding {
	return Finding{
		Code: code, Severity: severity, Title: title, Reason: reason,
		Suggestion: suggestion, Tag: tag, Path: path,
	}
}

func BuildScenarioHistoryPlan(input AnalysisInput) ScenarioHistoryPlan {
	plan := ScenarioHistoryPlan{
		EffectiveStartDate: input.EffectiveStartDate,
		WorldMode:          input.WorldMode,
		Corpus:             collectCorpusProvenance(input.Provider, input.Countries),
		Findings:           []Finding{},
	}
	start, ok := parseHOI4Date(input.EffectiveStartDate)
	if !ok {
		plan.Findings = append(plan.Findings, finding(
			"timeline.project_start_date_unknown", "attention", "无法读取剧本开局日期",
			"钢之工坊无法确定哪些原版世界变化会在开局前发生。", "请设置有效的剧本开局日期。", "", "",
		))
		return plan
	}
	plan.EffectiveStartDate = normaliseDate(start)

	for _, country := range sortedCountries(input.Countries) {
		if country.OwnedStateCount != 0 {
			continue
		}
		switch country.Origin {
		case BaseCountry:
			plan.Findings = append(plan.Findings, finding(
				"scenario.zero_state_base_country", "attention", "一个原版国家没有开局州",
				"“"+country.Tag+"”在你的开局世界中没有州，但原版历史仍可能引用它。", "请检查该国家是否应保留、何时出现，以及首都是否有效。", country.Tag, country.RelativePath,
			))
		case CustomCountry:
			if country.FallbackCapitalStateID == nil || *country.FallbackCapitalStateID <= 0 {
				plan.Findings = append(plan.Findings, finding(
					"scenario.zero_state_custom_country_missing_capital", "attention", "一个自定义国家没有开局州或备用首都",
					"“"+country.Tag+"”可能在事件中出现，但钢之工坊无法猜测它应使用哪个首都州。", "请为该国家设置有效的备用首都州。", country.Tag, country.RelativePath,
				))
			}
		}
	}

	if input.WorldMode == InheritBaseWorld {
		return plan
	}
	if input.WorldMode != UseMyScenarioWorld {
		plan.Findings = append(plan.Findings, finding(
			"scenario.history_policy_required", "attention", "请选择开局世界设定",
			"钢之工坊需要知道你是沿用基础世界，还是使用自己的剧本世界。", "在项目设置中选择开局时使用哪套世界设定。", "", "",
		))
		return plan
	}
	if input.Provider.Kind != ProviderVanilla || !input.Provider.Standalone {
		plan.Findings = append(plan.Findings, finding(
			"scenario.history_source_unregistered", "attention", "基础世界来源尚未登记",
			"当前项目依赖其他 MOD，但钢之工坊还没有这套基础世界的可验证资料。", "请登记或导入基础 MOD 来源；程序只做分析，不会替你改写历史。", "", "",
		))
		return plan
	}

	if len(plan.Corpus) == 0 {
		plan.Findings = append(plan.Findings, finding(
			"scenario.history_corpus_missing", "attention", "没有可验证的原版世界资料",
			"钢之工坊没有找到可安全分析的原版国家历史，因此不会自动改变开局世界。", "请更新世界资料后重新检查。", "", "",
		))
		return plan
	}
	if input.Provider.GameVersion == "" || input.Provider.CorpusVersion == "" || input.Provider.GameVersion != input.Provider.CorpusVersion {
		plan.Findings = append(plan.Findings, finding(
			"scenario.base_version_mismatch", "attention", "游戏版本与世界资料不一致",
			"钢之工坊无法确认这份原版世界资料是否适用于当前游戏版本。", "请使用匹配的游戏版本，或等待程序更新世界资料。", "", "",
		))
		return plan
	}
	if strings.TrimSpace(input.Provider.ParserVersion) == "" {
		plan.Findings = append(plan.Findings, finding(
			"scenario.history_corpus_parser_unverified", "attention", "世界资料缺少读取版本信息",
			"钢之工坊无法确认这份世界资料是用哪一版规则读取的，因此不会自动改变开局世界。", "请更新世界资料后重新检查。", "", "",
		))
		return plan
	}
	risks, parserFindings, hashFindings := analyseRisksWithProvenance(start, input.Countries, &plan.Corpus)
	plan.Findings = append(plan.Findings, hashFindings...)
	if len(hashFindings) > 0 {
		return plan
	}
	plan.Findings = append(plan.Findings, parserFindings...)
	if len(parserFindings) > 0 {
		return plan
	}
	plan.Preview = makePreviewWithProvenance(plan.EffectiveStartDate, risks, plan.Corpus)
	if len(risks) == 0 {
		return plan
	}
	plan.Findings = append(plan.Findings, finding(
		"scenario.base_history_will_execute", "attention", "基础世界的部分变化会在开局前生效",
		"你的剧本从 "+plan.EffectiveStartDate+" 开始；钢之工坊已列出可能改变开局世界的历史段。", "查看来源和影响后，在自己的剧本内容中决定如何处理。", "", "",
	))
	return plan
}

func sortedCountries(countries []CountryHistory) []CountryHistory {
	ordered := append([]CountryHistory(nil), countries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].RelativePath != ordered[j].RelativePath {
			return ordered[i].RelativePath < ordered[j].RelativePath
		}
		return ordered[i].Tag < ordered[j].Tag
	})
	return ordered
}

func collectCorpusProvenance(provider ProviderEvidence, countries []CountryHistory) []CorpusProvenance {
	provenance := make([]CorpusProvenance, 0)
	for _, country := range sortedCountries(countries) {
		if country.Origin != BaseCountry {
			continue
		}
		contentHash := sha256Hex(country.Content)
		provenance = append(provenance, CorpusProvenance{
			ProviderKind: provider.Kind, GameVersion: provider.GameVersion,
			RelativePath: country.RelativePath, CorpusSHA256: country.CorpusSHA256,
			ObservedSHA256: country.ObservedSHA256, ContentSHA256: contentHash,
			ParserVersion: provider.ParserVersion,
			HashVerified:  sameSHA256(country.CorpusSHA256, country.ObservedSHA256) && sameSHA256(country.ObservedSHA256, contentHash),
		})
	}
	return provenance
}

func provenanceIndex(provenance []CorpusProvenance, path string) int {
	for i := range provenance {
		if provenance[i].RelativePath == path {
			return i
		}
	}
	return -1
}

func analyseRisksWithProvenance(start hoi4Date, countries []CountryHistory, provenance *[]CorpusProvenance) ([]CountryHistoryRisk, []Finding, []Finding) {
	risks := make([]CountryHistoryRisk, 0)
	parserFindings := make([]Finding, 0)
	hashFindings := make([]Finding, 0)
	for _, country := range sortedCountries(countries) {
		if country.Origin != BaseCountry {
			continue
		}
		index := provenanceIndex(*provenance, country.RelativePath)
		if index < 0 || !(*provenance)[index].HashVerified {
			hashFindings = append(hashFindings, finding(
				"scenario.base_version_mismatch", "attention", "原版世界资料已变化",
				"该国家历史文件与已验证的游戏资料不一致。", "请更新世界资料后重新检查。", country.Tag, country.RelativePath,
			))
			continue
		}
		blocks, err := ExtractTopLevelDatedBlocks(country.Content)
		if err != nil {
			parserFindings = append(parserFindings, finding(
				"scenario.history_parse_failed", "attention", "无法安全读取原版国家历史",
				"该国家历史文件结构不完整，钢之工坊不会猜测应如何修改。", "修复或更新世界资料后重新检查。", country.Tag, country.RelativePath,
			))
			continue
		}
		(*provenance)[index].ParserSucceeded = true
		executable := make([]DatedBlock, 0)
		for _, block := range blocks {
			date, _ := parseHOI4Date(block.Date)
			if dateLess(date, start) {
				executable = append(executable, block)
			}
		}
		if len(executable) == 0 {
			continue
		}
		risks = append(risks, CountryHistoryRisk{Tag: country.Tag, ExecutableDatedBlocks: append([]DatedBlock(nil), executable...)})
	}
	return risks, parserFindings, hashFindings
}

func makePreviewWithProvenance(startDate string, risks []CountryHistoryRisk, provenance []CorpusProvenance) Preview {
	preview := Preview{Risks: append([]CountryHistoryRisk(nil), risks...)}
	for _, risk := range risks {
		preview.AffectedCountries = append(preview.AffectedCountries, risk.Tag)
		preview.ExecutableDatedBlockCount += len(risk.ExecutableDatedBlocks)
	}
	preview.PlayerSummary = "剧本开局日期为 " + startDate + "；发现 " + strconv.Itoa(preview.ExecutableDatedBlockCount) + " 段基础世界历史会在开局前执行。钢之工坊只说明影响，不会自动修改。"
	var payload strings.Builder
	payload.WriteString("start=")
	payload.WriteString(startDate)
	for _, source := range provenance {
		payload.WriteString("\nprovider=")
		payload.WriteString(string(source.ProviderKind))
		payload.WriteString("\nversion=")
		payload.WriteString(source.GameVersion)
		payload.WriteString("\nparser=")
		payload.WriteString(source.ParserVersion)
		payload.WriteString("\npath=")
		payload.WriteString(source.RelativePath)
		payload.WriteString("\nsource=")
		payload.WriteString(strings.ToLower(source.ObservedSHA256))
	}
	for _, risk := range risks {
		payload.WriteString("\ntag=")
		payload.WriteString(risk.Tag)
		for _, block := range risk.ExecutableDatedBlocks {
			payload.WriteString("\nblock=")
			payload.WriteString(block.Date)
			payload.WriteByte(':')
			payload.WriteString(block.SHA256)
		}
	}
	preview.SHA256 = sha256Hex(payload.String())
	return preview
}

type tokenKind uint8

const (
	tokenWord tokenKind = iota
	tokenEqual
	tokenOpenBrace
	tokenCloseBrace
	tokenString
)

type token struct {
	kind       tokenKind
	value      string
	start, end int
}

func ExtractTopLevelDatedBlocks(content string) ([]DatedBlock, error) {
	tokens, err := scanTokens(content)
	if err != nil {
		return nil, err
	}
	depths := make([]int, len(tokens))
	matching := make(map[int]int)
	stack := make([]int, 0)
	depth := 0
	for i, tok := range tokens {
		depths[i] = depth
		switch tok.kind {
		case tokenOpenBrace:
			stack = append(stack, i)
			depth++
		case tokenCloseBrace:
			if depth == 0 || len(stack) == 0 {
				return nil, parseError("unexpected closing brace")
			}
			depth--
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			matching[open] = i
		}
	}
	if depth != 0 || len(stack) != 0 {
		return nil, parseError("unterminated brace")
	}

	blocks := make([]DatedBlock, 0)
	for i := 0; i+2 < len(tokens); i++ {
		if depths[i] != 0 || tokens[i].kind != tokenWord || tokens[i+1].kind != tokenEqual || tokens[i+2].kind != tokenOpenBrace {
			continue
		}
		date, ok := parseHOI4Date(tokens[i].value)
		if !ok {
			continue
		}
		closeIndex, ok := matching[i+2]
		if !ok {
			return nil, parseError("unmatched opening brace")
		}
		start, end := tokens[i].start, tokens[closeIndex].end
		blocks = append(blocks, DatedBlock{
			Date: normaliseDate(date), StartOffset: start, EndOffset: end,
			SHA256: sha256Hex(content[start:end]),
		})
	}
	return blocks, nil
}

type parseError string

func (e parseError) Error() string { return string(e) }

func scanTokens(content string) ([]token, error) {
	tokens := make([]token, 0)
	for i := 0; i < len(content); {
		switch content[i] {
		case ' ', '\t', '\r', '\n':
			i++
		case '#':
			for i < len(content) && content[i] != '\n' {
				i++
			}
		case '=':
			tokens = append(tokens, token{kind: tokenEqual, start: i, end: i + 1})
			i++
		case '{':
			tokens = append(tokens, token{kind: tokenOpenBrace, start: i, end: i + 1})
			i++
		case '}':
			tokens = append(tokens, token{kind: tokenCloseBrace, start: i, end: i + 1})
			i++
		case '"':
			start := i
			i++
			closed := false
			for i < len(content) {
				if content[i] == '\\' {
					i += 2
					continue
				}
				if content[i] == '"' {
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, parseError("unterminated quoted string")
			}
			tokens = append(tokens, token{kind: tokenString, start: start, end: i})
		default:
			start := i
			for i < len(content) && !isDelimiter(content[i]) {
				i++
			}
			if start == i {
				return nil, parseError("unrecognised token")
			}
			tokens = append(tokens, token{kind: tokenWord, value: content[start:i], start: start, end: i})
		}
	}
	return tokens, nil
}

func isDelimiter(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n', '#', '=', '{', '}', '"':
		return true
	default:
		return false
	}
}
