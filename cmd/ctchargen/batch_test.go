package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
)

const (
	cmdBatch  = "batch"
	flagCount = "--count"
)

// A batch is refused without a count of at least one.
func TestBatchRefusesWithoutACount(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no --count": {cmdBatch},
		"zero count": {cmdBatch, flagCount, "0"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil || !errors.Is(err, errUsage) || !strings.Contains(err.Error(), "--count") {
			t.Errorf("%s: batch %v said %v, want a usage error naming --count", name, args, err)
		}
	}
}

// -o naming a directory writes one note per character, named by its position.
func TestBatchIntoADirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	mustRun(t, []string{
		cmdBatch, flagCount, "3", flagService, other, flagOutput, dir,
	}, nil)

	written, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}

	// A second run refuses to replace them, and --force replaces them.
	err = run([]string{
		cmdBatch, flagCount, "3", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("a second batch overwrote the first without --force")
	}

	err = run([]string{
		cmdBatch, flagCount, "3", flagService, other, flagOutput, dir, flagForce,
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("--force did not replace the members: %v", err)
	}
}

// `-o dir/` names a directory that need not already exist. Without the
// trailing separator being read as "a directory", the README's own line
// fails, and `-o characters` quietly writes one file where twenty
// notes were asked for.
func TestBatchCreatesTheOutputDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "characters") + string(filepath.Separator)

	mustRun(t, []string{
		cmdBatch, flagCount, "3", flagService, other, flagOutput, dir,
	}, nil)

	written, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}
}

// A collision anywhere in the batch is found before any of it is written, so
// a refused batch leaves the directory as it was.
//
// Holds: A refused batch writes none of itself, on the members it will actually
// write.
func TestARefusedBatchWritesNothing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// A note at the second position, already there.
	blocked := filepath.Join(dir, "2 Navy Admiral 777777.md")

	err := os.WriteFile(blocked, []byte("{}\n"), 0o600)
	if err != nil {
		t.Fatalf("planting the collision: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "4", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch overwrote an existing member without --force")
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 1 {
		t.Errorf("a refused batch left %d files behind, want only the one that was there",
			len(written))
	}
}

// A batch cannot make its output directory under an existing file, and says
// so rather than reporting the first member's open.
func TestBatchReportsADirectoryItCannotMake(t *testing.T) {
	t.Parallel()

	blocking := filepath.Join(t.TempDir(), "not-a-directory")

	err := os.WriteFile(blocking, []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("planting the file: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "2", flagService, other,
		flagOutput, filepath.Join(blocking, "members") + string(filepath.Separator),
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch wrote into a directory it could not make")
	}

	if !strings.Contains(err.Error(), "creating") {
		t.Errorf("the failure reads %q, which does not name the directory", err)
	}
}

// A write that fails is reported as the write, and abandons what it opened
// rather than leaving the descriptor held or finishing a partial note.
func TestAFailedWriteIsReportedAndAbandoned(t *testing.T) {
	t.Parallel()

	finished, abandoned := false, false

	where := destination{
		out: &failsAfter{},
		close: func() error {
			finished = true

			return nil
		},
		abandon: func() { abandoned = true },
	}

	err := where.write("anything")
	if !errors.Is(err, errWriteFailed) {
		t.Errorf("a failed write was reported as %v, want the write's own error", err)
	}

	if finished {
		t.Error("a failed write finished the destination as though it went through")
	}

	if !abandoned {
		t.Error("a failed write left the destination open")
	}
}

// A batch that stops partway still says what it left behind, and says it
// stopped.
//
// Here the third member's note is already a directory - every member walks
// the same dice, so its name is known - and --force skips the
// up-front check that would have refused the run before writing anything, so
// two notes are on disk when it fails. It printed the error and no tally,
// and the tally is the only place the run says how many of what it wrote are
// corpses (#118).
func TestABatchThatStopsSaysWhatItWrote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := os.Mkdir(filepath.Join(dir, "3 Other 644776.md"), 0o700)
	if err != nil {
		t.Fatalf("planting the directory: %v", err)
	}

	var asking strings.Builder

	err = run([]string{
		cmdBatch, flagCount, "5", flagService, other,
		flagForce, flagOutput, dir + string(filepath.Separator),
	}, nil, io.Discard, &asking, withDice(otherByPolicy.partly(t)))
	if err == nil {
		t.Fatal("a batch wrote a note over a directory")
	}

	if !strings.HasPrefix(asking.String(), "2 written before the run stopped, ") {
		t.Errorf("a batch that stopped after two notes said %q", asking.String())
	}
}

