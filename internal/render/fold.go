package render

import (
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The codecs for the sums. Each is a fold, so a case added to a sum stops
// this package compiling until the view learns to read it.

type enlistmentCodec struct{ out enlistmentView }

func (e *enlistmentCodec) Enlisted(service traveller.ServiceName) error {
	e.out = enlistmentView{How: "enlisted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) Drafted(service traveller.ServiceName) error {
	e.out = enlistmentView{How: "drafted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) DeclinedTheDraft() error {
	e.out = enlistmentView{How: "declined the draft"}

	return nil
}

func foldEnlistment(from traveller.Enlistment) enlistmentView {
	if from == nil {
		return enlistmentView{}
	}

	var codec enlistmentCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = from.Fold(&codec)

	return codec.out
}

type departureCodec struct{ out departureView }

func (d *departureCodec) Discharged() error {
	d.out = departureView{How: "discharged"}

	return nil
}

func (d *departureCodec) ForcedOut() error {
	d.out = departureView{How: "forced out"}

	return nil
}

func (d *departureCodec) Retired() error {
	d.out = departureView{How: "retired"}

	return nil
}

func (d *departureCodec) KilledBySurvivalThrow() error {
	d.out = departureView{How: "killed by the survival throw"}

	return nil
}

func (d *departureCodec) KilledByMedicalCrisis(characteristic traveller.Characteristic) error {
	d.out = departureView{
		How: "killed by a medical crisis", Characteristic: characteristic.String(),
	}

	return nil
}

func foldDeparture(from traveller.Departure) departureView {
	var codec departureCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = from.Fold(&codec)

	// Which departures are deaths is traveller.Fatal's to say, not this
	// codec's. It was stated here and again in the command, and a changed
	// case would have moved one and not the other (#100).
	codec.out.Fatal = traveller.Fatal(from)

	return codec.out
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
	Errata       []string
}

// entryOf folds a domain event into its entry.
func entryOf(event traveller.Event) entry {
	var codec entryCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = event.Fold(&codec)

	return codec.out
}

type entryCodec struct{ out entry }

func (e *entryCodec) Step(from traveller.StepEvent) error {
	e.out = entry{Seq: from.Seq, Kind: kindStep, Step: from.Step, Pages: from.Pages}

	return nil
}

func (e *entryCodec) Throw(from traveller.ThrowEvent) error {
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
func (e *entryCodec) Roll(from traveller.RollEvent) error {
	e.out = entry{
		Seq: from.Seq, Kind: kindRoll, Step: from.Step,
		Dice: from.Dice, DM: from.DM, Total: from.Total(),
	}

	return nil
}

func (e *entryCodec) Choice(from traveller.ChoiceEvent) error {
	e.out = entry{
		Seq: from.Seq, Kind: kindChoice, Point: from.Point,
		Alternatives: from.Alternatives, Chosen: from.Chosen,
	}

	return nil
}

func (e *entryCodec) Outcome(from traveller.OutcomeEvent) error {
	e.out = entry{Seq: from.Seq, Kind: kindOutcome, Because: from.Because, Description: from.Description}

	for _, erratum := range from.Errata {
		e.out.Errata = append(e.out.Errata, erratum.String())
	}

	return nil
}
