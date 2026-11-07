package rules //nolint:testpackage // the lift's guards are unexported

import (
	"errors"
	"strings"
	"testing"
)

// A malformed table must fail loudly, and must say which cell.
//
// Every case below starts from the real data file and breaks exactly one
// thing, so what is under test is the guard rather than a hand-built fixture
// that resembles the data. The breakages are the ones the reprint's font can
// actually produce, plus the ones a careless edit can.

func mustRead[T any](t *testing.T, name string) T {
	t.Helper()

	wire, err := read[T](name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return wire
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
		breaks   func(*wireServices)
		mentions string
	}{
		"a missing column": {
			func(w *wireServices) { w.Services = w.Services[:5] },
			"columns",
		},
		// The count has to be checked before the order is, because the
		// order comparison indexes the domain's own array by column.
		"an extra column": {
			func(w *wireServices) { w.Services = append(w.Services, "Navy") },
			"columns",
		},
		"columns out of book order": {
			func(w *wireServices) { w.Services[0], w.Services[1] = w.Services[1], w.Services[0] },
			"want Navy",
		},
		"a row with a missing cell": {
			func(w *wireServices) { w.Survival = w.Survival[:5] },
			"survival",
		},
		"a target that will not parse": {
			func(w *wireServices) { w.Enlistment[0].Target = "eight" },
			"enlistment",
		},
		"a reenlist target that will not parse": {
			func(w *wireServices) { w.Reenlist[0] = "" },
			"reenlist",
		},
		"a commission printed for a service with no ranks": {
			func(w *wireServices) { w.Commission[3] = w.Commission[0] },
			"p. 10 makes ranks",
		},
		"a commission missing from a service with ranks": {
			func(w *wireServices) { w.Commission[0] = nil },
			"p. 10 makes ranks",
		},
		// Draft scans for the service printing the die, so a face printed
		// twice leaves another face naming nobody (#134).
		"a draft face printed twice": {
			func(w *wireServices) { w.Draft[5] = w.Draft[4] },
			"printed twice",
		},
		"a draft number that is not a die face": {
			func(w *wireServices) { w.Draft[0] = 7 },
			"not a die face",
		},
	} {
		wire := mustRead[wireServices](t, "services.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{}).liftPriorService(wire), tc.mentions)
	}
}

// retirementFixture lifts the real prior service table and hands back the
// wire beside it, so a case can break exactly one row of one table.
func retirementFixture(t *testing.T) (*Rules, wireServices) {
	t.Helper()

	wire := mustRead[wireServices](t, "services.json")

	r := &Rules{normalize: map[string]string{}}

	err := r.liftPriorService(wire)
	if err != nil {
		t.Fatalf("lifting the prior service table: %v", err)
	}

	// The cases below break one row of each of these, so a fixture that
	// quietly lost them would make them vacuous rather than failing.
	if len(wire.RankAndServiceSkills) == 0 || len(wire.RetirementPay.ByTerms) < 2 ||
		len(wire.RetirementPay.PaidBy) == 0 {
		t.Fatalf("the fixture is too thin to break: %d grants, %d pension rows, %d payers",
			len(wire.RankAndServiceSkills), len(wire.RetirementPay.ByTerms),
			len(wire.RetirementPay.PaidBy))
	}

	return r, wire
}

// The Rank and Service Skills box keys each grant to a service and a rank.
func TestMalformedGrants(t *testing.T) {
	t.Parallel()

	r, wire := retirementFixture(t)

	wire.RankAndServiceSkills[0].Service = "Marine"
	refuses(t, "a grant to a service that does not exist", r.liftGrants(wire), "Marine")

	r, wire = retirementFixture(t)
	wire.RankAndServiceSkills[0].Rank = 9
	refuses(t, "a grant at a rank past the table", r.liftGrants(wire), "past the service's table of ranks")

	r, wire = retirementFixture(t)
	wire.RankAndServiceSkills[0].Grant = "+1 Charisma"
	refuses(t, "a grant of something that is not a result", r.liftGrants(wire), "Charisma")
}

// The retirement pay table validates its own shape, so break it six ways.
func TestMalformedRetirement(t *testing.T) {
	t.Parallel()

	r, wire := retirementFixture(t)

	wire.RetirementPay.PaidBy[0] = "Merchant"
	refuses(t, "a pension paid by a service that does not exist", r.liftRetirement(wire), "Merchant")

	r, wire = retirementFixture(t)
	wire.RetirementPay.ByTerms[1].Pay = 0
	refuses(t, "a pension row that pays nothing", r.liftRetirement(wire), "is not a pension")

	r, wire = retirementFixture(t)
	wire.RetirementPay.ByTerms[0], wire.RetirementPay.ByTerms[1] =
		wire.RetirementPay.ByTerms[1], wire.RetirementPay.ByTerms[0]
	refuses(t, "pension rows out of order", r.liftRetirement(wire), "the table's rows ascend")

	r, wire = retirementFixture(t)
	wire.RetirementPay.PerTermBeyondEight = 0
	refuses(t, "nothing paid past the table", r.liftRetirement(wire), "per additional term")

	// The two cases that empty a slice come last: nothing may index these
	// fixtures afterwards.
	r, wire = retirementFixture(t)
	wire.RetirementPay.PaidBy = nil
	refuses(t, "a pension no service pays", r.liftRetirement(wire), "no service pays it")

	r, wire = retirementFixture(t)
	wire.RetirementPay.ByTerms = nil
	refuses(t, "a pension table with no rows", r.liftRetirement(wire), "prints no rows")
}

func TestMalformedSkills(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*wireSkills)
		mentions string
	}{
		"a missing table": {
			func(w *wireSkills) { delete(w.Tables, "Service Skills Table") },
			"3 tables",
		},
		"a table the type does not name": {
			func(w *wireSkills) {
				w.Tables["Basic Training"] = w.Tables["Service Skills Table"]
				delete(w.Tables, "Service Skills Table")
			},
			"Basic Training",
		},
		"a missing row": {
			func(w *wireSkills) {
				w.Tables["Service Skills Table"] = w.Tables["Service Skills Table"][:5]
			},
			"5 rows",
		},
		"a cell that alters a characteristic that does not exist": {
			func(w *wireSkills) { w.Tables["Service Skills Table"][0][0] = "+1 Charisma" },
			"Charisma",
		},
		// A signed cell that is not an alteration is refused outright.
		// Letting it through would accumulate it as a skill whose name is
		// the whole cell - a wrong value that looks right on screen, which
		// is the failure this package exists to prevent.
		"a signed cell that is not an alteration": {
			func(w *wireSkills) { w.Tables["Service Skills Table"][0][0] = "+x Social Standing" },
			"begins with a sign",
		},
		"an education gate on a table that does not exist": {
			func(w *wireSkills) { w.EducationGate.Table = "Basic Training" },
			"education gate",
		},
		"an education gate threshold that will not parse": {
			func(w *wireSkills) { w.EducationGate.Threshold = "eight" },
			"education gate",
		},
		"an education gate on a characteristic that does not exist": {
			func(w *wireSkills) { w.EducationGate.Characteristic = "Wisdom" },
			"education gate",
		},
	} {
		wire := mustRead[wireSkills](t, "skills.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftSkills(wire), tc.mentions)
	}
}

