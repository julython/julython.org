// internal/db/types.go
package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap handles JSONB columns as map[string]any
type JSONMap map[string]any

func (j *JSONMap) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	var source []byte
	switch v := src.(type) {
	case []byte:
		source = v
	case string:
		source = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONMap", src)
	}
	return json.Unmarshal(source, j)
}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}
