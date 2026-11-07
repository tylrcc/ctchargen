package chargen_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// Every choice point the engine offers is reached by some golden record.
//
// Until #34 nothing held the choice points to the roster, and Weapon was the
// result: eleven of the twelve points were exercised by a golden and that one
// was not, so what the policy answered there was covered by no character at
// all - before it changed and after.
//
// A choice point nothing reaches is the same defect as a rule with no test,
// one level up: the policy says what it answers, the code answers it, and
// nothing has seen the answer.
//
// Holds: Every choice point the engine offers is reached by a record.
func TestEveryChoicePointIsReachedByARecord(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}

	for _, rec := range readRecords(t) {
		for _, e := range rec.Events {
			if e.Kind == "choice" {
				reached[e.Point] = true
			}
		}
	}

	if len(reached) == 0 {
		t.Fatal("no record answers any choice point")
	}

	for _, point := range traveller.ChoicePoints {
		if !reached[point.String()] {
			t.Errorf("no golden record reaches %v, so what the policy answers there is untested",
				point)
		}
	}
}

// ChoicePoint is meant to be a rendering of the Decider interface rather
// than a second list kept parallel to it. It is a second list — this is what
// stops it drifting into a different one.
func TestChoicePointsMatchTheDecider(t *testing.T) {
	t.Parallel()

	points := make([]string, 0, len(traveller.ChoicePoints))
	for _, c := range traveller.ChoicePoints {
		points = append(points, c.String())
	}

	sameNames(t, points, "a ChoicePoint", deciderMethods(), "a method of Decider")
}

// sameNames reports every name in one list and not the other, both ways, and
// not only the first.
func sameNames(t *testing.T, these []string, theseAre string, those []string, thoseAre string) {
	t.Helper()

	for _, name := range these {
		if !slices.Contains(those, name) {
			t.Errorf("%s is %s but not %s", name, theseAre, thoseAre)
		}
	}

	for _, name := range those {
		if !slices.Contains(these, name) {
			t.Errorf("%s is %s but not %s", name, thoseAre, theseAre)
		}
	}
}

// deciderMethods is every method of the Decider interface, by name.
func deciderMethods() []string {
	iface := reflect.TypeFor[chargen.Decider]()

	names := make([]string, 0, iface.NumMethod())
	for m := range iface.Methods() {
		names = append(names, m.Name)
	}

	return names
}

// rewriteYesOrNo writes over the alternatives of every yes-or-no choice in
// events, and says how many it wrote to.
func rewriteYesOrNo(events ...traveller.Event) int {
	written := 0

	for _, e := range events {
		choice, ok := e.(traveller.ChoiceEvent)
		if ok && slices.Contains(choice.Alternatives, "yes") {
			choice.Alternatives[0] = "maybe"
			written++
		}
	}

	return written
}

// A write through a yes-or-no choice's alternatives changes that event and
// nothing else: not the next question's offer, and not the gate that checks
// its answer.
//
// All six yes-or-no points once offered one package-level pair and stored it
// uncopied in every event, so a reader that wrote "maybe" into one finished
// record made every later "yes" refused as not offered (#230). A watcher,
// handed each event as it is written, could do the same mid-generation.
//
// The two writers are subtests so that each can be run alone, which is how
// each was shown to fail against the shared pair: once one has written to
// it, every later generation fails, and the other proves nothing.
func TestAYesOrNoOfferIsEachEventsOwn(t *testing.T) {
	t.Parallel()

	t.Run("watcher", func(t *testing.T) {
		t.Parallel()

		written := 0

		for _, fixture := range fixtures {
			watch := chargen.WithObserver(func(e traveller.Event) {
				written += rewriteYesOrNo(e)
			})

			_, err := fixture.run(t, watch)
			if err != nil {
				t.Errorf("%s, watched by a writer: %v", fixture.name, err)
			}
		}

		if written < 2 {
			t.Fatal("no yes-or-no choice follows one a watcher wrote to, so this checks nothing")
		}
	})

	t.Run("record", func(t *testing.T) {
		t.Parallel()

		written := 0

		for _, fixture := range fixtures {
			written += rewriteYesOrNo(fixture.generate(t).Events...)
		}

		if written == 0 {
			t.Fatal("no fixture reaches a yes-or-no choice, so this checks nothing")
		}

		for _, fixture := range fixtures {
			_, err := fixture.run(t)
			if err != nil {
				t.Errorf("%s, after a record was written to: %v", fixture.name, err)
			}
		}
	})
}
