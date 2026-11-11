package main

import (
	"errors"
	"flag"
	"io"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// The help's own words, and the flags each subcommand offers, typed out here
// rather than read back off the flag set the printer walks. Reading the set
// would agree with the printer however wrong both were; a second list is
// what notices a flag the printer stopped writing.
const (
	flagHelp  = "--help"
	flagShort = "-h"
	cmdHelp   = "help"
	topLevel  = "top level"
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
		flagMuster, flagOutput, flagForce,
	}
	batchFlags = []string{
		flagCount, flagService, flagCareer,
		flagSkills, flagMuster, flagOutput,
	}
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
		cmdNew:   {cmdNew, newFlags},
		cmdBatch: {cmdBatch, batchFlags},
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
		cmdNew:   {declareNewFlags().set, newFlags},
		cmdBatch: {declareBatchFlags().set, batchFlags},
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
		"cash, goods, arsenal", "(default serve)", "(default advanced)",
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

		for _, command := range []string{cmdNew, cmdBatch, cmdVersion} {
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
		cmdBatch:          {cmdBatch, flagCount, "1", wantTypo},
		// The two that take no word and have no flag set refuse one too,
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

// A help nobody received is not a help that was given. Exiting 0 on a write
// that failed would tell a script the flags had been printed, and the whole
// point of this issue is a reader who cannot see them.
func TestHelpReportsAWriteThatFailed(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args []string
		out  io.Writer
	}{
		topLevel:          {[]string{flagHelp}, &failsAfter{}},
		cmdNew:            {[]string{cmdNew, flagHelp}, &failsAfter{}},
		cmdVersion:        {[]string{cmdVersion, flagHelp}, &failsAfter{}},
		"the flags alone": {[]string{cmdNew, flagHelp}, &failsAfter{left: 1}},
	} {
		err := run(tc.args, nil, tc.out, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: a help nobody received was reported as written", name)
		case !errors.Is(err, errWriteFailed):
			t.Errorf("%s: error %q does not carry the write failure", name, err)
		}
	}
}

// `version --help` is a help and not the build. The top-level help tells the
// reader that every command it lists answers --help, and version has no flag
// set to answer it; answering with the build breaks that promise for a
// script that asked what the command takes.
func TestTheCommandWithoutFlagsAnswersHelp(t *testing.T) {
	t.Parallel()

	const want = "usage: ctchargen version\n"

	for _, asked := range []string{flagHelp, flagShort} {
		var out strings.Builder

		err := run([]string{cmdVersion, asked}, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("version %s: %v", asked, err)

			continue
		}

		if out.String() != want {
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
// Every path refuses it in the same sentence, typed out here rather than
// read from the code that writes it (#372).
//
// Holds: A word after a help flag is refused, not discarded.
func TestAWordAfterAHelpFlagIsRefused(t *testing.T) {
	t.Parallel()

	refusal := `help takes no arguments, and was given "` + wantTypo + `"`

	for name, args := range map[string][]string{
		topLevel:           {flagHelp, wantTypo},
		"top level, -h":    {flagShort, wantTypo},
		"top level, -help": {"-help", wantTypo},
		cmdHelp:            {cmdHelp, wantTypo},
		cmdVersion:         {cmdVersion, flagHelp, wantTypo},
		cmdNew:             {cmdNew, flagHelp, wantTypo},
		cmdBatch:           {cmdBatch, flagHelp, wantTypo},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) printed the help and swallowed the word", name, args)

			continue
		}

		if !strings.Contains(err.Error(), refusal) {
			t.Errorf("%s: error %q does not say %q", name, err, refusal)
		}
	}
}

// A strategy is the --auto policy's, so one given to a guided run is refused
// rather than accepted and then ignored: the player answers every choice it
// would have decided, and the record credits no choice to the policy. The flag's
// own default is refused too, because what is refused is the operator setting
// it, and every strategy flag has one.
//
// Holds: A strategy given without --auto is refused, not ignored.
func TestAStrategyWithoutAutoIsRefused(t *testing.T) {
	t.Parallel()

	for flag, value := range map[string]string{
		flagCareer: serve,
		flagSkills: advanced,
		flagMuster: cash,
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

// The help says how flags are written (#466): Go's flag package reads them,
// so one dash or two, a value after = or a space, a switch with none, and --
// where a flag could stand ends them. The top-level help says it, and so does
// each command's.
//
// Holds: The help says how flags are written.
func TestTheHelpSaysHowFlagsAreWritten(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		topLevel: {flagHelp},
		cmdNew:   {cmdNew, flagHelp},
		cmdBatch: {cmdBatch, flagHelp},
	} {
		var help strings.Builder

		err := run(args, nil, &help, io.Discard)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		for _, says := range []string{
			"Go's", "one dash or two", "--auto", "-auto", "= or a space", "takes none", "-- where a flag could stand ends them",
		} {
			if !strings.Contains(help.String(), says) {
				t.Errorf("%s --help does not say %q:\n%s", name, says, help.String())
			}
		}
	}
}

// The help names the kind of value each flag takes, as the flag package
// names it: a count is an int and a name a string. Wrapping every flag to
// refuse a repeat or an empty value hid the kind behind the wrapper, and the
// help printed "value" for each (#466).
//
// Holds: The help names the kind of value each flag takes.
func TestTheHelpNamesEachFlagsKind(t *testing.T) {
	t.Parallel()

	for command, want := range map[string][]string{
		cmdNew:   {"--name string", "-o file", "--service string", "--career strategy"},
		cmdBatch: {"--count int", "-o folder"},
	} {
		var help strings.Builder

		err := run([]string{command, flagHelp}, nil, &help, io.Discard)
		if err != nil {
			t.Fatalf("%s --help: %v", command, err)
		}

		for _, line := range want {
			if !strings.Contains(help.String(), line) {
				t.Errorf("%s --help does not say %q:\n%s", command, line, help.String())
			}
		}
	}
}

// The help says what each strategy does, not only what it is called (#563).
// Under each strategy flag's heading, every value the flag takes heads a line
// of its own with a description beside it, on both commands that take the
// flags, and no other value is described there. The values are read from
// chargen's alphabets, through the lists the rejection names, so a strategy
// added there that the help does not describe fails here.
//
// Holds: The help says what each strategy does.
func TestTheHelpSaysWhatEachStrategyDoes(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		flagCareer: chargen.CareerChoices(),
		flagSkills: chargen.SkillsChoices(),
		flagMuster: chargen.MusterChoices(),
	}

	for _, command := range []string{cmdNew, cmdBatch} {
		help, _ := mustRun(t, []string{command, flagHelp}, nil)

		for named, choices := range values {
			want := strings.Split(choices, ", ")
			described := describedUnder(help, named)

			for _, value := range want {
				if !described[value] {
					t.Errorf("%s --help describes no %s %s under its heading:\n%s", command, named, value, help)
				}
			}

			if len(described) != len(want) {
				t.Errorf("%s --help describes %v under %s, which takes %s", command, described, named, choices)
			}
		}
	}
}

// describedValue is a line describing one strategy: two spaces, the value,
// two spaces or more, and its description. A continuation line, all indent,
// names no value.
var describedValue = regexp.MustCompile(`^  (\S+) {2,}\S.*$`)

// describedUnder is the values a help describes under a strategy flag's
// heading: a line that starts with the flag, then the indented lines after it
// up to the next line that is not indented.
func describedUnder(help, named string) map[string]bool {
	described := map[string]bool{}
	under := false

	for line := range strings.SplitSeq(help, "\n") {
		switch {
		case strings.HasPrefix(line, named):
			under = true
		case !strings.HasPrefix(line, " "):
			under = false
		case under:
			found := describedValue.FindStringSubmatch(line)
			if found != nil {
				described[found[1]] = true
			}
		}
	}

	return described
}

// The help says what the strategies have in common, where the throws overrule
// them, and what each command does with a death (#563). A referee with only
// the binary learns that --service is an attempt, that every strategy submits
// to the draft and attempts every commission and promotion, that a 12 on
// reenlistment forces another term, how the skills table is designated, the
// cap on Table 2, and that weapons are drawn. Typed out here, not read from
// the code that prints them; matched across the help's line breaks. And none
// of it calls the character he, him or his.
//
// Holds: The help says what each strategy does.
func TestTheHelpSaysWhatTheThrowsDecide(t *testing.T) {
	t.Parallel()

	shared := []string{
		"--service is an enlistment attempt", "a failed throw is the draft",
		"the throws decide", "submits to the draft", "attempts every commission and promotion",
		"A 12 on reenlistment forces another term", "trained on least", "Education 8",
		"up to three rolls", "draws each weapon",
	}

	for command, own := range map[string][]string{
		cmdNew: {
			"the player picks the service", "--auto draws it", "writes the note all the same, and exits 0",
			"terms counts every term begun, including the term in which a character dies",
		},
		cmdBatch: {"each member's service is drawn", "writes only the living"},
	} {
		help, _ := mustRun(t, []string{command, flagHelp}, nil)
		said := strings.Join(strings.Fields(help), " ")

		for _, says := range append(slices.Clone(shared), own...) {
			if !strings.Contains(said, says) {
				t.Errorf("%s --help does not say %q:\n%s", command, says, help)
			}
		}

		if found := pronoun.FindString(help); found != "" {
			t.Errorf("%s --help calls the character %q:\n%s", command, found, help)
		}
	}
}

// pronoun is a pronoun for the character, which no text a referee reads uses
// (#474, decision 5): the sheet, the record, the session and the README name
// the character without one, and so does the help.
var pronoun = regexp.MustCompile(`(?i)\b(?:he|him|his|himself)\b`)
