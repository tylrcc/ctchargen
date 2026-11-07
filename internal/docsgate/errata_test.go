package docsgate_test

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/rules"
	"github.com/philoserf/ctchargen/traveller"
)

// goldens is where the records this gate reads live. They are the tool's own
// output, regenerated from the engine and checked in, so reading them here
// reads what a referee would be handed.
const goldens = "../../chargen/testdata"

// nobility is the Social Standing at which Book 3 p. 22 confers a title, as
// ERRATA.md's E011 states it: "final Social Standing is 11 or greater".
const nobility = 11

// The step names the engine logs its dice under, throws and rolls alike. They
// are the transcript's own words, so this gate is coupled to them - a
// rewording breaks it loudly and names the reading, which is the half of that
// worth having.
const (
	enlistmentThrow = "enlistment,"
	survivalThrow   = "survival"
	reenlistThrow   = "reenlistment"
	promotionThrow  = "promotion"
	agingThrow      = "aging,"
	crisisThrow     = "medical crisis"
	socialThrow     = "Social Standing"
	termStep        = "term "
)

// The two kinds dice arrive as. A throw meets a target and a roll does not,
// and reading a step as the wrong one finds nothing and says nothing (#50).
const (
	aThrow = "throw"
	aRoll  = "roll"
)

// A reading, and the condition ERRATA.md states for it.
//
// The predicate is written from the document's prose, quoted above it, and
// never from the code that does the stamping. A condition transcribed from
// the thing it checks is one reading written twice, which would make this
// gate decorative (CLAUDE.md, and #77).
type reading struct {
	id    traveller.Erratum
	quote string
	holds func(record) bool

	// parked names the issue where this reading's document and its code
	// disagree and neither has yet been ruled wrong. A parked reading is
	// measured but not judged: shipping it as a failure would make the gate
	// red for something undecided, and dropping it would hide the
	// disagreement the gate exists to surface.
	//
	// disagreeOn is how many records they differ on. It is asserted, so the
	// predicate keeps running and the number stays true.
	parked     string
	disagreeOn int
}

// Every reading a record stamps is one whose condition it satisfies, and
// every condition it satisfies is stamped.
//
// TestEveryReadingIsReachable proves each reading is reachable from some
// path. It does not prove any reading is stamped on the RIGHT records: a
// reading applied one term too early, or to a civilian who never had a term,
// passes it. Both directions are checked here, because an erratum stamped
// that should not be is as much a defect as one missing - a record's errata
// list is the tool's claim about which readings its character rests on.
func TestEveryReadingIsStampedWhereItsConditionHolds(t *testing.T) {
	t.Parallel()

	tables, err := rules.Load()
	if err != nil {
		t.Fatalf("loading the rules: %v", err)
	}

	readings := conditions(tables)

	oneConditionEach(t, readings)

	stampedSomewhere := make(map[traveller.Erratum]bool)
	disagreements := make(map[traveller.Erratum]int)

	for _, rec := range readRecords(t) {
		for _, r := range readings {
			want := r.holds(rec)

			got := slices.Contains(rec.Errata, r.id.String())
			if got {
				stampedSomewhere[r.id] = true
			}

			// A parked reading is measured and not judged. Skipping the
			// predicate outright would leave it compiled and never run, and
			// a predicate nobody runs rots: rename a step and it quietly
			// starts answering about nothing.
			if r.parked != "" {
				if want != got {
					disagreements[r.id]++
				}

				continue
			}

			switch {
			case want && !got:
				t.Errorf("%s: satisfies %v but does not stamp it — %s", rec.name, r.id, r.quote)
			case got && !want:
				t.Errorf("%s: stamps %v but does not satisfy it — %s", rec.name, r.id, r.quote)
			}
		}
	}

	stillParked(t, readings, disagreements)
	unreached(t, readings, stampedSomewhere)
}

