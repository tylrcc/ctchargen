package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
	"github.com/philoserf/ctchargen/traveller"
)

// maxDrawnSeed bounds a seed the tool draws for itself at 2^53 - 1, above
// which consecutive integers are no longer all exactly representable in an
// IEEE-754 double. A reader that parses JSON numbers as doubles would
// otherwise round it silently, and a rounded seed is a different character.
//
// The bound covers only a drawn seed. An explicit --seed is the operator's
// own number and is written to the record as given.
const maxDrawnSeed = 1<<53 - 1

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
	seed                                  *uint64
	name, service, career, skills, muster *string
}

// characterHelp is what one command says about each of its characterFlags.
type characterHelp struct {
	seed, service, name, career, skills, muster string
}

func declareCharacterFlags(set *flag.FlagSet, help characterHelp) characterFlags {
	return characterFlags{
		set:     set,
		seed:    set.Uint64(seedFlag, 0, help.seed),
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

	auto, sheet, history, force *bool
	output, replay              *string
}

func declareNewFlags() newFlagSet {
	set := flag.NewFlagSet("new", flag.ContinueOnError)
	set.SetOutput(io.Discard)

	return newFlagSet{
		characterFlags: declareCharacterFlags(set, characterHelp{
			seed:    "the seed to generate from; one is drawn if absent",
			service: "attempt enlistment in this service: ",
			name:    "the character's name",
			career:  "the --auto career `strategy`: ",
			skills:  "the --auto skills `strategy`: ",
			muster:  "the --auto mustering out `strategy`: ",
		}),
		auto:    set.Bool("auto", false, "decide by the policy at every choice, rather than asking"),
		sheet:   set.Bool("sheet", false, "write the character sheet rather than JSON"),
		history: set.Bool("history", false, "write the generation record rather than JSON"),
		output:  set.String("o", "", "write to this file rather than to standard output"),
		force:   set.Bool("force", false, "replace the output file if it already exists"),
		replay: set.String(answersFlag, "",
			"answers to give before asking, as 1,2,1"),
	}
}

func newCharacter(args []string, in io.Reader, out, asking io.Writer) error {
	flags := declareNewFlags()

	err := flags.set.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, newUsage, flags.set)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen new --help`", errUsage, err)
	}

	// `new` takes no positional argument, and a stray word is a typo that
	// would otherwise become an interactive session on a seed nobody chose.
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

	answers, err := answersFrom(*flags.replay)
	if err != nil {
		return err
	}

	err = flagsThatContradict(*flags.auto, len(answers) > 0, *flags.sheet, *flags.history)
	if err != nil {
		return err
	}

	return writeCharacter(inputs, policy, answers, newRendering{
		auto: *flags.auto, sheet: *flags.sheet, history: *flags.history,
		output: *flags.output, force: *flags.force,
	}, in, out, asking)
}

// flagsThatContradict refuses two pairs that cannot both be meant.
//
// --auto decides at every choice, so there is nobody for a replayed answer to
// stand in for: taking the list and never consulting it would hand back a
// character the answers had no part in, and nothing on the sheet would say so.
//
// --sheet and --history are one character asked for two ways. rendered() took
// the sheet and said nothing, so a referee who asked for the transcript got a
// sheet and no reason (#58). Neither flag is in the PRD's CLI sketch; what
// settles it is that preferring one silently is the same defect as swallowing
// a word after a help flag, and this command refuses that.
func flagsThatContradict(auto, replaying, sheet, history bool) error {
	if auto && replaying {
		return fmt.Errorf("%w: --%s and --auto cannot both be given; --auto answers every question itself",
			errUsage, answersFlag)
	}

	if sheet && history {
		return fmt.Errorf(
			"%w: --sheet and --history cannot both be given; a character is written one way",
			errUsage,
		)
	}

	return nil
}

// newRendering is what the flags say to do with the character once it exists,
// gathered so that newCharacter reads as the four steps it takes rather than
// as its whole flag set threaded through them.
type newRendering struct {
	auto, sheet, history, force bool
	output                      string
}

func writeCharacter(inputs chargen.Inputs, policy chargen.Policy, answers []int, how newRendering,
	in io.Reader, out, asking io.Writer,
) error {
	character, err := generate(inputs, policy, how.auto, answers, in, asking)
	if err != nil {
		return err
	}

	stamp(character)

	text, err := rendered(character, how.sheet, how.history)
	if err != nil {
		return err
	}

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
}

func isSet(flags *flag.FlagSet, name string) bool {
	found := false

	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})

	return found
}

// drawSeed draws a seed the tool chose for itself, anywhere in the documented
// range including its top.
//
// rand.Uint64N's argument is exclusive, so passing maxDrawnSeed yielded
// [0, 2^53-2] where the bound above says 2^53-1 inclusive (#56). Exactly one
// seed of nine quadrillion was unreachable, which nobody would ever notice -
// but the comment is the promise, and the test that guarded it asserted the
// code's bound rather than the comment's.
//
// random is a parameter so the top of the range can be reached deliberately:
// drawing until it comes up is not a test anyone can run.
func drawSeed(random func(uint64) uint64) uint64 {
	return random(maxDrawnSeed + 1)
}

// inputs is what the parsed flags say to generate, and the policy that
// answers for it.
//
// This is the boundary the strategies are parsed at, and the only one: past
// here a Career is a Career, and no part of the engine asks again whether it
// is one of three. That is why chargen has no Validate any more (#41). The
// inputs and the policy are both written from that one parse, so what a
// record says it was given and what decided it cannot name different
// strategies.
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

	in := chargen.Inputs{
		Seed: *f.seed, Name: *f.name,
		Career: policy.Career, Skills: policy.Skills, Muster: policy.Muster,
	}

	if !isSet(f.set, seedFlag) {
		in.Seed = drawSeed(rand.Uint64N)
	}

	if *f.service == "" {
		return in, policy, nil
	}

	for _, known := range traveller.ServiceNames {
		if strings.EqualFold(known.String(), *f.service) {
			in.Service, in.Forced = known, true

			return in, policy, nil
		}
	}

	return in, policy, fmt.Errorf("%w: no service is called %q", errUsage, *f.service)
}

// answersFrom reads --answers, which is a list of the numbers a player typed.
//
// Empty is not an error and not an empty run: it is the ordinary case, a
// session with nothing typed in advance.
func answersFrom(list string) ([]int, error) {
	if list == "" {
		return nil, nil
	}

	fields := strings.Split(list, ",")

	answers := make([]int, 0, len(fields))

	for _, field := range fields {
		chosen, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return nil, fmt.Errorf("%w: --%s takes numbers separated by commas, and %q is not one",
				errUsage, answersFlag, field)
		}

		answers = append(answers, chosen)
	}

	return answers, nil
}

// generate runs the procedure under the auto policy, or asks the player.
//
// The policy is the one characterFlags.inputs parsed beside the inputs, so
// that what is checked and what is written into the record cannot name
// different strategies. They are parsed in both modes: an interactive run
// records them too, and a record naming a strategy that is not a POLICY.md
// row is one its own schema refuses.
//
// Interactive mode watches the record as it is written, so the throws and
// their consequences reach the player between his questions; --auto passes
// no observer, because nobody is reading.
func generate(inputs chargen.Inputs, policy chargen.Policy, auto bool, answers []int,
	in io.Reader, asking io.Writer,
) (*chargen.Character, error) {
	if auto {
		character, err := chargen.Generate(inputs, policy)
		if err != nil {
			return nil, fmt.Errorf("generating: %w", err)
		}

		return character, nil
	}

	asked := newPlayer(in, asking, answers)

	character, err := chargen.Generate(inputs, asked,
		chargen.WithObserver(asked.watch), chargen.WithAnswerer(traveller.ByPlayer))
	if err != nil {
		return nil, fmt.Errorf("generating: %w", err)
	}

	left := asked.leftover()
	if left != nil {
		return nil, left
	}

	return character, nil
}

// rendered is the character in whichever of the three shapes was asked for.
func rendered(character *chargen.Character, sheet, history bool) (string, error) {
	switch {
	case sheet:
		text, err := render.Sheet(character)

		return text, wrapRender(err)
	case history:
		text, err := render.Transcript(character)

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
