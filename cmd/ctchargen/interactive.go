package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// errNoAnswer reports that the input ended while the procedure was still
// asking. It is an error rather than a default, because a character finished
// by a decision nobody made is not the character anybody asked for.
//
// Implements: The input ending is an error, not a default [Choices].
var errNoAnswer = errors.New("the input ended while the procedure was still asking")

// player is the Decider a person drives: the third implementation, after the
// auto policy and the scripted deciders the tests use, and the only one that
// answers at a keyboard.
//
// It never invents an answer. Every question it asks is one the engine
// asked, with exactly the alternatives the engine offered, and an answer it
// cannot read is asked again rather than guessed at.
//
// Implements: Every question the player is asked is one the engine asked
// [Choices].
type player struct {
	in  *bufio.Scanner
	out io.Writer

	// given is every answer accepted so far, in order, which is what a
	// stopped session is told it loses and what nothing else records: the
	// record holds what was chosen, not which number was typed to choose it.
	given []int

	// wrote is the first failure writing to the player, kept so that a
	// broken pipe stops the generation at the next question rather than
	// being swallowed a dozen times over.
	wrote error
}

func newPlayer(in io.Reader, out io.Writer) *player {
	if in == nil {
		// Nothing to read from is not a reason to invent answers; the first
		// question asked will report that the input ended.
		in = strings.NewReader("")
	}

	return &player{in: bufio.NewScanner(in), out: out}
}

// pick asks which of a list the player wants and hands back the thing
// itself, so that every choice over a list is one function rather than the
// same "ask, then index" pair written out once per choice point - and its
// failure is one branch rather than one per choice point.
func pick[T any](p *player, question string, from []T, name func(T) string) (T, error) {
	options := make([]string, 0, len(from))
	for _, v := range from {
		options = append(options, name(v))
	}

	chosen, err := p.choose(question, options)
	if err != nil {
		var none T

		return none, err
	}

	return from[chosen], nil
}

func (p *player) Service(from []traveller.EnlistmentOffer, _ chargen.Vary) (traveller.ServiceName, error) {
	offer, err := pick(p, "Which service will you try to enlist in? (p. 10)", from,
		func(o traveller.EnlistmentOffer) string {
			return fmt.Sprintf("%v (enlistment %v, and you earn %+d)", o.Service, o.Target, o.DM)
		})

	return offer.Service, err
}

// SubmitToDraft says, before the answer, what declining costs. The page
// cite is the question's; the consequence is this tool's reading of it,
// E001, which ends generation with an eighteen-year-old civilian. It is
// stated plainly rather than credited to the page, and nothing later in the
// session would say so until the note came back with no service in it.
//
// Implements: The draft question says that declining ends generation [1:5
// (E001)].
func (p *player) SubmitToDraft() (bool, error) {
	return p.confirm("Rejected. Submit to the draft? (p. 5) " +
		"Declining ends the generation with a character who never served.")
}

// AttemptCommission shows the commission throw before the answer, and that
// p. 6 names no cost for a failed one. The page says only "A character may
// attempt to acquire a commission once per term of service until he is
// successful", and prices nothing for the throws in between.
//
// Implements: A throw's question shows its target and DM before the answer
// [Choices].
func (p *player) AttemptCommission(odds traveller.Odds) (bool, error) {
	return p.confirm(fmt.Sprintf(
		"Attempt a commission this term? (commission %v; p. 6 names no cost for a failure)", oddsText(odds),
	))
}

// AttemptPromotion shows the promotion throw as AttemptCommission shows its
// own. P. 6 names no cost for a failed promotion either; it says only "A
// character is eligible for one promotion per term of service."
//
// Implements: A throw's question shows its target and DM before the answer
// [Choices].
func (p *player) AttemptPromotion(odds traveller.Odds) (bool, error) {
	return p.confirm(fmt.Sprintf(
		"Attempt a promotion this term? (promotion %v; p. 6 names no cost for a failure)", oddsText(odds),
	))
}

// oddsText is a throw as the questions show it: its target and its DM.
func oddsText(odds traveller.Odds) string {
	return fmt.Sprintf("%v, DM %+d", odds.Target, odds.DM)
}

