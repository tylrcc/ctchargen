package chargen_test

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The record says what gave each count of rolls, where the cap sends the rest
// of mustering out, and when Education opens the fourth skills table (#565).
// Each is held over the roster and over a run of seeded characters, since no
// fixture's Education reaches 8 between two designations.

// seeded is a roller drawn from a fixed seed, so a run of characters is the
// same run every time and reaches paths the roster does not.
type seeded struct{ r *rand.Rand }

func (s seeded) Die() int        { return s.r.IntN(6) + 1 }
func (s seeded) Among(n int) int { return s.r.IntN(n) }

// characters is every fixture's character, then the seeded run's, each
// named so a failure says which.
func characters(t *testing.T) map[string]*chargen.Character {
	t.Helper()

	out := make(map[string]*chargen.Character, len(fixtures)+int(seededRun))

	for _, fixture := range fixtures {
		out[fixture.name] = fixture.generate(t)
	}

	for seed := range seededRun {
		roller := seeded{rand.New(rand.NewPCG(seed, seed))}

		character, err := chargen.Auto(chargen.Inputs{}, chargen.DefaultPolicy(), chargen.WithRoller(roller))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		out["seed "+strconv.FormatUint(seed, 10)] = character
	}

	return out
}

// seededRun is how many seeded characters the checks walk: enough that one
// reaches Education 8 between two designations.
const seededRun uint64 = 400

// The p. 6 Basic Skill Eligibility box, typed out here rather than read from
// the engine: two rolls for the initial term, one for each later term, and
// one each for a commission and a promotion.
const (
	initialTermRolls    = 2
	laterTermRolls      = 1
	rollsPerAdvancement = 1
)

var (
	skillsStep = regexp.MustCompile(`^skills and training, (\d+) rolls?: (.+)$`)
	musterStep = regexp.MustCompile(`^mustering out, (\d+) rolls?: (.+)$`)
)

// isSkillTable reports whether a roll's step is a skills table.
func isSkillTable(step string) bool {
	return slices.ContainsFunc(traveller.SkillTables[:], func(table traveller.SkillTable) bool {
		return table.String() == step
	})
}

// isMusterTable reports whether a roll's step is a mustering out table.
func isMusterTable(step string) bool {
	return step == traveller.TableOne.String() || step == traveller.TableTwo.String()
}

// Each skills step says how many rolls the term gave, which is how many
// tables are rolled on before the next step, and what gave them, which is
// the p. 6 box applied to the term and to what it raised him to.
//
// Holds: The skills step says how many rolls the term gave, and what gave
// them.
func TestTheSkillsStepSaysWhatGaveItsRolls(t *testing.T) {
	t.Parallel()

	for name, character := range characters(t) {
		var term, commissions, promotions int

		for i, event := range character.Events {
			switch e := event.(type) {
			case traveller.StepEvent:
				if e.Opens != 0 {
					term, commissions, promotions = int(e.Opens), 0, 0
				}

				if strings.HasPrefix(e.Step, "skills and training") {
					checkSkillsStep(t, name, e, character.Events[i+1:], term, commissions, promotions)
				}
			case traveller.OutcomeEvent:
				raised, ok := e.Fact.(traveller.RaisedFact)
				if !ok {
					continue
				}

				switch raised.How {
				case traveller.Commissioned:
					commissions++
				case traveller.Promoted:
					promotions++
				}
			}
		}
	}
}

func checkSkillsStep(t *testing.T, name string, step traveller.StepEvent, after []traveller.Event,
	term, commissions, promotions int,
) {
	t.Helper()

	match := skillsStep.FindStringSubmatch(step.Step)
	if match == nil {
		t.Errorf("%s: step %d reads %q, which does not say its rolls", name, step.Seq, step.Step)

		return
	}

	rolled := 0

	for _, event := range after {
		if _, next := event.(traveller.StepEvent); next {
			break
		}

		if roll, ok := event.(traveller.RollEvent); ok && isSkillTable(roll.Step) {
			rolled++
		}
	}

	if match[1] != strconv.Itoa(rolled) {
		t.Errorf("%s: step %d says %s rolls, and %d were rolled", name, step.Seq, match[1], rolled)
	}

	forTerm := laterTermRolls
	if term == 1 {
		forTerm = initialTermRolls
	}

	want := []string{fmt.Sprintf("%d for the term", forTerm)}
	if commissions > 0 {
		want = append(want, fmt.Sprintf("%d for the commission", commissions*rollsPerAdvancement))
	}

	if promotions > 0 {
		want = append(want, fmt.Sprintf("%d for the promotion", promotions*rollsPerAdvancement))
	}

	if got := match[2]; got != strings.Join(want, ", ") {
		t.Errorf("%s: step %d says the rolls came %q, want %q", name, step.Seq, got, strings.Join(want, ", "))
	}
}

