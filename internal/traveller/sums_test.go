package traveller_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Folding is where a copy-paste error hides: a case wired to its neighbour's
// method compiles, runs, and is wrong. Every test below asserts that each
// case reaches its own method and carries its own fields, and that no two
// cases of a sum reach the same one.

var errRefused = errors.New("refused")

type words struct{ got []string }

func (w *words) say(format string, args ...any) error {
	w.got = append(w.got, fmt.Sprintf(format, args...))

	return nil
}

func (w *words) check(t *testing.T, sum string, want []string) {
	t.Helper()

	if len(w.got) != len(want) {
		t.Fatalf("%s: folded to %v, want %v", sum, w.got, want)
	}

	seen := map[string]bool{}

	for i, got := range w.got {
		if got != want[i] {
			t.Errorf("%s: case %d folded to %q, want %q", sum, i, got, want[i])
		}

		if seen[got] {
			t.Errorf("%s: two cases both folded to %q; one is wired to the other's method", sum, got)
		}

		seen[got] = true
	}
}

type enlistmentWordsFold struct{ words }

func (w *enlistmentWordsFold) Enlisted(s traveller.ServiceName) error {
	return w.say("enlisted %v", s)
}

func (w *enlistmentWordsFold) Drafted(s traveller.ServiceName) error { return w.say("drafted %v", s) }
func (w *enlistmentWordsFold) DeclinedTheDraft() error               { return w.say("declined") }

