package main

import (
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The command line's own words, so a typo in one place is a compile error
// rather than a test that quietly stops testing the flag it names.
const (
	cmdNew         = "new"
	cmdVersion     = "version"
	wantUsage      = "usage"
	wantNoStrategy = "no such strategy"
	wantUnknown    = "unknown command"
	wantOneLine    = "more than one line"
	flagAuto       = "--auto"
	flagService    = "--service"
	flagName       = "--name"
	flagCareer     = "--career"
	flagSkills     = "--skills"
	flagMuster     = "--muster"
	flagForce      = "--force"
	flagOutput     = "-o"
	other          = "other"
	navy           = "navy"
	merchants      = "merchants"
	serve          = "serve"
)

func TestRunRejectsBadCommandLines(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"no command":      {nil, "new|batch|version"},
		"unknown command": {[]string{"generate"}, wantUnknown},
		// Interactive mode with nothing to read from asks its first
		// question and reports that the input ended, rather than answering
		// it on the player's behalf.
		"no input to read": {[]string{cmdNew}, "the input ended"},
		"unknown flag":     {[]string{cmdNew, flagAuto, "--wat"}, wantUsage},
		// The two flags #209 removed are refused like any unknown flag, so a
		// script that still passes them fails loudly rather than being ignored.
		"a seed":            {[]string{cmdNew, flagAuto, "--seed", "7"}, wantUsage},
		"a list of answers": {[]string{cmdNew, "--answers", "1,2,1"}, wantUsage},
		"no such service":   {[]string{cmdNew, flagAuto, flagService, "navvy"}, "no service is called"},
		// A name is one line, on both commands that take one (#128).
		"a name with a newline": {[]string{cmdNew, flagAuto, flagName, "Jamison\n\n## Possessions"}, wantOneLine},
		"a name with a return":  {[]string{cmdNew, flagAuto, flagName, "Jamison\r## Possessions"}, wantOneLine},
		// A name reaches the note's properties as YAML, which reads a byte
		// that is not UTF-8 back as another letter.
		"a name that is not UTF-8": {[]string{cmdNew, flagAuto, flagName, "Jami\xffson"}, "not UTF-8"},
		// A batch has no name to give its members (#406): every one would get
		// the same, so the flag is refused as one batch does not have.
		"a batch name": {[]string{cmdBatch, flagCount, "1", flagName, "Kira Vance"}, "flag provided but not defined: --name"},
		// All three, because each is now parsed by its own function at the
		// boundary rather than by one loop over a map. A test of one used
		// to reach the branch the other two take; it no longer does, and a
		// strategy flag nobody refuses is a word that reaches a record.
		"no such career": {[]string{cmdNew, flagAuto, flagCareer, "dawdle"}, wantNoStrategy},
		"no such skills": {[]string{cmdNew, flagAuto, flagSkills, "osmosis"}, wantNoStrategy},
		"no such muster": {[]string{cmdNew, flagAuto, flagMuster, "gold"}, wantNoStrategy},
		"no such batch muster": {
			[]string{cmdBatch, flagCount, "1", flagMuster, "gold"}, wantNoStrategy,
		},
		// Without --auto a strategy is refused for being given at all, but a
		// word that names none is still refused as naming none: the flags are
		// parsed before anything asks whether they were wanted.
		"no such strategy, asking": {[]string{cmdNew, flagCareer, "dawdle"}, wantNoStrategy},
	} {
		err := run(tc.args, nil, io.Discard, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: run(%v) was accepted", name, tc.args)
		case !strings.Contains(err.Error(), tc.mentions):
			t.Errorf("%s: error %q does not mention %q", name, err, tc.mentions)
		}
	}
}

// The note, and that it carries the sheet, the generation record and the
// build that wrote it, in its ctchargen property. The command is what fills
// the build, so a note it writes names it where a test-generated one leaves
// it empty; and so does every member of a batch, which generates on a path of
// its own: one that skipped the build would ship unnoticed (#282).
//
// Holds: A note names the build that wrote it.
func TestRunWritesTheNote(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"the sheet":             {[]string{cmdNew, flagAuto, flagService, other}, "UPP "},
		"the generation record": {[]string{cmdNew, flagAuto, flagService, other}, "## Generation record"},
		"the build":             {[]string{cmdNew, flagAuto, flagService, other}, "\nctchargen: \""},
		"a batch's build":       {[]string{cmdBatch, flagCount, "1", flagService, other}, "\nctchargen: \""},
		"version":               {[]string{cmdVersion}, "ctchargen"},
	} {
		var out strings.Builder

		err := run(tc.args, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(out.String(), tc.mentions) {
			t.Errorf("%s: output does not mention %q", name, tc.mentions)
		}
	}
}

