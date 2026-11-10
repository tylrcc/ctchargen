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

// eachNote is a line every note carries once, its generation record's
// heading, so counting it counts the notes a run wrote.
const eachNote = "## Generation record"

// scouts is the service the report used, and the one that kills most.
const (
	scouts = "scouts"
)

// A batch writes only the living (#421), and says what it did on the channel
// that is not the data.
//
// A dead character is written nowhere, and another is generated in his
// place, so a batch of twenty Scouts - a service that kills most of them -
// is twenty living Scouts. The closing line says how many were passed over,
// on the operator's channel and never on standard output, where it would be
// a line in the last note (#33).
//
// What it should say is read off the notes themselves rather than typed in:
// counting the batch's notes holds this file's reading of which departures
// are fatal against the one the note says.
//
// Holds: A batch writes only the living, and generates another.
func TestABatchWritesOnlyTheLiving(t *testing.T) {
	t.Parallel()

	const wanted = 20

	out, asking := mustRun(t, []string{
		cmdBatch, flagCount, strconv.Itoa(wanted), flagService, scouts,
	}, nil)

	written := livingCount(t, out)
	if written.members != wanted || written.dead != 0 {
		t.Fatalf("%d written, %d of them dead; want %d, all living", written.members, written.dead, wanted)
	}

	said := strings.TrimSpace(asking)
	if !strings.HasPrefix(said, fmt.Sprintf("%d written, ", wanted)) ||
		!strings.HasSuffix(said, " passed over for dying") {
		t.Errorf("the closing line is %q, want %d written and the number passed over", said, wanted)
	}

	if strings.Contains(out, "written") {
		t.Error("the summary reached standard output, among the notes")
	}
}

// failedSearch is a batch's search built to fail: twenty living asked
// for, one attempt each, from a walk that kills every scout it plays.
func failedSearch(t *testing.T) batchRun {
	t.Helper()

	return batchRun{
		base:   chargen.Inputs{Forced: chargen.ForceService(traveller.Scouts)},
		policy: chargen.DefaultPolicy(), count: 20, attempts: 1, dice: scoutKilled.roller(t),
	}
}

// A batch gives up rather than drawing forever.
//
// The bound is a hundred attempts per character asked for, which no measured run
// approaches. It is reachable here because a batchRun carries the bound rather
// than reading the constant, so a test can ask for the living twenty times in
// twenty draws from a service that kills three in four.
//
// Holds: A batch gives up rather than drawing forever.
func TestABatchGivesUpRatherThanDrawingForever(t *testing.T) {
	t.Parallel()

	_, err := failedSearch(t).each(func(int, *chargen.Character) error { return nil })
	if err == nil {
		t.Fatal("a bounded search reported success without finding the living")
	}

	for _, want := range []string{"batch generated", "20 living"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
}

// livesThenDies is a source whose first character survives and whose every
// later one is killed, so a batch writes one and then runs out.
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

// A batch's search that runs out, through the command (#250).
//
// The bound is count times attemptsPerMember, and the error
// says exactly how many it generated: one more or one fewer is a moved bound.
// One member was written before the search failed, so the stopped-run tally
// had something to say, and must not say it: the error already carries the
// counts, and a second line beside it would tell the same story twice.
//
// Holds: A batch gives up rather than drawing forever.
//
// Holds: A failed search prints no stopped-run tally beside its counts.
func TestAFailedSearchCountsOnce(t *testing.T) {
	t.Parallel()

	var out, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagService, other,
	}, nil, &out, &asking, withDice(livesThenDies(t)))
	if err == nil {
		t.Fatal("a search that never finds the second survivor reported success")
	}

	if !errors.Is(err, errSearch) {
		t.Fatalf("an exhausted search said %v, want the search's own failure", err)
	}

	want := fmt.Sprintf("generated %d characters for 2 living and found 1", 2*attemptsPerMember)

	if !strings.Contains(err.Error(), want) {
		t.Errorf("the search reported %q, want it to say it %s", err, want)
	}

	if notes := strings.Count(out.String(), eachNote); notes != 1 {
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

	if found.members != strings.Count(notes, eachNote) {
		t.Fatalf("%d headlines for %d notes", found.members, strings.Count(notes, eachNote))
	}

	return found
}

