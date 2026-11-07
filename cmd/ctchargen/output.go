package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// errWouldOverwrite reports an output path that already holds something.
var errWouldOverwrite = errors.New("already exists")

// recordFileMode is the mode a written record carries. A character sheet is
// the operator's own, and nothing else needs to read it.
const recordFileMode = 0o600

// recordDirMode is the mode a directory of records carries, for the same
// reason: it is the operator's own.
const recordDirMode = 0o700

// destination is where a subcommand writes: the stream it was handed when no
// path was given, or a file it opens.
//
// A path that exists is never written over without --force. That refusal is
// the whole reason the flag exists: a referee who writes a new character
// into the file holding last week's has lost last week's, and the tool that
// did it said nothing.
//
// close finishes a write that went through; abandon gives up one that did
// not. They differ only under --force, where finishing puts the new record in
// place and abandoning leaves the old one.
type destination struct {
	out     io.Writer
	close   func() error
	abandon func()
}

func openDestination(out io.Writer, path string, force bool) (destination, error) {
	if path == "" {
		return destination{out: out, close: func() error { return nil }, abandon: func() {}}, nil
	}

	if force {
		return replacing(path)
	}

	file, err := create(path)
	if err != nil {
		return destination{}, err
	}

	return destination{out: file, close: file.Close, abandon: func() { _ = file.Close() }}, nil
}

// replacing stages a record beside path and renames it over path only once
// the whole of it is written (#191).
//
// A truncating open emptied the old record before the first byte of the new
// one landed, so a full disk or an I/O error partway through left neither.
// The rename is atomic on one filesystem. os.CreateTemp creates at 0600,
// which is recordFileMode, so the record is not widened on its way through.
//
// A path that is a directory is refused as the open refused it: a rename
// would report it as "file exists", which reads as the --force refusal.
func replacing(path string) (destination, error) {
	if isDirectory(path) {
		return destination{}, fmt.Errorf("opening %s: %w", path,
			&fs.PathError{Op: "open", Path: path, Err: syscall.EISDIR})
	}

	staged, err := os.CreateTemp(filepath.Dir(path), ".ctchargen-*")
	if err != nil {
		return destination{}, fmt.Errorf("staging %s: %w", path, err)
	}

	finish := func() error {
		err := staged.Close()
		if err == nil {
			err = os.Rename(staged.Name(), path)
		}

		if err != nil {
			_ = os.Remove(staged.Name())

			return fmt.Errorf("putting %s in place: %w", path, err)
		}

		return nil
	}

	abandon := func() {
		_ = staged.Close()
		_ = os.Remove(staged.Name())
	}

	return destination{out: staged, close: finish, abandon: abandon}, nil
}

// create opens a file for writing, refusing to replace one.
func create(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, recordFileMode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %s %w; pass --force to replace it",
				errUsage, path, errWouldOverwrite)
		}

		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	return file, nil
}

// namesDirectory reports whether `-o` asked for a directory rather than a
// .jsonl file. A directory that already exists is one; so is a path written
// with a trailing separator, which is how a referee names a directory that
// is not there yet. Without the second reading, `-o characters/` is a file
// the open fails on and `-o characters` is one opaque file where twenty
// records were asked for.
func namesDirectory(path string) bool {
	if path == "" {
		return false
	}

	if os.IsPathSeparator(path[len(path)-1]) {
		return true
	}

	return isDirectory(path)
}

// isDirectory reports whether a path names a directory that already exists.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// write writes text to a destination and closes it.
func (d destination) write(text string) error {
	_, err := io.WriteString(d.out, text)
	if err != nil {
		// Give it up - a failed write still holds the descriptor, and under
		// --force finishing would put the partial record in place - but
		// report the write, which is what went wrong.
		d.abandon()

		return fmt.Errorf("writing: %w", err)
	}

	return d.done()
}

func (d destination) done() error {
	err := d.close()
	if err != nil {
		return fmt.Errorf("closing the output: %w", err)
	}

	return nil
}

// memberPath is where batch writes member index of count when -o names a
// directory: the one-based index, zero-padded to the width of the count, so a
// directory sorts in generation order and its names say nothing about seeds.
func memberPath(dir string, index, count int) string {
	return filepath.Join(dir, fmt.Sprintf("%0*d.json", len(strconv.Itoa(count)), index))
}
