package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

type worldLint interface {
	Count() int

	Headline() string

	LogEntry(modDir string) Entry

	CoverageTag() string
}

func ApplyWorldLints(artifact *NativeArtifact, meta worldout.Meta, msg exportmsg.Printer) {
	if artifact == nil {
		return
	}
	modDir := artifactRoot(*artifact)
	for _, lint := range worldLintsFor(meta, msg) {
		if lint.Count() == 0 {
			continue
		}
		if headline := lint.Headline(); headline != "" {
			artifact.Warnings = append(artifact.Warnings, headline)
		}
		if logEntry := lint.LogEntry(modDir); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
			artifact.Coverage = append(artifact.Coverage, lint.CoverageTag())
		}
	}
}

func worldLintsFor(meta worldout.Meta, msg exportmsg.Printer) []worldLint {
	return []worldLint{
		ScanPoliticsWarnings(meta, msg),
		ScanYMLSymmetryWarnings(meta, msg),
		ScanStateOrphanTags(meta, msg),
	}
}

func worldLintLogText(header []string, lines []string) string {
	all := make([]string, 0, len(header)+1+len(lines))
	all = append(all, header...)
	all = append(all, "")
	all = append(all, lines...)
	return strings.Join(all, "\n") + "\n"
}

func worldLintEntry(modDir, name, text, sourceEntity string) Entry {
	if modDir == "" || text == "" {
		return Entry{}
	}
	return Entry{
		Path:         modDir + "/" + name,
		Bytes:        []byte(text),
		Origin:       "generated",
		SourceEntity: sourceEntity,
	}
}

func worldLintSubjects(lines []exportmsg.Message) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range lines {
		subject, _ := line.Args["subject"].(string)
		if subject == "" || seen[subject] {
			continue
		}
		seen[subject] = true
		out = append(out, subject)
	}
	return out
}

func worldLintRender(lines []exportmsg.Message, msg exportmsg.Printer) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, msg.Render(line))
	}
	return out
}
