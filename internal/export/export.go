package export

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportjob"
	"haganeworks/internal/project"
)

type Options struct {
	IncludeImported     bool     `json:"include_imported"`
	ReplacePaths        []string `json:"replace_paths"`
	AcknowledgeBlockers bool     `json:"acknowledge_blockers"`
	BuildIdentity       bool     `json:"build_identity"`

	Locale string `json:"locale"`

	Issues func(*project.Project) exportjob.IssueCollector `json:"-"`
}

func Snapshot(p *project.Project) (exportengine.Snapshot, error) {
	record, err := json.Marshal(p.Record())
	if err != nil {
		return exportengine.Snapshot{}, err
	}
	tables := map[string][]json.RawMessage{}
	for _, table := range exportengine.CoreTables() {
		rows, err := p.Rows(table)
		if err != nil {
			return exportengine.Snapshot{}, err
		}
		raw := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			encoded, err := json.Marshal(row)
			if err != nil {
				return exportengine.Snapshot{}, fmt.Errorf("%s row %v: %w", table, row["id"], err)
			}
			raw = append(raw, encoded)
		}
		tables[table] = raw
	}
	return exportengine.NewSnapshot(p.ID(), record, tables)
}

func BlobResolver(p *project.Project) exportengine.BlobResolver {
	return p.Blobs().Get
}

func Build(ctx context.Context, p *project.Project, opts Options, artifactPath string) (exportjob.Result, error) {

	if err := AwaitGameIndex(ctx); err != nil {
		return exportjob.Result{}, err
	}
	snapshot, err := Snapshot(p)
	if err != nil {
		return exportjob.Result{}, err
	}
	return exportjob.Build(ctx, exportjob.BuildInput{
		Snapshot:            snapshot,
		IncludeImported:     opts.IncludeImported,
		ReplacePaths:        opts.ReplacePaths,
		AcknowledgeBlockers: opts.AcknowledgeBlockers,
		BuildIdentity:       opts.BuildIdentity,
		ArtifactPath:        artifactPath,
		ResolveBlob:         BlobResolver(p),
		Issues:              issues(p, opts),
		Locale:              opts.Locale,
	})
}

func AwaitGameIndex(ctx context.Context) error {
	status, err := gameindex.Default().Wait(ctx)
	if errors.Is(err, gameindex.ErrNoGame) {
		return nil
	}
	if err != nil {
		return err
	}
	if status.State == "failed" {
		return fmt.Errorf("the base game's data could not be read (%s); rebuild the game index from the settings", status.Error)
	}
	return nil
}

func issues(p *project.Project, opts Options) exportjob.IssueCollector {
	if opts.Issues == nil {
		return nil
	}
	return opts.Issues(p)
}

type Job struct {
	ID        string
	ProjectID int64
	Status    string
	Error     string
	Result    exportjob.Result
	Path      string
	Started   time.Time
}

type Jobs struct {
	dir  string
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewJobs() (*Jobs, error) {
	dir, err := os.MkdirTemp("", "haganeworks-exports-")
	if err != nil {
		return nil, err
	}
	return &Jobs{dir: dir, jobs: map[string]*Job{}}, nil
}

func (j *Jobs) Close() error { return os.RemoveAll(j.dir) }

func (j *Jobs) Start(p *project.Project, opts Options) Job {
	id := newID()
	job := &Job{ID: id, ProjectID: p.ID(), Status: "running", Path: filepath.Join(j.dir, id+".zip"), Started: time.Now()}
	j.mu.Lock()
	j.jobs[id] = job
	started := *job
	j.mu.Unlock()
	go func() {
		result, err := Build(context.Background(), p, opts, job.Path)
		j.mu.Lock()
		defer j.mu.Unlock()
		if err != nil {
			job.Status, job.Error = "failed", err.Error()
			return
		}
		job.Status, job.Result = "complete", result
	}()
	return started
}

func (j *Jobs) Get(projectID int64, id string) (Job, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	job, ok := j.jobs[id]
	if !ok || job.ProjectID != projectID {
		return Job{}, false
	}
	return *job, true
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const installMarker = ".haganeworks-install"

var ErrFolderTaken = errors.New("a different mod already uses this folder name in the mod folder")

type InstallResult struct {
	ModFolder  string `json:"mod_folder"`
	Descriptor string `json:"descriptor"`
	Files      int    `json:"files"`
}

func Install(zipPath, modDir, projectUUID string) (InstallResult, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return InstallResult{}, err
	}
	defer reader.Close()
	var descriptor, folder string
	for _, file := range reader.File {
		name := file.Name
		if !strings.Contains(name, "/") && strings.HasSuffix(name, ".mod") {
			descriptor = name
			folder = strings.TrimSuffix(name, ".mod")
		}
	}
	if descriptor == "" {
		return InstallResult{}, errors.New("the exported ZIP has no .mod descriptor")
	}
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		return InstallResult{}, err
	}
	target := filepath.Join(modDir, folder)
	if _, err := os.Stat(target); err == nil {
		marker, readErr := os.ReadFile(filepath.Join(target, installMarker))
		if readErr != nil || strings.TrimSpace(string(marker)) != projectUUID {
			return InstallResult{}, fmt.Errorf("%w: %s", ErrFolderTaken, target)
		}

		if err := os.RemoveAll(target); err != nil {
			return InstallResult{}, err
		}
	}
	count := 0
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rel := path.Clean(file.Name)
		if rel != descriptor && !strings.HasPrefix(rel, folder+"/") || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
			continue
		}
		out := filepath.Join(modDir, filepath.FromSlash(rel))
		if err := extract(file, out); err != nil {
			return InstallResult{}, err
		}
		count++
	}
	if err := os.WriteFile(filepath.Join(target, installMarker), []byte(projectUUID+"\n"), 0o644); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{ModFolder: target, Descriptor: filepath.Join(modDir, descriptor), Files: count}, nil
}

func extract(file *zip.File, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(out)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