func TestMalformedMustering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*wireMustering)
		mentions string
	}{
		"a missing row": {
			func(w *wireMustering) { w.Table1 = w.Table1[:6] },
			"rows",
		},
		// The font trap, lifted: p. 9's dash cells extract as the digit 4.
		"a dash cell that extracted as a 4": {
			func(w *wireMustering) { w.Table1[6][3] = "4" },
			"not a benefit",
		},
		"a benefit no row prints": {
			func(w *wireMustering) { w.Table1[0][0] = "Yacht" },
			"Yacht",
		},
		"a cash row with a missing cell": {
			func(w *wireMustering) { w.Table2[0] = w.Table2[0][:5] },
			"table 2",
		},
		"a cash allowance of nothing": {
			func(w *wireMustering) { w.Table2[0][0] = 0 },
			"not a cash allowance",
		},
		"no skill earning the table 2 modifier": {
			func(w *wireMustering) { w.Rolls.Table2ModifierFrom = "" },
			"earns the table 2 modifier",
		},
		"a passage with no price": {
			func(w *wireMustering) { delete(w.Passages.Prices, "Low Passage") },
			"no price for Low Passage",
		},
		// There is no case here for a price that is not a number. It used to
		// be one: the prices shared an object with resalePercent, so the
		// field was map[string]any and the lift asserted its way back to a
		// float. Prices are map[string]int64 now, so "cheap" is refused by
		// json.Unmarshal before the lift is called, and what the compiler
		// proves is not also tested (#49).
		"no resale percentage": {
			func(w *wireMustering) { w.Passages.ResalePercent = 0 },
			"resale",
		},
	} {
		wire := mustRead[wireMustering](t, "mustering.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftMustering(wire), tc.mentions)
	}
}

