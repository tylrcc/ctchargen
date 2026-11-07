package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// Transcript renders the generation record as prose: every step entered,
// every throw made, every choice answered, every consequence.
//
// It is what makes a character auditable against Book 1 - walk the log with
// the page open - and it is the narrative service record besides.
func Transcript(character *chargen.Character) (string, error) {
	projected, err := project(character)
	if err != nil {
		return "", err
	}

	return transcriptOf(projected)
}

// TranscriptFrom renders the log of a record written earlier, through the
// same renderer, for the same reason SheetFrom does.
func TranscriptFrom(text []byte) (string, error) {
	projected, err := decode(text)
	if err != nil {
		return "", err
	}

	return transcriptOf(projected)
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
	By           string   `json:"by"`
	Alternatives []string `json:"alternatives"`
	Chosen       string   `json:"chosen"`
	Because      int      `json:"because"`
	Description  string   `json:"description"`
	Errata       []string `json:"errata"`
}

func transcriptOf(r record) (string, error) {
	var out strings.Builder

	fmt.Fprintf(&out, "# Generation record: %s\n", nameOrBlank(r.Name))

	for _, raw := range r.Events {
		var event eventJSON

		err := unmarshalEvent(raw, &event)
		if err != nil {
			return "", err
		}

		writeEvent(&out, event)
	}

	out.WriteString("\n")
	endWithKeep(&out)

	return out.String(), nil
}

// Implements: An event kind this build does not know is said to be unknown
// [FR11] in The record.
func writeEvent(out *strings.Builder, event eventJSON) {
	switch event.Kind {
	case kindStep:
		fmt.Fprintf(out, "\n## %s (%s)\n\n", event.Step, event.Pages)
	case kindThrow, kindRoll:
		// One line for both: a roll is a throw whose target is empty, which
		// is exactly what throwLine already prints. Splitting the kinds
		// changed the record, not the reading of it.
		fmt.Fprintln(out, throwLine(event))
	case kindChoice:
		fmt.Fprintf(out, "%3d. %s: %s chose %s from %s\n",
			event.Seq, event.Point, event.By, event.Chosen,
			strings.Join(event.Alternatives, ", "))
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
