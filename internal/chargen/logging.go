package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// logging wraps a Decider, writes each answer into the generation record,
// and delegates.
//
// It keeps the offer's two rules, so that no asking site has to (#278). Every
// slice a decider is handed is its own copy, made after the offer is named:
// a decider may write to what it is given, and the copy is what keeps that
// write from reaching anything the engine offers afterwards - an exported
// table, the shared ruleset, or the next question's offer (#55, #137). And a
// question with one legal answer is refused before it is asked.
//
// The copies guard every site, including the ones that build their slices
// fresh for each question. Where a site does, no test can see the copy, and
// a mutation removing it survives; it is there for the site that does not.
//
// Implements: An offer is the decider's own, and writing to it changes nothing
// the engine offers [Choice points] in The record.
//
// It must implement every method of Decider to compile, which is the whole
// point of the shape: a choice point cannot reach the engine without
// reaching the log. Adding a method to Decider breaks this type until the
// log learns to record it.
//
// Implements: Each choice names what was offered and what was chosen
// [Generation record] in The record.
type logging struct {
	to  Decider
	log *log
}

func names[T any](from []T, name func(T) string) []string {
	out := make([]string, 0, len(from))
	for _, v := range from {
		out = append(out, name(v))
	}

	return out
}

// expertiseIn names the p. 22 alternative, in one place, so that the record
// of what was offered and the record of what was chosen cannot word it
// differently.
func expertiseIn(weapon traveller.WeaponName) string {
	return "expertise in " + string(weapon)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}

	return "no"
}

// offered is what a yes-or-no choice point puts in front of the decider,
// built fresh for each question: the slice goes into the event uncopied, so
// a shared one would let a write through any record or watcher change what
// every later question is offered and checked against (#230).
func offered() []string {
	return []string{"yes", "no"}
}

func (l logging) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	offer := names(from, func(o traveller.EnlistmentOffer) string { return o.Service.String() })

	err := asked(traveller.ChoiceService, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.Service(own(from))
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceService, offer, chosen.String())
}

func (l logging) SubmitToDraft() (bool, error) {
	chosen, err := l.to.SubmitToDraft()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSubmitToDraft, offered(), yesNo(chosen))
}

func (l logging) AttemptCommission() (bool, error) {
	chosen, err := l.to.AttemptCommission()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAttemptCommission, offered(), yesNo(chosen))
}

func (l logging) AttemptPromotion() (bool, error) {
	chosen, err := l.to.AttemptPromotion()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAttemptPromotion, offered(), yesNo(chosen))
}

func (l logging) SkillTable(
	from []traveller.SkillTable, taken []int,
) (traveller.SkillTable, error) {
	offer := names(from, traveller.SkillTable.String)

	err := asked(traveller.ChoiceSkillTable, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.SkillTable(own(from), own(taken))
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSkillTable, offer, chosen.String())
}

func (l logging) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, vary Vary,
) (traveller.WeaponName, error) {
	offer := names(from, func(w traveller.WeaponName) string { return string(w) })

	err := asked(traveller.ChoiceWeapon, offer)
	if err != nil {
		return "", err
	}

	chosen, err := l.to.Weapon(category, own(from), vary)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceWeapon, offer, string(chosen))
}

func (l logging) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	offer := names(from, traveller.Intent.String)

	err := asked(traveller.ChoiceReenlistIntent, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.ReenlistIntent(own(from))
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceReenlistIntent, offer, chosen.String())
}

func (l logging) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	offer := names(from, traveller.MusterTable.String)

	err := asked(traveller.ChoiceMusterTable, offer)
	if err != nil {
		return 0, err
	}

	chosen, err := l.to.MusterTable(own(from))
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable, offer, chosen.String())
}

func (l logging) MusterTable1DM() (bool, error) {
	chosen, err := l.to.MusterTable1DM()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable1DM, offered(), yesNo(chosen))
}

func (l logging) MusterTable2DM() (bool, error) {
	chosen, err := l.to.MusterTable2DM()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable2DM, offered(), yesNo(chosen))
}

