package api

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"haganeworks/internal/export"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportjob"
	"haganeworks/internal/hoi4/exportmerge"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/exportplan/timeline"
	"haganeworks/internal/project"
	"haganeworks/internal/server"
	"haganeworks/internal/workbench"
)

const (
	advancedFieldBytes = int64(2 << 20)
)

var advancedUploadLimit = int64(512 << 20)

type advancedExports struct {
	projects *projects
	issues   func(*project.Project) exportjob.IssueCollector
}

func registerAdvancedExports(mux *http.ServeMux, p *projects, issues func(*project.Project) exportjob.IssueCollector) {
	a := &advancedExports{projects: p, issues: issues}
	mux.HandleFunc("GET /api/go/projects/{id}/full-mod-files", a.fullModFiles)
	mux.HandleFunc("POST /api/go/projects/{id}/export/diff-preview", a.diffPreview)
	mux.HandleFunc("POST /api/go/projects/{id}/export/diff-resolve", a.diffResolve)
	mux.HandleFunc("POST /api/go/projects/{id}/export/diff-export-zip", a.diffExportZIP)
	mux.HandleFunc("POST /api/go/projects/{id}/export/append-to-existing", a.appendToExisting)
}

func (a *advancedExports) fullModFiles(w http.ResponseWriter, r *http.Request) {
	p, err := a.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	includeImported := true
	if raw := strings.TrimSpace(r.URL.Query().Get("include_imported")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			server.Fail(w, http.StatusUnprocessableEntity, errors.New("include_imported must be true or false"))
			return
		}
		includeImported = parsed
	}
	if err := export.AwaitGameIndex(r.Context()); err != nil {
		server.Fail(w, http.StatusServiceUnavailable, err)
		return
	}
	snapshot, err := export.Snapshot(p)
	if err != nil {
		fail(w, err)
		return
	}
	entries, err := fullModEntries(r.Context(), p, snapshot, includeImported, exportmsg.For(r.URL.Query().Get("locale")))
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, fmt.Errorf("export failed: %w", err))
		return
	}
	var record struct {
		Name       string `json:"name"`
		CountryTag string `json:"country_tag"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	files := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Path, "_CROSS_REF_SKIPPED.log") {
			continue
		}
		item := map[string]any{"path": entry.Path, "size": len(entry.Bytes), "is_binary": isAdvancedBinary(entry.Path)}
		if item["is_binary"].(bool) {
			item["content_b64"] = base64.StdEncoding.EncodeToString(entry.Bytes)
		} else {
			body := entry.Bytes
			if len(body) >= 3 && string(body[:3]) == "\xef\xbb\xbf" {
				body = body[3:]
			}
			if utf8.Valid(body) {
				item["content"] = string(body)
			} else {
				item["is_binary"] = true
				item["content_b64"] = base64.StdEncoding.EncodeToString(entry.Bytes)
			}
		}
		files = append(files, item)
	}
	server.JSON(w, http.StatusOK, map[string]any{
		"mod_name": record.Name, "primary_tag": strings.ToUpper(record.CountryTag),
		"total_files": len(files), "include_imported": includeImported, "files": files,
	})
}

func fullModEntries(ctx context.Context, p *project.Project, snapshot exportengine.Snapshot, includeImported bool, msg exportmsg.Printer) ([]exportengine.Entry, error) {
	artifact, err := safeBuildNative(p, snapshot, nil, includeImported, msg)
	if err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp("", "hoi4-full-mod-files-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	destination := filepath.Join(tempDir, "artifact.zip")
	budget := exportengine.DefaultArtifactBudget()
	if _, err := exportengine.BuildContextWithMetadata(ctx, destination, snapshot.Fingerprint, artifact.Entries, budget, artifact.Timeline); err != nil {
		return nil, err
	}
	archiveBudget := exportmerge.DefaultArchiveBudget()
	archiveBudget.MaxCompressedBytes = budget.MaxTotalBytes
	archiveBudget.MaxEntries = budget.MaxEntries + 1
	archiveBudget.MaxEntryBytes = budget.MaxTotalBytes
	archiveBudget.MaxUncompressedBytes = budget.MaxTotalBytes
	archive, err := exportmerge.ReadArchive(ctx, destination, archiveBudget)
	if err != nil {
		return nil, err
	}
	return exportmerge.ArchiveEntries(archive, false), nil
}

func safeBuildNative(p *project.Project, snapshot exportengine.Snapshot, replacePaths []string, includeImported bool, msg exportmsg.Printer) (artifact exportengine.NativeArtifact, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("native exporter rejected project data: %v", value)
		}
	}()
	resolve := export.BlobResolver(p)
	plan, err := exportengine.BuildNativeTimelinePlan(
		snapshot, replacePaths, includeImported, resolve,
	)
	if err != nil {
		return exportengine.NativeArtifact{}, err
	}
	return exportengine.BuildNativeShadowWithTimelinePlan(snapshot, plan, includeImported, resolve, msg)
}

func (a *advancedExports) diffPreview(w http.ResponseWriter, r *http.Request) {
	p, err := a.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	upload, cleanup, err := receiveAdvancedUpload(w, r)
	if err != nil {
		writeAdvancedUploadError(w, err)
		return
	}
	defer cleanup()
	snapshot, _, target, ok := prepareAdvancedDiff(w, r, p, upload)
	if !ok {
		return
	}
	preview, err := exportmerge.BuildPreview(snapshot, target)
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, fmt.Errorf("diff preview failed: %w", err))
		return
	}
	server.JSON(w, http.StatusOK, preview)
}

func (a *advancedExports) diffResolve(w http.ResponseWriter, r *http.Request) {
	p, err := a.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	upload, cleanup, err := receiveAdvancedUpload(w, r)
	if err != nil {
		writeAdvancedUploadError(w, err)
		return
	}
	defer cleanup()
	payload, err := decodeResolutionPayload(upload.Fields["resolutions_json"])
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	snapshot, _, target, ok := prepareAdvancedDiff(w, r, p, upload)
	if !ok {
		return
	}
	final, warnings, err := exportmerge.ComputeFinalOps(snapshot, target, payload)
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"final_ops_by_entity": final, "warnings": warnings})
}

func (a *advancedExports) diffExportZIP(w http.ResponseWriter, r *http.Request) {
	p, err := a.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	upload, cleanup, err := receiveAdvancedUpload(w, r)
	if err != nil {
		writeAdvancedUploadError(w, err)
		return
	}
	defer cleanup()
	payload, err := decodeResolutionPayload(upload.Fields["resolutions_json"])
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	acknowledge, err := parseOptionalBool(upload.Fields["acknowledge_blockers"])
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := export.AwaitGameIndex(r.Context()); err != nil {
		server.Fail(w, http.StatusServiceUnavailable, err)
		return
	}
	gateOK, gateIssues := advancedExportGateWithIssues(w, r, a.issues(p), p.ID(), exportjob.ExportIssuesOptions{IncludeImported: true}, acknowledge)
	if !gateOK {
		return
	}
	gateFindings := advancedExportFindings(gateIssues, acknowledge)
	snapshot, archive, target, ok := prepareAdvancedDiff(w, r, p, upload)
	if !ok {
		return
	}
	final, warnings, err := exportmerge.ComputeFinalOps(snapshot, target, payload)
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	var record struct {
		CountryTag string `json:"country_tag"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	destination := filepath.Join(upload.TempDir, "merged.zip")

	manifest, err := safeApplyAdvancedOps(r.Context(), archive, target, final, record.CountryTag, destination, "", gateFindings, snapshot, advancedPrinter(upload))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		server.Fail(w, http.StatusUnprocessableEntity, fmt.Errorf("zip-write failed: %w", err))
		return
	}
	filename := "merged_" + sanitiseAdvancedZIPName(upload.Filename)
	if err := setAdvancedArtifactHeaders(w, destination, manifest); err != nil {
		server.Fail(w, http.StatusInternalServerError, errors.New("artifact metadata unavailable"))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("X-Merge-Entity-Count", strconv.Itoa(len(final)))
	if len(warnings) > 0 {
		w.Header().Set("X-Merge-Parse-Warnings", strconv.Itoa(len(warnings)))
	}
	serveTemporaryArtifact(w, r, destination, filename)
}

