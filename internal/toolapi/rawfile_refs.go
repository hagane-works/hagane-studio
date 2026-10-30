package toolapi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/exportengine"
)

type RawFileReference struct {
	RawFileID  int64  `json:"raw_file_id,omitempty"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Effect     string `json:"effect"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

func rawFileReferencesTo(ctx context.Context, proxy *RESTProxy, projectID int64, targetKey string) ([]RawFileReference, error) {
	if strings.TrimSpace(targetKey) == "" {
		return []RawFileReference{}, nil
	}
	files, err := proxyAuthedCallAny(ctx, proxy, "GET",
		fmt.Sprintf("/api/projects/%d/raw-files", projectID), nil)
	if err != nil {
		return nil, err
	}
	found := []RawFileReference{}
	for _, file := range files {
		if value, _ := file["is_binary"].(bool); value {
			continue
		}
		text, _ := file["content"].(string)
		if strings.TrimSpace(text) == "" {
			continue
		}
		for _, ref := range exportengine.ExtractTextRefs(text) {
			if !strings.EqualFold(ref.TargetID, targetKey) {
				continue
			}
			found = append(found, RawFileReference{
				File: stringField(file, "dest_path"), Line: ref.Line,
				Effect: ref.Effect, TargetType: ref.TargetType, TargetID: ref.TargetID,
			})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].File != found[j].File {
			return found[i].File < found[j].File
		}
		return found[i].Line < found[j].Line
	})
	return found, nil
}
