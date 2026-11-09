package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// writeGenerationRecord writes the generation log as prose: every step
// entered, every throw made, every choice answered, every consequence.
//
// It is what makes a character auditable against Book 1 - walk the log with
// the page open; the sheet above it tells the service in brief, and this is
// every step of it. It is also the only way back to the character (ADR 0006).
//
// Implements: The kept generation record numbers every event, so no number is
// skipped [Generation record].
func writeGenerationRecord(out *strings.Builder, r view) {
	out.WriteString("## Generation record\n")

	for _, event := range r.Events {
		writeEvent(out, event, r.answerer())
	}

	out.WriteString("\n")
}

// writeEvent writes one entry of the generation record.
//
// A step line carries its number like every other line (#249): bare, it
// made the numbers appear to skip, and a record kept for the table opened at
// event 2.
//
// Implements: The generation record is one heading, its steps bold lines
// [Generation record].
func writeEvent(out *strings.Builder, event entry, answerer string) {
	switch event.Kind {
	case kindStep:
		fmt.Fprintf(out, "\n**%s**\n\n", stepLine(event))
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints.
		fmt.Fprintln(out, throwLine(event))
	case kindChoice:
		fmt.Fprintln(out, choiceLine(event, answerer))
	case kindOutcome:
		fmt.Fprintln(out, outcomeLine(event))
	}
}

// choiceLine is a choice's line. It names who answered, which is the run's to
// say and not the event's; the live view, which is watched by the person
// answering, prints its own.
func choiceLine(event entry, answerer string) string {
	return fmt.Sprintf("%3d. %s: %s chose %s from %s",
		event.Seq, choiceWords(event.Point), answerer, event.Chosen,
		strings.Join(event.Alternatives, ", "))
}

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

func throwLine(event entry) string {
	dice := make([]string, 0, len(event.Dice))
	for _, die := range event.Dice {
		dice = append(dice, strconv.Itoa(die))
	}

	line := fmt.Sprintf("%3d. %s: rolled %s", event.Seq, event.Step, strings.Join(dice, "+"))

	if event.DM != 0 {
		line += fmt.Sprintf(" %+d", event.DM)
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
	line := fmt.Sprintf("%3d. %s", event.Seq, event.Description)

	if event.Because != 0 {
		line += fmt.Sprintf(" (from %d)", event.Because)
	}

	if len(event.Errata) > 0 {
		line += " [" + strings.Join(event.Errata, " ") + "]"
	}

	return line
}

// stepLine is a step's number, name and pages: what the note marks bold and
// the live view prints plain.
func stepLine(event entry) string {
	return fmt.Sprintf("%d. %s (%s)", event.Seq, event.Step, event.Pages)
}
