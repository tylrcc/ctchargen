package chargen

import (
	"github.com/philoserf/ctchargen/internal/traveller"
)

// ServiceHistory is what the service gave and took, a term at a time, so a
// sheet can tell it the way the Jamison example does (pp. 24-25) without
// reading it back out of the record's prose (#401). It is empty for a civilian,
// who served none.
//
// It is not kept beside the record but read from it (#412): each entry is an
// outcome's typed fact, and each term is the step that opened it, so the
// history and the record are one fact stated once.
type ServiceHistory struct {
	// OnEntering is what the Rank and Service Skills box grants on entering
	// the service, before the first term's survival throw (p. 23, E005).
	OnEntering []Moved

	Terms []TermServed

	// RetiredDenied is a retirement the reenlistment throw refused: p. 21
	// makes leaving at the end of the fifth term or later retirement either
	// way, and the record says which (#561). It stays set when a crisis then
	// kills him, and the death is the departure the sheet tells.
	RetiredDenied bool

	// MusteredOut is what mustering out changed in the character, in the
	// order rolled: a characteristic a Table 1 row altered (p. 9), "applied
	// to the character immediately" (p. 23, #490), and expertise taken
	// instead of a repeat weapon (p. 22, #561).
	MusteredOut []Moved
}

// Moved is a gain and the value it moved its skill or characteristic from:
// the level the skill was held at before it, zero where he had none, or the
// characteristic's value before the change. A sheet tells a skill's gains as
// the level it went from and the level it reached (#562), and the gain
// carries only the second.
type Moved struct {
	Gain traveller.Gain
	From int
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
	Gained []Moved

	// Retained is a 12 exactly on the term's reenlistment throw, which
	// required another term "regardless of his desires" (p. 6, #561).
	Retained bool

	// Aged is what the aging round at the end of the term took (pp. 7-9),
	// and Recovered each characteristic that reached zero and came back
	// from the crisis (pp. 7-8).
	Aged      []traveller.Change
	Recovered []Recovery
}

// Recovery is a characteristic come back from a medical crisis, the value it
// came back to and the months of age the recovery added (pp. 7-8, #561).
type Recovery struct {
	Characteristic traveller.Characteristic
	To, Months     int
}

// Raise is a commission or a promotion, the rank it conferred, and what the
// Rank and Service Skills box granted at that rank (p. 23), in the order
// granted.
type Raise struct {
	How    traveller.Advancement
	Rank   string
	Grants []Moved
}

// History is the service a term at a time, read from the record (#412). It is
// empty for a civilian, who served none.
//
// Implements: The service history records each term's rank, gains and aging
// [Sheet].
func (c *Character) History() ServiceHistory {
	fold := historyFold{levels: map[traveller.SkillName]int{}}

	for _, event := range c.Events {
		// This fold cannot fail: every case only appends.
		_ = event.Fold(&fold)
	}

	return fold.out
}

// historyFold reads the service history out of the record: a term from the step
// that opens it, and each entry from the fact its outcome carries. Its levels
// are each skill at the level the last fact naming it reached.
type historyFold struct {
	out    ServiceHistory
	levels map[traveller.SkillName]int
}

func (h *historyFold) Step(e traveller.StepEvent) error {
	if e.Opens > 0 {
		h.out.Terms = append(h.out.Terms, TermServed{Term: e.Opens})
	}

	return nil
}

func (*historyFold) Throw(traveller.ThrowEvent) error   { return nil }
func (*historyFold) Roll(traveller.RollEvent) error     { return nil }
func (*historyFold) Choice(traveller.ChoiceEvent) error { return nil }

func (h *historyFold) Outcome(e traveller.OutcomeEvent) error {
	if e.Fact == nil {
		return nil
	}

	// This fold cannot fail either: every case only appends.
	_ = e.Fact.Fold(h)

	return nil
}

func (h *historyFold) Applied(f traveller.AppliedFact) error {
	h.gained(f.Gain, f.Source)

	return nil
}

func (h *historyFold) WeaponPicked(f traveller.WeaponPickedFact) error {
	h.gained(traveller.SkillGain{Skill: f.Skill}, f.Source)

	return nil
}

func (h *historyFold) Raised(f traveller.RaisedFact) error {
	h.serving().Raised = append(h.serving().Raised, Raise{How: f.How, Rank: f.Name})

	return nil
}

func (h *historyFold) Aged(f traveller.AgedFact) error {
	h.serving().Aged = append(h.serving().Aged, f.Change)

	return nil
}

