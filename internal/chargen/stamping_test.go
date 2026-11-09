package chargen_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The stamping predicates, and the checks that the roster reaches every
// choice point. They read the record each fixture writes, the bytes a
// referee would be handed, as the engine writes them now: TestGoldens holds
// those to the checked-in goldens, and reading them fresh lets an engine
// change be named here by what it did (#373).

// nobility is the Social Standing at which Book 3 p. 22 confers a title, as
// E011's reading states it: "final Social Standing is 11 or
// greater".
const nobility = 11

// The step names the engine logs its dice under, throws and rolls alike. They
// are the transcript's own words, so these checks are coupled to them - a
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

// A reading, and the condition it states.
//
// The predicate is written by hand from the reading's stamped-on prose, and
// never from the code that does the stamping. A condition transcribed from
// the thing it checks is one reading written twice, which would make the
// check decorative (CLAUDE.md, and #77). A failure quotes the prose itself,
// as rules declares it, rather than a copy typed beside the predicate; the
// comments below quote it only where a predicate needs its reason shown.
type reading struct {
	id    traveller.Erratum
	holds func(record) bool

	// parked names the issue where this reading's prose and its code
	// disagree and neither has yet been ruled wrong. A parked reading is
	// measured but not judged: shipping it as a failure would make the gate
	// red for something undecided, and dropping it would hide the
	// disagreement the check exists to surface.
	//
	// disagreeOn is how many records they differ on. It is asserted, so the
	// predicate keeps running and the number stays true.
	parked     string
	disagreeOn int

	// withdrawn is whether the reading is withdrawn, as rules declares it,
	// which changes what verdict holds a record to.
	withdrawn bool
}

// Every reading a record stamps is one whose condition it satisfies, and
// every condition it satisfies is stamped, unless the reading is withdrawn:
// a withdrawn reading is stamped on no record, and its condition is not
// judged (#332).
//
// Reaching some golden is not enough: a reading applied one term too early,
// or to a civilian who never had a term, would still reach one. So both
// directions are checked record by record, because an erratum stamped that
// should not be is as much a defect as one missing - a record's errata list
// is the tool's claim about which readings its character rests on - and
// unreached then holds that every reading reaches some golden at all.
//
// Holds: E011 is stamped only where the reading changed the answer.
// Holds: The errata a record stamps are exactly those whose reading's condition
// holds.
// Holds: Every reading that governed a record is named on it.
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
			if r.parked != "" && !r.withdrawn {
				if want != got {
					disagreements[r.id]++
				}

				continue
			}

			for _, wrong := range verdict(rec, r) {
				t.Error(wrong)
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
// goes unchecked. That is the failure these checks exist to catch, and it
// happened here: splitting the event kinds (#50) made "Social Standing" a
// roll rather than a throw, E011's predicate started reading nothing, and
// nothing said so, because the count it is measured by did not move.
//
// The pairing is asserted too, since a step is read as one kind or the other
// and reading it as the wrong one is exactly how the above went unnoticed.
//
// Holds: Every step the errata conditions read is found, as the kind they read
// it as.
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

// Every reading has exactly one condition, and every condition names a
// reading.
//
// This was a comparison of lengths, which is not the same claim and does not
// hold it: giving one reading another's id leaves the count right, compares
// that id twice against agreeing predicates, and drops the other reading
// entirely - and the check stayed green. Adding a sixteenth erratum and
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
			t.Errorf("%v has no condition; every reading needs one", id)
		default:
			t.Errorf("%v has %d conditions, so some other reading has none", id, conditions[id])
		}

		delete(conditions, id)
	}

	for id := range conditions {
		t.Errorf("%v has a condition but is not one of traveller.Errata", id)
	}
}

// The readings this check lists but does not compare, pinned so the list
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

// namesNoRecord is the readings no golden stamps by design, and the one
// list of them.
//
// Both are correct and permanent: E012 is a spelling, which is a
// transcription rather than a reading and changes nothing about a character;
// E015 settles the worked example against the table it illustrates, and only
// the reproduction of that example depends on it. Each says "No record" in its own
// stamped-on prose. E003 and E014 were on this list while fixtures were
// seeds found by search, because no seed produced the run of 12s they turn
// on; since #205 the roster scripts its dice and two goldens carry them.
func namesNoRecord() []traveller.Erratum {
	return []traveller.Erratum{traveller.E012, traveller.E015}
}

