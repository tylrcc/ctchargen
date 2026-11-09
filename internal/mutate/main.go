// Command mutate shows whether a test kills a deliberate mutation, without
// touching the tree it is run from.
//
// CLAUDE.md asks that a new invariant be shown killed by a mutation, and the
// failure it produced be quoted. Doing that by editing the tree and copying the
// file back is how a mutation gets left behind, or lands on a line it was not
// meant for. This copies the module's tracked and unignored files into a
// scratch directory, makes one exact replacement there, builds the copy,
// runs the command given, and says whether the mutation was killed:
//
//	go run ./internal/mutate -file internal/chargen/logging.go \
//	    -old 'return []string{"yes", "no"}' -new 'return shared' \
//	    go test ./internal/chargen -run TestAYesOrNoOfferIsEachEventsOwn
//
// The old text must occur in the file exactly once, so a mutation lands where
// it was meant to or not at all. The mutated copy is built, tests and all, as
// go test builds it, vet checks included, and nothing is run: a mutation that
// does not build would fail the command before a test ran, so it is reported
// NOT BUILT and never KILLED. The command's output is printed, and mutate
// exits 0 only when the command failed on a copy that builds (the mutation
// was killed), and 1 otherwise, so the result reads as a test of the test.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
)

// The scratch copy is this user's own, as git's checkout would be.
const (
	dirMode  = 0o700
	fileMode = 0o600
)

var (
	errUsage     = errors.New("usage")
	errSurvived  = errors.New("the mutation survived")
	errNotBuilt  = errors.New("the mutation does not build, so no test ran")
	errNotUnique = errors.New("the text to replace must occur exactly once")
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	root, err := os.Getwd()
	if err == nil {
		err = run(ctx, os.Args[1:], root, os.Stdout)
	}

	stop()

	if err != nil {
		fmt.Fprintln(os.Stderr, "mutate:", err)
		os.Exit(1)
	}
}

// run parses the flags, mutates a copy of the module rooted at root, and
// runs the command there.
func run(ctx context.Context, args []string, root string, out io.Writer) error {
	flags := flag.NewFlagSet("mutate", flag.ContinueOnError)
	file := flags.String("file", "", "the file to mutate, relative to the module root")
	old := flags.String("old", "", "the exact text to replace; it must occur once")
	replacement := flags.String("new", "", "what to replace it with")

	err := flags.Parse(args)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	command := flags.Args()
	if *file == "" || *old == "" || len(command) == 0 {
		return fmt.Errorf("%w: mutate -file F -old TEXT -new TEXT command [args...]", errUsage)
	}

	scratch, err := os.MkdirTemp("", "ctchargen-mutate-*")
	if err != nil {
		return fmt.Errorf("making a scratch copy: %w", err)
	}

	defer func() { _ = os.RemoveAll(scratch) }()

	err = copyModule(ctx, root, scratch)
	if err != nil {
		return err
	}

	err = replaceOnce(filepath.Join(scratch, *file), *old, *replacement)
	if err != nil {
		return fmt.Errorf("%s: %w", *file, err)
	}

	err = compile(ctx, scratch, out)
	if err != nil {
		return err
	}

	return verdict(ctx, scratch, command, out)
}

// compile builds every package in dir and every test binary, as go test
// builds them, vet checks included, and runs nothing, so no TestMain or init
// can fail it. A build that fails is errNotBuilt, with what go said.
func compile(ctx context.Context, dir string, out io.Writer) error {
	binaries, err := os.MkdirTemp("", "ctchargen-mutate-tests-*")
	if err != nil {
		return fmt.Errorf("making a directory for the test binaries: %w", err)
	}

	defer func() { _ = os.RemoveAll(binaries) }()

	for _, args := range [][]string{
		{"build", "./..."},
		{"test", "-c", "-o", binaries, "./..."},
	} {
		//nolint:gosec // go itself, with arguments fixed above
		build := exec.CommandContext(ctx, "go", args...)

		build.Dir = dir

		printed, err := build.CombinedOutput()

		failed, err := commandFailed(err)
		if err != nil {
			return fmt.Errorf("building the mutated copy: %w", err)
		}

		if failed {
			return say(out, errNotBuilt, "%sNOT BUILT: %v\n", printed, errNotBuilt)
		}
	}

	return nil
}

// commandFailed reports whether a command ran and exited non-zero, and
// returns any other error, such as a command that could not be started.
func commandFailed(err error) (bool, error) {
	var exit *exec.ExitError

	switch {
	case errors.As(err, &exit):
		return true, nil
	case err != nil:
		return false, err
	}

	return false, nil
}

// say writes a verdict and returns result, or the error writing it.
func say(out io.Writer, result error, format string, args ...any) error {
	_, err := fmt.Fprintf(out, format, args...)
	if err != nil {
		return fmt.Errorf("writing the verdict: %w", err)
	}

	return result
}

// copyModule copies every file git tracks, and every untracked file it does
// not ignore, from root into dest: the tree as it stands, uncommitted work
// included, and nothing built.
func copyModule(ctx context.Context, root, dest string) error {
	list := exec.CommandContext(ctx, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")

	list.Dir = root

	names, err := list.Output()
	if err != nil {
		return fmt.Errorf("listing the tree: %w", err)
	}

	for name := range strings.SplitSeq(strings.TrimSuffix(string(names), "\x00"), "\x00") {
		if strings.HasSuffix(name, "/") {
			continue // a nested repository, such as a worktree, is not this module
		}

		err := copyFile(filepath.Join(root, name), filepath.Join(dest, name))
		if errors.Is(err, os.ErrNotExist) {
			continue // tracked but deleted in the working tree
		}

		if err != nil {
			return fmt.Errorf("copying %s: %w", name, err)
		}
	}

	return nil
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err //nolint:wrapcheck // copyModule names the file
	}

	err = os.MkdirAll(filepath.Dir(to), dirMode)
	if err != nil {
		return err //nolint:wrapcheck // copyModule names the file
	}

	return os.WriteFile(to, data, fileMode) //nolint:wrapcheck // copyModule names the file
}

// replaceOnce replaces old with replacement in path, refusing unless old
// occurs exactly once.
func replaceOnce(path, old, replacement string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading: %w", err)
	}

	if count := strings.Count(string(data), old); count != 1 {
		return fmt.Errorf("%w; it occurs %d times", errNotUnique, count)
	}

	mutated := strings.Replace(string(data), old, replacement, 1)

	err = os.WriteFile(path, []byte(mutated), fileMode)
	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}

// verdict runs command in dir, prints what it printed, and reports whether
// it failed.
func verdict(ctx context.Context, dir string, command []string, out io.Writer) error {
	//nolint:gosec // the command is the one this tool was asked to run
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	cmd.Dir = dir

	var printed bytes.Buffer

	cmd.Stdout = &printed
	cmd.Stderr = &printed

	err := cmd.Run()

	_, writeErr := out.Write(printed.Bytes())
	if writeErr != nil {
		return fmt.Errorf("writing the command's output: %w", writeErr)
	}

	failed, err := commandFailed(err)
	if err != nil {
		return fmt.Errorf("running %s: %w", command[0], err)
	}

	if failed {
		return say(out, nil, "KILLED: the command failed with the mutation in place\n")
	}

	return say(out, errSurvived, "SURVIVED: the command passed with the mutation in place\n")
}
