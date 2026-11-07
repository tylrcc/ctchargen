package main

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The command line's own words, so a typo in one place is a compile error
// rather than a test that quietly stops testing the flag it names.
const (
	cmdNew         = "new"
	cmdRender      = "render"
	cmdVersion     = "version"
	wantUsage      = "usage"
	wantNoRow      = "no such strategy"
	wantUnknown    = "unknown command"
	wantOneLine    = "more than one line"
	flagSheet      = "--sheet"
	flagAuto       = "--auto"
	flagSeed       = "--seed"
	flagService    = "--service"
	flagName       = "--name"
	flagCareer     = "--career"
	flagSkills     = "--skills"
	flagMuster     = "--muster"
	flagHistory    = "--history"
	flagForce      = "--force"
	flagOutput     = "-o"
	flagAnswers    = "--answers"
	wantSheet      = "sheet"
	wantTranscript = "transcript"
	other          = "other"
	navy           = "navy"
	merchants      = "merchants"
)

func TestRunRejectsBadCommandLines(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"no command":      {nil, "new|batch|render|version"},
		"unknown command": {[]string{"generate"}, wantUnknown},
		// Interactive mode with nothing to read from asks its first
		// question and reports that the input ended, rather than answering
		// it on the player's behalf.
		"no input to read": {[]string{cmdNew}, "the input ended"},
		"unknown flag":     {[]string{cmdNew, flagAuto, "--wat"}, wantUsage},
		"no such service":  {[]string{cmdNew, flagAuto, flagService, "navvy"}, "no service is called"},
		// A name is one line, on both commands that take one (#128).
		"a name with a newline": {[]string{cmdNew, flagAuto, flagName, "Jamison\n\n## Possessions"}, wantOneLine},
		"a name with a return":  {[]string{cmdNew, flagAuto, flagName, "Jamison\r## Possessions"}, wantOneLine},
		"a batch name with a newline": {
			[]string{cmdBatch, flagAuto, flagCount, "1", flagName, "Jamison\n# Forged"}, wantOneLine,
		},
		// All three, because each is now parsed by its own function at the
		// boundary rather than by one loop over a map. A test of one used
		// to reach the branch the other two take; it no longer does, and a
		// strategy flag nobody refuses is a word that reaches a record.
		"no such career": {[]string{cmdNew, flagAuto, flagCareer, "dawdle"}, wantNoRow},
		"no such skills": {[]string{cmdNew, flagAuto, flagSkills, "osmosis"}, wantNoRow},
		"no such muster": {[]string{cmdNew, flagAuto, flagMuster, "gold"}, wantNoRow},
		"no such batch muster": {
			[]string{cmdBatch, flagAuto, flagCount, "1", flagMuster, "gold"}, wantNoRow,
		},
		// The strategies are recorded whichever mode runs, and the schema
		// restricts them to POLICY.md's rows, so an unknown one is refused
		// without --auto too rather than being written into a record.
		"no such strategy, asking": {[]string{cmdNew, flagCareer, "dawdle"}, wantNoRow},
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

// The three renderings, and that each writes something recognisable.
func TestRunWritesEachRendering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"json": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"seed": 7`},
		// The command is what fills the build, so a record it writes says
		// which tool wrote it where a test-generated one says nothing.
		"json build": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"build"`},
		"json ruleset": {
			[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other},
			`"ruleset": "Classic Traveller, Books 1-3`,
		},
		"json upp":     {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"upp"`},
		wantSheet:      {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagSheet}, "UPP "},
		wantTranscript: {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagHistory}, "Generation record"},
		"version":      {[]string{cmdVersion}, "ctchargen"},
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

// mustRun runs a command line expected to work, and returns what it wrote to
// each channel, standard output first. A run that does not work fails the
// test, naming the command line.
func mustRun(t *testing.T, args []string, in io.Reader) (string, string) {
	t.Helper()

	var o, a strings.Builder

	err := run(args, in, &o, &a)
	if err != nil {
		t.Fatalf("ctchargen %s: %v", strings.Join(args, " "), err)
	}

	return o.String(), a.String()
}

// mustRunTogether is mustRun with both channels written into one, in the
// order they were written: an interactive session reads as a conversation
// only when its questions and its record are interleaved as the player saw
// them.
func mustRunTogether(t *testing.T, args []string, in io.Reader) string {
	t.Helper()

	var both strings.Builder

	err := run(args, in, &both, &both)
	if err != nil {
		t.Fatalf("ctchargen %s: %v", strings.Join(args, " "), err)
	}

	return both.String()
}

// parsedInputs is what new's flag set makes of a command line, without
// generating anything.
func parsedInputs(t *testing.T, args ...string) chargen.Inputs {
	t.Helper()

	flags := declareNewFlags()

	err := flags.set.Parse(args)
	if err != nil {
		t.Fatalf("parsing %v: %v", args, err)
	}

	in, _, err := flags.inputs()
	if err != nil {
		t.Fatalf("the inputs of %v: %v", args, err)
	}

	return in
}

// A run with no --seed draws one, and draws a different one each time, which
// is what makes an unseeded character reproducible afterwards. That the
// record then carries the seed is held by TestRunWritesEachRendering, from a
// seed the test names.
//
// It is asserted of the parsed flags rather than of a whole run: the draw
// is what is under test, and a whole run would roll a character to read one
// number out of it.
func TestADrawnSeedIsRecorded(t *testing.T) {
	t.Parallel()

	drawn := func() uint64 {
		t.Helper()

		in := parsedInputs(t, flagService, other)

		// Strictly above, not at: the bound is inclusive, and this said
		// otherwise because it was written against the call rather than
		// against the sentence the call was supposed to keep (#56).
		if in.Seed > maxDrawnSeed {
			t.Errorf("drew %d, which is above the bound a JSON reader parses exactly", in.Seed)
		}

		return in.Seed
	}

	if first, second := drawn(), drawn(); first == second {
		t.Errorf("two unseeded runs both drew %d", first)
	}
}

// A --seed given is the operator's own number, recorded as given rather than
// redrawn.
func TestAGivenSeedIsKept(t *testing.T) {
	t.Parallel()

	const given = 7

	in := parsedInputs(t, flagSeed, strconv.Itoa(given), flagService, other)

	if in.Seed != given {
		t.Errorf("seed %d, want the %d that was given", in.Seed, given)
	}
}

// A drawn seed can be the top of its own range.
//
// rand.Uint64N's argument is exclusive, so passing maxDrawnSeed left the
// documented top unreachable (#56) - and the test that guarded the bound
// asserted the code's behaviour rather than the comment's promise, so nothing
// said so. Drawing until 2^53-1 comes up is not a test anyone can run, which
// is why the source is a parameter.
func TestADrawnSeedReachesTheTopOfItsRange(t *testing.T) {
	t.Parallel()

	// The largest value an exclusive bound of n can yield.
	top := func(n uint64) uint64 { return n - 1 }

	if got := drawSeed(top); got != maxDrawnSeed {
		t.Errorf("the highest drawable seed is %d, and the bound says %d", got, maxDrawnSeed)
	}

	// And the reason for the bound, which is what it is for: every integer
	// up to it is exactly representable as a float64, and past it they are
	// not, so a reader parsing JSON numbers as doubles rounds silently.
	if uint64(float64(maxDrawnSeed)) != maxDrawnSeed {
		t.Error("the bound itself does not survive a float64, so it is the wrong bound")
	}

	if uint64(float64(maxDrawnSeed+2)) == maxDrawnSeed+2 {
		t.Error("two above the bound survives a float64, so the bound is lower than it need be")
	}
}
