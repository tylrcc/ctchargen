package rules //nolint:testpackage // the lift's guards are unexported

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// A malformed table must fail loudly, and must say which cell.
//
// Every case below starts from a copy of the real transcription and breaks
// exactly one thing, so what is under test is the guard rather than a
// hand-built fixture that resembles the table. The breakages are the ones the
// reprint's font can actually produce, plus the ones a careless edit can. A
// table, a service or a column the type does not have no longer compiles,
// and needs no case here.

// The copies deep enough to break: a transcription holds slices, maps and
// pointers, and a case that wrote through one would break the real table
// under every other test running beside it.

func servicesCopy() servicesTranscription {
	s := transcribedServices

	for _, rows := range []*printedThrows{&s.enlistment, &s.survival, &s.commission, &s.promotion} {
		rows.dms = slices.Clone(rows.dms)
	}

	s.grants = slices.Clone(s.grants)
	s.pensions = slices.Clone(s.pensions)
	s.pensionPaidBy = slices.Clone(s.pensionPaidBy)

	return s
}

// copyColumn prints the service at from's throw in the column of the service
// at to: its target and its condition in every DM row.
func copyColumn(rows *printedThrows, to, from traveller.ServiceName) {
	rows.targets[to] = rows.targets[from]

	for r := range rows.dms {
		rows.dms[r].conditions[to] = rows.dms[r].conditions[from]
	}
}

// blankColumn leaves the service's column of a throw empty, as a short Go
// array would.
func blankColumn(rows *printedThrows, service traveller.ServiceName) {
	rows.targets[service] = ""

	for r := range rows.dms {
		rows.dms[r].conditions[service] = ""
	}
}

func musteringCopy() musteringTranscription {
	m := transcribedMustering

	m.names = maps.Clone(m.names)

	return m
}

func agingCopy() agingTranscription {
	a := transcribedAging

	a.terms = slices.Clone(a.terms)
	a.ages = slices.Clone(a.ages)

	a.rows = slices.Clone(a.rows)
	for i := range a.rows {
		a.rows[i].cells = slices.Clone(a.rows[i].cells)
	}

	return a
}

func weaponsCopy() [len(traveller.WeaponCategories)][]string {
	w := transcribedWeapons
	for i := range w {
		w[i] = slices.Clone(w[i])
	}

	return w
}

// refuses asserts that lifting fails and that the message names what broke.
func refuses(t *testing.T, name string, err error, mentions string) {
	t.Helper()

	switch {
	case err == nil:
		t.Errorf("%s: lifted without complaint", name)
	case !strings.Contains(err.Error(), mentions):
		t.Errorf("%s: error %q does not mention %q", name, err, mentions)
	case !errors.Is(err, errMalformed):
		t.Errorf("%s: error %q is not a malformed table; every lift error wraps errMalformed", name, err)
	}
}

func TestMalformedServices(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*servicesTranscription)
		mentions string
	}{
		// A short array is padded with an empty throw in silence.
		"a throw left empty": {
			func(s *servicesTranscription) { blankColumn(&s.survival, traveller.Other) },
			"survival",
		},
		"a target that will not parse": {
			func(s *servicesTranscription) { s.enlistment.targets[0] = "eight" },
			"enlistment",
		},
		"a DM row heading that will not parse": {
			func(s *servicesTranscription) { s.survival.dms[0].heading = "DM of +x if" },
			"DM of +x if",
		},
		"a DM row heading without its if": {
			func(s *servicesTranscription) { s.survival.dms[0].heading = "DM of +2" },
			"DM of +2",
		},
		"a DM row heading with no sign": {
			func(s *servicesTranscription) { s.survival.dms[0].heading = "DM of x2 if" },
			"has no sign",
		},
		"a reenlist target that will not parse": {
			func(s *servicesTranscription) { s.reenlist[0] = "" },
			"reenlist",
		},
		"a commission printed for a service with no ranks": {
			func(s *servicesTranscription) { copyColumn(&s.commission, traveller.Scouts, traveller.Navy) },
			"p. 10 makes ranks",
		},
		"a commission missing from a service with ranks": {
			func(s *servicesTranscription) { copyColumn(&s.commission, traveller.Navy, traveller.Scouts) },
			"p. 10 makes ranks",
		},
		// Draft scans for the service printing the die, so a face printed
		// twice leaves another face naming nobody (#134).
		"a draft face printed twice": {
			func(s *servicesTranscription) { s.draft[5] = s.draft[4] },
			"printed twice",
		},
		"a draft number that is not a die face": {
			func(s *servicesTranscription) { s.draft[0] = 7 },
			"not a die face",
		},
	} {
		broken := servicesCopy()

		tc.breaks(&broken)
		refuses(t, name, (&Rules{}).liftPriorService(broken), tc.mentions)
	}
}

