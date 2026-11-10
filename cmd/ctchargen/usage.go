package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// commandList is the subcommands in the order run's switch takes them. Both
// messages that name them read it here, so the two cannot come to disagree
// about the order again.
const commandList = "new|batch|version"

// The usage line of each subcommand: what it takes that is not a flag.
//
// Each is the shape alone: errUsage prints the word "usage" ahead of an
// error, and writeHelp prints "usage: ctchargen " ahead of the help.
//
// None of them enumerates its flags. An enumeration typed beside a flag set
// is a line that goes stale the first time a flag is added; the list
// writeFlags prints from the set itself is what carries the detail, and it
// cannot go stale.
const (
	newUsage     = "new [flags]"
	batchUsage   = "batch --count N [flags]"
	versionUsage = "version"
)

// topLevelUsage answers `ctchargen --help`: what the commands are, where to
// ask about one of them, and how a flag is written.
//
// Implements: The help says how flags are written [Command line].
const topLevelUsage = "usage: ctchargen <command> [flags]\n" +
	"\n" +
	"  new      generate one character, asking at every choice unless --auto\n" +
	"  batch    generate many characters, each decided by the policy\n" +
	"  version  write the build\n" +
	"\n" +
	"Run `ctchargen <command> --help` for that command's flags.\n" +
	"\n" + flagGrammar

// flagGrammar is how a flag is written, which is Go's flag package's grammar,
// said in the top-level help (topLevelUsage) and after each command's flags
// (writeHelp) (#466). A switch such as --auto takes no value after a space,
// only after =, and -- ends the flags only where a flag could stand: after
// -o it is -o's value.
const flagGrammar = "Flags are Go's: one dash or two (-auto or --auto). A flag that takes a value\n" +
	"has it after = or a space (-o note.md or -o=note.md); a switch such as --auto\n" +
	"takes none. -- where a flag could stand ends them.\n"

// The three strategy flags, named once, because `new` and `batch` each
// define all three and a name typed six times is a name that can be typed
// wrong once.
const (
	careerFlag = "career"
	skillsFlag = "skills"
	musterFlag = "muster"
)

// shortFlag is the length of a flag name written with one dash. There is one
// of them, `-o`, and it is written that way everywhere else.
const shortFlag = 1

// writeTopLevelHelp answers --help before a subcommand is named.
func writeTopLevelHelp(out io.Writer) error {
	return writeUsageText(out, topLevelUsage)
}

// helpTakesNoArguments is the refusal of a word after a help flag, wherever
// one is met (#69, #84).
func helpTakesNoArguments(given string) string {
	return fmt.Sprintf("help takes no arguments, and was given %q", given)
}

// writeUsageText writes a usage that is already composed.
func writeUsageText(out io.Writer, text string) error {
	_, err := io.WriteString(out, text)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	return nil
}

// writeHelp answers --help on a subcommand: its usage line, then its flags.
//
// It writes to out and returns nil, so that help asked for leaves by the
// same door as the note and exits 0 - not to `asking`, which would put
// `new --help | less` on the wrong stream. It cannot reach a `-o` file:
// parsing happens before openDestination, and a run that comes here never
// opens one.
//
// A word after a help flag is refused, not discarded, the same as everywhere
// else a command meets an unexpected word (#69): `new --help junk`, `batch
// --help junk` and `version --help junk` refuse it as
// `ctchargen help junk` does (#84). The flag package returns ErrHelp before a
// command reaches its own NArg check, so the word has to be looked at here.
//
// Nobody loses a character to it. It is consistency, and the kind of thing a
// referee finds by typing.
//
// Implements: A word after a help flag is refused, not discarded
// [Command line].
//
// Implements: The help says how flags are written [Command line].
func writeHelp(out io.Writer, line string, flags *flag.FlagSet) error {
	if flags.NArg() > 0 {
		return fmt.Errorf("%w: %s; %s", errUsage, line, helpTakesNoArguments(flags.Arg(0)))
	}

	_, err := fmt.Fprintf(out, "usage: ctchargen %s\n\n", line)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	err = writeFlags(out, flags)
	if err != nil {
		return err
	}

	return writeUsageText(out, "\n"+flagGrammar)
}

// helpLine is one flag as the help prints it: the name and kind on the left,
// the description on the right.
type helpLine struct{ named, description string }

// writeFlags lists a command's flags, with their descriptions and defaults.
//
// It exists rather than flag.PrintDefaults because PrintDefaults writes one
// dash and everything else here writes two: the README, the error messages,
// and the referee's own notes. The defaults are printed because a
// strategy flag left alone still chooses, and the reader who could not see
// which one it chose is the one who reported this.
//
// The description column is measured from the set rather than typed as a
// constant, so a flag with a longer name than any here arrives aligned
// instead of pushing one line out of the column nothing checks.
func writeFlags(out io.Writer, flags *flag.FlagSet) error {
	var (
		lines []helpLine
		width int
	)

	flags.VisitAll(func(each *flag.Flag) {
		unwrapped, description := flag.UnquoteUsage(each)
		kind := kindOf(each, unwrapped)

		named := "  " + dashed(each.Name)
		if kind != "" {
			named += " " + kind
		}

		if !zeroDefault(each.DefValue) {
			description += " (default " + each.DefValue + ")"
		}

		// A column one wider than the longest name, so that even the
		// longest line has two spaces before its description rather than
		// running into it.
		width = max(width, len(named)+1)
		lines = append(lines, helpLine{named: named, description: description})
	})

	for _, line := range lines {
		_, err := fmt.Fprintf(out, "%-*s %s\n", width, line.named, line.description)
		if err != nil {
			return fmt.Errorf("writing the flags: %w", err)
		}
	}

	return nil
}

