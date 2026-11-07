package rules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// The wire types. They are JSON shapes and nothing more: every value is a
// string or a number, and every one of them is checked on the way into a
// domain value below. Nothing outside this file may hold a wire type.

type wireThrow struct {
	Target string   `json:"target"`
	DMs    []wireDM `json:"dms"`
}

type wireDM struct {
	DM int    `json:"dm"`
	If string `json:"if"`
}

type wireServices struct {
	Services   []string     `json:"services"`
	Enlistment []*wireThrow `json:"enlistment"`
	Draft      []int        `json:"draft"`
	Survival   []*wireThrow `json:"survival"`
	Commission []*wireThrow `json:"commission"`
	Promotion  []*wireThrow `json:"promotion"`
	Reenlist   []string     `json:"reenlist"`
	Ranks      [][]string   `json:"ranks"`

	RankAndServiceSkills []struct {
		Service string `json:"service"`
		Rank    int    `json:"rank"`
		Grant   string `json:"grant"`
	} `json:"rankAndServiceSkills"`

	RetirementPay struct {
		ByTerms []struct {
			Terms int   `json:"terms"`
			Pay   int64 `json:"pay"`
		} `json:"byTerms"`
		PerTermBeyondEight int64    `json:"perTermBeyondEight"`
		PaidBy             []string `json:"paidBy"`
	} `json:"retirementPay"`
}

type wireSkills struct {
	Services    []string              `json:"services"`
	Tables      map[string][][]string `json:"tables"`
	Eligibility struct {
		InitialTerm       int `json:"initialTerm"`
		PerSubsequentTerm int `json:"perSubsequentTerm"`
		OnCommission      int `json:"onCommission"`
		OnPromotion       int `json:"onPromotion"`
	} `json:"eligibility"`
	EducationGate struct {
		Table          string `json:"table"`
		Characteristic string `json:"characteristic"`
		Threshold      string `json:"threshold"`
	} `json:"educationGate"`
	Normalizations map[string]string `json:"normalizations"`
}

type wireMustering struct {
	Services []string   `json:"services"`
	Table1   [][]string `json:"table1"`
	Table2   [][]int64  `json:"table2"`
	Rolls    struct {
		Table1DMFromRank5or6 int    `json:"table1ModifierFromRank5or6"`
		Table2DMFromGambling int    `json:"table2ModifierFromGambling"`
		Table2ModifierFrom   string `json:"table2ModifierFrom"`
	} `json:"rolls"`
	Names map[string]string `json:"names"`

	// Prices and the resale rate are separate fields, so the type carries
	// what the lift would otherwise check: a price that is not a number
	// fails on the way in (#49). One object keyed by passage class would
	// force the field to any and the lift to assert its way back.
	Passages struct {
		Prices        map[string]int64 `json:"prices"`
		ResalePercent int              `json:"resalePercent"`
	} `json:"passages"`
}

type wireAging struct {
	LastPrintedTerm int `json:"lastPrintedTerm"`
	Bands           []struct {
		FromTerm int `json:"fromTerm"`
		Effects  []struct {
			Characteristic string `json:"characteristic"`
			Reduction      int    `json:"reduction"`
			Saving         string `json:"saving"`
		} `json:"effects"`
	} `json:"bands"`
	Crisis struct {
		Saving     string `json:"saving"`
		RecoversTo int    `json:"recoversTo"`
		MonthsDice int    `json:"monthsDice"`
	} `json:"crisis"`
}

type wireWeapons struct {
	Lists map[string][]string `json:"lists"`
}

type wireShips struct {
	Ships []struct {
		Kind     string `json:"kind"`
		HullTons int    `json:"hullTons"`
	} `json:"ships"`
}

type wireNobility struct {
	Titles []struct {
		SocialStanding int    `json:"socialStanding"`
		Title          string `json:"title"`
	} `json:"titles"`
}

// withoutNotes drops the keys a data file uses for its own citations and
// commentary, which begin with an underscore and are not data.
func withoutNotes[T any](m map[string]T) map[string]T {
	kept := make(map[string]T, len(m))
	for k, v := range m {
		if !strings.HasPrefix(k, "_") {
			kept[k] = v
		}
	}

	return kept
}

