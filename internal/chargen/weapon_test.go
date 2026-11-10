package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// --auto names more than one weapon.
//
// Across thirty Army characters it named Body Pistol twenty-two times and
// Dagger seventeen, and nothing else, though every name on both printed lists
// was offered (#34). It took from[0] every time, and no strategy reached it,
// so no flag could change that either.
//
// The threshold is deliberately low, and this counts the answers rather than
// weighing them. It is not a test of the draw's distribution - that is
// math/rand's business - but of the property the report was about: a roster
// that reads as copies.
//
// Holds: The weapon for a Blade or Gun result is drawn, not always the first
// printed.
func TestTheAutoPolicyNamesMoreThanOneWeapon(t *testing.T) {
	t.Parallel()

	named := map[string]int{}

	for _, fixture := range fixtures {
		for _, event := range fixture.generate(t).Events {
			choice, isChoice := event.(traveller.ChoiceEvent)
			if isChoice && choice.Point == traveller.ChoiceWeapon {
				named[choice.Chosen]++
			}
		}
	}

	if len(named) == 0 {
		t.Fatal("no fixture names a weapon, so this checks nothing")
	}

	if len(named) < 2 {
		t.Errorf("the roster names one weapon and no other: %v", named)
	}
}

// One character trains on more than one table.
//
// Under `advanced` the policy designated Advanced Education every time it was
// offered and Personal Development never - ninety times to none over thirty
// characters (#34) - so a character generated under the default never raised
// a characteristic, though p. 11's first table is how that is done.
//
// The count is per character and not over the roster. The first version of
// this test counted across all fixtures and passed under the old ranking too:
// three strategies between them reach all four tables however each one
// chooses, so the roster says nothing about what any character did.
//
// Holds: The skills table designated is the one he has trained on least.
func TestOneCharacterTrainsOnMoreThanOneTable(t *testing.T) {
	t.Parallel()

	widest, widestBy := 0, ""

	for _, fixture := range fixtures {
		designated := map[string]bool{}

		for _, event := range fixture.generate(t).Events {
			choice, isChoice := event.(traveller.ChoiceEvent)
			if isChoice && choice.Point == traveller.ChoiceSkillTable {
				designated[choice.Chosen] = true
			}
		}

		if len(designated) > widest {
			widest, widestBy = len(designated), fixture.name
		}
	}

	// Three of the four. A career long enough to be offered the fourth is
	// offered it only while Education stands at 8, which not every fixture
	// reaches; a career that designates three has stopped hammering one.
	const spread = 3

	if widest < spread {
		t.Errorf("no fixture designates more than %d tables (%s is the widest)", widest, widestBy)
	}
}

// reaching is the --auto policy, plus one attempt to take back the dice that
// Vary leaves out. It records what it found rather than using it, so the
// characters it decides are the policy's own.
type reaching struct {
	chargen.Policy

	asked, reached *int
}

func (r reaching) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, vary chargen.Vary,
) (traveller.WeaponName, error) {
	*r.asked++

	if _, isRoller := vary.(chargen.Roller); isRoller {
		*r.reached++
	}

	return r.Policy.Weapon(category, from, vary) //nolint:wrapcheck // the policy's own error, unchanged
}

// A decider handed a Vary cannot assert its way back to the Roller.
//
// The narrowing was by static type alone, so vary.(chargen.Roller) gave a
// decider Die back, and the dice it threw reached no event: a
// different character from the same dice, recorded as an honest one (#136).
//
// Every fixture walks its own script, so the same Weapon choices are reached
// on every run (#251): reaching answers as the policy does, so each fixture
// is asked exactly as often as its own run records a Weapon choice.
func TestADeciderCannotReachTheRollerThroughVary(t *testing.T) {
	t.Parallel()

	asked, reached := 0, 0

	for _, fixture := range fixtures {
		want := chosenAt(fixture.generate(t), traveller.ChoiceWeapon)
		before := asked

		decider := reaching{Policy: fixture.policy, asked: &asked, reached: &reached}

		_, err := fixture.runWith(t, decider)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		if got := asked - before; got != want {
			t.Errorf("%s: asked Weapon %d times, and its run records %d", fixture.name, got, want)
		}
	}

	if asked == 0 {
		t.Fatal("no fixture reaches the Weapon choice, so this checks nothing")
	}

	if reached > 0 {
		t.Errorf("%d of %d Weapon calls asserted the Vary back to a Roller", reached, asked)
	}
}

