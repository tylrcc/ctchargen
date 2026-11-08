package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// countingWriter records how many times it was written to, and what it got.
type countingWriter struct {
	writes int
	got    bytes.Buffer
}

func (c *countingWriter) Write(text []byte) (int, error) {
	c.writes++

	// bytes.Buffer.Write never returns an error, so there is nothing here to
	// wrap and nothing a caller could do about it.
	n, _ := c.got.Write(text)

	return n, nil
}

// A batch to standard output arrives a record at a time.
//
// It used to accumulate the whole run in a strings.Builder and write once at
// the end, including to stdout, so `batch --count 50000 --auto | jq` produced
// nothing until the run finished and held the run in memory while it did
// (#44). What distinguishes the two is not the bytes - they are the same
// bytes - but when they leave, and the count of writes is what says so.
//
// Holds: A batch to standard output arrives a record at a time.
func TestABatchToStandardOutputArrivesARecordAtATime(t *testing.T) {
	t.Parallel()

	const members = 5

	var out countingWriter

	err := run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagService, other,
	}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if out.writes != members {
		t.Errorf("%d writes for %d members; a buffered run writes once",
			out.writes, members)
	}

	if lines := strings.Count(out.got.String(), "\n"); lines != members {
		t.Errorf("%d lines for %d members", lines, members)
	}
}

// A batch to a file still writes all of it or none of it.
//
// The atomicity argument is about a file that can be left half-written and
// clobbered on the retry; standard output has no such file, which is the
// whole of why the two paths differ now (#64). This holds the half that did
// not change: nothing is opened until the run has finished generating.
//
// Holds: A batch to a file is written all at once, never a record at a time.
func TestABatchToAFileIsStillWrittenAtOnce(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "roster.jsonl")

	var out countingWriter

	err := run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagService, other,
		flagOutput, path,
	}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if out.writes != 0 {
		t.Errorf("a batch bound for a file wrote %d times to the stream", out.writes)
	}

	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if lines := strings.Count(string(text), "\n"); lines != 5 {
		t.Errorf("%d lines in the file, want 5", lines)
	}
}

// failsAfter writes a few records and then fails, as a filling disk does.
// errClosedPipe is reused only as a sentinel to match on; nothing here is a
// pipe.
type failsAfter struct {
	left   int
	writes int
}

func (c *failsAfter) Write(text []byte) (int, error) {
	if c.left == 0 {
		return 0, errClosedPipe
	}

	c.left--

	c.writes++

	return len(text), nil
}

// A write that fails partway stops the batch, and says which member.
//
// This branch exists only because the run streams: a buffered run writes once
// and either succeeds or does not. Streaming means a write can fail with
// records already gone, and the run has to report where it stopped rather
// than carrying on.
//
// A closed pipe is not this case, whatever it looks like. `batch --auto |
// head -1` dies by SIGPIPE before Write ever returns an error, which is what
// a tool in a pipeline should do; what reaches here is a failure the runtime
// hands back, such as a redirected file that has run out of room.
//
// Holds: A write that fails partway stops the batch, naming the member.
func TestABatchReportsAWriteThatFailedPartway(t *testing.T) {
	t.Parallel()

	out := failsAfter{left: 2}

	err := run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagService, other,
	}, nil, &out, io.Discard)

	// A streamed member has no file name, so the failure names its position
	// among those written: the write fails on the third.
	switch {
	case err == nil:
		t.Fatal("a batch hit a failing write and reported success")
	case !errors.Is(err, errClosedPipe):
		t.Errorf("error %q does not carry the write failure", err)
	case !strings.Contains(err.Error(), "member 3"):
		t.Errorf("error %q does not name the member it stopped on", err)
	}

	if out.writes != 2 {
		t.Errorf("%d members were written before the failure, want 2", out.writes)
	}
}

// countingDice plays a walk, counting how many characters asked for it, so a
// test can say whether a run generated anybody.
type countingDice struct{ asked int }

func (c *countingDice) source(played source) source {
	return func() chargen.Roller {
		c.asked++

		return played()
	}
}

// A batch to a .jsonl file that is already there is refused before anyone is
// generated, and the refusal claims no record written (#246).
//
// The open at the end refused it too, but after the whole batch had been
// made, and the stopped-run tally then counted records that were only ever
// in memory: "5 written before the run stopped", with the old file untouched.
//
// Holds: A buffered batch that stops claims no record written.
func TestAnExistingFileIsRefusedFirst(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "roster.jsonl")

	err := os.WriteFile(path, []byte(oldRecord), 0o600)
	if err != nil {
		t.Fatalf("planting the old roster: %v", err)
	}

	var (
		asking strings.Builder
		rolled countingDice
	)

	err = run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagService, other, flagOutput, path,
	}, nil, io.Discard, &asking, withDice(rolled.source(otherByPolicy.roller(t))))
	if !errors.Is(err, errWouldOverwrite) {
		t.Fatalf("a batch onto an existing file said %v, want the --force refusal", err)
	}

	if rolled.asked != 0 {
		t.Errorf("a refused batch generated %d characters before refusing", rolled.asked)
	}

	if asking.Len() != 0 {
		t.Errorf("a refused batch reported %q, which claims records it never wrote",
			asking.String())
	}

	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if string(text) != oldRecord {
		t.Errorf("the refused batch left %q, want the old roster", text)
	}
}

// A batch to a .jsonl file that fails after generating reports no records
// written, because none reached the file (#246).
//
// Its directory is not there, so --force cannot stage the replacement: every
// member is made, and the write is what fails. Before, the tally counted the
// five held in memory as written.
//
// Holds: A buffered batch that stops claims no record written.
func TestAFailedFileWriteClaimsNothing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing", "roster.jsonl")

	var (
		asking strings.Builder
		rolled countingDice
	)

	err := run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagService, other,
		flagForce, flagOutput, path,
	}, nil, io.Discard, &asking, withDice(rolled.source(otherByPolicy.roller(t))))
	if err == nil {
		t.Fatal("a batch wrote into a directory that is not there")
	}

	if rolled.asked != 5 {
		t.Errorf("%d characters generated, want all 5 made before the write failed",
			rolled.asked)
	}

	if asking.Len() != 0 {
		t.Errorf("a buffered batch that wrote nothing reported %q", asking.String())
	}
}
