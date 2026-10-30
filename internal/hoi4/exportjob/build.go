package exportjob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportmsg"
)

const MetadataSchema = "hoi4.export-job-result.v1"

type IssueCollector interface {
	CollectExportIssues(context.Context, int64, ExportIssuesOptions) ([]ExportIssue, bool, error)
}

type BuildInput struct {
	Snapshot            exportengine.Snapshot
	IncludeImported     bool
	ReplacePaths        []string
	AcknowledgeBlockers bool
	BuildIdentity       bool
	ArtifactPath        string
	ResolveBlob         exportengine.BlobResolver
	Issues              IssueCollector

	Locale string
}

type Result struct {
	Schema              string                `json:"schema"`
	ProjectID           int64                 `json:"project_id"`
	Filename            string                `json:"filename"`
	SizeBytes           int64                 `json:"size_bytes"`
	SHA256              string                `json:"sha256"`
	Files               []string              `json:"files"`
	SnapshotFingerprint string                `json:"snapshot_fingerprint"`
	SnapshotTables      map[string]int        `json:"snapshot_tables"`
	NativeWarnings      []string              `json:"native_warnings"`
	ExportIssues        []ExportIssue         `json:"export_issues"`
	NativeManifest      exportengine.Manifest `json:"native_manifest"`
}

func Build(ctx context.Context, input BuildInput) (Result, error) {
	if input.Snapshot.ProjectID <= 0 || input.Snapshot.Schema != exportengine.SnapshotSchema {
		return Result{}, errors.New("exportjob: valid snapshot is required")
	}
	if strings.TrimSpace(input.ArtifactPath) == "" || !filepath.IsAbs(input.ArtifactPath) {
		return Result{}, errors.New("exportjob: absolute artifact path is required")
	}
	plan, err := exportengine.BuildNativeTimelinePlan(
		input.Snapshot, input.ReplacePaths, input.IncludeImported, input.ResolveBlob,
	)
	if err != nil {
		return Result{}, fmt.Errorf("exportjob: build timeline plan: %w", err)
	}
	msg := exportmsg.For(input.Locale)
	issues, issueNotes := collectIssues(ctx, input, msg)
	artifact, err := exportengine.BuildNativeShadowWithTimelinePlan(
		input.Snapshot, plan, input.IncludeImported, input.ResolveBlob, msg,
	)
	if err != nil {
		return Result{}, fmt.Errorf("exportjob: build native artifact: %w", err)
	}
	artifact.Warnings = append(issueNotes, artifact.Warnings...)
	if input.BuildIdentity {
		buildID, identityErr := exportengine.NewBuildIdentity()
		if identityErr == nil {
			identityErr = exportengine.ApplyBuildIdentity(&artifact, buildID)
		}
		if identityErr != nil {
			return Result{}, fmt.Errorf("exportjob: apply build identity: %w", identityErr)
		}
	}
	if len(issues) > 0 {
		entries, mergeErr := exportengine.MergeExportFindingsEntry(
			artifact.Entries, exportFindings(issues, input.AcknowledgeBlockers),
		)
		if mergeErr != nil {
			artifact.Warnings = append(artifact.Warnings,
				msg.Text("job.findings_not_merged", exportmsg.Args{"reason": mergeErr.Error()}))
		} else {
			artifact.Entries = entries
		}
	}
	manifest, err := exportengine.BuildContextWithMetadata(
		ctx, input.ArtifactPath,
		artifactFingerprint(input.Snapshot.Fingerprint, plan.PlanSHA256),
		artifact.Entries, exportengine.DefaultArtifactBudget(), artifact.Timeline,
	)
	if err != nil {
		return Result{}, fmt.Errorf("exportjob: write artifact: %w", err)
	}
	filename := strings.TrimSpace(artifact.Filename)
	if filename == "" {
		filename = fmt.Sprintf("project-%d.zip", input.Snapshot.ProjectID)
	}
	files := make([]string, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		files = append(files, entry.Path)
	}
	sort.Strings(files)
	counts := make(map[string]int, len(input.Snapshot.Tables))
	for table, rows := range input.Snapshot.Tables {
		counts[table] = len(rows)
	}
	warnings := append([]string{}, artifact.Warnings...)
	if issues == nil {
		issues = []ExportIssue{}
	}
	return Result{
		Schema: MetadataSchema, ProjectID: input.Snapshot.ProjectID,
		Filename: filename, SizeBytes: manifest.ZIPBytes, SHA256: manifest.ZIPSHA256,
		Files: files, SnapshotFingerprint: input.Snapshot.Fingerprint,
		SnapshotTables: counts, NativeWarnings: warnings, ExportIssues: issues,
		NativeManifest: manifest,
	}, nil
}

