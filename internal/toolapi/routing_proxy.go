package toolapi

import (
	"fmt"
	"net/url"
	"strings"
)

func programTargetPath(path string) (string, error) {
	if !safeProjectProxyPath(path) {
		return "", fmt.Errorf("tool path is outside the project API: %s", path)
	}
	switch {
	case exactQueryOrTree(path, "/api/projects"):
		return "/api/go" + strings.TrimPrefix(path, "/api"), nil
	case exactQueryOrTree(path, "/api/go/export/jobs"):
		return path, nil
	case path == "/api/app/status" || path == "/api/app/game-index":
		return path, nil
	default:
		return "", fmt.Errorf("tool path is outside the project API: %s", path)
	}
}

func exactQueryOrTree(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"?") || strings.HasPrefix(path, root+"/")
}

func safeProjectProxyPath(target string) bool {
	if !strings.HasPrefix(target, "/") || strings.ContainsAny(target, "\\\r\n\x00#") {
		return false
	}
	parsed, err := url.ParseRequestURI(target)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	decodedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return false
	}
	for _, segment := range strings.Split(decodedPath, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
