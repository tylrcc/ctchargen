package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/traveller"
)

// The two term counts p. 7 prints: "A character may serve up to 7 terms
// voluntarily, and retire any time after the end of the 5th term."
const (
	lastVoluntaryTerm = 7
	firstRetiringTerm = 5
)

// forcedByTwelve is the reenlistment throw that overrides the character's
// own intent. P. 6: "If the throw is a 12 (exactly), the needs of the
// service require that the character serve another term, regardless of his
// desires."
const forcedByTwelve = 12

// reenlist makes the term's reenlistment throw and takes the character's
// intent, reporting the intent and whether another term was forced.
//
// The throw is made every term, whether or not the character means to stay
// (p. 6). It is made before the intent is asked because a failed throw and a
// 12 both settle the question, and a choice with one answer is not a choice.
func (r *run) reenlist(term traveller.Term) (traveller.Intent, bool, error) {
	attempt := roll(r.roll, r.service.Reenlist, 0)
	seq := r.log.throw("reenlistment", attempt)

	if attempt.dice[0]+attempt.dice[1] == forcedByTwelve {
		errata := []traveller.Erratum(nil)
		if term > lastVoluntaryTerm {
			// P. 7 grants "an additional term" in the singular and says
			// nothing about a second 12; E003 reads the rule as recurring.
			// A 12 at the end of term 7 is the case p. 7 prints in its own
			// words, so the reading is stamped only from term 8 on, which is
			// what ERRATA.md says it is stamped on. The roster's goldens
			// never reached term 7's 12 until #205, and this bound was one term low.
			errata = append(errata, traveller.E003)
		}

		r.log.outcomef(seq, errata, "a 12 exactly: the service requires another term")

		return traveller.Continue, true, nil
	}

	if !attempt.succeeded {
		r.depart(seq, term, traveller.ForcedOut{}, "reenlistment denied")

		return traveller.Discharge, false, nil
	}

	return r.intent(seq, term)
}

// intent asks what the character means to do, when more than one answer is
// legal.
func (r *run) intent(seq int, term traveller.Term) (traveller.Intent, bool, error) {
	offered := r.offeredIntents(term)

	if len(offered) == 1 {
		r.settle(seq, term, offered[0])

		return offered[0], false, nil
	}

	chosen, err := r.decide.ReenlistIntent(offered)
	if err != nil {
		return chosen, false, fmt.Errorf("deciding whether to reenlist: %w", err)
	}

	r.settle(seq, term, chosen)

	return chosen, false, nil
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
		r.log.outcomef(seq, nil, "reenlisted for term %d", term+1)

		return
	}

	// Discharged and not Retired: offeredIntents never offers Discharge at
	// or past the retiring term, and depart makes a departure at or past it
	// a retirement regardless.
	r.depart(seq, term, traveller.Discharged{}, intent.String())
}

// depart ends the service. P. 21 decides which departure it is by the term
// count rather than by how it came about, so a departure at or past the
// retiring term is a retirement whatever brought it on.
//
// earlyDeparture is what it is below that term, and it is passed in rather
// than inferred from why (#45): why is also the sentence the log prints, so
// choosing by comparing it against a literal would let a change of wording
// change the domain outcome. The type carries that invariant.
func (r *run) depart(
	seq int, term traveller.Term, earlyDeparture traveller.Departure, why string,
) {
	if term >= firstRetiringTerm {
		r.char.Departure = traveller.Retired{}
		r.log.outcomef(seq, nil, "left the service after term %d and is retired: %s", term, why)

		return
	}

	r.char.Departure = earlyDeparture

	r.log.outcomef(seq, nil, "left the service after term %d: %s", term, why)
}

// pension records the annual retirement pay, for a departure at the end of
// the fifth term or later from a service that pays one (p. 21).
func (r *run) pension() {
	if !r.service.PaysPension {
		return
	}

	// The term count is the table's to judge, not this function's. It said
	// "at least five terms" here as well, which is the pension table's own
	// reach restated - and a restated precondition is one that can come to
	// disagree with the thing it restates (#52). Pay says whether p. 21
	// prints a pension for a term at all.
	pay, tabled := r.tables.Retirement().Pay(r.char.Terms)
	if !tabled {
		return
	}

	r.char.Pension = pay
	r.log.outcomef(0, nil, "retirement pay of %v a year", r.char.Pension)
}
