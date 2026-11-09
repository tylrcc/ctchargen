package render

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The shape a reader gets back is the shape that was written.
//
// eventCodec produces two things from one fold: the per-kind struct that goes
// on the wire, and the flat shape the transcript and the live view both read.
// There used to be a second codec producing the second of those, mapping the
// same domain fields a second time. Fold caught a fifth kind - both codecs
// stopped compiling - but not a field: liveCodec could stop carrying DM and
// only a transcript golden would notice (#46).
//
// One mapping is not enough on its own, because flat is still assigned field
// by field. This is what holds it: unmarshal what was written and compare.
// A field dropped from flat fails here, naming the kind.
//
// Holds: A roll with nothing to meet is not recorded as a throw that succeeded.
// Holds: The shape a reader gets back is the shape that was written.
func TestTheFlatShapeIsWhatWasWritten(t *testing.T) {
	t.Parallel()

	fivePlus, err := traveller.ParseTarget("5+")
	if err != nil {
		t.Fatalf("parsing a target: %v", err)
	}

	for name, event := range map[string]traveller.Event{
		"step": traveller.StepEvent{Seq: 1, Step: "enlistment", Pages: "pp. 5, 10"},
		"throw against a target": traveller.ThrowEvent{
			Seq: 2, Step: "survival", Dice: []int{3, 4}, DM: 2,
			Target: fivePlus, Succeeded: true,
		},
		"roll with nothing to meet": traveller.RollEvent{
			Seq: 3, Step: "Table 2, Cash Allowances", Dice: []int{4}, DM: 1,
		},
		"choice": traveller.ChoiceEvent{
			Seq: 4, Point: traveller.ChoiceSubmitToDraft,
			Alternatives: []string{"yes", "no"}, Chosen: "yes",
		},
		"outcome": traveller.OutcomeEvent{
			Seq: 5, Because: 2, Description: "survived",
			Errata: []traveller.Erratum{traveller.E002},
		},
	} {
		var codec eventCodec

		err := event.Fold(&codec)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		var written eventJSON

		err = json.Unmarshal(codec.out, &written)
		if err != nil {
			t.Errorf("%s: reading back what was written: %v", name, err)

			continue
		}

		if !reflect.DeepEqual(written, codec.flat) {
			t.Errorf("%s: the flat shape is not what was written\n  on the wire %+v\n  flat        %+v",
				name, written, codec.flat)
		}
	}
}