func (h *historyFold) Recovered(f traveller.RecoveredFact) error {
	h.serving().Recovered = append(h.serving().Recovered, Recovery{
		Characteristic: f.Characteristic, To: f.To, Months: f.Months,
	})

	return nil
}

// Expertise taken instead of a weapon is a mustering-out gain: its skill is
// on the sheet's skills, and nowhere else would the sheet say where it came
// from (#561).
//
// Implements: Expertise taken at mustering out is on the service record
// [Sheet].
func (h *historyFold) Expertise(f traveller.ExpertiseFact) error {
	h.gained(traveller.SkillGain(f), traveller.AtMusteringOut)

	return nil
}

func (h *historyFold) Retained(traveller.RetainedFact) error {
	h.serving().Retained = true

	return nil
}

func (h *historyFold) Retired(f traveller.RetiredFact) error {
	h.out.RetiredDenied = f.Denied

	return nil
}

func (*historyFold) Enlisted(traveller.EnlistedFact) error { return nil }

// A characteristic as rolled is the sheet's, not the service's.
func (*historyFold) Rolled(traveller.RolledFact) error { return nil }

// What mustering out delivered and the title assessed at the end are the
// sheet's benefits, not the service's.
func (*historyFold) Cash(traveller.CashFact) error                       { return nil }
func (*historyFold) Passage(traveller.PassageFact) error                 { return nil }
func (*historyFold) TravellersAid(traveller.TravellersAidFact) error     { return nil }
func (*historyFold) WeaponReceived(traveller.WeaponReceivedFact) error   { return nil }
func (*historyFold) ShipReceived(traveller.ShipReceivedFact) error       { return nil }
func (*historyFold) FreeTraderAgain(traveller.FreeTraderAgainFact) error { return nil }
func (*historyFold) Pension(traveller.PensionFact) error                 { return nil }
func (*historyFold) Titled(traveller.TitledFact) error                   { return nil }

// How the character entered, the age each term ended at and how the service
// ended are the sheet's to tell; a term is read from the step that opens it.
func (*historyFold) Started(traveller.StartedFact) error                       { return nil }
func (*historyFold) Drafted(traveller.DraftedFact) error                       { return nil }
func (*historyFold) Declined(traveller.DeclinedFact) error                     { return nil }
func (*historyFold) Died(traveller.DiedFact) error                             { return nil }
func (*historyFold) DiedOfCrisis(traveller.DiedOfCrisisFact) error             { return nil }
func (*historyFold) Reenlisted(traveller.ReenlistedFact) error                 { return nil }
func (*historyFold) DeniedReenlistment(traveller.DeniedReenlistmentFact) error { return nil }
func (*historyFold) Discharged(traveller.DischargedFact) error                 { return nil }

// gained files a gain where its source says the sheet tells it: on entering,
// at a rank, in a term, or at mustering out, with the value it moved from. A
// skill's is the level the last fact naming that skill reached, read from
// the record and not from the rule that raises it, so each fact that
// acquires a skill must pass through here.
//
// Implements: The service history keeps the level each skill gain moved from
// [Sheet].
func (h *historyFold) gained(gain traveller.Gain, source traveller.Source) {
	from := fromFold{levels: h.levels}

	// This fold cannot fail: every case only assigns.
	_ = gain.Fold(&from)

	moved := Moved{Gain: gain, From: from.from}

	switch source {
	case traveller.OnEntering:
		h.out.OnEntering = append(h.out.OnEntering, moved)
	case traveller.AtRank:
		// A grant at a rank follows the raise that conferred it, in the same
		// term, so the term has one.
		raised := h.serving().Raised

		latest := &raised[len(raised)-1]

		latest.Grants = append(latest.Grants, moved)
	case traveller.InTerm:
		h.serving().Gained = append(h.serving().Gained, moved)
	case traveller.AtMusteringOut:
		h.out.MusteredOut = append(h.out.MusteredOut, moved)
	}
}

// fromFold reads the value a gain moved from, and keeps a skill's new level
// for the next gain of it. Adding a kind of gain stops this compiling until
// it has one.
type fromFold struct {
	levels map[traveller.SkillName]int
	from   int
}

func (f *fromFold) Skill(skill traveller.Skill) error {
	f.from = f.levels[skill.Name]
	f.levels[skill.Name] = skill.Level

	return nil
}

func (f *fromFold) Change(change traveller.Change) error {
	f.from = change.From

	return nil
}

// serving is the term being served, the last one opened. Only what happens
// inside a term carries a fact that asks for it, so there is always one.
func (h *historyFold) serving() *TermServed {
	return &h.out.Terms[len(h.out.Terms)-1]
}