// kindOf is the kind of value a flag takes, as the help names it. The flag
// package names it from the value's concrete type, and every flag here is
// wrapped (strictly), so it says only "value"; the kind is read from what
// the value holds instead (#466). A name the description gives in backquotes
// stands, as the package's own does.
//
// Implements: The help names the kind of value each flag takes
// [Command line].
func kindOf(declared *flag.Flag, named string) string {
	getter, ok := declared.Value.(flag.Getter)
	if named != "value" || !ok {
		return named
	}

	return fmt.Sprintf("%T", getter.Get())
}

// dashed writes a flag the way this command line writes it: -o, --auto. The
// flag package accepts either spelling for either name; this is about what
// the reader is shown.
func dashed(name string) string {
	if len(name) == shortFlag {
		return "-" + name
	}

	return "--" + name
}

// flagNamed finds the flag in the flag package's errors: what precedes the
// one dash it writes every flag with, and the name after it. One error, a
// switch given a value it refuses, writes the name with no dash at all.
var flagNamed = regexp.MustCompile(
	`^(flag provided but not defined: |flag needs an argument: |invalid boolean flag |` +
		`invalid (?:boolean )?value "(?:[^"\\]|\\.)*" for (?:flag )?)-?([^\s:=]+)`,
)

// flagMistake is the flag package's error, as text, with its flag written as
// the help writes it (#472): `--count`, `--output`, `-o`. The package writes
// every flag with one dash, whichever the operator typed, so its errors named
// a flag the way the help, the README and the other errors never write it.
//
// Implements: A flag in an error is written as the help writes it
// [Command line].
func flagMistake(err error) string {
	message := err.Error()

	named := flagNamed.FindStringSubmatchIndex(message)
	if named == nil {
		return message
	}

	return message[:named[3]] + dashed(message[named[4]:named[5]]) + message[named[5]:]
}

// zeroDefault reports a default not worth printing. The flag package's own
// isZeroValue is unexported; these three cover every flag this command line
// has, and a flag whose default is worth reading is never one of them.
func zeroDefault(value string) bool {
	return value == "" || value == "0" || value == "false"
}

// serviceChoices is the --service flag's own description: the six of p. 10,
// in the book's order, spelled the way they are typed.
func serviceChoices() string {
	names := make([]string, 0, len(traveller.ServiceNames))

	for _, service := range traveller.ServiceNames {
		names = append(names, strings.ToLower(service.String()))
	}

	return strings.Join(names, ", ")
}

// errGivenTwice reports a flag given more than once. The flag package takes
// the last value; which one was meant is unclear, so it is refused (#473).
var errGivenTwice = errors.New("given twice")

// errEmptyValue reports a flag given an empty value, which this command read
// as the flag not given: an empty --service picked a service, and an empty -o
// wrote to standard output (#473).
var errEmptyValue = errors.New("an empty value")

// strictly wraps every flag a set declares, so that a flag given twice, or
// given an empty value, is refused and not quietly read one way. Where an
// input's meaning is unclear the command line refuses it: giving it a meaning
// later is an addition, and taking one away is a break (#466).
//
// The flag package wraps the refusal as "invalid value ... for flag -name",
// which flagMistake writes with the flag as the help writes it.
//
// Implements: A flag given twice, or given an empty value, is refused
// [Command line].
func strictly(set *flag.FlagSet) {
	set.VisitAll(func(declared *flag.Flag) {
		getter, ok := declared.Value.(flag.Getter)
		if !ok {
			panic("every flag this command line declares can be read back: " + declared.Name)
		}

		strict := strictValue{Getter: getter}

		if boolean, isBool := declared.Value.(interface{ IsBoolFlag() bool }); isBool && boolean.IsBoolFlag() {
			declared.Value = &strictBool{strictValue: strict}

			return
		}

		declared.Value = &strict
	})
}

// strictValue is a flag's value that may be set once, to something.
type strictValue struct {
	flag.Getter

	given bool
}

func (v *strictValue) Set(value string) error {
	switch {
	case v.given:
		return errGivenTwice
	case value == "":
		return errEmptyValue
	}

	v.given = true

	// The flag package wraps what Set returns as "invalid value ... for flag
	// -name", so the flag's own refusal needs nothing added here.
	return v.Getter.Set(value) //nolint:wrapcheck // the flag package names the flag and the value around it.
}

// strictBool is a strictValue the flag package still reads as a switch, so
// --auto needs no value after it.
type strictBool struct{ strictValue }

func (*strictBool) IsBoolFlag() bool { return true }

// errNotAWholeNumber reports a --count that is not decimal digits.
var errNotAWholeNumber = errors.New("not a whole number")

// decimalCount is --count: decimal digits and nothing else (#466). The flag
// package's own int reads base 0, so "010" was eight and "0x10" sixteen; a
// count is a number a person types, and its digits mean what they say.
//
// Implements: --count is decimal digits only [Command line].
type decimalCount int

func (c *decimalCount) Set(value string) error {
	if strings.TrimLeft(value, "0123456789") != "" {
		return errNotAWholeNumber
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%w: %w", errNotAWholeNumber, err)
	}

	*c = decimalCount(n)

	return nil
}

func (c *decimalCount) String() string { return strconv.Itoa(int(*c)) }

// Get is the count as the int it is, which the CLI baseline reads its type
// from.
func (c *decimalCount) Get() any { return int(*c) }
