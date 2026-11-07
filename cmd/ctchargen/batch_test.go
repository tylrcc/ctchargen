package main

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The seed arithmetic, which is what a batch means.

const (
	cmdBatch  = "batch"
	flagCount = "--count"
)

// P. of the PRD's Determinism section: "Members number from zero, so
// `batch --count 1 --seed N` produces exactly what `new --seed N` produces,
// and every seed a referee can type is reachable as a member."
//
// Asserted rather than assumed: it is the one property that makes a batch
// member no different from a character generated on its own.
func TestOneMemberIsTheSameCharacterAsNew(t *testing.T) {
	t.Parallel()

	const seed = "42"

	alone, _ := mustRun(t, []string{cmdNew, flagAuto, flagSeed, seed, flagService, other}, nil)

	batched, _ := mustRun(t, []string{cmdBatch, flagCount, "1", flagAuto, flagSeed, seed, flagService, other}, nil)

	if compact(t, alone) != compact(t, strings.TrimSpace(batched)) {
		t.Error("batch --count 1 --seed 42 is not the character new --seed 42 produces")
	}
}

// Each member carries its own derived seed, not the base, so that its record
// regenerates it.
func TestEachMemberCarriesItsOwnSeed(t *testing.T) {
	t.Parallel()

	const base = 1000

	members := generateBatch(t, strconv.Itoa(base), 5)

	for i, member := range members {
		want := uint64(base + i)
		if member.Inputs.Seed != want {
			t.Errorf("member %d carries seed %d, want %d", i, member.Inputs.Seed, want)
		}
	}

	// Regenerating a member from its own recorded seed reproduces it.
	again, _ := mustRun(t, []string{
		cmdNew, flagAuto, flagSeed, strconv.FormatUint(members[3].Inputs.Seed, 10), flagService, other,
	}, nil)

	var reproduced batchMember

	err := json.Unmarshal([]byte(again), &reproduced)
	if err != nil {
		t.Fatalf("reading the regenerated member: %v", err)
	}

	if reproduced.UPP != members[3].UPP {
		t.Errorf("member 3 regenerated as %s, want %s", reproduced.UPP, members[3].UPP)
	}
}

// "an explicit --seed above [2^53], and a derived base + i that passes it
// because the base landed within --count of the bound, are both written to
// the record as given, not silently re-bounded."
func TestAnExplicitSeedIsNeverReBounded(t *testing.T) {
	t.Parallel()

	const bound = uint64(1)<<53 - 1

	base := bound - 1
	members := generateBatch(t, strconv.FormatUint(base, 10), 3)

	for i, member := range members {
		want := base + uint64(i)
		if member.Inputs.Seed != want {
			t.Errorf("member %d carries seed %d, want %d: a derived seed past the bound is written as given",
				i, member.Inputs.Seed, want)
		}
	}

	// And the addition wraps rather than failing: a base at the very top of
	// the range is a seed like any other.
	wrapped := generateBatch(t, strconv.FormatUint(math.MaxUint64, 10), 2)
	if wrapped[1].Inputs.Seed != 0 {
		t.Errorf("the member after the largest seed is %d, want 0: the addition wraps",
			wrapped[1].Inputs.Seed)
	}
}

// batch writes what it rolled: the dead are written like anyone else.
func TestBatchRefusesWithoutAuto(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no --auto":  {cmdBatch, flagCount, "3"},
		"no --count": {cmdBatch, flagAuto},
		"zero count": {cmdBatch, flagAuto, flagCount, "0"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: batch accepted %v", name, args)
		}
	}
}

// -o naming a directory writes one file per character, named by its index.
func TestBatchIntoADirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	mustRun(t, []string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil)

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}

	// A second run refuses to replace them, and --force replaces them.
	err = run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("a second batch overwrote the first without --force")
	}

	err = run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir, flagForce,
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("--force did not replace the members: %v", err)
	}
}

// batchMember is as much of a record as these tests read.
type batchMember struct {
	UPP    string `json:"upp"`
	Inputs struct {
		Seed uint64 `json:"seed"`
	} `json:"inputs"`
}

func generateBatch(t *testing.T, seed string, count int) []batchMember {
	t.Helper()

	out, _ := mustRun(t, []string{
		cmdBatch, flagCount, strconv.Itoa(count), flagAuto, flagSeed, seed, flagService, other,
	}, nil)

	var members []batchMember

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		var m batchMember

		err := json.Unmarshal([]byte(line), &m)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		members = append(members, m)
	}

	if len(members) != count {
		t.Fatalf("read %d members, want %d", len(members), count)
	}

	return members
}

// compact removes the formatting difference between an indented record and a
// JSONL line, which is the only way the two are allowed to differ.
func compact(t *testing.T, text string) string {
	t.Helper()

	var record any

	err := json.Unmarshal([]byte(text), &record)
	if err != nil {
		t.Fatalf("reading a record: %v", err)
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("re-encoding a record: %v", err)
	}

	return string(encoded)
}

// `-o dir/` names a directory that need not already exist. Without the
// trailing separator being read as "a directory", the README's own line
// fails, and `-o characters` quietly writes one JSONL file where twenty
// records were asked for.
func TestBatchCreatesTheOutputDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "characters") + string(filepath.Separator)

	mustRun(t, []string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil)

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}
}

// A collision anywhere in the batch is found before any of it is written, so
// a refused batch leaves the directory as it was.
func TestARefusedBatchWritesNothing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// The second member's file, already there.
	blocked := filepath.Join(dir, "2.json")

	err := os.WriteFile(blocked, []byte("{}\n"), 0o600)
	if err != nil {
		t.Fatalf("planting the collision: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "4", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch overwrote an existing member without --force")
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
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
		cmdBatch, flagCount, "2", flagAuto, flagSeed, "7", flagService, other,
		flagOutput, filepath.Join(blocking, "members") + string(filepath.Separator),
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch wrote into a directory it could not make")
	}

	if !strings.Contains(err.Error(), "creating") {
		t.Errorf("the failure reads %q, which does not name the directory", err)
	}
}

