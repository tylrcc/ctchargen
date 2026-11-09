package rules

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

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

	// A DM row's heading belongs to the row, not to a service, so a bad one
	// is refused as the row's before any service's column is read.
	for _, throw := range []struct {
		label string
		rows  printedThrows
	}{
		{"enlistment", t.enlistment},
		{"survival", t.survival},
		{"commission", t.commission},
		{"promotion", t.promotion},
	} {
		for _, row := range throw.rows.dms {
			_, err := parseDMHeading(row.heading)
			if err != nil {
				return fmt.Errorf("prior service table, %s: %w", throw.label, err)
			}
		}
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
	ranks, err := liftRanks(rankColumn(t.ranks, i))
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service := Service{Name: name, Draft: t.draft[i], Ranks: ranks}

	service.Enlistment, err = liftColumn(t.enlistment, i, "enlistment")
	if err != nil {
		return Service{}, fmt.Errorf("%v: %w", name, err)
	}

	service.Survival, err = liftColumn(t.survival, i, "survival")
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
		rows  printedThrows
		into  *Throw
	}{
		{"commission", t.commission, &service.commission},
		{"promotion", t.promotion, &service.promotion},
	} {
		cell, err := printed.rows.column(i, printed.label)
		if err != nil {
			return Service{}, fmt.Errorf("%v: %w", name, err)
		}

		throw, isPrinted, err := liftOptionalThrow(cell, printed.label)
		if err != nil {
			return Service{}, fmt.Errorf("%v: %w", name, err)
		}

		if isPrinted != hasRanks {
			return Service{}, fmt.Errorf(
				"%w: %v prints %s but %s ranks: p. 10 makes ranks, commissions and promotions one fact",
				errMalformed, name, printed.label, map[bool]string{true: "also", false: "no"}[hasRanks],
			)
		}

		*printed.into = throw
	}

	return service, nil
}

// rankColumn is one service's cells of the Table of Ranks, Rank 1 down, read
// out of the rows the page prints.
func rankColumn(rows [rankRows]perService, i int) [rankRows]string {
	var column [rankRows]string

	for row, cells := range rows {
		column[row] = cells[i]
	}

	return column
}

// liftRanks reads a service's column of the Table of Ranks (p. 10): a name
// for each rank it has, from Rank 1, and an em dash for each it does not. A
// name below a dash would be a rank the service has without the one before
// it, which the table never prints.
func liftRanks(cells [rankRows]string) ([]string, error) {
	var ranks []string

	for row, cell := range cells {
		err := refuseEmpty(cell)
		if err == nil {
			err = refuseDashMistyped(cell)
		}

		if err != nil {
			return nil, fmt.Errorf("table of ranks, rank %d: %w", row+1, err)
		}

		if cell == emDash {
			continue
		}

		if len(ranks) != row {
			return nil, fmt.Errorf("%w: table of ranks, rank %d: %q is printed below a dash", errMalformed, row+1, cell)
		}

		ranks = append(ranks, cell)
	}

	return ranks, nil
}

// printedThrow is one service's column of a throw: its target, and its
// condition from each DM row with the amount that row's heading prints.
type printedThrow struct {
	target string
	dms    []printedDM
}

// printedDM is one DM: the amount its row's heading prints, and the
// condition the page prints for the service.
type printedDM struct {
	dm        int
	condition string
}

// column is the throw the rows print for the service at index i, each DM's
// amount read off its row's heading.
func (rows printedThrows) column(i int, what string) (printedThrow, error) {
	throw := printedThrow{target: rows.targets[i]}

	for _, row := range rows.dms {
		amount, err := parseDMHeading(row.heading)
		if err != nil {
			return printedThrow{}, fmt.Errorf("%s: %w", what, err)
		}

		throw.dms = append(throw.dms, printedDM{dm: amount, condition: row.conditions[i]})
	}

	return throw, nil
}