// A summary nobody received is not a summary that was given.
//
// The whole of #33 is that a run which said nothing was a trap. Reporting
// success when the closing line could not be written would be the same trap
// with an extra step, so the write is checked like any other.
//
// Holds: A batch says how many it wrote and how many it passed over.
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
// A batch's passing over rests entirely on this classification, and until now
// nothing asserted it directly: it was checked by counting fatal notes in a
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
		"discharged":          {traveller.Discharged{}, false},
		"denied reenlistment": {traveller.DeniedReenlistment{}, false},
		"retired":             {traveller.Retired{}, false},
		"survival":            {traveller.KilledBySurvivalThrow{}, true},
		"medical":             {traveller.KilledByMedicalCrisis{Characteristic: traveller.Endurance}, true},
		"no service":          {nil, false},
	} {
		if got := traveller.Fatal(tc.departure); got != tc.fatal {
			t.Errorf("%s: fatal is %v, want %v", name, got, tc.fatal)
		}
	}
}

// A note says who made each choice, on the choice's own line: the policy on
// an auto run, the player on a guided one, and never the other. A note that
// credited the policy with a choice the player made would misdescribe how the
// character was made.
//
// Holds: A note says who made each choice.
func TestANoteSaysWhoMadeEachChoice(t *testing.T) {
	t.Parallel()

	// Each walks a scripted path that lives to be asked: random dice can kill
	// a character at the first survival throw, before any choice is put.
	for name, tc := range map[string]struct {
		args          []string
		in            io.Reader
		walk          walk
		made, notMade string
	}{
		"the policy answered": {
			[]string{cmdNew, flagAuto, flagService, other}, nil, otherByPolicy, ": policy chose ", ": player chose ",
		},
		"the player answered": {
			[]string{cmdNew, flagService, other}, answers("1", 300), otherWalked, ": player chose ", ": policy chose ",
		},
	} {
		var out strings.Builder

		err := run(tc.args, tc.in, &out, io.Discard, withDice(tc.walk.roller(t)))
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(out.String(), tc.made) || strings.Contains(out.String(), tc.notMade) {
			t.Errorf("%s: the record does not credit every choice to the one who made it:\n%s", name, out.String())
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
		cmdBatch, flagCount, "2", flagService, other, flagOutput, dir,
	}, nil, io.Discard, &asking, withDice(livesThenDies(t)))
	if !errors.Is(err, errSearch) {
		t.Fatalf("an exhausted search said %v, want the search's own failure", err)
	}

	want := dir + " holds what was written; remove it or choose another folder to run again\n"
	if asking.String() != want {
		t.Errorf("a failed search into a folder said %q, want %q", asking.String(), want)
	}
}

// diesThenLives is a source that alternates: a character the walk kills, then
// one it doesn't, then again. A batch of two from it passes over exactly two.
func diesThenLives(t *testing.T) source {
	t.Helper()

	lives, dies := otherByPolicy.roller(t), otherKilled.roller(t)
	next := 0

	return func() chargen.Roller {
		next++
		if next%2 == 1 {
			return dies()
		}

		return lives()
	}
}

// The closing line counts exactly the dead it passed over: two members from a
// source that kills every other character is two written and two passed over.
//
// Holds: A batch says how many it wrote and how many it passed over.
func TestABatchCountsWhatItPassedOver(t *testing.T) {
	t.Parallel()

	var asking strings.Builder

	err := run([]string{cmdBatch, flagCount, "2", flagService, other},
		nil, io.Discard, &asking, withDice(diesThenLives(t)))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := asking.String(), "2 written, 2 passed over for dying\n"; got != want {
		t.Errorf("the closing line is %q, want %q", got, want)
	}
}