// retirementFixture lifts the real prior service table and hands back a copy
// of the transcription beside it, so a case can break exactly one row of one
// table.
func retirementFixture(t *testing.T) (*Rules, servicesTranscription) {
	t.Helper()

	r := &Rules{normalize: map[string]string{}}

	err := r.liftPriorService(transcribedServices)
	if err != nil {
		t.Fatalf("lifting the prior service table: %v", err)
	}

	broken := servicesCopy()

	// The cases below break one row of each of these, so a fixture that
	// quietly lost them would make them vacuous rather than failing.
	if len(broken.grants) == 0 || len(broken.pensions) < 2 || len(broken.pensionPaidBy) == 0 {
		t.Fatalf("the fixture is too thin to break: %d grants, %d pension rows, %d payers",
			len(broken.grants), len(broken.pensions), len(broken.pensionPaidBy))
	}

	return r, broken
}

// The Rank and Service Skills box keys each grant to a label as p. 23 prints
// it, which names a service in the box's own spelling and, after it, a rank
// title from that service's column of the Table of Ranks; and each skill grant
// carries the level suffix the box prints. Each part is refused by name.
func TestMalformedGrants(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		label, grant, mentions string
	}{
		"a service the box does not name":        {"Marines Lieutenant", "Revolver-1", "Marines"},
		"a rank title the service does not have": {"Navy Commodore", "+1 Social", "Commodore"},
		"a rank title another service has":       {"Scout Captain", "Pilot-1", "Captain"},
		"a level other than one":                 {"Marine", "Cutlass-2", "Cutlass-2"},
		"a skill with no level":                  {"Marine", "Cutlass", "Cutlass"},
		"an alteration with a level":             {"Navy Captain", "+1 Social-1", "which the box prints with no level"},
		"something that is not a result":         {"Navy Captain", "+1 Charisma", "Charisma"},
	} {
		r, broken := retirementFixture(t)

		broken.grants[0] = printedGrant{label: tc.label, grant: tc.grant}
		refuses(t, name, r.liftGrants(broken), tc.mentions)
	}
}

// The retirement pay table validates its own shape, so break it five ways.
func TestMalformedRetirement(t *testing.T) {
	t.Parallel()

	r, broken := retirementFixture(t)

	broken.pensions[1].pay = 0
	refuses(t, "a pension row that pays nothing", r.liftRetirement(broken), "is not a pension")

	r, broken = retirementFixture(t)
	broken.pensions[0], broken.pensions[1] = broken.pensions[1], broken.pensions[0]
	refuses(t, "pension rows out of order", r.liftRetirement(broken), "the table's rows ascend")

	r, broken = retirementFixture(t)
	broken.perAdditionalTerm = 0
	refuses(t, "nothing paid past the table", r.liftRetirement(broken), "per additional term")

	r, broken = retirementFixture(t)
	broken.pensionPaidBy = nil
	refuses(t, "a pension no service pays", r.liftRetirement(broken), "no service pays it")

	r, broken = retirementFixture(t)
	broken.pensions = nil
	refuses(t, "a pension table with no rows", r.liftRetirement(broken), "prints no rows")
}

