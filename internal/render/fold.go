package render

import (
	"encoding/json"
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The codecs for the sums. Each is a fold, so a case added to a sum stops
// this package compiling until the projection learns to write it.

type enlistmentCodec struct{ out enlistmentRecord }

func (e *enlistmentCodec) Enlisted(service traveller.ServiceName) error {
	e.out = enlistmentRecord{How: "enlisted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) Drafted(service traveller.ServiceName) error {
	e.out = enlistmentRecord{How: "drafted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) DeclinedTheDraft() error {
	e.out = enlistmentRecord{How: "declined the draft"}

	return nil
}

func foldEnlistment(from traveller.Enlistment) enlistmentRecord {
	if from == nil {
		return enlistmentRecord{}
	}

	var codec enlistmentCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = from.Fold(&codec)

	return codec.out
}

type departureCodec struct{ out departureRecord }

func (d *departureCodec) Discharged() error {
	d.out = departureRecord{How: "discharged"}

	return nil
}

func (d *departureCodec) ForcedOut() error {
	d.out = departureRecord{How: "forced out"}

	return nil
}

func (d *departureCodec) Retired() error {
	d.out = departureRecord{How: "retired"}

	return nil
}

func (d *departureCodec) KilledBySurvivalThrow() error {
	d.out = departureRecord{How: "killed by the survival throw"}

	return nil
}

func (d *departureCodec) KilledByMedicalCrisis(characteristic traveller.Characteristic) error {
	d.out = departureRecord{
		How: "killed by a medical crisis", Characteristic: characteristic.String(),
	}

	return nil
}

func foldDeparture(from traveller.Departure) departureRecord {
	var codec departureCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = from.Fold(&codec)

	// Which departures are deaths is traveller.Fatal's to say, not this
	// codec's. It was stated here and again in the command, and a changed
	// case would have moved one and not the other (#100).
	codec.out.Fatal = traveller.Fatal(from)

	return codec.out
}

// The kinds an event's discriminator can be, on the wire.
const (
	kindStep    = "step"
	kindThrow   = "throw"
	kindRoll    = "roll"
	kindChoice  = "choice"
	kindOutcome = "outcome"
)

type stepJSON struct {
	Seq   int    `json:"seq"`
	Kind  string `json:"kind"`
	Step  string `json:"step"`
	Pages string `json:"pages,omitempty"`
}

type throwJSON struct {
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	Step      string `json:"step"`
	Dice      []int  `json:"dice"`
	DM        int    `json:"dm,omitempty"`
	Target    string `json:"target,omitempty"`
	Total     int    `json:"total"`
	Succeeded bool   `json:"succeeded"`
}

// rollJSON is a roll with nothing to meet. It carries no target and no
// outcome (#50): written as a throw, it would claim to have succeeded against
// nothing.
type rollJSON struct {
	Seq   int    `json:"seq"`
	Kind  string `json:"kind"`
	Step  string `json:"step"`
	Dice  []int  `json:"dice"`
	DM    int    `json:"dm,omitempty"`
	Total int    `json:"total"`
}

type choiceJSON struct {
	Seq          int      `json:"seq"`
	Kind         string   `json:"kind"`
	Point        string   `json:"point"`
	By           string   `json:"by"`
	Alternatives []string `json:"alternatives"`
	Chosen       string   `json:"chosen"`
}

type outcomeJSON struct {
	Seq         int      `json:"seq"`
	Kind        string   `json:"kind"`
	Because     int      `json:"because,omitempty"`
	Description string   `json:"description"`
	Errata      []string `json:"errata,omitempty"`
}

// eventCodec folds a domain event into the shape it is written in, and into
// the flat shape a reader gets it back as.
//
// It produces both so that there is one mapping from the domain, and flat is
// derived from what is written rather than built beside it (#46). Two codecs
// would each map the same domain fields: Fold would catch a fifth kind, since
// both would stop compiling, but not a field, and one could stop carrying DM
// with only a transcript golden to notice.
//
// Implements: The shape a reader gets back is the shape that was written
// [Generation record] in The record.
type eventCodec struct {
	out  json.RawMessage
	flat eventJSON
}

func (e *eventCodec) Step(from traveller.StepEvent) error {
	out := stepJSON{Seq: from.Seq, Kind: kindStep, Step: from.Step, Pages: from.Pages}

	e.flat = eventJSON{Seq: out.Seq, Kind: out.Kind, Step: out.Step, Pages: out.Pages}

	return e.write(out)
}

func (e *eventCodec) Throw(from traveller.ThrowEvent) error {
	out := throwJSON{
		Seq: from.Seq, Kind: kindThrow, Step: from.Step, Dice: from.Dice,
		DM: from.DM, Total: from.Total(), Succeeded: from.Succeeded,
	}

	// Guarded, though every throw has a target by construction: a zero
	// Target stringifies to "0", not to nothing, so writing it unguarded
	// would put "target": "0" on a throw that has none and satisfy the
	// schema's requirement with it. Omitted, the record fails its own schema,
	// which is what should happen to a throw with nothing to meet.
	if from.Target.Number() != 0 {
		out.Target = from.Target.String()
	}

	e.flat = eventJSON{
		Seq: out.Seq, Kind: out.Kind, Step: out.Step, Dice: out.Dice, DM: out.DM,
		Target: out.Target, Total: out.Total, Succeeded: out.Succeeded,
	}

	return e.write(out)
}

func (e *eventCodec) Roll(from traveller.RollEvent) error {
	out := rollJSON{
		Seq: from.Seq, Kind: kindRoll, Step: from.Step,
		Dice: from.Dice, DM: from.DM, Total: from.Total(),
	}

	e.flat = eventJSON{
		Seq: out.Seq, Kind: out.Kind, Step: out.Step,
		Dice: out.Dice, DM: out.DM, Total: out.Total,
	}

	return e.write(out)
}

func (e *eventCodec) Choice(from traveller.ChoiceEvent) error {
	out := choiceJSON{
		Seq: from.Seq, Kind: kindChoice, Point: from.Point.String(), By: from.By.String(),
		Alternatives: from.Alternatives, Chosen: from.Chosen,
	}

	e.flat = eventJSON{
		Seq: out.Seq, Kind: out.Kind, Point: out.Point, By: out.By,
		Alternatives: out.Alternatives, Chosen: out.Chosen,
	}

	return e.write(out)
}

func (e *eventCodec) Outcome(from traveller.OutcomeEvent) error {
	out := outcomeJSON{
		Seq: from.Seq, Kind: kindOutcome, Because: from.Because, Description: from.Description,
	}
	for _, erratum := range from.Errata {
		out.Errata = append(out.Errata, erratum.String())
	}

	e.flat = eventJSON{
		Seq: out.Seq, Kind: out.Kind, Because: out.Because,
		Description: out.Description, Errata: out.Errata,
	}

	return e.write(out)
}

// projectEvents writes every event, and refuses to write any of them if one
// will not marshal. The record claims to be the complete account of a
// generation, so an entry silently missing from it is worse than a failure
// to write the record at all.
func projectEvents(events []traveller.Event) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(events))

	for _, event := range events {
		var codec eventCodec

		err := event.Fold(&codec)
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", event.Sequence(), err)
		}

		out = append(out, codec.out)
	}

	return out, nil
}

func (e *eventCodec) write(v any) error {
	text, err := json.Marshal(v)

	e.out = text

	//nolint:wrapcheck // the caller adds the context; this is one marshal.
	return err
}
