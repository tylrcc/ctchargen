package chargen_test

import (
	"reflect"
	"strings"
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

func (s scribbling) Service(from []traveller.EnlistmentOffer, vary chargen.Vary) (traveller.ServiceName, error) {
	chosen, err := s.Policy.Service(from, vary)
	for i := range from {
		from[i] = traveller.EnlistmentOffer{Service: chosen}
	}

	return chosen, err //nolint:wrapcheck // the policy's own error, unchanged
}

func (s scribbling) SkillTable(
	from []traveller.SkillTableOffer, roll traveller.SkillRoll,
) (traveller.SkillTable, error) {
	chosen, err := s.Policy.SkillTable(from, roll)
	for i := range from {
		from[i] = traveller.SkillTableOffer{Table: chosen, Taken: 99}
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

func (s scribbling) ReenlistIntent(from []traveller.Intent, survival traveller.Odds) (traveller.Intent, error) {
	chosen, err := s.Policy.ReenlistIntent(from, survival)
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

// answeringOutsideTheOffer is a decider that reaches past what the procedure
// put in front of it. Everything else it defers to the policy.
type answeringOutsideTheOffer struct {
	chargen.Decider

	table     bool
	expertise bool
}

func (a answeringOutsideTheOffer) SkillTable(
	from []traveller.SkillTableOffer, roll traveller.SkillRoll,
) (traveller.SkillTable, error) {
	if a.table {
		// P. 11 opens this table only at Education 8, and the character
		// generated below stands at 7, so it is not among `from`.
		return traveller.AdvancedEducationEight, nil
	}

	return a.Decider.SkillTable(from, roll) //nolint:wrapcheck // the policy's own error, unchanged
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

			// Not f.run: the decider is not the fixture's policy, and a
			// refusal stops the procedure partway, so the script's unread
			// tail is expected rather than a sign it carries dice it never
			// throws.
			_, err := chargen.Guided(f.inputs(), tc.decider, chargen.WithRoller(f.roller(t)))
			if err == nil {
				t.Fatal("the engine applied an answer it had not offered")
			}

			if !strings.Contains(err.Error(), "answered outside what was offered") {
				t.Errorf("error %q does not say the answer was not offered", err)
			}
		})
	}
}