func (a *advancedExports) appendToExisting(w http.ResponseWriter, r *http.Request) {
	p, err := a.projects.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	upload, cleanup, err := receiveAdvancedUpload(w, r)
	if err != nil {
		writeAdvancedUploadError(w, err)
		return
	}
	defer cleanup()
	includeImported, err := parseOptionalBool(upload.Fields["include_imported"])
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}

	if _, err := parseOptionalBool(upload.Fields["build_identity"]); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	acknowledge, err := parseOptionalBool(upload.Fields["acknowledge_blockers"])
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	replacePaths := parseAdvancedReplacePaths(upload.Fields["replace_paths"])
	if err := export.AwaitGameIndex(r.Context()); err != nil {
		server.Fail(w, http.StatusServiceUnavailable, err)
		return
	}
	gateOptions := exportjob.ExportIssuesOptions{IncludeImported: includeImported, ReplacePaths: replacePaths}
	gateOK, gateIssues := advancedExportGateWithIssues(w, r, a.issues(p), p.ID(), gateOptions, acknowledge)
	if !gateOK {
		return
	}
	gateFindings := advancedExportFindings(gateIssues, acknowledge)
	snapshot, err := export.Snapshot(p)
	if err != nil {
		fail(w, err)
		return
	}
	base, err := readAdvancedBase(r.Context(), upload, true)
	if err != nil {
		writeAdvancedArchiveError(w, err)
		return
	}
	msg := advancedPrinter(upload)
	platform, err := safeBuildNative(p, snapshot, replacePaths, includeImported, msg)
	if err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, fmt.Errorf("export failed: %w", err))
		return
	}
	destination := filepath.Join(upload.TempDir, "appended.zip")
	manifest, conflicts, err := safeMergeAdvancedAppend(r.Context(), base, platform.Entries, destination, gateFindings, snapshot, platform.Timeline, msg)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		server.Fail(w, http.StatusUnprocessableEntity, fmt.Errorf("append zip-write failed: %w", err))
		return
	}
	var record struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(snapshot.Project, &record)
	filename := strings.TrimSpace(record.Name)
	if filename == "" {
		filename = fmt.Sprintf("project-%d", p.ID())
	}
	filename = sanitiseAdvancedUnicodeZIPName(filename + "_appended.zip")
	baseFilename := upload.Fields["base_mod_filename"]
	if strings.TrimSpace(baseFilename) == "" {
		baseFilename = upload.Filename
	}
	if err := setAdvancedArtifactHeaders(w, destination, manifest); err != nil {
		server.Fail(w, http.StatusInternalServerError, errors.New("artifact metadata unavailable"))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("X-Merge-Conflict-Count", strconv.Itoa(len(conflicts)))
	w.Header().Set("X-Base-Mod-Filename", strings.ReplaceAll(url.QueryEscape(baseFilename), "+", "%20"))
	serveTemporaryArtifact(w, r, destination, filename)
}

