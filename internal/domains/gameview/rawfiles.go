package gameview

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/clausewitz"
	"haganeworks/internal/project"
)

var ErrRawFileMissing = errors.New("raw file missing")

var ErrRawFileConflict = errors.New("raw file dest_path conflict")

type RawFile struct {
	ID                  int64   `json:"id"`
	ProjectID           int64   `json:"project_id"`
	DestPath            string  `json:"dest_path"`
	Content             string  `json:"content"`
	IsBinary            bool    `json:"is_binary"`
	GenerateReplacePath bool    `json:"generate_replace_path"`
	RawKind             string  `json:"raw_kind"`
	TargetLayer         *string `json:"target_layer"`
	TargetSHA256        *string `json:"target_sha256"`
	Note                string  `json:"note"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	ContentSHA256       *string `json:"-"`
	ContentOBSKey       *string `json:"-"`
	Source              string  `json:"-"`
}

type RawFileInsert struct {
	DestPath            string
	Content             string
	IsBinary            bool
	GenerateReplacePath bool
	RawKind             string
	TargetLayer         *string
	TargetSHA256        *string
	Note                string

	ContentSHA256 *string
	ContentOBSKey *string

	Source string
}

type RawFileIntent struct {
	RawKind             string
	DestPath            string
	Content             string
	GenerateReplacePath bool
	TargetLayer         *string
	TargetSHA256        *string
}

type RawFiles struct{ tx *project.Tx }

func NewRawFiles(tx *project.Tx) *RawFiles { return &RawFiles{tx: tx} }

func optionalString(v any) *string {
	if v == nil {
		return nil
	}
	s := str(v)
	return &s
}

func rawFileFromRow(row project.Row) RawFile {
	kind := str(row["raw_kind"])
	if kind == "" {
		kind = "verbatim_override"
	}
	isBinary, _ := row["is_binary"].(bool)
	replace, _ := row["generate_replace_path"].(bool)
	return RawFile{
		ID: project.Int64(row["id"]), ProjectID: project.Int64(row["project_id"]),
		DestPath: str(row["dest_path"]), Content: str(row["content"]), IsBinary: isBinary,
		GenerateReplacePath: replace, RawKind: kind,
		TargetLayer: optionalString(row["target_layer"]), TargetSHA256: optionalString(row["target_sha256"]),
		Note: str(row["note"]), CreatedAt: str(row["created_at"]), UpdatedAt: str(row["updated_at"]),
		ContentSHA256: optionalString(row["content_sha256"]), ContentOBSKey: optionalString(row["content_obs_key"]),
		Source: str(row["source"]),
	}
}

func optionalValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (r *RawFiles) List() []RawFile {
	rows := r.tx.Rows("project_raw_files")
	out := make([]RawFile, 0, len(rows))
	for _, row := range rows {
		out = append(out, rawFileFromRow(row))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DestPath < out[j].DestPath })
	return out
}

func (r *RawFiles) Create(in RawFileInsert) (RawFile, error) {
	if in.IsBinary && (in.ContentSHA256 == nil || strings.TrimSpace(*in.ContentSHA256) == "" ||
		in.ContentOBSKey == nil || strings.TrimSpace(*in.ContentOBSKey) == "") {
		return RawFile{}, errors.New("binary raw files require a content pointer")
	}
	for _, row := range r.tx.Rows("project_raw_files") {
		if str(row["dest_path"]) == in.DestPath {
			return RawFile{}, ErrRawFileConflict
		}
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "created"
	}
	kind := in.RawKind
	if kind == "" {
		kind = "verbatim_override"
	}
	row := r.tx.Insert("project_raw_files", project.Row{
		"dest_path": in.DestPath, "content": in.Content, "is_binary": in.IsBinary,
		"generate_replace_path": in.GenerateReplacePath, "raw_kind": kind,
		"target_layer": optionalValue(in.TargetLayer), "target_sha256": optionalValue(in.TargetSHA256),
		"note": in.Note, "content_binary": nil,
		"content_sha256": optionalValue(in.ContentSHA256), "content_obs_key": optionalValue(in.ContentOBSKey),
		"source": source,
	})
	return rawFileFromRow(row), nil
}

func (r *RawFiles) Update(id int64, decision RawFileIntent, note *string) (RawFile, error) {
	for _, row := range r.tx.Rows("project_raw_files") {
		if project.Int64(row["id"]) != id && str(row["dest_path"]) == decision.DestPath {
			return RawFile{}, ErrRawFileConflict
		}
	}
	row, err := r.tx.Update("project_raw_files", id, func(row project.Row) {
		row["dest_path"], row["content"] = decision.DestPath, decision.Content
		row["generate_replace_path"], row["raw_kind"] = decision.GenerateReplacePath, decision.RawKind
		row["target_layer"], row["target_sha256"] = optionalValue(decision.TargetLayer), optionalValue(decision.TargetSHA256)
		if note != nil {
			row["note"] = *note
		}
		if str(row["source"]) == "imported" {
			row["source"] = "created"
		}
	})
	if errors.Is(err, project.ErrNotFound) {
		return RawFile{}, ErrRawFileMissing
	}
	if err != nil {
		return RawFile{}, err
	}
	return rawFileFromRow(row), nil
}

func (r *RawFiles) Delete(id int64) error {
	if r.tx.Delete("project_raw_files", func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
		return ErrRawFileMissing
	}
	return nil
}

func rawFileAt(files []RawFile, dest string) (RawFile, bool) {
	for _, file := range files {
		if strings.EqualFold(file.DestPath, dest) {
			return file, true
		}
	}
	return RawFile{}, false
}

func SanitizeDestPath(raw string) (string, string) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if s == "" {
		return "", "dest_path is required (e.g. 'common/on_actions/TNO_Reich_on_actions.txt')"
	}
	if len(s) >= 2 && s[1] == ':' {
		return "", "dest_path must be mod-relative (no drive letter)"
	}
	s = strings.TrimLeft(s, "/")
	var parts []string
	for _, seg := range strings.Split(s, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", "dest_path may not contain '..' (path traversal)"
		}
		parts = append(parts, seg)
	}
	if len(parts) == 0 {
		return "", "dest_path is empty after normalisation"
	}
	if !strings.Contains(parts[len(parts)-1], ".") {
		return "", "dest_path must point at a file (e.g. common/on_actions/TNO_Reich_on_actions.txt)"
	}
	return strings.Join(parts, "/"), ""
}

func RejectReservedDest(joined string) string {
	slashed := strings.ReplaceAll(joined, "\\", "/")
	base := slashed
	if slash := strings.LastIndex(slashed, "/"); slash >= 0 {
		base = slashed[slash+1:]
	}
	if strings.EqualFold(base, "_HAGANE_EXPORT_FINDINGS.json") {
		return "_HAGANE_EXPORT_FINDINGS.json is reserved: the export pipeline writes " +
			"this diagnostic record after inspecting the final artifact"
	}
	if strings.EqualFold(base, "_REFERENCE_INTEGRITY.log") {
		return "_REFERENCE_INTEGRITY.log is reserved: the export pipeline writes " +
			"this relationship review after inspecting the final artifact"
	}
	switch strings.ToLower(joined) {
	case "descriptor.mod":
		return "descriptor.mod 是保留文件:导出时由钢之工坊根据「Mod 描述符」设置自动生成," +
			"不能作为 raw 覆盖文件创建。如需修改 mod 名称/版本/依赖,请到项目的导出设置里改。"
	case "thumbnail.png":
		return "thumbnail.png is reserved: upload the Workshop cover from Mod Descriptor settings"
	}
	return ""
}

func GateRawContent(destPath, content string, isBinary bool) error {
	lower := strings.ToLower(strings.TrimSpace(destPath))
	if isBinary || content == "" ||
		!(strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".gui") || strings.HasSuffix(lower, ".gfx")) {
		return nil
	}
	text := []rune(content)
	var open []int
	var problems []string
	stray := 0
	inComment, quoteStart := false, -1
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case inComment:
			inComment = c != '\n'
		case quoteStart >= 0:
			if c == '\\' && i+1 < len(text) {
				i++
			} else if c == '"' {
				quoteStart = -1
			}
		case c == '#':
			inComment = true
		case c == '"':
			quoteStart = i
		case c == '{':
			open = append(open, i)
		case c == '}':
			if len(open) > 0 {
				open = open[:len(open)-1]
			} else {
				stray++
				problems = append(problems, rawProblem(text, "clausewitz.stray_closing_brace", i))
			}
		}
	}
	if quoteStart >= 0 {
		problems = append(problems, rawProblem(text, "clausewitz.unterminated_quote", quoteStart))
	}
	for k := len(open) - 1; k >= 0; k-- {
		problems = append(problems, rawProblem(text, "clausewitz.unclosed_block", open[k]))
	}
	if len(problems) == 0 {
		return nil
	}
	if len(problems) > 5 {
		problems = problems[:5]
	}
	return fmt.Errorf("raw file content does not close its braces (net %+d); HOI4 would nest "+
		"every later block of this file into the wrong scope: %s", len(open)-stray, strings.Join(problems, "; "))
}

func rawProblem(text []rune, code string, at int) string {
	span := clausewitz.Span(text, at, min(len(text), at+1))
	return fmt.Sprintf("%s @ line %d:%d", code, span.StartLine, span.StartColumn)
}
