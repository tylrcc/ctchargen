package traveller

import "fmt"

// Event is one entry in the generation record: the ordered log of the whole
// generation, which serves audit — verify any character against Book 1 by
// walking the log — and narrative.
//
// It is a sum for the same reason the others are: each kind carries only its
// own fields, and a renderer that grows a further kind cannot forget to
// handle it.
type Event interface {
	Fold(cases EventCases) error
	Sequence() int
	sealedEvent()
}

// EventCases handles each kind of log entry.
type EventCases interface {
	Step(e StepEvent) error
	Throw(e ThrowEvent) error
	Roll(e RollEvent) error
	Choice(e ChoiceEvent) error
	Outcome(e OutcomeEvent) error
}

// StepEvent records a procedure step being entered, with the pages that
// govern it.
type StepEvent struct {
	Seq   int
	Step  string
	Pages string
}

// ThrowEvent records one throw against a target: the dice as rolled, the
// modifier applied, the target, and whether it was met.
//
// Every throw here has something to succeed at. A roll with nothing to meet
// is a RollEvent, which is a separate case rather than this one with an empty
// target (#50). One type told apart by an empty target would write every
// targetless roll as "succeeded": true, which is a claim about nothing.
//
// The dice are kept, not only their sum, because the log is what an auditor
// walks against the page — and because a 12 exactly on the reenlistment
// throw is a different event from a 12 reached any other way (pp. 6-7).
type ThrowEvent struct {
	Seq       int
	Step      string
	Dice      []int
	DM        int
	Target    Target
	Succeeded bool
}

// RollEvent records a roll with nothing to meet: the draft die, a skills
// table, a mustering out table, the months of recovery.
//
// It keeps the modifier, because p. 9's two mustering out DMs are the only
// ones a one-die roll takes and a record printing the face alone would send
// an auditor to the wrong row. It has no target and no outcome, because
// there is nothing for the roll to succeed at.
//
// Implements: A roll with nothing to meet is not recorded as a throw that
// succeeded [Generation record] in The record.
type RollEvent struct {
	Seq  int
	Step string
	Dice []int
	DM   int
}

// Total is the roll's result: the dice plus the modifier, which is the row
// the page is read at.
func (r RollEvent) Total() int {
	total := r.DM
	for _, die := range r.Dice {
		total += die
	}

	return total
}

// Total is the throw's result: the dice plus the modifier.
func (t ThrowEvent) Total() int {
	total := t.DM
	for _, die := range t.Dice {
		total += die
	}

	return total
}

// DecidedBy names who answered a choice — the player, or the auto policy.
//
// It is not the chargen.Decider interface and must not be confused with it:
// that is the thing asked, this is which kind of thing answered.
type DecidedBy int

// The two who can answer.
const (
	ByPlayer DecidedBy = iota
	ByPolicy
)

// DecidedBys is both answerers, for iteration.
var DecidedBys = [...]DecidedBy{ByPlayer, ByPolicy}

func (d DecidedBy) String() string {
	switch d {
	case ByPlayer:
		return "player"
	case ByPolicy:
		return "policy"
	}

	return fmt.Sprintf("DecidedBy(%d)", int(d))
}

// ChoiceEvent records a choice point: which one, who answered, what the
// alternatives were, and what was chosen.
type ChoiceEvent struct {
	Seq          int
	Point        ChoicePoint
	By           DecidedBy
	Alternatives []string
	Chosen       string
}

// OutcomeEvent records a consequence — a skill gained, a characteristic
// changed, years elapsed, a death — and the throw, roll or choice that caused
// it.
//
// Because is the sequence number of the causing event, or zero where the
// consequence follows from the procedure itself rather than from an event.
type OutcomeEvent struct {
	Seq         int
	Because     int
	Description string
	Errata      []Erratum
}

func (e StepEvent) Fold(c EventCases) error    { return c.Step(e) }
func (t ThrowEvent) Fold(c EventCases) error   { return c.Throw(t) }
func (r RollEvent) Fold(c EventCases) error    { return c.Roll(r) }
func (e ChoiceEvent) Fold(c EventCases) error  { return c.Choice(e) }
func (e OutcomeEvent) Fold(c EventCases) error { return c.Outcome(e) }
func (e StepEvent) Sequence() int              { return e.Seq }
func (t ThrowEvent) Sequence() int             { return t.Seq }
func (r RollEvent) Sequence() int              { return r.Seq }
func (e ChoiceEvent) Sequence() int            { return e.Seq }
func (e OutcomeEvent) Sequence() int           { return e.Seq }
func (StepEvent) sealedEvent()                 {}
func (ThrowEvent) sealedEvent()                {}
func (RollEvent) sealedEvent()                 {}
func (ChoiceEvent) sealedEvent()               {}
func (OutcomeEvent) sealedEvent()              {}
