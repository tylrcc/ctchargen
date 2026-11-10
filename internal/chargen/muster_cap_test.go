package chargen_test

import (
	"testing"
)

// tableTwoStep is the step the engine records a Table 2 throw under.
const tableTwoStep = "Table 2, Cash Allowances"

// printedCap is p. 9's limit on how many benefit rolls may go on Table 2,
// typed out here rather than read from the engine.
const printedCap = 3

// No character takes more than three rolls on Table 2 (p. 9).
//
// The cap used to sit in mustering.json as maxOnTable2 and is now a constant
// beside the procedure that applies it (#43), because it bounds a procedure
// and indexes no printed table. The check moved with it, and became a check
// of the rule rather than of the number: asserting that a field equals 3 says
// nothing about whether anything stops at 3.
//
// Holds: At most three rolls on Table 2.
func TestAtMostThreeRollsGoOnTableTwo(t *testing.T) {
	t.Parallel()

	var reached bool

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		onTableTwo := 0

		for _, event := range character.Events {
			// A mustering out roll meets no target, so it is a RollEvent
			// since #50 and not a throw.
			step, rolled := stepOf(event)
			if rolled && step == tableTwoStep {
				onTableTwo++
			}
		}

		if onTableTwo > printedCap {
			t.Errorf("%s: %d rolls on Table 2, and p. 9 allows %d", fixture.name, onTableTwo, printedCap)
		}

		reached = reached || onTableTwo == printedCap
	}

	if !reached {
		t.Errorf("no fixture reaches %d rolls on Table 2, so the cap stops nothing here", printedCap)
	}
}
