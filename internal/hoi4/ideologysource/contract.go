package ideologysource

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
)

const (
	SchemaVersion      = 1
	IdeologyDirectory  = "common/ideologies"
	SourceVanilla      = "vanilla"
	SourceDependency   = "dependency"
	CodeLayerAmbiguous = "ideology_source_layer_ambiguous"
)

var scriptIDRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Layer struct {
	SourceKind    string       `json:"source_kind"`
	DependencyKey string       `json:"dependency_key,omitempty"`
	LoadOrder     int          `json:"load_order"`
	ReplacePaths  []string     `json:"replace_paths,omitempty"`
	Files         []SourceFile `json:"files"`
}

type SourceFile struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

type RootOwner struct {
	RootID        string   `json:"root_id"`
	Body          string   `json:"body,omitempty"`
	SourceKind    string   `json:"source_kind"`
	DependencyKey string   `json:"dependency_key,omitempty"`
	OwnerPath     string   `json:"owner_path"`
	FileSHA256    string   `json:"file_sha256"`
	RootSHA256    string   `json:"root_sha256"`
	SubtypeIDs    []string `json:"subtype_ids"`
	TypesCount    int      `json:"types_count"`
}

type Discovery struct {
	Roots      []RootOwner `json:"roots"`
	Suppressed []RootOwner `json:"suppressed,omitempty"`
	locations  map[string]rootLocation
	files      map[string]discoveredFile
}

type Manifest struct {
	SchemaVersion int             `json:"schema_version"`
	Entries       []ManifestEntry `json:"entries"`
}

type ManifestEntry struct {
	SourceKind    string `json:"source_kind"`
	DependencyKey string `json:"dependency_key,omitempty"`
	OwnerPath     string `json:"owner_path"`
	FileSHA256    string `json:"file_sha256"`
	RootSHA256    string `json:"root_sha256"`
	ScriptID      string `json:"script_id"`
	ParentRootID  string `json:"parent_root_id"`
}

type Subtype struct {
	ScriptID     string            `json:"script_id"`
	ParentRootID string            `json:"parent_root_id"`
	Names        map[string]string `json:"names,omitempty"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
}

type PatchedFile struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

type LocalisationRow struct {
	Language string `json:"language"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type Compilation struct {
	Files         []PatchedFile     `json:"files"`
	Localisations []LocalisationRow `json:"localisations"`
}

type Finding struct {
	Code         string `json:"code"`
	ScriptID     string `json:"script_id,omitempty"`
	ParentRootID string `json:"parent_root_id,omitempty"`
	OwnerPath    string `json:"owner_path,omitempty"`
	Detail       string `json:"detail"`
}

type FindingsError struct{ Findings []Finding }

func (e FindingsError) Error() string {
	if len(e.Findings) == 0 {
		return "ideology source validation failed"
	}
	return e.Findings[0].Code + ": " + e.Findings[0].Detail
}

func findingsError(findings ...Finding) error {
	sortFindings(findings)
	return FindingsError{Findings: findings}
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.ParentRootID != b.ParentRootID {
			return a.ParentRootID < b.ParentRootID
		}
		if a.ScriptID != b.ScriptID {
			return a.ScriptID < b.ScriptID
		}
		if a.OwnerPath != b.OwnerPath {
			return a.OwnerPath < b.OwnerPath
		}
		return a.Detail < b.Detail
	})
}

func Hash(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func normalizePath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	return strings.Trim(value, "/")
}

func normalizedDirectory(value string) string { return strings.ToLower(normalizePath(value)) }

func isIdeologyFile(path string) bool {
	p := strings.ToLower(normalizePath(path))
	return strings.HasPrefix(p, IdeologyDirectory+"/") && strings.HasSuffix(p, ".txt")
}

func validID(value string) bool { return scriptIDRE.MatchString(strings.TrimSpace(value)) }

func finding(code, detail string) Finding { return Finding{Code: code, Detail: detail} }

func withEntry(f Finding, entry ManifestEntry) Finding {
	f.ScriptID, f.ParentRootID, f.OwnerPath = entry.ScriptID, entry.ParentRootID, entry.OwnerPath
	return f
}

func keyForOwner(rootID, path string) string {
	return strings.TrimSpace(rootID) + "\x00" + strings.ToLower(normalizePath(path))
}

func ownerKey(entry ManifestEntry) string {
	return strings.ToLower(normalizePath(entry.OwnerPath)) + "\x00" + entry.ParentRootID
}
