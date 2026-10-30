package rawfiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/project"
)

const table = "project_raw_files"

const maxRawBinaryBytes = 25 * 1024 * 1024

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := func(fn func(*project.Project, http.ResponseWriter, *http.Request) error) http.HandlerFunc {
		return apiutil.Handler(deps.Open, fn)
	}
	const prefix = "/api/go/projects/{id}/raw-files"
	mux.HandleFunc("GET "+prefix, h(listRawFiles))
	mux.HandleFunc("POST "+prefix, h(createRawFile))
	mux.HandleFunc("GET "+prefix+"/templates", h(rawTemplates))
	mux.HandleFunc("POST "+prefix+"/binary", h(createRawBinary))
	mux.HandleFunc("POST "+prefix+"/from-template", h(fromTemplate))
	mux.HandleFunc("POST "+prefix+"/from-template/{route}", h(fromTemplate))
	mux.HandleFunc("GET "+prefix+"/{rid}", h(getRawFile))
	mux.HandleFunc("PUT "+prefix+"/{rid}", h(updateRawFile))
	mux.HandleFunc("DELETE "+prefix+"/{rid}", h(deleteRawFile))

	mux.HandleFunc("POST /api/go/projects/{id}/cross-ref-repairs/preview", h(crossRefPreview))
	mux.HandleFunc("POST /api/go/projects/{id}/cross-ref-repairs/apply", h(crossRefApply))

	mux.HandleFunc("POST /api/go/projects/{id}/import/characters", h(importCharacters))
	mux.HandleFunc("POST /api/go/projects/{id}/import/technologies", h(importTechnologies))
}

type RawFile struct {
	ID                  int64                        `json:"id"`
	ProjectID           int64                        `json:"project_id"`
	DestPath            string                       `json:"dest_path"`
	Content             string                       `json:"content"`
	IsBinary            bool                         `json:"is_binary"`
	GenerateReplacePath bool                         `json:"generate_replace_path"`
	RawKind             string                       `json:"raw_kind"`
	TargetLayer         *string                      `json:"target_layer"`
	TargetSHA256        *string                      `json:"target_sha256"`
	Note                string                       `json:"note"`
	CreatedAt           string                       `json:"created_at"`
	UpdatedAt           string                       `json:"updated_at"`
	Warnings            []clausewitz.AnalysisFinding `json:"warnings"`
}

func str(row project.Row, key string) string {
	s, _ := row[key].(string)
	return s
}

func strPtr(row project.Row, key string) *string {
	if s, ok := row[key].(string); ok {
		return &s
	}
	return nil
}

func boolean(row project.Row, key string) bool {
	b, _ := row[key].(bool)
	return b
}

func rawKindOrDefault(k string) string {
	if strings.TrimSpace(k) == "" {
		return "verbatim_override"
	}
	return k
}

func rawFileFromRow(row project.Row) RawFile {
	kind := str(row, "raw_kind")
	if kind == "" {
		kind = "verbatim_override"
	}
	return RawFile{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		DestPath: str(row, "dest_path"), Content: str(row, "content"),
		IsBinary: boolean(row, "is_binary"), GenerateReplacePath: boolean(row, "generate_replace_path"),
		RawKind: kind, TargetLayer: strPtr(row, "target_layer"), TargetSHA256: strPtr(row, "target_sha256"),
		Note: str(row, "note"), CreatedAt: str(row, "created_at"), UpdatedAt: str(row, "updated_at"),
		Warnings: []clausewitz.AnalysisFinding{},
	}
}

func withContentWarnings(rf RawFile) RawFile {
	rf.Warnings = []clausewitz.AnalysisFinding{}
	if rf.IsBinary || rf.Content == "" {
		return rf
	}
	if ws, rerr := GateRawContent(rf.DestPath, rf.Content, false); rerr == nil && len(ws) > 0 {
		rf.Warnings = ws
	}
	return rf
}

func supportedVersion(p *project.Project) string {
	v, _ := p.Record()["supported_version"].(string)
	return v
}

func schema422(loc []any, msg string, input any) error {
	return &apiutil.StatusError{Status: http.StatusUnprocessableEntity, Body: map[string]any{"detail": []map[string]any{{
		"type": "value_error", "loc": loc, "msg": "Value error, " + msg, "input": input,
	}}}}
}