// liftColumn lifts the throw the rows print for the service at index i, one
// every service has.
func liftColumn(rows printedThrows, i int, what string) (Throw, error) {
	cell, err := rows.column(i, what)
	if err != nil {
		return Throw{}, err
	}

	return liftThrow(cell, what)
}

// liftOptionalThrow lifts a throw a service may not have: an em dash for a
// target is the page saying it has none, and every DM beside it must then be
// a dash too.
func liftOptionalThrow(printed printedThrow, what string) (Throw, bool, error) {
	err := refuseDashMistyped(printed.target)
	if err != nil {
		return Throw{}, false, fmt.Errorf("%s: %w", what, err)
	}

	if printed.target != emDash {
		throw, err := liftThrow(printed, what)

		return throw, true, err
	}

	for _, dm := range printed.dms {
		if dm.condition != emDash {
			return Throw{}, false, fmt.Errorf("%w: %s: a DM of %q beside a throw the page does not print",
				errMalformed, what, dm.condition)
		}
	}

	return Throw{}, false, nil
}

// liftThrow lifts a throw the page prints: its target, and each DM whose
// condition is not an em dash. The three dashes are decided once, on emDash.
func liftThrow(printed printedThrow, what string) (Throw, error) {
	target, err := traveller.ParseTarget(printed.target)
	if err != nil {
		return Throw{}, fmt.Errorf("%w: %s: %w", errMalformed, what, err)
	}

	throw := Throw{Target: target}

	for _, dm := range printed.dms {
		if dm.condition == emDash {
			continue
		}

		err := refuseDashMistyped(dm.condition)
		if err != nil {
			return Throw{}, fmt.Errorf("%s: %w", what, err)
		}

		lifted, err := parseDM(dm.dm, dm.condition)
		if err != nil {
			return Throw{}, fmt.Errorf("%s: %w", what, err)
		}

		throw.DMs = append(throw.DMs, lifted)
	}

	return throw, nil
}

// liftGrants lifts the Rank and Service Skills box (p. 23). Each label names
// a service in the box's own spelling and, after it, a rank title from that
// service's column of the Table of Ranks (p. 10); a label with no title is
// service-wide, held as rank 0, which E005 reads as granted once, at entry. The prior service table is
// lifted first, so the titles are there to be read.
func (r *Rules) liftGrants(t servicesTranscription) error {
	for _, g := range t.grants {
		service, rank, err := r.parseBoxLabel(g.label)
		if err != nil {
			return fmt.Errorf("rank and service skills: %w", err)
		}

		printed, err := withoutBoxLevel(g.grant)
		if err != nil {
			return fmt.Errorf("rank and service skills, %s: %w", g.label, err)
		}

		result, err := parseTableResult(printed, r.normalize)
		if err != nil {
			return fmt.Errorf("rank and service skills, %s: %w", g.label, err)
		}

		r.grants = append(r.grants, Grant{Service: service, Rank: rank, Result: result})
	}

	return nil
}

// parseBoxLabel reads a box label as the service it names and the rank its
// title holds in that service's Table of Ranks column, 0 for none.
func (r *Rules) parseBoxLabel(label string) (traveller.ServiceName, traveller.Rank, error) {
	word, title, _ := strings.Cut(label, " ")

	service, ok := boxServices[word]
	if !ok {
		return 0, 0, fmt.Errorf("%w: %q opens with %q, which the box does not spell a service as", errMalformed, label, word)
	}

	if title == "" {
		return service, 0, nil
	}

	for i, name := range r.services[service].Ranks {
		if name == title {
			return service, traveller.Rank(i + 1), nil
		}
	}

	return 0, 0, fmt.Errorf("%w: %q: %v's Table of Ranks prints no %q", errMalformed, label, service, title)
}

// boxLevel is the suffix the box prints after each skill it grants: one
// level, added "in the normal manner" (p. 23), which is what a grant adds.
const boxLevel = "-1"

