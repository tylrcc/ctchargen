package chargen_test

import (
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
// Holds: A question with one legal answer is never put to a decider.
// Holds: Leaving at the end of term 5 or later is retirement.
func TestVoluntaryServiceStopsAtSeven(t *testing.T) {
	t.Parallel()

	inputs := chargen.Inputs{
		Forced: chargen.ForceService(traveller.Other),
	}

	// Every throw a 10: every reenlistment is made, and none is the 12 that
	// forces one.
	character, err := chargen.Auto(inputs, chargen.DefaultPolicy(), chargen.WithRoller(&alwaysFive{}))
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

// alwaysFive makes every die a 5, so every throw a 10: enough for every
// target Other prints, and never the 12 that forces another term.
type alwaysFive struct{}

func (alwaysFive) Die() int      { return 5 }
func (alwaysFive) Among(int) int { return 0 }
