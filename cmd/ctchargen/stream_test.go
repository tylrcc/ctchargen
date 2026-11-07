package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
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

// A batch to standard output arrives a record at a time.
//
// It used to accumulate the whole run in a strings.Builder and write once at
// the end, including to stdout, so `batch --count 50000 --auto | jq` produced
// nothing until the run finished and held the run in memory while it did
// (#44). What distinguishes the two is not the bytes - they are the same
// bytes - but when they leave, and the count of writes is what says so.
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
