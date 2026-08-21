package common

import (
	"encoding/json"
	"fmt"
)

func MustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	}
	return b
}

func JSONString(v any) string {
	return string(MustJSON(v))
}
