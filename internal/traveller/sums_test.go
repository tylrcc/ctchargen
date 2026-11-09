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

type recorder struct{ got []string }

func (r *recorder) note(format string, args ...any) error {
	r.got = append(r.got, fmt.Sprintf(format, args...))

	return nil
}

func (r *recorder) check(t *testing.T, sum string, want []string) {
	t.Helper()

	if len(r.got) != len(want) {
		t.Fatalf("%s: folded to %v, want %v", sum, r.got, want)
	}

	seen := map[string]bool{}

	for i, got := range r.got {
		if got != want[i] {
			t.Errorf("%s: case %d folded to %q, want %q", sum, i, got, want[i])
		}

		if seen[got] {
			t.Errorf("%s: two cases both folded to %q; one is wired to the other's method", sum, got)
		}

		seen[got] = true
	}
}

type enlistmentRecorder struct{ recorder }

func (r *enlistmentRecorder) Enlisted(s traveller.ServiceName) error { return r.note("enlisted %v", s) }

func (r *enlistmentRecorder) Drafted(s traveller.ServiceName) error { return r.note("drafted %v", s) }
func (r *enlistmentRecorder) DeclinedTheDraft() error               { return r.note("declined") }

func TestEnlistmentFolds(t *testing.T) {
	t.Parallel()

	var r enlistmentRecorder

	for _, e := range []traveller.Enlistment{
		traveller.Enlisted{Service: traveller.Navy},
		traveller.Drafted{Service: traveller.Other},
		traveller.DeclinedTheDraft{},
	} {
		err := e.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	r.check(t, "Enlistment", []string{"enlisted Navy", "drafted Other", "declined"})
}

type tableResultRecorder struct{ recorder }

func (r *tableResultRecorder) Alteration(c traveller.Characteristic, d int) error {
	return r.note("alteration %v %+d", c, d)
}
func (r *tableResultRecorder) Skill(n traveller.SkillName) error { return r.note("skill %s", n) }
func (r *tableResultRecorder) WeaponPick(c traveller.WeaponCategory) error {
	return r.note("weapon %v", c)
}

// Holds: Three kinds of result.
func TestTableResultFolds(t *testing.T) {
	t.Parallel()

	var r tableResultRecorder

	for _, result := range []traveller.TableResult{
		// Other's personal development table is the procedure's one
		// negative result (p. 11).
		traveller.AlterationResult{Characteristic: traveller.SocialStanding, Delta: -1},
		traveller.SkillResult{Name: "Jack of all Trades"},
		traveller.WeaponPickResult{Category: traveller.Blade},
	} {
		err := result.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	r.check(t, "TableResult", []string{
		"alteration Social Standing -1", "skill Jack of all Trades", "weapon Blade Combat",
	})
}

type benefitRecorder struct{ recorder }

func (r *benefitRecorder) Passage(p traveller.PassageClass) error {
	return r.note("passage %v", p)
}

func (r *benefitRecorder) Alteration(c traveller.Characteristic, d int) error {
	return r.note("alteration %v %+d", c, d)
}

func (r *benefitRecorder) WeaponPick(c traveller.WeaponCategory) error { return r.note("weapon %v", c) }

func (r *benefitRecorder) TravellersAid() error { return r.note("travellers aid") }

func (r *benefitRecorder) Ship(k traveller.ShipKind) error { return r.note("ship %v", k) }
func (r *benefitRecorder) Nothing() error                  { return r.note("nothing") }

// Holds: The six kinds of Table 1 row.
// Holds: Travellers' Aid only once; duplicates wasted.
func TestBenefitRowFolds(t *testing.T) {
	t.Parallel()

	var r benefitRecorder

	for _, row := range []traveller.BenefitRow{
		traveller.PassageBenefit{Class: traveller.LowPassage},
		traveller.AlterationBenefit{Characteristic: traveller.Intelligence, Delta: 2},
		traveller.WeaponCategoryBenefit{Category: traveller.Gun},
		traveller.TravellersAidBenefit{},
		traveller.ShipBenefit{Kind: traveller.ScoutShip},
		traveller.NoBenefit{},
	} {
		err := row.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	r.check(t, "BenefitRow", []string{
		"passage Low Passage", "alteration Intelligence +2",
		"weapon Gun Combat", "travellers aid", "ship Scout ship, Type S", "nothing",
	})
}

type departureRecorder struct{ recorder }

func (r *departureRecorder) Discharged() error            { return r.note("discharged") }
func (r *departureRecorder) ForcedOut() error             { return r.note("forced out") }
func (r *departureRecorder) Retired() error               { return r.note("retired") }
func (r *departureRecorder) KilledBySurvivalThrow() error { return r.note("killed by survival") }
func (r *departureRecorder) KilledByMedicalCrisis(c traveller.Characteristic) error {
	return r.note("killed by crisis in %v", c)
}

func TestDepartureFolds(t *testing.T) {
	t.Parallel()

	var r departureRecorder

	for _, d := range []traveller.Departure{
		traveller.Discharged{},
		traveller.ForcedOut{},
		traveller.Retired{},
		traveller.KilledBySurvivalThrow{},
		traveller.KilledByMedicalCrisis{Characteristic: traveller.Endurance},
	} {
		err := d.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	r.check(t, "Departure", []string{
		"discharged", "forced out", "retired", "killed by survival", "killed by crisis in Endurance",
	})
}

type weaponBenefitRecorder struct{ recorder }

func (r *weaponBenefitRecorder) TakeWeapon(w traveller.WeaponName) error {
	return r.note("weapon %s", w)
}

func (r *weaponBenefitRecorder) TakeExpertise(w traveller.WeaponName) error {
	return r.note("expertise %s", w)
}

func TestWeaponBenefitFolds(t *testing.T) {
	t.Parallel()

	var r weaponBenefitRecorder

	// P. 22's own worked sentence: a cutlass, then either another blade or
	// expertise in the cutlass already received.
	for _, b := range []traveller.WeaponBenefit{
		traveller.TakeWeapon{Weapon: "Cutlass"},
		traveller.TakeExpertise{Weapon: "Cutlass"},
	} {
		err := b.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}
	}

	r.check(t, "WeaponBenefit", []string{"weapon Cutlass", "expertise Cutlass"})
}

type eventRecorder struct{ recorder }

func (r *eventRecorder) Step(e traveller.StepEvent) error   { return r.note("step %s", e.Step) }
func (r *eventRecorder) Throw(e traveller.ThrowEvent) error { return r.note("throw %d", e.Total()) }
func (r *eventRecorder) Roll(e traveller.RollEvent) error   { return r.note("roll %d", e.Total()) }
func (r *eventRecorder) Choice(e traveller.ChoiceEvent) error {
	return r.note("choice %v", e.Point)
}

func (r *eventRecorder) Outcome(e traveller.OutcomeEvent) error {
	return r.note("outcome %s", e.Description)
}

func TestEventFolds(t *testing.T) {
	t.Parallel()

	var r eventRecorder

	events := []traveller.Event{
		traveller.StepEvent{Seq: 1, Step: "enlistment"},
		traveller.ThrowEvent{Seq: 2, Dice: []int{3, 4}, DM: 2},
		traveller.RollEvent{Seq: 3, Dice: []int{5}, DM: 1},
		traveller.ChoiceEvent{Seq: 4, Point: traveller.ChoiceSubmitToDraft},
		traveller.OutcomeEvent{Seq: 5, Description: "drafted into the Marines"},
	}
	for i, e := range events {
		err := e.Fold(&r)
		if err != nil {
			t.Fatalf("Fold: %v", err)
		}

		if got := e.Sequence(); got != i+1 {
			t.Errorf("event %d reports sequence %d", i+1, got)
		}
	}

	r.check(t, "Event", []string{
		"step enlistment", "throw 9", "roll 6", "choice SubmitToDraft", "outcome drafted into the Marines",
	})
}

// A fold must carry its handler's error out rather than swallow it: applying
// a result can fail, and a generation that failed must stop.
type refusing struct{ tableResultRecorder }

func (refusing) Skill(traveller.SkillName) error { return errRefused }

func TestFoldCarriesTheHandlersError(t *testing.T) {
	t.Parallel()

	var r refusing

	err := (traveller.SkillResult{Name: "Pilot"}).Fold(&r)

	if !errors.Is(err, errRefused) {
		t.Errorf("Fold returned %v, want %v", err, errRefused)
	}
}
