package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	ExportFindingsBasename = "_HAGANE_EXPORT_FINDINGS.json"
	ExportFindingsSchema   = "hagane.export-findings.v1"
	maxExportFindingsBytes = 8 << 20
)

type ExportFinding struct {
	Code           string          `json:"code"`
	Severity       string          `json:"severity,omitempty"`
	Source         json.RawMessage `json:"source"`
	Path           *string         `json:"path"`
	Line           *int            `json:"line"`
	Message        string          `json:"message"`
	PlayerImpact   string          `json:"player_impact"`
	EditRoute      string          `json:"edit_route"`
	Acknowledged   bool            `json:"acknowledged"`
	SourceEvidence map[string]any  `json:"source_evidence,omitempty"`
}

type ExportFindingsDocument struct {
	Schema   string          `json:"schema"`
	Findings []ExportFinding `json:"findings"`
}

func NewExportFindingsDocument(findings []ExportFinding) (ExportFindingsDocument, error) {
	doc := ExportFindingsDocument{
		Schema:   ExportFindingsSchema,
		Findings: append([]ExportFinding(nil), findings...),
	}
	if doc.Findings == nil {
		doc.Findings = []ExportFinding{}
	}
	for i := range doc.Findings {
		if err := validateExportFinding(doc.Findings[i]); err != nil {
			return ExportFindingsDocument{}, fmt.Errorf("finding %d: %w", i, err)
		}
	}
	sortExportFindings(doc.Findings)
	return doc, nil
}

func MarkExportFindingsAcknowledged(findings []ExportFinding, acknowledged bool) []ExportFinding {
	out := append([]ExportFinding(nil), findings...)
	for i := range out {
		out[i].Acknowledged = acknowledged
	}
	return out
}

func ExportFindingsEntry(entries []Entry, findings []ExportFinding) (Entry, error) {
	for _, entry := range entries {
		if isExportFindingsPath(entry.Path) {
			return Entry{}, fmt.Errorf("export findings path collides with an emitted entry: %q", entry.Path)
		}
	}
	doc, err := NewExportFindingsDocument(findings)
	if err != nil {
		return Entry{}, err
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return Entry{}, err
	}
	body = append(body, '\n')

	manifestPath := artifactManifestPath(entries)
	root := ""
	if slash := strings.LastIndexByte(manifestPath, '/'); slash >= 0 {
		root = manifestPath[:slash+1]
	}
	return Entry{
		Path:         root + ExportFindingsBasename,
		Bytes:        body,
		Origin:       "generated",
		SourceEntity: "export_findings",
	}, nil
}

func MergeExportFindingsEntry(entries []Entry, additions []ExportFinding) ([]Entry, error) {
	kept := make([]Entry, 0, len(entries)+1)
	combined := make([]ExportFinding, 0, len(additions))
	found := false
	for _, entry := range entries {
		if !isExportFindingsPath(entry.Path) {
			kept = append(kept, entry)
			continue
		}
		if found {
			return nil, errors.New("entries contain more than one export findings record")
		}
		found = true
		doc, err := DecodeExportFindingsDocument(entry.Bytes)
		if err != nil {
			return nil, fmt.Errorf("decode existing export findings: %w", err)
		}
		combined = append(combined, doc.Findings...)
	}
	combined = append(combined, additions...)
	if len(combined) == 0 {
		return kept, nil
	}
	report, err := ExportFindingsEntry(kept, combined)
	if err != nil {
		return nil, err
	}
	return append(kept, report), nil
}

func isExportFindingsPath(value string) bool {
	value = strings.TrimRight(strings.ReplaceAll(value, "\\", "/"), "/")
	if slash := strings.LastIndexByte(value, '/'); slash >= 0 {
		value = value[slash+1:]
	}
	return strings.EqualFold(value, ExportFindingsBasename)
}

func ReadExportFindingsFromZIP(filename string) (ExportFindingsDocument, string, error) {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return ExportFindingsDocument{}, "", err
	}
	defer reader.Close()
	return readExportFindingsArchive(&reader.Reader)
}

func ReadExportFindingsBytes(body []byte) (ExportFindingsDocument, string, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ExportFindingsDocument{}, "", err
	}
	return readExportFindingsArchive(reader)
}

func DecodeExportFindingsDocument(body []byte) (ExportFindingsDocument, error) {
	if len(body) > maxExportFindingsBytes {
		return ExportFindingsDocument{}, errors.New("export findings record exceeds readback budget")
	}
	var doc ExportFindingsDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return ExportFindingsDocument{}, fmt.Errorf("decode export findings: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return ExportFindingsDocument{}, err
	}
	if doc.Schema != ExportFindingsSchema {
		return ExportFindingsDocument{}, fmt.Errorf("unsupported export findings schema %q", doc.Schema)
	}
	if doc.Findings == nil {
		doc.Findings = []ExportFinding{}
	}
	for i := range doc.Findings {
		if err := validateExportFinding(doc.Findings[i]); err != nil {
			return ExportFindingsDocument{}, fmt.Errorf("finding %d: %w", i, err)
		}
	}
	return doc, nil
}

func readExportFindingsArchive(reader *zip.Reader) (ExportFindingsDocument, string, error) {
	var match *zip.File
	for _, file := range reader.File {
		if !isExportFindingsPath(file.Name) {
			continue
		}
		if match != nil {
			return ExportFindingsDocument{}, "", errors.New("artifact contains more than one export findings record")
		}
		match = file
	}
	if match == nil {
		return ExportFindingsDocument{}, "", os.ErrNotExist
	}
	if match.UncompressedSize64 > maxExportFindingsBytes {
		return ExportFindingsDocument{}, match.Name, errors.New("export findings record exceeds readback budget")
	}
	rc, err := match.Open()
	if err != nil {
		return ExportFindingsDocument{}, match.Name, err
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, maxExportFindingsBytes+1))
	if err != nil {
		return ExportFindingsDocument{}, match.Name, err
	}
	if len(body) > maxExportFindingsBytes {
		return ExportFindingsDocument{}, match.Name, errors.New("export findings record exceeds readback budget")
	}
	doc, err := DecodeExportFindingsDocument(body)
	if err != nil {
		return ExportFindingsDocument{}, match.Name, err
	}
	return doc, match.Name, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("export findings record contains trailing JSON")
		}
		return fmt.Errorf("decode trailing export findings data: %w", err)
	}
	return nil
}

func validateExportFinding(finding ExportFinding) error {
	if strings.TrimSpace(finding.Code) == "" {
		return errors.New("code is required")
	}
	if len(finding.Source) == 0 || !json.Valid(finding.Source) || bytes.Equal(bytes.TrimSpace(finding.Source), []byte("null")) {
		return errors.New("source must be traceable JSON")
	}
	if finding.Path != nil && strings.TrimSpace(*finding.Path) == "" {
		return errors.New("path must be null or non-empty")
	}
	if finding.Line != nil && *finding.Line < 1 {
		return errors.New("line must be null or positive")
	}
	if strings.TrimSpace(finding.Message) == "" {
		return errors.New("message is required")
	}

	return nil
}

func sortExportFindings(findings []ExportFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if string(a.Source) != string(b.Source) {
			return string(a.Source) < string(b.Source)
		}
		if pointerString(a.Path) != pointerString(b.Path) {
			return pointerString(a.Path) < pointerString(b.Path)
		}
		return pointerInt(a.Line) < pointerInt(b.Line)
	})
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pointerInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
