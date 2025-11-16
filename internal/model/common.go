package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONBMap represents a JSONB column as a map[string]interface{}
type JSONBMap map[string]interface{}

// Value implements the driver.Valuer interface for database writes
func (j JSONBMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database reads
func (j *JSONBMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONB value: not a byte slice")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}
