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

// noteFileMode is the mode a note is created with, before the umask takes its
// share: the operator's umask decides who else may read a note, as it decides
// for any file he makes (#469).
//
// Implements: A new note and a new folder take the operator's umask
// [Command line].
const noteFileMode = 0o666

// noteDirMode is the mode a batch's new folder is created with, before the
// umask, for the same reason.
//
// Implements: A new note and a new folder take the operator's umask
// [Command line].
const noteDirMode = 0o777

// destination is where a subcommand writes: the stream it was handed when no
// path was given, or the file it opened at path. exclusive says the file was
// opened create-exclusive, so this run made it and a write that fails removes
// it.
type destination struct {
	out       io.Writer
	path      string
	file      *os.File
	exclusive bool
}

// openDestination opens where the note goes: out where no path was given, and
// otherwise the file at path, written in place.
//
// A path that exists is never written over without --force. That refusal is
// the whole reason the flag exists: a referee who writes a new character
// into the file holding last week's has lost last week's, and the tool that
// did it said nothing.
//
// Under --force the file is opened once, created or truncated, and written in
// place, as the shell's > writes (#469). That leaves the file to the
// operating system: a symlink is followed to the file it names, even one that
// does not exist yet, and the file keeps its mode and its hard links, with no
// code here. The trade is that a write that fails has already lost the old
// note. A directory never reaches the open: refusedBefore refuses one first
// (#424).
//
// Implements: --force overwrites the file in place [Command line].
func openDestination(out io.Writer, path string, force bool) (destination, error) {
	if path == "" {
		return destination{out: out}, nil
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(path, flags, noteFileMode)
	if errors.Is(err, fs.ErrExist) {
		return destination{}, inTheWay(path)
	}

	if err != nil {
		return destination{}, fmt.Errorf("opening %s: %w", path, withoutPath(path, err))
	}

	return destination{out: file, path: path, file: file, exclusive: !force}, nil
}

// inTheWay is the refusal for a path that already holds something.
//
// A file is refused with the --force advice, because --force overwrites a
// file. A directory is named as a directory (#254): the exclusive open reads
// one as a file that exists, and advising --force sent the referee round
// again only to be told what was really in the way, which --force cannot
// remove.
//
// It is not a usage error: the command line was well formed, and what is in
// the way was found on disk (#472).
//
// Implements: A directory in the way is named as one, with no --force advice
// [Command line].
func inTheWay(path string) error {
	if isDirectory(path) {
		return fmt.Errorf("%s %w", path, syscall.EISDIR)
	}

	return fmt.Errorf("%s %w; pass --force to overwrite it", path, errWouldOverwrite)
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
//
// Without --force, a write that fails removes the note, which the exclusive
// open made this run's own (#468): left, it was a truncated note under its
// own name, and the rerun was refused because the name was taken. Under
// --force a failed write leaves the file as it left it, as the shell's >
// does. An interrupt is the operating system's, and can leave a partial note,
// as it can for cp or curl -o.
//
// Implements: A failed write leaves no note under its name [Command line].
func (d destination) write(text string) error {
	_, err := io.WriteString(d.out, text)

	if d.file != nil {
		closed := d.file.Close()
		if err == nil {
			err = closed
		}
	}

	switch {
	case err == nil:
		return nil
	case d.path == "":
		return fmt.Errorf("writing: %w", err)
	}

	if d.exclusive {
		_ = os.Remove(d.path)
	}

	return fmt.Errorf("writing %s: %w", d.path, withoutPath(d.path, err))
}

// withoutPath is a failure on path without the path the failure names, where it
// names path itself. Every message here names the path the operator gave, so
// the operation's own would name it a second time (#472):
// `opening ro/x.md: open ro/x.md: permission denied`. A failure that names
// another path is left as it stands: MkdirAll reports the ancestor it could
// not make, which the message about path does not name.
//
// Implements: A path is named once in a message [Command line].
func withoutPath(path string, err error) error {
	onPath, ok := errors.AsType[*fs.PathError](err)
	if ok && filepath.Clean(onPath.Path) == filepath.Clean(path) {
		return onPath.Err
	}

	return err
}

// position is member index of count as a note's name opens: the one-based
// index, zero-padded to the width of the count, so a directory sorts in
// generation order.
func position(index, count int) string {
	return fmt.Sprintf("%0*d", len(strconv.Itoa(count)), index)
}

// memberPath is where batch writes the member at a position: the position,
// then the title the character is known by.
//
// Implements: A batch member's note is named for its character [Command line].
func memberPath(dir, at, title string) string {
	return filepath.Join(dir, at+" "+title+".md")
}
