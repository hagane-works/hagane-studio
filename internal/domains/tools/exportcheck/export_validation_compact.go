package exportcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const validationPageBudget = 64 << 10

type validationPageRequest struct {
	offset, limit int
	revision      string
	issueIndex    *int
	detailPath    string
}

type validationPageError struct {
	status int
	code   string
	detail string
}

func parseValidationPage(q url.Values) (validationPageRequest, *validationPageError) {
	p := validationPageRequest{limit: 20, revision: q.Get("revision"), detailPath: q.Get("detail_path")}
	bad := func(detail string) (validationPageRequest, *validationPageError) {
		return p, &validationPageError{422, "invalid_validation_page", detail}
	}
	for key, target := range map[string]*int{"offset": &p.offset, "limit": &p.limit} {
		if raw := q.Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				return bad(key + " must be an integer")
			}
			*target = n
		}
	}
	if p.offset < 0 || p.limit < 1 || p.limit > 100 {
		return bad("offset must be nonnegative and limit must be 1..100")
	}
	if raw, present := q["issue_index"]; present {
		if len(raw) != 1 {
			return bad("issue_index must be one nonnegative integer")
		}
		n, err := strconv.Atoi(raw[0])
		if err != nil || n < 0 {
			return bad("issue_index must be one nonnegative integer")
		}
		p.issueIndex = &n
	}
	if p.offset > 0 || p.issueIndex != nil {
		if p.revision == "" {
			return bad("continuation and issue detail require the previous revision")
		}
	}
	if len(p.revision) > 128 || len(p.detailPath) > 512 || (p.detailPath != "" && (p.issueIndex == nil || !strings.HasPrefix(p.detailPath, "/"))) {
		return bad("detail_path must be a bounded JSON pointer with issue_index")
	}
	return p, nil
}

func validationLevel(issue ExportIssue) string {
	switch issue["severity"] {
	case "blocker":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

func validationPreview(value any, maxBytes int) (any, bool) {
	switch v := value.(type) {
	case string:
		if len(v) <= maxBytes {
			return v, false
		}
		end := maxBytes
		for end > 0 && !utf8.RuneStart(v[end]) {
			end--
		}
		return v[:end] + "…", true
	case json.Number:
		if len(v.String()) > maxBytes {
			return nil, true
		}
		return v, false
	case nil, bool, int, int32, int64, uint, uint64, float32, float64:
		return v, false
	default:
		return nil, true
	}
}

func validationIssuePreview(issue ExportIssue, index int) map[string]any {
	out := map[string]any{"issue_index": index, "level": validationLevel(issue), "detail_path": "", "details_omitted": true}
	for _, key := range []string{"code", "severity", "message"} {
		if value, ok := issue[key]; ok {
			limit := 128
			if key == "message" {
				limit = 512
			}
			preview, truncated := validationPreview(value, limit)
			if preview != nil {
				out[key] = preview
			}
			if truncated {
				out[key+"_truncated"] = true
			}
		}
	}
	locators := map[string]any{}
	for _, key := range []string{"entity_ref", "entity_id", "entity_kind", "id", "kind", "field", "source_file", "file", "path", "line", "column", "country_tag", "script_id", "parent_id"} {
		if value, ok := issue[key]; ok {
			preview, truncated := validationLocatorPreview(value)
			if preview != nil {
				locators[key] = preview
			}
			if truncated {
				locators[key+"_truncated"] = true
			}
		}
	}
	out["locators"] = locators
	return out
}

func validationLocatorPreview(value any) (any, bool) {
	ref, ok := value.(map[string]any)
	if !ok {
		return validationPreview(value, 128)
	}
	out := map[string]any{}
	omitted := false
	for _, key := range []string{"type", "id", "kind", "entity_id", "state_id", "tree_id", "field", "path", "line", "column", "country_tag", "parent_id"} {
		value, present := ref[key]
		if !present {
			continue
		}
		preview, truncated := validationPreview(value, 128)
		if preview == nil {
			omitted = true
			continue
		}
		out[key] = preview
		if truncated {
			out[key+"_truncated"] = true
		}
		encoded, err := json.Marshal(out)

		if err != nil || len(encoded) > 960 {
			delete(out, key)
			delete(out, key+"_truncated")
			omitted = true
		}
		omitted = omitted || truncated
	}
	for key := range ref {
		if _, present := out[key]; !present {
			omitted = true
		}
	}
	if omitted {
		out["details_omitted"] = true
	}
	return out, omitted
}

func compactValidationPage(issues []ExportIssue, summary map[string]any, p validationPageRequest) (map[string]any, *validationPageError) {

	type fingerprint struct {
		issue ExportIssue
		hash  string
	}
	ordered := make([]fingerprint, 0, len(issues))
	for _, issue := range issues {
		h := sha256.New()
		if err := json.NewEncoder(h).Encode(issue); err != nil {
			return nil, &validationPageError{503, "validation_unavailable", "validation revision could not be computed"}
		}
		ordered = append(ordered, fingerprint{issue, hex.EncodeToString(h.Sum(nil))})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].hash < ordered[j].hash })
	issues = make([]ExportIssue, len(ordered))
	fingerprints := make([]string, len(ordered))
	for index, item := range ordered {
		issues[index], fingerprints[index] = item.issue, item.hash
	}
	hash := sha256.New()
	if err := json.NewEncoder(hash).Encode([]any{fingerprints, summary}); err != nil {
		return nil, &validationPageError{503, "validation_unavailable", "validation revision could not be computed"}
	}
	revision := hex.EncodeToString(hash.Sum(nil))
	if p.revision != "" && p.revision != revision {
		return nil, &validationPageError{409, "validation_changed", "validation inputs changed; restart at offset 0 without mixing pages"}
	}
	counts := map[string]int{"error": 0, "warning": 0, "info": 0}
	for _, issue := range issues {
		counts[validationLevel(issue)]++
	}
	out := map[string]any{"view": "compact", "summary": summary, "issue_counts": counts, "issue_total": len(issues),
		"revision": revision, "is_snapshot": false, "offset": p.offset, "limit": p.limit}
	var items []map[string]any
	total := len(issues)
	if p.issueIndex == nil {
		if p.offset > 0 && p.offset >= total {
			return nil, &validationPageError{422, "invalid_validation_page", "offset is outside the findings; use the returned next_offset"}
		}
		items = make([]map[string]any, 0, p.limit)
		budget := 0
		for index := p.offset; index < total && len(items) < p.limit; index++ {
			item := validationIssuePreview(issues[index], index)
			encoded, _ := json.Marshal(item)
			if len(encoded) > validationPageBudget {
				return nil, &validationPageError{503, "validation_unavailable", "one compact finding exceeds the response budget"}
			}
			if budget+len(encoded) > validationPageBudget && len(items) > 0 {
				break
			}
			budget += len(encoded)
			items = append(items, item)
		}
		out["issues"] = items
	} else {
		if *p.issueIndex >= len(issues) {
			return nil, &validationPageError{422, "invalid_validation_page", "issue_index is outside this revision"}
		}

		encoded, err := json.Marshal(issues[*p.issueIndex])
		var value any
		if err != nil || json.Unmarshal(encoded, &value) != nil {
			return nil, &validationPageError{503, "validation_unavailable", "issue detail could not be read"}
		}
		value, err = validationPointer(value, p.detailPath)
		if err != nil {
			return nil, &validationPageError{422, "invalid_validation_detail", "detail_path does not identify a JSON node in this issue"}
		}
		items, total, err = validationDetailItems(value, p)
		if err != nil {
			return nil, &validationPageError{422, "invalid_validation_page", "detail offset is outside this node; use its next_offset"}
		}
		out["issue_index"], out["detail_path"] = *p.issueIndex, p.detailPath
		out["detail_items"] = items
		out["detail_unit"] = "object keys, array items, or 256-codepoint text chunks"
	}
	out["total"], out["returned"] = total, len(items)
	out["done"] = p.offset+len(items) >= total
	out["next_offset"] = nil
	if out["done"] != true {
		out["next_offset"] = p.offset + len(items)
	}
	return out, nil
}

