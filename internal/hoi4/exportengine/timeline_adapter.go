package exportengine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportplan/timeline"
)

var rawBookmarkDateRE = regexp.MustCompile(`(?m)^\s*date\s*=\s*([0-9]{1,4}(?:\.[0-9]{1,2}){2,3})`)

func BuildNativeTimelinePlan(snapshot Snapshot, requestedReplacePaths []string, includeImported bool, resolve BlobResolver) (timeline.Plan, error) {
	rawEntries, rawReplacePaths, err := nativeInlineRawFiles(snapshot, includeImported, resolve)
	if err != nil {
		return timeline.Plan{}, err
	}
	return BuildTimelinePlanFromSnapshot(snapshot, requestedReplacePaths, rawReplacePaths, rawEntries, includeImported)
}

func BuildTimelinePlanFromSnapshot(snapshot Snapshot, requestedReplacePaths, rawReplacePaths []string, rawEntries []Entry, includeImported bool) (timeline.Plan, error) {
	var project struct {
		Name         string            `json:"name"`
		CountryTag   string            `json:"country_tag"`
		StartDate    string            `json:"start_date"`
		EndDate      string            `json:"end_date"`
		Dependencies []json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(snapshot.Project, &project); err != nil {
		return timeline.Plan{}, fmt.Errorf("decode project timeline: %w", err)
	}
	fallback := "mod_" + strings.ToUpper(strings.TrimSpace(project.CountryTag))
	if strings.TrimSpace(project.CountryTag) == "" {
		fallback = "mod_" + strconv.FormatInt(snapshot.ProjectID, 10)
	}
	modDir := SanitizeModDirName(project.Name, fallback)

	structured, err := timelineStructuredBookmarks(snapshot, includeImported)
	if err != nil {
		return timeline.Plan{}, err
	}
	rawBookmarks, rawDefines := timelineRawEvidence(rawEntries, modDir)
	requested := append([]string(nil), requestedReplacePaths...)
	requested = append(requested, rawReplacePaths...)
	ideologyPlan, err := nativeIdeologyExportPlan(snapshot, rawEntries)
	if err != nil {
		return timeline.Plan{}, err
	}
	if ideologyPlan.Facts.RequiresReplacePath() {
		requested = append(requested, "common/ideologies")
	}
	return timeline.Build(timeline.Input{
		ProjectStartDate:      project.StartDate,
		RequestedEndDate:      project.EndDate,
		HasDependencies:       len(project.Dependencies) > 0,
		StructuredBookmarks:   structured,
		RawBookmarks:          rawBookmarks,
		RawDefines:            rawDefines,
		RequestedReplacePaths: requested,
		InheritedReplacePaths: InheritedReplacePaths(snapshot),
		ModDirectory:          modDir,
		DefinesPath:           timeline.DefinesPathForModDirectory(modDir),
	}), nil
}

func timelineStructuredBookmarks(snapshot Snapshot, includeImported bool) ([]timeline.BookmarkEvidence, error) {
	type bookmarkRow struct {
		ID         int64  `json:"id"`
		BookmarkID string `json:"bookmark_id"`
		Date       string `json:"date"`
		Source     string `json:"source"`
	}
	items := make([]timeline.BookmarkEvidence, 0, len(snapshot.Tables["bookmarks"]))
	for _, raw := range snapshot.Tables["bookmarks"] {
		var row bookmarkRow
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode timeline bookmark: %w", err)
		}
		if !includeImported && row.Source == "imported" {
			continue
		}
		id := strings.TrimSpace(row.BookmarkID)
		if id == "" {
			id = strconv.FormatInt(row.ID, 10)
		}
		items = append(items, timeline.BookmarkEvidence{
			ID: id, Path: timeline.BookmarksPath + "/" + id + ".txt",
			Source: row.Source, DateTokens: []string{row.Date}, Emitted: true,
		})
	}
	return items, nil
}

func timelineRawEvidence(rawEntries []Entry, modDir string) ([]timeline.BookmarkEvidence, []timeline.RawDefinesEvidence) {
	prefix := strings.Trim(modDir, "/") + "/"
	bookmarks := make([]timeline.BookmarkEvidence, 0)
	defines := make([]timeline.RawDefinesEvidence, 0)
	for _, entry := range rawEntries {
		path := strings.TrimPrefix(strings.ReplaceAll(entry.Path, "\\", "/"), prefix)
		if strings.HasPrefix(path, timeline.BookmarksPath+"/") && strings.HasSuffix(strings.ToLower(path), ".txt") {
			matches := rawBookmarkDateRE.FindAllStringSubmatch(string(entry.Bytes), -1)
			dates := make([]string, 0, len(matches))
			for _, match := range matches {
				if len(match) == 2 {
					dates = append(dates, match[1])
				}
			}
			bookmarks = append(bookmarks, timeline.BookmarkEvidence{
				ID: entry.SourceEntity, Path: path, Source: entry.Origin,
				DateTokens: dates, Emitted: true,
			})
		}
		lowerPath := strings.ToLower(path)
		if strings.HasPrefix(lowerPath, "common/defines/") && strings.HasSuffix(lowerPath, ".lua") {
			defines = append(defines, timeline.RawDefinesEvidence{
				ID: entry.SourceEntity, Path: path, Source: entry.Origin,
				Content: string(entry.Bytes), Emitted: true,
			})
		}
	}
	return bookmarks, defines
}

func nativeIdeologyExportPlan(snapshot Snapshot, rawEntries []Entry) (worldout.IdeologyExportPlan, error) {
	_, manifest, _, inheritedSourceActive, err := decodeIdeologySourceProject(snapshot.Project)
	if err != nil && !staleIdeologySourceLock(err) {
		return worldout.IdeologyExportPlan{}, err
	}

	if inheritedSourceActive || err != nil {
		project := worldout.NewOMap()
		if decodeErr := project.UnmarshalJSON(snapshot.Project); decodeErr != nil {
			return worldout.IdeologyExportPlan{}, fmt.Errorf("decode world project: %w", decodeErr)
		}
		dependencies, _ := project.Get("dependencies")
		dependencyList, _ := dependencies.([]worldout.Value)
		sourceRoots := map[string]bool{}
		for _, entry := range manifest.Entries {
			if id := strings.ToLower(strings.TrimSpace(entry.ParentRootID)); id != "" {
				sourceRoots[id] = true
			}
		}
		return worldout.IdeologyExportPlan{
			Facts: worldout.IdeologyExportFacts{
				Standalone:  len(dependencyList) == 0,
				RawOverride: rawEntriesOwnDirectory(rawEntries, "common/ideologies"),
			},
			InheritedSource: true,
			SourceRoots:     sourceRoots,
		}, nil
	}
	project := worldout.NewOMap()
	if err := project.UnmarshalJSON(snapshot.Project); err != nil {
		return worldout.IdeologyExportPlan{}, fmt.Errorf("decode world project: %w", err)
	}
	tables, err := worldout.LoadVanillaTables(context.Background())
	if err != nil {
		return worldout.IdeologyExportPlan{}, fmt.Errorf("load the base game's world tables: %w", err)
	}
	dependencies, _ := project.Get("dependencies")
	dependencyList, _ := dependencies.([]worldout.Value)
	return worldout.BuildIdeologyExportPlan(
		tables, project, len(dependencyList) > 0,
		rawEntriesOwnDirectory(rawEntries, "common/ideologies"),
	), nil
}