func TestEnlistmentFolds(t *testing.T) {
	t.Parallel()

	var fold enlistmentWordsFold

	for _, e := range []traveller.Enlistment{
		traveller.Enlisted{Service: traveller.Navy},
		traveller.Drafted{Service: traveller.Other},
		traveller.DeclinedTheDraft{},
	} {
		err := e.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	fold.check(t, "Enlistment", []string{"enlisted Navy", "drafted Other", "declined"})
}

type tableResultWordsFold struct{ words }

func (w *tableResultWordsFold) Alteration(c traveller.Characteristic, d int) error {
	return w.say("alteration %v %+d", c, d)
}
func (w *tableResultWordsFold) Skill(n traveller.SkillName) error { return w.say("skill %s", n) }
func (w *tableResultWordsFold) WeaponPick(c traveller.WeaponCategory) error {
	return w.say("weapon %v", c)
}

// Holds: Three kinds of result.
func TestTableResultFolds(t *testing.T) {
	t.Parallel()

	var fold tableResultWordsFold

	for _, result := range []traveller.TableResult{
		// Other's personal development table is the procedure's one
		// negative result (p. 11).
		traveller.AlterationResult{Characteristic: traveller.SocialStanding, Delta: -1},
		traveller.SkillResult{Name: "Jack of all Trades"},
		traveller.WeaponPickResult{Category: traveller.Blade},
	} {
		err := result.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	fold.check(t, "TableResult", []string{
		"alteration Social Standing -1", "skill Jack of all Trades", "weapon Blade Combat",
	})
}

type benefitWordsFold struct{ words }

func (w *benefitWordsFold) Passage(p traveller.PassageClass) error {
	return w.say("passage %v", p)
}

func (w *benefitWordsFold) Alteration(c traveller.Characteristic, d int) error {
	return w.say("alteration %v %+d", c, d)
}

func (w *benefitWordsFold) WeaponPick(c traveller.WeaponCategory) error {
	return w.say("weapon %v", c)
}

func (w *benefitWordsFold) TravellersAid() error { return w.say("travellers aid") }

func (w *benefitWordsFold) Ship(k traveller.ShipKind) error { return w.say("ship %v", k) }
func (w *benefitWordsFold) Nothing() error                  { return w.say("nothing") }

// Holds: The six kinds of Table 1 row.
// Holds: Travellers' Aid only once; duplicates wasted.
func TestBenefitRowFolds(t *testing.T) {
	t.Parallel()

	var fold benefitWordsFold

	for _, row := range []traveller.BenefitRow{
		traveller.PassageBenefit{Class: traveller.LowPassage},
		traveller.AlterationBenefit{Characteristic: traveller.Intelligence, Delta: 2},
		traveller.WeaponCategoryBenefit{Category: traveller.Gun},
		traveller.TravellersAidBenefit{},
		traveller.ShipBenefit{Kind: traveller.ScoutShip},
		traveller.NoBenefit{},
	} {
		err := row.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	fold.check(t, "BenefitRow", []string{
		"passage Low Passage", "alteration Intelligence +2",
		"weapon Gun Combat", "travellers aid", "ship Scout ship, Type S", "nothing",
	})
}

type departureWordsFold struct{ words }

func (w *departureWordsFold) Discharged() error            { return w.say("discharged") }
func (w *departureWordsFold) ForcedOut() error             { return w.say("forced out") }
func (w *departureWordsFold) Retired() error               { return w.say("retired") }
func (w *departureWordsFold) KilledBySurvivalThrow() error { return w.say("killed by survival") }
func (w *departureWordsFold) KilledByMedicalCrisis(c traveller.Characteristic) error {
	return w.say("killed by crisis in %v", c)
}

func TestDepartureFolds(t *testing.T) {
	t.Parallel()

	var fold departureWordsFold

	for _, d := range []traveller.Departure{
		traveller.Discharged{},
		traveller.ForcedOut{},
		traveller.Retired{},
		traveller.KilledBySurvivalThrow{},
		traveller.KilledByMedicalCrisis{Characteristic: traveller.Endurance},
	} {
		err := d.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	fold.check(t, "Departure", []string{
		"discharged", "forced out", "retired", "killed by survival", "killed by crisis in Endurance",
	})
}

type weaponBenefitWordsFold struct{ words }

func (w *weaponBenefitWordsFold) TakeWeapon(weapon traveller.WeaponName) error {
	return w.say("weapon %s", weapon)
}

func (w *weaponBenefitWordsFold) TakeExpertise(weapon traveller.WeaponName) error {
	return w.say("expertise %s", weapon)
}

func TestWeaponBenefitFolds(t *testing.T) {
	t.Parallel()

	var fold weaponBenefitWordsFold

	// P. 22's own worked sentence: a cutlass, then either another blade or
	// expertise in the cutlass already received.
	for _, b := range []traveller.WeaponBenefit{
		traveller.TakeWeapon{Weapon: "Cutlass"},
		traveller.TakeExpertise{Weapon: "Cutlass"},
	} {
		err := b.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	fold.check(t, "WeaponBenefit", []string{"weapon Cutlass", "expertise Cutlass"})
}

type eventWordsFold struct{ words }

func (w *eventWordsFold) Step(e traveller.StepEvent) error   { return w.say("step %s", e.Step) }
func (w *eventWordsFold) Throw(e traveller.ThrowEvent) error { return w.say("throw %d", e.Total()) }
func (w *eventWordsFold) Roll(e traveller.RollEvent) error   { return w.say("roll %d", e.Total()) }
func (w *eventWordsFold) Choice(e traveller.ChoiceEvent) error {
	return w.say("choice %v", e.Point)
}

func (w *eventWordsFold) Outcome(e traveller.OutcomeEvent) error {
	return w.say("outcome %s", e.Description)
}

func TestEventFolds(t *testing.T) {
	t.Parallel()

	var fold eventWordsFold

	events := []traveller.Event{
		traveller.StepEvent{Seq: 1, Step: "enlistment"},
		traveller.ThrowEvent{Seq: 2, Dice: []int{3, 4}, DM: 2},
		traveller.RollEvent{Seq: 3, Dice: []int{5}, DM: 1},
		traveller.ChoiceEvent{Seq: 4, Point: traveller.ChoiceSubmitToDraft},
		traveller.OutcomeEvent{Seq: 5, Description: "drafted into the Marines"},
	}
	for i, e := range events {
		err := e.Fold(&fold)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}

		if got := e.Sequence(); got != i+1 {
			t.Errorf("event %d reports sequence %d", i+1, got)
		}
	}

	fold.check(t, "Event", []string{
		"step enlistment", "throw 9", "roll 6", "choice SubmitToDraft", "outcome drafted into the Marines",
	})
}

// refusing is a handler that refuses a case, not a fold: a sum's Fold method
// must carry its handler's error out rather than swallow it, because applying
// a result can fail, and a generation that failed must stop.
type refusing struct{ tableResultWordsFold }

func (refusing) Skill(traveller.SkillName) error { return errRefused }

func TestFoldCarriesTheHandlersError(t *testing.T) {
	t.Parallel()

	var handler refusing

	err := (traveller.SkillResult{Name: "Pilot"}).Fold(&handler)

	if !errors.Is(err, errRefused) {
		t.Errorf("Fold returned %v, want %v", err, errRefused)
	}
}