func collectIssues(ctx context.Context, input BuildInput, msg exportmsg.Printer) ([]ExportIssue, []string) {
	if input.Issues == nil {
		return nil, []string{msg.Text("job.checks_not_run", nil)}
	}
	issues, found, err := input.Issues.CollectExportIssues(ctx, input.Snapshot.ProjectID, ExportIssuesOptions{
		IncludeImported: input.IncludeImported, ReplacePaths: input.ReplacePaths,
	})
	if err != nil {
		return nil, []string{msg.Text("job.checks_failed_reason", exportmsg.Args{"reason": err.Error()})}
	}
	if !found {
		return nil, []string{msg.Text("job.checks_failed", nil)}
	}
	return issues, nil
}

func artifactFingerprint(snapshotFingerprint, timelinePlanSHA256 string) string {
	raw, _ := json.Marshal(struct {
		SnapshotFingerprint string `json:"snapshot_fingerprint"`
		TimelinePlanSHA256  string `json:"timeline_plan_sha256"`
	}{snapshotFingerprint, timelinePlanSHA256})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func exportFindings(issues []ExportIssue, acknowledge bool) []exportengine.ExportFinding {
	findings := make([]exportengine.ExportFinding, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(fmt.Sprint(issue["code"]))
		if code == "" || code == "<nil>" {
			continue
		}
		message := strings.TrimSpace(fmt.Sprint(issue["message"]))
		if message == "" || message == "<nil>" {
			message = code
		}
		sourceValue := issue["source"]
		if sourceValue == nil {
			sourceValue = map[string]any{"producer": "export_job"}
		}
		source, err := json.Marshal(sourceValue)
		if err != nil {
			source, _ = json.Marshal(map[string]any{"producer": "export_job", "source_encoding": "unavailable"})
		}
		var pathValue *string
		if value, ok := issue["path"].(string); ok && strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			pathValue = &value
		}
		lineValue := issueLine(issue["line"])
		playerImpact, _ := issue["player_impact"].(string)
		editRoute, _ := issue["repair_route"].(string)
		if strings.TrimSpace(editRoute) == "" {
			editRoute, _ = issue["fix_hint"].(string)
		}
		severity := strings.TrimSpace(fmt.Sprint(issue["severity"]))
		findings = append(findings, exportengine.ExportFinding{
			Code: code, Severity: severity, Source: source, Path: pathValue, Line: lineValue,
			Message: message, PlayerImpact: playerImpact, EditRoute: editRoute,
			Acknowledged:   acknowledge && severity == "blocker" && !strings.Contains(strings.ToLower(code), "raw_binary_bytes_unavailable"),
			SourceEvidence: map[string]any{"producer_issue": issue},
		})
	}
	return findings
}

func issueLine(value any) *int {
	var line int
	switch item := value.(type) {
	case int:
		line = item
	case float64:
		if float64(int(item)) == item {
			line = int(item)
		}
	case json.Number:
		line, _ = strconv.Atoi(item.String())
	}
	if line < 1 {
		return nil
	}
	return &line
}

type ExportIssue = map[string]any

type ExportIssuesOptions struct {
	IncludeImported bool
	ReplacePaths    []string
}
