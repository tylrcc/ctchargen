package chargen

import (
	"github.com/philoserf/ctchargen/internal/traveller"
)

// ServiceHistory is what the service gave and took, a term at a time, so a
// sheet can tell it the way the Jamison example does (pp. 24-25) without
// reading it back out of the log's prose (#401). It is empty for a civilian,
// who served none.
//
// It is not kept beside the log but read from it (#412): each entry is an
// outcome's typed fact, and each term is the step that opened it, so the
// history and the record are one fact stated once.
type ServiceHistory struct {
	// OnEntering is what the Rank and Service Skills box grants on entering
	// the service, before the first term's survival throw (p. 23, E005).
	OnEntering []traveller.Gain

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
	Gained []traveller.Gain

	// Aged is what the aging round at the end of the term took (pp. 7-9),
	// and Recovered each characteristic that reached zero and came back
	// from the crisis (pp. 7-8).
	Aged      []traveller.Change
	Recovered []traveller.Characteristic
}

// Raise is a commission or a promotion, the rank it conferred, and what the
// Rank and Service Skills box granted at that rank (p. 23), in the order
// granted.
type Raise struct {
	How    traveller.Advancement
	Rank   string
	Grants []traveller.Gain
}

// History is the service a term at a time, read from the log (#412). It is
// empty for a civilian, who served none.
//
// Implements: The service history records each term's rank, gains and aging
// [Sheet].
func (c *Character) History() ServiceHistory {
	var fold historyFold

	for _, event := range c.Events {
		// A fold over this codec cannot fail: every case only appends.
		_ = event.Fold(&fold)
	}

	return fold.out
}

// historyFold reads the service history out of the log: a term from the step
// that opens it, and each entry from the fact its outcome carries.
type historyFold struct{ out ServiceHistory }

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

	// A fold over this codec cannot fail either: every case only appends.
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
	h.serving().Raised = append(h.serving().Raised, Raise{How: f.How, Rank: f.Rank})

	return nil
}

func (h *historyFold) Aged(f traveller.AgedFact) error {
	h.serving().Aged = append(h.serving().Aged, f.Change)

	return nil
}

func (h *historyFold) Recovered(f traveller.RecoveredFact) error {
	h.serving().Recovered = append(h.serving().Recovered, f.Characteristic)

	return nil
}

func (*historyFold) Enlisted(traveller.EnlistedFact) error { return nil }

// A characteristic as rolled is the sheet's, not the service's, and expertise
// taken instead of a weapon is mustering out's, which the history leaves out
// as it leaves out every result applied there.
func (*historyFold) Rolled(traveller.RolledFact) error       { return nil }
func (*historyFold) Expertise(traveller.ExpertiseFact) error { return nil }

// gained files a gain where its source says the sheet tells it. A benefit
// is on the sheet's possessions and characteristics, not its service record.
func (h *historyFold) gained(gain traveller.Gain, source traveller.Source) {
	switch source {
	case traveller.OnEntering:
		h.out.OnEntering = append(h.out.OnEntering, gain)
	case traveller.AtRank:
		// A grant at a rank follows the raise that conferred it, in the same
		// term, so the term has one.
		raised := h.serving().Raised

		latest := &raised[len(raised)-1]

		latest.Grants = append(latest.Grants, gain)
	case traveller.InTerm:
		h.serving().Gained = append(h.serving().Gained, gain)
	case traveller.AtMusteringOut:
	}
}

// serving is the term being served, the last one opened. Only what happens
// inside a term carries a fact that asks for it, so there is always one.
func (h *historyFold) serving() *TermServed {
	return &h.out.Terms[len(h.out.Terms)-1]
}
