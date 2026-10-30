package exportengine

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/hoi4/exportmsg"
)

type vanillaCollisionTable struct {
	Icons     []string `json:"icons"`
	Bookmarks []string `json:"bookmarks"`
}

type vanillaCollisionSets struct {
	icons, bookmarks map[string]bool
}

type vanillaCollisionKey struct{}

func loadVanillaCollisionTable() vanillaCollisionSets {
	sets, err := gameindex.Load(context.Background(), vanillaCollisionKey{}, gameindex.CollisionTable, func(raw []byte) (vanillaCollisionSets, error) {
		var table vanillaCollisionTable
		if err := json.Unmarshal(raw, &table); err != nil {
			return vanillaCollisionSets{}, err
		}
		return vanillaCollisionSets{icons: stringSetOf(table.Icons), bookmarks: stringSetOf(table.Bookmarks)}, nil
	})
	if err != nil {

		return vanillaCollisionSets{}
	}
	return sets
}

func VanillaCollisions(entries []Entry, msg exportmsg.Printer) []string {
	if len(entries) == 0 {
		return nil
	}
	table := loadVanillaCollisionTable()
	vanillaIconPaths, vanillaBookmarkPaths := table.icons, table.bookmarks
	if len(vanillaIconPaths) == 0 && len(vanillaBookmarkPaths) == 0 {
		return nil
	}

	prefix := artifactRootFromEntries(entries)
	seen := map[string]bool{}
	var lines []string
	for _, entry := range entries {
		rel := strings.ReplaceAll(entry.Path, "\\", "/")
		if prefix != "" && strings.HasPrefix(rel, prefix) {
			rel = rel[len(prefix):]
		}
		var line string
		switch low := strings.ToLower(rel); {
		case vanillaIconPaths[low]:
			line = msg.Text("collision.icon", exportmsg.Args{"path": rel})
		case vanillaBookmarkPaths[low]:
			line = msg.Text("collision.bookmark", exportmsg.Args{"path": rel})
		default:
			continue
		}
		if !seen[line] {
			seen[line] = true
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return lines
}

func VanillaCollisionLogEntry(modDir string, lines []string, msg exportmsg.Printer) Entry {
	if modDir == "" || len(lines) == 0 {
		return Entry{}
	}
	body := append(commentLines(msg.Text("collision.log.header", nil)), "")
	body = append(body, lines...)
	return Entry{
		Path:         modDir + "/_VANILLA_COLLISION_LINT.log",
		Bytes:        []byte(strings.Join(body, "\n") + "\n"),
		Origin:       "generated",
		SourceEntity: "vanilla_collision_lint",
	}
}

func artifactRootFromEntries(entries []Entry) string {
	const descriptor = "/descriptor.mod"
	for _, entry := range entries {
		path := strings.ReplaceAll(entry.Path, "\\", "/")
		if strings.HasSuffix(path, descriptor) {
			return path[:len(path)-len(descriptor)+1]
		}
	}
	return ""
}
