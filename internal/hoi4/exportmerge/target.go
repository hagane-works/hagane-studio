package exportmerge

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/importsvc/characters"
	eventsvc "haganeworks/internal/hoi4/importsvc/events"
	"haganeworks/internal/hoi4/importsvc/ideas"
	treesvc "haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
)

var (
	ErrArchiveTooLarge = errors.New("base mod archive exceeds safety budget")
	ErrUnsafeArchive   = errors.New("base mod archive is unsafe")
	ErrInvalidArchive  = errors.New("base mod archive is not a valid zip")
)

type ArchiveBudget struct {
	MaxCompressedBytes   int64
	MaxEntries           int
	MaxEntryBytes        int64
	MaxUncompressedBytes int64
	MaxCompressionRatio  uint64
}

func DefaultArchiveBudget() ArchiveBudget {
	return ArchiveBudget{
		MaxCompressedBytes: 512 << 20, MaxEntries: 20_000,
		MaxEntryBytes: 256 << 20, MaxUncompressedBytes: 1 << 30,
		MaxCompressionRatio: 200,
	}
}

type ArchiveEntry struct {
	Path  string
	Bytes []byte
}

type Archive struct {
	Entries  []ArchiveEntry
	ByPath   map[string][]byte
	Warnings []string
}

type TargetMod struct {
	focusSources             map[string]*FocusSourceFile
	Characters               map[string]map[string]any
	Events                   map[string]map[string]any
	Decisions                map[string]map[string]any
	FocusNodes               map[string]map[string]any
	Ideas                    map[string]map[string]any
	Warnings                 []string
	SourceFiles              map[string]string
	EventNamespaces          map[string]string
	DecisionCategoriesInFile map[string][]string
}

func NewTargetMod() *TargetMod {
	return &TargetMod{
		focusSources: map[string]*FocusSourceFile{},
		Characters:   map[string]map[string]any{}, Events: map[string]map[string]any{},
		Decisions: map[string]map[string]any{}, FocusNodes: map[string]map[string]any{},
		Ideas: map[string]map[string]any{}, SourceFiles: map[string]string{},
		EventNamespaces: map[string]string{}, DecisionCategoriesInFile: map[string][]string{},
	}
}

func (t *TargetMod) ForType(kind string) map[string]map[string]any {
	switch kind {
	case "character":
		return t.Characters
	case "event":
		return t.Events
	case "decision":
		return t.Decisions
	case "focus_node":
		return t.FocusNodes
	case "idea":
		return t.Ideas
	default:
		return nil
	}
}

func entityKey(kind, id string) string { return kind + "/" + id }

func ReadArchive(ctx context.Context, filename string, budget ArchiveBudget) (Archive, error) {
	var out Archive
	out.ByPath = map[string][]byte{}
	info, err := os.Stat(filename)
	if err != nil {
		return out, err
	}
	if info.Size() <= 0 {
		return out, ErrInvalidArchive
	}
	if info.Size() > budget.MaxCompressedBytes {
		return out, ErrArchiveTooLarge
	}
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return out, ErrInvalidArchive
	}
	defer zr.Close()
	if len(zr.File) > budget.MaxEntries {
		return out, ErrArchiveTooLarge
	}
	seen := map[string]string{}
	var total uint64
	for _, file := range zr.File {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		name := strings.TrimSuffix(file.Name, "/")
		if name == "" {
			continue
		}
		clean, err := safeArchivePath(name)
		if err != nil {
			return out, fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return out, fmt.Errorf("%w: symlink %q", ErrUnsafeArchive, clean)
		}
		folded := strings.ToLower(clean)
		if prior, ok := seen[folded]; ok {
			return out, fmt.Errorf("%w: duplicate paths %q and %q", ErrUnsafeArchive, prior, clean)
		}
		seen[folded] = clean
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > uint64(budget.MaxEntryBytes) {
			return out, ErrArchiveTooLarge
		}
		total += file.UncompressedSize64
		if total > uint64(budget.MaxUncompressedBytes) {
			return out, ErrArchiveTooLarge
		}
		if file.UncompressedSize64 > 1<<20 {
			if file.CompressedSize64 == 0 || file.UncompressedSize64/file.CompressedSize64 > budget.MaxCompressionRatio {
				return out, fmt.Errorf("%w: suspicious compression ratio for %q", ErrArchiveTooLarge, clean)
			}
		}
		rc, err := file.Open()
		if err != nil {
			return out, ErrInvalidArchive
		}
		body, err := readBoundedContext(ctx, rc, budget.MaxEntryBytes)
		_ = rc.Close()
		if err != nil {
			return out, err
		}
		out.Entries = append(out.Entries, ArchiveEntry{Path: clean, Bytes: body})
		out.ByPath[clean] = body
	}
	return out, nil
}

