package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

func RawOverrideLogEntry(modDir string, lines []string, msg exportmsg.Printer) Entry {
	if modDir == "" || len(lines) == 0 {
		return Entry{}
	}
	body := append(commentLines(msg.Text("raw_override.log.header", nil)), "")
	body = append(body, lines...)
	return Entry{
		Path:         modDir + "/_RAW_OVERRIDE_FILES.log",
		Bytes:        []byte(strings.Join(body, "\n") + "\n"),
		Origin:       "generated",
		SourceEntity: "raw_override_log",
	}
}
