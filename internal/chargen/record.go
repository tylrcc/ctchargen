package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// record is the generation record: the ordered account of everything the
// procedure did, which serves audit - any character can be checked against
// Book 1 by walking it - and narrative.
//
// It is wired in from the first step rather than added afterwards, because a
// record that is added afterwards holds what the author remembered.
//
// Implements: Every step, throw, choice and consequence, in order [Generation
// record].
type record struct {
	events []traveller.Event
	errata map[traveller.Erratum]bool

	// watch is called with each event as it is appended, or nil when nobody
	// is reading. It is how interactive mode shows the procedure between
	// its questions. It is handed the event the record holds, so a write
	// through the event's slices reaches this record and no other.
	watch func(traveller.Event)
}

func newRecord() *record {
	return &record{errata: make(map[traveller.Erratum]bool)}
}

// next is the number the next event appended will carry: events are
// numbered from 1 in the order they are appended, so it is read off the
// count rather than kept beside it (#368).
func (r *record) next() int { return len(r.events) + 1 }

// add appends an event and shows it to whoever is watching.
func (r *record) add(event traveller.Event) {
	r.events = append(r.events, event)

	if r.watch != nil {
		r.watch(event)
	}
}

// step records a procedure step being entered, with the pages that govern it.
func (r *record) step(name, pages string) {
	r.add(traveller.StepEvent{Seq: r.next(), Step: name, Pages: pages})
}

// throw records a throw and returns its sequence number, so the consequence
// can name what caused it.
func (r *record) throw(step string, t throw) int {
	seq := r.next()

	r.add(traveller.ThrowEvent{
		Seq:       seq,
		Step:      step,
		Dice:      []int{t.dice[0], t.dice[1]},
		DM:        t.modifier,
		Target:    t.target,
		Succeeded: t.succeeded,
		DMFrom:    t.from,
	})

	return seq
}

// dieWithModifier records a one-die roll taken with a die modifier, so the
// record carries the row that was consulted and not only the face thrown.
//
// P. 9's two mustering out modifiers are the only ones a one-die throw takes,
// and without the modifier the record misreports: a die of 2 with the +1
// gambling expertise allows reads row 3 off the page, and a record that printed
// the 2 alone would send an auditor to the wrong row.
func (r *record) dieWithModifier(step string, face, modifier int) int {
	seq := r.next()

	r.add(traveller.RollEvent{Seq: seq, Step: step, Dice: []int{face}, DM: modifier})

	return seq
}

// dice records a roll with no target to meet, keeping every die. The record
// keeps the dice and not only their total, because that is what an auditor
// walks against the page.
func (r *record) dice(step string, faces ...int) int {
	seq := r.next()

	r.add(traveller.RollEvent{Seq: seq, Step: step, Dice: faces})

	return seq
}

// choice records a choice point: which one, what was offered, and what was
// chosen. Who answered is the run's, not the event's: one run has one
// answerer.
func (r *record) choice(point traveller.ChoicePoint, from []string, chosen string) {
	r.add(traveller.ChoiceEvent{
		Seq: r.next(), Point: point, Alternatives: from, Chosen: chosen,
	})
}

// outcomef records a consequence and the event that caused it, and stamps the
// character with any readings that governed it.
func (r *record) outcomef(because int, errata []traveller.Erratum, format string, args ...any) {
	for _, e := range errata {
		r.errata[e] = true
	}

	r.add(traveller.OutcomeEvent{
		Seq:         r.next(),
		Because:     because,
		Description: fmt.Sprintf(format, args...),
	})
}

// stamped is every erratum that governed this generation, in id order.
//
// Implements: Every reading that governed a character is stamped on it
// [Authority].
func (r *record) stamped() []traveller.Erratum {
	var stamped []traveller.Erratum

	for _, e := range traveller.Errata {
		if r.errata[e] {
			stamped = append(stamped, e)
		}
	}

	return stamped
}

// fact records an outcome that carries a typed fact: its prose is the fact's
// own String, so the record and what is read from the fact cannot disagree
// (#412). Like outcomef, it stamps the character with any readings that
// governed it.
func (r *record) fact(because int, errata []traveller.Erratum, fact traveller.Fact) {
	for _, e := range errata {
		r.errata[e] = true
	}

	r.add(traveller.OutcomeEvent{
		Seq: r.next(), Because: because, Description: fact.String(), Fact: fact,
	})
}

// opens records the step that opens a term.
func (r *record) opens(term traveller.Term, pages string) {
	r.add(traveller.StepEvent{Seq: r.next(), Step: fmt.Sprintf("term %d", term), Pages: pages, Opens: term})
}
