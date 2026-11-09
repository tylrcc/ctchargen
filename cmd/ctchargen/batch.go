package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// batch generates many characters.
//
// It emits what it rolled. Death is an outcome and not an error, so a
// character killed by a survival throw is written like anyone else and no
// flag rerolls him. --survivors does not reroll either: it passes over a dead
// character and generates another, so every character written is one the
// dice made. What changes is who fills a slot (#33).
//
// It always says what it did. A run that wrote seventy-four corpses and
// printed nothing was the trap the report named.
//
// A batch is always an auto run: it has nobody to ask, so it declares no
// --auto, and a command line passing one is refused by the flag package as a
// flag batch does not have, not accepted and ignored (#336).
//
// Implements: A batch is always an auto run, and takes no --auto [CLI sketch].
func batch(args []string, out, asking io.Writer, dice source) error {
	flags := declareBatchFlags()

	err := flags.set.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, batchUsage, flags.set)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen batch --help`", errUsage, err)
	}

	// batch takes no positional argument either, for the reason `new` does
	// not: a word the parser stops at leaves the flags after it unparsed.
	if flags.set.NArg() > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, batchUsage, flags.set.Arg(0))
	}

	if *flags.count < 1 {
		return fmt.Errorf("%w: --count must be at least 1", errUsage)
	}

	base, policy, err := flags.inputs()
	if err != nil {
		return err
	}

	run := batchRun{
		base: base, policy: policy, count: *flags.count,
		survivors: *flags.survivors, attempts: survivorAttempts, dice: dice,
	}

	return write(out, asking, run, *flags.output, *flags.force)
}

// batchRun is what every member of a batch is generated from, and how many
// are wanted: the settings that hold for the whole run, whatever it is
// written to. Where it is written is each writer's own argument.
type batchRun struct {
	base      chargen.Inputs
	policy    chargen.Policy
	count     int
	survivors bool

	// attempts bounds a --survivors search at count*attempts characters.
	attempts int
	dice     source
}

// batchFlagSet is batch's flag set, declared apart from running the command so
// a test can walk the set it declares.
type batchFlagSet struct {
	characterFlags

	count            *int
	force, survivors *bool
	output           *string
}

func declareBatchFlags() batchFlagSet {
	set := flag.NewFlagSet("batch", flag.ContinueOnError)
	set.SetOutput(io.Discard)

	return batchFlagSet{
		characterFlags: declareCharacterFlags(set, characterHelp{
			service: "attempt every enlistment in this service: ",
			name:    "the name to give every character",
			career:  "the career `strategy`: ",
			skills:  "the skills `strategy`: ",
			muster:  "the mustering out `strategy`: ",
		}),
		count: set.Int("count", 0, "how many characters to generate"),
		output: set.String("o", "",
			"a directory to write one note per character; absent, the notes to standard output"),
		force: set.Bool("force", false, "replace output files that already exist"),
		survivors: set.Bool("survivors", false,
			"pass over characters who died, and go on until --count of them are living"),
	}
}

// write sends the batch where it was asked for, and says what it did either
// way: a directory takes one note per character, and standard output takes
// the notes one after another.
//
// Implements: A failed search prints no stopped-run tally beside its counts
// [CLI sketch].
func write(out, asking io.Writer, run batchRun, path string, force bool) error {
	var (
		done tally
		err  error
	)

	if path == "" {
		done, err = run.streamed(out)
	} else {
		done, err = run.intoDirectory(path, force)
	}

	// A run that stopped partway has left notes on disk, and the tally is
	// the only place it says how many and how many of them are corpses (#118).
	// The tally is skipped where it would say nothing - no record written -
	// and for a failed search, whose error already carries the counts.
	if err != nil {
		if done.written > 0 && !errors.Is(err, errSearch) {
			reported := done.reportStopped(asking)
			if reported != nil {
				return errors.Join(err, reported)
			}
		}

		return err
	}

	return done.report(asking)
}

// member generates one member of the batch, from dice of its own.
//
// A failure names the attempt, which under --survivors counts the dead the
// run passed over as well as the living it wrote.
func (b batchRun) member(i int) (*chargen.Character, error) {
	character, err := chargen.Auto(b.base, b.policy, b.dice.options()...)
	if err != nil {
		return nil, fmt.Errorf("attempt %d: %w", i+1, err)
	}

	return character, nil
}

// tally is what a batch did, for the line it prints when it is done.
//
// Implements: A batch says how many it wrote and how many died [CLI sketch].
type tally struct {
	written int
	died    int
	passed  int
}

// report writes the closing line.
//
// It goes to the channel the tool talks to the operator on and never to the
// data channel: a summary on standard output would be a line in the last
// note. After #44 it also arrives after notes the reader has already
// seen, which is why it counts rather than warns.
func (t tally) report(asking io.Writer) error {
	return t.reportAs(asking, fmt.Sprintf("%d written", t.written))
}

// reportStopped writes the closing line of a run that failed partway. It says
// the run stopped, because the error printed after it does, and a line that
// read like a finished batch would contradict the line beneath it.
func (t tally) reportStopped(asking io.Writer) error {
	return t.reportAs(asking, fmt.Sprintf("%d written before the run stopped", t.written))
}

func (t tally) reportAs(asking io.Writer, line string) error {
	switch {
	case t.passed > 0:
		line += fmt.Sprintf(", %d passed over for dying", t.passed)
	case t.died > 0:
		line += fmt.Sprintf(", %d died", t.died)
	default:
		line += ", none died"
	}

	_, err := fmt.Fprintln(asking, line)
	if err != nil {
		return fmt.Errorf("reporting the batch: %w", err)
	}

	return nil
}

// survivorAttempts bounds how many characters --survivors generates per one
// asked for. It reaches the search as batchRun's attempts, which each reads
// rather than this constant, so that a test can reach the bound without
// needing a hundred deaths per character to do it.
//
// It bounds a loop that would otherwise have none, which is the shape of
// defect #54 names elsewhere. The referee who reported #33 measured 74 dead
// in 100 under the default strategy - about four generated a survivor - so a
// hundred is a margin of twenty-five over the worst case anyone has seen, and
// reaching it means something is wrong rather than unlucky.
//
// Implements: `--survivors` gives up rather than drawing forever [CLI sketch].
const survivorAttempts = 100

// each walks the batch, handing each member that is to be written to
// take with its one-based position among them, and reports what it did.
//
// Without --survivors every member is written, dead or alive. With it a dead
// character is passed over and another generated, so a written member is
// still exactly what the dice made. What changes is who fills a slot, not
// what any member means (#33).
//
// Implements: `--survivors` writes only the living, and generates another [CLI
// sketch].
func (b batchRun) each(take func(index int, character *chargen.Character) error) (tally, error) {
	var done tally

	for i := 0; done.written < b.count; i++ {
		if b.survivors && i >= b.count*b.attempts {
			// Not a usage error: the flags were well formed, and what
			// failed was the search. The bound's own comment says
			// reaching it means something is wrong rather than unlucky,
			// and that is a run-time condition and not a misuse.
			return done, fmt.Errorf(
				"%w: --survivors generated %d characters for %d living and found %d",
				errSearch, i, b.count, done.written,
			)
		}

		character, err := b.member(i)
		if err != nil {
			return done, err
		}

		if traveller.Fatal(character.Departure) {
			if b.survivors {
				done.passed++

				continue
			}

			done.died++
		}

		err = take(done.written+1, character)
		if err != nil {
			return done, err
		}

		done.written++
	}

	return done, nil
}

// streamed writes the batch to standard output, one note after another,
// each as it is generated.
//
// A stream is read as it arrives, so `batch --count 50000 | less` shows the
// first note at once and the run holds one character in memory rather than
// all of them (#44). The atomicity that justifies buffering is about not
// leaving a file half-written and not clobbering one on a retry, and there is
// no file to clobber on standard output (#64).
func (b batchRun) streamed(out io.Writer) (tally, error) {
	return b.each(writingTo(out))
}

// writingTo is the take a streamed run uses: write the member's note, a blank
// line after the one before it, as each member is made.
//
// Implements: A batch to standard output arrives a note at a time [CLI
// sketch].
//
// A failure names the member by its position among those written, which is
// the one name a streamed member has.
//
// Implements: A write that fails partway stops the batch, naming the member
// [CLI sketch].
func writingTo(out io.Writer) func(int, *chargen.Character) error {
	return func(index int, character *chargen.Character) error {
		separator := ""
		if index > 1 {
			separator = "\n"
		}

		_, err := io.WriteString(out, separator+render.Note(character, false))
		if err != nil {
			return fmt.Errorf("member %d: %w", index, err)
		}

		return nil
	}
}

// intoDirectory writes one note per character, named by its position and for
// the character in it: "01 Army Captain 938788.md" (#400).
//
// One character is held at a time (#99). A name is not known until its
// character is, so the collision check is by position: a note named for a
// position, alone or followed by a title, holds it, and the positions a batch
// fills are 1 through count whether or not --survivors passes anyone over.
// The check reads the directory once, before anyone is generated, and without
// --force a held position refuses the batch.
//
// Under --force a member's new note is written first, through the same
// staged replace as any other (#191), and only once it is in place are the
// position's other notes removed, so a write that fails leaves the old note
// where it was. A directory at a position is never removed: it is not a
// note.
//
// A batch that fails partway leaves what it has written. The guarantee here is
// that a run never silently replaces a note, not that a directory is written
// atomically.
//
// Implements: A directory batch that cannot write a member leaves what came
// before [CLI sketch].
//
// Implements: A refused batch writes none of itself, on the members it will
// actually write [CLI sketch].
func (b batchRun) intoDirectory(dir string, force bool) (tally, error) {
	err := os.MkdirAll(dir, noteDirMode)
	if err != nil {
		return tally{}, fmt.Errorf("creating %s: %w", dir, err)
	}

	held, err := occupied(dir, b.count)
	if err != nil {
		return tally{}, err
	}

	if !force {
		for index := 1; index <= b.count; index++ {
			if notes := held[index]; len(notes) > 0 {
				return tally{}, inTheWay(notes[0].path)
			}
		}
	}

	return b.each(
		func(index int, character *chargen.Character) error {
			path := memberPath(dir, position(index, b.count), render.Title(character))

			where, err := openDestination(nil, path, force)
			if err != nil {
				return err
			}

			err = where.write(render.Note(character, false))
			if err != nil {
				return err
			}

			return clearStale(held[index], path)
		},
	)
}

// heldNote is an entry in the directory that holds a position.
type heldNote struct {
	path      string
	directory bool
}

// occupied is what already holds each position from 1 through count in dir:
// an entry named for the position alone, as earlier builds named notes, or
// for the position and a title, after a space. The position is read as a
// number, so "1 Kira.md" holds the first position of a batch of twelve as
// "01 Kira.md" does.
//
// It takes a directory and a count, and no policy and no roller, so it cannot
// generate a character even by accident. That signature is the guarantee:
// checking for collisions costs a directory listing and nothing more (#99),
// with or without --survivors, because a member's position is his place among
// the notes written and not who survived to fill it.
//
// Implements: A batch checks for collisions without generating anybody [CLI
// sketch].
func occupied(dir string, count int) (map[int][]heldNote, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	held := map[int][]heldNote{}

	for _, entry := range entries {
		stem, isNote := strings.CutSuffix(entry.Name(), ".md")
		at, _, _ := strings.Cut(stem, " ")

		index, err := strconv.Atoi(at)
		if !isNote || err != nil || !allDigits(at) || index < 1 || index > count {
			continue
		}

		held[index] = append(held[index], heldNote{
			path: filepath.Join(dir, entry.Name()), directory: entry.IsDir(),
		})
	}

	return held, nil
}

// allDigits is whether a word is a position as a batch writes one: digits
// and nothing else, where strconv.Atoi would also take a sign.
func allDigits(word string) bool {
	return word != "" && strings.Trim(word, "0123456789") == ""
}

// clearStale removes a position's other notes once its new one is written.
// Only a note is removed: a directory at the position is left, because it is
// not a note. Without --force nothing is held at a position the batch writes,
// so there is nothing to remove.
func clearStale(held []heldNote, written string) error {
	for _, note := range held {
		if note.path == written || note.directory {
			continue
		}

		err := os.Remove(note.path)
		if err != nil {
			return fmt.Errorf("replacing %s: %w", note.path, err)
		}
	}

	return nil
}