// Every step name the conditions read finds events in the roster.
//
// A predicate keyed on a step that matches nothing does not fail - it answers
// "no" about every record, quietly, and the gate stays green while a reading
// goes unchecked. That is the failure this whole file exists to catch, and it
// happened here: splitting the event kinds (#50) made "Social Standing" a
// roll rather than a throw, E011's predicate started reading nothing, and
// nothing said so, because the count it is measured by did not move.
//
// The pairing is asserted too, since a step is read as one kind or the other
// and reading it as the wrong one is exactly how the above went unnoticed.
func TestEveryStepTheConditionsReadIsFound(t *testing.T) {
	t.Parallel()

	records := readRecords(t)

	for step, kind := range map[string]string{
		enlistmentThrow: aThrow,
		survivalThrow:   aThrow,
		reenlistThrow:   aThrow,
		promotionThrow:  aThrow,
		agingThrow:      aThrow,
		crisisThrow:     aThrow,
		socialThrow:     aRoll,
	} {
		found := false

		for _, rec := range records {
			if len(rec.dice(step, kind)) > 0 {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("no record has a %q %s; a condition reading it answers no about everything",
				step, kind)
		}
	}

	// firstTotal is what E011 reads the rolled Social Standing with, and the
	// step-and-kind check above cannot see which kind it asks for. Asking
	// for the wrong one returns (0, false) about every record, silently,
	// which is exactly how the split shipped green.
	for _, rec := range records {
		if len(rec.rolls(socialThrow)) == 0 {
			continue
		}

		rolled, found := rec.firstTotal(socialThrow)
		if !found || rolled < 2 || rolled > 12 {
			t.Errorf("%s: firstTotal read %d (found %v) for a Social Standing roll",
				rec.name, rolled, found)
		}
	}

	// The term step is read by E003 and E013 and is not a throw of any kind.
	opened := false

	for _, rec := range records {
		for _, e := range rec.Events {
			if e.Kind == "step" && e.term > 0 {
				opened = true
			}
		}
	}

	if !opened {
		t.Error("no record opens a term; E003 and E013 both read the term an event fell in")
	}
}

// Every entry of ERRATA.md has exactly one condition, and every condition
// names an entry.
//
// This was a comparison of lengths, which is not the same claim and does not
// hold it: giving one reading another's id leaves the count right, compares
// that id twice against agreeing predicates, and drops the other reading
// entirely - and the gate stayed green. Adding a sixteenth erratum and
// copying a block without changing its id is the way that happens for real.
func oneConditionEach(t *testing.T, readings []reading) {
	t.Helper()

	conditions := make(map[traveller.Erratum]int)
	for _, r := range readings {
		conditions[r.id]++
	}

	for _, id := range traveller.Errata {
		switch conditions[id] {
		case 1:
		case 0:
			t.Errorf("%v has no condition; every entry of ERRATA.md needs one", id)
		default:
			t.Errorf("%v has %d conditions, so some other reading has none", id, conditions[id])
		}

		delete(conditions, id)
	}

	for id := range conditions {
		t.Errorf("%v has a condition but is not one of traveller.Errata", id)
	}
}

// The readings this gate lists but does not compare, pinned so the list
// cannot grow quietly.
//
// Parking one is how a disagreement between a reading and its code gets
// recorded without being decided in passing. It is meant to be temporary and
// nothing but this assertion would say if it were not.
func stillParked(t *testing.T, readings []reading, disagreements map[traveller.Erratum]int) {
	t.Helper()

	var want []string

	var got []string

	for _, r := range readings {
		if r.parked == "" {
			continue
		}

		got = append(got, r.id.String())

		// The disagreement is counted, not just noted, so the number in the
		// reading's own comment is re-measured every run. Settling the issue
		// takes it to zero, which fails here saying so - which is the right
		// way for a parked reading to ask to be unparked.
		if n := disagreements[r.id]; n != r.disagreeOn {
			t.Errorf("%v is parked as %s over %d records, and disagrees on %d now",
				r.id, r.parked, r.disagreeOn, n)
		}
	}

	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("%v are parked, and %v were when this was last decided", got, want)
	}
}

