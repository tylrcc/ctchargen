package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// scouts is the service the report used, and the one that kills most.
const (
	scouts        = "scouts"
	flagSurvivors = "--survivors"
)

// A batch says what it did, on the channel that is not the data.
//
// The run the report opened with wrote its corpses and printed nothing at all
// (#33). The count goes to the operator's channel and never to standard
// output, where it would be a line in the last note.
//
// What it should say is read off the notes themselves rather than typed in.
// A literal here would be a golden maintained by hand, which the dice-stream
// order is free to invalidate at any time; counting the roster instead also
// holds this file's reading of which departures are fatal to the one the
// note says, so the two transcriptions of that rule are checked against
// each other rather than only against themselves.
//
// Holds: A batch says how many it wrote and how many died.
func TestABatchSaysHowManyDied(t *testing.T) {
	t.Parallel()

	const wanted = 100

	out, asking := mustRun(t, []string{
		cmdBatch, flagCount, strconv.Itoa(wanted), flagService, scouts,
	}, nil)

	written := livingCount(t, out)
	if written.members != wanted || written.dead == 0 {
		t.Fatalf("%d written, %d of them dead; the run this reads is not the one it means",
			written.members, written.dead)
	}

	want := fmt.Sprintf("%d written, %d died", written.members, written.dead)
	if got := strings.TrimSpace(asking); got != want {
		t.Errorf("the closing line is %q, want %q", got, want)
	}

	if strings.Contains(out, "written") {
		t.Error("the summary reached standard output, among the notes")
	}
}

// --survivors writes the number asked for, and all of them are living.
//
// Holds: `--survivors` writes only the living, and generates another.
func TestSurvivorsWritesOnlyTheLiving(t *testing.T) {
	t.Parallel()

	const wanted = 20

	out, asking := mustRun(t, []string{
		cmdBatch, flagCount, "20", flagService, scouts,
		flagSurvivors,
	}, nil)

	written := livingCount(t, out)
	if written.members != wanted {
		t.Errorf("%d written, want %d", written.members, wanted)
	}

	if written.dead > 0 {
		t.Errorf("%d of the survivors are dead", written.dead)
	}

	if !strings.Contains(asking, "passed over for dying") {
		t.Errorf("the closing line does not say what was passed over: %q", asking)
	}
}

// failedSearch is a --survivors search built to fail: twenty living asked
// for, one attempt each, from a walk that kills every scout it plays.
func failedSearch(t *testing.T) batchRun {
	t.Helper()

	return batchRun{
		base:   chargen.Inputs{Forced: chargen.ForceService(traveller.Scouts)},
		policy: chargen.DefaultPolicy(), count: 20, survivors: true, attempts: 1, dice: scoutKilled.roller(t),
	}
}

// --survivors gives up rather than drawing forever.
//
// The bound is a hundred attempts per character asked for, which no measured run
// approaches. It is reachable here because a batchRun carries the bound rather
// than reading the constant, so a test can ask for the living twenty times in
// twenty draws from a service that kills three in four.
//
// Holds: `--survivors` gives up rather than drawing forever.
func TestSurvivorsGivesUpRatherThanDrawingForever(t *testing.T) {
	t.Parallel()

	_, err := failedSearch(t).each(func(int, *chargen.Character) error { return nil })
	if err == nil {
		t.Fatal("a bounded search reported success without finding the living")
	}

	for _, want := range []string{"--survivors", "20 living"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
}

// livesThenDies is a source whose first character survives and whose every
// later one is killed, so a --survivors search writes one and then runs out.
func livesThenDies(t *testing.T) source {
	t.Helper()

	lives, dies := otherByPolicy.roller(t), otherKilled.roller(t)
	first := true

	return func() chargen.Roller {
		if first {
			first = false

			return lives()
		}

		return dies()
	}
}

// A --survivors search that runs out, through the command (#250).
//
// The bound is count times survivorAttempts, and the error
// says exactly how many it generated: one more or one fewer is a moved bound.
// One member was written before the search failed, so the stopped-run tally
// had something to say, and must not say it: the error already carries the
// counts, and a second line beside it would tell the same story twice.
//
// Holds: `--survivors` gives up rather than drawing forever.
//
// Holds: A failed search prints no stopped-run tally beside its counts.
func TestAFailedSearchCountsOnce(t *testing.T) {
	t.Parallel()

	var out, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagService, other, flagSurvivors,
	}, nil, &out, &asking, withDice(livesThenDies(t)))
	if err == nil {
		t.Fatal("a search that never finds the second survivor reported success")
	}

	if !errors.Is(err, errSearch) {
		t.Fatalf("an exhausted search said %v, want the search's own failure", err)
	}

	want := fmt.Sprintf("generated %d characters for 2 living and found 1", 2*survivorAttempts)

	if !strings.Contains(err.Error(), want) {
		t.Errorf("the search reported %q, want it to say it %s", err, want)
	}

	if notes := strings.Count(out.String(), "Keep this note whole"); notes != 1 {
		t.Errorf("%d notes streamed, want the one survivor", notes)
	}

	if asking.Len() != 0 {
		t.Errorf("a failed search printed %q beside its error, which carries the counts alone",
			asking.String())
	}
}

