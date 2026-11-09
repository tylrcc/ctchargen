package traveller

import "fmt"

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
	Skill(skill Skill) error
	Change(change Change) error
}

// SkillGain is a skill, at the level it reached.
type SkillGain struct{ Skill Skill }

// Change is a characteristic from what it was to what it became, and the
// change the table gave. Both are kept, because a floor or a ceiling can hold
// a characteristic where the table would move it (#409).
type Change struct {
	Characteristic Characteristic
	From, To       int
	Delta          int
}

// Fold hands the skill to its case.
func (s SkillGain) Fold(c GainCases) error { return c.Skill(s.Skill) }

// Fold hands the change to its case.
func (ch Change) Fold(c GainCases) error { return c.Change(ch) }

func (SkillGain) sealedGain() {}
func (Change) sealedGain()    {}
