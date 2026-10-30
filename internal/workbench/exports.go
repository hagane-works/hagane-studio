package workbench

func SplitLocName(displayName, tag string) (en, zh string) { return splitLocName(displayName, tag) }

func FirstNonBlank(values ...string) string { return firstNonBlank(values...) }

func StrDefault(p *string, def string) string { return strDefault(p, def) }

func StringList(v any) []string { return stringList(v) }

func NowTimestamp() string { return nowTimestamp() }

var SyncProjectCountryTags = syncProjectCountryTags

func NormalizeCountryLocs(f *CountryLocsFields) { normalizeCountryLocs(f) }

func NormalizeRegistry(raw map[string]any) map[string]any { return normalizeRegistry(raw) }

func ValidateRegistry(registry map[string]any, deps []string) error {
	return validateRegistry(registry, deps)
}

func ValidateRoot(id, body string) string { return validateRoot(id, body) }

var (
	ContentLanguages = languages

	CountryTagRE = countryTagRE

	RootIDRE = rootIDRE
)

const CountryTagPatternMessage = countryTagPatternMessage
