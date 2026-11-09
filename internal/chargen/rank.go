package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// commission attempts a commission, where the character is eligible for one
// (p. 6): "in order to be commissioned, a character must throw the stated
// number. DMs may apply to the throw. If a commission is achieved, the
// character acquires level 1 rank in his service."
//
// It is attempted once per term until achieved, and never by a draftee in
// his first term - p. 5: "Draftees are not eligible for commissions during
// their first term of service; they do become eligible during the second and
// subsequent terms of service if they reenlist."
//
// Implements: Draftees are not eligible for commission in the first term only
// [1:5].
// Implements: Commission: once per term until achieved [1:6, 10].
func (r *run) commission(term traveller.Term) error {
	throw, ok := r.service.Commission()
	if !ok || r.char.Rank.Commissioned() || (r.drafted() && term == 1) {
		return nil
	}

	attempt, err := r.decide.AttemptCommission()
	if err != nil {
		return fmt.Errorf("attempting a commission: %w", err)
	}

	if !attempt {
		return nil
	}

	made := roll(r.roll, throw.Target, throw.Modifier(r.char.Profile))
	seq := r.log.throw("commission", made)

	if !made.succeeded {
		r.log.outcomef(seq, nil, "not commissioned this term")

		return nil
	}

	return r.raise(seq, 1, r.tables.Eligibility().OnCommission, Commissioned)
}

// promote attempts a promotion (p. 6): "In the same term of service that he
// is commissioned, and in each subsequent term of service, a character may
// attempt to be promoted ... A character is eligible for one promotion per
// term of service."
//
// No throw is made at the top of the service's column: p. 6 defines the
// whole of a promotion's effect as advancing "to the next higher rank in his
// service", and where the column ends there is no next rank (E013). A throw
// that is not made consumes nothing.
//
// Implements: Promotion: the commissioning term and every term after [1:6, 10].
// Implements: No promotion at the top of the ranks table [1:6, 10 (E013)].
func (r *run) promote(term traveller.Term) error {
	throw, ok := r.service.Promotion()
	if !ok || !r.char.Rank.Commissioned() {
		return nil
	}

	if r.char.Rank >= r.service.MaxRank() {
		r.log.outcomef(0, []traveller.Erratum{traveller.E013},
			"already %s, the highest rank the %v print, so no promotion is thrown for in term %d",
			r.rankName(), r.service.Name, term)

		return nil
	}

	attempt, err := r.decide.AttemptPromotion()
	if err != nil {
		return fmt.Errorf("attempting a promotion: %w", err)
	}

	if !attempt {
		return nil
	}

	made := roll(r.roll, throw.Target, throw.Modifier(r.char.Profile))
	seq := r.log.throw("promotion", made)

	if !made.succeeded {
		r.log.outcomef(seq, nil, "not promoted this term")

		return nil
	}

	return r.raise(seq, r.char.Rank+1, r.tables.Eligibility().OnPromotion, Promoted)
}

// raise confers a rank, the skill eligibility that comes with it (p. 6), and
// whatever the Rank and Service Skills box grants at it (p. 23).
//
// Implements: A commission or promotion confers one skill eligibility [1:6].
// Implements: Rank-keyed grants, at the moment the rank is conferred [1:23].
func (r *run) raise(seq int, conferred traveller.Rank, eligibility int, how Advancement) error {
	r.char.Rank = conferred
	r.char.RankName = r.rankName()

	r.eligible += eligibility

	r.log.outcomef(seq, nil, "%v: %s", how, r.rankName())

	raised := r.serving().raised(Raise{How: how, Rank: r.rankName()})

	// No erratum is stamped here. E005 reads the timing of a service-wide
	// entry, which the page leaves unstated; a rank entry's moment is exact
	// on p. 23 - "as soon as he becomes eligible" - and the worked example
	// shows it, so this needed no reading.
	for _, result := range r.tables.GrantsAtRank(r.service.Name, conferred) {
		err := r.apply(result, seq, nil, &raised.Grants)
		if err != nil {
			return err
		}
	}

	return nil
}

// rankName is what the Table of Ranks calls the character's rank (p. 10),
// or "uncommissioned" where he holds none.
func (r *run) rankName() string {
	name, ok := r.service.RankName(r.char.Rank)
	if !ok {
		return "uncommissioned"
	}

	return name
}
