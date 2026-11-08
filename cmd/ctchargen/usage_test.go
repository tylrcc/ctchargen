package main

import (
	"errors"
	"flag"
	"io"
	"slices"
	"strings"
	"testing"
)

// The help's own words, and the flags each subcommand offers, typed out here
// rather than read back off the flag set the printer walks. Reading the set
// would agree with the printer however wrong both were; a second list is
// what notices a flag the printer stopped writing.
const (
	flagHelp  = "--help"
	flagShort = "-h"
	cmdHelp   = "help"
	wantTypo  = "extra-arg"
)

// The three flag lists, in no order, because the help prints them sorted and
// this is about which flags are there.
//
// Each is held both ways. TestHelpNamesEveryFlag notices a flag the printer
// stopped writing; TestEveryDeclaredFlagIsListed notices a flag the set
// gained without the list learning it, which is the direction the list
// actually drifted - `new` once gained a flag and this list stayed at eleven
// (#133).
var (
	newFlags = []string{
		flagAuto, flagService, flagName, flagCareer, flagSkills,
		flagMuster, flagSheet, flagHistory, flagOutput, flagForce,
	}
	batchFlags = []string{
		flagCount, flagAuto, flagService, flagName, flagCareer,
		flagSkills, flagMuster, flagOutput, flagForce, flagSurvivors,
	}
	renderFlags = []string{flagHistory, flagOutput, flagForce}
)

// Help is not an error: it exits 0, and it names every flag the command has.
//
// Both spellings, because the referee who reported this tried `new --help`
// and `batch -h` and got the same "help requested" line from each.
func TestHelpNamesEveryFlag(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		command string
		flags   []string
	}{
		cmdNew:    {cmdNew, newFlags},
		cmdBatch:  {cmdBatch, batchFlags},
		cmdRender: {cmdRender, renderFlags},
	} {
		for _, asked := range []string{flagHelp, flagShort} {
			var out strings.Builder

			err := run([]string{tc.command, asked}, nil, &out, io.Discard)
			if err != nil {
				t.Errorf("%s %s: %v", name, asked, err)

				continue
			}

			help := out.String()

			if !strings.Contains(help, tc.command+" [flags]") &&
				!strings.HasPrefix(help, "usage: ctchargen "+tc.command) {
				t.Errorf("%s %s: no usage line: %q", name, asked, help)
			}

			// Anchored to the start of a printed line. An unanchored
			// search passes on a description: `new`'s strategy flags all
			// say "the --auto ... strategy", so a `--auto` the printer had
			// stopped writing would still be found in three of them.
			for _, wanted := range tc.flags {
				if !strings.Contains(help, "\n  "+wanted+" ") {
					t.Errorf("%s %s: the help does not name %s", name, asked, wanted)
				}
			}
		}
	}
}

// Every flag a command declares is in that command's list above.
//
// This is the one place the test reads the flag set, and it reads it only to
// hold the list to it: the list stays the statement of what ought to be
// there, and this reports a flag that arrived without the list learning it.
func TestEveryDeclaredFlagIsListed(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		set   *flag.FlagSet
		flags []string
	}{
		cmdNew:    {declareNewFlags().set, newFlags},
		cmdBatch:  {declareBatchFlags().set, batchFlags},
		cmdRender: {declareRenderFlags().set, renderFlags},
	} {
		tc.set.VisitAll(func(declared *flag.Flag) {
			spelled := "--" + declared.Name
			if len(declared.Name) == shortFlag {
				spelled = "-" + declared.Name
			}

			if !slices.Contains(tc.flags, spelled) {
				t.Errorf("%s declares %s, and its list does not name it", name, spelled)
			}
		})
	}
}

// The values a strategy flag takes, and the one it takes when left alone.
//
// This is where a `go install` user learns them before he gets one wrong,
// and the release notes he read document three strategies that were renamed
// under him. The rejection names the same values, from the same function.
func TestHelpNamesTheStrategiesAndTheirDefaults(t *testing.T) {
	t.Parallel()

	out, _ := mustRun(t, []string{cmdNew, flagHelp}, nil)

	for _, want := range []string{
		"serve, retire, oneterm", "advanced, service, personal",
		"cash, goods, spartan", "(default serve)", "(default advanced)",
		"(default cash)", merchants, navy,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the help does not mention %q", want)
		}
	}
}

// Help asked for generates nothing. `new` with no --auto and nothing to read
// from otherwise reports that the input ended, so a help that reached the
// procedure would say so.
func TestHelpDoesNotGenerate(t *testing.T) {
	t.Parallel()

	out, asking := mustRun(t, []string{cmdNew, flagHelp}, nil)

	if len(asking) > 0 {
		t.Errorf("new --help asked a question: %q", asking)
	}

	if strings.Contains(out, "UPP") {
		t.Errorf("new --help rolled a character: %q", out)
	}
}

// --help before a subcommand is named is a request, not a mistyped command.
func TestTopLevelHelpNamesEverySubcommand(t *testing.T) {
	t.Parallel()

	for _, asked := range []string{flagHelp, flagShort, cmdHelp} {
		var out strings.Builder

		err := run([]string{asked}, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("ctchargen %s: %v", asked, err)

			continue
		}

		for _, command := range []string{cmdNew, cmdBatch, cmdRender, cmdVersion} {
			if !strings.Contains(out.String(), command) {
				t.Errorf("ctchargen %s: does not name %s", asked, command)
			}
		}
	}
}