// SkillTable says which of the term's rolls this is and why the term has
// that many, and shows each table's six results beside its name, so the
// table is chosen knowing what each result gives. The name is the record's, so the
// menu, the echo and the note call a table one thing.
//
// Implements: The skills question says which roll of how many, and why
// [Choices].
// Implements: A skills table offered shows its six results [Choices].
func (p *player) SkillTable(from []traveller.SkillTableOffer, roll traveller.SkillRoll) (traveller.SkillTable, error) {
	question := fmt.Sprintf("Which skills table? Roll %d of %d: %s. It is designated before the die (p. 11)",
		roll.N, roll.Of.Rolls(), earnedText(roll.Of))

	offer, err := pick(p, question, from, func(o traveller.SkillTableOffer) string {
		results := make([]string, 0, len(o.Results))
		for _, result := range o.Results {
			results = append(results, resultText(result))
		}

		return o.Table.String() + ": " + strings.Join(results, ", ")
	})

	return offer.Table, err
}

// earnedText says what gave the term its skills rolls, leaving out a source
// that gave none.
func earnedText(earned traveller.Earned) string {
	var parts []string

	for _, source := range []struct {
		rolls int
		what  string
	}{
		{earned.Term, "the term"},
		{earned.Commission, "the commission"},
		{earned.Promotion, "the promotion"},
	} {
		if source.rolls > 0 {
			parts = append(parts, fmt.Sprintf("%d for %s", source.rolls, source.what))
		}
	}

	return strings.Join(parts, ", ")
}

// resultText is one table result as the skills menu shows it: a
// characteristic change with its sign, a skill by name, and a weapon
// category, whose weapon is named when it is rolled.
func resultText(result traveller.TableResult) string {
	var text resultWords

	// A fold that only assigns cannot fail.
	_ = result.Fold(&text)

	return text.words
}

// resultWords is the fold resultText reads a result through, so a fourth
// kind of result cannot be added without this seeing it.
type resultWords struct{ words string }

func (r *resultWords) Alteration(characteristic traveller.Characteristic, delta int) error {
	r.words = fmt.Sprintf("%+d %v", delta, characteristic)

	return nil
}

func (r *resultWords) Skill(name traveller.SkillName) error {
	r.words = string(name)

	return nil
}

func (r *resultWords) WeaponPick(category traveller.WeaponCategory) error {
	r.words = category.String()

	return nil
}

func (p *player) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, _ chargen.Vary,
) (traveller.WeaponName, error) {
	question := fmt.Sprintf("%v: name the weapon, immediately (pp. 11-13)", category)

	return pick(p, question, from, func(w traveller.WeaponName) string { return string(w) })
}

// ReenlistIntent shows, beside continuing, the survival throw the next term
// opens with. Its DM is the characteristics' before this term's aging round,
// which comes after the answer and can lower them, and the line says so.
//
// Implements: A throw's question shows its target and DM before the answer
// [Choices].
func (p *player) ReenlistIntent(from []traveller.Intent, survival traveller.Odds) (traveller.Intent, error) {
	return pick(p, "The term is over. What now? (pp. 6-7, 21)", from, func(intent traveller.Intent) string {
		if intent == traveller.Continue {
			return fmt.Sprintf("continue (next term's survival %v, before any aging)", oddsText(survival))
		}

		return intent.String()
	})
}

func (p *player) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	return pick(p, "Which mustering out table? It is designated before the die (p. 9)",
		from, traveller.MusterTable.String)
}

func (p *player) MusterTable1DM() (bool, error) {
	return p.confirm("Your rank allows +1 on this table. Take it? (p. 9)")
}

// Implements: The gambling modifier is offered only to one who has the
// expertise [1:9].
func (p *player) MusterTable2DM() (bool, error) {
	return p.confirm("Your gambling expertise allows +1 on this table. Take it? (p. 9)")
}

// weaponOffer pairs one entry of the p. 22 weapon row with what taking it
// means, so that the wording and the benefit are built in one place and
// nothing has to work out afterwards which list an index fell into.
type weaponOffer struct {
	text  string
	means traveller.WeaponBenefit
}

// MusterWeapon offers a weapon of the category, or expertise in one already
// received, each line carrying the benefit it names.
//
// Implements: Each weapon-benefit line records the benefit it says [Choices].
func (p *player) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	offers := make([]weaponOffer, 0, len(from)+len(received))
	for _, weapon := range from {
		offers = append(offers, weaponOffer{
			text: string(weapon), means: traveller.TakeWeapon{Weapon: weapon},
		})
	}

	// P. 22 offers the expertise only in lieu of a weapon "of exactly the
	// same type", so it appears only for one already received.
	//
	// The wording here is the menu's, not the record's: a person choosing
	// from a list reads "+1 expertise in the Dagger" better than the
	// record's "expertise in Dagger". They are deliberately two, because
	// they address two readers, and nothing compares them - the engine's
	// gate folds the benefit and compares that.
	for _, held := range received {
		offers = append(offers, weaponOffer{
			text: "+1 expertise in the " + string(held), means: traveller.TakeExpertise{Weapon: held},
		})
	}

	question := fmt.Sprintf("%v: take a weapon, or expertise in one already received (p. 22)", category)

	chosen, err := pick(p, question, offers, func(o weaponOffer) string { return o.text })

	return chosen.means, err
}