func safeArchivePath(name string) (string, error) {
	if strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	return clean, nil
}

func readBoundedContext(ctx context.Context, reader io.Reader, max int64) ([]byte, error) {
	var buf bytes.Buffer
	chunk := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := reader.Read(chunk)
		if n > 0 {
			total += int64(n)
			if total > max {
				return nil, ErrArchiveTooLarge
			}
			_, _ = buf.Write(chunk[:n])
		}
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func ParseTarget(ctx context.Context, archive Archive) *TargetMod {
	target := NewTargetMod()
	for _, entry := range archive.Entries {
		if ctx.Err() != nil {
			break
		}
		kind := classifyTargetPath(entry.Path)
		if kind == "" {
			continue
		}
		text, warning := decodeModText(entry.Bytes)
		if warning != "" {
			target.Warnings = append(target.Warnings, entry.Path+" · "+warning)
		}
		if err := ingestTarget(target, kind, text, entry.Path); err != nil {
			target.Warnings = append(target.Warnings, fmt.Sprintf("%s · %s parse failed: %v", entry.Path, kind, err))
		}
	}
	return target
}

func classifyTargetPath(name string) string {
	n := "/" + strings.ToLower(strings.TrimLeft(strings.ReplaceAll(name, "\\", "/"), "/"))
	if !strings.HasSuffix(n, ".txt") {
		return ""
	}
	switch {
	case strings.Contains(n, "/common/national_focus/"):
		return "focus_node"
	case strings.Contains(n, "/common/characters/"):
		return "character"
	case strings.Contains(n, "/common/ideas/"):
		return "idea"
	case strings.Contains(n, "/common/decisions/") && !strings.Contains(n, "/common/decisions/categories/"):
		return "decision"
	case strings.Contains(n, "/events/"):
		return "event"
	default:
		return ""
	}
}

func decodeModText(raw []byte) (string, string) {
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) && utf8.Valid(raw[3:]) {
		return string(raw[3:]), ""
	}
	if utf8.Valid(raw) {
		return string(raw), ""
	}
	if decoded, _, err := transform.Bytes(charmap.Windows1252.NewDecoder(), raw); err == nil {
		return string(decoded), ""
	}
	if decoded, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), raw); err == nil {
		return string(decoded), ""
	}
	return strings.ToValidUTF8(string(raw), "�"), "could not decode cleanly; used lossy replacement"
}

