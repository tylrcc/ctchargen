package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// E003 reads p. 7's "an additional term" as recurring, and E014 reads the
// Aging Table's last column as terminal. Both turn on a run of 12s no random
// career produces, and both are load-bearing: without the first the career
// stops at term 8, and without the second there is no row to read for term
// 15. The fixture that walks it is on the roster, so its golden carries the
// stamps and this test holds the arithmetic.
//
// Holds: A term is four years.
// Holds: A 12 exactly forces another term.
// Holds: A 12 recurs past the seventh term.
// Holds: The last column is terminal.
func TestACareerPastTheTablesLastColumn(t *testing.T) {
	t.Parallel()

	character := byName(t, "other-past-the-table").generate(t)

	const pastTheTable = 15

	if character.Terms < pastTheTable {
		t.Fatalf("served %d terms; the Aging Table's last printed term is 14, so this reaches nothing new",
			character.Terms)
	}

	stamped := make(map[traveller.Erratum]bool)
	for _, erratum := range character.Errata {
		stamped[erratum] = true
	}

	for _, want := range []traveller.Erratum{traveller.E003, traveller.E014} {
		if !stamped[want] {
			t.Errorf("a career of %d terms did not stamp %v", character.Terms, want)
		}
	}

	// Age is 18 plus four years a term (p. 5, p. 9's note). That holds for
	// this career because it survives no medical crisis, whose months would
	// go on top (pp. 7-8, E004).
	const startingAge = 18

	if want := startingAge + traveller.Years*character.Terms; character.Age.Years() != want {
		t.Errorf("age %v after %d terms, want %d", character.Age, character.Terms, want)
	}
}

// A voluntary career stops at seven terms. P. 7: "A character may serve up
// to 7 terms voluntarily, and retire any time after the end of the 5th
// term", and "Service beyond the seventh term is normally impossible".
//
// Holds: Seven terms of voluntary service.
// Holds: Leaving at the end of term 5 or later is retirement.
func TestVoluntaryServiceStopsAtSeven(t *testing.T) {
	t.Parallel()

	inputs := chargen.Inputs{
		Service: traveller.Other, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	// Twelves for enlistment and the first term's survival, then throws that
	// make every reenlistment without ever rolling the 12 that forces one.
	character, err := chargen.Generate(inputs, chargen.DefaultPolicy(), chargen.WithRoller(&alwaysNine{}))
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	const voluntaryCap = 7

	if character.Terms != voluntaryCap {
		t.Errorf("served %d terms voluntarily, want %d", character.Terms, voluntaryCap)
	}

	if _, retired := character.Departure.(traveller.Retired); !retired {
		t.Errorf("left after seven terms as %T; p. 21 makes that retirement", character.Departure)
	}
}

// alwaysNine makes every throw a 9: enough for every target Other prints,
// and never the 12 that forces another term.
type alwaysNine struct{}

func (alwaysNine) Die() int            { return 1 }
func (alwaysNine) Among(int) int       { return 0 }
func (alwaysNine) TwoDice() (int, int) { return 4, 5 }

// answeringOutsideTheOffer is a decider that reaches past what the procedure
// put in front of it. Everything else it defers to the policy.
type answeringOutsideTheOffer struct {
	chargen.Decider

	table     bool
	expertise bool
}

func (a answeringOutsideTheOffer) SkillTable(
	from []traveller.SkillTable, taken []int,
) (traveller.SkillTable, error) {
	if a.table {
		// P. 11 opens this table only at Education 8, and the character
		// generated below stands at 7, so it is not among `from`.
		return traveller.AdvancedEducationEight, nil
	}

	return a.Decider.SkillTable(from, taken) //nolint:wrapcheck // the policy's own error, unchanged
}

func (a answeringOutsideTheOffer) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	if a.expertise && len(received) == 0 {
		// P. 22 offers the expertise only in a weapon already received as a
		// benefit, and this character has received none of this category.
		return traveller.TakeExpertise{Weapon: from[0]}, nil
	}

	return a.Decider.MusterWeapon(category, from, received) //nolint:wrapcheck // the policy's own error, unchanged
}

// A decider that answers outside the offered set is refused, not obeyed.
//
// The engine offers exactly what the page allows; applying an answer from
// outside that set builds a character the book does not permit - one trained
// on a table his Education closes to him, or holding expertise in a weapon he
// never received. The mutation that found this had the interactive loop
// designate a closed table, and the engine consulted it.
//
// Holds: An answer outside the offered set is refused, not applied.
func TestAnAnswerOutsideTheOfferIsRefused(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		fixture string
		decider answeringOutsideTheOffer
	}{
		// other-serve stands at Education 7 for the whole career, so p. 11
		// never opens the fourth table to him.
		"a skills table his Education closes": {
			fixture: "other-serve",
			decider: answeringOutsideTheOffer{Decider: chargen.DefaultPolicy(), table: true},
		},
		// scouts-expertise draws weapon benefits, so the choice point is
		// reached - and the first time it is, nothing of that category has
		// been received yet.
		"expertise in a weapon never received": {
			fixture: "scouts-expertise",
			decider: answeringOutsideTheOffer{Decider: chargen.DefaultPolicy(), expertise: true},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := byName(t, tc.fixture)

			_, err := chargen.Generate(f.inputs(), tc.decider, chargen.WithRoller(f.roller(t)))
			if err == nil {
				t.Fatal("the engine applied an answer it had not offered")
			}

			if !strings.Contains(err.Error(), "answered outside what was offered") {
				t.Errorf("error %q does not say the answer was not offered", err)
			}
		})
	}
}
