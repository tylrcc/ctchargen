package chargen_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// twelve is the reenlistment throw p. 6 singles out, and E003 turns on.
const twelve = 12

// record is as much of a finished character as the conditions need, in the
// words the readings state their conditions in - "records where the
// enlistment throw failed" - rather than the engine's types, so a predicate
// reads words and not code. It is read once off the finished Character by
// recordOf, and never off the run that made it.
type record struct {
	name string

	Terms           int
	Rank            int
	Service         string
	Characteristics map[string]int
	Errata          []string

	Departure *struct {
		How   string
		Fatal bool
	}

	Events []event
}

// stepKind is the kind of event that opens a step, the one kind the
// conditions find by its kind.
const stepKind = "step"

// event is one line of the generation record. term is filled in below from
// the step events, so a throw can be asked which term it fell in - which E003
// and E013 both need and neither can get otherwise.
type event struct {
	Kind      string
	Step      string
	Point     string
	Total     int
	Succeeded bool

	term int
}

// recordOf reads a finished character into the words the conditions use.
func recordOf(name string, character *chargen.Character) record {
	rec := record{
		name:            name,
		Terms:           character.Terms(),
		Rank:            int(character.Rank()),
		Characteristics: map[string]int{},
	}

	if service, served := character.ServedIn(); served {
		rec.Service = service.String()
	}

	profile := character.Profile()
	for _, c := range traveller.Characteristics {
		rec.Characteristics[c.String()] = profile[c]
	}

	for _, erratum := range character.Errata {
		rec.Errata = append(rec.Errata, erratum.String())
	}

	if character.Departure() != nil {
		rec.Departure = &struct {
			How   string
			Fatal bool
		}{How: departureWords(character.Departure()), Fatal: traveller.Fatal(character.Departure())}
	}

	for _, e := range character.Events {
		rec.Events = append(rec.Events, eventOf(e))
	}

	return rec
}

// eventOf is one event in the words the conditions use. It is a fold, so a
// kind of event added to traveller stops this compiling until it is read.
func eventOf(e traveller.Event) event {
	var fold eventFold

	// This fold cannot fail: every case only assigns.
	_ = e.Fold(&fold)

	return fold.out
}

type eventFold struct{ out event }

func (w *eventFold) Step(e traveller.StepEvent) error {
	w.out = event{Kind: stepKind, Step: e.Step}

	return nil
}

func (w *eventFold) Throw(e traveller.ThrowEvent) error {
	w.out = event{Kind: "throw", Step: e.Step, Total: e.Total(), Succeeded: e.Succeeded}

	return nil
}

func (w *eventFold) Roll(e traveller.RollEvent) error {
	w.out = event{Kind: "roll", Step: e.Step, Total: e.Total()}

	return nil
}

func (w *eventFold) Choice(e traveller.ChoiceEvent) error {
	w.out = event{Kind: "choice", Point: e.Point.String()}

	return nil
}

func (w *eventFold) Outcome(traveller.OutcomeEvent) error {
	w.out = event{Kind: "outcome"}

	return nil
}

// departureWords is how a departure is named where a condition reads it. It
// is a fold, for the reason eventOf is.
func departureWords(d traveller.Departure) string {
	var words departureWordsFold

	// This fold cannot fail: every case only assigns.
	_ = d.Fold(&words)

	return words.out
}

type departureWordsFold struct{ out string }

func (n *departureWordsFold) Discharged() error {
	n.out = "discharged"

	return nil
}

func (n *departureWordsFold) ForcedOut() error {
	n.out = "forced out"

	return nil
}

func (n *departureWordsFold) Retired() error {
	n.out = "retired"

	return nil
}

func (n *departureWordsFold) KilledBySurvivalThrow() error {
	n.out = "killed by the survival throw"

	return nil
}

func (n *departureWordsFold) KilledByMedicalCrisis(traveller.Characteristic) error {
	n.out = "killed by a medical crisis"

	return nil
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

// readRecords reads the character every fixture generates, as the engine
// generates it now, and refuses to run on none: a check over an empty roster
// passes by saying nothing.
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
		rec := recordOf(fixture.name+".md", fixture.generate(t))
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
		if e.Kind == stepKind && strings.HasPrefix(e.Step, termStep) {
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
