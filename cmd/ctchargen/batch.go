package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// batch generates many characters.
//
// It writes only the living (#421). Death is an outcome and not an error, and
// nothing rerolls a character: a dead one is passed over and another
// generated, so every member written is one the dice made. What changes is
// who fills a slot, not what any member means (#33).
//
// It always says what it did. A run that wrote seventy-four corpses, in the
// days a batch wrote the dead, and printed nothing was the trap #33 named.
//
// A batch is always an auto run: it has nobody to ask, so it declares no
// --auto, and a command line passing one is refused by the flag package as a
// flag batch does not have, not accepted and ignored (#336).
//
// Implements: A batch is always an auto run, and takes no --auto [CLI sketch].
//
// Implements: A batch's -o names a folder, and a batch takes no --force [CLI
// sketch].
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

	// -o names the folder a batch writes into. One ending in .md almost
	// certainly meant a file, which a batch does not write, so it is refused
	// before anything is made rather than made into a folder of that name
	// (#416).
	if strings.EqualFold(filepath.Ext(filepath.Clean(*flags.output)), ".md") {
		return fmt.Errorf("%w: -o %s names a file; a batch writes its notes into a folder",
			errUsage, *flags.output)
	}

	base, policy, err := flags.inputs()
	if err != nil {
		return err
	}

	run := batchRun{
		base: base, policy: policy, count: *flags.count,
		attempts: attemptsPerMember, dice: dice,
	}

	return write(out, asking, run, *flags.output)
}

// batchRun is what every member of a batch is generated from, and how many
// are wanted: the settings that hold for the whole run, whatever it is
// written to. Where it is written is each writer's own argument.
type batchRun struct {
	base   chargen.Inputs
	policy chargen.Policy
	count  int

	// attempts bounds the search for the living at count*attempts characters.
	attempts int
	dice     source
}

// batchFlagSet is batch's flag set, declared apart from running the command so
// a test can walk the set it declares.
type batchFlagSet struct {
	characterFlags

	count  *int
	output *string
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
			"a folder of the batch's own to write one note per character into; absent, the notes to standard output"),
	}
}

// write sends the batch where it was asked for, and says what it did either
// way: a directory takes one note per character, and standard output takes
// the notes one after another.
//
// Implements: A failed search prints no stopped-run tally beside its counts
// [CLI sketch].
func write(out, asking io.Writer, run batchRun, path string) error {
	var (
		done tally
		err  error
	)

	if path == "" {
		done, err = run.streamed(out)
	} else {
		done, err = run.intoDirectory(path)
	}

	// A run that stopped partway has left notes on disk, and the tally is
	// the only place it says how many it wrote and passed over (#118).
	// The tally is skipped where it would say nothing - no note written -
	// and for a failed search, whose error already carries the counts; a
	// failed search into a folder still says the folder holds what it wrote.
	if err != nil && done.written > 0 {
		var reported error

		if errors.Is(err, errSearch) {
			reported = heldIn(asking, path)
		} else {
			reported = done.reportStopped(asking, path)
		}

		if reported != nil {
			return errors.Join(err, reported)
		}
	}

	if err != nil {
		return err
	}

	return done.report(asking)
}

// member generates one member of the batch, from dice of its own.
//
// A failure names the attempt, which counts the dead the run passed over as
// well as the living it wrote.
func (b batchRun) member(i int) (*chargen.Character, error) {
	character, err := chargen.Auto(b.base, b.policy, b.dice.options()...)
	if err != nil {
		return nil, fmt.Errorf("attempt %d: %w", i+1, err)
	}

	return character, nil
}

