package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// logging wraps a Decider, writes each answer into the generation record,
// and delegates.
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

	chosen, err := l.to.Service(from)
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

	chosen, err := l.to.SkillTable(from, taken)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSkillTable, offer, chosen.String())
}

func (l logging) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, vary Vary,
) (traveller.WeaponName, error) {
	offer := names(from, func(w traveller.WeaponName) string { return string(w) })

	chosen, err := l.to.Weapon(category, from, vary)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceWeapon, offer, string(chosen))
}

func (l logging) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	offer := names(from, traveller.Intent.String)

	chosen, err := l.to.ReenlistIntent(from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceReenlistIntent, offer, chosen.String())
}

func (l logging) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	offer := names(from, traveller.MusterTable.String)

	chosen, err := l.to.MusterTable(from)
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

	chosen, err := l.to.MusterWeapon(category, from, received)
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
