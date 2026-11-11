package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// serve runs terms until one of them ends the service.
//
// The loop has no bound, and needs none under dice: the reenlistment throw of
// p. 6 fails eventually and the survival throw of p. 5 ends things sooner
// than that, so a career terminates with probability 1. That is a contract on
// the Roller, not a property of this function - a roller whose every die is
// a 6 makes every 2D throw a 12, never fails a reenlistment, and this never
// returns, which is why the fixture that walks past the Aging Table's last
// column scripts a finite list of dice rather than an endless supply of 6s
// (#54).
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
//
// Implements: Per-term order is the exposition's [1:5–6 (E002)].
// Implements: The example's per-term order is not followed [1:24 (E002)].
// Implements: The dead character's age and term [1:5, 9 (E004)].
func (r *run) term(term traveller.Term) (bool, error) {
	r.record.opens(term, "pp. 5-7")

	r.terms = int(term)

	if !r.survive(term) {
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

	err = r.trainSkills(term)
	if err != nil {
		return false, err
	}

	// The term's four years are counted before the reenlistment throw, whose
	// outcome carries the age the term left him at.
	r.atAge = r.atAge.PlusYears(traveller.Years)

	intent, err := r.reenlist(term)
	if err != nil {
		return false, err
	}

	err = r.agingRound(term)
	if err != nil {
		return false, err
	}

	if r.dead() {
		return true, nil
	}

	if intent == traveller.Continue {
		return false, nil
	}

	return true, nil
}

// survive makes the term's survival throw. P. 5: "Failure to successfully
// achieve the survival throw results in death." The fatal term's four years
// are added here (E004); term counts the term itself.
//
// Implements: Survival throw; failure is death [1:5].
// Implements: The dead character's age and term [1:5, 9 (E004)].
func (r *run) survive(term traveller.Term) bool {
	survival := r.against(r.service.Survival)
	seq := r.record.throw("survival", survival)

	if survival.succeeded {
		r.record.outcomef(seq, nil, "survived term %d", term)

		return true
	}

	// The age is the character's own, advanced by the fatal term's four years
	// (E004), rather than recomputed from the term count: a medical crisis
	// survived in an earlier term put months on it that arithmetic drops.
	r.atAge = r.atAge.PlusYears(traveller.Years)
	r.departure = traveller.KilledBySurvivalThrow{}
	r.record.fact(seq, []traveller.Erratum{traveller.E004}, traveller.DiedFact{Term: term, Age: r.atAge})

	return false
}

// odds is a throw as a decider is shown it before the answer that leads to
// it: the printed target, and the DM the character's characteristics earn
// against it now.
func (r *run) odds(throw rules.Throw) traveller.Odds {
	return traveller.Odds{Target: throw.Target, DM: throw.Modifier(r.profile), From: throw.Met(r.profile)}
}

// against makes a throw the page prints with characteristic DMs: the target,
// the DM the character's characteristics earn now, and what earned it.
func (r *run) against(printed rules.Throw) throw {
	made := roll(r.roll, printed.Target, printed.Modifier(r.profile))

	made.from = printed.Met(r.profile)

	return made
}

// grantEligibility adds the term's skill eligibilities, per the Basic Skill
// Eligibility box (p. 6): two for the initial term, one per subsequent term.
// They are tallied as the term's, apart from what a commission or a
// promotion gives, so the skills question can say why the term has them.
//
// Implements: Two eligibilities for the initial term, one thereafter [1:6].
// Implements: The skills question says which roll of how many, and why
// [Choices].
func (r *run) grantEligibility(term traveller.Term) {
	if term == 1 {
		r.eligible.Term += r.tables.Eligibility().InitialTerm

		return
	}

	r.eligible.Term += r.tables.Eligibility().PerSubsequentTerm
}

// trainSkills spends every eligibility the character has accrued this term.
// P. 11: "must specify the table being consulted prior to the die throw."
//
// The tally is taken whole and cleared before the first roll: each roll is
// asked as one of the term's whole count, and nothing carries into the next
// term.
//
// The step says how many rolls the term gave and what gave them, from the
// same tally the rolls are spent from.
//
// Implements: The skills question says which roll of how many, and why
// [Choices].
// Implements: The skills step says how many rolls the term gave, and what
// gave them [Generation record].
func (r *run) trainSkills(term traveller.Term) error {
	earned := r.eligible

	r.eligible = traveller.Earned{}

	if earned.Rolls() == 0 {
		return nil
	}

	r.record.phase(term,
		fmt.Sprintf("skills and training, %d %s: %v", earned.Rolls(), rollsWord(earned.Rolls()), earned),
		"pp. 6, 11")

	for n := 1; n <= earned.Rolls(); n++ {
		err := r.trainOnce(traveller.SkillRoll{N: n, Of: earned})
		if err != nil {
			return err
		}
	}

	return nil
}

// trainOnce designates a table and rolls on it. Each table is offered with
// its six results from the service's column, so it is designated knowing
// what each result gives.
//
// The first designation after Education reaches the fourth table's threshold
// says so before the table is designated, pointing at the change that took it
// there, so the offer that has grown by a table is read as the page's doing.
//
// Implements: The skills table is designated before the die [1:11].
// Implements: A skills table offered shows its six results [Choices].
// Implements: The first designation that offers the fourth table says the
// Education that opened it [Generation record].
func (r *run) trainOnce(roll traveller.SkillRoll) error {
	if r.fourthTableOpenedBy != 0 {
		gate := r.tables.Education()
		r.record.outcomef(r.fourthTableOpenedBy, nil, "%v %d, so this designation offers the %v",
			gate.Characteristic, r.profile[gate.Characteristic], gate.Table)

		r.fourthTableOpenedBy = 0
	}

	tables := r.offeredTables()

	offered := make([]traveller.SkillTableOffer, 0, len(tables))
	for _, table := range tables {
		offered = append(offered, traveller.SkillTableOffer{
			Table:   table,
			Taken:   r.trained[table],
			Results: slices.Clone(r.service.Skills[table][:]),
		})
	}

	table, err := r.decide.SkillTable(offered, roll)
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

	seq := r.record.dice(table.String(), face)

	return r.apply(result, seq, []traveller.Erratum{traveller.E002}, traveller.InTerm)
}

// offeredTables is the three tables always available, and the fourth while
// the character's Education stands at 8 or greater (p. 11).
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. The Education 8+ gate
// is tested against the current value. P. 11: "Characters may distribute their
// rolls over the three tables (the four tables if the character is of education
// 8 or greater), but must specify the table being consulted prior to the die
// throw" — present tense, at the moment the table is designated, so an
// Education raised to 8 mid-career opens the fourth table from the next
// designation on.
//
// Implements: The fourth table needs Education 8+ [1:11].
func (r *run) offeredTables() []traveller.SkillTable {
	offered := make([]traveller.SkillTable, 0, len(traveller.SkillTables))

	for _, table := range traveller.SkillTables {
		if table == r.tables.Education().Table && !r.tables.Education().Open(r.profile) {
			continue
		}

		offered = append(offered, table)
	}

	return offered
}
