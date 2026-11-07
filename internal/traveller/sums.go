package traveller

// The sums: the places where "exactly one of" is the rule.
//
// Each is an interface with a Fold method taking a cases interface that has
// one method per case, and an unexported seal so the set cannot be extended
// from outside this package. Adding a case adds a method to the cases
// interface, and every implementation stops compiling until it handles it.
//
// This is the same mechanism the Decider uses for choice points, and it is
// here for the same reason: Go's compiler does not check that a type switch
// covers every implementation of an interface, and neither does the
// exhaustive linter. A type switch over a sealed interface is an
// exhaustiveness promise that nothing keeps.

// Enlistment is how — or whether — a character entered a service (p. 5).
//
// The civilian who declined the draft is a case of the type, not a nil
// service with a flag beside it. The draft is declinable per E001: p. 5
// prints both "he may submit to the draft" and "the character must submit to
// the draft", and the may reading governs.
type Enlistment interface {
	Fold(cases EnlistmentCases) error
	sealedEnlistment()
}

// EnlistmentCases handles each way p. 5's enlistment step can end.
type EnlistmentCases interface {
	Enlisted(service ServiceName) error
	Drafted(service ServiceName) error
	DeclinedTheDraft() error
}

// Enlisted is a successful enlistment throw against the Prior Service Table
// (pp. 5, 10).
type Enlisted struct{ Service ServiceName }

// Drafted is entry by the draft: one die, and the service with that draft
// number — "possibly the very service which had just previously rejected his
// enlistment" (p. 5).
type Drafted struct{ Service ServiceName }

// DeclinedTheDraft ends generation with an eighteen-year-old civilian: no
// service, no terms, no skills, no benefits. A complete record (E001).
type DeclinedTheDraft struct{}

func (e Enlisted) Fold(c EnlistmentCases) error       { return c.Enlisted(e.Service) }
func (d Drafted) Fold(c EnlistmentCases) error        { return c.Drafted(d.Service) }
func (DeclinedTheDraft) Fold(c EnlistmentCases) error { return c.DeclinedTheDraft() }
func (Enlisted) sealedEnlistment()                    {}
func (Drafted) sealedEnlistment()                     {}
func (DeclinedTheDraft) sealedEnlistment()            {}

// Fatal reports a departure that ended the character's life.
//
// It lives here because the cases live here. It was stated twice - once in the
// command, to decide what --survivors passes over, and once in the renderer,
// to set the record's fatal flag - and both are folds, so adding a case broke
// both at compile time. Changing which case is a death would have broken
// neither, and the two would have disagreed with nothing to say so (#100).
//
// A civilian never joined a service and so never left one: nil is not a death.
//
// Implements: Which departures are deaths is said in one place [Domain model]
// in Mustering out.
// Implements: Only the two deaths count as dying [1:5, 1:7–8] in The record.
func Fatal(departure Departure) bool {
	if departure == nil {
		return false
	}

	var died fatalDeparture

	// A fold over a codec cannot fail: every case only assigns.
	_ = departure.Fold(&died)

	return died.fatal
}

// fatalDeparture is Fatal's fold. The two Killed cases are the deaths; the
// three others are ways of leaving alive.
type fatalDeparture struct{ fatal bool }

func (*fatalDeparture) Discharged() error { return nil }
func (*fatalDeparture) ForcedOut() error  { return nil }
func (*fatalDeparture) Retired() error    { return nil }

func (f *fatalDeparture) KilledBySurvivalThrow() error {
	f.fatal = true

	return nil
}

func (f *fatalDeparture) KilledByMedicalCrisis(Characteristic) error {
	f.fatal = true

	return nil
}

// TableResult is one row of an Acquired Skills table (pp. 11-12). P. 12
// names exactly three kinds: "Skills are of three basic types:
// characteristic alterations (such as +1 Strength), weapon expertise (such
// as Blade Combat), and basic skill (such as Navigation)."
type TableResult interface {
	Fold(cases TableResultCases) error
	sealedTableResult()
}

