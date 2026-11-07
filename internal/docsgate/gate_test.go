// Package docsgate holds the documents to the code, in both directions.
//
// They exist for a failure mode nothing else catches: a reading stamped on a
// record its condition does not describe, or a choice point the engine grew
// without POLICY.md noticing. Each gate is verified by breaking it; a gate
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
package docsgate_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// docs is where the governing documents live, relative to this package.
const docs = "../../docs"

// policyRow matches POLICY.md's own row form, a level-three heading whose
// text is the Go method it describes: "### `SubmitToDraft() (bool, error)`".
var policyRow = regexp.MustCompile("(?m)^### `([A-Z][A-Za-z0-9]*)\\(")

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

// found returns the first capture of every match, sorted and deduplicated.
//
// A repeat is reported before it is compacted away. Comparing sets cannot
// notice one — two rows for the same method agree with the code as well as
// one does — and this is the only place positioned to: a second POLICY.md
// row for a method is a second answer to a question that has one.
func found(t *testing.T, re *regexp.Regexp, text, what string) []string {
	t.Helper()

	matches := re.FindAllStringSubmatch(text, -1)

	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}

	if len(names) == 0 {
		t.Fatalf("no %s found; the gate's pattern no longer matches the document", what)
	}

	slices.Sort(names)

	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			t.Errorf("%s: %s appears more than once; an id is never reused", what, names[i])
		}
	}

	return slices.Compact(names)
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

// policyVersion matches POLICY.md's own way of stating it: "**Version 3.**"
// on a line of its own near the top.
var policyVersion = regexp.MustCompile(`(?m)^\*\*Version (\d+)\.\*\*`)

// POLICY.md's version is the one every record carries.
//
// The document decides what a bump means and the code writes the number, so
// the two can disagree - and a record would then name a policy that is not
// the one that answered it, which is worse than carrying no number at all.
// This milestone has found a document and a constant disagreeing four times;
// this is the one place it would be unfalsifiable from the record alone.
func TestThePolicyVersionMatchesTheDocument(t *testing.T) {
	t.Parallel()

	found := policyVersion.FindStringSubmatch(read(t, "POLICY.md"))
	if found == nil {
		t.Fatal("POLICY.md states no version; every record carries one")
	}

	stated, err := strconv.Atoi(found[1])
	if err != nil {
		t.Fatalf("POLICY.md's version %q is not a number: %v", found[1], err)
	}

	if stated != chargen.PolicyVersion {
		t.Errorf("POLICY.md is version %d and the records say %d",
			stated, chargen.PolicyVersion)
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

// Every POLICY.md row names a Decider method, and every method has a row.
// This is what makes the auto policy total: a method with no row is a
// question the policy was never told how to answer.
func TestPolicyRowsMatchTheDecider(t *testing.T) {
	t.Parallel()

	inDoc := found(t, policyRow, read(t, "POLICY.md"), "policy rows")

	compare(t, deciderMethods(), inDoc, "the Decider interface", "POLICY.md")
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

// coverageTest matches a test COVERAGE.md cites, as `package.TestName`.
var coverageTest = regexp.MustCompile("`([a-z]+)\\.(Test[A-Za-z0-9]+)`")

// coverageGolden matches a fixture COVERAGE.md cites, as golden `name`.
var coverageGolden = regexp.MustCompile("golden `([a-z0-9-]+)`")

// testFunctions is every test in the module, keyed by the package directory
// that declares it.
func testFunctions(t *testing.T) map[string]map[string]bool {
	t.Helper()

	declaration := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9]+)\(`)
	found := map[string]map[string]bool{}

	err := filepath.WalkDir("../..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}

		text, err := os.ReadFile(path) //nolint:gosec // a path the walk produced
		if err != nil {
			return err //nolint:wrapcheck // the walk's own error, unchanged
		}

		pkg := filepath.Base(filepath.Dir(path))
		for _, m := range declaration.FindAllStringSubmatch(string(text), -1) {
			if found[pkg] == nil {
				found[pkg] = map[string]bool{}
			}

			found[pkg][m[1]] = true
		}

		return nil
	})
	if err != nil {
		t.Fatalf("looking for tests: %v", err)
	}

	return found
}

// Every test COVERAGE.md names exists.
//
// The document's own preamble says a row with no test is a defect, and it has
// twice carried a citation that named a test which does not hold the rule
// beside it - once a golden that covered nothing, once a test that checked
// the wrong thing. A citation nothing checks is how that happens: the rest of
// the file is held to the code and this column was held to nothing.
func TestCoverageCitesTestsThatExist(t *testing.T) {
	t.Parallel()

	tests := testFunctions(t)
	cited := coverageTest.FindAllStringSubmatch(read(t, "COVERAGE.md"), -1)

	if len(cited) == 0 {
		t.Fatal("COVERAGE.md cites no tests; the gate's pattern no longer matches the document")
	}

	for _, citation := range cited {
		pkg, name := citation[1], citation[2]
		if !tests[pkg][name] {
			t.Errorf("COVERAGE.md cites %s.%s, which no test file declares", pkg, name)
		}
	}
}

// Every golden COVERAGE.md names exists.
func TestCoverageCitesGoldensThatExist(t *testing.T) {
	t.Parallel()

	cited := coverageGolden.FindAllStringSubmatch(read(t, "COVERAGE.md"), -1)

	if len(cited) == 0 {
		t.Fatal("COVERAGE.md cites no goldens; the gate's pattern no longer matches the document")
	}

	for _, citation := range cited {
		fixture := filepath.Join("..", "..", "chargen", "testdata", citation[1]+".json")

		_, err := os.Stat(fixture)
		if err != nil {
			t.Errorf("COVERAGE.md cites golden %q, which is not in chargen/testdata", citation[1])
		}
	}
}
