package technology

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	TechnologySourceKindNone       = "none"
	TechnologySourceKindPlatform   = "platform_certified"
	TechnologySourceKindLocalHOI4  = "local_hoi4"
	TechnologySourceKindProjectMod = "project_mod"

	TechnologySourceNotConnected  = "not_connected"
	TechnologySourceConnected     = "connected"
	TechnologySourceProjectSource = "project_source"
)

type technologySourceMetadata struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	Label         string    `json:"label"`
	Fingerprint   string    `json:"fingerprint"`
	Reused        bool      `json:"reused,omitempty"`
	ConnectedAt   time.Time `json:"connected_at"`
}

type TechnologySourceConnection struct {
	State       string    `json:"state"`
	Kind        string    `json:"kind"`
	Label       string    `json:"label,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	Reused      bool      `json:"reused,omitempty"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
}

func technologySourceFingerprint(source techsurfaceprofiles.SourceFingerprint) string {
	files := append([]techsurfaceprofiles.SourceFile(nil), source.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	var canonical strings.Builder
	for _, file := range files {
		canonical.WriteString(strings.ToLower(strings.TrimSpace(file.Path)))
		canonical.WriteByte('\x00')
		canonical.WriteString(strings.ToLower(strings.TrimSpace(file.SHA256)))
		canonical.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(canonical.String()))
	return hex.EncodeToString(sum[:])
}

func technologySourceConnection(provenance technologySurfaceProvenance, loadedExistingMod bool) TechnologySourceConnection {
	if provenance.TechnologySource != nil && (provenance.TechnologySource.Kind == TechnologySourceKindPlatform || provenance.TechnologySource.Kind == TechnologySourceKindLocalHOI4) {
		meta := provenance.TechnologySource
		return TechnologySourceConnection{State: TechnologySourceConnected, Kind: meta.Kind, Label: meta.Label,
			Fingerprint: meta.Fingerprint, Reused: meta.Reused, ConnectedAt: meta.ConnectedAt}
	}
	if len(provenance.Source.Files) > 0 {
		label := "This project's imported Mod"
		if !loadedExistingMod {
			label = "This project's imported source"
		}
		return TechnologySourceConnection{State: TechnologySourceProjectSource, Kind: TechnologySourceKindProjectMod, Label: label,
			Fingerprint: technologySourceFingerprint(provenance.Source)}
	}
	return TechnologySourceConnection{State: TechnologySourceNotConnected, Kind: TechnologySourceKindNone}
}

func normalizedTechnologySourceLabel(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return "Local HOI4"
	}
	if len([]rune(value)) > 100 {
		value = string([]rune(value)[:100])
	}
	return value
}

func (h *handlers) surfaceSource(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	var input struct {
		Label string `json:"label"`
	}
	if err := decodeBody(w, r, 1<<20, true, &input); err != nil {
		detail(w, http.StatusUnprocessableEntity, "Invalid technology source payload")
		return
	}
	result, err := markTechnologySourceConnected(p, input.Label)
	var branchError surfaceError
	switch {
	case errors.As(err, &branchError):
		detail(w, http.StatusUnprocessableEntity, branchError.Detail)
	case err != nil:
		detail(w, http.StatusServiceUnavailable, "technology source unavailable")
	default:
		apiutil.JSON(w, http.StatusOK, result)
	}
}

