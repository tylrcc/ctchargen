package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The render subcommand, end to end.
//
// It shipped with no test through the command at all, and three of the
// findings against this change lived in the branches no test reached.

// writeRecord generates a character into a temporary file and returns its
// path, which is what render is given. It walks otherByPolicy, so a test
// that generates the same character again gets the same one.
func writeRecord(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "character.json")

	mustRun(t, []string{cmdNew, flagAuto, flagService, other, flagOutput, path}, nil,
		withDice(otherByPolicy.roller(t)))

	return path
}

// A record read back renders to what generating it rendered.
func TestRenderReadsARecordBack(t *testing.T) {
	t.Parallel()

	path := writeRecord(t)

	// Read from the record rather than typed here. The UPP a walk makes is
	// not a fact about rendering, and #34 moved it - a hand-typed one is a
	// golden kept somewhere goldens are not regenerated.
	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		wantSheet:      {[]string{cmdRender, path}, "UPP " + uppIn(t, path)},
		wantTranscript: {[]string{cmdRender, flagHistory, path}, "Generation record"},
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

	// And the sheet render matches what generating it produced, which is the
	// same walk generated again.
	direct, _ := mustRun(t, []string{cmdNew, flagAuto, flagService, other, flagSheet}, nil,
		withDice(otherByPolicy.roller(t)))

	viaFile, _ := mustRun(t, []string{cmdRender, path}, nil)

	if direct != viaFile {
		t.Error("a record read back rendered differently from the run that wrote it")
	}
}

func TestRenderWritesToAFile(t *testing.T) {
	t.Parallel()

	source := writeRecord(t)
	out := filepath.Join(t.TempDir(), "sheet.md")

	mustRun(t, []string{cmdRender, flagOutput, out, source}, nil)

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading the sheet: %v", err)
	}

	if !strings.Contains(string(written), "UPP ") {
		t.Error("the written sheet has no UPP line")
	}

	// The same refusal new makes: an existing file is not replaced.
	err = run([]string{cmdRender, flagOutput, out, source}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("render replaced an existing file without --force")
	}

	err = run([]string{cmdRender, flagOutput, out, flagForce, source}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("--force did not replace the sheet: %v", err)
	}
}

func TestRenderRejectsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	notARecord := filepath.Join(dir, "sheet.md")

	err := os.WriteFile(notARecord, []byte("# a character sheet\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"no file named": {[]string{"render"}, wantUsage},
		"two files":     {[]string{cmdRender, notARecord, notARecord}, wantUsage},
		"unknown flag":  {[]string{cmdRender, "--wat", notARecord}, wantUsage},
		"absent file":   {[]string{cmdRender, filepath.Join(dir, "nowhere.json")}, "reading"},
		"not a record":  {[]string{cmdRender, notARecord}, "not a character record"},
	} {
		err := run(tc.args, nil, io.Discard, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: render accepted %v", name, tc.args)
		case !strings.Contains(err.Error(), tc.mentions):
			t.Errorf("%s: error %q does not mention %q", name, err, tc.mentions)
		}
	}
}

// -o that cannot be opened is reported rather than swallowed.
func TestOutputRefusesAPathItCannotOpen(t *testing.T) {
	t.Parallel()

	unwritable := filepath.Join(t.TempDir(), "no-such-directory", "character.json")

	err := run([]string{cmdNew, flagAuto, flagService, other, flagOutput, unwritable},
		nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("new wrote to a path whose directory does not exist")
	}
}

// version reports the build, from the same source a record's build stamp
// comes from.
func TestVersionReportsTheBuild(t *testing.T) {
	t.Parallel()

	out, _ := mustRun(t, []string{cmdVersion}, nil)

	if !strings.Contains(out, "ctchargen") {
		t.Errorf("version reported %q", out)
	}
}

// uppIn is the UPP of a written record.
func uppIn(t *testing.T, path string) string {
	t.Helper()

	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	var record struct {
		UPP string `json:"upp"`
	}

	err = json.Unmarshal(text, &record)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if record.UPP == "" {
		t.Fatalf("%s carries no UPP", path)
	}

	return record.UPP
}
