package traveller

// Event is one entry in the generation record: the ordered account of the whole
// generation, which serves audit — verify any character against Book 1 by
// walking the record — and narrative.
//
// It is a sum for the same reason the others are: each kind carries only its
// own fields, and a renderer that grows a further kind cannot forget to
// handle it.
type Event interface {
	Fold(cases EventCases) error
	Sequence() int
	sealedEvent()
}

// EventCases handles each kind of event.
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

	// Opens is the term this step opens, and zero for every step that opens
	// none, so a term a death ended is still read as a term served (#412).
	Opens Term
}

// ThrowEvent records one throw against a target: the dice as rolled, the
// modifier applied, the target, and whether it was met.
//
// Every throw here has something to succeed at. A roll with nothing to meet
// is a RollEvent, which is a separate case rather than this one with an empty
// target (#50). One type told apart by an empty target would write every
// targetless roll as "succeeded": true, which is a claim about nothing.
//
// The dice are kept, not only their sum, because the record is what an auditor
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
// succeeded [Generation record].
type RollEvent struct {
	Seq  int
	Step string
	Dice []int
	DM   int
}

// Total is the roll's result: the dice plus the modifier, which is the row
// the page is read at.
func (r RollEvent) Total() int { return total(r.Dice, r.DM) }

// Total is the throw's result: the dice plus the modifier.
func (t ThrowEvent) Total() int { return total(t.Dice, t.DM) }

// total is dice plus a modifier, the one sum a throw and a roll both take
// (#371).
func total(dice []int, modifier int) int {
	sum := modifier
	for _, die := range dice {
		sum += die
	}

	return sum
}

// ChoiceEvent records a choice point: which one, what the alternatives were,
// and what was chosen. Who answered is not the event's to say: a run has one
// answerer, and the record names it once.
type ChoiceEvent struct {
	Seq          int
	Point        ChoicePoint
	Alternatives []string
	Chosen       string
}

// OutcomeEvent records a consequence — a skill gained, a characteristic
// changed, a benefit, a death — and the throw, roll or choice that caused it.
//
// Because is the sequence number of the causing event, or zero where the
// consequence follows from the procedure itself rather than from an event.
type OutcomeEvent struct {
	Seq         int
	Because     int
	Description string
	Errata      []Erratum

	// Fact is what the outcome did, typed, wherever a fact about the character
	// comes from it: Character's folds read every one (#441), the service
	// history leaving out what mustering out applied. Its Description is then
	// its String (#412).
	Fact Fact
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
