package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// crisisFloor is what a characteristic reduced by aging may reach. E010
// reads p. 4's floor as 0 for aging alone: zero is the only value below one
// the rules give any meaning to, and pp. 7-8 give it a complete one.
const crisisFloor = 0

// agingRound runs the term's aging round, which is the last step of the term
// (E006) and is read off the table's term row rather than its age row.
//
// The saving throws are made in the table's own row order (E007), and a
// characteristic that reaches zero resolves its crisis inline, before the
// next characteristic is thrown for - p. 7 says the recovery "is made
// immediately".
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. Aging runs at the end
// of every term from the fourth, not only at 34. P. 7 ("and in 4 year
// increments thereafter"), p. 9's table (terms 4 through 14) and p. 25's
// example (rounds owed for terms 4 and 5) agree.
//
// Implements: A round at the end of term 4 and every term after [1:7, 9
// (E006)].
// Implements: Saving throws in the table's row order [1:9 (E007)].
// Implements: The example's batched aging is not followed [1:25 (E006)].
func (r *run) agingRound(term traveller.Term) error {
	effects := r.tables.Aging().At(term)
	if len(effects) == 0 {
		return nil
	}

	r.log.step(fmt.Sprintf("aging, end of term %d", term), "pp. 7-9")

	for _, effect := range effects {
		throw := roll(r.roll, effect.Saving, 0)
		seq := r.log.throw("aging, "+effect.Characteristic.String(), throw)

		if throw.succeeded {
			r.log.outcomef(seq, r.agingErrata(term), "%v holds", effect.Characteristic)

			continue
		}

		err := r.age(seq, term, effect.Characteristic, effect.Reduction)
		if err != nil {
			return err
		}

		if r.dead() {
			return nil
		}
	}

	return nil
}

// agingErrata names the readings every aging round rests on: where the round
// sits in the term, and the order its throws are made in. Past the table's
// last printed term, E014 joins them.
func (r *run) agingErrata(term traveller.Term) []traveller.Erratum {
	errata := []traveller.Erratum{traveller.E006, traveller.E007}

	if term > r.tables.Aging().LastPrintedTerm() {
		errata = append(errata, traveller.E014)
	}

	return errata
}

// age applies one reduction and resolves the crisis if it reaches zero. The
// service history keeps the reduction the table gave beside the two ends,
// since the floor under aging can hold it short (#409).
//
// Implements: A held change keeps the table's change [Generation record].
func (r *run) age(
	seq int, term traveller.Term, characteristic traveller.Characteristic, reduction int,
) error {
	before := r.profile[characteristic]

	r.profile = r.profile.AgeReduce(characteristic, reduction)

	after := r.profile[characteristic]

	errata := r.agingErrata(term)
	if before-reduction < traveller.MinCharacteristic {
		// Below 1 is where the two readings of p. 4's floor diverge: the
		// ordinary floor would have stopped here and no crisis would
		// follow, and E010 is what carries it to 0 instead.
		errata = append(errata, traveller.E010)
	}

	r.log.fact(seq, errata, traveller.AgedFact{
		Change: traveller.Change{Characteristic: characteristic, From: before, To: after, Delta: -reduction},
	})

	if after > crisisFloor {
		return nil
	}

	return r.crisis(term, characteristic)
}

// crisis resolves a characteristic reduced to zero (pp. 7-8): "A basic
// saving throw of 8+ applies (and may be modified by the expertise of
// attending medical personnel)."
//
// No modifier applies: generation has no attending personnel for the printed
// DM to refer to (E009). A failed throw is death, which the page does not
// state and E008 reads from "If the character survives".
//
// Implements: A characteristic at zero is a medical crisis, resolved inline
// [1:7–8 (E007)].
// Implements: Saving throw 8+, with no modifier during generation [1:7 (E009)].
// Implements: Survival recovers to 1 and adds 1D months [1:7–8].
// Implements: A failed crisis throw is death [1:7–8 (E008)].
func (r *run) crisis(term traveller.Term, characteristic traveller.Characteristic) error {
	crisis := r.tables.Aging().Crisis
	errata := []traveller.Erratum{traveller.E008, traveller.E009}

	// No step of its own: the crisis is resolved inside the aging round, and
	// a step line here would put the round's remaining saving throws under it.
	// The throw names the characteristic instead.
	throw := roll(r.roll, crisis.Saving, 0)
	seq := r.log.throw("medical crisis, "+characteristic.String()+" at zero", throw)

	if !throw.succeeded {
		r.departure = traveller.KilledByMedicalCrisis{Characteristic: characteristic}
		r.log.fact(seq, errata, traveller.DiedOfCrisisFact{Term: term, Characteristic: characteristic})

		return nil
	}

	r.profile = r.profile.Alter(characteristic, crisis.RecoversTo-r.profile[characteristic])

	// Every die is kept, not their sum: the record logs what was thrown, and
	// a sum written into a one-die event would read as a face no die has.
	faces := make([]int, 0, crisis.MonthsDice)
	months := 0

	for range crisis.MonthsDice {
		face := r.roll.Die()

		faces = append(faces, face)

		months += face
	}

	r.atAge = r.atAge.PlusMonths(months)

	monthSeq := r.log.dice("months of recovery", faces...)
	r.log.fact(monthSeq, errata, traveller.RecoveredFact{
		Characteristic: characteristic, To: crisis.RecoversTo, Months: months, Age: r.atAge,
	})

	return nil
}