// The p. 9 benefit rolls, typed out here rather than read from the engine:
// one per term, one more at rank 1 or 2, two more at rank 3 or higher.
func extraBenefitRolls(rank traveller.Rank) int {
	switch {
	case rank >= 3:
		return 2
	case rank >= 1:
		return 1
	default:
		return 0
	}
}

// rollsOn is how many of events are rolls on a table that is accepts.
func rollsOn(events []traveller.Event, is func(string) bool) int {
	count := 0

	for _, event := range events {
		if roll, ok := event.(traveller.RollEvent); ok && is(roll.Step) {
			count++
		}
	}

	return count
}

// linesMatching is how many of events are outcomes whose line matches.
func linesMatching(events []traveller.Event, line *regexp.Regexp) int {
	count := 0

	for _, event := range events {
		if outcome, ok := event.(traveller.OutcomeEvent); ok && line.MatchString(outcome.Description) {
			count++
		}
	}

	return count
}

// The mustering out step says how many rolls there are, which is how many
// are rolled, and what gave them: the terms served, and the rank he left at.
//
// Holds: The mustering out step says how many rolls there are, and what gave
// them.
func TestTheMusterStepSaysWhatGaveItsRolls(t *testing.T) {
	t.Parallel()

	var reachedRank bool

	for name, character := range characters(t) {
		for i, event := range character.Events {
			step, ok := event.(traveller.StepEvent)
			if ok && strings.HasPrefix(step.Step, "mustering out") {
				reachedRank = checkMusterStep(t, name, character, step, character.Events[i+1:]) || reachedRank
			}
		}
	}

	if !reachedRank {
		t.Error("no character musters out with rolls for rank, so that half says nothing here")
	}
}

// checkMusterStep holds one mustering out step to the rolls after it, and
// reports whether the rank gave any.
func checkMusterStep(t *testing.T, name string, character *chargen.Character, step traveller.StepEvent,
	after []traveller.Event,
) bool {
	t.Helper()

	match := musterStep.FindStringSubmatch(step.Step)
	if match == nil {
		t.Errorf("%s: step %d reads %q, which does not say its rolls", name, step.Seq, step.Step)

		return false
	}

	if rolled := rollsOn(after, isMusterTable); match[1] != strconv.Itoa(rolled) {
		t.Errorf("%s: step %d says %s rolls, and %d were rolled", name, step.Seq, match[1], rolled)
	}

	terms := character.Terms()

	want := fmt.Sprintf("%d for the terms", terms)
	if terms == 1 {
		want = "1 for the term"
	}

	extra := extraBenefitRolls(character.Rank())
	if extra > 0 {
		want += fmt.Sprintf(", %d for rank %d", extra, character.Rank())
	}

	if match[2] != want {
		t.Errorf("%s: step %d says the rolls came %q, want %q", name, step.Seq, match[2], want)
	}

	return extra > 0
}

// capLine is the record's line where the third roll on Table 2 leaves rolls
// to take, matched whole.
var capLine = regexp.MustCompile(
	`^3 rolls on Table 2, the most allowed: the remaining (\d+) rolls? must be on Table 1$`,
)

// Where the third roll on Table 2 leaves rolls to take, the outcome after
// its cash says the rest must go on Table 1, how many there are, and comes
// from that third roll. Nowhere else says so.
//
// Holds: The record says when the cap sends the remaining rolls to Table 1.
func TestTheRecordSaysWhenTheCapSendsTheRestToTableOne(t *testing.T) {
	t.Parallel()

	var reached bool

	for name, character := range characters(t) {
		said := 0

		if i, ok := thirdRollOnTableTwo(character.Events); ok {
			if left := rollsOn(character.Events[i+1:], isMusterTable); left > 0 {
				reached = true
				said = 1

				checkCapLine(t, name, character.Events, i, left)
			}
		}

		if lines := linesMatching(character.Events, capLine); lines != said {
			t.Errorf("%s: %d lines say the cap sends rolls to Table 1, where it sent them %d times",
				name, lines, said)
		}
	}

	if !reached {
		t.Error("no character takes three rolls on Table 2 with rolls left, so the line is never held")
	}
}

