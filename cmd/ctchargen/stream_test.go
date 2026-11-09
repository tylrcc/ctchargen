package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
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

// A batch to standard output arrives a note at a time.
//
// It used to accumulate the whole run in a strings.Builder and write once at
// the end, including to stdout, so `batch --count 50000 | less` showed
// nothing until the run finished and held the run in memory while it did
// (#44). What distinguishes the two is not the bytes - they are the same
// bytes - but when they leave, and the count of writes is what says so.
//
// Holds: A batch to standard output arrives a note at a time.
func TestABatchToStandardOutputArrivesANoteAtATime(t *testing.T) {
	t.Parallel()

	const members = 5

	var out countingWriter

	err := run([]string{
		cmdBatch, flagCount, "5", flagService, other,
	}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if out.writes != members {
		t.Errorf("%d writes for %d members; a buffered run writes once",
			out.writes, members)
	}

	if notes := strings.Count(out.got.String(), "Keep this note whole"); notes != members {
		t.Errorf("%d notes for %d members", notes, members)
	}

	// One note after another, a blank line between each and the next, so each
	// opens its own heading rather than running on from the last.
	if apart := strings.Count(out.got.String(), "character.\n\n---\nname:"); apart != members-1 {
		t.Errorf("%d notes set apart from the one before, want %d", apart, members-1)
	}
}

// failsAfter lets left writes through and fails every one after, with
// errWriteFailed. Zero fails at once, as a closed pipe or a refusing disk
// does; more is a disk that fills, or a usage line that got out before the
// flag list met a closed pipe. It is the one failing writer the command's
// tests use (#373).
type failsAfter struct {
	left   int
	writes int
}

var errWriteFailed = errors.New("the write failed")

func (c *failsAfter) Write(text []byte) (int, error) {
	if c.left == 0 {
		return 0, errWriteFailed
	}

	c.left--

	c.writes++

	return len(text), nil
}

// A write that fails partway stops the batch, and says which member.
//
// This branch exists only because the run streams: a buffered run writes once
// and either succeeds or does not. Streaming means a write can fail with
// notes already gone, and the run has to report where it stopped rather
// than carrying on.
//
// A closed pipe is not this case, whatever it looks like. `batch |
// head -1` dies by SIGPIPE before Write ever returns an error, which is what
// a tool in a pipeline should do; what reaches here is a failure the runtime
// hands back, such as a redirected file that has run out of room.
//
// Holds: A write that fails partway stops the batch, naming the member.
func TestABatchReportsAWriteThatFailedPartway(t *testing.T) {
	t.Parallel()

	out := failsAfter{left: 2}

	var asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "5", flagService, other,
	}, nil, &out, &asking)

	// A streamed batch has no folder, so its stopped-run tally names none.
	if strings.Contains(asking.String(), "holds what was written") {
		t.Errorf("a streamed batch that stopped spoke of a folder: %q", asking.String())
	}

	// A streamed member has no file name, so the failure names its position
	// among those written: the write fails on the third.
	switch {
	case err == nil:
		t.Fatal("a batch hit a failing write and reported success")
	case !errors.Is(err, errWriteFailed):
		t.Errorf("error %q does not carry the write failure", err)
	case !strings.Contains(err.Error(), "member 3"):
		t.Errorf("error %q does not name the member it stopped on", err)
	}

	if out.writes != 2 {
		t.Errorf("%d members were written before the failure, want 2", out.writes)
	}
}

// A batch of more members than its search bound can count still generates
// (#423). The bound is count times attemptsPerMember, and a count near the
// top of an int made that product overflow, so the batch gave up having
// generated nobody. Here it writes its first member and stops only when the
// output fails on the second.
//
// Holds: A batch gives up rather than drawing forever.
func TestAHugeBatchGeneratesRatherThanGivingUpAtOnce(t *testing.T) {
	t.Parallel()

	out := failsAfter{left: 1}

	err := run([]string{cmdBatch, flagCount, "4611686018427387904", flagService, other},
		nil, &out, io.Discard)
	if !errors.Is(err, errWriteFailed) || out.writes != 1 {
		t.Errorf("a huge batch wrote %d notes and said %v, want one note and then the write failure",
			out.writes, err)
	}
}