// When the tally of a stopped batch cannot be written either, both failures
// are reported: the run's own, which is what stopped it, and the tally's.
func TestAStoppedBatchThatCannotReportSaysBoth(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := os.Mkdir(filepath.Join(dir, "3 Other 644776.md"), 0o700)
	if err != nil {
		t.Fatalf("planting the directory: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "5", flagService, other,
		flagForce, flagOutput, dir + string(filepath.Separator),
	}, nil, io.Discard, &failsAfter{}, withDice(otherByPolicy.partly(t)))
	if err == nil {
		t.Fatal("a batch wrote a note over a directory")
	}

	if !errors.Is(err, errWriteFailed) || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("a stopped batch that could not report said %v, want both failures", err)
	}
}

// oldRecord is what a --force test plants before replacing it.
const oldRecord = "last week's\n"

// --force leaves the note it replaces whole until the new one is (#191).
// Opening with a truncating flag emptied the old note before the first byte
// of the new one, so a write that failed left neither.
func TestForceKeepsTheOldRecordUntilTheNewOneIsWhole(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.txt")

	err := os.WriteFile(path, []byte(oldRecord), noteFileMode)
	if err != nil {
		t.Fatalf("planting the old note: %v", err)
	}

	where, err := openDestination(nil, path, true)
	if err != nil {
		t.Fatalf("opening with --force: %v", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("opening with --force left the old note as %q", got)
	}

	where.out = &failsAfter{}

	err = where.write("this week's\n")
	if !errors.Is(err, errWriteFailed) {
		t.Fatalf("a failed write was reported as %v, want the write's own error", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("a failed --force write left the old note as %q", got)
	}

	where, err = openDestination(nil, path, true)
	if err != nil {
		t.Fatalf("reopening with --force: %v", err)
	}

	err = where.write("this week's\n")
	if err != nil {
		t.Fatalf("writing with --force: %v", err)
	}

	if got := mustRead(t, path); got != "this week's\n" {
		t.Errorf("a whole --force write left %q", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("listing the directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("--force left %d files beside the note, want none", len(entries)-1)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(data)
}

// Where the note cannot be staged, --force says so and leaves the old one.
func TestForceThatCannotStageLeavesTheOldRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.txt")

	err := os.WriteFile(path, []byte(oldRecord), noteFileMode)
	if err != nil {
		t.Fatalf("planting the old note: %v", err)
	}

	err = os.Chmod(dir, 0o400)
	if err != nil {
		t.Fatalf("closing the directory: %v", err)
	}

	_, err = openDestination(nil, path, true)
	if err == nil || !strings.Contains(err.Error(), "staging") {
		t.Errorf("--force into a closed directory said %v, want the staging failure", err)
	}

	err = os.Chmod(dir, noteDirMode)
	if err != nil {
		t.Fatalf("reopening the directory: %v", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("a --force that could not stage left the old note as %q", got)
	}
}

// Members are named first by their one-based position, zero-padded to the
// width of the batch and followed by a space, so a directory sorts in
// generation order and says nothing about seeds.
func TestBatchMembersAreNamedByPosition(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		count int
		wants []string
	}{
		"a batch of three": {count: 3, wants: []string{"1 ", "2 ", "3 "}},
		"a batch of twelve": {count: 12, wants: []string{
			"01 ", "02 ", "03 ", "04 ", "05 ", "06 ",
			"07 ", "08 ", "09 ", "10 ", "11 ", "12 ",
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			mustRun(t, []string{
				cmdBatch, flagCount, strconv.Itoa(tc.count), flagService, other, flagOutput, dir,
			}, nil)

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("looking for the members: %v", err)
			}

			if len(entries) != len(tc.wants) {
				t.Fatalf("%d members written, want %d", len(entries), len(tc.wants))
			}

			for i, entry := range entries {
				if !strings.HasPrefix(entry.Name(), tc.wants[i]) || !strings.HasSuffix(entry.Name(), ".md") {
					t.Errorf("member %d is named %q, want it to open %q and end .md", i+1, entry.Name(), tc.wants[i])
				}
			}
		})
	}
}

// A directory where a note is to go is reported as a directory, the first
// time and without --force advice (#254).
//
// The exclusive open read a directory as a file that already exists, and
// said "pass --force to replace it"; with --force it then failed as a
// directory. So the advice cost the referee a second run to learn the real
// obstruction, and --force could never have removed it.
//
// Holds: A directory in the way is named as one, with no --force advice.
func TestADirectoryInTheWayIsNamedAsOne(t *testing.T) {
	t.Parallel()

	for name, plant := range map[string]func(dir string) (args []string, blocked string){
		"new": func(dir string) ([]string, string) {
			blocked := filepath.Join(dir, "sheet.md")

			return []string{cmdNew, flagAuto, flagService, other, flagOutput, blocked}, blocked
		},
		"a batch member": func(dir string) ([]string, string) {
			blocked := filepath.Join(dir, "2 Navy Admiral 777777.md")

			return []string{cmdBatch, flagCount, "3", flagService, other, flagOutput, dir}, blocked
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			args, blocked := plant(dir)

			err := os.Mkdir(blocked, 0o700)
			if err != nil {
				t.Fatalf("planting the directory: %v", err)
			}

			err = run(args, nil, io.Discard, io.Discard)

			switch {
			case err == nil:
				t.Fatal("a note was written over a directory")
			case !strings.Contains(err.Error(), "is a directory"):
				t.Errorf("a directory in the way was reported as %q, which does not say so", err)
			case !strings.Contains(err.Error(), blocked):
				t.Errorf("a directory in the way was reported as %q, which does not name %s", err, blocked)
			case strings.Contains(err.Error(), "--force"):
				t.Errorf("a directory in the way was reported as %q, advising --force, which cannot help", err)
			}
		})
	}
}

// A batch is always an auto run: it has nobody to ask, so it needs no flag to
// say so (#336). `batch --count 1` writes a note that names the policy and
// each strategy given,
// and --auto, which no longer means anything to a batch, is the flag
// package's own usage error rather than a word accepted and ignored. A
// strategy is accepted without it, since it is always the policy's.
//
// Holds: A batch is always an auto run, and takes no --auto.
func TestABatchIsAlwaysAuto(t *testing.T) {
	t.Parallel()

	out, _ := mustRun(t, []string{
		cmdBatch, flagCount, "1", flagCareer, "retire", flagSkills, "service", flagMuster, "goods",
	}, nil)

	for _, want := range []string{"answered by the policy", "career retire", "skills service", "muster goods"} {
		if !strings.Contains(out, want) {
			t.Errorf("batch with all three strategies wrote no %s in its note:\n%s", want, out)
		}
	}

	err := run([]string{cmdBatch, flagAuto, flagCount, "1"}, nil, io.Discard, io.Discard)

	switch {
	case err == nil:
		t.Error("batch --auto was accepted")
	case !errors.Is(err, errUsage) || !strings.Contains(err.Error(), "auto"):
		t.Errorf("batch --auto: %v, want a usage error naming the flag", err)
	}
}

// A batch names each note for the character in it, after its position: the
// position, the service and rank, and the UPP, so a referee's vault lists
// "01 Army Captain 938788" where it listed "01" (#400). The names are typed
// out from the walks' notes, not read from the code that writes them.
//
// Holds: A batch member's note is named for its character.
func TestABatchNamesEachNoteForItsCharacter(t *testing.T) {
	t.Parallel()

	oneterm := []string{flagService, navy, flagCareer, "oneterm"}

	for name, tc := range map[string]struct {
		walk  walk
		flags []string
		want  string
	}{
		"an officer":            {navyOfficer, oneterm, "1 Navy Lieutenant A85379.md"},
		"a civilian":            {navyDeclined, oneterm, "1 Civilian 87665B.md"},
		"a service of no ranks": {otherByPolicy, []string{flagService, other}, "1 Other 644776.md"},
		"a death":               {scoutKilled, []string{flagService, scouts}, "1 Scouts 4387B9.md"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			args := append([]string{cmdBatch, flagCount, "1", flagOutput, dir}, tc.flags...)

			err := run(args, nil, io.Discard, io.Discard, withDice(tc.walk.roller(t)))
			if err != nil {
				t.Fatal(err)
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}

			if len(entries) != 1 || entries[0].Name() != tc.want {
				t.Errorf("wrote %v, want [%s]", entries, tc.want)
			}
		})
	}
}

// plantedNote is the note plantHeld leaves at the first position, for a
// character the batch will not generate.
const plantedNote = "1 Navy Admiral 777777.md"

// plantHeld is a directory whose first position is held by plantedNote.
func plantHeld(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, plantedNote), []byte(oldRecord), 0o600)
	if err != nil {
		t.Fatalf("planting the old note: %v", err)
	}

	return dir
}