// TableResultCases handles each of p. 12's three kinds.
type TableResultCases interface {
	Alteration(characteristic Characteristic, delta int) error
	Skill(name SkillName) error
	WeaponPick(category WeaponCategory) error
}

// AlterationResult is a characteristic alteration, "applied immediately,
// increasing or decreasing the character's current ability" (p. 12).
type AlterationResult struct {
	Characteristic Characteristic
	Delta          int
}

// SkillResult is a basic skill, accumulating as Skill-1, Skill-2 with no cap
// (p. 12).
type SkillResult struct{ Name SkillName }

// WeaponPickResult is Blade Combat or Gun Combat, which demand the specific
// weapon at once (p. 11).
type WeaponPickResult struct{ Category WeaponCategory }

func (a AlterationResult) Fold(c TableResultCases) error {
	return c.Alteration(a.Characteristic, a.Delta)
}
func (s SkillResult) Fold(c TableResultCases) error      { return c.Skill(s.Name) }
func (w WeaponPickResult) Fold(c TableResultCases) error { return c.WeaponPick(w.Category) }
func (AlterationResult) sealedTableResult()              {}
func (SkillResult) sealedTableResult()                   {}
func (WeaponPickResult) sealedTableResult()              {}

// BenefitRow is one row of the two Mustering Out Tables (p. 9). Cash is
// Table 2's whole content; Table 1 prints no money row, and does print rows
// that are a bare dash.
type BenefitRow interface {
	Fold(cases BenefitRowCases) error
	sealedBenefitRow()
}

// BenefitRowCases handles each of the seven things a mustering out row can
// be (p. 9, with the definitions at pp. 21-23).
type BenefitRowCases interface {
	Cash(amount Credits) error
	Passage(class PassageClass) error
	Alteration(characteristic Characteristic, delta int) error
	WeaponPick(category WeaponCategory) error
	TravellersAid() error
	Ship(kind ShipKind) error
	Nothing() error
}

// CashBenefit is a Table 2 row (p. 9).
type CashBenefit struct{ Amount Credits }

// PassageBenefit is a High, Middle or Low Passage row (p. 9; pp. 21-22).
type PassageBenefit struct{ Class PassageClass }

// AlterationBenefit is a +1 or +2 characteristic row, "applied to the
// character immediately" (p. 9; p. 23).
type AlterationBenefit struct {
	Characteristic Characteristic
	Delta          int
}

// WeaponCategoryBenefit is a Blade or Gun row. P. 9: "Weapon benefits must
// be declared as to type immediately; additional benefits of that type may
// be declared as skill."
type WeaponCategoryBenefit struct{ Category WeaponCategory }

// TravellersAidBenefit is Travellers' Aid Society membership, achievable
// "only once per character" — a duplicate roll is wasted and "the character
// receives nothing for it" (p. 22).
type TravellersAidBenefit struct{}

// ShipBenefit is a Scout ship or a Free Trader (pp. 22-23).
type ShipBenefit struct{ Kind ShipKind }

// NoBenefit is one of Table 1's dash rows (p. 9). Those cells are the font
// trap: under text extraction they come out as the digit 4, which would give
// a Scout a seventh benefit he does not have.
type NoBenefit struct{}

func (b CashBenefit) Fold(c BenefitRowCases) error    { return c.Cash(b.Amount) }
func (b PassageBenefit) Fold(c BenefitRowCases) error { return c.Passage(b.Class) }
func (b AlterationBenefit) Fold(c BenefitRowCases) error {
	return c.Alteration(b.Characteristic, b.Delta)
}
func (b WeaponCategoryBenefit) Fold(c BenefitRowCases) error { return c.WeaponPick(b.Category) }
func (TravellersAidBenefit) Fold(c BenefitRowCases) error    { return c.TravellersAid() }
func (b ShipBenefit) Fold(c BenefitRowCases) error           { return c.Ship(b.Kind) }
func (NoBenefit) Fold(c BenefitRowCases) error               { return c.Nothing() }
func (CashBenefit) sealedBenefitRow()                        {}
func (PassageBenefit) sealedBenefitRow()                     {}
func (AlterationBenefit) sealedBenefitRow()                  {}
func (WeaponCategoryBenefit) sealedBenefitRow()              {}
func (TravellersAidBenefit) sealedBenefitRow()               {}
func (ShipBenefit) sealedBenefitRow()                        {}
func (NoBenefit) sealedBenefitRow()                          {}

