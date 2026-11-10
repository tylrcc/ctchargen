package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// The flags v1.0.0 ships, each with its type and its default (#126).
//
// One direction only, on purpose. A flag here that a subcommand no longer
// declares, or declares with another type or default, breaks a script that
// passed it; a flag the subcommand gained is a new option and breaks nothing,
// so it is accepted here and held by TestEveryDeclaredFlagIsListed instead.
// Typed out rather than read back, for the reason usage_test.go gives.
const (
	baseForce  = "force bool false"
	baseOutput = "o string "
)

var cliBaseline = map[string][]string{
	cmdNew: {
		"auto bool false", "career string serve", baseForce,
		"muster string cash", "name string ", baseOutput,
		"service string ", "skills string advanced",
	},
	// new's "readings bool false" was removed on purpose before v1.0.0, with
	// the readings subcommand (#460): the readings are the authority's
	// record, not the referee's (ADR 0007), and a note names no erratum for
	// the flag to explain.
	// batch's "auto bool false" was removed on purpose, before v1.0.0
	// freezes the CLI: a batch is always decided by the policy, and the flag
	// said nothing (#336). Like seed and answers before it (#216), it went
	// while the CLI could still change; after v1.0.0 no entry is taken out.
	// batch's "force bool false" went the same way (#426): a batch writes only
	// into a folder of its own and never removes anything, so --force had
	// nothing left to do. So did "survivors bool false" (#421): a batch
	// writes only the living, always, so there is nothing for the flag to
	// switch on. And "name string " (#406): it gave every member one name,
	// which a referee never wants; a batch member is named in the vault.
	cmdBatch: {
		"career string serve", "count int 0",
		"muster string cash", baseOutput,
		"service string ", "skills string advanced",
	},
	// render, and new's --sheet and --history, were removed on purpose before
	// v1.0.0 with the JSON record they read and chose between (#398). The
	// note is the one thing written, so there is nothing to render back and
	// no second shape to ask for.
}

func TestEveryBaselineFlagIsStillDeclared(t *testing.T) {
	t.Parallel()

	for command, set := range map[string]*flag.FlagSet{
		cmdNew:   declareNewFlags().set,
		cmdBatch: declareBatchFlags().set,
	} {
		declared := map[string]string{}

		set.VisitAll(func(f *flag.Flag) {
			getter, ok := f.Value.(flag.Getter)
			if !ok {
				t.Errorf("%s --%s: a value the gate cannot read the type of", command, f.Name)

				return
			}

			declared[f.Name] = fmt.Sprintf("%s %T %s", f.Name, getter.Get(), f.DefValue)
		})

		for _, want := range cliBaseline[command] {
			name, _, _ := strings.Cut(want, " ")

			got, ok := declared[name]

			switch {
			case !ok:
				t.Errorf("%s no longer declares --%s", command, name)
			case got != want:
				t.Errorf("%s --%s is now %q, and the baseline is %q", command, name, got, want)
			}
		}
	}
}

// The readings are the authority's record, not the referee's (ADR 0007), so
// the command line has no readings subcommand and no --readings flag (#460).
// Each is refused as any unknown word is, with exit status 1.
func TestTheReadingsAreNotOnTheCommandLine(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args []string
		says string
	}{
		"the readings subcommand": {[]string{"readings"}, `unknown command "readings"`},
		"new --readings":          {[]string{cmdNew, flagAuto, "--readings"}, "-readings"},
	} {
		err := run(tc.args, nil, io.Discard, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: run(%v) was accepted", name, tc.args)
		case exitStatus(err) != 1:
			t.Errorf("%s: exit status %d, want 1 (%v)", name, exitStatus(err), err)
		case !strings.Contains(err.Error(), tc.says):
			t.Errorf("%s: error %q does not say %q", name, err, tc.says)
		}
	}
}

// The exit status is 0 for a run that did what it was asked and 1 for one
// that did not (#121), whichever kind of failure it was.
func TestTheExitStatusIsZeroOrOne(t *testing.T) {
	t.Parallel()

	_, searched := failedSearch(t).each(func(int, *chargen.Character) error { return nil })

	var record strings.Builder

	died := run([]string{cmdNew, flagAuto, flagService, "scouts"}, nil, &record, io.Discard,
		withDice(scoutKilled.roller(t)))
	if !strings.Contains(record.String(), "killed by the survival throw") {
		t.Fatal("the scout walk no longer dies, so the death case checks nothing")
	}

	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"version": {run([]string{cmdVersion}, nil, io.Discard, io.Discard), 0},
		"help":    {run([]string{cmdNew, flagHelp}, nil, io.Discard, io.Discard), 0},
		"usage":   {run(nil, nil, io.Discard, io.Discard), 1},
		"search":  {searched, 1},
		"a death": {died, 0},
	} {
		if got := exitStatus(tc.err); got != tc.want {
			t.Errorf("%s: exit status %d, want %d (%v)", name, got, tc.want, tc.err)
		}
	}
}

// usageRefusal is a command line refused as malformed: exit status 1, the
// error a usage error, and the message naming what was wrong.
func usageRefusal(t *testing.T, name string, err error, says ...string) {
	t.Helper()

	switch {
	case err == nil:
		t.Errorf("%s: accepted", name)

		return
	case exitStatus(err) != 1:
		t.Errorf("%s: exit status %d, want 1 (%v)", name, exitStatus(err), err)
	case !errors.Is(err, errUsage):
		t.Errorf("%s: %q is not a usage error", name, err)
	}

	for _, want := range says {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %q does not say %q", name, err, want)
		}
	}
}