func safeMergeAdvancedAppend(ctx context.Context, base exportmerge.Archive, platform []exportengine.Entry, destination string, findings []exportengine.ExportFinding, snapshot exportengine.Snapshot, metadata *timeline.ArtifactMetadata, msg exportmsg.Printer) (manifest exportengine.Manifest, conflicts []string, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("Go append emitter rejected project data: %v", value)
		}
	}()
	return exportmerge.MergeAppendWithFindingsAndSnapshot(ctx, base, platform, destination, "", findings, snapshot, metadata, msg)
}

func parseAdvancedReplacePaths(raw string) []string {
	seen := map[string]bool{}
	paths := []string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.Trim(strings.ReplaceAll(strings.TrimSpace(item), "\\", "/"), "/")
		if item != "" && !seen[item] {
			seen[item] = true
			paths = append(paths, item)
		}
	}
	return paths
}

func prepareAdvancedDiff(w http.ResponseWriter, r *http.Request, p *project.Project, upload advancedUpload) (exportengine.Snapshot, exportmerge.Archive, *exportmerge.TargetMod, bool) {
	snapshot, err := export.Snapshot(p)
	if err != nil {
		fail(w, err)
		return exportengine.Snapshot{}, exportmerge.Archive{}, nil, false
	}
	archive, err := readAdvancedBase(r.Context(), upload, false)
	if err != nil {
		writeAdvancedArchiveError(w, err)
		return exportengine.Snapshot{}, exportmerge.Archive{}, nil, false
	}
	target := exportmerge.ParseTarget(r.Context(), archive)
	if err := r.Context().Err(); err != nil {
		return exportengine.Snapshot{}, exportmerge.Archive{}, nil, false
	}
	return snapshot, archive, target, true
}

