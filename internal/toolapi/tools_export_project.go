package toolapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	exportJobPollInterval = 2 * time.Second
	exportJobPollBudget   = 10 * time.Minute
)

type exportProjectIn struct {
	ProjectID           int64  `json:"project_id"`
	IncludeImported     bool   `json:"include_imported,omitempty" jsonschema:"false = source=created rows only; true = also emit untouched imported rows"`
	ReplacePaths        string `json:"replace_paths,omitempty" jsonschema:"comma-separated whole vanilla directories to suppress"`
	AcknowledgeBlockers bool   `json:"acknowledge_blockers,omitempty" jsonschema:"exports are never blocked; true only records blocker findings in the artifact's findings report as acknowledged"`
	SaveTo              string `json:"save_to,omitempty" jsonschema:"an existing folder on this computer to save the mod ZIP in (as suggested_filename, replacing a file of that name); omitted = only build and summarize"`
}

type exportProjectOut struct {
	ProjectID         int64    `json:"project_id"`
	FileCount         int      `json:"file_count"`
	SizeBytes         int64    `json:"size_bytes"`
	SuggestedFilename string   `json:"suggested_filename"`
	Files             []string `json:"files"`
	SavedTo           string   `json:"saved_to,omitempty"`
	Note              string   `json:"note"`
	JobID             string   `json:"job_id"`
}

const exportZIPMaxBytes = 2 << 30

func splitReplacePaths(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func exportJobStatusActive(status string) bool {
	switch status {
	case "queued", "snapshotting", "running":
		return true
	default:
		return false
	}
}

func summarizeExportFiles(files []string) (kept []string, total int) {
	kept = make([]string, 0, len(files))
	for _, name := range files {
		if strings.HasSuffix(name, "_CROSS_REF_SKIPPED.log") {
			continue
		}
		kept = append(kept, name)
	}
	sort.Strings(kept)
	total = len(kept)
	if len(kept) > 200 {
		kept = kept[:200]
	}
	return kept, total
}

func registerExportProject(s *Server, jobs *RESTProxy) {
	AddTool(s, &Tool{
		Name: "export_project",
		Description: "Build the mod ZIP and return a summary (file list + count). Give save_to, an existing folder " +
			"on this computer, to save the ZIP there; the player can also export or install the mod from the " +
			"program's Export page.\n" +
			"include_imported=false emits source=created rows only; include_imported=true also emits " +
			"untouched imported rows. replace_paths is a comma-separated list of whole vanilla " +
			"directories to suppress, so each listed directory must be fully re-supplied by the mod. " +
			"Projects loaded from an existing mod inherit its descriptor replace_path entries on full " +
			"export; a created-only delta export is blocked until that replacement scope can be " +
			"supplied safely.\n" +
			"Exports are never blocked on validation findings: review them with the validation tool " +
			"before shipping. acknowledge_blockers is retained for compatibility and has no effect.",
	}, func(ctx context.Context, _ *CallToolRequest, in exportProjectIn) (*CallToolResult, exportProjectOut, error) {
		saveTo := strings.TrimSpace(in.SaveTo)
		if saveTo != "" {
			info, err := os.Stat(saveTo)
			if err != nil || !info.IsDir() {
				return nil, exportProjectOut{}, fmt.Errorf("save_to must be an existing folder: %s", saveTo)
			}
		}
		body := map[string]any{
			"project_id":           in.ProjectID,
			"include_imported":     in.IncludeImported,
			"replace_paths":        splitReplacePaths(in.ReplacePaths),
			"acknowledge_blockers": in.AcknowledgeBlockers,
		}
		job, err := proxyAuthedCall(ctx, jobs, "POST", "/api/go/export/jobs", body)
		if err != nil {
			return nil, exportProjectOut{}, err
		}
		jobID := stringField(job, "job_id")
		if jobID == "" {
			return nil, exportProjectOut{}, fmt.Errorf("export job endpoint returned no job_id")
		}

		deadline := time.NewTimer(exportJobPollBudget)
		defer deadline.Stop()
		statusPath := fmt.Sprintf("/api/go/export/jobs/%d/%s", in.ProjectID, url.PathEscape(jobID))
		for exportJobStatusActive(stringField(job, "status")) {
			select {
			case <-ctx.Done():
				return nil, exportProjectOut{}, fmt.Errorf(
					"export cancelled while job %s was %s: %w", jobID, stringField(job, "status"), ctx.Err())
			case <-deadline.C:
				return nil, exportProjectOut{}, fmt.Errorf(
					"export job %s still %s after %s; export the mod from the program's Export page instead",
					jobID, stringField(job, "status"), exportJobPollBudget)
			case <-time.After(exportJobPollInterval):
			}
			job, err = proxyAuthedCall(ctx, jobs, "GET", statusPath, nil)
			if err != nil {
				return nil, exportProjectOut{}, err
			}
		}

		switch status := stringField(job, "status"); status {
		case "complete":

		case "failed":
			detail := stringField(job, "error")
			if detail == "" {
				detail = fmt.Sprintf("%v", job["error"])
			}
			return nil, exportProjectOut{}, fmt.Errorf("export job %s failed: %s", jobID, detail)
		case "cancelled":
			return nil, exportProjectOut{}, fmt.Errorf("export job %s was cancelled before completing", jobID)
		default:
			return nil, exportProjectOut{}, fmt.Errorf("export job %s ended in unexpected status %q", jobID, status)
		}

		files, total := summarizeExportFiles(stringSliceField(job, "files"))
		out := exportProjectOut{
			ProjectID:         in.ProjectID,
			FileCount:         total,
			SizeBytes:         int64Field(job, "size_bytes"),
			SuggestedFilename: stringField(job, "filename"),
			Files:             files,
			Note:              "Export built. Give save_to to save the ZIP in a folder, or export/install it from the program's Export page.",
			JobID:             jobID,
		}
		if saveTo != "" {
			saved, err := saveExportZIP(ctx, jobs, statusPath+"/download", saveTo, out.SuggestedFilename)
			if err != nil {
				return nil, exportProjectOut{}, err
			}
			out.SavedTo = saved
			out.Note = "Export built and saved to saved_to."
		}
		return nil, out, nil
	})
}

func saveExportZIP(ctx context.Context, jobs *RESTProxy, downloadPath, folder, name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || !strings.HasSuffix(strings.ToLower(name), ".zip") {
		return "", errors.New("the export job named no ZIP file")
	}
	data, err := jobs.callRawLimited(ctx, "GET", downloadPath, nil, exportZIPMaxBytes)
	if err != nil {
		return "", fmt.Errorf("could not read the finished ZIP: %w", err)
	}
	target := filepath.Join(folder, name)
	tmp, err := os.CreateTemp(folder, name+".*.tmp")
	if err != nil {
		return "", err
	}
	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), target)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("could not save the ZIP in %s: %w", folder, err)
	}
	return target, nil
}