func read[T any](name string) (T, error) {
	var wire T

	text, err := files.ReadFile("data/" + name)
	if err != nil {
		return wire, fmt.Errorf("%w: reading %s: %w", errMalformed, name, err)
	}

	err = json.Unmarshal(text, &wire)
	if err != nil {
		return wire, fmt.Errorf("%w: %s: %w", errMalformed, name, err)
	}

	return wire, nil
}

// load lifts every table. It runs once, on first use, so a data file that
// will not lift stops the program where the defect is rather than where a
// path happens to reach it.
func load() (*Rules, error) {
	rules := &Rules{}

	skills, err := read[wireSkills]("skills.json")
	if err != nil {
		return nil, err
	}

	rules.normalize = withoutNotes(skills.Normalizations)

	mustering, err := read[wireMustering]("mustering.json")
	if err != nil {
		return nil, err
	}

	services, err := read[wireServices]("services.json")
	if err != nil {
		return nil, err
	}

	aging, err := read[wireAging]("aging.json")
	if err != nil {
		return nil, err
	}

	weapons, err := read[wireWeapons]("weapons.json")
	if err != nil {
		return nil, err
	}

	nobility, err := read[wireNobility]("nobility.json")
	if err != nil {
		return nil, err
	}

	ships, err := read[wireShips]("ships.json")
	if err != nil {
		return nil, err
	}

	for _, lift := range []func() error{
		func() error { return rules.liftPriorService(services) },
		func() error { return rules.liftGrants(services) },
		func() error { return rules.liftRetirement(services) },
		func() error { return rules.liftSkills(skills) },
		func() error { return rules.liftMustering(mustering) },
		func() error { return rules.liftAging(aging) },
		func() error { return rules.liftWeapons(weapons) },
		func() error { return rules.liftNobility(nobility) },
		func() error { return rules.liftShips(ships) },
		rules.liftReadings,
	} {
		err := lift()
		if err != nil {
			return nil, err
		}
	}

	return rules, nil
}

// eachService checks that a row of a table has exactly one cell per service,
// in the order the domain lists them, and hands each cell to fn.
func eachService(header, row []string, what string, fn func(traveller.ServiceName, string) error) error {
	err := checkColumns(header, what)
	if err != nil {
		return err
	}

	if len(row) != len(header) {
		return fmt.Errorf("%w: %s: %d cells, want %d", errMalformed, what, len(row), len(header))
	}

	for i, cell := range row {
		err := fn(traveller.ServiceNames[i], cell)
		if err != nil {
			return fmt.Errorf("%w: %s, %v: %w", errMalformed, what, traveller.ServiceNames[i], err)
		}
	}

	return nil
}

// checkColumns holds a table's column headings to the six services, in the
// order p. 10 prints them.
//
// The count is checked before the order, because the order comparison
// indexes the domain's own array by column: a seventh column has to be
// refused before it is reached.
func checkColumns(headings []string, what string) error {
	if len(headings) != len(traveller.ServiceNames) {
		return fmt.Errorf("%w: %s: %d columns, want %d", errMalformed, what,
			len(headings), len(traveller.ServiceNames))
	}

	for i, name := range headings {
		service, err := parseService(name)
		if err != nil {
			return fmt.Errorf("%w: %s column %d: %w", errMalformed, what, i+1, err)
		}

		if service != traveller.ServiceNames[i] {
			return fmt.Errorf("%w: %s column %d is %v, want %v", errMalformed, what,
				i+1, service, traveller.ServiceNames[i])
		}
	}

	return nil
}

