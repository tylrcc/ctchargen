package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A flag in an error is written as the help writes it (#472): the flag
// package names every flag with one dash, whichever the operator typed, and
// the help, the README and the other errors write --count and -o.
//
// Holds: A flag in an error is written as the help writes it.
func TestAFlagInAnErrorIsWrittenAsTheHelpWritesIt(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"an unknown flag": {
			[]string{cmdNew, flagAuto, "--output", "kira.md"}, "flag provided but not defined: --output",
		},
		"an unknown short flag":  {[]string{cmdNew, flagAuto, "-x"}, "flag provided but not defined: -x"},
		"a value that is no int": {[]string{cmdBatch, flagCount, "many"}, `invalid value "many" for flag --count:`},
		"a value that is no bool": {
			[]string{cmdNew, "--auto=maybe"}, `invalid boolean value "maybe" for --auto:`,
		},
		"a missing value":        {[]string{cmdNew, flagAuto, flagOutput}, "flag needs an argument: -o"},
		"a missing long value":   {[]string{cmdBatch, flagCount}, "flag needs an argument: --count"},
		"one dash on a long one": {[]string{cmdBatch, "-count", "x"}, `invalid value "x" for flag --count:`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := run(tc.args, nil, io.Discard, io.Discard)

			switch {
			case err == nil:
				t.Fatalf("ctchargen %v was accepted", tc.args)
			case !strings.Contains(err.Error(), tc.want):
				t.Errorf("ctchargen %v said %q, want it to say %q", tc.args, err, tc.want)
			case !errors.Is(err, errUsage):
				t.Errorf("ctchargen %v said %q, which is not a usage error", tc.args, err)
			}
		})
	}
}

// Only a mistake in the command line is called usage (#472, #508). A file in
// the way, a missing folder, a folder in use and a build with nothing
// embedded are found at run time; a reader told "usage" goes looking for a
// flag he typed wrong.
func TestOnlyACommandLineMistakeIsCalledUsage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existing := filepath.Join(dir, "kira.md")
	plantFile("kira.md")(t, dir)

	for name, err := range map[string]error{
		"a file in the way": run([]string{cmdNew, flagAuto, flagOutput, existing}, nil, io.Discard, io.Discard),
		"a missing folder": run([]string{cmdNew, flagAuto, flagOutput, filepath.Join(dir, "gone", "kira.md")},
			nil, io.Discard, io.Discard),
		"a folder in use": run([]string{cmdBatch, flagCount, "1", flagOutput, dir}, nil, io.Discard, io.Discard),
		"no build":        writeVersion("", io.Discard),
	} {
		switch {
		case err == nil:
			t.Errorf("%s was accepted", name)
		case errors.Is(err, errUsage) || strings.HasPrefix(err.Error(), wantUsage):
			t.Errorf("%s was reported as %q, a usage error", name, err)
		}
	}

	for name, args := range map[string][]string{
		"a flag batch has not": {cmdBatch, flagCount, "1", flagForce},
		"a word new takes not": {cmdNew, "kira"},
		"a batch -o for a file": {
			cmdBatch, flagCount, "1", flagOutput, filepath.Join(dir, "crew.md"),
		},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil || !errors.Is(err, errUsage) || !strings.HasPrefix(err.Error(), wantUsage+": ") {
			t.Errorf("%s was reported as %v, want a usage error", name, err)
		}
	}
}

// searchOnly is the mode of a folder a test can find a file in and cannot
// write into.
const searchOnly = 0o500

// A path appears once in a message (#472). The error an open returns names
// its path, and the message wrapped around it named it again:
// `opening ro/x.md: open ro/x.md: permission denied`.
//
// Holds: A path is named once in a message.
func TestAPathIsNamedOnce(t *testing.T) {
	t.Parallel()

	closed := filepath.Join(t.TempDir(), "ro")
	unread := filepath.Join(t.TempDir(), "unread")

	for _, dir := range []string{closed, unread} {
		err := os.Mkdir(dir, noteDirMode)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = os.Chmod(dir, noteDirMode) })
	}

	blocked := filepath.Join(t.TempDir(), "sheet.md")
	plantDir("sheet.md")(t, filepath.Dir(blocked))

	for _, chmod := range []struct {
		dir  string
		mode os.FileMode
	}{{closed, searchOnly}, {unread, 0o300}} {
		err := os.Chmod(chmod.dir, chmod.mode)
		if err != nil {
			t.Fatal(err)
		}
	}

	note := filepath.Join(closed, "x.md")

	for name, tc := range map[string]struct {
		args []string
		path string
	}{
		"a folder it cannot write": {[]string{cmdNew, flagAuto, flagOutput, note}, note},
		"--force into it":          {[]string{cmdNew, flagAuto, flagOutput, note, flagForce}, note},
		"a directory in the way":   {[]string{cmdNew, flagAuto, flagOutput, blocked}, blocked},
		"a folder it cannot read":  {[]string{cmdBatch, flagCount, "1", flagOutput, unread}, unread},
	} {
		err := run(tc.args, nil, io.Discard, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s was accepted", name)
		case strings.Count(err.Error(), tc.path) != 1:
			t.Errorf("%s was reported as %q, which does not name %s exactly once", name, err, tc.path)
		}
	}
}