// thirdRollOnTableTwo is where the third roll on Table 2 is in events, if
// there is one.
func thirdRollOnTableTwo(events []traveller.Event) (int, bool) {
	onTableTwo := 0

	for i, event := range events {
		if roll, ok := event.(traveller.RollEvent); ok && roll.Step == tableTwoStep {
			onTableTwo++
			if onTableTwo == printedCap {
				return i, true
			}
		}
	}

	return 0, false
}

// checkCapLine holds the line after the third roll on Table 2, at i, and its
// cash: it says how many rolls are left, and comes from that roll.
func checkCapLine(t *testing.T, name string, events []traveller.Event, i, left int) {
	t.Helper()

	roll := events[i]
	line, ok := events[i+2].(traveller.OutcomeEvent)

	match := capLine.FindStringSubmatch(line.Description)
	if !ok || match == nil || match[1] != strconv.Itoa(left) || line.Because != roll.Sequence() {
		t.Errorf("%s: after the third roll on Table 2 (step %d), with %d left, the record reads %+v",
			name, roll.Sequence(), left, events[i+2])
	}
}

// fourthTableLine is the line at the designation where Education has opened
// the fourth table, matched whole.
var fourthTableLine = regexp.MustCompile(
	`^Education (\d+), so this designation offers the Advanced Education Table \(education 8\+\)$`,
)

// The first designation that offers the fourth table after one that did not
// is preceded by a line saying the Education that opened it, from the change
// that took Education to 8 or more. No other designation has one.
//
// Holds: The first designation that offers the fourth table says the
// Education that opened it.
func TestTheRecordSaysWhenEducationOpensTheFourthTable(t *testing.T) {
	t.Parallel()

	var reached bool

	for name, character := range characters(t) {
		openings := checkFourthTable(t, name, character.Events)

		reached = reached || openings > 0

		if lines := linesMatching(character.Events, fourthTableLine); lines != openings {
			t.Errorf("%s: %d lines say Education opened the fourth table, and it opened %d times",
				name, lines, openings)
		}
	}

	if !reached {
		t.Error("no character's Education reaches 8 between two designations, so the line is never held")
	}
}

// checkFourthTable holds the line before each designation that first offers
// the fourth table, and returns how many there were.
func checkFourthTable(t *testing.T, name string, events []traveller.Event) int {
	t.Helper()

	fourth := traveller.AdvancedEducationEight.String()
	designated, offeredBefore, openings := false, false, 0

	for i, event := range events {
		choice, ok := event.(traveller.ChoiceEvent)
		if !ok || choice.Point != traveller.ChoiceSkillTable {
			continue
		}

		offered := slices.Contains(choice.Alternatives, fourth)
		opened := designated && offered && !offeredBefore

		designated, offeredBefore = true, offered

		if !opened {
			continue
		}

		openings++

		line, _ := events[i-1].(traveller.OutcomeEvent)

		match := fourthTableLine.FindStringSubmatch(line.Description)
		if match == nil {
			t.Errorf("%s: step %d offers the fourth table first, and step %d before it reads %+v",
				name, choice.Seq, events[i-1].Sequence(), events[i-1])

			continue
		}

		// The line comes from the change that took Education to 8 or more,
		// and names the value it reached.
		cause, _ := eventAt(events, line.Because).(traveller.OutcomeEvent)
		applied, _ := cause.Fact.(traveller.AppliedFact)
		change, _ := applied.Gain.(traveller.Change)

		if change.Characteristic != traveller.Education || change.From >= 8 || change.To < 8 ||
			match[1] != strconv.Itoa(change.To) {
			t.Errorf("%s: step %d says %q from step %d, which reads %+v", name, line.Seq, line.Description,
				line.Because, eventAt(events, line.Because))
		}
	}

	return openings
}

// eventAt is the event numbered seq, or nil where there is none: events are
// numbered from 1 in order.
func eventAt(events []traveller.Event, seq int) traveller.Event {
	if seq < 1 || seq > len(events) {
		return nil
	}

	return events[seq-1]
}
