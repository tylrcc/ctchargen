package render

import (
	"encoding/json"
	"errors"
	"fmt"
)

// errNotARecord reports input the render subcommand cannot read as a
// character record.
var errNotARecord = errors.New("not a character record")

// decode reads a record back from what JSON wrote.
//
// Nothing is rebuilt into a domain type. The record is a projection of the
// domain values, and a sheet is a projection of the record: rebuilding an
// Enlistment or a Departure back into an interface would give the renderer
// nothing it does not already have, and would need a decoder for every sum
// that could go wrong in a way the wire shape cannot.
func decode(text []byte) (record, error) {
	var decoded record

	err := json.Unmarshal(text, &decoded)
	if err != nil {
		return record{}, fmt.Errorf("%w: %w", errNotARecord, err)
	}

	if decoded.UPP == "" {
		return record{}, fmt.Errorf("%w: it carries no UPP", errNotARecord)
	}

	if decoded.Ruleset == "" {
		return record{}, fmt.Errorf("%w: it names no ruleset", errNotARecord)
	}

	return decoded, nil
}

// unmarshalEvent reads one logged event out of the record.
func unmarshalEvent(raw json.RawMessage, into *eventJSON) error {
	err := json.Unmarshal(raw, into)
	if err != nil {
		return fmt.Errorf("%w: an event will not read: %w", errNotARecord, err)
	}

	return nil
}