type census struct{ members, dead int }

// livingCount reads a stream of notes: how many members it holds, and how
// many of them died, which a note's headline says as the departure that
// killed him.
func livingCount(t *testing.T, notes string) census {
	t.Helper()

	var found census

	for line := range strings.Lines(notes) {
		if !strings.HasPrefix(line, "UPP ") {
			continue
		}

		found.members++

		if strings.Contains(line, "killed by") {
			found.dead++
		}
	}

	if found.members != strings.Count(notes, "Keep this note whole") {
		t.Fatalf("%d headlines for %d notes", found.members, strings.Count(notes, "Keep this note whole"))
	}

	return found
}

// A summary nobody received is not a summary that was given.
//
// The whole of #33 is that a run which said nothing was a trap. Reporting
// success when the closing line could not be written would be the same trap
// with an extra step, so the write is checked like any other.
//
// Holds: A batch says how many it wrote and how many died.
func TestABatchReportsASummaryThatCouldNotBeWritten(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagService, other,
	}, nil, &out, &failsAfter{})
	if err == nil {
		t.Fatal("a summary nobody received was reported as written")
	}

	if !errors.Is(err, errWriteFailed) {
		t.Errorf("error %q does not carry the write failure", err)
	}
}

// Which departures count as death, stated once and checked.
//
// --survivors rests entirely on this classification, and until now nothing
// asserted it directly: the flag was checked by counting fatal records in a
// roster, which passes just as well if a case is misread in a way the dice
// happen not to reach. Fold makes a new case a compile error here; this makes
// a wrong answer for an existing one a test failure.
//
// Holds: Which departures are deaths is said in one place.
// Holds: Only the two deaths count as dying.
func TestOnlyTheTwoDeathsCountAsDying(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		departure traveller.Departure
		fatal     bool
	}{
		"discharged": {traveller.Discharged{}, false},
		"forced out": {traveller.ForcedOut{}, false},
		"retired":    {traveller.Retired{}, false},
		"survival":   {traveller.KilledBySurvivalThrow{}, true},
		"medical":    {traveller.KilledByMedicalCrisis{Characteristic: traveller.Endurance}, true},
		"no service": {nil, false},
	} {
		if got := traveller.Fatal(tc.departure); got != tc.fatal {
			t.Errorf("%s: fatal is %v, want %v", name, got, tc.fatal)
		}
	}
}

// A --survivors batch is refused over a member it would write, and not over a
// file beyond the batch.
//
// Members are named first by position, so who survives does not change which
// positions the batch fills: 1 through count, whoever fills them. The check can
// key on those without generating anyone, and a note one past the count is
// nobody's to refuse over or to touch.
//
// Holds: A refused batch writes none of itself, on the members it will actually
// write.
func TestARefusedSurvivorBatchKeysOnThePositionsItFills(t *testing.T) {
	t.Parallel()

	const (
		base   = uint64(1000)
		wanted = 5
	)

	// The names are typed out rather than asked of position, so a roster
	// that started counting from zero would be caught here and not only in
	// the plain batch's test.
	for name, tc := range map[string]struct {
		plant   string
		refuses bool
		left    int
	}{
		"the last member it writes": {"5 Navy Admiral 777777.md", true, 1},
		"a name past the end":       {"6 Navy Admiral 777777.md", false, wanted + 1},
	} {
		dir := t.TempDir()
		planted := filepath.Join(dir, tc.plant)

		err := os.WriteFile(planted, []byte("{}\n"), 0o600)
		if err != nil {
			t.Fatalf("%s: planting the collision: %v", name, err)
		}

		var out strings.Builder

		err = run([]string{
			cmdBatch, flagCount, strconv.Itoa(wanted),
			flagService, scouts,
			flagSurvivors, flagOutput, dir,
		}, nil, &out, io.Discard)

		switch {
		case tc.refuses && err == nil:
			t.Errorf("%s: the batch replaced a member that was already there", name)
		case !tc.refuses && err != nil:
			t.Errorf("%s: the batch was refused over a name it never writes: %v", name, err)
		}

		found, err := filepath.Glob(filepath.Join(dir, "*.md"))
		if err != nil {
			t.Fatalf("%s: looking for the members: %v", name, err)
		}

		if len(found) != tc.left {
			t.Errorf("%s: the run left %d files behind, want %d", name, len(found), tc.left)
		}

		// The planted file is never the run's to replace, either way.
		kept, err := os.ReadFile(planted)
		if err != nil {
			t.Fatalf("%s: reading the planted file: %v", name, err)
		}

		if string(kept) != "{}\n" {
			t.Errorf("%s: the planted file was written over", name)
		}
	}
}