// A position is held by any note named for it, whatever character it was
// for (#400), and without --force the batch is refused before anyone is
// generated, naming the note in the way.
//
// Holds: A refused batch writes none of itself, on the members it will
// actually write.
func TestAHeldPositionRefusesTheBatch(t *testing.T) {
	t.Parallel()

	err := run([]string{cmdBatch, flagCount, "1", flagService, other, flagOutput, plantHeld(t)},
		nil, io.Discard, io.Discard, withDice(func() chargen.Roller {
			t.Fatal("a refused batch generated a character")

			return nil
		}))
	if err == nil || !strings.Contains(err.Error(), plantedNote) {
		t.Errorf("a batch over a held position said %v, want it refused naming %s", err, plantedNote)
	}
}

// With --force, the old note at a position is replaced by the new one rather
// than left beside it.
func TestForceReplacesAPositionsNote(t *testing.T) {
	t.Parallel()

	dir := plantHeld(t)

	err := run([]string{cmdBatch, flagCount, "1", flagService, other, flagOutput, dir, flagForce},
		nil, io.Discard, io.Discard, withDice(otherByPolicy.roller(t)))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 || entries[0].Name() != "1 Other 644776.md" {
		t.Errorf("--force left %v, want the new note alone", entries)
	}
}

// A directory at a position is not a note, and --force leaves it.
func TestForceLeavesADirectoryAtAPosition(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	beside := filepath.Join(dir, "1 Not a note.md")

	err := os.Mkdir(beside, noteDirMode)
	if err != nil {
		t.Fatal(err)
	}

	err = run([]string{cmdBatch, flagCount, "1", flagService, other, flagOutput, dir, flagForce},
		nil, io.Discard, io.Discard, withDice(otherByPolicy.roller(t)))
	if err != nil {
		t.Fatal(err)
	}

	if !isDirectory(beside) {
		t.Error("--force removed a directory at the position, which is not a note")
	}
}

// With --force the new note is written before the old one goes, so a write
// that fails leaves the old note where it was (#191).
func TestAFailedForceKeepsThePositionsNote(t *testing.T) {
	t.Parallel()

	dir := plantHeld(t)

	err := os.Mkdir(filepath.Join(dir, "1 Other 644776.md"), noteDirMode)
	if err != nil {
		t.Fatal(err)
	}

	err = run([]string{cmdBatch, flagCount, "1", flagService, other, flagOutput, dir, flagForce},
		nil, io.Discard, io.Discard, withDice(otherByPolicy.roller(t)))
	if err == nil {
		t.Fatal("a batch wrote a note over a directory")
	}

	kept, err := os.ReadFile(filepath.Join(dir, plantedNote))
	if err != nil || string(kept) != oldRecord {
		t.Errorf("a failed --force write left the old note as %q (%v)", kept, err)
	}
}