// --count is decimal digits and nothing else (#466): "010" is ten, not the
// eight the flag package's base-0 reading made of it, and a hex or
// underscored literal is not a whole number at all.
//
// Holds: --count is decimal digits only.
func TestCountIsDecimal(t *testing.T) {
	t.Parallel()

	var notes strings.Builder

	err := run([]string{cmdBatch, flagCount, "010"}, nil, &notes, io.Discard)
	if err != nil {
		t.Fatalf("--count 010: %v", err)
	}

	if got := strings.Count(notes.String(), "## Generation record"); got != 10 {
		t.Errorf("--count 010 wrote %d notes, want 10", got)
	}

	for _, given := range []string{"0x10", "1_000", "+3", "-3", "3.0", " 3"} {
		err := run([]string{cmdBatch, flagCount, given}, nil, io.Discard, io.Discard)
		usageRefusal(t, "--count "+given, err, flagCount, "not a whole number")
	}
}

// An empty value, and a flag given twice, are refused, naming the flag
// (#473). Each is a line whose meaning is unclear, so it is refused now
// rather than given one.
//
// Holds: A flag given twice, or given an empty value, is refused.
func TestAnEmptyOrRepeatedFlagIsRefused(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, named := range []string{flagService, flagCareer, flagSkills, flagMuster, flagName, flagOutput} {
		err := run([]string{cmdNew, flagAuto, named, ""}, nil, io.Discard, io.Discard)
		usageRefusal(t, "new "+named+" ''", err, named, "empty")
	}

	for _, named := range []string{flagService, flagCareer, flagSkills, flagMuster, flagOutput} {
		err := run([]string{cmdBatch, flagCount, "1", named, ""}, nil, io.Discard, io.Discard)
		usageRefusal(t, "batch "+named+" ''", err, named, "empty")
	}

	for name, args := range map[string][]string{
		"--service twice": {cmdNew, flagAuto, flagService, navy, flagService, "army"},
		"--auto twice":    {cmdNew, flagAuto, flagAuto},
		"-o twice":        {cmdNew, flagAuto, flagOutput, filepath.Join(dir, "a.md"), flagOutput, filepath.Join(dir, "b.md")},
		"--count twice":   {cmdBatch, flagCount, "1", flagCount, "2"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		usageRefusal(t, name, err, strings.Fields(name)[0], "twice")
	}
}

// -o - is standard output, for both commands, exactly as no -o is (#467).
//
// Holds: -o - is standard output.
//
// Not parallel: it changes into an empty folder, to show nothing is written
// there.
func TestDashIsStandardOutput(t *testing.T) { //nolint:paralleltest // t.Chdir cannot run in a parallel test.
	for name, args := range map[string][]string{
		cmdNew:   {cmdNew, flagAuto, flagOutput, "-"},
		cmdBatch: {cmdBatch, flagCount, "1", flagOutput, "-"},
	} {
		dir := t.TempDir()

		t.Chdir(dir)

		var notes strings.Builder

		err := run(args, nil, &notes, io.Discard)
		if err != nil {
			t.Errorf("%s -o -: %v", name, err)

			continue
		}

		if !strings.Contains(notes.String(), "## Generation record") {
			t.Errorf("%s -o -: no note on standard output", name)
		}

		if entries, _ := os.ReadDir(dir); len(entries) > 0 {
			t.Errorf("%s -o -: wrote %s, where standard output was asked for", name, entries[0].Name())
		}
	}
}

// --force overwrites an -o file, so without one it has nothing to do and is
// refused rather than ignored (#473).
//
// Holds: --force without an -o file is refused.
//
// Not parallel: it changes into an empty folder, so a regression that writes
// a file named - writes it there and not into the package.
func TestForceWithoutAFileIsRefused(t *testing.T) {
	t.Chdir(t.TempDir())

	for name, args := range map[string][]string{
		"no -o": {cmdNew, flagAuto, flagForce},
		"-o -":  {cmdNew, flagAuto, flagForce, flagOutput, "-"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		usageRefusal(t, name, err, flagForce, flagOutput)
	}
}

// A batch's -o folder is made only inside a folder that exists (#470), as new
// writes only into one that exists: neither command makes the path to its
// output.
//
// Holds: A batch makes its own folder and no folder above it.
func TestABatchMakesOnlyItsOwnFolder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")

	err := run([]string{cmdBatch, flagCount, "1", flagOutput, filepath.Join(missing, "b")}, nil, io.Discard, io.Discard)

	switch {
	case err == nil:
		t.Error("a batch into a missing folder's subfolder was accepted")
	case exitStatus(err) != 1 || !strings.Contains(err.Error(), "is not a folder that exists"):
		t.Errorf("refused as %q, exit %d; want the missing folder named, exit 1", err, exitStatus(err))
	}

	_, statErr := os.Stat(missing)
	if statErr == nil {
		t.Errorf("%s was created", missing)
	}

	err = run([]string{cmdBatch, flagCount, "1", flagOutput, filepath.Join(dir, "b")}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("a batch into a new folder inside one that exists: %v", err)
	}
}