func (p *player) AssumeTitle(title traveller.Title) (bool, error) {
	return p.confirm(fmt.Sprintf(
		"Your social standing makes you %v. Assume the title? (p. 5; Book 3 p. 22)", title,
	))
}

// sayf writes to the player, remembering the first failure.
//
// Implements: A player nobody can be shown anything is not asked [Choices].
func (p *player) sayf(format string, args ...any) {
	if p.wrote != nil {
		return
	}

	_, p.wrote = fmt.Fprintf(p.out, format, args...)
}

// watch shows the player the procedure between his questions.
//
// Every event renders to something, so nothing is skipped here and the
// numbering has no holes: writeEvent has a case for every kind, which the
// exhaustive linter holds.
func (p *player) watch(event traveller.Event) {
	p.sayf("%s", render.EventLine(event))
}

// choose asks for one of a numbered list and returns its index.
//
// The list is printed once. A bad answer gets a complaint naming what was
// typed, and the prompt comes back on its own: re-printing the whole list
// would bury the complaint under six lines of menu and read as though the tool
// had said nothing at all.
//
// The prompt carries the range it accepts, so the question says how to
// answer it before any answer is refused (#461). A refusal says what it
// wants; an empty line has nothing to quote back, and gets the nudge alone.
//
// Implements: The alternatives offered are exactly the engine's [Choices].
// Implements: An unreadable answer is asked again, never guessed at [Choices].
// Implements: A question shows the numbers it accepts before it is answered
// [Choices].
func (p *player) choose(question string, options []string) (int, error) {
	p.sayf("\n%s\n", question)

	for i, option := range options {
		p.sayf("  %d) %s\n", i+1, option)
	}

	nudge := fmt.Sprintf("answer with one of the numbers shown, 1-%d", len(options))

	for {
		p.sayf("answer 1-%d> ", len(options))

		if p.wrote != nil {
			return 0, fmt.Errorf("asking the player: %w", p.wrote)
		}

		if !p.in.Scan() {
			return 0, p.stopped()
		}

		typed := strings.TrimSpace(p.in.Text())

		chosen, err := strconv.Atoi(typed)
		if err == nil && chosen >= 1 && chosen <= len(options) {
			p.sayf("\n")

			p.given = append(p.given, chosen)

			return chosen - 1, nil
		}

		if typed == "" {
			p.sayf("  %s\n", nudge)

			continue
		}

		p.sayf("  %q is refused; %s\n", typed, nudge)
	}
}

// stopped reports why the answers ran out, and says what was lost.
//
// A scanner returns false for two different reasons and only one of them is
// the input ending: a line past bufio's 64KB bound, or a read failure on the
// reader beneath, both come back the same way. Blaming the input for ending
// sends the reader looking in the wrong place.
//
// Either way the half-built character is lost: it has no note, nothing
// carries its dice, and there is no way back to the question it stopped on
// (ADR 0001). The message says so, and lists what was answered, before the
// two are told apart: a read that failed is the stop the operator did not
// choose, and the one that most needs telling what it cost.
//
// Implements: A stopped session says what was answered and that nothing was
// written [ADR 0001].
// Implements: A read that fails is not the input ending [Choices].
func (p *player) stopped() error {
	p.sayf("\n%s\n", p.lostLine())

	err := p.in.Err()
	if err != nil {
		return fmt.Errorf("reading the answer: %w", err)
	}

	return errNoAnswer
}

// lostLine says what the stopped session answered and that none of it was
// kept. It offers no way back in: the tool has withdrawn replay (ADR 0001),
// and a half-built character is not a record.
func (p *player) lostLine() string {
	if len(p.given) == 0 {
		return "Nothing was answered; the session is lost and nothing was written."
	}

	answered := make([]string, 0, len(p.given))
	for _, chosen := range p.given {
		answered = append(answered, strconv.Itoa(chosen))
	}

	return fmt.Sprintf("Answered %s; the session is lost and nothing was written.",
		strings.Join(answered, ", "))
}

// confirm asks a question the procedure puts as yes or no.
func (p *player) confirm(question string) (bool, error) {
	chosen, err := p.choose(question, []string{"yes", "no"})

	return chosen == 0, err
}
