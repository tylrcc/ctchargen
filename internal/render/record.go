package render

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// writeGenerationRecord writes the generation record as prose: every step
// entered, every throw made, every choice answered, every consequence.
//
// It is what makes a character auditable against Book 1 - walk the record with
// the page open; the sheet above it tells the service in brief, and this is
// every step of it. It is also the only way back to the character (ADR 0006).
//
// It says once, before its first step, what a line's "from step N" names: the
// one step it came from, which is not always its only cause (#577). A weapon
// gained also follows from the table designated and the weapon chosen.
//
// The lines between two step headings are one Markdown ordered list, which a
// viewer numbers from its first line's number up. Every event is one line, in
// order, and every step a heading that ends the list before it, so the number
// a viewer shows is the line's own.
//
// Implements: The kept generation record numbers every event, so no number is
// skipped [Generation record].
//
// Implements: The generation record says what "from step N" names
// [Generation record].
//
// Implements: No viewer renumbers the generation record's lines [Generation
// record].
func writeGenerationRecord(out *strings.Builder, r view) {
	out.WriteString("## Generation record\n\n")
	out.WriteString("A line's `(from step N)` names the step it came from, most often the roll " +
		"it was read off, and not every cause behind it.\n")

	for _, event := range r.Events {
		writeEvent(out, event, r.Answerer)
	}

	out.WriteString("\n")
}

// writeEvent writes one entry of the generation record.
//
// A step is a heading, so a note's outline shows where a reader is in the
// record (#577): a step outside every term, and the step that opens a term,
// at the third level, and a term's skills and training and its aging at the
// fourth, under that term. The heading carries the step's number like every
// other line (#249): bare, it made the numbers appear to skip, and a "from
// step N" naming the step would find nothing.
//
// Implements: The generation record heads each step, a term's phases under the
// term [Generation record].
func writeEvent(out *strings.Builder, event entry, answeredBy answerer) {
	switch event.Kind {
	case kindStep:
		level := "###"
		if event.Within != 0 {
			level = "####"
		}

		fmt.Fprintf(out, "\n%s %s\n\n", level, stepLine(event))
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints.
		fmt.Fprintln(out, throwLine(event))
	case kindChoice:
		fmt.Fprintln(out, choiceLine(event, answeredBy))
	case kindOutcome:
		fmt.Fprintln(out, outcomeLine(event))
	}
}

// choiceLine is a choice's line. It names who answered, which is the run's to
// say and not the event's; the live view, which is watched by the person
// answering, prints its own. An answer drawn reads "drew" (#564): the event
// says it was, because the decider drew it.
//
// Implements: A choice's line says what a drawn or least-trained answer was
// decided on [Generation record].
func choiceLine(event entry, answeredBy answerer) string {
	verb := "chose"
	if event.Drawn {
		verb = "drew"
	}

	return fmt.Sprintf("%d. %s: %s %s %s%s from %s",
		event.Seq, choiceWords(event.Point), answeredBy.Who, verb, event.Chosen,
		leastTrained(event, answeredBy), strings.Join(event.Alternatives, ", "))
}

// leastTrained is the policy's reason for a skills table, from the counts
// the record carries (#564): the one table least trained on, with the
// counts of the others in the order offered, "least trained on (1, against
// 2 and 2)"; or, where several tie, how many others and that the skills
// strategy's ranking took the first of them, "tied least trained on (0)
// with three others, the advanced strategy's first". Each clause is set off
// by commas before the alternatives.
//
// It is empty for a player, whose choice has no policy reason, and for every
// choice without counts. A policy's table that was not least trained on is
// a generation defect, since the policy designates among those only, and
// fails here rather than being worded as a reason it was not.
func leastTrained(event entry, answeredBy answerer) string {
	if !answeredBy.ByPolicy || len(event.Taken) == 0 {
		return ""
	}

	chosen := slices.Index(event.Alternatives, event.Chosen)
	fewest := slices.Min(event.Taken)

	if chosen < 0 || len(event.Taken) != len(event.Alternatives) || event.Taken[chosen] != fewest {
		panic(fmt.Sprintf("render: the policy designated %q, not least trained on, from %v with %v taken",
			event.Chosen, event.Alternatives, event.Taken))
	}

	others := make([]string, 0, len(event.Taken)-1)
	tied := 0

	for i, taken := range event.Taken {
		if i == chosen {
			continue
		}

		others = append(others, strconv.Itoa(taken))

		if taken == fewest {
			tied++
		}
	}

	if tied == 0 {
		return fmt.Sprintf(", least trained on (%d, against %s),", fewest, listed(others))
	}

	return fmt.Sprintf(", tied least trained on (%d) with %s, the %s strategy's first,",
		fewest, othersInWords[tied], answeredBy.Skills)
}

