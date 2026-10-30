package ideologysource

import (
	"bytes"
	"fmt"
)

var (
	managedBegin       = []byte("# Hagane Works managed subtypes BEGIN")
	managedEnd         = []byte("# Hagane Works managed subtypes END")
	legacyManagedBegin = []byte("# HOI4 Hagane managed subtypes BEGIN")
	legacyManagedEnd   = []byte("# HOI4 Hagane managed subtypes END")
)

func stripManagedRegions(source []byte) ([]byte, int, error) {
	working, currentCount, err := stripManagedRegionsWithMarkers(source, managedBegin, managedEnd)
	if err != nil {
		return nil, currentCount, err
	}
	working, legacyCount, err := stripManagedRegionsWithMarkers(working, legacyManagedBegin, legacyManagedEnd)
	return working, currentCount + legacyCount, err
}

func stripManagedRegionsWithMarkers(source, beginMarker, endMarker []byte) ([]byte, int, error) {
	working := append([]byte(nil), source...)
	count := 0
	for {
		begin := bytes.Index(working, beginMarker)
		endOnly := bytes.Index(working, endMarker)
		if begin < 0 {
			if endOnly >= 0 {
				return nil, count, fmt.Errorf("managed END marker has no BEGIN")
			}
			return working, count, nil
		}
		if endOnly >= 0 && endOnly < begin {
			return nil, count, fmt.Errorf("managed END marker precedes BEGIN")
		}
		lineStart := begin
		for lineStart > 0 && working[lineStart-1] != '\n' {
			lineStart--
		}
		for i := lineStart; i < begin; i++ {
			if working[i] != ' ' && working[i] != '\t' {
				return nil, count, fmt.Errorf("managed BEGIN marker is not a standalone comment")
			}
		}
		removeStart := lineStart
		if removeStart > 0 && working[removeStart-1] == '\n' {
			removeStart--
			if removeStart > 0 && working[removeStart-1] == '\r' {
				removeStart--
			}
		}
		end := bytes.Index(working[begin+len(beginMarker):], endMarker)
		if end < 0 {
			return nil, count, fmt.Errorf("managed BEGIN marker has no END")
		}
		end += begin + len(beginMarker)
		endLineStart := end
		for endLineStart > 0 && working[endLineStart-1] != '\n' {
			endLineStart--
		}
		for i := endLineStart; i < end; i++ {
			if working[i] != ' ' && working[i] != '\t' {
				return nil, count, fmt.Errorf("managed END marker is not a standalone comment")
			}
		}
		removeEnd := end + len(endMarker)
		for removeEnd < len(working) && working[removeEnd] != '\n' {
			if working[removeEnd] != '\r' && working[removeEnd] != ' ' && working[removeEnd] != '\t' {
				return nil, count, fmt.Errorf("content follows managed END marker")
			}
			removeEnd++
		}
		if removeEnd < len(working) && working[removeEnd] == '\n' {
			removeEnd++
		}
		for removeEnd < len(working) && (working[removeEnd] == ' ' || working[removeEnd] == '\t') {
			removeEnd++
		}
		if removeEnd >= len(working) || working[removeEnd] != '}' {
			return nil, count, fmt.Errorf("managed region is not immediately before a types closing brace")
		}
		working = append(working[:removeStart], working[removeEnd:]...)
		count++
	}
}

func stripLayers(layers []Layer) ([]Layer, map[string]int, error) {
	result := make([]Layer, len(layers))
	regions := map[string]int{}
	for i, layer := range layers {
		result[i] = layer
		result[i].ReplacePaths = append([]string(nil), layer.ReplacePaths...)
		result[i].Files = make([]SourceFile, len(layer.Files))
		for j, file := range layer.Files {
			stripped, count, err := stripManagedRegions(file.Bytes)
			if err != nil {
				return nil, nil, findingsError(Finding{Code: "ideology_source_managed_region_malformed", OwnerPath: normalizePath(file.Path), Detail: err.Error()})
			}
			result[i].Files[j] = SourceFile{Path: file.Path, Bytes: stripped}
			regions[layerKind(layer.SourceKind)+"\x00"+stringsKey(layer.DependencyKey)+"\x00"+stringsKey(file.Path)] = count
		}
	}
	return result, regions, nil
}

func stringsKey(value string) string {
	path := normalizePath(value)
	if path != "" {
		return bytesToLower(path)
	}
	return bytesToLower(value)
}

func bytesToLower(value string) string {
	b := []byte(value)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
