package importsvc

import (
	"regexp"
	"strconv"
)

var hoi4DateRE = regexp.MustCompile(
	`\A([0-9]{1,4})\.([0-9]{1,2})\.([0-9]{1,2})(?:\.([0-9]{1,2}))?\z`)

type HistoryDateBlockValidationError struct{ Msg string }

func (e *HistoryDateBlockValidationError) Error() string { return e.Msg }

type ExtractedHistoryDateBlock struct {
	Date      string
	Script    string
	Line      int
	Start     int
	End       int
	BodyStart int
	BodyEnd   int
}

func skipSpaceAndComments(source []rune, index int) int {
	length := len(source)
	for index < length {
		if pyIsSpace(source[index]) {
			index++
			continue
		}
		if source[index] == '#' {
			newline := -1
			for j := index; j < length; j++ {
				if source[j] == '\n' {
					newline = j
					break
				}
			}
			if newline < 0 {
				return length
			}
			index = newline + 1
			continue
		}
		break
	}
	return index
}

func matchingBlockClose(source []rune, opening int) int {
	depth := 1
	index := opening + 1
	inQuote := false
	escaped := false
	inComment := false
	for index < len(source) {
		char := source[index]
		if inComment {
			if char == '\n' {
				inComment = false
			}
			index++
			continue
		}
		if inQuote {
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inQuote = false
			}
			index++
			continue
		}
		switch char {
		case '#':
			inComment = true
		case '"':
			inQuote = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
		index++
	}
	return -1
}

func quotedEnd(source []rune, opening int) int {
	index := opening + 1
	escaped := false
	for index < len(source) {
		char := source[index]
		if escaped {
			escaped = false
		} else if char == '\\' {
			escaped = true
		} else if char == '"' {
			return index + 1
		}
		index++
	}
	return len(source)
}

func isRuneIn(r rune, set string) bool {
	for _, c := range set {
		if r == c {
			return true
		}
	}
	return false
}

func ExtractHistoryDateBlocks(sourceText string) []ExtractedHistoryDateBlock {
	source := []rune(sourceText)
	found := []ExtractedHistoryDateBlock{}
	index := 0
	if len(source) > 0 && source[0] == 0xFEFF {
		index = 1
	}
	length := len(source)
	for index < length {
		index = skipSpaceAndComments(source, index)
		if index >= length {
			break
		}

		if source[index] == '"' {
			index = quotedEnd(source, index)
			operator := skipSpaceAndComments(source, index)
			if operator < length && source[operator] == '=' {
				valueStart := skipSpaceAndComments(source, operator+1)
				if valueStart < length && source[valueStart] == '{' {
					closing := matchingBlockClose(source, valueStart)
					if closing < 0 {
						break
					}
					index = closing + 1
				} else if valueStart < length && source[valueStart] == '"' {
					index = quotedEnd(source, valueStart)
				} else {
					index = valueStart
					for index < length && !pyIsSpace(source[index]) && source[index] != '#' {
						index++
					}
				}
			}
			continue
		}
		if source[index] == '{' {
			closing := matchingBlockClose(source, index)
			if closing < 0 {
				break
			}
			index = closing + 1
			continue
		}
		if isRuneIn(source[index], "}=<>!") {
			index++
			continue
		}
		keyStart := index
		for index < length && !pyIsSpace(source[index]) && !isRuneIn(source[index], "{}=<>!\"#") {
			index++
		}
		if index == keyStart {
			index++
			continue
		}
		key := string(source[keyStart:index])

		operator := skipSpaceAndComments(source, index)
		if operator >= length || source[operator] != '=' {
			if index < keyStart+1 {
				index = keyStart + 1
			}
			continue
		}
		opening := skipSpaceAndComments(source, operator+1)
		if opening >= length || source[opening] != '{' {

			if opening < length && source[opening] == '"' {
				index = quotedEnd(source, opening)
			} else {
				index = opening
				for index < length && !pyIsSpace(source[index]) && source[index] != '#' {
					index++
				}
			}
			continue
		}
		closing := matchingBlockClose(source, opening)
		if closing < 0 {
			break
		}

		if date, ok := TryParseHistoryDateKey(key); ok {
			line := 1
			for j := 0; j < keyStart; j++ {
				if source[j] == '\n' {
					line++
				}
			}
			found = append(found, ExtractedHistoryDateBlock{
				Date:      date,
				Script:    string(source[opening+1 : closing]),
				Line:      line,
				Start:     keyStart,
				End:       closing + 1,
				BodyStart: opening + 1,
				BodyEnd:   closing,
			})
		}
		index = closing + 1
	}
	return found
}

func ValidateHistoryDate(value string) (string, error) {
	raw := pyStrip(value)
	m := hoi4DateRE.FindStringSubmatch(raw)
	if m == nil {
		return "", &HistoryDateBlockValidationError{
			Msg: "date must match Y.M.D or Y.M.D.H (for example 1939.1.1.12)"}
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	hour := 12
	if m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
	}
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return "", &HistoryDateBlockValidationError{
			Msg: "date must use a positive year, month 1-12, and day 1-31"}
	}
	if hour < 0 || hour > 23 {
		return "", &HistoryDateBlockValidationError{
			Msg: "date hour must be between 0 and 23"}
	}
	return raw, nil
}

func TryParseHistoryDateKey(value string) (string, bool) {
	date, err := ValidateHistoryDate(value)
	if err != nil {
		return "", false
	}
	return date, true
}