// unreached holds the readings no golden stamps to namesNoRecord: every
// reading not withdrawn and not pinned there is stamped on some golden, and
// one that reaches none is named. It is the one check of that (#373). A
// withdrawn reading is stamped on none by definition, and verdict holds it;
// it is not one of these.
//
// The set is asserted rather than logged so that a change in either
// direction is a finding: a golden that stamps one of these is wrong, and a
// reading that quietly stops appearing should not go unseen.
func unreached(t *testing.T, readings []reading, stamped map[traveller.Erratum]bool) {
	t.Helper()

	var got []traveller.Erratum

	for _, r := range readings {
		if !stamped[r.id] && !r.withdrawn {
			got = append(got, r.id)
		}
	}

	slices.Sort(got)

	if want := namesNoRecord(); !slices.Equal(got, want) {
		t.Errorf("no golden stamps %v; namesNoRecord pins %v", got, want)
	}
}

// stampedOn is a reading's stamped-on prose as its reading states it, on one
// line.
func stampedOn(id traveller.Erratum) string {
	return strings.Join(strings.Fields(rules.Readings()[id-1].StampedOn), " ")
}

// conditions is the readings' stamped-on prose as predicates over a finished
// record, one per reading.
//
// Implements: The errata a record stamps are exactly those whose reading's
// condition holds [Documents].
func conditions(tables *rules.Rules) []reading {
	all := slices.Concat(
		enlistmentAndService(),
		agingAndCrisis(),
		rankAndTitle(tables),
	)

	for i := range all {
		all[i].withdrawn = rules.Readings()[all[i].id-1].Withdrawn()
	}

	return all
}

// verdict is what is wrong with how a record stamps one reading, if anything.
// A standing reading is stamped exactly where its condition holds. A
// withdrawn one is stamped on no record, and that is all it is held to: the
// goldens are new records, and this is what holds that no new record carries
// a withdrawn reading. Its condition is not judged, since a reading replaced
// by another still describes records the replacement governs (#332). A record
// written before the withdrawal keeps its stamp, and nothing here reads it.
//
// A parked reading is not judged here unless it is withdrawn, which a parked
// disagreement does not excuse.
//
// Implements: The errata a record stamps are exactly those whose reading's
// condition holds [Documents].
//
// Implements: A withdrawn reading is stamped on no new record [Documents].
func verdict(rec record, r reading) []string {
	got := slices.Contains(rec.Errata, r.id.String())

	if r.withdrawn {
		if got {
			return []string{fmt.Sprintf("%s: stamps %v, which is withdrawn", rec.name, r.id)}
		}

		return nil
	}

	want := r.holds(rec)

	switch {
	case want && !got:
		return []string{fmt.Sprintf("%s: satisfies %v but does not stamp it. Stamped on: %s",
			rec.name, r.id, stampedOn(r.id))}
	case got && !want:
		return []string{fmt.Sprintf("%s: stamps %v but does not satisfy it. Stamped on: %s",
			rec.name, r.id, stampedOn(r.id))}
	}

	return nil
}

func enlistmentAndService() []reading {
	return []reading{
		{
			id:    traveller.E001,
			holds: func(r record) bool { return r.threw(enlistmentThrow, failed) },
		},
		{
			id:    traveller.E002,
			holds: func(r record) bool { return r.threw(survivalThrow, passed) },
		},
		{
			id: traveller.E003,
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
			holds: func(r record) bool { return r.departedBy("killed by the survival throw") },
		},
		{
			id: traveller.E005,
			holds: func(r record) bool {
				return slices.Contains([]string{"Marines", "Army", "Scouts"}, r.Service)
			},
		},
	}
}

func agingAndCrisis() []reading {
	// E006 and E007 name the same records, and E008, E009 and E010 name
	// another same three. The readings say so themselves: E007 is "stamped
	// alongside E006", E009 alongside E008, and E010's set is "every
	// reduction that reaches 0, and so every medical crisis that aging
	// caused" - which during generation is every crisis there is, since
	// nothing else takes a characteristic to zero.
	ranAnAgingRound := func(r record) bool { return r.threw(agingThrow, either) }
	madeACrisisThrow := func(r record) bool { return r.threw(crisisThrow, either) }

	return []reading{
		{
			id:    traveller.E006,
			holds: ranAnAgingRound,
		},
		{
			id:    traveller.E007,
			holds: ranAnAgingRound,
		},
		{
			id:    traveller.E008,
			holds: madeACrisisThrow,
		},
		{
			id:    traveller.E009,
			holds: madeACrisisThrow,
		},
		{
			id:    traveller.E010,
			holds: madeACrisisThrow,
		},
	}
}

