package importassets

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ideologySourceSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func IdeologySourceSnapshotKey(envPrefix, projectUUID, sourceKey, archiveSHA256 string) (string, error) {
	envPrefix = strings.Trim(strings.TrimSpace(envPrefix), "/")
	projectUUID = strings.TrimSpace(projectUUID)
	sourceKey = strings.TrimSpace(sourceKey)
	archiveSHA256 = strings.ToLower(strings.TrimSpace(archiveSHA256))
	if envPrefix == "" || projectUUID == "" || strings.ContainsAny(projectUUID, `/\`) ||
		!ideologySourceSegment.MatchString(sourceKey) || sourceKey == "." || sourceKey == ".." ||
		len(archiveSHA256) != 64 || strings.Trim(archiveSHA256, "0123456789abcdef") != "" {
		return "", errors.New("importassets: invalid ideology source snapshot scope")
	}

	return fmt.Sprintf("%s/p/%s/ideology-source/%s/%s.zip", envPrefix, projectUUID, sourceKey, archiveSHA256), nil
}
