package rawfiles

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
)

// Limits for one bulk request. A client sends a large mod in several requests.
const (
	maxBulkFiles     = 2000
	maxBulkBodyBytes = 1 << 30
)

// bulkTextExtensions are stored as editable text when the bytes are valid
// UTF-8 and pass the same content gate as a single text raw file; anything
// else, including a text file the gate refuses, is stored byte-for-byte as a
// binary raw file.
var bulkTextExtensions = map[string]bool{
	".txt": true, ".yml": true, ".gfx": true, ".gui": true, ".asset": true, ".lua": true,
	".csv": true, ".shader": true, ".fxh": true, ".fnt": true, ".sfx": true, ".map": true,
}

type bulkResult struct {
	DestPath string `json:"dest_path"`
	Status   string `json:"status"` // created | exists | rejected
	StoredAs string `json:"stored_as,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// createRawBulk adds many raw files in one project write. Writing them one by
// one rewrites the raw file table per file, which does not scale to a mod with
// tens of thousands of art, sound and model files.
//
// Multipart fields: repeated "paths" (mod-relative destination) and "files"
// (the bytes), in the same order. A path that already has a raw file is left
// as it is and reported as "exists", so an interrupted upload can be resent.
func createRawBulk(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBulkBodyBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return apiutil.ValidationError("invalid multipart form")
	}
	defer r.MultipartForm.RemoveAll()
	paths := r.MultipartForm.Value["paths"]
	files := r.MultipartForm.File["files"]
	if len(paths) == 0 || len(paths) != len(files) {
		return apiutil.ValidationError("send one 'paths' field for each 'files' part")
	}
	if len(paths) > maxBulkFiles {
		return apiutil.ValidationError("too many files in one request")
	}
	note := r.FormValue("note")
	version := supportedVersion(p)

	results := make([]bulkResult, len(paths))
	inserts := make([]rawFileInsert, 0, len(paths))
	insertAt := make([]int, 0, len(paths))
	seen := map[string]bool{}
	for i, raw := range paths {
		res := &results[i]
		res.DestPath = raw
		safe, errMsg := SanitizeDestPath(raw)
		if errMsg != "" {
			res.Status, res.Detail = "rejected", errMsg
			continue
		}
		res.DestPath = safe
		if seen[safe] {
			res.Status, res.Detail = "rejected", "the same path is sent twice"
			continue
		}
		seen[safe] = true
		data, err := readBulkPart(files[i])
		if err != nil {
			res.Status, res.Detail = "rejected", err.Error()
			continue
		}
		in, detail, err := bulkInsert(p, safe, data, version)
		if err != nil {
			return err
		}
		if in == nil {
			res.Status, res.Detail = "rejected", detail
			continue
		}
		in.Note = note
		res.StoredAs, res.Detail = "text", detail
		if in.IsBinary {
			res.StoredAs = "binary"
		}
		inserts = append(inserts, *in)
		insertAt = append(insertAt, i)
	}

	created := 0
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		taken := map[string]bool{}
		for _, row := range tx.Rows(table) {
			taken[str(row, "dest_path")] = true
		}
		for n, in := range inserts {
			res := &results[insertAt[n]]
			if taken[in.DestPath] {
				res.Status, res.StoredAs = "exists", ""
				continue
			}
			taken[in.DestPath] = true
			tx.Insert(table, project.Row{
				"dest_path": in.DestPath, "content": in.Content, "is_binary": in.IsBinary,
				"generate_replace_path": false, "raw_kind": in.RawKind,
				"target_layer": nil, "target_sha256": nil,
				"note": in.Note, "content_binary": nil,
				"content_sha256": optional(in.ContentSHA256), "content_obs_key": optional(in.ContentOBSKey),
				"source": "created",
			})
			res.Status = "created"
			created++
		}
		return nil
	})
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"created": created, "results": results})
	return nil
}

var (
	errBulkRead     = errors.New("failed to read upload")
	errBulkTooLarge = errors.New("file is larger than the raw binary cap")
)

func readBulkPart(header *multipart.FileHeader) ([]byte, error) {
	file, err := header.Open()
	if err != nil {
		return nil, errBulkRead
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxRawBinaryBytes+1))
	if err != nil {
		return nil, errBulkRead
	}
	if len(data) > maxRawBinaryBytes {
		return nil, errBulkTooLarge
	}
	return data, nil
}

// bulkInsert decides how one file is stored. A nil insert with a detail
// means the file is refused.
func bulkInsert(p *project.Project, safe string, data []byte, version string) (*rawFileInsert, string, error) {
	detail := ""
	if bulkTextExtensions[strings.ToLower(path.Ext(safe))] && utf8.Valid(data) && !bytes.Contains(data, []byte{0}) {
		decision, terr := ValidateRawFileIntent("verbatim_override", safe, string(data), false, false, version, "", "")
		if terr == nil {
			return &rawFileInsert{DestPath: decision.DestPath, Content: decision.Content, RawKind: decision.RawKind}, "", nil
		}
		detail = "kept byte-for-byte as binary: " + terr.Message
	}
	decision, terr := ValidateRawFileIntent("verbatim_override", safe, "", true, false, version, "", "")
	if terr != nil {
		return nil, terr.Message, nil
	}
	sha, key, err := storeRawBlob(p, data)
	if err != nil {
		return nil, "", err
	}
	return &rawFileInsert{
		DestPath: decision.DestPath, IsBinary: true, RawKind: decision.RawKind,
		ContentSHA256: &sha, ContentOBSKey: &key,
	}, detail, nil
}