func TestMalformedSkills(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*skillsTranscription)
		mentions string
	}{
		// A row a cell short compiles, and so does a table left out: Go
		// pads either with "", and the lift refuses the empty cell. A table
		// name the type does not have, and a gate on one, do not compile,
		// and need no case here.
		"a cell left empty": {
			func(s *skillsTranscription) { s.tables[traveller.ServiceSkills][5][5] = "" },
			"Service Skills Table, row 6, Other: malformed rules table: an empty cell",
		},
		"a cell that alters a characteristic that does not exist": {
			func(s *skillsTranscription) { s.tables[traveller.ServiceSkills][0][0] = "+1 Charisma" },
			"Charisma",
		},
		// A signed cell that is not an alteration is refused outright.
		// Letting it through would accumulate it as a skill whose name is
		// the whole cell - a wrong value that looks right on screen, which
		// is the failure this package exists to prevent.
		"a signed cell that is not an alteration": {
			func(s *skillsTranscription) { s.tables[traveller.ServiceSkills][0][0] = "+x Social Standing" },
			"begins with a sign",
		},
		// An en dash looks like a minus on screen and is no sign, so the
		// cell is refused by name rather than lifted as a skill.
		"an alteration typed with an en dash": {
			func(s *skillsTranscription) { s.tables[traveller.ServiceSkills][0][0] = "–1 Social Standing" },
			"an en dash is not a minus",
		},
		"an education gate threshold that will not parse": {
			func(s *skillsTranscription) { s.educationGate.threshold = "eight" },
			"education gate",
		},
		"an education gate on a characteristic that does not exist": {
			func(s *skillsTranscription) { s.educationGate.characteristic = "Wisdom" },
			"education gate",
		},
	} {
		broken := transcribedSkills

		tc.breaks(&broken)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftSkills(broken), tc.mentions)
	}
}

// An en-dash cell's refusal is the same error a target's is, so a caller
// can tell it from any other malformed cell.
func TestAnEnDashCellWrapsErrEnDash(t *testing.T) {
	t.Parallel()

	broken := transcribedSkills

	broken.tables[traveller.ServiceSkills][0][0] = "–1 Social Standing"

	err := (&Rules{normalize: map[string]string{}}).liftSkills(broken)
	if !errors.Is(err, traveller.ErrEnDash) {
		t.Errorf("lifting an en-dash cell: %v; want it to wrap traveller.ErrEnDash", err)
	}
}

func TestMalformedMustering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*musteringTranscription)
		mentions string
	}{
		// A short array is padded with "" in silence.
		"a benefit cell left empty": {
			func(m *musteringTranscription) { m.table1[6][5] = "" },
			"an empty cell",
		},
		// The font trap, lifted: p. 9's dash cells extract as the digit 4.
		"a dash cell that extracted as a 4": {
			func(m *musteringTranscription) { m.table1[6][3] = "4" },
			"not a benefit",
		},
		"a benefit no row prints": {
			func(m *musteringTranscription) { m.table1[0][0] = "Yacht" },
			"Yacht",
		},
		"a cash allowance of nothing": {
			func(m *musteringTranscription) { m.table2[0][0] = 0 },
			"not a cash allowance",
		},
		"no skill earning the table 2 modifier": {
			func(m *musteringTranscription) { m.table2ModifierFrom = "" },
			"earns the table 2 modifier",
		},
		"a passage with no price": {
			func(m *musteringTranscription) { m.prices[traveller.LowPassage] = 0 },
			"no price for Low Passage",
		},
		"no resale percentage": {
			func(m *musteringTranscription) { m.resalePercent = 0 },
			"resale",
		},
	} {
		broken := musteringCopy()

		tc.breaks(&broken)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftMustering(broken), tc.mentions)
	}
}

func TestMalformedAging(t *testing.T) {
	t.Parallel()

	for name, tc := range agingBreakages {
		broken := agingCopy()

		tc.breaks(&broken)
		refuses(t, name, (&Rules{}).liftAging(broken), tc.mentions)
	}
}

