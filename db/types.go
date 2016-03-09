package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/pborman/uuid"
)

type NullUUID struct {
	Valid bool
	UUID  uuid.UUID
}

func (v NullUUID) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.UUID.String(), nil
}

func (v *NullUUID) Scan(value interface{}) error {
	if value == nil {
		v.UUID, v.Valid = nil, false
		return nil
	}
	v.Valid = true
	if b, ok := value.([]byte); ok {
		v.UUID = uuid.Parse(string(b))
		return nil
	}
	return fmt.Errorf("unexpected (non-string) value '%v' found in database for UUID parameter", value)
}
