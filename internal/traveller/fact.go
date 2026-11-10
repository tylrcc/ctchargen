package traveller

import "fmt"

// Fact is what an outcome did to the character, typed, so that what a sheet
// tells - the service attempted (#397), what the service gave and took
// (#412), the characteristics and the skills, the benefits, the pension, the
// title, and the career's enlistment, rank, age and departure (#441) - is
// read from the generation record rather than kept beside it. An outcome's
// prose is its fact's String, so the two cannot say different things. A
// result applied at mustering out carries one too, naming its source, and the
// service history leaves it out; an outcome nothing is read from - a
// characteristic that holds, a benefit wasted or lost - carries none.
type Fact interface {
	Fold(cases FactCases) error
	String() string
	sealedFact()
}

// FactCases handles each kind of fact.
//
//nolint:interfacebloat // one method per kind of fact closes the set
type FactCases interface {
	Applied(f AppliedFact) error
	WeaponPicked(f WeaponPickedFact) error
	Raised(f RaisedFact) error
	Aged(f AgedFact) error
	Recovered(f RecoveredFact) error
	Enlisted(f EnlistedFact) error
	Rolled(f RolledFact) error
	Expertise(f ExpertiseFact) error
	Cash(f CashFact) error
	Passage(f PassageFact) error
	TravellersAid(f TravellersAidFact) error
	WeaponReceived(f WeaponReceivedFact) error
	ShipReceived(f ShipReceivedFact) error
	FreeTraderAgain(f FreeTraderAgainFact) error
	Pension(f PensionFact) error
	Titled(f TitledFact) error
	Started(f StartedFact) error
	Drafted(f DraftedFact) error
	Declined(f DeclinedFact) error
	Died(f DiedFact) error
	DiedOfCrisis(f DiedOfCrisisFact) error
	Retained(f RetainedFact) error
	Reenlisted(f ReenlistedFact) error
	Retired(f RetiredFact) error
	DeniedReenlistment(f DeniedReenlistmentFact) error
	Discharged(f DischargedFact) error
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

// RaisedFact is a commission or promotion and the rank it conferred (p. 6):
// its place in the Table of Ranks and what the table calls it (p. 10).
type RaisedFact struct {
	How  Advancement
	Rank Rank
	Name string
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
	_ = f.Gain.Fold(gainWordsFold{&text})

	return text
}

func (f WeaponPickedFact) String() string {
	return fmt.Sprintf("%v: %s-%d", f.Category, f.Skill.Name, f.Skill.Level)
}

func (f RaisedFact) String() string { return fmt.Sprintf("%v: %s", f.How, f.Name) }

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

	return fmt.Sprintf("rejected by the %v", f.Service)
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

// CashFact is a cash benefit from Mustering Out Table 2 (p. 9), and the
// total the character's cash benefits have reached with it.
type CashFact struct{ Cash, Total Credits }

func (f CashFact) String() string {
	return fmt.Sprintf("%v, bringing the total to %v", f.Cash, f.Total)
}

// PassageFact is a passage received as a benefit (p. 21).
type PassageFact struct{ Class PassageClass }

func (f PassageFact) String() string { return fmt.Sprintf("a %v", f.Class) }

// TravellersAidFact is membership of the Travellers' Aid Society achieved
// (p. 22). A second roll for it is wasted, and carries no fact.
type TravellersAidFact struct{}

func (TravellersAidFact) String() string { return "Travellers' Aid Society membership" }

// WeaponReceivedFact is a weapon received as a benefit (p. 22), the first of
// its kind or a different one taken for a repeat.
type WeaponReceivedFact struct{ Weapon WeaponName }

func (f WeaponReceivedFact) String() string { return fmt.Sprintf("a %s", f.Weapon) }

// Ship is a starship received as a mustering out benefit (pp. 22-23).
type Ship struct {
	Kind ShipKind

	// Tons is the hull the ship is built on (Book 2 pp. 18-19).
	Tons int

	// Years is the ship's age. A Free Trader arrives new and ages ten years
	// with each further receipt; a Scout ship's age the page does not give.
	Years int

	// PaymentYears is how many years of payments remain on a Free Trader.
	// A Scout ship is held without title and carries none.
	PaymentYears int
}

// ShipReceivedFact is a starship benefit received for the first time (pp.
// 22-23). A second scout ship is lost (p. 23), and carries no fact.
type ShipReceivedFact struct{ Ship Ship }

// FreeTraderAgainFact is the Free Trader received a further time, as it
// stands after: aged and paid down (p. 22). Only the Free Trader is received
// again, so the type says so rather than a flag on any ship.
type FreeTraderAgainFact struct{ Ship Ship }

func (f FreeTraderAgainFact) String() string {
	return fmt.Sprintf("the Free Trader again: %d years old, with %d years of payments left",
		f.Ship.Years, f.Ship.PaymentYears)
}

func (f ShipReceivedFact) String() string {
	return fmt.Sprintf("a %v", f.Ship.Kind)
}

// PensionFact is the annual retirement pay a service pays (p. 21).
type PensionFact struct{ Pay Credits }

func (f PensionFact) String() string { return fmt.Sprintf("retirement pay of %v a year", f.Pay) }

// TitledFact is a final Social Standing that confers a title (p. 5; Book 3
// p. 22), and whether the character assumed it. AtDeath is a character who
// died: he is assessed but not asked, so he assumed nothing (E011).
type TitledFact struct {
	SocialStanding int
	Title          Title
	Assumed        bool
	AtDeath        bool
}

func (f TitledFact) String() string {
	if f.AtDeath {
		return fmt.Sprintf("held a social standing of %d at death, which is %v", f.SocialStanding, f.Title)
	}

	assumed := "and it is not"
	if f.Assumed {
		assumed = "and it is"
	}

	return fmt.Sprintf("social standing %d confers %v, %s assumed", f.SocialStanding, f.Title, assumed)
}

// StartedFact is the six characteristics as p. 4 rolled them, written as a
// UPP, and the age every character starts at. Each characteristic is read
// from its own RolledFact; this is where the age starts.
type StartedFact struct {
	Profile Profile
	Age     Age
}

func (f StartedFact) String() string { return fmt.Sprintf("UPP %s at age %v", f.Profile.UPP(), f.Age) }

// DraftedFact is the service the draft put the character in (p. 5).
type DraftedFact struct{ Service ServiceName }

func (f DraftedFact) String() string { return fmt.Sprintf("drafted into the %v", f.Service) }

// DeclinedFact is the draft declined, which leaves a civilian (p. 5, E001).
type DeclinedFact struct{}

func (DeclinedFact) String() string { return "declined the draft, and remains a civilian" }

// DiedFact is a failed survival throw (p. 5): the term it ended and the age
// it left him at, its four years counted (E004).
type DiedFact struct {
	Term Term
	Age  Age
}

func (f DiedFact) String() string { return fmt.Sprintf("died in term %d, aged %v", f.Term, f.Age) }

// DiedOfCrisisFact is a failed medical crisis throw (pp. 7-8, E008): the term
// and the characteristic that reached zero.
type DiedOfCrisisFact struct {
	Term           Term
	Characteristic Characteristic
}

func (f DiedOfCrisisFact) String() string {
	return fmt.Sprintf("died of a medical crisis in term %d, %v having reached zero", f.Term, f.Characteristic)
}

// The reenlistment throw's five facts below each carry the age the term left
// the character at, its four years counted (p. 5).

// RetainedFact is a reenlistment throw of 12 exactly, which keeps the
// character for another term whatever he meant to do (p. 6).
type RetainedFact struct{ Age Age }

func (RetainedFact) String() string { return "a 12 exactly: the service requires another term" }

// ReenlistedFact is the character staying on for the term named.
type ReenlistedFact struct {
	For Term
	Age Age
}

func (f ReenlistedFact) String() string { return fmt.Sprintf("reenlisted for term %d", f.For) }

// denied is why a character left whom the reenlistment throw refused.
const denied = "reenlistment denied"

// RetiredFact is leaving at the end of the fifth term or later, which p. 21
// makes retirement whether he chose to go or the throw refused him; Denied
// says which.
type RetiredFact struct {
	After  Term
	Age    Age
	Denied bool
}

func (f RetiredFact) String() string {
	why := Retire.String()
	if f.Denied {
		why = denied
	}

	return fmt.Sprintf("left the service after term %d and is retired: %s", f.After, why)
}

// DeniedReenlistmentFact is a failed reenlistment throw before the fifth term ends
// (p. 6).
type DeniedReenlistmentFact struct {
	After Term
	Age   Age
}

func (f DeniedReenlistmentFact) String() string {
	return fmt.Sprintf("left the service after term %d: %s", f.After, denied)
}

// DischargedFact is leaving by choice before the fifth term ends (pp. 6-7).
type DischargedFact struct {
	After Term
	Age   Age
}

func (f DischargedFact) String() string {
	return fmt.Sprintf("left the service after term %d: %v", f.After, Discharge)
}

// gainWordsFold writes a gain as the record says it.
type gainWordsFold struct{ out *string }

func (g gainWordsFold) Skill(skill Skill) error {
	*g.out = fmt.Sprintf("%s-%d", skill.Name, skill.Level)

	return nil
}

func (g gainWordsFold) Change(change Change) error {
	*g.out = fmt.Sprintf("%v %+d, %d to %d", change.Characteristic, change.Delta, change.From, change.To)

	return nil
}

func (f AppliedFact) Fold(c FactCases) error            { return c.Applied(f) }
func (f WeaponPickedFact) Fold(c FactCases) error       { return c.WeaponPicked(f) }
func (f RaisedFact) Fold(c FactCases) error             { return c.Raised(f) }
func (f AgedFact) Fold(c FactCases) error               { return c.Aged(f) }
func (f RecoveredFact) Fold(c FactCases) error          { return c.Recovered(f) }
func (f EnlistedFact) Fold(c FactCases) error           { return c.Enlisted(f) }
func (f RolledFact) Fold(c FactCases) error             { return c.Rolled(f) }
func (f ExpertiseFact) Fold(c FactCases) error          { return c.Expertise(f) }
func (f CashFact) Fold(c FactCases) error               { return c.Cash(f) }
func (f PassageFact) Fold(c FactCases) error            { return c.Passage(f) }
func (f TravellersAidFact) Fold(c FactCases) error      { return c.TravellersAid(f) }
func (f WeaponReceivedFact) Fold(c FactCases) error     { return c.WeaponReceived(f) }
func (f ShipReceivedFact) Fold(c FactCases) error       { return c.ShipReceived(f) }
func (f FreeTraderAgainFact) Fold(c FactCases) error    { return c.FreeTraderAgain(f) }
func (f PensionFact) Fold(c FactCases) error            { return c.Pension(f) }
func (f TitledFact) Fold(c FactCases) error             { return c.Titled(f) }
func (f StartedFact) Fold(c FactCases) error            { return c.Started(f) }
func (f DraftedFact) Fold(c FactCases) error            { return c.Drafted(f) }
func (f DeclinedFact) Fold(c FactCases) error           { return c.Declined(f) }
func (f DiedFact) Fold(c FactCases) error               { return c.Died(f) }
func (f DiedOfCrisisFact) Fold(c FactCases) error       { return c.DiedOfCrisis(f) }
func (f RetainedFact) Fold(c FactCases) error           { return c.Retained(f) }
func (f ReenlistedFact) Fold(c FactCases) error         { return c.Reenlisted(f) }
func (f RetiredFact) Fold(c FactCases) error            { return c.Retired(f) }
func (f DeniedReenlistmentFact) Fold(c FactCases) error { return c.DeniedReenlistment(f) }
func (f DischargedFact) Fold(c FactCases) error         { return c.Discharged(f) }
func (StartedFact) sealedFact()                         {}
func (DraftedFact) sealedFact()                         {}
func (DeclinedFact) sealedFact()                        {}
func (DiedFact) sealedFact()                            {}
func (DiedOfCrisisFact) sealedFact()                    {}
func (RetainedFact) sealedFact()                        {}
func (ReenlistedFact) sealedFact()                      {}
func (RetiredFact) sealedFact()                         {}
func (DeniedReenlistmentFact) sealedFact()              {}
func (DischargedFact) sealedFact()                      {}
func (CashFact) sealedFact()                            {}
func (PassageFact) sealedFact()                         {}
func (TravellersAidFact) sealedFact()                   {}
func (WeaponReceivedFact) sealedFact()                  {}
func (ShipReceivedFact) sealedFact()                    {}
func (FreeTraderAgainFact) sealedFact()                 {}
func (PensionFact) sealedFact()                         {}
func (TitledFact) sealedFact()                          {}
func (EnlistedFact) sealedFact()                        {}
func (RolledFact) sealedFact()                          {}
func (ExpertiseFact) sealedFact()                       {}
func (AppliedFact) sealedFact()                         {}
func (WeaponPickedFact) sealedFact()                    {}
func (RaisedFact) sealedFact()                          {}
func (AgedFact) sealedFact()                            {}
func (RecoveredFact) sealedFact()                       {}