func tombstoneFailure(terr *TombstoneError, rawKind, destPath, version string) error {
	status := http.StatusUnprocessableEntity
	if EmptyShadowConflictCodes[terr.Code] {
		status = http.StatusConflict
	}
	repairRoute := "raw_file.empty_shadow_evidence"
	if terr.RepairRoute != "" {
		repairRoute = terr.RepairRoute
	}
	detail := map[string]any{
		"code": terr.Code, "message": terr.Message, "raw_kind": rawKind,
		"target_path": destPath, "project_supported_version": version,
		"repair_route": repairRoute,
	}
	if len(terr.Findings) > 0 {
		detail["findings"] = terr.Findings
		detail["net_delta"] = terr.NetDelta
	}
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": detail}}
}

var errRawFileMissing = apiutil.NotFoundError("RawFile not found")

func sanitizeCreateDest(raw string) (string, error) {
	safe, errMsg := SanitizeDestPath(raw)
	if errMsg == "" {
		errMsg = RejectReservedDest(safe)
	}
	if errMsg != "" {
		return "", schema422([]any{"body", "dest_path"}, errMsg, raw)
	}
	return safe, nil
}

func listRawFiles(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	rows, err := p.Rows(table)
	if err != nil {
		return err
	}
	items := make([]RawFile, 0, len(rows))
	for _, row := range rows {
		items = append(items, rawFileFromRow(row))
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].DestPath < items[j].DestPath })
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func rawTemplates(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(RawTemplatesRegistry())
	return err
}

func rawFileID(r *http.Request) (int64, error) {
	raw := r.PathValue("rid")
	if raw == "from-template" {
		return 0, &apiutil.StatusError{Status: http.StatusMethodNotAllowed, Body: map[string]string{"error": "POST only"}}
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, apiutil.ValidationError("rf_id must be an integer")
	}
	return id, nil
}

func findRawFile(rows []project.Row, id int64) project.Row {
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row
		}
	}
	return nil
}

func getRawFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := rawFileID(r)
	if err != nil {
		return err
	}
	rows, err := p.Rows(table)
	if err != nil {
		return err
	}
	row := findRawFile(rows, id)
	if row == nil {
		return errRawFileMissing
	}
	apiutil.JSON(w, http.StatusOK, rawFileFromRow(row))
	return nil
}

func deleteRawFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := rawFileID(r)
	if err != nil {
		return err
	}
	err = p.Write([]string{table}, func(tx *project.Tx) error {
		if tx.Delete(table, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errRawFileMissing
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func destTaken(tx *project.Tx, dest string, except int64) bool {
	for _, row := range tx.Rows(table) {
		if str(row, "dest_path") == dest && project.Int64(row["id"]) != except {
			return true
		}
	}
	return false
}

type rawFileInsert struct {
	DestPath            string
	Content             string
	IsBinary            bool
	GenerateReplacePath bool
	RawKind             string
	TargetLayer         *string
	TargetSHA256        *string
	Note                string
	ContentSHA256       *string
	ContentOBSKey       *string
}

func optional(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func insertRawFile(p *project.Project, in rawFileInsert, conflict string) (RawFile, error) {
	var stored project.Row
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		if destTaken(tx, in.DestPath, 0) {
			return apiutil.ConflictError(conflict)
		}
		stored = tx.Insert(table, project.Row{
			"dest_path": in.DestPath, "content": in.Content, "is_binary": in.IsBinary,
			"generate_replace_path": in.GenerateReplacePath, "raw_kind": in.RawKind,
			"target_layer": optional(in.TargetLayer), "target_sha256": optional(in.TargetSHA256),
			"note": in.Note, "content_binary": nil,
			"content_sha256": optional(in.ContentSHA256), "content_obs_key": optional(in.ContentOBSKey),
			"source": "created",
		})
		return nil
	})
	if err != nil {
		return RawFile{}, err
	}
	return withContentWarnings(rawFileFromRow(stored)), nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func createRawFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in struct {
		DestPath            string  `json:"dest_path"`
		Content             string  `json:"content"`
		GenerateReplacePath bool    `json:"generate_replace_path"`
		RawKind             string  `json:"raw_kind"`
		TargetLayer         *string `json:"target_layer"`
		TargetSHA256        *string `json:"target_sha256"`
		Note                string  `json:"note"`
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil || json.Unmarshal(body, &in) != nil {
		return apiutil.ValidationError("invalid JSON body")
	}

	var probe map[string]json.RawMessage
	_ = json.Unmarshal(body, &probe)
	if _, ok := probe["dest_path"]; !ok {
		for _, alias := range []string{"path", "file_path", "dest", "filepath", "filename"} {
			if _, hit := probe[alias]; hit {
				return schema422([]any{"body"},
					"the field is 'dest_path', not '"+alias+"' — send "+
						`{"dest_path": "common/on_actions/My_on_actions.txt", ...} `+
						"(mod-relative path to the file to override/emit).", nil)
			}
		}
	}
	if in.RawKind == "" {
		in.RawKind = "verbatim_override"
	}
	safe, err := sanitizeCreateDest(in.DestPath)
	if err != nil {
		return err
	}
	version := supportedVersion(p)
	decision, terr := ValidateRawFileIntent(in.RawKind, safe, in.Content,
		false, in.GenerateReplacePath, version, deref(in.TargetLayer), deref(in.TargetSHA256))
	if terr != nil {
		return tombstoneFailure(terr, in.RawKind, safe, version)
	}
	item, err := insertRawFile(p, rawFileInsert{
		DestPath: decision.DestPath, Content: decision.Content,
		GenerateReplacePath: decision.GenerateReplacePath, RawKind: decision.RawKind,
		TargetLayer: decision.TargetLayer, TargetSHA256: decision.TargetSHA256, Note: in.Note,
	}, "a raw override file for path '"+safe+"' already exists in this project. PUT to update or DELETE first.")
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

func storeRawBlob(p *project.Project, data []byte) (sha, key string, err error) {
	sum := sha256.Sum256(data)
	sha = hex.EncodeToString(sum[:])
	blobs := p.Blobs()
	key = blobs.EnvPrefix() + "/rawblobs/" + sha[:2] + "/" + sha
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "application/octet-stream", int64(len(data)), bytes.NewReader(data)); err != nil {
			return "", "", err
		}
	}
	return sha, key, nil
}

func createRawBinary(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRawBinaryBytes+(4<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return apiutil.ValidationError("invalid multipart form")
	}
	defer r.MultipartForm.RemoveAll()
	safe, errMsg := SanitizeDestPath(r.FormValue("dest_path"))
	if errMsg != "" {
		return apiutil.ValidationError(errMsg)
	}
	rawKind := r.FormValue("raw_kind")
	if rawKind == "" {
		rawKind = "verbatim_override"
	}
	genRP := strings.EqualFold(r.FormValue("generate_replace_path"), "true")
	version := supportedVersion(p)
	decision, terr := ValidateRawFileIntent(rawKind, safe, "", true, genRP, version,
		r.FormValue("target_layer"), r.FormValue("target_sha256"))
	if terr != nil {
		return tombstoneFailure(terr, rawKind, safe, version)
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return apiutil.ValidationError("file is required")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRawBinaryBytes+1))
	_ = file.Close()
	if err != nil {
		return apiutil.ValidationError("failed to read upload")
	}
	if len(data) > maxRawBinaryBytes {
		return &apiutil.StatusError{Status: http.StatusRequestEntityTooLarge, Body: map[string]string{
			"detail": "binary override too large (" + strconv.Itoa(len(data)) + " bytes > " +
				strconv.Itoa(maxRawBinaryBytes) + " byte cap)"}}
	}
	sha, key, err := storeRawBlob(p, data)
	if err != nil {
		return &apiutil.StatusError{Status: http.StatusServiceUnavailable, Body: map[string]string{
			"detail": "object storage write failed; no database blob or pointer was created — retry when storage recovers"}}
	}
	item, err := insertRawFile(p, rawFileInsert{
		DestPath: decision.DestPath, IsBinary: true,
		GenerateReplacePath: decision.GenerateReplacePath, RawKind: decision.RawKind,
		TargetLayer: decision.TargetLayer, TargetSHA256: decision.TargetSHA256,
		Note: r.FormValue("note"), ContentSHA256: &sha, ContentOBSKey: &key,
	}, "a raw override file for path '"+safe+"' already exists in this project. DELETE it first to re-upload.")
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

func updateRawFile(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := rawFileID(r)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		return apiutil.ValidationError("invalid JSON body")
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(body, &probe) != nil {
		return apiutil.ValidationError("invalid JSON body")
	}
	rows, err := p.Rows(table)
	if err != nil {
		return err
	}
	state := findRawFile(rows, id)
	if state == nil {
		return errRawFileMissing
	}

	getStr := func(k string) (*string, bool) {
		raw, ok := probe[k]
		if !ok {
			return nil, false
		}
		var v *string
		if json.Unmarshal(raw, &v) != nil {
			return nil, false
		}
		return v, true
	}
	destPath := str(state, "dest_path")
	if v, ok := getStr("dest_path"); ok && v != nil {
		safe, err := sanitizeCreateDest(*v)
		if err != nil {
			return err
		}
		destPath = safe
	}
	content := str(state, "content")
	if v, ok := getStr("content"); ok && v != nil {
		content = *v
	}
	rawKind := rawKindOrDefault(str(state, "raw_kind"))
	if v, ok := getStr("raw_kind"); ok && v != nil {
		rawKind = *v
	}
	genRP := boolean(state, "generate_replace_path")
	if raw, ok := probe["generate_replace_path"]; ok {
		_ = json.Unmarshal(raw, &genRP)
	}
	layer := str(state, "target_layer")
	if v, ok := getStr("target_layer"); ok {
		layer = deref(v)
	}
	sha := str(state, "target_sha256")
	if v, ok := getStr("target_sha256"); ok {
		sha = deref(v)
	}
	version := supportedVersion(p)
	decision, terr := ValidateRawFileIntent(rawKind, destPath, content,
		boolean(state, "is_binary"), genRP, version, layer, sha)
	if terr != nil {
		return tombstoneFailure(terr, rawKind, destPath, version)
	}
	note, setNote := getStr("note")
	setNote = setNote && note != nil
	var stored project.Row
	err = p.Write([]string{table}, func(tx *project.Tx) error {
		if tx.Find(table, id) == nil {
			return errRawFileMissing
		}
		if destTaken(tx, decision.DestPath, id) {
			return apiutil.ConflictError("dest_path conflict in this project")
		}
		var err error
		stored, err = tx.Update(table, id, func(row project.Row) {
			row["dest_path"] = decision.DestPath
			row["content"] = decision.Content
			row["generate_replace_path"] = decision.GenerateReplacePath
			row["raw_kind"] = decision.RawKind
			row["target_layer"] = optional(decision.TargetLayer)
			row["target_sha256"] = optional(decision.TargetSHA256)
			if setNote {
				row["note"] = *note
			}
			if str(row, "source") == "imported" {
				row["source"] = "created"
			}
		})
		return err
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, withContentWarnings(rawFileFromRow(stored)))
	return nil
}

var registryKeyRe = regexp.MustCompile(`^[a-z0-9_]{2,40}$`)

func fromTemplate(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	route := r.PathValue("route")
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		return apiutil.ValidationError("invalid JSON body")
	}
	var files []RawTemplateFile
	var note string
	switch route {
	case "":
		var in struct {
			PanelKey  string           `json:"panel_key"`
			Title     string           `json:"title"`
			Variables []RawTemplateVar `json:"variables"`
		}
		if json.Unmarshal(body, &in) != nil || in.PanelKey == "" {
			return apiutil.ValidationError("panel_key is required")
		}
		files = BuildScriptedGuiPanel(in.PanelKey, in.Title, in.Variables)
		note = "generated from scripted_gui template"
	case "on-actions", "ai-strategy":
		var in struct {
			PanelKey   string `json:"panel_key"`
			CountryTag string `json:"country_tag"`
		}
		if json.Unmarshal(body, &in) != nil || in.PanelKey == "" {
			return apiutil.ValidationError("panel_key is required")
		}
		if route == "on-actions" {
			files = BuildOnActions(in.PanelKey, in.CountryTag)
			note = "generated from on_actions template"
		} else {
			files = BuildAIStrategy(in.PanelKey, in.CountryTag)
			note = "generated from ai_strategy template"
		}
	case "game-rules":
		var in struct {
			PanelKey string `json:"panel_key"`
			Title    string `json:"title"`
		}
		if json.Unmarshal(body, &in) != nil || in.PanelKey == "" {
			return apiutil.ValidationError("panel_key is required")
		}
		files = BuildGameRules(in.PanelKey, in.Title)
		note = "generated from game_rules template"
	case "gfx":
		var in struct {
			PanelKey string `json:"panel_key"`
		}
		if json.Unmarshal(body, &in) != nil || in.PanelKey == "" {
			return apiutil.ValidationError("panel_key is required")
		}
		files = BuildGfxSprite(in.PanelKey)
		note = "generated from gfx sprite template"
	case "frontendgamesetup":
		var in struct {
			BaseGUIText string `json:"base_gui_text"`
			SlotWidth   *int   `json:"slot_width"`
		}
		if json.Unmarshal(body, &in) != nil || in.BaseGUIText == "" {
			return apiutil.ValidationError("base_gui_text is required")
		}
		width := 540
		if in.SlotWidth != nil {
			if *in.SlotWidth < 240 || *in.SlotWidth > 900 {
				return apiutil.ValidationError("slot_width must be between 240 and 900")
			}
			width = *in.SlotWidth
		}
		var perr string
		files, perr = BuildFrontendGameSetupSlotsize(in.BaseGUIText, width)
		if perr != "" {
			return apiutil.ValidationError(perr)
		}
		note = "patched complete frontendgamesetup slotsize override"
	case "defines", "bop", "music", "peace-conference":
		var in struct {
			Key string `json:"key"`
		}
		if json.Unmarshal(body, &in) != nil || !registryKeyRe.MatchString(in.Key) {
			return apiutil.ValidationError("key must match ^[a-z0-9_]{2,40}$")
		}
		switch route {
		case "defines":
			files, note = BuildDefines(in.Key), "generated from defines template"
		case "bop":
			files, note = BuildBop(in.Key), "generated from bop template"
		case "music":
			files, note = BuildMusic(in.Key), "generated from music template"
		case "peace-conference":
			files, note = BuildPeaceConference(in.Key), "generated from peace_conference template"
		}
	default:
		return apiutil.NotFoundError("Not Found")
	}
	version := supportedVersion(p)
	out, err := upsertTemplateFiles(p, files, note, version)
	if terr, ok := err.(*TombstoneError); ok {
		return tombstoneFailure(terr, "verbatim_override", "", version)
	}
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, out)
	return nil
}

func upsertTemplateFiles(p *project.Project, files []RawTemplateFile, note, version string) ([]RawFile, error) {
	out := []RawFile{}
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		for _, f := range files {
			safe, errMsg := SanitizeDestPath(f.DestPath)
			if errMsg != "" {
				return tombstoneReject("dest_path_invalid", errMsg)
			}
			var existing project.Row
			for _, row := range tx.Rows(table) {
				if str(row, "dest_path") == safe {
					existing = row
					break
				}
			}
			existingGenRP := existing != nil && boolean(existing, "generate_replace_path")
			decision, terr := ValidateRawFileIntent("verbatim_override", safe, f.Content,
				false, existingGenRP, version, "", "")
			if terr != nil {
				return terr
			}
			var stored project.Row
			if existing != nil {
				var err error
				stored, err = tx.Update(table, project.Int64(existing["id"]), func(row project.Row) {
					row["dest_path"] = decision.DestPath
					row["content"] = decision.Content
					row["is_binary"] = false
					row["generate_replace_path"] = decision.GenerateReplacePath
					row["raw_kind"] = decision.RawKind
					row["target_layer"] = optional(decision.TargetLayer)
					row["target_sha256"] = optional(decision.TargetSHA256)
					row["content_binary"] = nil
					row["content_sha256"] = nil
					row["content_obs_key"] = nil
					if str(row, "source") == "imported" {
						row["source"] = "created"
					}
				})
				if err != nil {
					return err
				}
			} else {
				stored = tx.Insert(table, project.Row{
					"dest_path": decision.DestPath, "content": decision.Content, "is_binary": false,
					"generate_replace_path": decision.GenerateReplacePath, "raw_kind": decision.RawKind,
					"target_layer": optional(decision.TargetLayer), "target_sha256": optional(decision.TargetSHA256),
					"note": note, "source": "created",
				})
			}
			out = append(out, rawFileFromRow(stored))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
