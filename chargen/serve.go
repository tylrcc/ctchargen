package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/traveller"
)

// serve runs terms until one of them ends the service.
//
// The loop has no bound, and needs none under dice: the reenlistment throw of
// p. 6 fails eventually and the survival throw of p. 5 ends things sooner
// than that, so a career terminates with probability 1. That is a contract on
// the Roller, not a property of this function - a roller that answers every
// 2D throw with 12 never fails a reenlistment and this never returns, which is
// why the fixture that walks past the Aging Table's last column scripts a
// finite list of dice rather than an endless supply of 12s (#54).
func (r *run) serve() error {
	for term := 1; ; term++ {
		done, err := r.term(traveller.Term(term))
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}
}

// term runs one term of service in the exposition's order (E002): survival,
// commission, promotion, skills, reenlistment, and then the aging round at
// the end of the term (E006). It reports whether the service ended.
func (r *run) term(term traveller.Term) (bool, error) {
	r.log.step(fmt.Sprintf("term %d", term), "pp. 5-7")

	r.char.Terms = int(term)

	if !r.survive(term) {
		// The fatal term counts, and its four years with it (E004).
		r.char.Age = r.char.Age.PlusYears(traveller.Years)
		r.char.Departure = traveller.KilledBySurvivalThrow{}
		r.dead = true

		return true, nil
	}

	err := r.commission(term)
	if err != nil {
		return false, err
	}

	err = r.promote(term)
	if err != nil {
		return false, err
	}

	r.grantEligibility(term)

	err = r.trainSkills()
	if err != nil {
		return false, err
	}

	intent, forced, err := r.reenlist(term)
	if err != nil {
		return false, err
	}

	r.char.Age = r.char.Age.PlusYears(traveller.Years)

	err = r.agingRound(term)
	if err != nil {
		return false, err
	}

	if r.dead {
		return true, nil
	}

	if forced || intent == traveller.Continue {
		return false, nil
	}

	return true, nil
}

// survive makes the term's survival throw. P. 5: "Failure to successfully
// achieve the survival throw results in death."
func (r *run) survive(term traveller.Term) bool {
	throw := roll(r.roll, r.service.Survival.Target, r.service.Survival.Modifier(r.char.Profile))
	seq := r.log.throw("survival", throw)

	if throw.succeeded {
		r.log.outcomef(seq, nil, "survived term %d", term)

		return true
	}

	// The age is the character's own, advanced by the fatal term's four years
	// (E004), rather than recomputed from the term count: a medical crisis
	// survived in an earlier term put months on it that arithmetic drops.
	r.log.outcomef(seq, []traveller.Erratum{traveller.E004},
		"died in term %d, aged %v", term, r.char.Age.PlusYears(traveller.Years))

	return false
}

// grantEligibility adds the term's skill eligibilities, per the Basic Skill
// Eligibility box (p. 6): two for the initial term, one per subsequent term.
func (r *run) grantEligibility(term traveller.Term) {
	if term == 1 {
		r.eligible += r.tables.Eligibility().InitialTerm

		return
	}

	r.eligible += r.tables.Eligibility().PerSubsequentTerm
}

// trainSkills spends every eligibility the character has accrued. P. 11:
// "must specify the table being consulted prior to the die throw."
func (r *run) trainSkills() error {
	if r.eligible == 0 {
		return nil
	}

	r.log.step("skills and training", "pp. 6, 11")

	for r.eligible > 0 {
		r.eligible--

		err := r.trainOnce()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *run) trainOnce() error {
	offered := r.offeredTables()

	taken := make([]int, len(offered))
	for i, table := range offered {
		taken[i] = r.trained[table]
	}

	table, err := r.decide.SkillTable(offered, taken)
	if err != nil {
		return fmt.Errorf("designating a skills table: %w", err)
	}

	if r.trained == nil {
		r.trained = make(map[traveller.SkillTable]int)
	}

	r.trained[table]++

	face := r.roll.Die()

	result, err := r.service.Result(table, face)
	if err != nil {
		return fmt.Errorf("consulting %v: %w", table, err)
	}

	seq := r.log.dice(table.String(), face)

	return r.apply(result, seq, []traveller.Erratum{traveller.E002})
}

// offeredTables is the three tables always available, and the fourth while
// the character's Education stands at 8 or greater (p. 11).
func (r *run) offeredTables() []traveller.SkillTable {
	offered := make([]traveller.SkillTable, 0, len(traveller.SkillTables))

	for _, table := range traveller.SkillTables {
		if table == r.tables.Education().Table && !r.tables.Education().Open(r.char.Profile) {
			continue
		}

		offered = append(offered, table)
	}

	return offered
}
