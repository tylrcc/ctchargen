package chargen_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
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
