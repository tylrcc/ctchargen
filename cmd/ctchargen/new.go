package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// characterFlags are the flags both generating subcommands take: what a
// character is generated from, as opposed to what is done with it afterwards.
//
// Each command gives them its own descriptions, which legitimately differ;
// the names, types and defaults are declared once, in declareCharacterFlags,
// so the two cannot come to disagree about them. They are carried as one
// value from there on, because five adjacent strings in a signature are five
// strings that can be transposed, and a transposed strategy is one the record
// then reports as the one the operator asked for (#132).
type characterFlags struct {
	set                                   *flag.FlagSet
	name, service, career, skills, muster *string
}

// characterHelp is what one command says about each of its characterFlags.
type characterHelp struct {
	service, name, career, skills, muster string
}

func declareCharacterFlags(set *flag.FlagSet, help characterHelp) characterFlags {
	return characterFlags{
		set:     set,
		service: set.String("service", "", help.service+serviceChoices()),
		name:    set.String("name", "", help.name),
		career:  set.String(careerFlag, chargen.CareerServe.String(), help.career+chargen.CareerChoices()),
		skills:  set.String(skillsFlag, chargen.SkillsAdvanced.String(), help.skills+chargen.SkillsChoices()),
		muster:  set.String(musterFlag, chargen.MusterCash.String(), help.muster+chargen.MusterChoices()),
	}
}

// newFlagSet is new's flag set, declared apart from running the command so a
// test can walk the set it declares.
type newFlagSet struct {
	characterFlags

	auto, sheet, history, readings, force *bool
	output                                *string
}

func declareNewFlags() newFlagSet {
	set := flag.NewFlagSet("new", flag.ContinueOnError)
	set.SetOutput(io.Discard)

	return newFlagSet{
		characterFlags: declareCharacterFlags(set, characterHelp{
			service: "attempt enlistment in this service: ",
			name:    "the character's name",
			career:  "the --auto career `strategy`: ",
			skills:  "the --auto skills `strategy`: ",
			muster:  "the --auto mustering out `strategy`: ",
		}),
		auto:     set.Bool("auto", false, "decide by the policy at every choice, rather than asking"),
		sheet:    set.Bool("sheet", false, "write the character sheet rather than JSON"),
		history:  set.Bool("history", false, "write the generation record rather than JSON"),
		readings: set.Bool("readings", false, readingsHelp),
		output:   set.String("o", "", "write to this file rather than to standard output"),
		force:    set.Bool("force", false, "replace the output file if it already exists"),
	}
}

