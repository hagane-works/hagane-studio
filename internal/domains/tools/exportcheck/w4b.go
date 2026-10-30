package exportcheck

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type HistoryFile map[string]any

func normalizeHistoryPopularities(raw any) (map[string]int, error) {
	if raw == nil {
		return map[string]int{}, nil
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("set_popularities must be an object keyed by ROOT ideology id")
	}
	out := make(map[string]int, len(values))
	total := 0
	for rawID, rawValue := range values {
		id := strings.TrimSpace(rawID)
		if !rootIDRe.MatchString(id) {
			return nil, fmt.Errorf("set_popularities contains invalid ROOT ideology id %q", rawID)
		}
		value, ok := historyPopularityInt(rawValue)
		if !ok || value < 0 || value > 100 {
			return nil, fmt.Errorf("set_popularities[%q] must be an integer from 0 to 100", id)
		}
		out[id] = value
		total += value
	}
	if len(out) > 0 && total != 100 {
		return nil, fmt.Errorf("set_popularities must add up to 100 (got %d)", total)
	}
	return out, nil
}

func historyPopularityInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), int64(int(value)) == value
	case int32:
		return int(value), true
	case float64:
		return int(value), value == float64(int(value))
	case float32:
		return int(value), value == float32(int(value))
	case json.Number:
		n, err := strconv.Atoi(string(value))
		return n, err == nil
	default:
		return 0, false
	}
}
