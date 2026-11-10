package main

import (
	"errors"
	"io"
	"io/fs"
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

	// A second batch into the same folder is refused: it is not empty.
	err = run([]string{
		cmdBatch, flagCount, "3", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "is not empty") {
		t.Errorf("a second batch into the same folder said %v, want it refused as not empty", err)
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

// A batch cannot make its output directory under an existing file, and says
// so, naming the file that is in the way, rather than reporting the first
// member's open. Since #470 it is refused before anything is made, as a
// folder that does not exist.
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

	if !strings.Contains(err.Error(), blocking+" is not a folder that exists") {
		t.Errorf("the failure reads %q, which does not name what is in the way", err)
	}
}

// A write that fails is reported as the write, and closes what it opened
// rather than leaving the descriptor held.
func TestAFailedWriteIsReportedAndClosed(t *testing.T) {
	t.Parallel()

	where, err := openDestination(nil, filepath.Join(t.TempDir(), "kira.md"), false)
	if err != nil {
		t.Fatal(err)
	}

	where.out = &failsAfter{}

	err = where.write("anything")
	if !errors.Is(err, errWriteFailed) {
		t.Errorf("a failed write was reported as %v, want the write's own error", err)
	}

	_, err = where.file.Write([]byte("more"))
	if !errors.Is(err, os.ErrClosed) {
		t.Errorf("a failed write left the file open: a write after it said %v", err)
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

	blocked := filepath.Join(t.TempDir(), "sheet.md")

	err := os.Mkdir(blocked, 0o700)
	if err != nil {
		t.Fatalf("planting the directory: %v", err)
	}

	err = run([]string{cmdNew, flagAuto, flagService, other, flagOutput, blocked}, nil, io.Discard, io.Discard)

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

// A batch writes into a folder of its own: one that is new, or empty but for
// hidden entries such as the .DS_Store Finder leaves (#426). Anything a
// person would see in a listing - a note, another file, a subfolder - makes
// the folder not empty, and the batch is refused before anyone is generated,
// naming the folder.
//
// Holds: A batch writes only into a folder of its own.
// Holds: A batch refuses its folder before generating anybody.
func TestABatchWritesOnlyIntoAFolderOfItsOwn(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		plant   func(t *testing.T, dir string)
		refused bool
	}{
		"a note of the referee's": {plant: plantFile("3 Session recap.md"), refused: true},
		"another file":            {plant: plantFile("map.png"), refused: true},
		"a subfolder":             {plant: plantDir("handouts"), refused: true},
		"only hidden entries":     {plant: plantFile(".DS_Store"), refused: false},
		"nothing":                 {plant: func(*testing.T, string) {}, refused: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tc.plant(t, dir)

			dice := otherByPolicy.roller(t)
			if tc.refused {
				dice = func() chargen.Roller {
					t.Fatal("a refused batch generated a character")

					return nil
				}
			}

			err := run([]string{cmdBatch, flagCount, "1", flagService, other, flagOutput, dir},
				nil, io.Discard, io.Discard, withDice(dice))

			switch {
			case tc.refused && (err == nil || !strings.Contains(err.Error(), "is not empty") ||
				!strings.Contains(err.Error(), dir)):
				t.Errorf("a batch into a folder with %s said %v, want it refused naming the folder", name, err)
			case !tc.refused && err != nil:
				t.Errorf("a batch into a folder with %s was refused: %v", name, err)
			}
		})
	}
}

func plantFile(name string) func(t *testing.T, dir string) {
	return func(t *testing.T, dir string) {
		t.Helper()

		err := os.WriteFile(filepath.Join(dir, name), []byte("mine\n"), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func plantDir(name string) func(t *testing.T, dir string) {
	return func(t *testing.T, dir string) {
		t.Helper()

		err := os.Mkdir(filepath.Join(dir, name), noteDirMode)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// closingOn plays a walk for every member, and closes dir - to writing and to
// entering - as member is generated, so that member's note cannot be written and the run
// stops with the members before it on disk. The folder is reopened when the
// test ends, so its own cleanup can remove it.
func closingOn(t *testing.T, dir string, member int, w walk) source {
	t.Helper()

	t.Cleanup(func() { _ = os.Chmod(dir, noteDirMode) })

	played := w.partly(t)
	generated := 0

	return func() chargen.Roller {
		generated++
		if generated == member {
			err := os.Chmod(dir, 0o400)
			if err != nil {
				t.Fatalf("closing the folder: %v", err)
			}
		}

		return played()
	}
}

// A batch that stops partway says how many it wrote (#118), and that its
// folder holds them now: the folder is not empty any more, so running again
// needs it removed or another folder (#426).
func TestABatchThatStopsSaysWhatItWrote(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "crew")

	var asking strings.Builder

	err := run([]string{cmdBatch, flagCount, "5", flagService, other, flagOutput, dir},
		nil, io.Discard, &asking, withDice(closingOn(t, dir, 3, otherByPolicy)))
	if err == nil {
		t.Fatal("a batch into a closed folder reported success")
	}

	said := asking.String()

	if !strings.HasPrefix(said, "2 written before the run stopped\n") {
		t.Errorf("a batch that stopped after two notes said %q", said)
	}

	want := dir + " holds what was written; remove it or choose another folder to run again"
	if !strings.Contains(said, want) {
		t.Errorf("a stopped batch said %q, want it to say %q", said, want)
	}
}

// When the tally of a stopped batch cannot be written either, both failures
// are reported: the run's own, which is what stopped it, and the tally's.
func TestAStoppedBatchThatCannotReportSaysBoth(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "crew")

	err := run([]string{cmdBatch, flagCount, "5", flagService, other, flagOutput, dir},
		nil, io.Discard, &failsAfter{}, withDice(closingOn(t, dir, 3, otherByPolicy)))

	if !errors.Is(err, errWriteFailed) || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("a stopped batch that could not report said %v, want both failures", err)
	}
}

// A batch's -o names a folder, so one ending in .md, which almost certainly
// meant a file, is refused before anything is made (#416). And a batch has no
// --force (#426): it writes only into a folder of its own and never removes
// anything, so there is nothing for --force to do, and the flag package
// refuses it as a flag batch does not have.
//
// Holds: A batch's -o names a folder, and a batch takes no --force.
func TestABatchTakesAFolderAndNoForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Spelled with a trailing separator, or in capitals, it is still a .md
	// name, and a case-insensitive filesystem treats crew.MD as crew.md.
	for _, spelled := range []string{"crew.md", "crew.md" + string(filepath.Separator), "crew.MD"} {
		target := filepath.Join(dir, spelled)

		err := run([]string{cmdBatch, flagCount, "1", flagOutput, target}, nil, io.Discard, io.Discard)
		if err == nil || !errors.Is(err, errUsage) || !strings.Contains(err.Error(), "folder") {
			t.Errorf("batch -o %s said %v, want a usage error saying -o names a folder", spelled, err)
		}

		_, statErr := os.Stat(target)
		if statErr == nil {
			t.Errorf("batch -o %s made %s", spelled, target)
		}
	}

	err := run([]string{cmdBatch, flagCount, "1", flagOutput, dir, flagForce}, nil, io.Discard, io.Discard)
	if err == nil || !errors.Is(err, errUsage) || !strings.Contains(err.Error(), "force") {
		t.Errorf("batch --force said %v, want the flag refused", err)
	}
}
