package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The two bounds the mustering procedure applies (p. 9). Neither indexes a
// printed table - one caps how many rolls may go on Table 2, the other is the
// rank at which the Table 1 modifier becomes available - so both are
// constants here beside the code that applies them rather than cells of the
// transcription (#43, and CLAUDE.md's Authority point 6).
const (
	maxRollsOnTableTwo       = 3
	minRankForTable1Modifier = 5
)

// musterOut spends the benefit rolls the service earned (pp. 7, 9).
//
// Implements: Its ship, cash and pension [1:25].
func (r *run) musterOut() error {
	rolls := r.tables.Muster().Rolls(r.char.Terms, r.char.Rank)
	if rolls == 0 {
		return nil
	}

	r.log.step(fmt.Sprintf("mustering out, %d rolls", rolls), "pp. 7, 9, 21-23")

	onTableTwo := 0

	for range rolls {
		table, err := r.chooseMusterTable(onTableTwo)
		if err != nil {
			return err
		}

		if table == traveller.TableTwo {
			onTableTwo++
		}

		err = r.benefit(table)
		if err != nil {
			return err
		}
	}

	return nil
}

// chooseMusterTable designates the table before the die. P. 9: "A maximum of
// three rolls on table 2 are allowed per character; all remaining rolls must
// be on table 1."
//
// Implements: The mustering-out table is designated before the die [1:9].
// Implements: At most three rolls on Table 2 [1:9].
func (r *run) chooseMusterTable(onTableTwo int) (traveller.MusterTable, error) {
	if onTableTwo >= maxRollsOnTableTwo {
		return traveller.TableOne, nil
	}

	// A window onto the exported array, handed over uncopied: the gate gives
	// every decider its own copy of what it is offered (#55, #278).
	table, err := r.decide.MusterTable(traveller.MusterTables[:])
	if err != nil {
		return table, fmt.Errorf("designating a mustering out table: %w", err)
	}

	return table, nil
}

// benefit takes one roll on the designated table.
func (r *run) benefit(table traveller.MusterTable) error {
	modifier, err := r.musterModifier(table)
	if err != nil {
		return err
	}

	face := r.roll.Die()

	row := face + modifier
	seq := r.log.dieWithModifier(table.String(), face, modifier)

	benefit, cash, err := r.service.Row(row)
	if err != nil {
		return fmt.Errorf("consulting %v: %w", table, err)
	}

	if table == traveller.TableTwo {
		r.benefits.Cash += cash
		r.log.fact(seq, nil, traveller.CashFact{Cash: cash, Total: r.benefits.Cash})

		return nil
	}

	applied := &applyBenefit{run: r, because: seq}

	err = benefit.Fold(applied)
	if err != nil {
		return fmt.Errorf("taking a benefit: %w", err)
	}

	return nil
}

// musterModifier offers the +1 each table allows, where the character has
// earned it (p. 9).
//
// Implements: The +1 at rank 5 or 6 on Table 1 [1:9].
// Implements: The +1 on Table 1 may be declined [1:9].
// Implements: The +1 with gambling on Table 2 [1:9].
func (r *run) musterModifier(table traveller.MusterTable) (int, error) {
	if table == traveller.TableOne {
		// P. 9: "Characters with rank 5 or 6 may add +1 to their rolls on
		// this table." The rank bounds the procedure and indexes no printed
		// table, so it is the constant declared at the top of this file.
		if int(r.char.Rank) < minRankForTable1Modifier {
			return 0, nil
		}

		take, err := r.decide.MusterTable1DM()
		if err != nil {
			return 0, fmt.Errorf("the table 1 modifier: %w", err)
		}

		if take {
			return r.tables.Muster().Table1DMFromRank5or6, nil
		}

		return 0, nil
	}

	if !r.has(r.tables.Muster().Table2ModifierFrom) {
		return 0, nil
	}

	take, err := r.decide.MusterTable2DM()
	if err != nil {
		return 0, fmt.Errorf("the table 2 modifier: %w", err)
	}

	if take {
		return r.tables.Muster().Table2DMFromGambling, nil
	}

	return 0, nil
}

