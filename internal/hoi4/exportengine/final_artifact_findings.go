package exportengine

import (
	"bytes"
	"encoding/json"
	"path"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

func RescanFinalArtifactFindings(entries []Entry, existing []ExportFinding, msg exportmsg.Printer) []ExportFinding {
	_, findings := RefreshFinalArtifactFindings(entries, existing, nil, msg)
	return findings
}

func RefreshFinalArtifactFindings(entries []Entry, existing []ExportFinding, snapshot *Snapshot, msg exportmsg.Printer) ([]Entry, []ExportFinding) {
	finalEntries := append([]Entry(nil), entries...)
	if snapshot != nil {
		kept := finalEntries[:0]
		for _, entry := range finalEntries {
			if isGeneratedReferenceIntegrityLog(entry) {
				continue
			}
			kept = append(kept, entry)
		}
		finalEntries = kept
	}

	out := make([]ExportFinding, 0, len(existing))
	for _, finding := range existing {
		if finalArtifactScannerSource(finding.Source, snapshot != nil) {
			continue
		}
		out = append(out, finding)
	}

	literal := ScanLiteralNewlines(finalEntries, msg)
	out = append(out, ExportFindingsFromLiteralNewlines(literal, false)...)
	syntax := ScanFinalArtifactSyntax(finalEntries, msg)
	out = append(out, ExportFindingsFromFinalSyntax(syntax, false)...)
	registry := ScanRegistryValidation(finalEntries, msg)
	out = append(out, ExportFindingsFromRegistryValidation(registry, false)...)
	if snapshot != nil {
		referenceReport := BuildReferenceIntegrityReport(*snapshot, finalEntries, msg)
		out = append(out, ExportFindingsFromReferenceIntegrity(referenceReport, false)...)
		for _, reportEntry := range referenceReport.Entries(referenceArtifactRoot(finalEntries)) {
			collision := false
			for _, entry := range finalEntries {
				if strings.EqualFold(strings.ReplaceAll(entry.Path, "\\", "/"), reportEntry.Path) {
					collision = true
					break
				}
			}
			if !collision {
				finalEntries = append(finalEntries, reportEntry)
			}
		}
	}
	return finalEntries, out
}

func isGeneratedReferenceIntegrityLog(entry Entry) bool {
	if !strings.EqualFold(path.Base(strings.ReplaceAll(entry.Path, "\\", "/")), ReferenceIntegrityLogBasename) {
		return false
	}
	if entry.SourceEntity == "reference_integrity" {
		return true
	}

	for _, locale := range exportmsg.Locales {
		if bytes.HasPrefix(entry.Bytes, []byte("# "+exportmsg.For(locale).Text("refs.log.title", nil))) {
			return true
		}
	}
	return false
}

func finalArtifactScannerSource(raw json.RawMessage, includeReferenceIntegrity bool) bool {
	var source struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(raw, &source) != nil {
		return false
	}
	switch source.Kind {
	case "literal_newline_lint", "final_artifact_syntax", "registry_validation":
		return true
	case "reference_integrity":
		return includeReferenceIntegrity
	default:
		return false
	}
}