func readAdvancedBase(ctx context.Context, upload advancedUpload, underFolder bool) (exportmerge.Archive, error) {
	if upload.Folder == "" {
		return exportmerge.ReadArchive(ctx, upload.FilenameOnDisk, exportmerge.DefaultArchiveBudget())
	}
	prefix := ""
	if underFolder {
		prefix = filepath.Base(upload.Folder)
	}
	return exportmerge.ReadFolder(ctx, upload.Folder, prefix, exportmerge.DefaultArchiveBudget())
}

func advancedPrinter(upload advancedUpload) exportmsg.Printer {
	return exportmsg.For(upload.Fields["locale"])
}

func safeApplyAdvancedOps(ctx context.Context, archive exportmerge.Archive, target *exportmerge.TargetMod, final map[string][]exportmerge.DiffOp, tag, destination, fingerprint string, findings []exportengine.ExportFinding, snapshot exportengine.Snapshot, msg exportmsg.Printer) (manifest exportengine.Manifest, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("Go emitter rejected merged entity data: %v", value)
		}
	}()
	return exportmerge.ApplyOpsToArchiveWithFindingsAndSnapshot(ctx, archive, target, final, tag, destination, fingerprint, findings, snapshot, msg)
}

func decodeResolutionPayload(raw string) (exportmerge.ResolutionRequest, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	var payload exportmerge.ResolutionRequest
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return payload, fmt.Errorf("resolutions_json invalid: %v", err)
	}
	if payload.Resolutions == nil {
		payload.Resolutions = map[string]string{}
	}
	if payload.ManualValues == nil {
		payload.ManualValues = map[string]any{}
	}
	if err := exportmerge.ValidateResolutionRequest(payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func advancedExportGateWithIssues(w http.ResponseWriter, r *http.Request, collector exportjob.IssueCollector, projectID int64, options exportjob.ExportIssuesOptions, acknowledge bool) (bool, []exportjob.ExportIssue) {
	issues, found, err := collector.CollectExportIssues(r.Context(), projectID, options)
	if err != nil {
		server.Fail(w, http.StatusServiceUnavailable, errors.New("export gate unavailable"))
		return false, nil
	}
	if !found {
		server.Fail(w, http.StatusNotFound, errors.New("Project not found"))
		return false, nil
	}
	blockers := make([]exportjob.ExportIssue, 0)
	for _, issue := range issues {
		if fmt.Sprint(issue["severity"]) != "blocker" {
			continue
		}
		if acknowledge && !nonOverridableAdvancedBlocker(fmt.Sprint(issue["code"])) {
			continue
		}
		blockers = append(blockers, issue)
	}
	if len(blockers) == 0 {
		return true, issues
	}
	server.JSON(w, http.StatusConflict, map[string]any{"detail": map[string]any{
		"error":    "ctd_blockers",
		"message":  fmt.Sprintf("Your mod has %d blocking issue(s) that could make the artifact unsafe or omit authored content. Review them in Check my mod, then repair them or confirm the risk to export. Confirming lets this export continue; it does not repair or remove the finding.", len(blockers)),
		"blockers": blockers,
	}})
	return false, nil
}

func advancedExportFindings(issues []exportjob.ExportIssue, acknowledge bool) []exportengine.ExportFinding {
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
			sourceValue = map[string]any{"producer": "advanced_export_gate"}
		}
		source, err := json.Marshal(sourceValue)
		if err != nil {
			source, _ = json.Marshal(map[string]any{"producer": "advanced_export_gate", "source_encoding": "unavailable"})
		}
		var pathValue *string
		if value, ok := issue["path"].(string); ok && strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			pathValue = &value
		}
		var lineValue *int
		switch value := issue["line"].(type) {
		case int:
			if value > 0 {
				lineValue = &value
			}
		case float64:
			line := int(value)
			if line > 0 && float64(line) == value {
				lineValue = &line
			}
		case json.Number:
			if line, err := strconv.Atoi(value.String()); err == nil && line > 0 {
				lineValue = &line
			}
		}
		playerImpact, _ := issue["player_impact"].(string)
		editRoute, _ := issue["repair_route"].(string)
		if strings.TrimSpace(editRoute) == "" {
			editRoute, _ = issue["fix_hint"].(string)
		}
		severity := strings.TrimSpace(fmt.Sprint(issue["severity"]))
		findings = append(findings, exportengine.ExportFinding{
			Code: code, Severity: severity, Source: source, Path: pathValue, Line: lineValue,
			Message: message, PlayerImpact: playerImpact, EditRoute: editRoute,
			Acknowledged:   acknowledge && severity == "blocker" && !nonOverridableAdvancedBlocker(code),
			SourceEvidence: map[string]any{"producer_issue": issue},
		})
	}
	return findings
}

