package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// ServiceHistory is what the service gave and took, a term at a time, so a
// sheet can tell it the way the Jamison example does (pp. 24-25) without
// reading it back out of the log's prose (#401). It is empty for a civilian,
// who served none.
//
// Every entry in it is also an outcome in the log, which says the same thing
// in prose for a reader walking the page. The log is the audit and this is
// what a sheet is written from. What holds the two to each other is a test
// whose expected histories are typed from three fixtures' logs.
type ServiceHistory struct {
	// OnEntering is what the Rank and Service Skills box grants on entering
	// the service, before the first term's survival throw (p. 23, E005).
	OnEntering []Gain

	Terms []TermServed
}

// TermServed is one term of service. The term a survival throw ended has
// nothing in it but its number.
type TermServed struct {
	Term traveller.Term

	// Raised is the term's commission and promotion, in that order; p. 6
	// allows both in one term.
	Raised []Raise

	// Gained is what the term's skill eligibilities gave, in the order
	// they were rolled (pp. 6, 11).
	Gained []Gain

	// Aged is what the aging round at the end of the term took (pp. 7-9),
	// and Recovered each characteristic that reached zero and came back
	// from the crisis (pp. 7-8).
	Aged      []Change
	Recovered []traveller.Characteristic
}

// raised records a commission or promotion conferred this term, and returns
// it, so the grants at its rank can be recorded on it.
//
// Implements: The service history records each term's rank, gains and aging
// [Sheet].
func (t *TermServed) raised(raise Raise) *Raise {
	t.Raised = append(t.Raised, raise)

	return &t.Raised[len(t.Raised)-1]
}

// aged records what the term's aging round took from one characteristic.
//
// Implements: The service history records each term's rank, gains and aging
// [Sheet].
func (t *TermServed) aged(change Change) { t.Aged = append(t.Aged, change) }

// recovered records a characteristic that came back from a medical crisis.
//
// Implements: The service history records each term's rank, gains and aging
// [Sheet].
func (t *TermServed) recovered(characteristic traveller.Characteristic) {
	t.Recovered = append(t.Recovered, characteristic)
}

// Raise is a commission or a promotion, the rank it conferred, and what the
// Rank and Service Skills box granted at that rank (p. 23), in the order
// granted.
type Raise struct {
	How    Advancement
	Rank   string
	Grants []Gain
}

// Advancement is how a rank was conferred: a commission, which confers the
// first (p. 6), or a promotion, which advances "to the next higher rank in his
// service" (p. 6).
type Advancement int

// The two (p. 6).
const (
	Commissioned Advancement = iota
	Promoted
)

func (a Advancement) String() string {
	switch a {
	case Commissioned:
		return "commissioned"
	case Promoted:
		return "promoted"
	}

	return fmt.Sprintf("Advancement(%d)", int(a))
}

// Gain is one thing a source gave: a skill at the level it reached, or a
// characteristic changed. Exactly one of the two, and a list of them keeps
// the order they were received in, which a list of each would lose.
type Gain interface {
	Fold(cases GainCases) error
	sealedGain()
}

// GainCases handles each kind of gain.
type GainCases interface {
	Skill(skill traveller.Skill) error
	Change(change Change) error
}

// SkillGain is a skill, at the level it reached.
type SkillGain struct{ Skill traveller.Skill }

// Change is a characteristic from what it was to what it became. Both ends
// are kept rather than a delta, because a floor or a ceiling can make the two
// differ from the result printed in the table.
type Change struct {
	Characteristic traveller.Characteristic
	From, To       int
}

// Fold hands the skill to its case.
func (s SkillGain) Fold(c GainCases) error { return c.Skill(s.Skill) }

// Fold hands the change to its case.
func (ch Change) Fold(c GainCases) error { return c.Change(ch) }

func (SkillGain) sealedGain() {}
func (Change) sealedGain()    {}
