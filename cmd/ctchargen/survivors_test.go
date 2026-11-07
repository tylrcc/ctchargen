package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
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
// output, where it would be a line of JSONL that is not JSON.
//
// What it should say is read off the records themselves rather than typed in.
// A literal here would be a golden maintained by hand, which the dice-stream
// order is free to invalidate at any time; counting the roster instead also
// holds this file's reading of which departures are fatal to the one the
// record carries, so the two transcriptions of that rule are checked against
// each other rather than only against themselves.
func TestABatchSaysHowManyDied(t *testing.T) {
	t.Parallel()

	const wanted = 100

	out, asking := mustRun(t, []string{
		cmdBatch, flagCount, strconv.Itoa(wanted), flagAuto, flagService, scouts,
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
		t.Error("the summary reached standard output, where it is not JSON")
	}
}

// --survivors writes the number asked for, and all of them are living.
func TestSurvivorsWritesOnlyTheLiving(t *testing.T) {
	t.Parallel()

	const wanted = 20

	out, asking := mustRun(t, []string{
		cmdBatch, flagCount, "20", flagAuto, flagService, scouts,
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

// --survivors gives up rather than drawing forever.
//
// The bound is a hundred attempts per character asked for, which no measured run
// approaches. It is reachable here because eachMember takes the bound rather
// than reading the constant, so a test can ask for the living twenty times in
// twenty draws from a service that kills three in four.
func TestSurvivorsGivesUpRatherThanDrawingForever(t *testing.T) {
	t.Parallel()

	base := chargen.Inputs{
		Service: traveller.Scouts, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	_, err := eachMember(base, chargen.DefaultPolicy(), 20, true, 1, scoutKilled.roller(t),
		func(int, *chargen.Character) error { return nil })
	if err == nil {
		t.Fatal("a bounded search reported success without finding the living")
	}

	for _, want := range []string{"--survivors", "20 living"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
}

type census struct{ members, dead int }

func livingCount(t *testing.T, jsonl string) census {
	t.Helper()

	var found census

	for line := range strings.SplitSeq(strings.TrimSpace(jsonl), "\n") {
		var member struct {
			Departure *struct {
				Fatal bool `json:"fatal"`
			} `json:"departure"`
		}

		err := json.Unmarshal([]byte(line), &member)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		found.members++

		if member.Departure != nil && member.Departure.Fatal {
			found.dead++
		}
	}

	return found
}

// A summary nobody received is not a summary that was given.
//
// The whole of #33 is that a run which said nothing was a trap. Reporting
// success when the closing line could not be written would be the same trap
// with an extra step, so the write is checked like any other.
func TestABatchReportsASummaryThatCouldNotBeWritten(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagAuto, flagService, other,
	}, nil, &out, brokenPipe{})
	if err == nil {
		t.Fatal("a summary nobody received was reported as written")
	}

	if !errors.Is(err, errClosedPipe) {
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
// Members are named by index, so who survives does not change which
// files the batch writes: 1 through count, whoever fills them. The check can
// key on those without generating anyone, and a file one past the count is
// nobody's to refuse over or to touch.
func TestARefusedSurvivorBatchKeysOnTheIndicesItWrites(t *testing.T) {
	t.Parallel()

	const (
		base   = uint64(1000)
		wanted = 5
	)

	// The names are typed out rather than asked of memberPath, so a roster
	// that started counting from zero would be caught here and not only in
	// the plain batch's test.
	for name, tc := range map[string]struct {
		plant   string
		refuses bool
		left    int
	}{
		"the last member it writes": {"5.json", true, 1},
		"a name past the end":       {"6.json", false, wanted + 1},
	} {
		dir := t.TempDir()
		planted := filepath.Join(dir, tc.plant)

		err := os.WriteFile(planted, []byte("{}\n"), 0o600)
		if err != nil {
			t.Fatalf("%s: planting the collision: %v", name, err)
		}

		var out strings.Builder

		err = run([]string{
			cmdBatch, flagCount, strconv.Itoa(wanted), flagAuto,
			flagService, scouts,
			flagSurvivors, flagOutput, dir,
		}, nil, &out, io.Discard)

		switch {
		case tc.refuses && err == nil:
			t.Errorf("%s: the batch replaced a member that was already there", name)
		case !tc.refuses && err != nil:
			t.Errorf("%s: the batch was refused over a name it never writes: %v", name, err)
		}

		found, err := filepath.Glob(filepath.Join(dir, "*.json"))
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

// A record names the policy that answered it, and only then.
//
// The field exists so a referee reading a record can tell which table
// answered it. On a record he walked by hand the policy answered nothing,
// and naming a decision table that decided none of it is a claim the record
// cannot support - the same reason a civilian carries no service and no
// departure.
func TestOnlyARecordThePolicyAnsweredNamesOne(t *testing.T) {
	t.Parallel()

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

		var record struct {
			Policy int `json:"policy"`
			Events []struct {
				Kind string `json:"kind"`
				By   string `json:"by"`
			} `json:"events"`
		}

		err = json.Unmarshal([]byte(out.String()), &record)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		byPolicy := false

		for _, event := range record.Events {
			if event.Kind == "choice" && event.By == "policy" {
				byPolicy = true
			}
		}

		if byPolicy != tc.wants {
			t.Errorf("%s: choices by the policy = %v, want %v", name, byPolicy, tc.wants)
		}

		if named := record.Policy != 0; named != tc.wants {
			t.Errorf("%s: names a policy = %v, want %v", name, named, tc.wants)
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
// What holds the memory property is plannedPaths' signature - no policy, no
// roller, so it cannot generate - and the test below holds that.
func TestADirectoryBatchLeavesWhatItWroteBeforeAnObstruction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// The third member of six, so two are written before the obstruction.
	const blockedMember = 2

	blocked := filepath.Join(dir, "3.json")

	err := os.Mkdir(blocked, recordDirMode)
	if err != nil {
		t.Fatalf("blocking a member's path: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "6", flagAuto, flagService, other,
		flagOutput, dir, flagForce,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch wrote a member over a directory")
	}

	found, err := filepath.Glob(filepath.Join(dir, "*.json"))
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

// The collision check cannot generate a character.
//
// That is its signature, not its body: plannedPaths takes a count and a
// directory, and no policy and no roller. It is the whole of why checking a
// batch for collisions costs arithmetic and not a run (#99).
func TestTheCollisionCheckNeedsNoCharacters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	paths := plannedPaths(3, dir)

	want := []string{
		filepath.Join(dir, "1.json"), filepath.Join(dir, "2.json"), filepath.Join(dir, "3.json"),
	}

	if !slices.Equal(paths, want) {
		t.Errorf("3 members are planned at %v, want %v", paths, want)
	}
}