// tally is what a batch did, for the line it prints when it is done.
//
// Implements: A batch says how many it wrote and how many it passed over [CLI
// sketch].
type tally struct {
	written int
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
// read like a finished batch would contradict the line beneath it. A batch
// into a folder then says the folder holds what was written (heldIn).
func (t tally) reportStopped(asking io.Writer, dir string) error {
	err := t.reportAs(asking, fmt.Sprintf("%d written before the run stopped", t.written))
	if err != nil {
		return err
	}

	return heldIn(asking, dir)
}

// heldIn says a stopped batch's folder holds what was written: the folder is
// not empty now, so the batch would refuse it on a rerun (#426). A batch to
// standard output has no folder, and says nothing.
func heldIn(asking io.Writer, dir string) error {
	if dir == "" {
		return nil
	}

	_, err := fmt.Fprintf(asking,
		"%s holds what was written; remove it or choose another folder to run again\n", dir)
	if err != nil {
		return fmt.Errorf("reporting the batch: %w", err)
	}

	return nil
}

// reportAs ends a closing line with how many were passed over for dying,
// where any were; a batch that passed over nobody says only what it wrote.
func (t tally) reportAs(asking io.Writer, line string) error {
	if t.passed > 0 {
		line += fmt.Sprintf(", %d passed over for dying", t.passed)
	}

	_, err := fmt.Fprintln(asking, line)
	if err != nil {
		return fmt.Errorf("reporting the batch: %w", err)
	}

	return nil
}

// attemptsPerMember bounds how many characters a batch generates per one
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
// Implements: A batch gives up rather than drawing forever [CLI sketch].
const attemptsPerMember = 100

// each walks the batch, handing each living member to take with its
// one-based position among them, and reports what it did.
//
// A dead character is passed over and another generated (#421), so a
// written member is still exactly what the dice made. What changes is who
// fills a slot, not what any member means (#33).
//
// Implements: A batch writes only the living, and generates another [CLI
// sketch].
func (b batchRun) each(take func(index int, character *chargen.Character) error) (tally, error) {
	var done tally

	for i := 0; done.written < b.count; i++ {
		if i >= b.count*b.attempts {
			// Not a usage error: the flags were well formed, and what
			// failed was the search. The bound's own comment says
			// reaching it means something is wrong rather than unlucky,
			// and that is a run-time condition and not a misuse.
			return done, fmt.Errorf(
				"%w: the batch generated %d characters for %d living and found %d",
				errSearch, i, b.count, done.written,
			)
		}

		character, err := b.member(i)
		if err != nil {
			return done, err
		}

		if traveller.Fatal(character.Departure) {
			done.passed++

			continue
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

// intoDirectory writes one note per member into a folder of the batch's own,
// named by its position and for the character in it: "01 Army Captain
// 938788.md" (#400).
//
// A batch writes only into a folder that is new, or empty but for hidden
// entries (#426), so nothing in the folder can be anyone's but this run's
// and nothing is ever removed. The check reads the folder once, before
// anyone is generated (#99). One character is held at a time.
//
// A batch that fails partway leaves what it has written, and says so: the
// folder is not empty any more, so running again needs it removed or another
// folder.
//
// Implements: A directory batch that cannot write a member leaves what came
// before [CLI sketch].
func (b batchRun) intoDirectory(dir string) (tally, error) {
	err := os.MkdirAll(dir, noteDirMode)
	if err != nil {
		return tally{}, fmt.Errorf("creating %s: %w", dir, err)
	}

	err = ownFolder(dir)
	if err != nil {
		return tally{}, err
	}

	return b.each(
		func(index int, character *chargen.Character) error {
			where, err := openDestination(nil,
				memberPath(dir, position(index, b.count), render.Title(character)), false)
			if err != nil {
				return err
			}

			return where.write(render.Note(character, false))
		},
	)
}

// ownFolder refuses a folder holding anything a person would see in a
// listing: a note, another file, a subfolder. Hidden entries, such as the
// .DS_Store Finder leaves, do not count.
//
// It takes a folder and nothing that could generate, so the check costs a
// listing and never a character (#99).
//
// Implements: A batch writes only into a folder of its own [CLI sketch].
//
// Implements: A batch refuses its folder before generating anybody [CLI
// sketch].
func ownFolder(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			return fmt.Errorf("%w: %s is not empty; a batch writes into a folder of its own, a new or empty one",
				errUsage, dir)
		}
	}

	return nil
}