// The readings no golden record stamps, pinned so the gap is visible.
//
// Both are correct and permanent: E012 and E015 say "No record" in so many
// words, and this gate is the only thing that would notice if one were
// stamped anyway. E003 and E014 were on this list while the roster was seeds,
// because no seed produces the run of 12s they turn on; since #205 the roster
// scripts its dice and two goldens carry them.
//
// The set is asserted rather than logged so that a change in either
// direction is a finding: a golden that stamps one of these is wrong, and a
// reading that quietly stops appearing should not go unseen.
func unreached(t *testing.T, readings []reading, stamped map[traveller.Erratum]bool) {
	t.Helper()

	want := []string{"E012", "E015"}

	var got []string

	for _, r := range readings {
		if !stamped[r.id] {
			got = append(got, r.id.String())
		}
	}

	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("no golden stamps %v; the roster reached %v last time this was checked", got, want)
	}
}

// conditions is ERRATA.md's fifteen "Stamped on" lines as predicates over a
// finished record. Each quote is the document's own sentence.
func conditions(tables *rules.Rules) []reading {
	return slices.Concat(
		enlistmentAndService(),
		agingAndCrisis(),
		rankAndTitle(tables),
	)
}

func enlistmentAndService() []reading {
	return []reading{
		{
			id: traveller.E001,
			quote: "records where the enlistment throw failed, whether he then submitted to " +
				"the draft or declined",
			holds: func(r record) bool { return r.threw(enlistmentThrow, failed) },
		},
		{
			id:    traveller.E002,
			quote: "records in which at least one term proceeded past its survival throw",
			holds: func(r record) bool { return r.threw(survivalThrow, passed) },
		},
		{
			id:    traveller.E003,
			quote: "records in which a 12 was thrown on a reenlistment throw at the end of term 8 or later",
			holds: func(r record) bool {
				const firstRecurrence = 8

				for _, e := range r.throws(reenlistThrow) {
					if e.Total == twelve && e.term >= firstRecurrence {
						return true
					}
				}

				return false
			},
		},
		{
			id:    traveller.E004,
			quote: "records ending in death by a failed survival throw",
			holds: func(r record) bool { return r.departedBy("killed by the survival throw") },
		},
		{
			id:    traveller.E005,
			quote: "records in a service with a service-wide entry: Marines, Army, Scouts",
			holds: func(r record) bool {
				return slices.Contains([]string{"Marines", "Army", "Scouts"}, r.Service)
			},
		},
	}
}

func agingAndCrisis() []reading {
	// E006 and E007 name the same records, and E008, E009 and E010 name
	// another same three. ERRATA.md says so itself: E007 is "stamped
	// alongside E006", E009 alongside E008, and E010's set is "every
	// reduction that reaches 0, and so every medical crisis that aging
	// caused" - which during generation is every crisis there is, since
	// nothing else takes a characteristic to zero.
	ranAnAgingRound := func(r record) bool { return r.threw(agingThrow, either) }
	madeACrisisThrow := func(r record) bool { return r.threw(crisisThrow, either) }

	return []reading{
		{
			id:    traveller.E006,
			quote: "records in which at least one aging round was run",
			holds: ranAnAgingRound,
		},
		{
			id:    traveller.E007,
			quote: "records in which at least one aging round was run",
			holds: ranAnAgingRound,
		},
		{
			id:    traveller.E008,
			quote: "records in which a medical-crisis saving throw was made, whether it succeeded or failed",
			holds: madeACrisisThrow,
		},
		{
			id:    traveller.E009,
			quote: "records in which a medical-crisis saving throw was made",
			holds: madeACrisisThrow,
		},
		{
			id: traveller.E010,
			quote: "records in which an aging reduction carried a characteristic below 1, " +
				"which is every medical crisis that aging caused",
			holds: madeACrisisThrow,
		},
	}
}

