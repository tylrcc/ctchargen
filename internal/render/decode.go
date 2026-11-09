package render

import (
	"encoding/json"
	"errors"
	"fmt"
)

// errNotARecord reports input that cannot be read as a character record.
var errNotARecord = errors.New("not a character record")

// decode reads a record back from what JSON wrote.
//
// Nothing is rebuilt into a domain type. The record is a projection of the
// domain values, and a sheet is a projection of the record: rebuilding an
// Enlistment or a Departure back into an interface would give the renderer
// nothing it does not already have, and would need a decoder for every sum
// that could go wrong in a way the wire shape cannot.
//
// Every event is read here, once, before anything is rendered, and one that
// will not read refuses the record: a sheet ends by telling the referee the
// record is the only way back to this character, and that is not something to
// say about a record that could not be read to its end. Reading is the only
// thing that can fail; what is rendered from what was read cannot (#362).
func decode(text []byte) (record, []eventJSON, error) {
	var decoded record

	err := json.Unmarshal(text, &decoded)
	if err != nil {
		return record{}, nil, fmt.Errorf("%w: %w", errNotARecord, err)
	}

	if decoded.UPP == "" {
		return record{}, nil, fmt.Errorf("%w: it carries no UPP", errNotARecord)
	}

	if decoded.Ruleset == "" {
		return record{}, nil, fmt.Errorf("%w: it names no ruleset", errNotARecord)
	}

	events := make([]eventJSON, 0, len(decoded.Events))

	for _, raw := range decoded.Events {
		var event eventJSON

		err := json.Unmarshal(raw, &event)
		if err != nil {
			return record{}, nil, fmt.Errorf("%w: an event will not read: %w", errNotARecord, err)
		}

		events = append(events, event)
	}

	return decoded, events, nil
}
