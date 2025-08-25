// services/utils.go

package src

import (
	"fmt"
	"strconv"
	"strings"
)

func toIntFromAny(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int(f)
		}
	case bool:
		if v {
			return 1
		}
		return 0
	}

	if str := fmt.Sprintf("%v", value); str != "" && str != "<nil>" {
		if i, err := strconv.Atoi(str); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return int(f)
		}
	}

	return 0
}

func toString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func isEmail(str string) bool {
	return strings.Contains(str, "@") && strings.Contains(str, ".")
}
