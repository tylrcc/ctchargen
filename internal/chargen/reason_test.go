package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The record says an answer was drawn where the decider drew it, and nowhere
// else (#564). Policy says which of its answers draw: the service, the weapon
// a combat result names, and the weapon a benefit names, which cash and goods
// take as expertise without drawing once one has been received. The worked
// example names every answer it gives, so nothing in its record was drawn.
//
// Holds: An answer is recorded as drawn only where the decider drew it.
func TestTheRecordSaysWhichAnswersWereDrawn(t *testing.T) {
	t.Parallel()

	drawn := 0

	for _, fixture := range fixtures {
		for _, event := range fixture.generate(t).Events {
			choice, isChoice := event.(traveller.ChoiceEvent)
			if !isChoice {
				continue
			}

			want := policyDraws(choice)
			if choice.Drawn != want {
				t.Errorf("%s: step %d, %v answered %q, is recorded drawn %v, want %v",
					fixture.name, choice.Seq, choice.Point, choice.Chosen, choice.Drawn, want)
			}

			if choice.Drawn {
				drawn++
			}
		}
	}

	if drawn == 0 {
		t.Error("no fixture records a drawn answer, so the check above held nothing")
	}

	for _, event := range reproduceJamison(t).Events {
		if choice, isChoice := event.(traveller.ChoiceEvent); isChoice && choice.Drawn {
			t.Errorf("the worked example: step %d, %v is recorded drawn, and the example draws nothing",
				choice.Seq, choice.Point)
		}
	}
}

// policyDraws is whether Policy's doc says it draws its answer at a choice:
// the service, the weapon a combat result names, and the weapon a benefit
// names, where expertise is not taken.
func policyDraws(choice traveller.ChoiceEvent) bool {
	switch choice.Point {
	case traveller.ChoiceService, traveller.ChoiceWeapon:
		return true
	case traveller.ChoiceMusterWeapon:
		return !strings.HasPrefix(choice.Chosen, "expertise in ")
	case traveller.ChoiceSubmitToDraft, traveller.ChoiceAttemptCommission, traveller.ChoiceAttemptPromotion,
		traveller.ChoiceReenlistIntent, traveller.ChoiceAssumeTitle, traveller.ChoiceSkillTable,
		traveller.ChoiceMusterTable, traveller.ChoiceMusterTable1DM, traveller.ChoiceMusterTable2DM:
		return false
	}

	return false
}

// A skills table choice records, for each table offered, how many results the
// character had already had off it (#564): the count the policy weighs, which
// the record had never carried. It is read back here from the record's own
// rolls, one result for each roll on a table, and every other choice carries
// no counts.
//
// Holds: A skills table choice records what each table offered had given.
func TestASkillsTableChoiceRecordsWhatEachTableHadGiven(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		rolled := map[string]int{}

		for _, event := range fixture.generate(t).Events {
			switch event := event.(type) {
			case traveller.RollEvent:
				rolled[event.Step]++
			case traveller.ChoiceEvent:
				want := []int(nil)

				if event.Point == traveller.ChoiceSkillTable {
					for _, table := range event.Alternatives {
						want = append(want, rolled[table])
					}
				}

				if !equalCounts(event.Taken, want) {
					t.Errorf("%s: step %d, %v records %v taken off %v, want %v",
						fixture.name, event.Seq, event.Point, event.Taken, event.Alternatives, want)
				}
			}
		}
	}
}

// equalCounts is whether two lists of counts are the same, nil and empty
// told apart: a choice that is not a skills table choice carries none.
func equalCounts(got, want []int) bool {
	if (got == nil) != (want == nil) || len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
