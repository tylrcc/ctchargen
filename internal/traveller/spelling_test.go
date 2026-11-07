package traveller_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// No two values of an alphabet share a spelling.
//
// This is the assumption chargen's FR9 gate rests on. logging.record refuses
// an answer that was not offered by comparing rendered names, not values: the
// decorator holds both the offered set and the answer, and checking there
// leaves one definition of "answered outside the offer" rather than one per
// application site. That check is sound exactly as long as String() is
// injective on every alphabet the engine puts in an offered set.
//
// It was true when the gate was written and stated nowhere (#48). Two values
// that spelled the same would let a decider answer with one and have the
// other recorded, and nothing anywhere would notice.
//
// Every alphabet is here, not only the ones a Decider offers today, because
// what makes one reachable is a method signature - and a method that starts
// offering PassageClasses should not also have to remember this file.
//
// "Every" is a claim, so it is held rather than asserted in a comment: the
// test below reads this package's own source for the alphabets it declares
// and fails on one nobody checks here.
//
// Holds: The spellings that gate rests on tell every value of an alphabet
// apart.
func TestNoTwoValuesOfAnAlphabetShareASpelling(t *testing.T) {
	t.Parallel()

	spelledOnce(t, "Characteristic", traveller.Characteristics[:])
	spelledOnce(t, "Intent", traveller.Intents[:])
	spelledOnce(t, "MusterTable", traveller.MusterTables[:])
	spelledOnce(t, "ChoicePoint", traveller.ChoicePoints[:])
	spelledOnce(t, "Erratum", traveller.Errata[:])
	spelledOnce(t, "DecidedBy", traveller.DecidedBys[:])
	spelledOnce(t, "SkillTable", traveller.SkillTables[:])
	spelledOnce(t, "WeaponCategory", traveller.WeaponCategories[:])
	spelledOnce(t, "ServiceName", traveller.ServiceNames[:])
	spelledOnce(t, "PassageClass", traveller.PassageClasses[:])
	spelledOnce(t, "ShipKind", traveller.ShipKinds[:])
	spelledOnce(t, "Title", traveller.Titles[:])
}

// spelledOnce reports two values of one alphabet that render the same.
//
// The values are named by position rather than by %v, which would call the
// String() this is checking and print the same word twice.
func spelledOnce[T fmt.Stringer](t *testing.T, alphabet string, all []T) {
	t.Helper()

	if len(all) == 0 {
		t.Errorf("%s: no values, so this checks nothing", alphabet)

		return
	}

	seen := map[string]int{}

	for i, value := range all {
		spelled := value.String()

		first, taken := seen[spelled]
		if taken {
			t.Errorf("%s: values %d and %d both spell %q; the FR9 gate compares spellings",
				alphabet, first, i, spelled)

			continue
		}

		seen[spelled] = i
	}
}

// alphabet matches this package's own way of writing one down: an exported
// array of every value of a type, "var ServiceNames = [...]ServiceName{...}".
var alphabet = regexp.MustCompile(`(?m)^var ([A-Z]\w*) = \[\.\.\.\]`)

// Every alphabet this package declares is checked above.
//
// The list up there is written by hand, and a hand-written list of everything
// is a claim that goes stale the first time someone adds the thirteenth. The
// gap it would leave is the one #48 is about: a Decider method that starts
// offering a new alphabet, and a spelling gate that never looks at it.
//
// Reading the source is what holds it. This package is the whole domain and
// it declares its alphabets one way, so the pattern above finds them all.
func TestEveryAlphabetDeclaredHereIsChecked(t *testing.T) {
	t.Parallel()

	checked, err := os.ReadFile("spelling_test.go")
	if err != nil {
		t.Fatalf("reading this file: %v", err)
	}

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("looking for the package: %v", err)
	}

	var declared []string

	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		for _, found := range alphabet.FindAllSubmatch(text, -1) {
			declared = append(declared, string(found[1]))
		}
	}

	if len(declared) == 0 {
		t.Fatal("no alphabets found, so this checks nothing")
	}

	slices.Sort(declared)

	for _, name := range declared {
		if !strings.Contains(string(checked), "traveller."+name+"[:]") {
			t.Errorf("%s is an alphabet and no case above checks its spellings", name)
		}
	}
}