// refusingWriter fails every write, which is how the write path's failure is
// reached without filling a disk.
type refusingWriter struct{}

var errRefused = errors.New("refused")

func (refusingWriter) Write([]byte) (int, error) { return 0, errRefused }

// A write that fails is reported as the write, and abandons what it opened
// rather than leaving the descriptor held or finishing a partial record.
func TestAFailedWriteIsReportedAndAbandoned(t *testing.T) {
	t.Parallel()

	finished, abandoned := false, false

	where := destination{
		out: refusingWriter{},
		close: func() error {
			finished = true

			return nil
		},
		abandon: func() { abandoned = true },
	}

	err := where.write("anything")
	if !errors.Is(err, errRefused) {
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
// Here the third member's file is already a directory, and --force skips the
// up-front check that would have refused the run before writing anything, so
// two records are on disk when it fails. It printed the error and no tally,
// and the tally is the only place the run says how many of what it wrote are
// corpses (#118).
func TestABatchThatStopsSaysWhatItWrote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := os.Mkdir(filepath.Join(dir, "3.json"), 0o700)
	if err != nil {
		t.Fatalf("planting the directory: %v", err)
	}

	var asking strings.Builder

	err = run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagSeed, "100", flagService, other,
		flagForce, flagOutput, dir + string(filepath.Separator),
	}, nil, io.Discard, &asking)
	if err == nil {
		t.Fatal("a batch wrote a record over a directory")
	}

	if !strings.HasPrefix(asking.String(), "2 written before the run stopped, ") {
		t.Errorf("a batch that stopped after two records said %q", asking.String())
	}
}

// When the tally of a stopped batch cannot be written either, both failures
// are reported: the run's own, which is what stopped it, and the tally's.
func TestAStoppedBatchThatCannotReportSaysBoth(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := os.Mkdir(filepath.Join(dir, "3.json"), 0o700)
	if err != nil {
		t.Fatalf("planting the directory: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagSeed, "100", flagService, other,
		flagForce, flagOutput, dir + string(filepath.Separator),
	}, nil, io.Discard, refusingWriter{})
	if err == nil {
		t.Fatal("a batch wrote a record over a directory")
	}

	if !errors.Is(err, errRefused) || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("a stopped batch that could not report said %v, want both failures", err)
	}
}

// oldRecord is what a --force test plants before replacing it.
const oldRecord = "last week's\n"

// --force leaves the record it replaces whole until the new one is (#191).
// Opening with a truncating flag emptied the old record before the first byte
// of the new one, so a write that failed left neither.
func TestForceKeepsTheOldRecordUntilTheNewOneIsWhole(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.txt")

	err := os.WriteFile(path, []byte(oldRecord), recordFileMode)
	if err != nil {
		t.Fatalf("planting the old record: %v", err)
	}

	where, err := openDestination(nil, path, true)
	if err != nil {
		t.Fatalf("opening with --force: %v", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("opening with --force left the old record as %q", got)
	}

	where.out = refusingWriter{}

	err = where.write("this week's\n")
	if !errors.Is(err, errRefused) {
		t.Fatalf("a failed write was reported as %v, want the write's own error", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("a failed --force write left the old record as %q", got)
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
		t.Errorf("--force left %d files beside the record, want none", len(entries)-1)
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

// Where the record cannot be staged, --force says so and leaves the old one.
func TestForceThatCannotStageLeavesTheOldRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.txt")

	err := os.WriteFile(path, []byte(oldRecord), recordFileMode)
	if err != nil {
		t.Fatalf("planting the old record: %v", err)
	}

	err = os.Chmod(dir, 0o400)
	if err != nil {
		t.Fatalf("closing the directory: %v", err)
	}

	_, err = openDestination(nil, path, true)
	if err == nil || !strings.Contains(err.Error(), "staging") {
		t.Errorf("--force into a closed directory said %v, want the staging failure", err)
	}

	err = os.Chmod(dir, recordDirMode)
	if err != nil {
		t.Fatalf("reopening the directory: %v", err)
	}

	if got := mustRead(t, path); got != oldRecord {
		t.Errorf("a --force that could not stage left the old record as %q", got)
	}
}

// Members are named by their one-based index, zero-padded to the width of
// the batch, so a directory sorts in generation order and says nothing about
// seeds.
func TestBatchMembersAreNamedByIndex(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		count int
		wants []string
	}{
		"a batch of three": {count: 3, wants: []string{"1.json", "2.json", "3.json"}},
		"a batch of twelve": {count: 12, wants: []string{
			"01.json", "02.json", "03.json", "04.json", "05.json", "06.json",
			"07.json", "08.json", "09.json", "10.json", "11.json", "12.json",
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			mustRun(t, []string{
				cmdBatch, flagCount, strconv.Itoa(tc.count), flagAuto, flagSeed, "7",
				flagService, other, flagOutput, dir,
			}, nil)

			found, err := filepath.Glob(filepath.Join(dir, "*.json"))
			if err != nil {
				t.Fatalf("looking for the members: %v", err)
			}

			got := make([]string, 0, len(found))
			for _, path := range found {
				got = append(got, filepath.Base(path))
			}

			if !slices.Equal(got, tc.wants) {
				t.Errorf("the members are named %v, want %v", got, tc.wants)
			}
		})
	}
}
