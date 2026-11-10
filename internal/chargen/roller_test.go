package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/internal/dice"
	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// Every throw is two dice, the next two of the stream, and the draft is one.
// P. 10: "All rolls except draft are two-die throws."
//
// Each fixture's record keeps its dice in the order its script handed them
// out, so a throw that drew its dice in any other way - one die, a die
// skipped, the second before the first - records a sequence the script does
// not state.
//
// Holds: Two-die throws except the draft.
func TestEveryThrowIsTheNextTwoDice(t *testing.T) {
	t.Parallel()

	var drafts int

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		var recorded []int

		for _, event := range character.Events {
			switch e := event.(type) {
			case traveller.ThrowEvent:
				if len(e.Dice) != 2 {
					t.Errorf("%s: the %s throw is %d dice", fixture.name, e.Step, len(e.Dice))
				}

				recorded = append(recorded, e.Dice...)
			case traveller.RollEvent:
				if e.Step == "draft" {
					drafts++

					if len(e.Dice) != 1 {
						t.Errorf("%s: the draft is %d dice", fixture.name, len(e.Dice))
					}
				}

				recorded = append(recorded, e.Dice...)
			}
		}

		if !slices.Equal(recorded, fixture.dice) {
			t.Errorf("%s: the record holds the dice %v; the script threw %v", fixture.name, recorded, fixture.dice)
		}
	}

	if drafts == 0 {
		t.Error("no fixture is drafted, so the draft's one die is checked nowhere")
	}
}

// dice.Faces and rules.Faces say a die's six faces twice, for the reason
// each one's comment gives (#514). chargen imports both, so its test holds
// them equal.
//
// Holds: The dice and the tables agree on a die's faces.
func TestTheDiceAndTheTablesAgreeOnADiesFaces(t *testing.T) {
	t.Parallel()

	if dice.Faces != rules.Faces {
		t.Errorf("dice rolls %d faces, and rules sizes its tables for %d", dice.Faces, rules.Faces)
	}
}