// assessTitle assesses nobility once, at the end of generation, against the
// final Social Standing (E011). A character who died is assessed but not
// asked: eligibility is a condition of the value, while assuming a title is
// an act he does not perform.
//
// Implements: Social Standing 11+ may assume the hereditary title [1:5; 3:22].
// Implements: Assessed once, at the end, against the final value [1:5; 3:22
// (E011)].
// Implements: The dead are assessed but not asked [(E011)].
// Implements: E011 is stamped only where the reading changed the answer
// [(E011)].
func (r *run) assessTitle() error {
	social := r.profile[traveller.SocialStanding]

	rank, eligible := r.tables.TitleFor(social)

	_, eligibleAtEighteen := r.tables.TitleFor(r.rolledSocial)

	// The reading's two consequences, and the only records it governed:
	// an alteration conferred or removed a title he did not have or did
	// have at 18, or he died and is assessed anyway. A character eligible
	// at 18 and still eligible is assessed the same way under either
	// reading, so his record does not name a reading that did nothing to
	// it (#92).
	governed := eligible != eligibleAtEighteen || (r.dead() && eligible)

	errata := []traveller.Erratum(nil)
	if governed {
		errata = append(errata, traveller.E011)
	}

	if !eligible {
		// Eligible at 18 and not at the end: the reading took the title,
		// and a record that said nothing at all would be the one place it
		// mattered most and showed least.
		if governed {
			r.log.step("titles", "p. 5; Book 3 p. 22")
			r.log.outcomef(0, errata,
				"social standing %d at the end, from %d at 18: no title, where the rolled value allowed one",
				social, r.rolledSocial)
		}

		return nil
	}

	r.log.step("titles", "p. 5; Book 3 p. 22")

	if r.dead() {
		r.log.fact(0, errata, traveller.TitledFact{SocialStanding: social, Title: rank, AtDeath: true})

		return nil
	}

	assume, err := r.decide.AssumeTitle(rank)
	if err != nil {
		return fmt.Errorf("assuming a title: %w", err)
	}

	r.log.fact(0, errata, traveller.TitledFact{SocialStanding: social, Title: rank, Assumed: assume})

	return nil
}

// applyBenefit is the fold that takes one row of Mustering Out Table 1.
// Adding a seventh kind to the sum stops this compiling until it is handled.
//
// Implements: The six kinds of Table 1 row [1:9, 21–23].
type applyBenefit struct {
	run     *run
	because int
}

func (a *applyBenefit) Passage(class traveller.PassageClass) error {
	a.run.log.fact(a.because, nil, traveller.PassageFact{Class: class})

	return nil
}

func (a *applyBenefit) Alteration(characteristic traveller.Characteristic, delta int) error {
	return a.run.apply(traveller.AlterationResult{Characteristic: characteristic, Delta: delta},
		a.because, nil, traveller.AtMusteringOut)
}

func (a *applyBenefit) WeaponPick(category traveller.WeaponCategory) error {
	list := a.run.tables.Weapons(category)
	received := a.run.receivedIn(category)

	taken, err := a.run.decide.MusterWeapon(category, list, received)
	if err != nil {
		return fmt.Errorf("taking a weapon benefit: %w", err)
	}

	took := &takeWeapon{run: a.run, because: a.because}

	err = taken.Fold(took)
	if err != nil {
		return fmt.Errorf("taking a weapon benefit: %w", err)
	}

	return nil
}

// Implements: Travellers' Aid only once; duplicates wasted [1:22].
func (a *applyBenefit) TravellersAid() error {
	// P. 22: membership "may be achieved only once per character. If a die
	// roll indicates membership after it has already been achieved, the die
	// roll is wasted, and the character receives nothing for it."
	if a.run.benefits.TravellersAid {
		a.run.log.outcomef(a.because, nil, "Travellers' Aid again, which is wasted")

		return nil
	}

	a.run.benefits.TravellersAid = true
	a.run.log.fact(a.because, nil, traveller.TravellersAidFact{})

	return nil
}

func (a *applyBenefit) Ship(kind traveller.ShipKind) error {
	return a.run.receiveShip(a.because, kind)
}