// withoutBoxLevel is a grant without its level: a skill must carry exactly the
// box's "-1", and a characteristic alteration carries none, as printed.
func withoutBoxLevel(grant string) (string, error) {
	skill, hasLevel := strings.CutSuffix(grant, boxLevel)

	signed, err := startsWithSign(grant)
	if err != nil {
		return "", err
	}

	switch {
	case signed && hasLevel:
		return "", fmt.Errorf("%w: %q is an alteration, which the box prints with no level", errMalformed, grant)
	case signed:
		return grant, nil
	case !hasLevel:
		return "", fmt.Errorf("%w: %q carries no %q, the level the box prints after every skill", errMalformed,
			grant, boxLevel)
	}

	return skill, nil
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

	return nil
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

// liftAging lifts the Aging Table as p. 9 prints it, and the medical crisis.
// The headers give the last printed term, each row's cells give its effects
// across the term columns, and the bands are where the rows' cells begin,
// taken together.
//
// Implements: Education and Social Standing are unaffected by aging [1:9].
// Implements: Intelligence is unaffected before age 66 [1:9].
func (r *Rules) liftAging(t agingTranscription) error {
	first, last, err := liftAgingHeaders(t.terms, t.ages)
	if err != nil {
		return err
	}

	rows, err := liftAgingRows(t.rows, first, last)
	if err != nil {
		return err
	}

	r.aging.bands = agingBands(rows)
	r.aging.lastPrintedTerm = traveller.Term(last)

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

// liftAgingHeaders reads the table's two header rows: the terms, each one more
// than the term before it, and the ages, each the age its term ends at
// (traveller.StartingAge plus traveller.Years a term), with a plus on the last
// age and nowhere else. That plus is the page fact
// E014 rests on; E014 itself is a reading, and the lift holds only the fact.
func liftAgingHeaders(terms, ages []string) (int, int, error) {
	if len(terms) == 0 || len(terms) != len(ages) {
		return 0, 0, fmt.Errorf("%w: aging table: %d terms and %d ages", errMalformed, len(terms), len(ages))
	}

	var first, previous int

	for i, printed := range terms {
		term, err := strconv.Atoi(printed)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: aging table: term %q is not a number", errMalformed, printed)
		}

		if i > 0 && term != previous+1 {
			return 0, 0, fmt.Errorf("%w: aging table: term %q does not follow %d", errMalformed, printed, previous)
		}

		age, open := strings.CutSuffix(ages[i], "+")

		switch last := i == len(ages)-1; {
		case open && !last:
			return 0, 0, fmt.Errorf("%w: aging table: age %q carries a plus before the last column", errMalformed, ages[i])
		case !open && last:
			return 0, 0, fmt.Errorf("%w: aging table: the last age, %q, carries no plus; "+
				"E014 reads that plus as the open column", errMalformed, ages[i])
		}

		if age != strconv.Itoa(traveller.StartingAge+traveller.Years*term) {
			return 0, 0, fmt.Errorf("%w: aging table: age %q is not the age term %d ends at", errMalformed, ages[i], term)
		}

		if i == 0 {
			first = term
		}

		previous = term
	}

	return first, previous, nil
}

// agingRow is one characteristic's row, lifted: its effect over each printed
// span, nil where the cell says it has none.
type agingRow struct {
	characteristic traveller.Characteristic
	cells          []agingCell
}

type agingCell struct {
	from, through int
	effect        *AgingEffect
}

// liftAgingRows reads the six rows, in the order p. 9 prints them, which is
// p. 4's. Each row's cells cover the term columns once, left to right.
func liftAgingRows(printed []printedAgingRow, first, last int) ([]agingRow, error) {
	if len(printed) != len(traveller.Characteristics) {
		return nil, fmt.Errorf("%w: aging table: %d rows, want %d", errMalformed,
			len(printed), len(traveller.Characteristics))
	}

	rows := make([]agingRow, 0, len(printed))

	for i, row := range printed {
		characteristic, err := parseCharacteristic(row.label)
		if err != nil {
			return nil, fmt.Errorf("aging table: %w", err)
		}

		if characteristic != traveller.Characteristics[i] {
			return nil, fmt.Errorf("%w: aging table: row %d is %v, want %v", errMalformed, i+1, characteristic,
				traveller.Characteristics[i])
		}

		cells, err := liftAgingCells(row.cells, characteristic, first, last)
		if err != nil {
			return nil, err
		}

		rows = append(rows, agingRow{characteristic: characteristic, cells: cells})
	}

	return rows, nil
}

// liftAgingCells reads one row's cells, refusing a gap, an overlap or a cell
// past the last printed term.
func liftAgingCells(printed []printedSpan, row traveller.Characteristic, first, last int) ([]agingCell, error) {
	next := first

	cells := make([]agingCell, 0, len(printed))

	for _, span := range printed {
		if span.through > last {
			return nil, fmt.Errorf("%w: aging table, %v: a cell across terms %d to %d runs past the last printed term, %d",
				errMalformed, row, span.from, span.through, last)
		}

		if span.from != next || span.through < span.from {
			return nil, fmt.Errorf("%w: aging table, %v: a cell across terms %d to %d, where term %d is next",
				errMalformed, row, span.from, span.through, next)
		}

		effect, err := parseAgingCell(span.text, row)
		if err != nil {
			return nil, fmt.Errorf("aging table, %v, terms %d to %d: %w", row, span.from, span.through, err)
		}

		cells = append(cells, agingCell{from: span.from, through: span.through, effect: effect})
		next = span.through + 1
	}

	if next != last+1 {
		return nil, fmt.Errorf("%w: aging table, %v: the cells stop before term %d", errMalformed, row, last)
	}

	return cells, nil
}

// agingCellText is a cell that reduces a characteristic, as p. 9 prints one:
// "−1 (8+)", the reduction and, in parentheses, the saving throw.
var agingCellText = regexp.MustCompile(`^(.)(\d+) \((\d+\+)\)$`)

// The two things p. 9 prints where a cell has no effect.
const (
	noEffectBefore66 = "no effect before age 66"
	unaffected       = "unaffected by aging"
)

// parseAgingCell reads a cell's text as a reduction and its saving throw, or
// as one of the two phrases the page prints for no effect, nil.
func parseAgingCell(text string, characteristic traveller.Characteristic) (*AgingEffect, error) {
	if text == noEffectBefore66 || text == unaffected {
		return nil, nil //nolint:nilnil // no effect is a cell, not an error
	}

	match := agingCellText.FindStringSubmatch(text)
	if match == nil {
		return nil, fmt.Errorf("%w: %q is not a reduction and saving throw, nor a phrase p. 9 prints", errMalformed, text)
	}

	glyph, _ := utf8.DecodeRuneInString(match[1])

	direction, err := traveller.Sign(glyph)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", errMalformed, text, err)
	}

	if direction >= 0 {
		return nil, fmt.Errorf("%w: %q is not a reduction", errMalformed, text)
	}

	reduction, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", errMalformed, text, err)
	}

	saving, err := traveller.ParseTarget(match[3])
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", errMalformed, text, err)
	}

	return &AgingEffect{Characteristic: characteristic, Reduction: reduction, Saving: saving}, nil
}

// agingBands is the table's bands: each begins where some row's cell begins,
// and holds, in row order, the reduction of each row whose cell over it has
// one.
func agingBands(rows []agingRow) []agingBand {
	var starts []int

	for _, row := range rows {
		for _, cell := range row.cells {
			if !slices.Contains(starts, cell.from) {
				starts = append(starts, cell.from)
			}
		}
	}

	slices.Sort(starts)

	bands := make([]agingBand, 0, len(starts))

	for _, start := range starts {
		band := agingBand{fromTerm: traveller.Term(start)}

		for _, row := range rows {
			for _, cell := range row.cells {
				if cell.from <= start && start <= cell.through && cell.effect != nil {
					band.effects = append(band.effects, *cell.effect)
				}
			}
		}

		bands = append(bands, band)
	}

	return bands
}
