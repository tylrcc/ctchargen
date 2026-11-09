package main

import (
	"flag"
	"fmt"
	"io"
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
	baseForce    = "force bool false"
	baseOutput   = "o string "
	baseReadings = "readings bool false"
)

var cliBaseline = map[string][]string{
	cmdNew: {
		"auto bool false", "career string serve", baseForce,
		"muster string cash", "name string ", baseOutput, baseReadings,
		"service string ", "skills string advanced",
	},
	// batch's "auto bool false" was removed on purpose, before v1.0.0
	// freezes the CLI: a batch is always decided by the policy, and the flag
	// said nothing (#336). Like seed and answers before it (#216), it went
	// while the CLI could still change; after v1.0.0 no entry is taken out.
	// batch's "force bool false" went the same way (#426): a batch writes only
	// into a folder of its own and never removes anything, so --force had
	// nothing left to do.
	cmdBatch: {
		"career string serve", "count int 0",
		"muster string cash", "name string ", baseOutput,
		"service string ", "skills string advanced", "survivors bool false",
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
		"version":    {run([]string{cmdVersion}, nil, io.Discard, io.Discard), 0},
		"help":       {run([]string{cmdNew, flagHelp}, nil, io.Discard, io.Discard), 0},
		"usage":      {run(nil, nil, io.Discard, io.Discard), 1},
		"search":     {searched, 1},
		"a death":    {died, 0},
		"readings":   {run([]string{cmdReadings}, nil, io.Discard, io.Discard), 0},
		"no reading": {run([]string{cmdReadings, "E099"}, nil, io.Discard, io.Discard), 1},
	} {
		if got := exitStatus(tc.err); got != tc.want {
			t.Errorf("%s: exit status %d, want %d (%v)", name, got, tc.want, tc.err)
		}
	}
}
