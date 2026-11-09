package rules

import (
	"fmt"
	"maps"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// load lifts every table. It runs once, on first use, so a transcription
// that will not lift stops the program where the defect is rather than where
// a path happens to reach it.
func load() (*Rules, error) {
	rules := &Rules{}

	rules.normalize = maps.Clone(transcribedSkills.normalizations)

	for _, lift := range []func() error{
		func() error { return rules.liftPriorService(transcribedServices) },
		func() error { return rules.liftGrants(transcribedServices) },
		func() error { return rules.liftRetirement(transcribedServices) },
		func() error { return rules.liftSkills(transcribedSkills) },
		func() error { return rules.liftMustering(transcribedMustering) },
		func() error { return rules.liftAging(transcribedAging) },
		func() error { return rules.liftWeapons(transcribedWeapons) },
		func() error { return rules.liftNobility(transcribedNobility) },
		func() error { return rules.liftShips(transcribedShips) },
	} {
		err := lift()
		if err != nil {
			return nil, err
		}
	}

	return rules, nil
}

// liftPriorService lifts the Prior Service Table and the Table of Ranks (p. 10),
// a service at a time in traveller's order.
func (r *Rules) liftPriorService(t servicesTranscription) error {
	err := checkDraft(t.draft)
	if err != nil {
		return err
	}

	for i, name := range traveller.ServiceNames {
		service, err := liftServiceColumn(t, i, name)
		if err != nil {
			return err
		}

		r.services[i] = service
	}

	return nil
}

// checkDraft holds the draft row to what Draft needs of it (p. 5): one die
// face per service, each face printed once. Six distinct faces over six
// services is every face, so a draft die always names a service.
func checkDraft(draft [len(traveller.ServiceNames)]int) error {
	seen := make(map[int]bool, len(draft))

	for i, face := range draft {
		if face < 1 || face > Faces {
			return fmt.Errorf("%w: prior service table, draft, %v: %d is not a die face",
				errMalformed, traveller.ServiceNames[i], face)
		}

		if seen[face] {
			return fmt.Errorf("%w: prior service table, draft: %d is printed twice",
				errMalformed, face)
		}

		seen[face] = true
	}

	return nil
}

// liftServiceColumn lifts one column of the Prior Service Table and the
// Table of Ranks beside it (p. 10).
//
// Implements: A table broken two ways reports the same one every run [JSON
// conventions].
func liftServiceColumn(t servicesTranscription, i int, name traveller.ServiceName) (Service, error) {
	service := Service{Name: name, Draft: t.draft[i], Ranks: t.ranks[i]}

	var err error

	service.Enlistment, err = liftThrow(t.enlistment[i], "enlistment")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.Survival, err = liftThrow(t.survival[i], "survival")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.Reenlist, err = traveller.ParseTarget(t.reenlist[i])
	if err != nil {
		return Service{}, fmt.Errorf("%w: %v reenlist: %w", errMalformed, name, err)
	}

	// P. 10 makes ranks, commissions and promotions one fact: "Ranks,
	// commissions, and promotions are non-existent in the scout and other
	// services." A column that prints one without the others is malformed.
	hasRanks := len(service.Ranks) > 0

	// A slice and not a map: with both columns malformed, a range over a map
	// names an arbitrary one of them, so the same broken transcription reports a
	// different problem run to run and a reader cannot tell whether he fixed
	// anything (#53). The order is the page's.
	for _, printed := range []struct {
		label string
		cell  *printedThrow
	}{
		{"commission", t.commission[i]},
		{"promotion", t.promotion[i]},
	} {
		label, cell := printed.label, printed.cell

		if (cell != nil) != hasRanks {
			return Service{}, fmt.Errorf(
				"%w: %v prints %s but %s ranks: p. 10 makes ranks, commissions and promotions one fact",
				errMalformed, name, label, map[bool]string{true: "also", false: "no"}[hasRanks],
			)
		}
	}

	if !hasRanks {
		return service, nil
	}

	// Both are printed here: the check above refused a column with ranks
	// that leaves either out.
	service.commission, err = liftThrow(*t.commission[i], "commission")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.promotion, err = liftThrow(*t.promotion[i], "promotion")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	return service, nil
}

func liftThrow(printed printedThrow, what string) (Throw, error) {
	target, err := traveller.ParseTarget(printed.target)
	if err != nil {
		return Throw{}, fmt.Errorf("%w: %s: %w", errMalformed, what, err)
	}

	throw := Throw{Target: target}

	for _, dm := range printed.dms {
		lifted, err := parseDM(dm.dm, dm.condition)
		if err != nil {
			return Throw{}, fmt.Errorf("%s: %w", what, err)
		}

		throw.DMs = append(throw.DMs, lifted)
	}

	return throw, nil
}

func (r *Rules) liftGrants(t servicesTranscription) error {
	for _, g := range t.grants {
		service := g.service

		result, err := parseTableResult(g.grant, r.normalize)
		if err != nil {
			return fmt.Errorf("rank and service skills, %v: %w", service, err)
		}

		if g.rank < 0 || traveller.Rank(g.rank) > r.services[service].MaxRank() {
			return fmt.Errorf("%w: rank and service skills, %v: rank %d is past the service's table of ranks", errMalformed,
				service, g.rank)
		}

		r.grants = append(r.grants, Grant{Service: service, Rank: traveller.Rank(g.rank), Result: result})
	}

	return nil
}

func (r *Rules) liftRetirement(t servicesTranscription) error {
	if len(t.pensions) == 0 {
		return fmt.Errorf("%w: retirement pay: the table prints no rows", errMalformed)
	}

	r.retirement.byTerms = make(map[int]traveller.Credits, len(t.pensions))
	for _, row := range t.pensions {
		if row.pay <= 0 {
			return fmt.Errorf("%w: retirement pay at %d terms: %d is not a pension", errMalformed, row.terms, row.pay)
		}

		if row.terms <= r.retirement.lastTabled {
			return fmt.Errorf("%w: retirement pay: %d terms does not follow %d; the table's rows ascend", errMalformed,
				row.terms, r.retirement.lastTabled)
		}

		r.retirement.byTerms[row.terms] = traveller.Credits(row.pay)
		r.retirement.lastTabled = row.terms
	}

	if t.perAdditionalTerm <= 0 {
		return fmt.Errorf("%w: retirement pay: %d per additional term past the table", errMalformed,
			t.perAdditionalTerm)
	}

	r.retirement.PerAdditionalTerm = traveller.Credits(t.perAdditionalTerm)

	if len(t.pensionPaidBy) == 0 {
		return fmt.Errorf("%w: retirement pay: no service pays it", errMalformed)
	}

	for _, service := range t.pensionPaidBy {
		r.services[service].PaysPension = true
	}

	return nil
}

// liftSkills lifts the Acquired Skills transcription: the four tables, the
// eligibility and the education gate. The tables are lifted in the order p. 11
// prints them and each row's cells in traveller's order of services, so a
// transcription broken two ways names the same fault every run.
//
// Implements: A table broken two ways reports the same one every run [JSON
// conventions].
func (r *Rules) liftSkills(t skillsTranscription) error {
	for _, table := range traveller.SkillTables {
		for die, row := range t.tables[table] {
			for i, cell := range row {
				service := traveller.ServiceNames[i]

				result, err := parseTableResult(cell, r.normalize)
				if err != nil {
					return fmt.Errorf("acquired skills, %v, row %d, %v: %w", table, die+1, service, err)
				}

				r.services[service].Skills[table][die] = result
			}
		}
	}

	r.eligibility = t.eligibility

	characteristic, err := parseCharacteristic(t.educationGate.characteristic)
	if err != nil {
		return fmt.Errorf("education gate: %w", err)
	}

	threshold, err := traveller.ParseTarget(t.educationGate.threshold)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	r.education = Gate{Table: t.educationGate.table, Characteristic: characteristic, Threshold: threshold}

	return nil
}

func (r *Rules) liftMustering(t musteringTranscription) error {
	err := r.liftBenefits(t)
	if err != nil {
		return err
	}

	err = r.liftCash(t)
	if err != nil {
		return err
	}

	// The skill that earns the Table 2 DM goes through E012's normalization
	// on the way in, exactly as a table cell does. The engine compares it
	// against skills the character holds, and those were spelled by
	// parseTableResult through the same map - a name lifted raw here would
	// stop matching the moment a normalization for it was added, and the
	// modifier would silently never apply.
	earnedBy := r.Normalize(t.table2ModifierFrom)
	if earnedBy == "" {
		return fmt.Errorf("%w: mustering out: no skill earns the table 2 modifier", errMalformed)
	}

	r.muster = Muster{
		Table1DMFromRank5or6: t.table1DMFromRank5or6,
		Table2DMFromGambling: t.table2DMFromGambling,
		Table2ModifierFrom:   earnedBy,
	}

	return r.liftPassages(t)
}

// liftBenefits lifts Mustering Out Table 1, Material Benefits (p. 9).
//
// Implements: The dash rows deliver nothing [1:9].
// Implements: A printed table governs over the example's stated result [1:9, 25
// (E015)].
func (r *Rules) liftBenefits(t musteringTranscription) error {
	for n, row := range t.table1 {
		for i, cell := range row {
			benefit, err := parseBenefitRow(cell, t.names)
			if err != nil {
				return fmt.Errorf("mustering out table 1, row %d, %v: %w", n+1, traveller.ServiceNames[i], err)
			}

			r.services[i].Benefits[n] = benefit
		}
	}

	return nil
}

// liftCash lifts Mustering Out Table 2, Cash Allowances (p. 9).
func (r *Rules) liftCash(t musteringTranscription) error {
	for n, row := range t.table2 {
		for i, cash := range row {
			if cash <= 0 {
				return fmt.Errorf("%w: mustering out table 2, row %d, %v: %d is not a cash allowance",
					errMalformed, n+1, traveller.ServiceNames[i], cash)
			}

			r.services[i].Cash[n] = traveller.Credits(cash)
		}
	}

	return nil
}

// liftPassages lifts the purchase prices of pp. 21-22 and the resale rate.
func (r *Rules) liftPassages(t musteringTranscription) error {
	for _, class := range traveller.PassageClasses {
		price := t.prices[class]
		if price <= 0 {
			return fmt.Errorf("%w: passages: no price for %v", errMalformed, class)
		}

		r.passages[class] = traveller.Credits(price)
	}

	if t.resalePercent <= 0 {
		return fmt.Errorf("%w: passages: no resale percentage", errMalformed)
	}

	r.muster.ResalePercent = t.resalePercent

	return nil
}

// Implements: Education and Social Standing are unaffected by aging [1:9].
// Implements: Intelligence is unaffected before age 66 [1:9].
func (r *Rules) liftAging(t agingTranscription) error {
	for _, band := range t.bands {
		lifted := agingBand{fromTerm: traveller.Term(band.fromTerm)}
		for _, e := range band.effects {
			characteristic, err := parseCharacteristic(e.characteristic)
			if err != nil {
				return fmt.Errorf("aging band from term %d: %w", band.fromTerm, err)
			}

			saving, err := traveller.ParseTarget(e.saving)
			if err != nil {
				return fmt.Errorf("%w: aging band from term %d: %w", errMalformed, band.fromTerm, err)
			}

			lifted.effects = append(lifted.effects, AgingEffect{
				Characteristic: characteristic, Reduction: e.reduction, Saving: saving,
			})
		}

		if len(r.aging.bands) > 0 && lifted.fromTerm <= r.aging.bands[len(r.aging.bands)-1].fromTerm {
			return fmt.Errorf("%w: aging bands are out of order at term %d", errMalformed, band.fromTerm)
		}

		r.aging.bands = append(r.aging.bands, lifted)
	}

	if len(r.aging.bands) == 0 {
		return fmt.Errorf("%w: aging: no bands", errMalformed)
	}

	// The last term the table's header row prints. It has to reach the last
	// band, or the band would begin at a term the table never names.
	last := traveller.Term(t.lastPrintedTerm)
	if last < r.aging.bands[len(r.aging.bands)-1].fromTerm {
		return fmt.Errorf("%w: aging: the last printed term is %d, before the last band begins",
			errMalformed, t.lastPrintedTerm)
	}

	r.aging.lastPrintedTerm = last

	saving, err := traveller.ParseTarget(t.crisisSaving)
	if err != nil {
		return fmt.Errorf("%w: medical crisis: %w", errMalformed, err)
	}

	r.aging.Crisis = Crisis{
		Saving: saving, RecoversTo: t.crisisRecoversTo, MonthsDice: t.crisisMonthsDice,
	}

	return nil
}

// liftWeapons lifts the printed weapon list of each category, in the order
// the page prints it.
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. The weapon lists are
// read column-major. The Blades and Polearms box (p. 12) and the Guns box
// (p. 13) are each set in two columns, and the page never states a reading order —
// but only one order makes either box coherent. Column-major gives blades
// Dagger, Blade, Foil, Cutlass, Sword, Broadsword and then polearms Spear,
// Halberd, Pike, Cudgel, Bayonet, which is the box's own title read in order;
// and it gives guns Body Pistol, Automatic Pistol, Revolver, Carbine, Rifle
// before Laser Carbine, Laser Rifle, Automatic Rifle, Submachine Gun, Shotgun,
// conventional arms before advanced. Row-major interleaves both groups into
// nonsense. The order matters because it fixes what "first in the printed list"
// names in the auto policy's answers.
func (r *Rules) liftWeapons(lists [len(traveller.WeaponCategories)][]string) error {
	for _, category := range traveller.WeaponCategories {
		printed := lists[category]
		if len(printed) == 0 {
			return fmt.Errorf("%w: weapons: no list for %v", errMalformed, category)
		}

		seen := make(map[string]bool, len(printed))
		for _, name := range printed {
			if seen[name] {
				return fmt.Errorf("%w: weapons, %v: %q is listed twice", errMalformed, category, name)
			}

			seen[name] = true
			r.weapons[category] = append(r.weapons[category], traveller.WeaponName(name))
		}
	}

	return nil
}

// liftShips reads the two starships a mustering out benefit can deliver
// (Book 2 pp. 18-19).
func (r *Rules) liftShips(hulls [len(traveller.ShipKinds)]int) error {
	for _, kind := range traveller.ShipKinds {
		if hulls[kind] <= 0 {
			return fmt.Errorf("%w: ships, %v: %d is not a hull size", errMalformed, kind, hulls[kind])
		}

		r.hulls[kind] = hulls[kind]
	}

	return nil
}

func (r *Rules) liftNobility(titles []printedTitle) error {
	for _, row := range titles {
		title, err := parseTitle(row.title)
		if err != nil {
			return fmt.Errorf("nobility: %w", err)
		}

		r.nobility = append(r.nobility, Nobility{SocialStanding: row.socialStanding, Title: title})
	}

	if len(r.nobility) != len(traveller.Titles) {
		return fmt.Errorf("%w: nobility: %d rows, want %d", errMalformed, len(r.nobility), len(traveller.Titles))
	}

	return nil
}
