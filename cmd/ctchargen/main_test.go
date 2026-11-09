package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// The command line's own words, so a typo in one place is a compile error
// rather than a test that quietly stops testing the flag it names.
const (
	cmdNew         = "new"
	cmdRender      = "render"
	cmdVersion     = "version"
	cmdReadings    = "readings"
	wantUsage      = "usage"
	wantNoStrategy = "no such strategy"
	wantUnknown    = "unknown command"
	wantOneLine    = "more than one line"
	flagSheet      = "--sheet"
	flagAuto       = "--auto"
	flagService    = "--service"
	flagName       = "--name"
	flagCareer     = "--career"
	flagSkills     = "--skills"
	flagMuster     = "--muster"
	flagHistory    = "--history"
	flagReadings   = "--readings"
	flagForce      = "--force"
	flagOutput     = "-o"
	wantSheet      = "sheet"
	wantTranscript = "transcript"
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
		"no command":      {nil, "new|batch|render|readings|version"},
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
		"a batch name with a newline": {
			[]string{cmdBatch, flagCount, "1", flagName, "Jamison\n# Forged"}, wantOneLine,
		},
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

// The three renderings, and that each writes something recognisable.
//
// Holds: The record names its ruleset and the build that wrote it.
func TestRunWritesEachRendering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"json": {[]string{cmdNew, flagAuto, flagService, other}, `"inputs"`},
		// The command is what fills the build, so a record it writes says
		// which tool wrote it where a test-generated one says nothing.
		"json build": {[]string{cmdNew, flagAuto, flagService, other}, `"build"`},
		// And so does every member of a batch, which generates on a path of
		// its own: one that skipped the build would ship unnoticed (#282).
		"batch build": {[]string{cmdBatch, flagCount, "1", flagService, other}, `"build"`},
		"json ruleset": {
			[]string{cmdNew, flagAuto, flagService, other},
			`"ruleset": "Classic Traveller, Books 1-3`,
		},
		"json upp":     {[]string{cmdNew, flagAuto, flagService, other}, `"upp"`},
		wantSheet:      {[]string{cmdNew, flagAuto, flagService, other, flagSheet}, "UPP "},
		wantTranscript: {[]string{cmdNew, flagAuto, flagService, other, flagHistory}, "Generation record"},
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

// The record carries the name it was given. Every fixture is unnamed, so
// nothing else would notice a record that wrote no name, or another one
// (#369).
func TestTheRecordCarriesTheNameGiven(t *testing.T) {
	t.Parallel()

	const given = "Alexander Jamison"

	record, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, other, flagName, given}, nil)

	var written struct {
		Name string `json:"name"`
	}

	err := json.Unmarshal([]byte(record), &written)
	if err != nil {
		t.Fatalf("reading the record: %v", err)
	}

	if written.Name != given {
		t.Errorf("the record is named %q; want %q", written.Name, given)
	}
}

// An auto run that names no strategy applies chargen.DefaultPolicy, and its
// record says so. The three names are typed out, not read from DefaultPolicy,
// so a default changed there is caught here, on the command a referee runs
// (#367).
func TestAnAutoRunNamingNoStrategyAppliesTheDefault(t *testing.T) {
	t.Parallel()

	record, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, other}, nil)

	type strategies struct {
		Career string `json:"career"`
		Skills string `json:"skills"`
		Muster string `json:"muster"`
	}

	var written struct {
		Policy strategies `json:"policy"`
	}

	err := json.Unmarshal([]byte(record), &written)
	if err != nil {
		t.Fatalf("reading the record: %v", err)
	}

	want := strategies{Career: serve, Skills: "advanced", Muster: "cash"}
	if written.Policy != want {
		t.Errorf("an auto run naming no strategy applied %+v; want %+v", written.Policy, want)
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