// The note carries the name it was given, as its heading. Every fixture is
// unnamed, so nothing else would notice a note that wrote no name, or
// another one (#369).
func TestTheNoteCarriesTheNameGiven(t *testing.T) {
	t.Parallel()

	const given = "Alexander Jamison"

	note, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, other, flagName, given}, nil)

	if !strings.HasPrefix(note, "---\nname: \""+given+"\"\n") || !strings.Contains(note, "\n# "+given+"\n") {
		t.Errorf("the note does not carry %q as its name property and heading:\n%s", given, note)
	}
}

// An auto run that names no strategy applies chargen.DefaultPolicy. The three
// names are typed out, not read from DefaultPolicy, so a default changed there
// is caught here, at the flags a referee types (#367).
func TestAnAutoRunNamingNoStrategyAppliesTheDefault(t *testing.T) {
	t.Parallel()

	flags := declareNewFlags()

	err := flags.set.Parse([]string{flagAuto})
	if err != nil {
		t.Fatal(err)
	}

	_, policy, err := flags.inputs()
	if err != nil {
		t.Fatal(err)
	}

	if got := []string{policy.Career.String(), policy.Skills.String(), policy.Muster.String()}; !slices.Equal(
		got, []string{"serve", "advanced", "cash"},
	) {
		t.Errorf("an auto run naming no strategy applies %v, want serve, advanced and cash", got)
	}
}

// mustRun runs a command line expected to work, and returns what it wrote to
// each channel, standard output first. A run that does not work fails the
// test, naming the command line.
func mustRun(t *testing.T, args []string, in io.Reader, options ...option) (string, string) {
	t.Helper()

	var o, a strings.Builder

	err := run(args, in, &o, &a, options...)
	if err != nil {
		t.Fatalf("ctchargen %s: %v", strings.Join(args, " "), err)
	}

	return o.String(), a.String()
}

// mustRunTogether is mustRun with both channels written into one, in the
// order they were written: an interactive session reads as a conversation
// only when its questions and its record are interleaved as the player saw
// them.
func mustRunTogether(t *testing.T, args []string, in io.Reader, options ...option) string {
	t.Helper()

	var both strings.Builder

	err := run(args, in, &both, &both, options...)
	if err != nil {
		t.Fatalf("ctchargen %s: %v", strings.Join(args, " "), err)
	}

	return both.String()
}

// A note's ctchargen property is the version of the build that wrote it, as
// the version subcommand reports it (#476): a vault can query which build
// wrote a note, and the key already names the tool, so the module path is
// left out.
//
// Holds: A note's ctchargen property is the build's version.
func TestTheCtchargenPropertyIsTheBuildsVersion(t *testing.T) {
	t.Parallel()

	var reported, note strings.Builder

	err := run([]string{cmdVersion}, nil, &reported, io.Discard)
	if err != nil {
		t.Fatalf("version: %v", err)
	}

	fields := strings.Fields(reported.String())
	if len(fields) == 0 {
		t.Fatal("the version subcommand reported nothing")
	}

	version := fields[len(fields)-1]

	err = run([]string{cmdNew, flagAuto}, nil, &note, io.Discard)
	if err != nil {
		t.Fatalf("new --auto: %v", err)
	}

	if want := "\nctchargen: " + strconv.Quote(version) + "\n"; !strings.Contains(note.String(), want) {
		t.Errorf("the note's properties do not carry %q:\n%s", want, note.String())
	}
}

// An auto run applies the strategies given, all the way to the record: under
// --career oneterm the policy leaves after the first term, where the default
// would continue. The footer used to show the strategies; with it gone (#474's
// PR), the choice they decide is what shows them.
//
// The dice are live, and a character can die before the end of the first term
// is asked about, so runs are made until one reaches it.
func TestAnAutoRunAppliesTheStrategiesGiven(t *testing.T) {
	t.Parallel()

	const (
		attempts = 100
		asked    = "leave the service or remain for another term (pp. 6-7, 21): "
	)

	for range attempts {
		note, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, other, flagCareer, "oneterm"}, nil)

		for line := range strings.Lines(note) {
			if !strings.Contains(line, asked) {
				continue
			}

			if !strings.Contains(line, "policy chose discharge from") {
				t.Errorf("--career oneterm did not leave after the first term:\n%s", line)
			}

			return
		}
	}

	t.Fatalf("no run in %d reached the end of a term", attempts)
}