func nonOverridableAdvancedBlocker(code string) bool {
	return strings.Contains(strings.ToLower(code), "raw_binary_bytes_unavailable")
}

type advancedUpload struct {
	TempDir        string
	FilenameOnDisk string
	Filename       string

	Folder string
	Fields map[string]string
}

var errAdvancedUploadTooLarge = errors.New("advanced export upload too large")

func receiveAdvancedUpload(w http.ResponseWriter, r *http.Request) (advancedUpload, func(), error) {
	emptyCleanup := func() {}
	r.Body = http.MaxBytesReader(w, r.Body, advancedUploadLimit+advancedFieldBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		return advancedUpload{}, emptyCleanup, fmt.Errorf("multipart form required: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "hoi4-advanced-export-*")
	if err != nil {
		return advancedUpload{}, emptyCleanup, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	out := advancedUpload{TempDir: tempDir, FilenameOnDisk: filepath.Join(tempDir, "base.zip"), Fields: map[string]string{}}
	fileSeen := false
	for {
		if err := r.Context().Err(); err != nil {
			cleanup()
			return advancedUpload{}, emptyCleanup, err
		}
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cleanup()
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				return advancedUpload{}, emptyCleanup, errAdvancedUploadTooLarge
			}
			return advancedUpload{}, emptyCleanup, err
		}
		name := part.FormName()
		if name == "base_mod_file" && part.FileName() != "" {
			if fileSeen {
				_ = part.Close()
				cleanup()
				return advancedUpload{}, emptyCleanup, errors.New("base_mod_file supplied more than once")
			}
			fileSeen = true
			out.Filename = part.FileName()
			file, err := os.OpenFile(out.FilenameOnDisk, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err == nil {
				var size int64
				size, err = copyContextBounded(r.Context(), file, part, advancedUploadLimit)
				if closeErr := file.Close(); err == nil {
					err = closeErr
				}
				if err == nil && size == 0 {
					err = errors.New("base_mod_file is empty")
				}
			}
			_ = part.Close()
			if err != nil {
				cleanup()
				if errors.Is(err, errAdvancedUploadTooLarge) {
					return advancedUpload{}, emptyCleanup, errAdvancedUploadTooLarge
				}
				return advancedUpload{}, emptyCleanup, err
			}
			continue
		}
		body, err := io.ReadAll(io.LimitReader(part, advancedFieldBytes+1))
		_ = part.Close()
		if err != nil || int64(len(body)) > advancedFieldBytes {
			cleanup()
			return advancedUpload{}, emptyCleanup, errAdvancedUploadTooLarge
		}
		if name != "" {
			out.Fields[name] = string(body)
		}
	}
	folder := strings.TrimSpace(out.Fields["base_mod_folder"])
	switch {
	case fileSeen && folder != "":
		cleanup()
		return advancedUpload{}, emptyCleanup, errors.New("send base_mod_file or base_mod_folder, not both")
	case folder != "":
		dir, err := absoluteDir(folder, false)
		if err == nil {
			out.Filename, err = workbench.ModName(dir)
		}
		if err != nil {
			cleanup()
			return advancedUpload{}, emptyCleanup, err
		}
		out.Folder = dir
	case !fileSeen:
		cleanup()
		return advancedUpload{}, emptyCleanup, errors.New("base_mod_file or base_mod_folder is required")
	}
	return out, cleanup, nil
}

func copyContextBounded(ctx context.Context, dst io.Writer, src io.Reader, max int64) (int64, error) {
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := src.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > max {
				return total, errAdvancedUploadTooLarge
			}
			if _, writeErr := dst.Write(buffer[:n]); writeErr != nil {
				return total, writeErr
			}
		}
		if errors.Is(err, io.EOF) {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func parseOptionalBool(raw string) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("invalid boolean value %q", raw)
	}
	return value, nil
}

func writeAdvancedUploadError(w http.ResponseWriter, err error) {
	if errors.Is(err, errAdvancedUploadTooLarge) || errors.Is(err, exportmerge.ErrArchiveTooLarge) {
		server.Fail(w, http.StatusRequestEntityTooLarge, errors.New("base mod archive exceeds safety budget"))
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	if errors.Is(err, workbench.ErrNotAMod) {
		server.JSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": map[string]any{
			"error": "not_a_mod", "message": err.Error(),
		}})
		return
	}
	server.Fail(w, http.StatusUnprocessableEntity, err)
}

