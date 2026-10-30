package importassets

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func GUISourceSnapshotKey(envPrefix, projectUUID string) (string, error) {
	envPrefix = strings.Trim(strings.TrimSpace(envPrefix), "/")
	projectUUID = strings.TrimSpace(projectUUID)
	if envPrefix == "" || projectUUID == "" || projectUUID == "." || projectUUID == ".." ||
		strings.Contains(projectUUID, "/") || strings.Contains(projectUUID, "\\") {
		return "", errors.New("importassets: invalid GUI source snapshot scope")
	}
	return fmt.Sprintf("%s/p/%s/gui-source/snapshot.zip", envPrefix, projectUUID), nil
}

const GUISourceGameDataMarker = "snapshot_meta/game_data.txt"

func CertifiedGUISourceSnapshotKey(envPrefix, projectUUID, archiveSHA256 string) (string, error) {
	base, err := GUISourceSnapshotKey(envPrefix, projectUUID)
	archiveSHA256 = strings.ToLower(strings.TrimSpace(archiveSHA256))
	if err != nil || !sha256Hex.MatchString(archiveSHA256) {
		return "", errors.New("importassets: invalid certified GUI source snapshot scope")
	}
	return path.Join(path.Dir(base), "certified", archiveSHA256+".zip"), nil
}

func EntityAssetKey(envPrefix, projectUUID, kind string, entityID int64, entityType, sha string) (string, error) {
	envPrefix = strings.Trim(strings.TrimSpace(envPrefix), "/")
	projectUUID = strings.TrimSpace(projectUUID)
	kind, err := safeObjectPath(kind)
	if err != nil || strings.Contains(kind, "/") {
		return "", errors.New("importassets: invalid asset kind")
	}
	entityType, err = safeObjectPath(entityType)
	if err != nil || strings.Contains(entityType, "/") {
		return "", errors.New("importassets: invalid asset type")
	}
	sha = strings.ToLower(strings.TrimSpace(sha))
	if envPrefix == "" || projectUUID == "" || entityID <= 0 || !sha256Hex.MatchString(sha) {
		return "", errors.New("importassets: invalid entity asset identity")
	}
	return fmt.Sprintf("%s/p/%s/upload/%s/%d/%s/%s.png", envPrefix, projectUUID, kind, entityID, entityType, sha), nil
}

func IsEntityAssetKey(envPrefix, projectUUID, kind string, entityID int64, entityType, key string) bool {
	prefix, err := EntityAssetKey(envPrefix, projectUUID, kind, entityID, entityType, strings.Repeat("0", 64))
	if err != nil {
		return false
	}
	prefix = strings.TrimSuffix(prefix, strings.Repeat("0", 64)+".png")
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(key, prefix)
	return len(suffix) == 68 && strings.HasSuffix(suffix, ".png") && sha256Hex.MatchString(strings.TrimSuffix(suffix, ".png"))
}

func NamedAssetKey(envPrefix, projectUUID, kind, logicalName, sha string) (string, error) {
	envPrefix = strings.Trim(strings.TrimSpace(envPrefix), "/")
	projectUUID = strings.TrimSpace(projectUUID)
	kind, err := safeObjectPath(kind)
	if err != nil || strings.Contains(kind, "/") {
		return "", errors.New("importassets: invalid named asset kind")
	}
	logicalName, err = safeObjectPath(logicalName)
	if err != nil || strings.Contains(logicalName, "/") {
		return "", errors.New("importassets: invalid named asset name")
	}
	sha = strings.ToLower(strings.TrimSpace(sha))
	if envPrefix == "" || projectUUID == "" || !sha256Hex.MatchString(sha) {
		return "", errors.New("importassets: invalid named asset identity")
	}
	return fmt.Sprintf("%s/p/%s/upload/%s/named/%s/%s.png", envPrefix, projectUUID, kind, logicalName, sha), nil
}

func IsNamedAssetKey(envPrefix, projectUUID, kind, logicalName, key string) bool {
	prefix, err := NamedAssetKey(envPrefix, projectUUID, kind, logicalName, strings.Repeat("0", 64))
	if err != nil {
		return false
	}
	prefix = strings.TrimSuffix(prefix, strings.Repeat("0", 64)+".png")
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(key, prefix)
	return len(suffix) == 68 && strings.HasSuffix(suffix, ".png") && sha256Hex.MatchString(strings.TrimSuffix(suffix, ".png"))
}

func safeObjectPath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	clean := path.Clean(value)
	if value == "" || strings.HasPrefix(value, "/") || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("importassets: unsafe object path")
	}
	return clean, nil
}

func RawKey(envPrefix, projectUUID, relative string) (string, error) {
	clean, err := safeObjectPath(relative)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/p/%s/import/raw/%s", strings.Trim(envPrefix, "/"), projectUUID, clean), nil
}

func UploadKey(envPrefix, projectUUID, kind, filename string) (string, error) {
	kind, err := safeObjectPath(kind)
	if err != nil || strings.Contains(kind, "/") {
		return "", errors.New("importassets: invalid upload kind")
	}
	filename, err = safeObjectPath(filename)
	if err != nil || strings.Contains(filename, "/") {
		return "", errors.New("importassets: invalid upload filename")
	}
	return fmt.Sprintf("%s/p/%s/upload/%s/%s", strings.Trim(envPrefix, "/"), projectUUID, kind, filename), nil
}
