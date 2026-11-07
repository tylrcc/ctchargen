package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// The flags v1.0.0 ships, each with its type and its default (#126).
//
// One direction only, on purpose. A flag here that a subcommand no longer
// declares, or declares with another type or default, breaks a script that
// passed it; a flag the subcommand gained is a new option and breaks nothing,
// so it is accepted here and held by TestEveryDeclaredFlagIsListed instead.
// Typed out rather than read back, for the reason usage_test.go gives.
const (
	baseForce   = "force bool false"
	baseOutput  = "o string "
	baseHistory = "history bool false"
)

var cliBaseline = map[string][]string{
	cmdNew: {
		"auto bool false", "career string serve", baseForce,
		baseHistory, "muster string cash", "name string ", baseOutput,
		"service string ", "sheet bool false", "skills string advanced",
	},
	cmdBatch: {
		"auto bool false", "career string serve", "count int 0", baseForce,
		"muster string cash", "name string ", baseOutput,
		"service string ", "skills string advanced", "survivors bool false",
	},
	cmdRender: {baseForce, baseHistory, baseOutput},
}

func TestEveryBaselineFlagIsStillDeclared(t *testing.T) {
	t.Parallel()

	for command, set := range map[string]*flag.FlagSet{
		cmdNew:    declareNewFlags().set,
		cmdBatch:  declareBatchFlags().set,
		cmdRender: declareRenderFlags().set,
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

	_, searched := eachMember(chargen.Inputs{
		Service: traveller.Scouts, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}, chargen.DefaultPolicy(), 20, true, 1, scoutKilled.roller(t), func(int, *chargen.Character) error { return nil })

	var record strings.Builder

	died := run([]string{cmdNew, flagAuto, flagService, "scouts"}, nil, &record, io.Discard,
		withDice(scoutKilled.roller(t)))
	if !strings.Contains(record.String(), `"fatal": true`) {
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
		"unreadable": {run([]string{cmdRender, "no-such-record.json"}, nil, io.Discard, io.Discard), 1},
		"a death":    {died, 0},
	} {
		if got := exitStatus(tc.err); got != tc.want {
			t.Errorf("%s: exit status %d, want %d (%v)", name, got, tc.want, tc.err)
		}
	}
}