// A note names the policy that answered it, and only then.
//
// The line exists so a referee reading a note can tell which version of the
// policy's answers made it, under which strategies. On a character he walked
// by hand the policy answered nothing, and naming a policy that decided none
// of it is a claim the note cannot support.
//
// Holds: A note names the policy that answered it.
func TestOnlyANoteThePolicyAnsweredNamesOne(t *testing.T) {
	t.Parallel()

	policyLine := fmt.Sprintf("answered by the policy, version %d: career %s,", chargen.PolicyVersion, serve)

	// Each walks a scripted path that lives to be asked: random dice can kill
	// a character in his first survival throw, before any choice is put.
	for name, tc := range map[string]struct {
		args  []string
		in    io.Reader
		walk  walk
		wants bool
	}{
		"the policy answered": {
			[]string{cmdNew, flagAuto, flagService, other}, nil, otherByPolicy, true,
		},
		"the player answered": {
			[]string{cmdNew, flagService, other}, answers("1", 300), otherWalked, false,
		},
	} {
		var out strings.Builder

		err := run(tc.args, tc.in, &out, io.Discard, withDice(tc.walk.roller(t)))
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if named := strings.Contains(out.String(), "answered by the policy"); named != tc.wants {
			t.Errorf("%s: names a policy = %v, want %v", name, named, tc.wants)
		}

		if tc.wants && !strings.Contains(out.String(), policyLine) {
			t.Errorf("%s: does not name version %d under the default career:\n%s",
				name, chargen.PolicyVersion, out.String())
		}
	}
}

// A directory batch that cannot write a member leaves what came before it.
//
// This is the trade #99's fix accepts and does not hide: the guarantee is that
// a run never silently replaces a file, not that a directory is written
// atomically. It was true before #98 too.
//
// It is deliberately NOT a test that the run writes as it goes. I wrote it as
// one and it could not have been: an obstruction sits in the write, which a
// buffered run reaches as well, so both designs leave the same two files.
// What holds the memory property is occupied's signature - no policy, no
// roller, so it cannot generate - and the test below holds that.
//
// Holds: A directory batch that cannot write a member leaves what came before.
func TestADirectoryBatchLeavesWhatItWroteBeforeAnObstruction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// The third member of six, so two are written before the obstruction.
	const blockedMember = 2

	// Every member walks the same dice, so the third is named before it is
	// generated, and a directory can sit where it will be written.
	blocked := filepath.Join(dir, "3 Other 644776.md")

	err := os.Mkdir(blocked, noteDirMode)
	if err != nil {
		t.Fatalf("blocking a member's path: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "6", flagService, other,
		flagOutput, dir, flagForce,
	}, nil, io.Discard, io.Discard, withDice(otherByPolicy.partly(t)))
	if err == nil {
		t.Fatal("a batch wrote a member over a directory")
	}

	found, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("looking for what was written: %v", err)
	}

	// Regular files only: the obstruction is a directory carrying a
	// member's name, so a glob counts it as one of them.
	written := make([]string, 0, len(found))

	for _, path := range found {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("looking at %s: %v", path, err)
		}

		if info.Mode().IsRegular() {
			written = append(written, path)
		}
	}

	if len(written) != blockedMember {
		t.Errorf("%d members are in the directory after the obstruction, want %d",
			len(written), blockedMember)
	}
}

// The collision check cannot generate a character, and reads a position from
// a note's name as a number.
//
// The first is its signature, not its body: occupied takes a directory and a
// count, and no policy and no roller. It is the whole of why checking a batch
// for collisions costs a directory listing and not a run (#99). The second is
// what makes a batch of twelve refuse over a note a batch of three wrote:
// "1 X.md" holds the position "01" names.
//
// Holds: A batch checks for collisions without generating anybody.
func TestTheCollisionCheckNeedsNoCharacters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, name := range []string{
		"1 Old Title.md",                      // titled, and narrower than the batch
		"02.md",                               // bare, as earlier builds wrote them
		"3 Not a note.txt", "04x.md", "+5.md", // none of them a note at a position
		"13 Past the end.md",
	} {
		err := os.WriteFile(filepath.Join(dir, name), nil, 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := os.Mkdir(filepath.Join(dir, "06 A directory.md"), noteDirMode)
	if err != nil {
		t.Fatal(err)
	}

	held, err := occupied(dir, 12)
	if err != nil {
		t.Fatal(err)
	}

	want := map[int][]heldNote{
		1: {{path: filepath.Join(dir, "1 Old Title.md")}},
		2: {{path: filepath.Join(dir, "02.md")}},
		6: {{path: filepath.Join(dir, "06 A directory.md"), directory: true}},
	}

	if !maps.EqualFunc(held, want, slices.Equal) {
		t.Errorf("held %v, want %v", held, want)
	}
}