// Implements: A data file broken two ways reports the same one every run [JSON
// conventions] in The record.
func (r *Rules) liftPriorService(wire wireServices) error {
	err := checkColumns(wire.Services, "prior service table")
	if err != nil {
		return err
	}

	rows := map[string]int{
		"enlistment": len(wire.Enlistment), "draft": len(wire.Draft),
		"survival": len(wire.Survival), "commission": len(wire.Commission),
		"promotion": len(wire.Promotion), "reenlist": len(wire.Reenlist),
		"ranks": len(wire.Ranks),
	}
	for row, n := range rows {
		if n != len(traveller.ServiceNames) {
			return fmt.Errorf("%w: prior service table, %s: %d cells, want %d", errMalformed,
				row, n, len(traveller.ServiceNames))
		}
	}

	err = checkDraft(wire.Draft)
	if err != nil {
		return err
	}

	for i, name := range traveller.ServiceNames {
		service, err := liftServiceColumn(wire, i, name)
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
func checkDraft(draft []int) error {
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
func liftServiceColumn(wire wireServices, i int, name traveller.ServiceName) (Service, error) {
	service := Service{Name: name, Draft: wire.Draft[i], Ranks: wire.Ranks[i]}

	var err error

	service.Enlistment, err = liftThrow(wire.Enlistment[i], "enlistment")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.Survival, err = liftThrow(wire.Survival[i], "survival")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.Reenlist, err = traveller.ParseTarget(wire.Reenlist[i])
	if err != nil {
		return Service{}, fmt.Errorf("%w: %v reenlist: %w", errMalformed, name, err)
	}

	// P. 10 makes ranks, commissions and promotions one fact: "Ranks,
	// commissions, and promotions are non-existent in the scout and other
	// services." A column that prints one without the others is malformed.
	hasRanks := len(service.Ranks) > 0

	// A slice and not a map: with both columns malformed, a range over a map
	// names an arbitrary one of them, so the same broken data file reports a
	// different problem run to run and a reader cannot tell whether he fixed
	// anything (#53). The order is the page's.
	for _, printed := range []struct {
		label string
		cell  *wireThrow
	}{
		{"commission", wire.Commission[i]},
		{"promotion", wire.Promotion[i]},
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

	service.commission, err = liftThrow(wire.Commission[i], "commission")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.promotion, err = liftThrow(wire.Promotion[i], "promotion")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	return service, nil
}

func liftThrow(wire *wireThrow, what string) (Throw, error) {
	if wire == nil {
		return Throw{}, fmt.Errorf("%w: %s: no throw printed", errMalformed, what)
	}

	target, err := traveller.ParseTarget(wire.Target)
	if err != nil {
		return Throw{}, fmt.Errorf("%w: %s: %w", errMalformed, what, err)
	}

	throw := Throw{Target: target}

	for _, dm := range wire.DMs {
		lifted, err := parseDM(dm.DM, dm.If)
		if err != nil {
			return Throw{}, fmt.Errorf("%w: %s: %w", errMalformed, what, err)
		}

		throw.DMs = append(throw.DMs, lifted)
	}

	return throw, nil
}

func (r *Rules) liftGrants(wire wireServices) error {
	for _, g := range wire.RankAndServiceSkills {
		service, err := parseService(g.Service)
		if err != nil {
			return fmt.Errorf("%w: rank and service skills: %w", errMalformed, err)
		}

		result, err := parseTableResult(g.Grant, r.normalize)
		if err != nil {
			return fmt.Errorf("%w: rank and service skills, %v: %w", errMalformed, service, err)
		}

		if g.Rank < 0 || traveller.Rank(g.Rank) > r.services[service].MaxRank() {
			return fmt.Errorf("%w: rank and service skills, %v: rank %d is past the service's table of ranks", errMalformed,
				service, g.Rank)
		}

		r.grants = append(r.grants, Grant{Service: service, Rank: traveller.Rank(g.Rank), Result: result})
	}

	return nil
}

func (r *Rules) liftRetirement(wire wireServices) error {
	pay := wire.RetirementPay
	if len(pay.ByTerms) == 0 {
		return fmt.Errorf("%w: retirement pay: the table prints no rows", errMalformed)
	}

	r.retirement.byTerms = make(map[int]traveller.Credits, len(pay.ByTerms))
	for _, row := range pay.ByTerms {
		if row.Pay <= 0 {
			return fmt.Errorf("%w: retirement pay at %d terms: %d is not a pension", errMalformed, row.Terms, row.Pay)
		}

		if row.Terms <= r.retirement.lastTabled {
			return fmt.Errorf("%w: retirement pay: %d terms does not follow %d; the table's rows ascend", errMalformed,
				row.Terms, r.retirement.lastTabled)
		}

		r.retirement.byTerms[row.Terms] = traveller.Credits(row.Pay)
		r.retirement.lastTabled = row.Terms
	}

	if pay.PerTermBeyondEight <= 0 {
		return fmt.Errorf("%w: retirement pay: %d per additional term past the table", errMalformed,
			pay.PerTermBeyondEight)
	}

	r.retirement.PerAdditionalTerm = traveller.Credits(pay.PerTermBeyondEight)

	if len(pay.PaidBy) == 0 {
		return fmt.Errorf("%w: retirement pay: no service pays it", errMalformed)
	}

	for _, name := range pay.PaidBy {
		service, err := parseService(name)
		if err != nil {
			return fmt.Errorf("%w: retirement pay: %w", errMalformed, err)
		}

		r.services[service].PaysPension = true
	}

	return nil
}

func (r *Rules) liftSkills(wire wireSkills) error {
	tables := withoutNotes(wire.Tables)
	if len(tables) != len(traveller.SkillTables) {
		return fmt.Errorf("%w: acquired skills: %d tables, want %d", errMalformed, len(tables), len(traveller.SkillTables))
	}

	for name, rows := range tables {
		table, err := parseSkillTable(name)
		if err != nil {
			return fmt.Errorf("%w: acquired skills: %w", errMalformed, err)
		}

		if len(rows) != Faces {
			return fmt.Errorf("%w: acquired skills, %v: %d rows, want %d", errMalformed, table, len(rows), Faces)
		}

		for die, row := range rows {
			what := fmt.Sprintf("acquired skills, %v, row %d", table, die+1)

			err := eachService(wire.Services, row, what, func(s traveller.ServiceName, cell string) error {
				result, err := parseTableResult(cell, r.normalize)
				if err != nil {
					return err
				}

				r.services[s].Skills[table][die] = result

				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	r.eligibility = Eligibility{
		InitialTerm:       wire.Eligibility.InitialTerm,
		PerSubsequentTerm: wire.Eligibility.PerSubsequentTerm,
		OnCommission:      wire.Eligibility.OnCommission,
		OnPromotion:       wire.Eligibility.OnPromotion,
	}

	table, err := parseSkillTable(wire.EducationGate.Table)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	characteristic, err := parseCharacteristic(wire.EducationGate.Characteristic)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	threshold, err := traveller.ParseTarget(wire.EducationGate.Threshold)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	r.education = Gate{Table: table, Characteristic: characteristic, Threshold: threshold}

	return nil
}

func (r *Rules) liftMustering(wire wireMustering) error {
	if len(wire.Table1) != MusterRows || len(wire.Table2) != MusterRows {
		return fmt.Errorf("%w: mustering out: %d and %d rows, want %d each", errMalformed,
			len(wire.Table1), len(wire.Table2), MusterRows)
	}

	err := r.liftBenefits(wire)
	if err != nil {
		return err
	}

	err = r.liftCash(wire)
	if err != nil {
		return err
	}

	// The skill that earns the Table 2 DM goes through E012's normalization
	// on the way in, exactly as a table cell does. The engine compares it
	// against skills the character holds, and those were spelled by
	// parseTableResult through the same map - a name lifted raw here would
	// stop matching the moment a normalization for it was added, and the
	// modifier would silently never apply.
	earnedBy := r.Normalize(wire.Rolls.Table2ModifierFrom)
	if earnedBy == "" {
		return fmt.Errorf("%w: mustering out: no skill earns the table 2 modifier", errMalformed)
	}

	r.muster = Muster{
		Table1DMFromRank5or6: wire.Rolls.Table1DMFromRank5or6,
		Table2DMFromGambling: wire.Rolls.Table2DMFromGambling,
		Table2ModifierFrom:   earnedBy,
	}

	return r.liftPassages(wire)
}

// liftBenefits lifts Mustering Out Table 1, Material Benefits (p. 9).
//
// Implements: The dash rows deliver nothing [1:9] in Mustering out.
// Implements: A printed table governs over the example's stated result [1:9, 25
// (E015)] in The book's own character.
func (r *Rules) liftBenefits(wire wireMustering) error {
	names := withoutNotes(wire.Names)

	for n, row := range wire.Table1 {
		what := fmt.Sprintf("mustering out table 1, row %d", n+1)

		err := eachService(wire.Services, row, what, func(s traveller.ServiceName, cell string) error {
			benefit, err := parseBenefitRow(cell, names)
			if err != nil {
				return err
			}

			r.services[s].Benefits[n] = benefit

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// liftCash lifts Mustering Out Table 2, Cash Allowances (p. 9).
func (r *Rules) liftCash(wire wireMustering) error {
	for n, row := range wire.Table2 {
		if len(row) != len(traveller.ServiceNames) {
			return fmt.Errorf("%w: mustering out table 2, row %d: %d cells, want %d", errMalformed,
				n+1, len(row), len(traveller.ServiceNames))
		}

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
func (r *Rules) liftPassages(wire wireMustering) error {
	for _, class := range traveller.PassageClasses {
		price, ok := wire.Passages.Prices[class.String()]
		if !ok {
			return fmt.Errorf("%w: passages: no price for %v", errMalformed, class)
		}

		r.passages[class] = traveller.Credits(price)
	}

	if wire.Passages.ResalePercent <= 0 {
		return fmt.Errorf("%w: passages: no resale percentage", errMalformed)
	}

	r.muster.ResalePercent = wire.Passages.ResalePercent

	return nil
}

// Implements: Education and Social Standing are unaffected by aging [1:9] in
// Aging.
// Implements: Intelligence is unaffected before age 66 [1:9] in Aging.
func (r *Rules) liftAging(wire wireAging) error {
	for _, band := range wire.Bands {
		lifted := agingBand{fromTerm: traveller.Term(band.FromTerm)}
		for _, e := range band.Effects {
			characteristic, err := parseCharacteristic(e.Characteristic)
			if err != nil {
				return fmt.Errorf("%w: aging band from term %d: %w", errMalformed, band.FromTerm, err)
			}

			saving, err := traveller.ParseTarget(e.Saving)
			if err != nil {
				return fmt.Errorf("%w: aging band from term %d: %w", errMalformed, band.FromTerm, err)
			}

			lifted.effects = append(lifted.effects, AgingEffect{
				Characteristic: characteristic, Reduction: e.Reduction, Saving: saving,
			})
		}

		if len(r.aging.bands) > 0 && lifted.fromTerm <= r.aging.bands[len(r.aging.bands)-1].fromTerm {
			return fmt.Errorf("%w: aging bands are out of order at term %d", errMalformed, band.FromTerm)
		}

		r.aging.bands = append(r.aging.bands, lifted)
	}

	if len(r.aging.bands) == 0 {
		return fmt.Errorf("%w: aging: no bands", errMalformed)
	}

	// The last term the table's header row prints. It has to reach the last
	// band, or the band would begin at a term the table never names.
	last := traveller.Term(wire.LastPrintedTerm)
	if last < r.aging.bands[len(r.aging.bands)-1].fromTerm {
		return fmt.Errorf("%w: aging: the last printed term is %d, before the last band begins",
			errMalformed, wire.LastPrintedTerm)
	}

	r.aging.lastPrintedTerm = last

	saving, err := traveller.ParseTarget(wire.Crisis.Saving)
	if err != nil {
		return fmt.Errorf("%w: medical crisis: %w", errMalformed, err)
	}

	r.aging.Crisis = Crisis{
		Saving: saving, RecoversTo: wire.Crisis.RecoversTo, MonthsDice: wire.Crisis.MonthsDice,
	}

	return nil
}

func (r *Rules) liftWeapons(wire wireWeapons) error {
	for _, category := range traveller.WeaponCategories {
		printed, ok := wire.Lists[category.String()]
		if !ok || len(printed) == 0 {
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
func (r *Rules) liftShips(wire wireShips) error {
	seen := make(map[traveller.ShipKind]bool, len(traveller.ShipKinds))

	for _, row := range wire.Ships {
		kind, err := lookup(shipKinds, row.Kind, "ship")
		if err != nil {
			return fmt.Errorf("%w: ships: %w", errMalformed, err)
		}

		if row.HullTons <= 0 {
			return fmt.Errorf("%w: ships, %v: %d is not a hull size", errMalformed, kind, row.HullTons)
		}

		if seen[kind] {
			return fmt.Errorf("%w: ships: %v is listed twice", errMalformed, kind)
		}

		r.hulls[kind] = row.HullTons
		seen[kind] = true
	}

	for _, kind := range traveller.ShipKinds {
		if !seen[kind] {
			return fmt.Errorf("%w: ships: no hull for %v", errMalformed, kind)
		}
	}

	return nil
}

func (r *Rules) liftNobility(wire wireNobility) error {
	for _, row := range wire.Titles {
		title, err := parseTitle(row.Title)
		if err != nil {
			return fmt.Errorf("%w: nobility: %w", errMalformed, err)
		}

		r.nobility = append(r.nobility, Nobility{SocialStanding: row.SocialStanding, Title: title})
	}

	if len(r.nobility) != len(traveller.Titles) {
		return fmt.Errorf("%w: nobility: %d rows, want %d", errMalformed, len(r.nobility), len(traveller.Titles))
	}

	return nil
}