func (a *applyBenefit) Nothing() error {
	a.run.log.outcomef(a.because, nil, "nothing: the table prints a dash for this service")

	return nil
}

// receivedIn is the weapons of one category already taken as benefits. P. 22
// bounds the expertise option twice: only "in a weapon received as a
// benefit", and only in lieu of one "of exactly the same type".
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. Mustering-out
// expertise stays inside the benefit's own category. P. 22: a repeat weapon
// benefit may be taken as "+1 expertise in lieu of receiving a second or
// subsequent weapon of exactly the same type", and "Expertise may only be taken
// in a weapon received as a benefit." The expertise substitutes for the weapon
// that benefit would have delivered, so it is taken in a weapon of that
// benefit's category — the page's worked example never leaves one (blade →
// cutlass → cutlass-1). A gun benefit cannot be converted into expertise in a
// blade.
func (r *run) receivedIn(category traveller.WeaponCategory) []traveller.WeaponName {
	list := r.tables.Weapons(category)

	var received []traveller.WeaponName

	for _, held := range r.benefits.Weapons {
		if slices.Contains(list, held) && !slices.Contains(received, held) {
			received = append(received, held)
		}
	}

	return received
}

// takeWeapon is the fold that records what a repeat weapon row was taken as.
//
// Neither case re-checks the answer against the list it came from: the
// logging decorator refuses an answer outside the offered set before the
// engine sees it, and one definition of that cannot disagree with itself.
//
// Implements: A repeat weapon may be taken as expertise, or as a different
// weapon [1:22].
type takeWeapon struct {
	run     *run
	because int
}

func (t *takeWeapon) TakeWeapon(weapon traveller.WeaponName) error {
	t.run.benefits.Weapons = append(t.run.benefits.Weapons, weapon)
	t.run.log.fact(t.because, nil, traveller.WeaponReceivedFact{Weapon: weapon})

	return nil
}

func (t *takeWeapon) TakeExpertise(weapon traveller.WeaponName) error {
	level := t.run.addSkill(traveller.SkillName(weapon))
	t.run.log.fact(t.because, nil, traveller.ExpertiseFact{
		Skill: traveller.Skill{Name: traveller.SkillName(weapon), Level: level},
	})

	return nil
}

// The Free Trader's terms, from p. 22: the ship is received "liable for the
// monthly payments (which amount to about 150,000 credits) for the next
// forty years", and each further receipt "is considered to represent actual
// possession of the ship for a ten-year period. The ship is 10 years older,
// and the total payment term is reduced by ten years."
const (
	freeTraderPaymentYears = 40
	freeTraderYearsPerRoll = 10
)

// receiveShip takes a starship benefit (pp. 22-23).
func (r *run) receiveShip(because int, kind traveller.ShipKind) error {
	for i := range r.benefits.Ships {
		if r.benefits.Ships[i].Kind == kind {
			return r.receiveShipAgain(because, i)
		}
	}

	ship := traveller.Ship{Kind: kind, Tons: r.tables.Hull(kind)}
	if kind == traveller.FreeTrader {
		ship.PaymentYears = freeTraderPaymentYears
	}

	r.benefits.Ships = append(r.benefits.Ships, ship)
	r.log.fact(because, nil, traveller.ShipReceivedFact{Ship: ship})

	return nil
}

// receiveShipAgain applies a duplicate. A Free Trader's repeats pay the ship
// off ten years at a time and age it by as much; a Scout ship's are lost -
// p. 23: "Only one scout ship may be acquired by a character, and throws
// resulting in additional ships are lost."
//
// Implements: Free Trader: 40 years of payments, 10 off per repeat [1:22–23].
// Implements: Scout ship: duplicates lost [1:23].
func (r *run) receiveShipAgain(because, at int) error {
	ship := &r.benefits.Ships[at]

	if ship.Kind == traveller.ScoutShip {
		r.log.outcomef(because, nil, "a second scout ship, which is lost")

		return nil
	}

	ship.Years += freeTraderYearsPerRoll

	ship.PaymentYears = max(0, ship.PaymentYears-freeTraderYearsPerRoll)

	r.log.fact(because, nil, traveller.FreeTraderAgainFact{Ship: *ship})

	return nil
}