// agingBreakages is each way to break the Aging Table's transcription, and
// what the refusal must name.
//
//nolint:gochecknoglobals // a test table too long for its function.
var agingBreakages = map[string]struct {
	breaks   func(*agingTranscription)
	mentions string
}{
	"a row for a characteristic that does not exist": {
		func(a *agingTranscription) { a.rows[0].label = "Wisdom" }, "Wisdom",
	},
	"rows out of the page's order": {
		func(a *agingTranscription) { a.rows[0], a.rows[1] = a.rows[1], a.rows[0] }, "row 1 is Dexterity",
	},
	"a row missing": {
		func(a *agingTranscription) { a.rows = a.rows[:5] }, "5 rows",
	},
	"a cell that is neither a reduction nor a phrase p. 9 prints": {
		func(a *agingTranscription) { a.rows[0].cells[0].text = "−1 (eight)" }, "is not a reduction",
	},
	"a reduction typed with an en dash": {
		func(a *agingTranscription) { a.rows[0].cells[0].text = "–1 (8+)" }, "an en dash is not a minus",
	},
	"a cell that adds rather than reduces": {
		func(a *agingTranscription) { a.rows[0].cells[0].text = "+1 (8+)" }, "is not a reduction",
	},
	"a gap between two cells": {
		func(a *agingTranscription) { a.rows[0].cells[1].from = 9 }, "where term 8 is next",
	},
	"two cells overlapping": {
		func(a *agingTranscription) { a.rows[0].cells[1].from = 7 }, "where term 8 is next",
	},
	"a row's cells stopping short": {
		func(a *agingTranscription) { a.rows[0].cells[2].through = 13 }, "stop before term 14",
	},
	"a cell past the last printed term": {
		func(a *agingTranscription) { a.rows[0].cells[2].through = 15 }, "runs past the last printed term",
	},
	"an age that is not the age its term ends at": {
		func(a *agingTranscription) { a.ages[3] = "45" }, "is not the age term 7 ends at",
	},
	"a plus on an age before the last": {
		func(a *agingTranscription) { a.ages[9] = "70+" }, "carries a plus before the last column",
	},
	"no plus on the last age": {
		func(a *agingTranscription) { a.ages[10] = "74" }, "carries no plus",
	},
	"a term out of sequence": {
		func(a *agingTranscription) { a.terms[2] = "7" }, "does not follow",
	},
	"a cell that runs backwards": {
		func(a *agingTranscription) { a.rows[0].cells[0].through = 3 }, "a cell across terms 4 to 3",
	},
	"no header rows": {
		func(a *agingTranscription) { a.terms, a.ages = nil, nil }, "0 terms and 0 ages",
	},
	"a term that is not a number": {
		func(a *agingTranscription) { a.terms[0] = "four" }, "is not a number",
	},
	"header rows of two lengths": {
		func(a *agingTranscription) { a.ages = a.ages[:10] }, "11 terms and 10 ages",
	},
	"a crisis saving throw that will not parse": {
		func(a *agingTranscription) { a.crisisSaving = "eight" }, "medical crisis",
	},
}

func TestMalformedWeaponsAndNobility(t *testing.T) {
	t.Parallel()

	weapons := weaponsCopy()

	weapons[traveller.Gun] = nil
	refuses(t, "a category with no list", (&Rules{}).liftWeapons(weapons), "no list for Gun Combat")

	weapons = weaponsCopy()
	weapons[traveller.Blade] = append(weapons[traveller.Blade], "Dagger")
	refuses(t, "a weapon listed twice", (&Rules{}).liftWeapons(weapons), "listed twice")

	nobility := slices.Clone(transcribedNobility)
	if len(nobility) != len(traveller.Titles) {
		t.Fatalf("the nobility transcription has %d rows to break, want %d", len(nobility), len(traveller.Titles))
	}

	nobility[0].title = "archduke"
	refuses(t, "a title Book 3 does not print", (&Rules{}).liftNobility(nobility), "archduke")

	nobility = slices.Clone(transcribedNobility[:len(traveller.Titles)-1])
	refuses(t, "a nobility table with a missing rank", (&Rules{}).liftNobility(nobility), "4 rows")
}

