package main

import (
	"io"
	"os"
	"strconv"
	"syscall"
	"testing"
)

// The variables a child started by limited reads: the first tells this test
// binary to be the command, running main on its arguments rather than the
// tests, and the second limits the size of any file it writes.
const (
	asMain    = "CTCHARGEN_TEST_AS_MAIN"
	fileLimit = "CTCHARGEN_TEST_FILE_LIMIT"
)

// TestMain lets a test run the command as a process of its own, which is the
// one way to fail a write partway through a note: a file size limit belongs
// to the whole process, and a write past it fails as a full disk does.
func TestMain(m *testing.M) {
	if os.Getenv(asMain) == "1" {
		limitFiles(os.Getenv(fileLimit))
		main()
	}

	os.Exit(m.Run())
}

// limitFiles sets the file size limit asked for, if one was. The process is
// not signalled past it: Go ignores SIGXFSZ, so the write fails instead.
func limitFiles(bytes string) {
	if bytes == "" {
		return
	}

	n, err := strconv.ParseUint(bytes, 10, 64)
	if err == nil {
		err = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: n, Max: n})
	}

	if err != nil {
		panic(err)
	}
}

// limited runs the command line as its own process, with every file it
// writes limited to limit bytes where limit is not empty, and returns its
// exit status and what it said on standard error.
func limited(t *testing.T, limit string, args ...string) (int, string) {
	t.Helper()

	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = null.Close() }()

	said, saying, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = said.Close() }()

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	process, err := os.StartProcess(self, append([]string{self}, args...), &os.ProcAttr{
		Env:   append(os.Environ(), asMain+"=1", fileLimit+"="+limit),
		Files: []*os.File{null, null, saying},
	})

	_ = saying.Close()

	if err != nil {
		t.Fatal(err)
	}

	stderr, err := io.ReadAll(said)
	if err != nil {
		t.Fatal(err)
	}

	state, err := process.Wait()
	if err != nil {
		t.Fatal(err)
	}

	return state.ExitCode(), string(stderr)
}