// Departure is how a character's service ended. Every case but the two
// deaths leads to mustering out (p. 7).
type Departure interface {
	Fold(cases DepartureCases) error
	sealedDeparture()
}

// DepartureCases handles each way service ends.
//
// The book's own word is "died"; it becomes two cases here because the two
// deaths do not carry the same fields. A survival failure carries nothing
// but the term (p. 5); a medical crisis carries the characteristic that
// reached zero (pp. 7-8, and E008 for the reading that a failed crisis throw
// is death at all).
type DepartureCases interface {
	Discharged() error
	ForcedOut() error
	Retired() error
	KilledBySurvivalThrow() error
	KilledByMedicalCrisis(characteristic Characteristic) error
}

// Discharged is leaving by choice before the fifth term ends (pp. 6-7).
type Discharged struct{}

// ForcedOut is a failed reenlistment throw: "reenlistment has been denied,
// and the person must leave the service" (p. 6).
type ForcedOut struct{}

// Retired is leaving at the end of the fifth term or later, which p. 21
// makes retirement by definition: "A character who leaves the service at the
// end of the 5th or later term of service is considered to have retired".
type Retired struct{}

// KilledBySurvivalThrow is a failed survival throw. P. 5: "Failure to
// successfully achieve the survival throw results in death."
type KilledBySurvivalThrow struct{}

// KilledByMedicalCrisis is a failed 8+ saving throw after a characteristic
// reached zero (pp. 7-8, E008).
type KilledByMedicalCrisis struct{ Characteristic Characteristic }

func (Discharged) Fold(c DepartureCases) error            { return c.Discharged() }
func (ForcedOut) Fold(c DepartureCases) error             { return c.ForcedOut() }
func (Retired) Fold(c DepartureCases) error               { return c.Retired() }
func (KilledBySurvivalThrow) Fold(c DepartureCases) error { return c.KilledBySurvivalThrow() }
func (k KilledByMedicalCrisis) Fold(c DepartureCases) error {
	return c.KilledByMedicalCrisis(k.Characteristic)
}
func (Discharged) sealedDeparture()            {}
func (ForcedOut) sealedDeparture()             {}
func (Retired) sealedDeparture()               {}
func (KilledBySurvivalThrow) sealedDeparture() {}
func (KilledByMedicalCrisis) sealedDeparture() {}

// WeaponBenefit is what a character takes when a Table 1 weapon row comes up
// again (p. 22): "the character may choose the same weapon again, or a
// different weapon. He may also elect to take +1 expertise in lieu of
// receiving a second or subsequent weapon of exactly the same type."
type WeaponBenefit interface {
	Fold(cases WeaponBenefitCases) error
	sealedWeaponBenefit()
}

// WeaponBenefitCases handles the two things a weapon row can be taken as.
type WeaponBenefitCases interface {
	TakeWeapon(weapon WeaponName) error
	TakeExpertise(weapon WeaponName) error
}

// TakeWeapon receives the weapon itself — a first one, a different one, or
// another of the same (p. 22).
type TakeWeapon struct{ Weapon WeaponName }

// TakeExpertise takes +1 expertise instead. P. 22 bounds it twice: only "in
// a weapon received as a benefit", and only "in lieu of receiving a second
// or subsequent weapon of exactly the same type", which keeps it inside the
// benefit's own category.
type TakeExpertise struct{ Weapon WeaponName }

func (t TakeWeapon) Fold(c WeaponBenefitCases) error    { return c.TakeWeapon(t.Weapon) }
func (t TakeExpertise) Fold(c WeaponBenefitCases) error { return c.TakeExpertise(t.Weapon) }
func (TakeWeapon) sealedWeaponBenefit()                 {}
func (TakeExpertise) sealedWeaponBenefit()              {}