func writeAdvancedArchiveError(w http.ResponseWriter, err error) {
	if errors.Is(err, exportmerge.ErrArchiveTooLarge) {
		server.Fail(w, http.StatusRequestEntityTooLarge, err)
		return
	}
	server.Fail(w, http.StatusUnprocessableEntity, err)
}

func isAdvancedBinary(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range []string{".dds", ".png", ".jpg", ".jpeg", ".tga", ".bmp", ".gif", ".ogg", ".wav", ".mp3"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

var unsafeZIPFilename = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitiseAdvancedZIPName(name string) string {
	name = unsafeZIPFilename.ReplaceAllString(strings.TrimSpace(filepath.Base(name)), "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		name = "mod"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

func sanitiseAdvancedUnicodeZIPName(name string) string {
	name = filepath.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.Trim(name, " ._-")
	if name == "" {
		name = "mod"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[:116]) + ".zip"
	}
	return name
}

func serveTemporaryArtifact(w http.ResponseWriter, r *http.Request, filename, downloadName string) {
	file, err := os.Open(filename)
	if err != nil {
		server.Fail(w, http.StatusInternalServerError, errors.New("artifact unavailable"))
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		server.Fail(w, http.StatusInternalServerError, errors.New("artifact unavailable"))
		return
	}
	http.ServeContent(w, r, downloadName, info.ModTime().Truncate(time.Second), file)
}

func setAdvancedArtifactHeaders(w http.ResponseWriter, filename string, manifest exportengine.Manifest) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer reader.Close()
	var manifestBody []byte
	for _, member := range reader.File {
		if member.Name != manifest.ManifestPath {
			continue
		}
		rc, err := member.Open()
		if err != nil {
			return err
		}
		manifestBody, err = io.ReadAll(io.LimitReader(rc, 16<<20+1))
		_ = rc.Close()
		if err != nil {
			return err
		}
		if len(manifestBody) > 16<<20 {
			return errors.New("artifact manifest exceeds metadata budget")
		}
		break
	}
	if manifestBody == nil {
		return errors.New("artifact manifest is missing")
	}
	manifestSum := sha256.Sum256(manifestBody)
	w.Header().Set("X-Artifact-Manifest-SHA256", hex.EncodeToString(manifestSum[:]))

	artifactHasher := sha256.New()
	for _, entry := range manifest.Entries {
		contentSum, err := hex.DecodeString(entry.SHA256)
		if err != nil {
			return err
		}
		_, _ = artifactHasher.Write([]byte(entry.Path))
		_, _ = artifactHasher.Write([]byte{0})
		_, _ = artifactHasher.Write(contentSum)
		_, _ = artifactHasher.Write([]byte{0})
	}
	w.Header().Set("X-Artifact-Id", hex.EncodeToString(artifactHasher.Sum(nil)))
	return nil
}
