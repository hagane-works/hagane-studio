package rawfiles

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

//go:embed source_file_shadow_allowlist_1_19_2.json
var shadowPolicyJSON []byte

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
	base := joined
	if slash := strings.LastIndex(strings.ReplaceAll(joined, "\\", "/"), "/"); slash >= 0 {
		base = strings.ReplaceAll(joined, "\\", "/")[slash+1:]
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

type TombstoneError struct {
	Code    string
	Message string

	Findings    []map[string]any
	NetDelta    int
	RepairRoute string
}

func (e *TombstoneError) Error() string { return e.Message }

func tombstoneReject(code, message string) *TombstoneError {
	return &TombstoneError{Code: code, Message: message}
}

var EmptyShadowConflictCodes = map[string]bool{
	"empty_shadow_game_version_not_allowlisted": true,
	"empty_shadow_policy_unverified":            true,
	"empty_shadow_path_not_allowlisted":         true,
	"empty_shadow_layer_not_allowlisted":        true,
	"empty_shadow_hash_not_allowlisted":         true,
	"policy_load_failed":                        true,
}

type RawFileIntent struct {
	RawKind             string
	DestPath            string
	Content             string
	IsBinary            bool
	GenerateReplacePath bool
	GameVersion         *string
	TargetLayer         *string
	TargetSHA256        *string
}

type shadowPolicy struct {
	SchemaVersion string `json:"schema_version"`
	GameVersion   string `json:"game_version"`
	Verified      bool   `json:"verified"`
	Entries       []struct {
		TargetPath   string `json:"target_path"`
		TargetLayer  string `json:"target_layer"`
		TargetSHA256 string `json:"target_sha256"`
	} `json:"entries"`
}

var shadowPolicyOnce = sync.OnceValues(func() (*shadowPolicy, error) {
	var p shadowPolicy
	if err := json.Unmarshal(shadowPolicyJSON, &p); err != nil {
		return nil, tombstoneReject("policy_load_failed",
			"source-file shadow policy could not be loaded: bundled allowlist")
	}

	if p.SchemaVersion != "hagane.source-file-shadow-policy.v1" {
		return nil, tombstoneReject("policy_schema_invalid",
			"source-file shadow policy schema is unsupported")
	}
	return &p, nil
})

var (
	tombstoneSHA256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)
	tombstoneDriveRe  = regexp.MustCompile(`^[A-Za-z]:`)
	tombstoneWSRe     = regexp.MustCompile(`\s+`)
)

func tombstoneNormPath(value string) (string, *TombstoneError) {
	raw := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if raw == "" {
		return "", tombstoneReject("target_path_required", "target path is required")
	}
	if strings.HasPrefix(raw, "/") || tombstoneDriveRe.MatchString(raw) {
		return "", tombstoneReject("target_path_absolute", "target path must be mod-relative")
	}
	var parts []string
	for _, part := range strings.Split(raw, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", tombstoneReject("target_path_traversal", "target path may not contain '..'")
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "", tombstoneReject("target_path_required", "target path is required")
	}
	return strings.Join(parts, "/"), nil
}

func tombstoneNormLayer(value string) (string, *TombstoneError) {
	normalised := strings.TrimSpace(tombstoneWSRe.ReplaceAllString(value, " "))
	if normalised == "" {
		return "", tombstoneReject("target_layer_required", "empty_shadow requires target_layer")
	}
	return normalised, nil
}

func tombstoneNormSHA(value string) (string, *TombstoneError) {
	normalised := strings.ToLower(strings.TrimSpace(value))
	if !tombstoneSHA256Re.MatchString(normalised) {
		return "", tombstoneReject("target_sha256_invalid", "target_sha256 must be 64 lowercase hex characters")
	}
	return normalised, nil
}

func tombstoneDangerousTarget(path string) bool {
	key := strings.ToLower(path)
	return key == "map/buildings.txt" || key == "common/bookmarks" ||
		strings.HasPrefix(key, "common/bookmarks/")
}

