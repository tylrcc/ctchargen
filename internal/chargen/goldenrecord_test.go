package chargen_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/render"
)

// twelve is the reenlistment throw p. 6 singles out, and E003 turns on.
const twelve = 12

// record is as much of the written record as the conditions need.
//
// It is decoded from the written record rather than read off the Character,
// because the readings state every condition over a record - "records where
// the enlistment throw failed" - and a record is what this reads. The record
// is the one each fixture writes now; TestGoldens holds it to its golden.
type record struct {
	name string

	Terms           int            `json:"terms"`
	Rank            int            `json:"rank"`
	Service         string         `json:"service"`
	Characteristics map[string]int `json:"characteristics"`
	Errata          []string       `json:"errata"`

	Departure *struct {
		How   string `json:"how"`
		Fatal bool   `json:"fatal"`
	} `json:"departure"`

	Events []event `json:"events"`
}

// event is one line of the generation record. term is not on the wire: it is
// filled in below from the step events, so a throw can be asked which term it
// fell in - which E003 and E013 both need and neither can get otherwise.
type event struct {
	Kind      string `json:"kind"`
	Step      string `json:"step"`
	Point     string `json:"point"`
	Total     int    `json:"total"`
	Succeeded bool   `json:"succeeded"`

	term int
}

// How a throw's outcome is read, where a condition cares.
type outcome int

const (
	either outcome = iota
	passed
	failed
)

func (o outcome) matches(succeeded bool) bool {
	switch o {
	case passed:
		return succeeded
	case failed:
		return !succeeded
	case either:
		return true
	}

	return false
}

// throws is every throw whose step begins with the given name.
//
// A prefix, because several steps carry what they were thrown for: the
// enlistment throw names its service, the aging saving throws name their
// characteristic.
func (r record) throws(step string) []event {
	return r.dice(step, aThrow)
}

// rolls is every roll of the named step. A roll meets no target, so it has
// no outcome to read - which is why it is a separate case since #50, and why
// asking for one where a throw is meant would silently find nothing.
func (r record) rolls(step string) []event {
	return r.dice(step, aRoll)
}

// dice is every event of the named step and kind.
//
// Implements: Every step the errata conditions read is found, as the kind they
// read it as [Documents].
func (r record) dice(step, kind string) []event {
	var found []event

	for _, e := range r.Events {
		if e.Kind == kind && strings.HasPrefix(e.Step, step) {
			found = append(found, e)
		}
	}

	return found
}

// threw reports a throw of the named step with the given outcome.
func (r record) threw(step string, want outcome) bool {
	for _, e := range r.throws(step) {
		if want.matches(e.Succeeded) {
			return true
		}
	}

	return false
}

// firstTotal is the total of the first roll of a step, and whether there was
// one. E011 needs the Social Standing as p. 4 rolled it, before any aging
// moved it.
func (r record) firstTotal(step string) (int, bool) {
	found := r.rolls(step)
	if len(found) == 0 {
		return 0, false
	}

	return found[0].Total, true
}

func (r record) departedBy(how string) bool {
	return r.Departure != nil && r.Departure.How == how
}

func (r record) died() bool { return r.Departure != nil && r.Departure.Fatal }

// readRecords decodes the record every fixture writes, as the engine writes
// it now, and refuses to run on none: a check over an empty roster passes by
// saying nothing.
//
// It reads what the engine generates rather than the goldens on disk, which
// TestGoldens holds equal, so an engine change is judged here by what it did
// to a reading rather than only as a golden diff (#373). Each record is named
// as its golden is.
func readRecords(t *testing.T) []record {
	t.Helper()

	if len(fixtures) == 0 {
		t.Fatal("no fixtures on the roster")
	}

	records := make([]record, 0, len(fixtures))

	for _, fixture := range fixtures {
		text, err := render.JSON(fixture.generate(t))
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		var rec record

		err = json.Unmarshal(text, &rec)
		if err != nil {
			t.Fatalf("reading %s: %v", fixture.name, err)
		}

		rec.name = fixture.name + ".json"
		withTerms(&rec)

		records = append(records, rec)
	}

	return records
}

// withTerms marks each event with the term it fell inside.
//
// The engine opens a term with a step event whose step is "term N", and
// everything after it belongs to that term until the next one. Events before
// the first - the characteristic rolls, the enlistment throw - are term 0,
// which is no term and is what the conditions want them to be.
func withTerms(rec *record) {
	current := 0

	for i, e := range rec.Events {
		if e.Kind == "step" && strings.HasPrefix(e.Step, termStep) {
			// Only "term N" opens a term. Any other step beginning with the
			// word is left alone rather than guessed at.
			n, err := strconv.Atoi(strings.TrimPrefix(e.Step, termStep))
			if err == nil {
				current = n
			}
		}

		rec.Events[i].term = current
	}
}
