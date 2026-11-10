package traveller

import "fmt"

// Fact is what an outcome did to the character, typed, so that what a sheet
// tells - the service attempted (#397), what the service gave and took
// (#412), the characteristics and the skills (#441) - is read from the
// generation record rather than kept beside it. An outcome's prose is its
// fact's String, so the two cannot say different things. A result applied at
// mustering out carries one too, naming its source, and the service history
// leaves it out; an outcome nothing is read from - a year elapsed, a
// benefit's cash - carries none.
type Fact interface {
	Fold(cases FactCases) error
	String() string
	sealedFact()
}

// FactCases handles each kind of fact.
type FactCases interface {
	Applied(f AppliedFact) error
	WeaponPicked(f WeaponPickedFact) error
	Raised(f RaisedFact) error
	Aged(f AgedFact) error
	Recovered(f RecoveredFact) error
	Enlisted(f EnlistedFact) error
	Rolled(f RolledFact) error
	Expertise(f ExpertiseFact) error
}

// Source is where an applied result came from, which is where a sheet tells
// it: on entering the service, at a rank, in a term's skill rolls, or among
// the benefits of mustering out.
type Source int

// The four.
const (
	OnEntering Source = iota
	AtRank
	InTerm
	AtMusteringOut
)

// AppliedFact is a table result applied: a skill at the level it reached, or
// a characteristic changed.
type AppliedFact struct {
	Gain   Gain
	Source Source
}

// WeaponPickedFact is a weapon named from a blade or gun list (pp. 12-13),
// at the level it reached.
type WeaponPickedFact struct {
	Category WeaponCategory
	Skill    Skill
	Source   Source
}

// RaisedFact is a commission or promotion and the rank it conferred (p. 6).
type RaisedFact struct {
	How  Advancement
	Rank string
}

// AgedFact is what an aging round took from one characteristic (pp. 7-9).
type AgedFact struct{ Change Change }

// RecoveredFact is a characteristic come back from a medical crisis, the
// months it took and the age it left him at (pp. 7-8).
type RecoveredFact struct {
	Characteristic Characteristic
	To, Months     int
	Age            Age
}

func (f AppliedFact) String() string {
	var text string

	// A fold over two cases that only format cannot fail.
	_ = f.Gain.Fold(gainText{&text})

	return text
}

func (f WeaponPickedFact) String() string {
	return fmt.Sprintf("%v: %s-%d", f.Category, f.Skill.Name, f.Skill.Level)
}

func (f RaisedFact) String() string { return fmt.Sprintf("%v: %s", f.How, f.Rank) }

func (f AgedFact) String() string {
	c := f.Change

	return fmt.Sprintf("%v -%d, %d to %d", c.Characteristic, -c.Delta, c.From, c.To)
}

// EnlistedFact is the enlistment throw's verdict on the service it was made
// against (p. 5), which is the service attempted, whatever the draft then
// gave (#397).
type EnlistedFact struct {
	Service  ServiceName
	Accepted bool
}

func (f EnlistedFact) String() string {
	if f.Accepted {
		return fmt.Sprintf("enlisted in the %v", f.Service)
	}

	return fmt.Sprintf("the %v rejected him", f.Service)
}

func (f RecoveredFact) String() string {
	return fmt.Sprintf("recovered: %v to %d, and %d months older, now aged %v",
		f.Characteristic, f.To, f.Months, f.Age)
}

// RolledFact is one characteristic as p. 4 rolled it, before anything moved
// it.
type RolledFact struct {
	Characteristic Characteristic
	Value          int
}

func (f RolledFact) String() string { return fmt.Sprintf("%v %d", f.Characteristic, f.Value) }

// ExpertiseFact is a repeat weapon benefit taken as expertise in that weapon
// instead (p. 22), at the level it reached.
type ExpertiseFact struct{ Skill Skill }

func (f ExpertiseFact) String() string {
	return fmt.Sprintf("expertise instead: %s-%d", f.Skill.Name, f.Skill.Level)
}

// gainText writes a gain as the record says it.
type gainText struct{ out *string }

func (g gainText) Skill(skill Skill) error {
	*g.out = fmt.Sprintf("%s-%d", skill.Name, skill.Level)

	return nil
}

func (g gainText) Change(change Change) error {
	*g.out = fmt.Sprintf("%v %+d, %d to %d", change.Characteristic, change.Delta, change.From, change.To)

	return nil
}

func (f AppliedFact) Fold(c FactCases) error      { return c.Applied(f) }
func (f WeaponPickedFact) Fold(c FactCases) error { return c.WeaponPicked(f) }
func (f RaisedFact) Fold(c FactCases) error       { return c.Raised(f) }
func (f AgedFact) Fold(c FactCases) error         { return c.Aged(f) }
func (f RecoveredFact) Fold(c FactCases) error    { return c.Recovered(f) }
func (f EnlistedFact) Fold(c FactCases) error     { return c.Enlisted(f) }
func (f RolledFact) Fold(c FactCases) error       { return c.Rolled(f) }
func (f ExpertiseFact) Fold(c FactCases) error    { return c.Expertise(f) }
func (EnlistedFact) sealedFact()                  {}
func (RolledFact) sealedFact()                    {}
func (ExpertiseFact) sealedFact()                 {}
func (AppliedFact) sealedFact()                   {}
func (WeaponPickedFact) sealedFact()              {}
func (RaisedFact) sealedFact()                    {}
func (AgedFact) sealedFact()                      {}
func (RecoveredFact) sealedFact()                 {}
