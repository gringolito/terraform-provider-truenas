package truenas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// DateTime represents a TrueNAS datetime value.
//
// TrueNAS serializes datetimes over the JSON-RPC wire as a Mongo-style object
//
//	{"$date": <epoch_milliseconds>}
//
// rather than as an ISO-8601 string, even though the API schema advertises the
// field as a date-time string. A plain *string field therefore fails to decode.
// DateTime knows how to unmarshal that object (and a JSON null).
type DateTime struct {
	time.Time
}

// UnmarshalJSON decodes either a {"$date": <epoch_ms>} object or null.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 || string(bytes.TrimSpace(data)) == "null" {
		d.Time = time.Time{}
		return nil
	}
	var wrapper struct {
		Date *int64 `json:"$date"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("decoding $date object: %w", err)
	}
	if wrapper.Date == nil {
		d.Time = time.Time{}
		return nil
	}
	d.Time = time.UnixMilli(*wrapper.Date).UTC()
	return nil
}
