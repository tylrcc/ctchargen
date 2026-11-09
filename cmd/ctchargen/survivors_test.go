package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
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
// order is free to invalidate at any time; counting the batch's notes instead also
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
// batch, which passes just as well if a case is misread in a way the dice
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
// a run never silently replaces a note, not that a directory is written
// atomically. Here the folder is closed as the third member is generated, so
// two notes are on disk when the run stops, and they stay.
//
// Holds: A directory batch that cannot write a member leaves what came before.
func TestADirectoryBatchLeavesWhatItWroteBeforeAnObstruction(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "crew")

	err := run([]string{cmdBatch, flagCount, "6", flagService, other, flagOutput, dir},
		nil, io.Discard, io.Discard, withDice(closingOn(t, dir, 3, otherByPolicy)))
	if err == nil {
		t.Fatal("a batch into a closed folder reported success")
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("looking for what was written: %v", err)
	}

	if len(written) != 2 {
		t.Errorf("%d members are in the folder after the run stopped, want 2", len(written))
	}
}

// A failed search into a folder prints no tally either, but says the folder
// holds the member it wrote: the folder is not empty now, so a rerun into it
// would be refused (#426), and the error alone would not say why.
func TestAFailedSearchIntoAFolderSaysWhatItHolds(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "crew")

	var asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagService, other, flagSurvivors, flagOutput, dir,
	}, nil, io.Discard, &asking, withDice(livesThenDies(t)))
	if !errors.Is(err, errSearch) {
		t.Fatalf("an exhausted search said %v, want the search's own failure", err)
	}

	want := dir + " holds what was written; remove it or choose another folder to run again\n"
	if asking.String() != want {
		t.Errorf("a failed search into a folder said %q, want %q", asking.String(), want)
	}
}
