package tdrpc

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
)

// Scan decodes the channel backup from base64
func (data *ChanBackupData) Scan(src interface{}) error {

	switch st := src.(type) {
	case string:
		var err error
		*data, err = base64.StdEncoding.DecodeString(st)
		return err
	default:
		return fmt.Errorf("Invalid ChanBackupData type: %T=%v", src, src)
	}

}

// Value encodes the channel backup to base64
func (data ChanBackupData) Value() (driver.Value, error) {

	return base64.StdEncoding.EncodeToString(data), nil

}