func (l logging) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	// P. 22 offers the expertise beside the weapons, so the record of what
	// was offered says so - and the gate has the whole offer to check
	// against. Both halves are worded by the same functions that word the
	// answer, so the comparison is like with like.
	alternatives := names(from, func(w traveller.WeaponName) string { return string(w) })
	for _, held := range received {
		alternatives = append(alternatives, expertiseIn(held))
	}

	err := asked(traveller.ChoiceMusterWeapon, alternatives)
	if err != nil {
		return nil, err
	}

	chosen, err := l.to.MusterWeapon(category, own(from), own(received))
	if err != nil {
		return chosen, err
	}

	var described describeWeaponBenefit

	err = chosen.Fold(&described)
	if err != nil {
		return chosen, fmt.Errorf("describing the weapon benefit: %w", err)
	}

	return chosen, l.record(traveller.ChoiceMusterWeapon, alternatives, described.text)
}

func (l logging) AssumeTitle(title traveller.Title) (bool, error) {
	chosen, err := l.to.AssumeTitle(title)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAssumeTitle, offered(), yesNo(chosen))
}

// own is a copy of a slice for the decider to keep: an offer, or what it is
// handed beside one. It is made rather than taken from slices.Clone, which
// hands back nil for nil, so a decider indexing what it is given is handed a
// slice and never a nil one.
func own[T any](from []T) []T {
	out := make([]T, len(from))
	copy(out, from)

	return out
}

// aChoice is the fewest alternatives that make a question a choice: more
// than one answer legal, as the Decider contract puts it.
const aChoice = 2

// asked refuses a question whose offer holds fewer than two alternatives,
// before the decider sees it. The six points that offer a list are the ones
// that can come to hold one; a yes-or-no point offers two by construction.
//
// It refuses rather than settling the question here. Each site that can reach
// one answer settles it itself, as a reading of the page - reenlistment at
// the seventh term (p. 7), the third roll on Table 2 (p. 9) - and a site that
// forgot would be an engine bug the record should not quietly absorb.
//
// Implements: A question with one legal answer is never put to a decider
// [Choice points] in The record.
func asked(point traveller.ChoicePoint, alternatives []string) error {
	if len(alternatives) < aChoice {
		return fmt.Errorf("%w: %v offered only %v; the engine asks only when more than one answer is legal",
			errOneAnswer, point, alternatives)
	}

	return nil
}

// describeWeaponBenefit renders what was taken for a weapon row, by folding,
// so a third way of taking one cannot be added without this seeing it.
type describeWeaponBenefit struct{ text string }

func (d *describeWeaponBenefit) TakeWeapon(weapon traveller.WeaponName) error {
	d.text = string(weapon)

	return nil
}

func (d *describeWeaponBenefit) TakeExpertise(weapon traveller.WeaponName) error {
	d.text = expertiseIn(weapon)

	return nil
}

// record writes the answer into the generation record, and refuses one that
// was not offered.
//
// It is the single gate: the decorator is the only way a decider reaches the
// engine (generate.go wraps every one of them), and it already holds both
// the offered set and the answer, because it records both. Each method names
// the offer before it asks, so what is checked is what the decider was
// shown: a decider can write to the slice it is handed, and an offer named
// afterwards would let it put its own answer there and pass. Checking here
// rather than at each application site leaves one definition of answering
// outside the offer, which two cannot disagree with.
//
// It compares rendered names rather than values, and every method above has
// the typed offered set and the typed answer in hand before rendering. That
// is deliberate: MusterWeapon answers with a sum, and comparing a sum against
// a list of them needs a fold, so one path here would have to be strings
// whatever the other eleven did.
//
// What it costs is an assumption. The check is sound exactly as long as
// String() is injective on every alphabet the engine offers - two values that
// spelled the same would let a decider answer with one and have the other
// recorded, and nothing would notice. It holds, and
// traveller.TestNoTwoValuesOfAnAlphabetShareASpelling holds it (#48).
//
// Implements: An answer outside the offered set is refused, not applied
// [Choice points] in The record.
func (l logging) record(point traveller.ChoicePoint, from []string, chosen string) error {
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.log.choice(point, from, chosen)

	return nil
}
