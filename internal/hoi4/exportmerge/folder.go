package exportmerge

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ReadFolder(ctx context.Context, dir, prefix string, budget ArchiveBudget) (Archive, error) {
	out := Archive{ByPath: map[string][]byte{}}
	if prefix != "" {
		if _, err := safeArchivePath(prefix); err != nil {
			return out, fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
		}
	}
	seen := map[string]string{}
	var total int64
	err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if name == dir {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		clean, err := safeArchivePath(path.Join(prefix, filepath.ToSlash(rel)))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%w: %q is not a regular file", ErrUnsafeArchive, clean)
		}
		folded := strings.ToLower(clean)
		if prior, ok := seen[folded]; ok {
			return fmt.Errorf("%w: duplicate paths %q and %q", ErrUnsafeArchive, prior, clean)
		}
		seen[folded] = clean
		if len(seen) > budget.MaxEntries {
			return ErrArchiveTooLarge
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > budget.MaxEntryBytes {
			return ErrArchiveTooLarge
		}
		total += info.Size()
		if total > budget.MaxUncompressedBytes {
			return ErrArchiveTooLarge
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		body, err := readBoundedContext(ctx, file, budget.MaxEntryBytes)
		_ = file.Close()
		if err != nil {
			return err
		}
		out.Entries = append(out.Entries, ArchiveEntry{Path: clean, Bytes: body})
		out.ByPath[clean] = body
		return nil
	})
	return out, err
}