func markTechnologySourceConnected(p *project.Project, label string) (TechnologySourceConnection, error) {
	var result TechnologySourceConnection
	err := p.Write([]string{assetsTable}, func(tx *project.Tx) error {
		if boolean(tx.Record(), "is_loaded_existing_mod") {
			return surfaceError{Detail: "An imported Mod keeps its own technology GUI and technology_tags source."}
		}
		row := guiSourceRow(tx.Rows(assetsTable))
		if row == nil || strings.TrimSpace(text(row, "obs_key", "")) == "" || row["provenance"] == nil {
			return surfaceError{Detail: "Choose the local HOI4 folder before confirming this connection."}
		}
		root, provenance, err := decodeTechnologySurfaceProvenance(row["provenance"])
		if err != nil {
			return err
		}
		if _, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(provenance.Source, techsurfaceprofiles.DLCState{})); err != nil {
			return err
		}
		provenance.TechnologySource = &technologySourceMetadata{SchemaVersion: 1, Kind: TechnologySourceKindLocalHOI4,
			Label: normalizedTechnologySourceLabel(label), Fingerprint: technologySourceFingerprint(provenance.Source), ConnectedAt: time.Now().UTC()}
		root["technology_surface"] = provenance
		if _, err := tx.Update(assetsTable, project.Int64(row["id"]), func(asset project.Row) {
			asset["provenance"] = jsonValue(root)
			asset["created_at"] = workbench.NowTimestamp()
		}); err != nil {
			return err
		}
		result = technologySourceConnection(provenance, false)
		return nil
	})
	return result, err
}

const (
	ExpectedVanillaRawVersion  = "1.19.2.0"
	ExpectedVanillaGameVersion = "1.19.2.0 (Operation Postern v1.19.2.0.a729 (d245))"
)

const certifiedTechnologySourceFileLimit int64 = 8 << 20

const (
	ErrCodeCertifiedSourceUnconfigured    = "certified_tech_source_unconfigured"
	ErrCodeCertifiedSourceFileMissing     = "certified_tech_source_file_missing"
	ErrCodeCertifiedSourceVersionMismatch = "certified_tech_source_version_mismatch"
	ErrCodeCertifiedSourceContentMismatch = "certified_tech_source_content_mismatch"
	ErrCodeCertifiedSourceUnavailable     = "certified_tech_source_unavailable"
)

type certifiedSourceError struct {
	Code        string
	Path        string
	ExpectedSHA string
	ServerSHA   string
	Cause       string
}

func (e certifiedSourceError) Error() string {
	parts := []string{e.Code}
	if e.Path != "" {
		parts = append(parts, e.Path)
	}
	if e.ServerSHA != "" && e.ExpectedSHA != "" {
		parts = append(parts, fmt.Sprintf("server=%s expected=%s", e.ServerSHA, e.ExpectedSHA))
	}
	if e.Cause != "" {
		parts = append(parts, e.Cause)
	}
	return strings.Join(parts, ": ")
}

