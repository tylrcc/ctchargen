package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The modes an ordinary file and folder are created with, typed here rather
// than read from noteFileMode and noteDirMode, so a mode changed there is
// caught rather than copied.
const (
	ordinaryFile   = 0o666
	ordinaryFolder = 0o777
)

// A note and a batch's folder take the operator's umask (#469): every note was
// 0600 and every folder 0700, whatever the umask said. The umask is the
// process's, so the test does not set one: it compares what the command makes
// with what an ordinary 0666 file and 0777 folder get from the same umask.
//
// Holds: A new note and a new folder take the operator's umask.
func TestANoteAndAFolderTakeTheUmask(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	note, crew := filepath.Join(dir, "kira.md"), filepath.Join(dir, "crew")

	plain, folder := filepath.Join(dir, "plain"), filepath.Join(dir, "folder")

	file, err := os.OpenFile(plain, os.O_WRONLY|os.O_CREATE, ordinaryFile)
	if err == nil {
		err = file.Close()
	}

	if err == nil {
		err = os.Mkdir(folder, ordinaryFolder)
	}

	if err != nil {
		t.Fatal(err)
	}

	if modeOf(t, plain) == 0o600 {
		t.Skip("the umask is 077, which makes 0666 the 0600 the old notes were; nothing here tells them apart")
	}

	mustRun(t, []string{cmdNew, flagAuto, flagOutput, note}, nil)
	mustRun(t, []string{cmdBatch, flagCount, "1", flagOutput, crew}, nil)

	members, err := filepath.Glob(filepath.Join(crew, "*.md"))
	if err != nil || len(members) != 1 {
		t.Fatalf("the batch wrote %v (%v), want one member", members, err)
	}

	for path, like := range map[string]string{note: plain, crew: folder, members[0]: plain} {
		if got, want := modeOf(t, path), modeOf(t, like); got != want {
			t.Errorf("%s is %v, want the %v the umask gives", path, got, want)
		}
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	return info.Mode()
}

// --force overwrites the file in place (#469), so the operating system keeps
// what belongs to the file: its mode, a symlink to it, and a hard link to it.
// It replaced the file with a new one, so the mode was 0600, a symlink became
// a file of its own, and a hard link kept the old note.
//
// Holds: --force overwrites the file in place.
func TestForceOverwritesTheFileInPlace(t *testing.T) {
	t.Parallel()

	note := filepath.Join(t.TempDir(), "kira.md")
	plantFile(filepath.Base(note))(t, filepath.Dir(note))

	// A mode no umask gives a new note, so keeping it is not an accident.
	const kept = 0o604

	err := os.Chmod(note, kept)
	if err != nil {
		t.Fatal(err)
	}

	link, hard := filepath.Join(t.TempDir(), "link.md"), filepath.Join(t.TempDir(), "hard.md")

	err = os.Symlink(note, link)
	if err == nil {
		err = os.Link(note, hard)
	}

	if err != nil {
		t.Fatal(err)
	}

	mustRun(t, []string{cmdNew, flagAuto, flagOutput, link, flagForce}, nil)

	if got := modeOf(t, note); got != kept {
		t.Errorf("--force left the note %v, want the %v it had", got, os.FileMode(kept))
	}

	to, err := os.Readlink(link)
	if err != nil || to != note {
		t.Errorf("--force left %s as %q (%v), want it still a link to %s", link, to, err, note)
	}

	for _, path := range []string{note, hard} {
		if got := mustRead(t, path); !strings.HasPrefix(got, "---\nname:") {
			t.Errorf("--force through the link left %s as %q, want the new note", path, got)
		}
	}
}

// --force says it overwrites the file in place (#469).
func TestForceSaysItOverwritesInPlace(t *testing.T) {
	t.Parallel()

	var help strings.Builder

	err := run([]string{cmdNew, "--help"}, nil, &help, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(help.String(), "overwrite the output file in place") {
		t.Errorf("new --help does not say --force overwrites in place:\n%s", help.String())
	}
}

// A write that fails without --force removes the note (#468), names the note
// once, and the same command run again is not refused. A note left truncated
// under its own name made the rerun refuse it as existing. Under --force a
// failed write leaves the file as it left it, as the shell's > does, whether
// a note was there or not (#469).
//
// Holds: A failed write leaves no note under its name.
func TestAFailedWriteLeavesNoNote(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		plant bool
		setUp func(dir string) (string, []string)
		left  int
	}{
		"new -o": {setUp: func(dir string) (string, []string) {
			return dir, []string{cmdNew, flagAuto, flagOutput, filepath.Join(dir, "kira.md")}
		}},
		"a batch member": {setUp: func(dir string) (string, []string) {
			crew := filepath.Join(dir, "crew")

			return crew, []string{cmdBatch, flagCount, "2", flagOutput, crew}
		}},
		"--force onto nothing": {left: 1, setUp: func(dir string) (string, []string) {
			return dir, []string{cmdNew, flagAuto, flagOutput, filepath.Join(dir, "kira.md"), flagForce}
		}},
		"--force onto a note": {plant: true, left: 1, setUp: func(dir string) (string, []string) {
			return dir, []string{cmdNew, flagAuto, flagOutput, filepath.Join(dir, "kira.md"), flagForce}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			watched, args := tc.setUp(t.TempDir())
			if tc.plant {
				plantFile("kira.md")(t, watched)
			}

			status, said := limited(t, "1024", args...)
			if status != 1 || strings.Count(said, watched) != 1 || !strings.Contains(said, "file too large") {
				t.Errorf("a write past the limit exited %d saying %q, want 1 and the failure, naming the note once",
					status, said)
			}

			entries, err := os.ReadDir(watched)
			if err != nil {
				t.Fatal(err)
			}

			if len(entries) != tc.left {
				t.Errorf("a failed write left %d entries in %s, want %d", len(entries), watched, tc.left)
			}

			status, said = limited(t, "", args...)
			if status != 0 {
				t.Errorf("the same command run again exited %d saying %q, want 0", status, said)
			}
		})
	}
}

// A new note is never written over a file, even one that appeared after the
// command looked: the open is exclusive, and refuses it as in the way.
func TestANoteDoesNotReplaceAFileThatAppearedMeanwhile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "kira.md")
	plantFile("kira.md")(t, dir)

	_, err := openDestination(nil, path, false)
	if !errors.Is(err, errWouldOverwrite) || !strings.HasSuffix(err.Error(), "; pass --force to overwrite it") {
		t.Errorf("a note onto a file that appeared meanwhile said %v, want it refused as in the way, "+
			"advising --force in the help's word", err)
	}

	if got := mustRead(t, path); got != "mine\n" {
		t.Errorf("the file that appeared was left as %q", got)
	}
}

// --force onto a symlink to nothing creates the note the link names, and the
// link survives (#469): the open creates or truncates, as the shell's > does.
//
// Holds: --force overwrites the file in place.
func TestForceThroughALinkToNothingCreatesTheNote(t *testing.T) {
	t.Parallel()

	note := filepath.Join(t.TempDir(), "kira.md")
	link := filepath.Join(t.TempDir(), "link.md")

	err := os.Symlink(note, link)
	if err != nil {
		t.Fatal(err)
	}

	mustRun(t, []string{cmdNew, flagAuto, flagOutput, link, flagForce}, nil)

	if got := mustRead(t, note); !strings.HasPrefix(got, "---\nname:") {
		t.Errorf("--force through a link to nothing left %s as %q, want the new note", note, got)
	}

	to, err := os.Readlink(link)
	if err != nil || to != note {
		t.Errorf("--force left %s as %q (%v), want it still a link to %s", link, to, err, note)
	}
}
