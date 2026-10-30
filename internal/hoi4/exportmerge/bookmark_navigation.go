package exportmerge

import (
	"regexp"
	"strings"
)

var (
	bookmarkBlockStartRE  = regexp.MustCompile(`(?m)^\s*bookmark\s*=\s*\{`)
	bookmarkPlaceholderRE = regexp.MustCompile(`(?ms)^([ \t]*)bookmark[ \t]*=[ \t]*\{[ \t]*(?:\r?\n[ \t]*)?date[ \t]*=[ \t]*([0-9]{1,4}(?:\.[0-9]{1,2}){2,3})[ \t]*(?:\r?\n[ \t]*)?\}`)
)

func isBookmarkScriptPath(value string) bool {
	clean := strings.ToLower(strings.ReplaceAll(value, "\\", "/"))
	return (strings.HasPrefix(clean, "common/bookmarks/") || strings.Contains(clean, "/common/bookmarks/")) && strings.HasSuffix(clean, ".txt")
}

func preserveBookmarkNavigationPlaceholder(base, platform []byte) ([]byte, bool) {
	if len(bookmarkBlockStartRE.FindAll(platform, -1)) != 1 {
		return platform, false
	}
	match := bookmarkPlaceholderRE.FindSubmatch(base)
	if len(match) != 3 {
		return platform, false
	}
	text := string(platform)
	outer := strings.LastIndex(text, "}")
	if outer < 0 || !strings.Contains(text[:outer], "bookmarks") {
		return platform, false
	}
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	indent := string(match[1])
	if indent == "" {
		indent = "\t"
	}
	placeholder := indent + "bookmark = {" + newline +
		indent + "\tdate = " + string(match[2]) + newline +
		indent + "}" + newline
	prefix := text[:outer]
	if !strings.HasSuffix(prefix, newline) {
		prefix += newline
	}
	return []byte(prefix + newline + placeholder + text[outer:]), true
}
