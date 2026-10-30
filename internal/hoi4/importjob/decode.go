package importjob

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func marshalPGJSON(value any) ([]byte, error) {
	normalized, err := NormalizeImportedBlockPayload(value)
	if err != nil {
		return nil, err
	}
	value = normalized.Value
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	raw = bytes.ReplaceAll(raw, []byte(`\u0000`), nil)
	return raw, nil
}

func decodeScriptBytes(body []byte) string {
	var text string
	switch {
	case bytes.HasPrefix(body, []byte{0xff, 0xfe}):
		decoded, _, err := transform.Bytes(unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder(), body)
		if err == nil {
			text = string(decoded)
		}
	case bytes.HasPrefix(body, []byte{0xfe, 0xff}):
		decoded, _, err := transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder(), body)
		if err == nil {
			text = string(decoded)
		}
	case utf8.Valid(body):
		text = strings.TrimPrefix(string(body), "\ufeff")
	default:
		decoded, _, err := transform.Bytes(charmap.Windows1252.NewDecoder(), body)
		if err == nil {
			text = string(decoded)
		} else {
			text = string(body)
		}
	}
	return strings.ReplaceAll(text, "\x00", "")
}