// othersInWords is how many other tables tied, in words, by the count: one
// to three, since p. 11 prints four tables.
var othersInWords = [...]string{1: "one other", 2: "two others", 3: "three others"}

// choiceWords is each choice in the words of the page that poses it, with
// that page. A switch rather than a map, so that a choice point added to
// traveller is a lint failure here until it is given its words.
//
// Implements: A choice is named in the words of the page that poses it
// [Generation record].
func choiceWords(point traveller.ChoicePoint) string {
	switch point {
	case traveller.ChoiceService:
		return "service to enlist in (pp. 5, 10)"
	case traveller.ChoiceSubmitToDraft:
		return "submit to the draft (p. 5)"
	case traveller.ChoiceAttemptCommission:
		return "attempt a commission (p. 6)"
	case traveller.ChoiceAttemptPromotion:
		return "attempt promotion (p. 6)"
	case traveller.ChoiceReenlistIntent:
		return "leave the service or remain for another term (pp. 6-7, 21)"
	case traveller.ChoiceAssumeTitle:
		return "assume the hereditary title (p. 5)"
	case traveller.ChoiceSkillTable:
		return "acquired skills table to consult (pp. 6, 11)"
	case traveller.ChoiceWeapon:
		return "weapon of expertise, specified immediately (p. 11)"
	case traveller.ChoiceMusterTable:
		return "mustering out table to consult (pp. 7, 9)"
	case traveller.ChoiceMusterTable1DM:
		return "add +1 on Table 1 for rank 5 or 6 (pp. 7, 9)"
	case traveller.ChoiceMusterTable2DM:
		return "DM +1 on Table 2 for gambling expertise (pp. 7, 9)"
	case traveller.ChoiceMusterWeapon:
		return "weapon benefit, or +1 expertise in lieu (p. 22)"
	}

	return fmt.Sprintf("ChoicePoint(%d)", int(point))
}

// throwLine is a throw as the record reads it: the dice, the DM and the
// characteristics it is earned by, the total, the target and whether it was
// met.
//
// Implements: A throw's DM names what earned it [Generation record].
func throwLine(event entry) string {
	dice := make([]string, 0, len(event.Dice))
	for _, die := range event.Dice {
		dice = append(dice, strconv.Itoa(die))
	}

	line := fmt.Sprintf("%d. %s: rolled %s", event.Seq, event.Step, strings.Join(dice, "+"))

	if event.DM != 0 {
		line += fmt.Sprintf(" %+d", event.DM)
	}

	// The characteristics the DM is earned by, with the thresholds p. 10
	// prints for them, so it can be checked against the page (#487).
	if event.DMFrom != "" {
		line += " (" + event.DMFrom + ")"
	}

	// The total is printed whenever it is not simply the dice: a throw with
	// a target is read against it, and a one-die roll with a modifier is
	// read off a row the face alone does not name.
	if event.DM != 0 || event.Target != "" {
		line += fmt.Sprintf(" = %d", event.Total)
	}

	if event.Target != "" {
		line += fmt.Sprintf(" against %s, %s", event.Target, met(event.Succeeded))
	}

	return line
}

func met(succeeded bool) string {
	if succeeded {
		return "made"
	}

	return "missed"
}

func outcomeLine(event entry) string {
	line := fmt.Sprintf("%d. %s", event.Seq, event.Description)

	if event.Because != 0 {
		line += fmt.Sprintf(" (from step %d)", event.Because)
	}

	return line
}

// stepLine is a step's number, name and pages: the note's heading for the
// step, and the live view's plain line.
func stepLine(event entry) string {
	return fmt.Sprintf("%d. %s (%s)", event.Seq, event.Step, event.Pages)
}