func TestMalformedShips(t *testing.T) {
	t.Parallel()

	hulls := transcribedShips

	hulls[traveller.ScoutShip] = 0
	refuses(t, "a hull of no size", (&Rules{}).liftShips(hulls), "not a hull size")
}

// A table broken two ways names the same one every time.
//
// P. 10 makes ranks, commissions and promotions one fact, so a column that
// prints one without the others is refused. Both columns can be wrong at
// once, and the check ranged over a two-entry map to find them - so the same
// file reported commission on one run and promotion on the next, and a reader
// could not tell whether his fix had worked (#53).
//
// Repeated (sameErrorEveryRun) because once proves nothing: a single pass
// through an unordered range agrees with an ordered one about half the time.
//
// Holds: A table broken two ways reports the same one every run.
func TestABrokenServiceNamesTheSameColumnEveryRun(t *testing.T) {
	t.Parallel()

	first := sameErrorEveryRun(t, func() error {
		broken := servicesCopy()

		// Scouts print no ranks, so a commission and a promotion in that
		// column are both wrong at once.
		copyColumn(&broken.commission, traveller.Scouts, traveller.Navy)
		copyColumn(&broken.promotion, traveller.Scouts, traveller.Navy)

		return (&Rules{}).liftPriorService(broken)
	})

	if !strings.Contains(first, "commission") {
		t.Errorf("the refusal names %q; the page prints commission before promotion", first)
	}
}

// sameErrorEveryRun lifts a freshly broken transcription fifty times and fails unless
// every run refuses it with the same error, which it returns. Once proves
// nothing: a lift that ranged over a map would agree with an ordered one
// about half the time, since Go's map order is deliberately unstable.
func sameErrorEveryRun(t *testing.T, lift func() error) string {
	t.Helper()

	const runs = 50

	first := ""

	for range runs {
		err := lift()
		if err == nil {
			t.Fatal("a table broken two ways was lifted")
		}

		if first == "" {
			first = err.Error()

			continue
		}

		if err.Error() != first {
			t.Fatalf("the same broken table reported two different problems:\n  %s\n  %s", first, err.Error())
		}
	}

	return first
}

// Two throws left empty in one column is two faults, and the lift reports the
// one p. 10 prints first on every run.
//
// Holds: A table broken two ways reports the same one every run.
func TestABrokenServiceNamesTheSameRowEveryRun(t *testing.T) {
	t.Parallel()

	got := sameErrorEveryRun(t, func() error {
		broken := servicesCopy()

		blankColumn(&broken.enlistment, traveller.Other)
		blankColumn(&broken.survival, traveller.Other)

		return (&Rules{}).liftPriorService(broken)
	})

	if !strings.Contains(got, "enlistment") {
		t.Errorf("the refusal names %q; p. 10 prints enlistment before survival", got)
	}
}

// Two skill tables with a cell left empty at once is two faults, and the lift
// reports the same one on every run: the table p. 11 prints first.
//
// Holds: A table broken two ways reports the same one every run.
func TestABrokenSkillsTranscriptionNamesTheSameTableEveryRun(t *testing.T) {
	t.Parallel()

	refused := sameErrorEveryRun(t, func() error {
		broken := transcribedSkills

		broken.tables[traveller.ServiceSkills][5][5] = ""
		broken.tables[traveller.PersonalDevelopment][5][5] = ""

		return (&Rules{normalize: map[string]string{}}).liftSkills(broken)
	})

	if !strings.Contains(refused, "Personal Development") {
		t.Errorf("the refusal names %q; p. 11 prints Personal Development before Service Skills", refused)
	}
}
