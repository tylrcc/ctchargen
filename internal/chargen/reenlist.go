package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The two term counts p. 7 prints: "A character may serve up to 7 terms
// voluntarily, and retire any time after the end of the 5th term."
const (
	// Settled by the page and not a reading: checked as a possible silence at
	// milestone 0, and kept so it is not re-litigated as one. A character may
	// voluntarily begin a seventh term but not an eighth. P. 7: "A character
	// may serve up to 7 terms voluntarily", and "Service beyond the seventh
	// term is normally impossible" — only a 12 carries him further (see E003).
	//
	// Implements: Seven terms of voluntary service [1:7].
	lastVoluntaryTerm = 7
	firstRetiringTerm = 5
)

// forcedByTwelve is the reenlistment throw that overrides the character's
// own intent. P. 6: "If the throw is a 12 (exactly), the needs of the
// service require that the character serve another term, regardless of his
// desires."
const forcedByTwelve = 12

// reenlist makes the term's reenlistment throw and takes the character's
// intent. A 12 exactly forces another term, which is the intent Continue.
//
// The throw is made every term, whether or not the character means to stay
// (p. 6). It is made before the intent is asked because a failed throw and a
// 12 both settle the question, and a choice with one answer is not a choice.
//
// Implements: Reenlistment throw every term, no DMs [1:6].
// Implements: A 12 exactly forces another term [1:6].
// Implements: A 12 recurs past the seventh term [1:6–7, 21 (E003)].
func (r *run) reenlist(term traveller.Term) (traveller.Intent, error) {
	attempt := roll(r.roll, r.service.Reenlist, 0)
	seq := r.record.throw("reenlistment", attempt)

	if attempt.dice[0]+attempt.dice[1] == forcedByTwelve {
		errata := []traveller.Erratum(nil)
		if term > lastVoluntaryTerm {
			// P. 7 grants "an additional term" in the singular and says
			// nothing about a second 12; E003 reads the rule as recurring.
			// A 12 at the end of term 7 is the case p. 7 prints in its own
			// words, so the reading is stamped only from term 8 on, which is
			// what E003's reading says it is stamped on.
			errata = append(errata, traveller.E003)
		}

		r.record.fact(seq, errata, traveller.RetainedFact{Age: r.atAge})

		return traveller.Continue, nil
	}

	if !attempt.succeeded {
		r.depart(seq, term, true)

		return traveller.Discharge, nil
	}

	return r.intent(seq, term)
}

// intent asks what the character means to do, when more than one answer is
// legal.
func (r *run) intent(seq int, term traveller.Term) (traveller.Intent, error) {
	offered := r.offeredIntents(term)

	if len(offered) == 1 {
		r.settle(seq, term, offered[0])

		return offered[0], nil
	}

	chosen, err := r.decide.ReenlistIntent(offered, r.odds(r.service.Survival))
	if err != nil {
		return chosen, fmt.Errorf("deciding whether to reenlist: %w", err)
	}

	r.settle(seq, term, chosen)

	return chosen, nil
}

// offeredIntents is what is legal at the end of a term.
//
// Discharge and Retire never both appear: p. 21 makes leaving at the end of
// the fifth term or later retirement by definition - "A character who leaves
// the service at the end of the 5th or later term of service is considered
// to have retired" - and p. 7 caps voluntary service at seven terms.
func (r *run) offeredIntents(term traveller.Term) []traveller.Intent {
	switch {
	case term >= lastVoluntaryTerm:
		return []traveller.Intent{traveller.Retire}
	case term >= firstRetiringTerm:
		return []traveller.Intent{traveller.Continue, traveller.Retire}
	default:
		return []traveller.Intent{traveller.Continue, traveller.Discharge}
	}
}

func (r *run) settle(seq int, term traveller.Term, intent traveller.Intent) {
	if intent == traveller.Continue {
		r.record.fact(seq, nil, traveller.ReenlistedFact{For: term + 1, Age: r.atAge})

		return
	}

	// Leaving by choice: offeredIntents never offers Discharge at or past the
	// retiring term, and depart makes a departure at or past it a retirement
	// regardless.
	r.depart(seq, term, false)
}

// depart ends the service. P. 21 decides which departure it is by the term
// count rather than by how it came about, so a departure at or past the
// retiring term is a retirement whatever brought it on.
//
// Below that term, denied - the reenlistment throw refused the character -
// makes it a denied reenlistment, and otherwise a discharge. It is passed in rather than inferred
// from the words the record prints (#45), so a change of wording cannot change
// the domain outcome; the words are the fact's.
//
// Implements: Leaving at the end of term 5 or later is retirement [1:21].
// Implements: A departure below term 5 is a denied reenlistment if the service
// refused it, a discharge if the character chose to go [1:6, 21].
func (r *run) depart(seq int, term traveller.Term, denied bool) {
	switch {
	case term >= firstRetiringTerm:
		r.departure = traveller.Retired{}
		r.record.fact(seq, nil, traveller.RetiredFact{After: term, Age: r.atAge, Denied: denied})
	case denied:
		r.departure = traveller.DeniedReenlistment{}
		r.record.fact(seq, nil, traveller.DeniedReenlistmentFact{After: term, Age: r.atAge})
	default:
		r.departure = traveller.Discharged{}
		r.record.fact(seq, nil, traveller.DischargedFact{After: term, Age: r.atAge})
	}
}

// pension records the annual retirement pay, for a departure at the end of
// the fifth term or later from a service that pays one (p. 21).
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. Retirement pay is
// earned by the term count, not by the manner of leaving. P. 21: "A character
// who leaves the service at the end of the 5th or later term of service is
// considered to have retired, and receives retirement pay" — so a character
// denied reenlistment by a failed throw at the end of term 5 or later draws
// it too. P. 21 also excludes the Scouts and Other services from retirement pay
// entirely, whatever the term count.
//
// Implements: Retirement pay from term 5, not for Scouts or Other [1:7, 21].
// Implements: Its ship, cash and pension [1:25].
func (r *run) pension() {
	if !r.service.PaysPension {
		return
	}

	// The term count is the table's to judge, not this function's. It said
	// "at least five terms" here as well, which is the pension table's own
	// reach restated - and a restated precondition is one that can come to
	// disagree with the thing it restates (#52). Pay says whether p. 21
	// prints a pension for a term at all.
	pay, tabled := r.tables.Retirement().Pay(r.terms)
	if !tabled {
		return
	}

	r.record.fact(0, nil, traveller.PensionFact{Pay: pay})
}