func TestMalformedAging(t *testing.T) {
	t.Parallel()

	valid := func(t *testing.T) wireAging {
		t.Helper()

		return mustRead[wireAging](t, "aging.json")
	}

	wire := valid(t)

	wire.Bands[0].Effects[0].Characteristic = "Wisdom"
	refuses(t, "a band reducing a characteristic that does not exist", (&Rules{}).liftAging(wire), "Wisdom")

	wire = valid(t)
	wire.Bands[0].Effects[0].Saving = "eight"
	refuses(t, "a saving throw that will not parse", (&Rules{}).liftAging(wire), "aging band")

	wire = valid(t)
	wire.Bands[0], wire.Bands[2] = wire.Bands[2], wire.Bands[0]
	refuses(t, "bands out of order", (&Rules{}).liftAging(wire), "out of order")

	wire = valid(t)
	wire.Bands = nil
	refuses(t, "no bands at all", (&Rules{}).liftAging(wire), "no bands")

	wire = valid(t)
	wire.LastPrintedTerm = 0
	refuses(t, "a last printed term before the last band", (&Rules{}).liftAging(wire), "last printed term")

	wire = valid(t)
	wire.Crisis.Saving = "eight"
	refuses(t, "a crisis saving throw that will not parse", (&Rules{}).liftAging(wire), "medical crisis")
}

func TestMalformedWeaponsAndNobility(t *testing.T) {
	t.Parallel()

	wire := mustRead[wireWeapons](t, "weapons.json")
	delete(wire.Lists, "Gun Combat")
	refuses(t, "a category with no list", (&Rules{}).liftWeapons(wire), "no list for Gun Combat")

	wire = mustRead[wireWeapons](t, "weapons.json")
	wire.Lists["Blade Combat"] = append(wire.Lists["Blade Combat"], "Dagger")
	refuses(t, "a weapon listed twice", (&Rules{}).liftWeapons(wire), "listed twice")

	nobility := mustRead[wireNobility](t, "nobility.json")

	nobility.Titles[0].Title = "archduke"
	refuses(t, "a title Book 3 does not print", (&Rules{}).liftNobility(nobility), "archduke")

	nobility = mustRead[wireNobility](t, "nobility.json")
	nobility.Titles = nobility.Titles[:4]
	refuses(t, "a nobility table with a missing rank", (&Rules{}).liftNobility(nobility), "4 rows")
}

func TestMalformedShips(t *testing.T) {
	t.Parallel()

	wire := mustRead[wireShips](t, "ships.json")

	wire.Ships[0].Kind = "Yacht"
	refuses(t, "a ship Book 1 p. 22 does not name", (&Rules{}).liftShips(wire), "Yacht")

	wire = mustRead[wireShips](t, "ships.json")
	wire.Ships[0].HullTons = 0
	refuses(t, "a hull of no size", (&Rules{}).liftShips(wire), "not a hull size")

	wire = mustRead[wireShips](t, "ships.json")
	wire.Ships[1].Kind = wire.Ships[0].Kind
	refuses(t, "one ship listed twice", (&Rules{}).liftShips(wire), "listed twice")

	wire = mustRead[wireShips](t, "ships.json")
	wire.Ships = wire.Ships[:1]
	refuses(t, "a ship with no hull at all", (&Rules{}).liftShips(wire), "no hull for")
}

// A data file broken two ways names the same one every time.
//
// P. 10 makes ranks, commissions and promotions one fact, so a column that
// prints one without the others is refused. Both columns can be wrong at
// once, and the check ranged over a two-entry map to find them - so the same
// file reported commission on one run and promotion on the next, and a reader
// could not tell whether his fix had worked (#53).
//
// Repeated because once proves nothing: Go's map order is deliberately
// unstable, and a single pass through an unordered range agrees with an
// ordered one about half the time.
//
// Holds: A data file broken two ways reports the same one every run.
func TestABrokenServiceNamesTheSameColumnEveryRun(t *testing.T) {
	t.Parallel()

	const runs = 50

	first := ""

	for range runs {
		wire := mustRead[wireServices](t, "services.json")

		// Scouts print no ranks, so a commission and a promotion in that
		// column are both wrong at once.
		const scouts = 3

		wire.Commission[scouts] = wire.Commission[0]
		wire.Promotion[scouts] = wire.Promotion[0]

		err := (&Rules{}).liftPriorService(wire)
		if err == nil {
			t.Fatal("a service printing a commission and a promotion without ranks was lifted")
		}

		if first == "" {
			first = err.Error()

			continue
		}

		if err.Error() != first {
			t.Fatalf("the same broken file reported two different problems:\n  %s\n  %s",
				first, err.Error())
		}
	}

	if !strings.Contains(first, "commission") {
		t.Errorf("the refusal names %q; the page prints commission before promotion", first)
	}
}
