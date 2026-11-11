// Package chargen walks Book 1's character generation procedure, pp. 4-25.
//
// It consumes a Decider for every point at which the procedure asks, and a
// roller for every point at which it throws. Nothing here decides on its own
// behalf: interactive mode, auto mode, and the tests each supply a Decider,
// and the engine cannot tell them apart.
package chargen

import "github.com/philoserf/ctchargen/internal/traveller"

// Decider answers every question the procedure asks.
//
// There is one method per choice point, each typed to its own alphabet, and
// that is what closes the set: adding a choice point breaks every
// implementation at compile time. Spelling it as one method over a label
// would lose exactly that. The interface is as wide as the places the
// procedure asks, and a count of its methods is the wrong measure of it.
//
// Each method says what it asks and what it answers with. When the engine
// asks it is said once, on the Policy method of the same name, beside what
// each --auto strategy answers.
//
// A method is called only when more than one answer is legal. A question
// with one answer is not a choice, and asking it would put an entry in the
// generation record that no reader could have decided differently. The gate
// every decider is reached through refuses such a question, and hands each
// decider its own copy of what it is offered.
//
//nolint:interfacebloat // one method per choice point closes the set
type Decider interface {
	// Service chooses which service to attempt enlistment in (pp. 5, 10).
	Service(from []traveller.EnlistmentOffer, vary Vary) (traveller.ServiceName, error)

	// SubmitToDraft asks whether to submit to the draft (p. 5; E001).
	SubmitToDraft() (bool, error)

	// AttemptCommission asks whether to try for a commission this term
	// (p. 6). odds is the commission throw that answering yes leads to.
	AttemptCommission(odds traveller.Odds) (bool, error)

	// AttemptPromotion asks whether to try for a promotion this term
	// (p. 6). odds is the promotion throw that answering yes leads to.
	AttemptPromotion(odds traveller.Odds) (bool, error)

	// SkillTable designates the skills table to consult (p. 11). Each offer
	// carries its table, the results already taken off it and its six
	// results. roll is which of the term's skills rolls this one is, and
	// what gave the term its rolls; it is not recorded.
	SkillTable(from []traveller.SkillTableOffer, roll traveller.SkillRoll) (traveller.SkillTable, error)

	// Weapon names the specific weapon for a Blade Combat or Gun Combat
	// result (pp. 11-13).
	//
	// It is handed a Vary, as Service is, because the book leaves the choice
	// to a player and gives no basis for it: the lists of pp. 12-13 are
	// printed in an order, and the order is not a preference. A decider that
	// wants the first name may ignore it (#34).
	Weapon(
		category traveller.WeaponCategory,
		from []traveller.WeaponName,
		vary Vary,
	) (traveller.WeaponName, error)

	// ReenlistIntent asks what the character means to do at the end of a
	// term (pp. 6-7, 21). survival is the throw the next term opens with,
	// which continuing leads to, with the DM the characteristics earn before
	// this term's aging round.
	ReenlistIntent(from []traveller.Intent, survival traveller.Odds) (traveller.Intent, error)

	// MusterTable designates the mustering out table for one benefit roll
	// (p. 9).
	MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error)

	// MusterTable1DM asks whether to add the +1 a rank of 5 or 6 allows on
	// Table 1 (p. 9).
	MusterTable1DM() (bool, error)

	// MusterTable2DM asks whether to add the +1 gambling expertise allows
	// on Table 2 (p. 9).
	MusterTable2DM() (bool, error)

	// MusterWeapon takes a weapon row as a weapon or as expertise (p. 22).
	// received is the weapons of this category already taken as benefits,
	// which is narrower than the weapons held: expertise may be taken only
	// "in a weapon received as a benefit", and only in lieu of a weapon
	// "of exactly the same type".
	//
	// It is handed a Vary for the reason Weapon is: the list a weapon is
	// taken from is printed in an order, and the order is not a preference
	// (#570).
	MusterWeapon(
		category traveller.WeaponCategory,
		from []traveller.WeaponName,
		received []traveller.WeaponName,
		vary Vary,
	) (traveller.WeaponBenefit, error)

	// AssumeTitle asks whether to assume the title a final Social Standing
	// of 11 or greater confers (p. 5; Book 3 p. 22).
	AssumeTitle(title traveller.Title) (bool, error)
}