// A positional argument is refused rather than ignored. `new extra-arg` used
// to start an interactive session nobody asked for, so a typo became a
// session; and because the flag package stops at the first non-flag word,
// `new foo --auto` was an interactive session too.
func TestAPositionalArgumentIsRefused(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		cmdNew:            {cmdNew, wantTypo},
		"new before auto": {cmdNew, wantTypo, flagAuto},
		cmdBatch:          {cmdBatch, flagAuto, flagCount, "1", wantTypo},
		// The two that have no flag set of their own refuse a word too,
		// rather than printing as though nothing had been typed.
		cmdVersion: {cmdVersion, wantTypo},
		cmdHelp:    {cmdHelp, wantTypo},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) was accepted", name, args)

			continue
		}

		if !strings.Contains(err.Error(), wantTypo) {
			t.Errorf("%s: error %q does not name the argument", name, err)
		}
	}
}

// The two messages that list the subcommands read the same const, so they
// cannot disagree about the order again.
func TestTheSubcommandListsAgree(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no command": nil,
		wantUnknown:  {"generate"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) was accepted", name, args)

			continue
		}

		if !strings.Contains(err.Error(), commandList) {
			t.Errorf("%s: error %q does not list %q", name, err, commandList)
		}
	}
}

// haltingPipe lets one write through and fails after, so that the flag list
// meets a closed pipe the usage line got through.
type haltingPipe struct{ wrote bool }

func (h *haltingPipe) Write(text []byte) (int, error) {
	if h.wrote {
		return 0, errClosedPipe
	}

	h.wrote = true

	return len(text), nil
}

// A help nobody received is not a help that was given. Exiting 0 on a write
// that failed would tell a script the flags had been printed, and the whole
// point of this issue is a reader who cannot see them.
func TestHelpReportsAWriteThatFailed(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args []string
		out  io.Writer
	}{
		"top level":       {[]string{flagHelp}, brokenPipe{}},
		cmdNew:            {[]string{cmdNew, flagHelp}, brokenPipe{}},
		"the flags alone": {[]string{cmdNew, flagHelp}, &haltingPipe{}},
	} {
		err := run(tc.args, nil, tc.out, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: a help nobody received was reported as written", name)
		case !errors.Is(err, errClosedPipe):
			t.Errorf("%s: error %q does not carry the write failure", name, err)
		}
	}
}

// `version --help` is a help and not the build. The top-level help tells the
// reader that every command it lists answers --help, and version is one of
// the four; answering with the build breaks that promise for a script that
// asked what the command takes.
func TestVersionAnswersHelp(t *testing.T) {
	t.Parallel()

	for _, asked := range []string{flagHelp, flagShort} {
		var out strings.Builder

		err := run([]string{cmdVersion, asked}, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("version %s: %v", asked, err)

			continue
		}

		if out.String() != "usage: ctchargen version\n" {
			t.Errorf("version %s: not the usage: %q", asked, out.String())
		}
	}
}

// A word after a help flag is refused, wherever the help is asked for.
//
// #69 made refusing an unexpected word its whole subject. Four paths still
// dropped one in silence and disagreed with the top level: `ctchargen help
// junk` refused, while a help flag on any subcommand printed the help and
// exited 0 (#84). The flag package returns ErrHelp before a command reaches
// its own NArg check, so the word was never looked at; `version` has no flag
// set and tested args[0] alone.
//
// Holds: A word after a help flag is refused, not discarded.
func TestAWordAfterAHelpFlagIsRefused(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"top level":     {flagHelp, wantTypo},
		"top level, -h": {flagShort, wantTypo},
		cmdHelp:         {cmdHelp, wantTypo},
		cmdVersion:      {cmdVersion, flagHelp, wantTypo},
		cmdNew:          {cmdNew, flagHelp, wantTypo},
		cmdBatch:        {cmdBatch, flagHelp, wantTypo},
		cmdRender:       {cmdRender, flagHelp, wantTypo},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) printed the help and swallowed the word", name, args)

			continue
		}

		if !strings.Contains(err.Error(), wantTypo) {
			t.Errorf("%s: error %q does not name the word", name, err)
		}
	}
}

// One character is written one way.
//
// rendered() took the sheet and said nothing, so a referee who asked for the
// transcript got a sheet and no reason (#58). Preferring one silently is the
// same defect as swallowing a word, and this command refuses that.
//
// Holds: A character is written one way, not two.
func TestTheTwoRenderingsCannotBothBeAskedFor(t *testing.T) {
	t.Parallel()

	err := run([]string{cmdNew, flagAuto, flagSheet, flagHistory},
		nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a character was asked for as a sheet and a transcript at once")
	}

	for _, want := range []string{flagSheet, flagHistory} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %s", err, want)
		}
	}
}

// A strategy is the --auto policy's, so one given to a guided run is refused
// rather than accepted and then ignored: the player answers every choice it
// would have decided, and the record names no policy to carry it. The flag's
// own default is refused too, because what is refused is the operator setting
// it, and every strategy flag has one.
//
// Holds: A strategy given without --auto is refused, not ignored.
func TestAStrategyWithoutAutoIsRefused(t *testing.T) {
	t.Parallel()

	for flag, value := range map[string]string{
		flagCareer: "serve",
		flagSkills: "advanced",
		flagMuster: "cash",
	} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			err := run([]string{cmdNew, flag, value}, answers("1", 300), io.Discard, io.Discard)

			switch {
			case err == nil:
				t.Fatalf("a guided run took %s %s and decided nothing with it", flag, value)
			case !errors.Is(err, errUsage):
				t.Errorf("the refusal %q is not a usage error", err)
			case !strings.Contains(err.Error(), flag):
				t.Errorf("the refusal %q does not name %s", err, flag)
			}

			err = run([]string{cmdNew, flagAuto, flag, value}, nil, io.Discard, io.Discard)
			if err != nil {
				t.Errorf("an auto run refused %s %s: %v", flag, value, err)
			}
		})
	}
}
