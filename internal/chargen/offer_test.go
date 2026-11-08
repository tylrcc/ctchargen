package chargen_test

import (
	"reflect"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// scribbled is what scribbling writes over a weapon list: no weapon of the
// books' lists, so an offer it reaches is plain to see.
const scribbled traveller.WeaponName = "scribbled"

// scribbling answers as its policy does, and then writes over every slice it
// was handed: the worst a decider can do to what it is offered without
// answering differently.
type scribbling struct {
	chargen.Policy
}

func (s scribbling) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	chosen, err := s.Policy.Service(from)
	for i := range from {
		from[i] = traveller.EnlistmentOffer{Service: chosen}
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) SkillTable(from []traveller.SkillTable, taken []int) (traveller.SkillTable, error) {
	chosen, err := s.Policy.SkillTable(from, taken)
	for i := range from {
		from[i] = chosen
		taken[i] = 99
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, vary chargen.Vary,
) (traveller.WeaponName, error) {
	chosen, err := s.Policy.Weapon(category, from, vary)
	for i := range from {
		from[i] = scribbled
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	chosen, err := s.Policy.ReenlistIntent(from)
	for i := range from {
		from[i] = chosen
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	chosen, err := s.Policy.MusterTable(from)
	for i := range from {
		from[i] = chosen
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	chosen, err := s.Policy.MusterWeapon(category, from, received)
	for i := range from {
		from[i] = scribbled
	}

	for i := range received {
		received[i] = scribbled
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

// An offer is the decider's own: one that writes over everything it is
// handed changes nothing the engine offers afterwards, in that run or in the
// next.
//
// The sites that ask once each had to remember to copy, and three did not
// (#55, #137, #230). The gate copies now, for every decider, so this walks
// every fixture with a decider that answers as the fixture's policy does and
// then writes over its offer, and holds each run's choices to the ones the
// policy alone makes - before and after.
//
// Holds: An offer is the decider's own, and writing to it changes nothing the
// engine offers.
func TestAnOfferIsTheDecidersOwn(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		before := choicesOf(fixture.generate(t))

		scribbled, err := fixture.runWith(t, scribbling{Policy: fixture.policy})
		if err != nil {
			t.Errorf("%s, answered by a scribbler: %v", fixture.name, err)

			continue
		}

		if during := choicesOf(scribbled); !reflect.DeepEqual(during, before) {
			t.Errorf("%s: writing to an offer changed what the same run offered later:\n%v\nwant\n%v",
				fixture.name, during, before)
		}

		if after := choicesOf(fixture.generate(t)); !reflect.DeepEqual(after, before) {
			t.Errorf("%s: writing to an offer changed what the next run offered:\n%v\nwant\n%v",
				fixture.name, after, before)
		}
	}
}

func choicesOf(character *chargen.Character) []traveller.ChoiceEvent {
	var choices []traveller.ChoiceEvent

	for _, event := range character.Events {
		if choice, ok := event.(traveller.ChoiceEvent); ok {
			choices = append(choices, choice)
		}
	}

	return choices
}