func rankAndTitle(tables *rules.Rules) []reading {
	return []reading{
		{
			// Parked while #92 was open, and compared since it was settled:
			// the document was right and assessTitle over-stamped, naming
			// the reading on nine records it did nothing to.
			id: traveller.E011,
			quote: "records where eligibility on the rolled Social Standing and on the final one " +
				"differ in either direction, and every record ending in death whose final " +
				"Social Standing is 11 or more",
			holds: func(r record) bool {
				rolled, threw := r.firstTotal(socialThrow)
				final := r.Characteristics[socialThrow]

				if r.died() && final >= nobility {
					return true
				}

				return threw && (rolled >= nobility) != (final >= nobility)
			},
		},
		{
			id:    traveller.E012,
			quote: "no record: a spelling is a transcription, not a reading",
			holds: func(record) bool { return false },
		},
		{
			id: traveller.E013,
			quote: "records in which a character reached the highest rank his service prints and " +
				"then proceeded past the survival throw of at least one further term",
			holds: func(r record) bool { return reachedTheTopAndServedOn(r, tables) },
		},
		{
			id:    traveller.E014,
			quote: "records with fifteen or more terms of service",
			holds: func(r record) bool {
				const pastThePrintedTerms = 15

				return r.Terms >= pastThePrintedTerms
			},
		},
		{
			id:    traveller.E015,
			quote: "no record: nothing but the worked example claims a result the table does not print",
			holds: func(record) bool { return false },
		},
	}
}

// reachedTheTopAndServedOn is E011's neighbour and the one condition that
// needs a table: which rank is the highest a service prints. That comes from
// the lifted Table of Ranks, not from the stamping code - it is a fact of
// p. 10, and reading it here is reading the page, not the answer.
func reachedTheTopAndServedOn(r record, tables *rules.Rules) bool {
	service, named := serviceNamed(r.Service)
	if !named {
		return false
	}

	// A service printing no ranks has no highest rank to reach. Scouts and
	// Other print none, and an unranked character's rank is 0 - so without
	// this every one of them "reached the top", which is what the first run
	// of this gate reported.
	top := int(tables.Service(service).MaxRank())
	if top == 0 || r.Rank != top {
		return false
	}

	// The term he arrived at that rank is the term of his last successful
	// promotion; "one further term" is a survival throw passed after it.
	arrived := 0

	for _, e := range r.throws(promotionThrow) {
		if e.Succeeded {
			arrived = e.term
		}
	}

	for _, e := range r.throws(survivalThrow) {
		if e.Succeeded && e.term > arrived {
			return true
		}
	}

	return false
}

func serviceNamed(printed string) (traveller.ServiceName, bool) {
	for _, name := range traveller.ServiceNames {
		if name.String() == printed {
			return name, true
		}
	}

	var none traveller.ServiceName

	return none, false
}

// Every choice point the engine offers is reached by some golden record.
//
// POLICY.md has a row per Decider method and docsgate holds the two to each
// other, so the document cannot describe a method that does not exist. Until
// #34 nothing held either to the roster, and Weapon was the result: eleven of
// the twelve points were exercised by a golden and that one was not, so what
// POLICY.md said about it was covered by no character at all - before it
// changed and after.
//
// A row nothing reaches is the same defect as a rule with no test, one level
// up: the document, the code and the gate all agree, and none of them has
// seen the answer.
func TestEveryChoicePointIsReachedByARecord(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}

	for _, rec := range readRecords(t) {
		for _, e := range rec.Events {
			if e.Kind == "choice" {
				reached[e.Point] = true
			}
		}
	}

	if len(reached) == 0 {
		t.Fatal("no record answers any choice point")
	}

	for _, point := range traveller.ChoicePoints {
		if !reached[point.String()] {
			t.Errorf("no golden record reaches %v, so what POLICY.md answers there is untested",
				point)
		}
	}
}