func ValidateRawFileIntent(rawKind, destPath, content string, isBinary, generateReplacePath bool,
	gameVersion, targetLayer, targetSHA256 string) (RawFileIntent, *TombstoneError) {

	if rawKind != "verbatim_override" && rawKind != "empty_shadow" {
		return RawFileIntent{}, tombstoneReject("raw_kind_invalid",
			"raw_kind must be verbatim_override or empty_shadow")
	}
	normalisedPath, terr := tombstoneNormPath(destPath)
	if terr != nil {
		return RawFileIntent{}, terr
	}
	if rawKind == "verbatim_override" {

		if _, rerr := GateRawContent(normalisedPath, content, isBinary); rerr != nil {
			return RawFileIntent{}, rerr.AsTombstone()
		}
		return RawFileIntent{
			RawKind:             rawKind,
			DestPath:            normalisedPath,
			Content:             content,
			IsBinary:            isBinary,
			GenerateReplacePath: generateReplacePath,
		}, nil
	}
	if tombstoneDangerousTarget(normalisedPath) {
		return RawFileIntent{}, tombstoneReject("dangerous_empty_shadow_target",
			"empty shadow is always blocked for "+normalisedPath)
	}
	if isBinary {
		return RawFileIntent{}, tombstoneReject("empty_shadow_binary_forbidden", "empty_shadow is text-only")
	}
	if content != "" {
		return RawFileIntent{}, tombstoneReject("empty_shadow_requires_canonical_empty",
			"empty_shadow content must be the canonical empty string")
	}
	if generateReplacePath {
		return RawFileIntent{}, tombstoneReject("empty_shadow_replace_path_forbidden",
			"empty_shadow may not generate a directory replace_path")
	}
	requestedVersion := strings.TrimSpace(gameVersion)
	if requestedVersion == "" {
		return RawFileIntent{}, tombstoneReject("empty_shadow_game_version_required",
			"empty_shadow requires game_version")
	}
	requestedLayer, terr := tombstoneNormLayer(targetLayer)
	if terr != nil {
		return RawFileIntent{}, terr
	}
	requestedSHA, terr := tombstoneNormSHA(targetSHA256)
	if terr != nil {
		return RawFileIntent{}, terr
	}
	policy, err := shadowPolicyOnce()
	if err != nil {
		return RawFileIntent{}, err.(*TombstoneError)
	}
	if requestedVersion != policy.GameVersion {
		return RawFileIntent{}, tombstoneReject("empty_shadow_game_version_not_allowlisted",
			"empty_shadow game_version does not match the verified policy")
	}
	if !policy.Verified {
		return RawFileIntent{}, tombstoneReject("empty_shadow_policy_unverified",
			"empty_shadow is blocked until real-engine policy evidence is verified")
	}
	pathKey := strings.ToLower(normalisedPath)
	matchedPath := false
	matchedLayer := false
	matchedSHA := false
	for _, e := range policy.Entries {
		ePath, terr := tombstoneNormPath(e.TargetPath)
		if terr != nil {
			continue
		}
		if strings.ToLower(ePath) != pathKey {
			continue
		}
		matchedPath = true
		eLayer, terr := tombstoneNormLayer(e.TargetLayer)
		if terr != nil || !strings.EqualFold(eLayer, requestedLayer) {
			continue
		}
		matchedLayer = true
		if strings.ToLower(e.TargetSHA256) == requestedSHA {
			matchedSHA = true
			break
		}
	}
	if !matchedPath {
		return RawFileIntent{}, tombstoneReject("empty_shadow_path_not_allowlisted",
			"empty_shadow target path is not in the verified allowlist")
	}
	if !matchedLayer {
		return RawFileIntent{}, tombstoneReject("empty_shadow_layer_not_allowlisted",
			"empty_shadow target layer is not in the verified allowlist")
	}
	if !matchedSHA {
		return RawFileIntent{}, tombstoneReject("empty_shadow_hash_not_allowlisted",
			"empty_shadow target SHA-256 does not match verified evidence")
	}
	return RawFileIntent{
		RawKind:             rawKind,
		DestPath:            normalisedPath,
		Content:             "",
		IsBinary:            false,
		GenerateReplacePath: false,
		GameVersion:         &requestedVersion,
		TargetLayer:         &requestedLayer,
		TargetSHA256:        &requestedSHA,
	}, nil
}
