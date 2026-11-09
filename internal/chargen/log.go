package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// log is the generation record: the ordered account of everything the
// procedure did, which serves audit - any character can be checked against
// Book 1 by walking it - and narrative.
//
// It is wired in from the first step rather than added afterwards, because a
// log that is added afterwards records what the author remembered.
//
// Implements: Every step, throw, choice and consequence, in order [Generation
// record].
type log struct {
	events []traveller.Event
	errata map[traveller.Erratum]bool
	seq    int

	// watch is called with each event as it is appended, or nil when nobody
	// is reading. It is how interactive mode shows the procedure between
	// its questions. It is handed the event the record holds, so a write
	// through the event's slices reaches this record and no other.
	watch func(traveller.Event)
}

func newLog() *log {
	return &log{errata: make(map[traveller.Erratum]bool)}
}

func (l *log) next() int {
	l.seq++

	return l.seq
}

// add appends an event and shows it to whoever is watching.
func (l *log) add(event traveller.Event) {
	l.events = append(l.events, event)

	if l.watch != nil {
		l.watch(event)
	}
}

// step records a procedure step being entered, with the pages that govern it.
func (l *log) step(name, pages string) {
	l.add(traveller.StepEvent{Seq: l.next(), Step: name, Pages: pages})
}

// throw records a throw and returns its sequence number, so the consequence
// can name what caused it.
func (l *log) throw(step string, t throw) int {
	seq := l.next()

	l.add(traveller.ThrowEvent{
		Seq:       seq,
		Step:      step,
		Dice:      []int{t.dice[0], t.dice[1]},
		DM:        t.modifier,
		Target:    t.target,
		Succeeded: t.succeeded,
	})

	return seq
}

// dieWithModifier records a one-die roll taken with a die modifier, so the
// record carries the row that was consulted and not only the face thrown.
//
// P. 9's two mustering out modifiers are the only ones a one-die throw takes,
// and without the modifier the record misreports: a die of 2 with the +1
// gambling expertise allows reads row 3 off the page, and a log that printed
// the 2 alone would send an auditor to the wrong row.
func (l *log) dieWithModifier(step string, face, modifier int) int {
	seq := l.next()

	l.add(traveller.RollEvent{Seq: seq, Step: step, Dice: []int{face}, DM: modifier})

	return seq
}

// dice records a roll with no target to meet, keeping every die. The record
// logs the dice and not only their total, because that is what an auditor
// walks against the page.
func (l *log) dice(step string, faces ...int) int {
	seq := l.next()

	l.add(traveller.RollEvent{Seq: seq, Step: step, Dice: faces})

	return seq
}

// choice records a choice point: which one, what was offered, and what was
// chosen. Who answered is the run's, not the event's: one run has one
// answerer.
func (l *log) choice(point traveller.ChoicePoint, from []string, chosen string) {
	l.add(traveller.ChoiceEvent{
		Seq: l.next(), Point: point, Alternatives: from, Chosen: chosen,
	})
}

// outcome records a consequence, the event that caused it, and any recorded
// readings that governed it.
func (l *log) outcomef(because int, errata []traveller.Erratum, format string, args ...any) {
	for _, e := range errata {
		l.errata[e] = true
	}

	l.add(traveller.OutcomeEvent{
		Seq:         l.next(),
		Because:     because,
		Description: fmt.Sprintf(format, args...),
		Errata:      errata,
	})
}

// stamped is every erratum that governed this generation, in id order.
//
// Implements: Every reading that governed a record is named on it [Authority].
func (l *log) stamped() []traveller.Erratum {
	var stamped []traveller.Erratum

	for _, e := range traveller.Errata {
		if l.errata[e] {
			stamped = append(stamped, e)
		}
	}

	return stamped
}