func newCharacter(args []string, in io.Reader, out, asking io.Writer, dice source) error {
	flags := declareNewFlags()

	err := flags.set.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, newUsage, flags.set)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen new --help`", errUsage, err)
	}

	// `new` takes no positional argument, and a stray word is a typo that
	// would otherwise become an interactive session nobody asked for.
	// The flag package stops at the first non-flag word, so `new foo --auto`
	// leaves --auto unparsed as well: refusing the word catches both.
	if flags.set.NArg() > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, newUsage, flags.set.Arg(0))
	}

	inputs, policy, err := flags.inputs()
	if err != nil {
		return err
	}

	err = strategiesWithoutAuto(flags.set, *flags.auto)
	if err != nil {
		return err
	}

	err = flagsThatContradict(*flags.sheet, *flags.history)
	if err != nil {
		return err
	}

	err = readingsWithoutHistory(*flags.readings, *flags.history)
	if err != nil {
		return err
	}

	return writeCharacter(inputs, policy, newRendering{
		auto: *flags.auto, sheet: *flags.sheet, history: *flags.history, readings: *flags.readings,
		output: *flags.output, force: *flags.force,
	}, in, out, asking, dice)
}

// strategiesWithoutAuto refuses a strategy flag on a guided run. The
// strategies are the policy's, and on a guided run the player answers every
// choice the policy would have: a strategy accepted there would decide
// nothing and appear nowhere in the record. The check is on what the operator
// set, not on the values, because every strategy flag has a default.
//
// Implements: A strategy given without --auto is refused, not ignored [CLI
// sketch].
func strategiesWithoutAuto(set *flag.FlagSet, auto bool) error {
	if auto {
		return nil
	}

	var given []string

	set.Visit(func(f *flag.Flag) {
		switch f.Name {
		case careerFlag, skillsFlag, musterFlag:
			given = append(given, "--"+f.Name)
		}
	})

	if len(given) > 0 {
		return fmt.Errorf("%w: %s %s the --auto policy's; without --auto the player answers every choice",
			errUsage, strings.Join(given, " and "), isOrAre(len(given)))
	}

	return nil
}

func isOrAre(n int) string {
	if n == 1 {
		return "is"
	}

	return "are"
}

// flagsThatContradict refuses a pair that cannot both be meant.
//
// --sheet and --history are one character asked for two ways. rendered() took
// the sheet and said nothing, so a referee who asked for the transcript got a
// sheet and no reason (#58). Neither flag outranks the other; what
// settles it is that preferring one silently is the same defect as swallowing
// a word after a help flag, and this command refuses that.
//
// Implements: A character is written one way, not two [CLI sketch].
func flagsThatContradict(sheet, history bool) error {
	if sheet && history {
		return fmt.Errorf(
			"%w: --sheet and --history cannot both be given; a character is written one way",
			errUsage,
		)
	}

	return nil
}

// readingsHelp is --readings' description, on both commands that write a
// transcript.
const readingsHelp = "end the generation record with the full text of each reading it names"

// readingsWithoutHistory refuses --readings where no transcript is written.
// The readings end a transcript, and nothing else: a flag accepted on a sheet
// or a JSON record would add nothing and say nothing, which is the defect
// flagsThatContradict exists for.
//
// Implements: --readings without --history is refused, not ignored [CLI
// sketch].
func readingsWithoutHistory(readings, history bool) error {
	if readings && !history {
		return fmt.Errorf("%w: --readings ends a transcript, and needs --history", errUsage)
	}

	return nil
}

// newRendering is what the flags say to do with the character once it exists,
// gathered so that newCharacter reads as the four steps it takes rather than
// as its whole flag set threaded through them.
type newRendering struct {
	auto, sheet, history, readings, force bool
	output                                string
}

func writeCharacter(inputs chargen.Inputs, policy chargen.Policy, how newRendering,
	in io.Reader, out, asking io.Writer, dice source,
) error {
	character, err := generate(inputs, policy, how.auto, in, asking, dice)
	if err != nil {
		return err
	}

	text, err := rendered(character, how)
	if err != nil {
		return err
	}

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
}

// inputs is what the parsed flags say to generate, and the policy the
// strategy flags name.
//
// This is the boundary the strategies are parsed at, and the only one: past
// here a Career is a Career, and no part of the engine asks again whether it
// is one of three. That is why chargen has no Validate any more (#41). The
// strategies go into the policy and nowhere else, so an auto record names
// the strategies that decided it because they are the policy's own.
func (f characterFlags) inputs() (chargen.Inputs, chargen.Policy, error) {
	policy, err := chargen.ParsePolicy(*f.career, *f.skills, *f.muster)
	if err != nil {
		return chargen.Inputs{}, policy, fmt.Errorf("%w: %w", errUsage, err)
	}

	// A name is one line. The sheet and the transcript print it as markdown,
	// so a line break in it starts sections the character does not have
	// (#128). Carriage return too: markdown ends a line at either.
	if strings.ContainsAny(*f.name, "\n\r") {
		return chargen.Inputs{}, policy, fmt.Errorf("%w: --name %q is more than one line", errUsage, *f.name)
	}

	in := chargen.Inputs{Name: *f.name}

	if *f.service == "" {
		return in, policy, nil
	}

	for _, known := range traveller.ServiceNames {
		if strings.EqualFold(known.String(), *f.service) {
			in.Forced = chargen.ForceService(known)

			return in, policy, nil
		}
	}

	return in, policy, fmt.Errorf("%w: no service is called %q", errUsage, *f.service)
}

// generate runs the procedure as an auto run under the policy, or as a
// guided run that asks the player.
//
// Interactive mode watches the record as it is written, so the throws and
// their consequences reach the player between his questions; --auto passes
// no observer, because nobody is reading.
func generate(inputs chargen.Inputs, policy chargen.Policy, auto bool,
	in io.Reader, asking io.Writer, dice source,
) (*chargen.Character, error) {
	if auto {
		character, err := chargen.Auto(inputs, policy, dice.options()...)
		if err != nil {
			return nil, fmt.Errorf("generating: %w", err)
		}

		return character, nil
	}

	asked := newPlayer(in, asking)

	character, err := chargen.Guided(inputs, asked, append(dice.options(),
		chargen.WithObserver(asked.watch))...)
	if err != nil {
		return nil, fmt.Errorf("generating: %w", err)
	}

	return character, nil
}

// rendered is the character in whichever of the three shapes was asked for.
func rendered(character *chargen.Character, how newRendering) (string, error) {
	switch {
	case how.sheet:
		text, err := render.Sheet(character)

		return text, wrapRender(err)
	case how.history:
		text, err := render.Transcript(character, how.readings)

		return text, wrapRender(err)
	default:
		encoded, err := render.JSON(character)
		if err != nil {
			return "", fmt.Errorf("rendering: %w", err)
		}

		return string(encoded), nil
	}
}

// wrapRender gives a rendering failure the context of the command that asked
// for it.
func wrapRender(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("rendering: %w", err)
}