// forging is the --auto policy, except that at one weapon choice it writes a
// weapon of its own into the offer and answers with it. The other is the
// policy's, so a run forging at MusterWeapon reaches it along the fixture's
// own path, its skill-table Weapon answered as the script expects.
type forging struct {
	chargen.Policy

	at    traveller.ChoicePoint
	asked *int
}

const forged traveller.WeaponName = "Lightsaber"

func (f forging) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, vary chargen.Vary,
) (traveller.WeaponName, error) {
	if f.at != traveller.ChoiceWeapon {
		return f.Policy.Weapon(category, from, vary) //nolint:wrapcheck // the policy's own error, unchanged
	}

	return f.forge(from), nil
}

func (f forging) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	if f.at != traveller.ChoiceMusterWeapon {
		return f.Policy.MusterWeapon(category, from, received) //nolint:wrapcheck // the policy's own error, unchanged
	}

	return traveller.TakeWeapon{Weapon: f.forge(from)}, nil
}

func (f forging) forge(from []traveller.WeaponName) traveller.WeaponName {
	*f.asked++

	if len(from) > 0 {
		from[0] = forged
	}

	return forged
}

// A decider that rewrites its own offer and answers with the rewrite is
// refused, rather than recorded as having been offered what it wrote.
//
// The offer is named before the decider is asked. Named afterwards, it read
// the slice the decider had written to, and the gate passed a weapon no page
// prints (#174).
//
// Each weapon choice is forged in a run of its own (#251): a forged Weapon
// answer ends the run, so a run forging at both never reached MusterWeapon.
// Every fixture walks its script, so the same choices are reached every run:
// a fixture whose own run makes the choice is refused there, and one whose
// run does not finishes cleanly, never asked.
func TestADeciderCannotForgeItsOwnOffer(t *testing.T) {
	t.Parallel()

	for _, point := range []traveller.ChoicePoint{traveller.ChoiceWeapon, traveller.ChoiceMusterWeapon} {
		t.Run(point.String(), func(t *testing.T) {
			t.Parallel()

			reachedBy := 0

			for _, fixture := range fixtures {
				reaches := chosenAt(fixture.generate(t), point) > 0
				asked := 0

				_, err := fixture.runWith(t, forging{Policy: fixture.policy, at: point, asked: &asked})

				switch {
				case reaches != (asked > 0):
					t.Errorf("%s: asked %v = %v, and its run records it = %v",
						fixture.name, point, asked > 0, reaches)

					continue
				case !reaches:
					if err != nil {
						t.Errorf("%s: never asked %v, and failed anyway: %v", fixture.name, point, err)
					}

					continue
				case err == nil:
					t.Errorf("%s: a forged %v was accepted as offered", fixture.name, point)
				case !strings.Contains(err.Error(), string(forged)):
					t.Errorf("%s: refused at %v, but not for the forgery: %v", fixture.name, point, err)
				}

				reachedBy++
			}

			if reachedBy == 0 {
				t.Fatalf("no fixture reaches %v, so this checks nothing", point)
			}
		})
	}
}

// chosenAt counts the choices a character's run made at a point.
func chosenAt(character *chargen.Character, point traveller.ChoicePoint) int {
	n := 0

	for _, event := range character.Events {
		choice, isChoice := event.(traveller.ChoiceEvent)
		if isChoice && choice.Point == point {
			n++
		}
	}

	return n
}
