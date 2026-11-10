package render

import (
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The folds for the sums. Each implements a cases interface, so a case added
// to a sum stops this package compiling until the view learns to read it.

type enlistmentFold struct{ out enlistmentView }

func (e *enlistmentFold) Enlisted(service traveller.ServiceName) error {
	e.out = enlistmentView{How: "enlisted", Service: service.String()}

	return nil
}

func (e *enlistmentFold) Drafted(service traveller.ServiceName) error {
	e.out = enlistmentView{How: "drafted", Service: service.String()}

	return nil
}

func (e *enlistmentFold) DeclinedTheDraft() error {
	e.out = enlistmentView{How: "declined the draft"}

	return nil
}

// foldEnlistment is how the character entered a service, as the view reads
// it. Only generation sets an Enlistment, and it always does: a nil one is a
// generation defect, and panics where it is folded rather than reading as no
// service, as #377 decided for ServedIn (#393).
func foldEnlistment(from traveller.Enlistment) enlistmentView {
	var fold enlistmentFold

	// This fold cannot fail: every case only assigns.
	_ = from.Fold(&fold)

	return fold.out
}

type departureFold struct{ out departureView }

func (d *departureFold) Discharged() error {
	d.out = departureView{How: "discharged"}

	return nil
}

func (d *departureFold) ForcedOut() error {
	d.out = departureView{How: "forced out"}

	return nil
}

func (d *departureFold) Retired() error {
	d.out = departureView{How: "retired"}

	return nil
}

func (d *departureFold) KilledBySurvivalThrow() error {
	d.out = departureView{How: "killed by the survival throw"}

	return nil
}

func (d *departureFold) KilledByMedicalCrisis(characteristic traveller.Characteristic) error {
	d.out = departureView{
		How: "killed by a medical crisis", Characteristic: characteristic.String(),
	}

	return nil
}

func foldDeparture(from traveller.Departure) departureView {
	var fold departureFold

	// This fold cannot fail: every case only assigns.
	_ = from.Fold(&fold)

	// Which departures are deaths is traveller.Fatal's to say, not this
	// fold's: the engine and the command ask it too, so one answer serves
	// all three, and a case added to Departure is answered once for them all
	// (#100).
	fold.out.Fatal = traveller.Fatal(from)

	return fold.out
}

// entryKind is which case of traveller.Event an entry was folded from. A
// switch over it is exhaustive by lint, so a sixth kind is a failure at every
// switch until each has written it.
type entryKind int

// One for each case of traveller.Event.
const (
	kindStep entryKind = iota
	kindThrow
	kindRoll
	kindChoice
	kindOutcome
)

// entry is every event kind at once, flat, which is what the generation
// record and the live view both read. It is built from the domain event by
// one fold, entryOf, so the two cannot come to read an event differently.
type entry struct {
	Seq          int
	Kind         entryKind
	Step         string
	Pages        string
	Dice         []int
	DM           int
	Target       string
	Total        int
	Succeeded    bool
	Point        traveller.ChoicePoint
	Alternatives []string
	Chosen       string
	Because      int
	Description  string
}

// entryOf folds a domain event into its entry.
func entryOf(event traveller.Event) entry {
	var fold entryFold

	// This fold cannot fail: every case only assigns.
	_ = event.Fold(&fold)

	return fold.out
}

type entryFold struct{ out entry }

func (e *entryFold) Step(from traveller.StepEvent) error {
	e.out = entry{Seq: from.Seq, Kind: kindStep, Step: from.Step, Pages: from.Pages}

	return nil
}

func (e *entryFold) Throw(from traveller.ThrowEvent) error {
	e.out = entry{
		Seq: from.Seq, Kind: kindThrow, Step: from.Step, Dice: from.Dice,
		DM: from.DM, Target: from.Target.String(), Total: from.Total(), Succeeded: from.Succeeded,
	}

	return nil
}

// Roll is a roll with nothing to meet. It carries no target and no outcome
// (#50): read as a throw, it would claim to have succeeded against nothing.
//
// Implements: A roll with nothing to meet is not recorded as a throw that
// succeeded [Generation record].
func (e *entryFold) Roll(from traveller.RollEvent) error {
	e.out = entry{
		Seq: from.Seq, Kind: kindRoll, Step: from.Step,
		Dice: from.Dice, DM: from.DM, Total: from.Total(),
	}

	return nil
}

func (e *entryFold) Choice(from traveller.ChoiceEvent) error {
	e.out = entry{
		Seq: from.Seq, Kind: kindChoice, Point: from.Point,
		Alternatives: from.Alternatives, Chosen: from.Chosen,
	}

	return nil
}

func (e *entryFold) Outcome(from traveller.OutcomeEvent) error {
	e.out = entry{Seq: from.Seq, Kind: kindOutcome, Because: from.Because, Description: from.Description}

	return nil
}
