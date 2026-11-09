package render

import (
	"fmt"
	"strconv"
	"strings"
)

// Transcript renders a record's generation log as prose: every step entered,
// every throw made, every choice answered, every consequence.
//
// It is what makes a character auditable against Book 1 - walk the log with
// the page open - and it is the narrative service record besides. With
// readings, it ends with the full text of each reading the record names, so
// the ids beside the outcomes can be read without the tool to hand. It takes
// the record's bytes, for the reason Sheet does.
func Transcript(text []byte, readings bool) (string, error) {
	projected, events, err := decode(text)
	if err != nil {
		return "", err
	}

	return transcriptOf(projected, events, readings), nil
}

// eventJSON is every event kind at once. On the wire they are a
// discriminated union, and on the way back in the discriminator is all a
// reader has to go on.
type eventJSON struct {
	Seq          int      `json:"seq"`
	Kind         string   `json:"kind"`
	Step         string   `json:"step"`
	Pages        string   `json:"pages"`
	Dice         []int    `json:"dice"`
	DM           int      `json:"dm"`
	Target       string   `json:"target"`
	Total        int      `json:"total"`
	Succeeded    bool     `json:"succeeded"`
	Point        string   `json:"point"`
	Alternatives []string `json:"alternatives"`
	Chosen       string   `json:"chosen"`
	Because      int      `json:"because"`
	Description  string   `json:"description"`
	Errata       []string `json:"errata"`
}

func transcriptOf(r record, events []eventJSON, readings bool) string {
	var out strings.Builder

	fmt.Fprintf(&out, "# Generation record: %s\n", nameOrBlank(r.Name))

	for _, event := range events {
		if event.Kind == kindChoice {
			fmt.Fprintln(&out, choiceLine(event, r.answerer()))

			continue
		}

		writeEvent(&out, event)
	}

	out.WriteString("\n")

	if readings {
		writeReadings(&out, r.Errata)
	}

	endWithKeep(&out)

	return out.String()
}

// writeEvent writes every kind but a choice, whose line is choiceLine's.
//
// A step heading carries its number like every other line (#249): bare, it
// made the numbers appear to skip, and a transcript kept for the table opened
// at event 2.
//
// Implements: An event kind this build does not know is said to be unknown
// [Generation record].
//
// Implements: The kept transcript numbers every event, so no number is skipped
// [Generation record].
func writeEvent(out *strings.Builder, event eventJSON) {
	switch event.Kind {
	case kindStep:
		fmt.Fprintf(out, "\n## %d. %s (%s)\n\n", event.Seq, event.Step, event.Pages)
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints. Splitting the kinds
		// changed the record, not the reading of it.
		fmt.Fprintln(out, throwLine(event))
	case kindOutcome:
		fmt.Fprintln(out, outcomeLine(event))
	default:
		// A record written by a build that logs a kind this one does not
		// know. Saying so is the point of reading another build's record at
		// all; rendering it as an outcome would print a blank numbered line
		// and claim the transcript was complete.
		fmt.Fprintf(out, "%3d. (unknown event kind %q)\n", event.Seq, event.Kind)
	}
}

// choiceLine is a choice's transcript line. It is not one of writeEvent's
// cases because it names who answered, which is the record's to say and not
// the event's, and the live view, which has no record yet, prints its own.
func choiceLine(event eventJSON, answerer string) string {
	return fmt.Sprintf("%3d. %s: %s chose %s from %s",
		event.Seq, event.Point, answerer, event.Chosen,
		strings.Join(event.Alternatives, ", "))
}

func throwLine(event eventJSON) string {
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

func outcomeLine(event eventJSON) string {
	line := fmt.Sprintf("%3d. %s", event.Seq, event.Description)

	if event.Because != 0 {
		line += fmt.Sprintf(" (from %d)", event.Because)
	}

	if len(event.Errata) > 0 {
		line += " [" + strings.Join(event.Errata, " ") + "]"
	}

	return line
}