func readGameVersion(fsys fs.FS) (raw, display string, known bool) {
	if fsys == nil {
		return "", "", false
	}
	body, err := fs.ReadFile(fsys, "launcher-settings.json")
	if err != nil {
		return "", "", false
	}
	var payload struct {
		Version    string `json:"version"`
		RawVersion string `json:"rawVersion"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")), &payload); err != nil {
		return "", "", false
	}
	raw, display = strings.TrimSpace(payload.RawVersion), strings.TrimSpace(payload.Version)
	return raw, display, raw != ""
}

func (e certifiedSourceError) detail(fsys fs.FS) map[string]any {
	d := map[string]any{
		"code":                  e.Code,
		"env":                   "game_dir",
		"expected_game_version": ExpectedVanillaGameVersion,
		"expected_raw_version":  ExpectedVanillaRawVersion,
	}
	if e.Path != "" {
		d["path"] = e.Path
	}
	if e.ExpectedSHA != "" {
		d["expected_sha256"] = e.ExpectedSHA
	}
	if e.ServerSHA != "" {
		d["server_sha256"] = e.ServerSHA
	}
	if raw, display, known := readGameVersion(fsys); known {
		d["server_raw_version"] = raw
		if display != "" {
			d["server_version"] = display
		} else {
			d["server_version"] = raw
		}
	}
	d["message"] = certifiedSourceMessage(e, d)
	return d
}

func certifiedSourceMessage(e certifiedSourceError, d map[string]any) string {
	switch e.Code {
	case ErrCodeCertifiedSourceUnconfigured:
		return "No Hearts of Iron IV folder is set, so the certified HOI4 research baseline " +
			"cannot be read. Nothing was changed. Set the game folder in settings."
	case ErrCodeCertifiedSourceFileMissing:
		return fmt.Sprintf(
			"The certified HOI4 research baseline needs %s in the Hearts of Iron IV folder, "+
				"but that file is not there. Nothing was changed. Point the settings at a "+
				"complete %s installation.",
			e.Path, ExpectedVanillaRawVersion)
	case ErrCodeCertifiedSourceContentMismatch:
		reported, _ := d["server_version"].(string)
		if reported == "" {
			reported = ExpectedVanillaRawVersion
		}
		return fmt.Sprintf(
			"The game folder reports version %s, but the contents of %s do not match "+
				"the audited copy (game sha256 %s, expected %s). The game files are "+
				"mixed-version or modified, so nothing was changed. Verify the game files "+
				"or use a clean %s installation.",
			reported, e.Path, shortSHA(e.ServerSHA), shortSHA(e.ExpectedSHA), ExpectedVanillaRawVersion)
	case ErrCodeCertifiedSourceVersionMismatch:
		server, _ := d["server_version"].(string)
		if server == "" {
			server = "an unidentified version"
		}
		return fmt.Sprintf(
			"The game file %s (%s) is not the audited %s copy the certified research "+
				"baseline was built from, so using it would mix versions. Nothing was "+
				"changed. Use a %s installation, or connect the game folder through the "+
				"import instead.",
			e.Path, server, ExpectedVanillaGameVersion, ExpectedVanillaRawVersion)
	default:
		return "The certified HOI4 research baseline is temporarily unavailable. " +
			"Nothing was changed; retrying may succeed."
	}
}

func shortSHA(sum string) string {
	sum = strings.ToLower(strings.TrimSpace(sum))
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}

func loadCertifiedTechnologySource(fsys fs.FS, expected techsurfaceprofiles.SourceFingerprint) ([]byte, techsurfaceprofiles.SourceFingerprint, error) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	source := techsurfaceprofiles.SourceFingerprint{Files: make([]techsurfaceprofiles.SourceFile, 0, len(expected.Files))}
	var total int64
	for _, expectedFile := range expected.Files {
		rel := strings.TrimSpace(expectedFile.Path)
		info, err := fs.Stat(fsys, rel)
		if err != nil {
			_ = writer.Close()
			return nil, source, certifiedSourceError{Code: ErrCodeCertifiedSourceFileMissing, Path: expectedFile.Path, Cause: err.Error()}
		}
		if !info.Mode().IsRegular() {
			_ = writer.Close()
			return nil, source, certifiedSourceError{Code: ErrCodeCertifiedSourceFileMissing, Path: expectedFile.Path, Cause: "not a regular file"}
		}
		if info.Size() < 0 || info.Size() > certifiedTechnologySourceFileLimit || total > certifiedTechnologySourceFileLimit-info.Size() {
			_ = writer.Close()
			return nil, source, certifiedSourceError{Code: ErrCodeCertifiedSourceUnavailable, Path: expectedFile.Path,
				Cause: fmt.Sprintf("exceeds %d byte limit", certifiedTechnologySourceFileLimit)}
		}
		body, err := fs.ReadFile(fsys, rel)
		if err != nil {
			_ = writer.Close()
			return nil, source, certifiedSourceError{Code: ErrCodeCertifiedSourceFileMissing, Path: expectedFile.Path, Cause: err.Error()}
		}
		total += int64(len(body))
		sum := fmt.Sprintf("%x", sha256.Sum256(body))
		if !strings.EqualFold(sum, strings.TrimSpace(expectedFile.SHA256)) {
			_ = writer.Close()

			code := ErrCodeCertifiedSourceVersionMismatch
			if raw, _, known := readGameVersion(fsys); known && strings.EqualFold(raw, ExpectedVanillaRawVersion) {
				code = ErrCodeCertifiedSourceContentMismatch
			}
			return nil, source, certifiedSourceError{Code: code, Path: expectedFile.Path,
				ExpectedSHA: strings.ToLower(strings.TrimSpace(expectedFile.SHA256)), ServerSHA: sum}
		}
		member, err := writer.Create(expectedFile.Path)
		if err != nil {
			_ = writer.Close()
			return nil, source, err
		}
		if _, err := member.Write(body); err != nil {
			_ = writer.Close()
			return nil, source, err
		}
		source.Files = append(source.Files, techsurfaceprofiles.SourceFile{Path: expectedFile.Path, SHA256: sum})
	}
	if err := writer.Close(); err != nil {
		return nil, source, err
	}
	return archive.Bytes(), source, nil
}

type certifiedTarget struct {
	replaceLegacy    bool
	legacyKey        string
	legacyProvenance string
}

func prepareCertifiedTechnologySource(p *project.Project) (certifiedTarget, TechnologySourceConnection, bool, error) {
	record := p.Record()
	loaded := boolean(record, "is_loaded_existing_mod")
	assets, err := p.Rows(assetsTable)
	if err != nil {
		return certifiedTarget{}, TechnologySourceConnection{}, false, err
	}
	if row := guiSourceRow(assets); row != nil && strings.TrimSpace(text(row, "obs_key", "")) != "" {
		root, provenance, err := decodeTechnologySurfaceProvenance(row["provenance"])
		if err != nil {
			return certifiedTarget{}, TechnologySourceConnection{}, false, err
		}

		if !loaded && len(provenance.Source.Files) == 0 && provenance.TechnologySource == nil {
			if strings.TrimSpace(text(record, "uuid", "")) == "" {
				return certifiedTarget{}, TechnologySourceConnection{}, false, errors.New("technology source project UUID is missing")
			}
			canonical, err := json.Marshal(root)
			if err != nil {
				return certifiedTarget{}, TechnologySourceConnection{}, false, err
			}
			return certifiedTarget{replaceLegacy: true, legacyKey: strings.TrimSpace(text(row, "obs_key", "")), legacyProvenance: string(canonical)},
				TechnologySourceConnection{}, true, nil
		}
		return certifiedTarget{}, technologySourceConnection(provenance, loaded), false, nil
	}
	if loaded {
		return certifiedTarget{}, TechnologySourceConnection{}, false,
			surfaceError{Detail: "This imported Mod does not contain a technology GUI and technology_tags source."}
	}
	if strings.TrimSpace(text(record, "uuid", "")) == "" {
		return certifiedTarget{}, TechnologySourceConnection{}, false, errors.New("technology source project UUID is missing")
	}
	return certifiedTarget{}, TechnologySourceConnection{}, true, nil
}

func commitCertifiedTechnologySource(p *project.Project, target certifiedTarget, key string, source techsurfaceprofiles.SourceFingerprint) (TechnologySourceConnection, error) {
	if _, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(source, techsurfaceprofiles.DLCState{})); err != nil {
		return TechnologySourceConnection{}, err
	}
	provenance := technologySurfaceProvenance{
		SchemaVersion: 1,
		Source:        source,
		TechnologySource: &technologySourceMetadata{
			SchemaVersion: 1,
			Kind:          TechnologySourceKindPlatform,
			Label:         "Hagane Works certified HOI4",
			Fingerprint:   technologySourceFingerprint(source),
			ConnectedAt:   time.Now().UTC(),
		},
	}
	root := map[string]any{}
	if target.replaceLegacy {
		if err := json.Unmarshal([]byte(target.legacyProvenance), &root); err != nil {
			return TechnologySourceConnection{}, err
		}
		if root == nil {
			root = map[string]any{}
		}
	}
	root["technology_surface"] = provenance
	acquired := surfaceError{Detail: "This project acquired a technology source before the certified baseline was attached."}
	err := p.Write([]string{assetsTable}, func(tx *project.Tx) error {
		if boolean(tx.Record(), "is_loaded_existing_mod") {
			return acquired
		}
		row := guiSourceRow(tx.Rows(assetsTable))
		if !target.replaceLegacy {
			if row != nil {
				return acquired
			}
			tx.Insert(assetsTable, project.Row{
				"asset_type": "gui_source", "gfx_name": "snapshot.zip", "obs_key": key,
				"provenance": jsonValue(root), "created_at": workbench.NowTimestamp(),
			})
			return nil
		}

		if row == nil || text(row, "obs_key", "") != target.legacyKey {
			return acquired
		}
		current, provenanceNow, err := decodeTechnologySurfaceProvenance(row["provenance"])
		if err != nil {
			return err
		}
		canonical, _ := json.Marshal(current)
		if string(canonical) != target.legacyProvenance || len(provenanceNow.Source.Files) > 0 || provenanceNow.TechnologySource != nil {
			return acquired
		}
		_, err = tx.Update(assetsTable, project.Int64(row["id"]), func(asset project.Row) {
			asset["obs_key"] = key
			asset["provenance"] = jsonValue(root)
			asset["created_at"] = workbench.NowTimestamp()
		})
		return err
	})
	if err != nil {
		return TechnologySourceConnection{}, err
	}
	return technologySourceConnection(provenance, false), nil
}

func (h *handlers) surfaceCertifiedSource(w http.ResponseWriter, r *http.Request) {
	p, ok := h.open(w, r)
	if !ok {
		return
	}
	unavailable := func(cause error, fsys fs.FS) {
		apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"detail": certifiedSourceError{Code: ErrCodeCertifiedSourceUnavailable, Cause: cause.Error()}.detail(fsys)})
	}
	target, existing, attach, err := prepareCertifiedTechnologySource(p)
	var branchError surfaceError
	if errors.As(err, &branchError) {
		detail(w, http.StatusUnprocessableEntity, branchError.Detail)
		return
	}
	if err != nil {
		unavailable(err, nil)
		return
	}
	if !attach {
		apiutil.JSON(w, http.StatusOK, existing)
		return
	}
	var fsys fs.FS
	if h.deps.Game != nil {
		fsys, err = h.deps.Game()
	}
	if fsys == nil || err != nil {
		apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"detail": certifiedSourceError{Code: ErrCodeCertifiedSourceUnconfigured}.detail(nil)})
		return
	}
	archive, source, err := loadCertifiedTechnologySource(fsys, techsurfaceprofiles.AuditedSourceFingerprint())
	if err != nil {
		var typed certifiedSourceError
		if errors.As(err, &typed) {
			apiutil.JSON(w, http.StatusServiceUnavailable, map[string]any{"detail": typed.detail(fsys)})
			return
		}
		unavailable(err, nil)
		return
	}
	blobs := p.Blobs()
	archiveSHA256 := fmt.Sprintf("%x", sha256.Sum256(archive))
	key, err := importassets.CertifiedGUISourceSnapshotKey(blobs.EnvPrefix(), blobs.ProjectUUID(), archiveSHA256)
	if err != nil {
		unavailable(err, nil)
		return
	}
	if err := blobs.Put(key, "application/zip", int64(len(archive)), bytes.NewReader(archive)); err != nil {
		unavailable(err, nil)
		return
	}
	result, err := commitCertifiedTechnologySource(p, target, key, source)
	if errors.As(err, &branchError) {
		detail(w, http.StatusConflict, branchError.Detail)
		return
	}
	if err != nil {
		unavailable(err, nil)
		return
	}
	apiutil.JSON(w, http.StatusOK, result)
}