func ingestTarget(target *TargetMod, kind, text, source string) error {
	switch kind {
	case "character":
		rows, _, err := characters.ImportCharacters(text)
		if err != nil {
			return err
		}
		for _, row := range characters.NativeEntities(rows) {
			if id := stringField(row, "tag"); id != "" {
				target.Characters[id] = row
				target.SourceFiles[entityKey(kind, id)] = source
			}
		}
	case "idea":
		rows, _, err := ideas.ImportIdeas(text)
		if err != nil {
			return err
		}
		for _, row := range ideas.NativeEntities(rows) {
			if id := stringField(row, "idea_id"); id != "" {
				target.Ideas[id] = row
				target.SourceFiles[entityKey(kind, id)] = source
			}
		}
	case "event":
		namespaces, rows, _, err := eventsvc.ImportEvents(text)
		if err != nil {
			return err
		}
		ns := ""
		if len(namespaces) > 0 {
			ns = namespaces[0]
		}
		for _, raw := range rows {
			row := nativeValue(raw).(map[string]any)
			if id := stringField(row, "event_id"); id != "" {
				target.Events[id] = row
				target.SourceFiles[entityKey(kind, id)] = source
				if ns != "" {
					target.EventNamespaces[id] = ns
				}
			}
		}
	case "decision":
		categories, rows, _, err := eventsvc.ImportDecisions(text)
		if err != nil {
			return err
		}
		for _, raw := range categories {
			row := nativeValue(raw).(map[string]any)
			if id := stringField(row, "category_id"); id != "" {
				target.DecisionCategoriesInFile[source] = append(target.DecisionCategoriesInFile[source], id)
			}
		}
		for _, raw := range rows {
			row := nativeValue(raw).(map[string]any)
			if id := stringField(row, "decision_id"); id != "" {
				target.Decisions[id] = row
				target.SourceFiles[entityKey(kind, id)] = source
			}
		}
	case "focus_node":
		file, err := ParseFocusSourceFile(text, source)
		if err != nil {
			return err
		}
		if target.focusSources == nil {
			target.focusSources = map[string]*FocusSourceFile{}
		}
		target.focusSources[source] = file
		target.Warnings = append(target.Warnings, file.Warnings...)
		for _, row := range file.Rows() {
			if id := stringField(row, "focus_id"); id != "" {
				target.FocusNodes[id] = row
				target.SourceFiles[entityKey(kind, id)] = source
			}
		}
	}
	return nil
}

func stringField(row map[string]any, key string) string { value, _ := row[key].(string); return value }

func nativeValue(value any) any {
	switch value := value.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return value
	case *big.Int:
		if value.IsInt64() {
			return value.Int64()
		}
		return value.String()
	case paradox.Scalar:
		switch value.Kind {
		case paradox.ScalarInt:
			return nativeValue(value.Int)
		case paradox.ScalarFloat:
			return value.Float
		default:
			return value.Str
		}
	case *importsvc.RawExtras:
		out := map[string]any{}
		for _, k := range value.Keys {
			v, _ := value.Get(k)
			out[k] = nativeValue(v)
		}
		return out
	case *treesvc.RawDict:
		out := map[string]any{}
		for _, k := range value.Keys {
			v, _ := value.Get(k)
			out[k] = nativeValue(v)
		}
		return out
	case importsvc.ScopeVerdict:
		kind, tag, shared := value.StorageTuple()
		return map[string]any{"kind": kind, "country_tag": tag, "shared_country_tags": shared}
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[k] = nativeValue(v)
		}
		return out
	case []map[string]any:
		out := make([]any, len(value))
		for i, v := range value {
			out[i] = nativeValue(v)
		}
		return out
	case []string:
		out := make([]any, len(value))
		for i, v := range value {
			out[i] = v
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, v := range value {
			out[i] = nativeValue(v)
		}
		return out
	default:
		raw, err := jsonMarshalCanonical(value)
		if err == nil {
			var out any
			if decodeJSON(raw, &out) == nil {
				return out
			}
		}
		return fmt.Sprint(value)
	}
}

func jsonMarshalCanonical(value any) ([]byte, error) { return json.Marshal(value) }

func ArchiveEntries(archive Archive, skipManifest bool) []exportengine.Entry {
	out := make([]exportengine.Entry, 0, len(archive.Entries))
	for _, entry := range archive.Entries {
		if skipManifest && strings.EqualFold(path.Base(entry.Path), exportengine.ArtifactManifestBasename) {
			continue
		}
		out = append(out, exportengine.Entry{Path: entry.Path, Bytes: entry.Bytes, Origin: "base_mod_preserved", SourceEntity: "base_zip_entry"})
	}
	return out
}