func rankAndTitle(tables *rules.Rules) []reading {
	return []reading{
		{
			// Parked while #92 was open, and compared since it was settled:
			// the reading was right and assessTitle over-stamped, naming
			// the reading on nine records it did nothing to.
			id: traveller.E011,
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
			holds: func(record) bool { return false },
		},
		{
			id:    traveller.E013,
			holds: func(r record) bool { return reachedTheTopAndServedOn(r, tables) },
		},
		{
			id: traveller.E014,
			holds: func(r record) bool {
				const pastThePrintedTerms = 15

				return r.Terms >= pastThePrintedTerms
			},
		},
		{
			id:    traveller.E015,
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
	// of this check reported.
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

// A withdrawn reading is stamped on no new record: the goldens are what this
// build writes, so a golden that stamps one is a record stamped with a
// decision that no longer governs. Its condition is not judged. A withdrawn
// reading keeps the prose its predicate is written from, and a reading
// replaced by another still describes records the replacement now governs
// (#332).
//
// No real reading is withdrawn, so E001 stands in, under its real predicate:
// every golden that stamps it is refused, and each of those goldens with E001
// struck from its errata passes. A struck copy is not what a build that had
// withdrawn E001 would write, which would walk other paths; it stands in for
// the case #332 is about, a record E001's prose still describes and that does
// not carry it, and the test checks that the prose does describe it.
//
// Holds: A withdrawn reading is stamped on no new record.
func TestAWithdrawnReadingOnAGoldenIsRefused(t *testing.T) {
	t.Parallel()

	tables, err := rules.Load()
	if err != nil {
		t.Fatalf("loading the rules: %v", err)
	}

	var e001 reading

	for _, r := range conditions(tables) {
		if r.id == traveller.E001 {
			e001 = r
		}
	}

	if e001.holds == nil {
		t.Fatal("E001 has no condition")
	}

	e001Withdrawn := e001

	e001Withdrawn.withdrawn = true

	stamped := 0

	for _, rec := range readRecords(t) {
		if !slices.Contains(rec.Errata, traveller.E001.String()) {
			continue
		}

		stamped++

		refused := slices.ContainsFunc(verdict(rec, e001Withdrawn), func(wrong string) bool {
			return strings.Contains(wrong, "which is withdrawn")
		})
		if !refused {
			t.Errorf("%s stamps E001, withdrawn, and was not refused", rec.name)
		}

		struck := rec

		struck.Errata = slices.DeleteFunc(slices.Clone(rec.Errata), func(id string) bool {
			return id == traveller.E001.String()
		})

		if !e001.holds(struck) {
			t.Fatalf("%s, without E001, no longer satisfies E001's condition, so passing it proves nothing", rec.name)
		}

		if wrong := verdict(struck, e001Withdrawn); len(wrong) > 0 {
			t.Errorf("%s, without E001, is refused for a withdrawn E001 its prose still describes: %v",
				rec.name, wrong)
		}
	}

	if stamped == 0 {
		t.Fatal("no golden stamps E001, so this checks nothing")
	}
}

// A standing reading's verdict refuses a record that disagrees with it, both
// ways. Every golden agrees with every reading, so the main stamping test
// never feeds verdict a disagreement; these copies do, E001's stamp struck
// from a record its condition fits and added to one its condition does not.
//
// Holds: The errata a record stamps are exactly those whose reading's
// condition holds.
func TestAStandingReadingRefusesADisagreement(t *testing.T) {
	t.Parallel()

	tables, err := rules.Load()
	if err != nil {
		t.Fatalf("loading the rules: %v", err)
	}

	var e001 reading

	for _, r := range conditions(tables) {
		if r.id == traveller.E001 {
			e001 = r
		}
	}

	if e001.holds == nil {
		t.Fatal("E001 has no condition")
	}

	var unstamped, overstamped int

	for _, rec := range readRecords(t) {
		copied := rec

		if e001.holds(rec) {
			copied.Errata = slices.DeleteFunc(slices.Clone(rec.Errata), func(id string) bool {
				return id == traveller.E001.String()
			})
			unstamped++
		} else {
			copied.Errata = append(slices.Clone(rec.Errata), traveller.E001.String())
			overstamped++
		}

		if len(verdict(copied, e001)) == 0 {
			t.Errorf("%s, its E001 stamp flipped, was not refused", rec.name)
		}
	}

	if unstamped == 0 || overstamped == 0 {
		t.Fatalf("the goldens give %d records E001 fits and %d it does not; both are needed", unstamped, overstamped)
	}
}
