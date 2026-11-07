// Package docsgate holds the documents to the code, in both directions.
//
// They exist for a failure mode nothing else catches: a reading stamped on a
// record its condition does not describe, or a choice point no record
// reaches. Each gate is verified by breaking it; a gate
// never seen to fail has not been shown to hold.
//
// That every erratum has an entry in the readings document, and every entry
// an erratum, is not held here: the rules package lifts the document, and a
// lift that fails either way is a build defect.
//
// The package has no non-test code on purpose. It is the one place allowed
// to know about both traveller and chargen and about the documents beside
// them; the readings document is the exception, read by the rules package
// that lifts it.
//
// Implements: The errata a record stamps are exactly those whose condition
// ERRATA.md states [Documents] in The record.
// Implements: Every step the errata conditions read is found, as the kind they
// read it as [Documents] in The record.
// Implements: Every choice point the engine offers is reached by a record
// [Documents] in The record.
package docsgate_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// docs is where the governing documents live, relative to this package.
const docs = "../../docs"

func read(t *testing.T, name string) string {
	t.Helper()

	text, err := os.ReadFile(filepath.Join(docs, name))
	if err != nil {
		t.Fatalf("the gate cannot read the document it guards: %v", err)
	}

	if len(text) == 0 {
		t.Fatalf("%s is empty", name)
	}

	return string(text)
}

// compare reports every disagreement between two sets, not only the first.
func compare(t *testing.T, inCode, inDoc []string, code, doc string) {
	t.Helper()

	for _, name := range inCode {
		if !slices.Contains(inDoc, name) {
			t.Errorf("%s is in %s but has no entry in %s", name, code, doc)
		}
	}

	for _, name := range inDoc {
		if !slices.Contains(inCode, name) {
			t.Errorf("%s is in %s but does not exist in %s", name, doc, code)
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

	slices.Sort(names)

	return names
}

// ChoicePoint is meant to be a rendering of the Decider interface rather
// than a second list kept parallel to it. It is a second list — this is what
// stops it drifting into a different one.
func TestChoicePointsMatchTheDecider(t *testing.T) {
	t.Parallel()

	inCode := make([]string, 0, len(traveller.ChoicePoints))
	for _, c := range traveller.ChoicePoints {
		inCode = append(inCode, c.String())
	}

	slices.Sort(inCode)

	compare(t, inCode, deciderMethods(), "the ChoicePoint enum", "the Decider interface")
}

// The PRD's fourth gate - every erratum heading reachable by some path - is
// held in chargen's TestEveryReadingIsReachable, because reaching a reading
// takes the engine and the golden roster, and both live there. It owes
// nothing now: every reading but E012 is carried by some generated or
// scripted character, and E012 by design names no record.