func validationPointer(value any, path string) (any, error) {
	if path == "" {
		return value, nil
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 16 {
		return nil, fmt.Errorf("pointer too deep")
	}
	for _, part := range parts {
		for index := 0; index < len(part); index++ {
			if part[index] == '~' {
				if index+1 >= len(part) || (part[index+1] != '0' && part[index+1] != '1') {
					return nil, fmt.Errorf("invalid pointer escape")
				}
				index++
			}
		}
		key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch node := value.(type) {
		case map[string]any:
			var exists bool
			value, exists = node[key]
			if !exists {
				return nil, fmt.Errorf("missing key")
			}
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(node) || strconv.Itoa(index) != key {
				return nil, fmt.Errorf("invalid array index")
			}
			value = node[index]
		default:
			return nil, fmt.Errorf("not a container")
		}
	}
	return value, nil
}

func validationDetailItems(value any, p validationPageRequest) ([]map[string]any, int, error) {
	items := []map[string]any{}
	keys := []string{}
	total := 1
	switch node := value.(type) {
	case map[string]any:
		for key := range node {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		total = len(keys)
	case []any:
		total = len(node)
	case string:
		total = (utf8.RuneCountInString(node) + 255) / 256
	}
	if p.offset > 0 && p.offset >= total {
		return nil, total, fmt.Errorf("offset outside node")
	}
	budget := 0
	for index := p.offset; index < total && len(items) < p.limit; index++ {
		item := map[string]any{}
		path, child := p.detailPath, value
		switch node := value.(type) {
		case map[string]any:
			key := keys[index]
			path += "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
			child = node[key]
			item["key"], item["key_truncated"] = validationPreview(key, 128)
		case []any:
			path += "/" + strconv.Itoa(index)
			child = node[index]
		case string:
			item["chunk"] = index
			child = validationTextChunk(node, index)
		}
		if len(path) <= 512 {
			item["detail_path"] = path
		} else {
			item["locator_unavailable"] = "JSON pointer exceeds 512 bytes"
		}
		switch node := child.(type) {
		case map[string]any:
			item["type"], item["children"], item["details_omitted"] = "object", len(node), true
		case []any:
			item["type"], item["children"], item["details_omitted"] = "array", len(node), true
		default:
			item["value"], item["value_truncated"] = validationPreview(child, 1024)
		}
		encoded, _ := json.Marshal(item)
		if len(encoded) > validationPageBudget {
			return nil, total, fmt.Errorf("one detail item exceeds the budget")
		}
		if budget+len(encoded) > validationPageBudget && len(items) > 0 {
			break
		}
		budget += len(encoded)
		items = append(items, item)
	}
	return items, total, nil
}

func validationTextChunk(text string, chunk int) string {
	start, end := len(text), len(text)
	position := 0
	for index := range text {
		if position == chunk*256 {
			start = index
		}
		if position == (chunk+1)*256 {
			end = index
			break
		}
		position++
	}
	return text[start:end]
}
