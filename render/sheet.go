package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Sheet renders a character the way Book 1 summarises one: the UPP line,
// then skills and possessions, as p. 25 does for Jamison.
func Sheet(character *chargen.Character) (string, error) {
	projected, err := project(character)
	if err != nil {
		return "", err
	}

	return sheetOf(projected)
}

// SheetFrom renders a record that was written earlier, which is what the
// render subcommand reads.
//
// It goes through the same renderer as Sheet rather than a second one over
// the same values: a sheet describes the record, so the record is what it
// should be written from, and one renderer cannot disagree with itself.
func SheetFrom(text []byte) (string, error) {
	projected, err := decode(text)
	if err != nil {
		return "", err
	}

	return sheetOf(projected)
}

// sheetOf writes the sheet, and ends it by saying what to keep.
//
// The footer is last because it is not the character: a referee reads the
// headline and the possessions, and thinks about keeping the record only once
// he has decided he wants this one.
func sheetOf(r record) (string, error) {
	// Every event is read before anything is printed, and one that will not
	// read refuses the sheet. The sheet shows one choice event at most, but
	// it ends by telling the referee that the record is the only way back to
	// this character, and that is not something to say about a record it
	// could not finish reading.
	_, err := choices(r)
	if err != nil {
		return "", err
	}

	var out strings.Builder

	fmt.Fprintf(&out, "# %s\n\n", nameOrBlank(r.Name))

	lead, err := headline(r)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(&out, "%s\n\n", lead)

	writeSection(&out, "Skills", skillLines(r))
	writeSection(&out, "Possessions", possessionLines(r))
	writeSection(&out, "Service record", serviceLines(r))

	// The errata are deliberately not here. They stay in the record and on
	// the transcript, where an id beside the outcome it governed is how a
	// reading is checked against the page. On a sheet handed to a player
	// they are four codes he cannot expand and does not need.

	endWithKeep(&out)

	return out.String(), nil
}

// keepTheRecord is the one line every rendering ends with.
//
// A sheet is what a referee reads and keeps, and the record is the only way
// back to the character on it (ADR 0001). Without this line a sheet spun,
// read and liked would be a character discarded. The transcript ends the same
// way, so the two printed renderings give the same advice.
const keepTheRecord = "Keep the JSON record: it is the only way back to this character."

// endWithKeep writes the footer every rendering ends with, after a rule.
func endWithKeep(out *strings.Builder) {
	fmt.Fprintf(out, "---\n\n%s\n", keepTheRecord)
}

func nameOrBlank(name string) string {
	if name == "" {
		return "(unnamed)"
	}

	return name
}

// age reads as whole years, or years and months where a medical crisis
// recovery added some (pp. 7-8).
func age(a ageRecord) string {
	if a.Months == 0 {
		return strconv.Itoa(a.Years)
	}

	return fmt.Sprintf("%d years %d months", a.Years, a.Months)
}

// headline is the line the book itself leads with: the UPP, the age, and
// what the character is.
func headline(r record) (string, error) {
	parts := []string{"UPP " + r.UPP, "age " + age(r.Age)}

	if r.Service == "" {
		return strings.Join(append(parts, "civilian, no prior service"), ", "), nil
	}

	serving := r.Service
	if r.RankName != "" {
		serving += " " + r.RankName
	}

	whose, err := whoseService(r)
	if err != nil {
		return "", err
	}

	parts = append(parts, serving+whose+", "+terms(r.Terms))

	if r.Nobility != nil && r.Nobility.Assumed {
		parts = append(parts, r.Nobility.Title)
	}

	if r.Departure != nil {
		parts = append(parts, r.Departure.How)
	}

	return strings.Join(parts, ", "), nil
}

// terms is the count and the word, so a one-term character does not read
// "1 terms".
func terms(count int) string {
	if count == oneTerm {
		return "1 term"
	}

	return fmt.Sprintf("%d terms", count)
}

// oneTerm is the only count the plural is wrong for.
const oneTerm = 1

// whoseService says, where it is not obvious, that the service on the
// headline is not the one that was attempted.
//
// The service record four sections down has always been honest - it says
// "drafted" - but the headline is the line a referee reads, and a character
// who typed --service marines and got a Navy man had nothing there to explain
// it.
//
// Everything turns on the service that was ATTEMPTED, which is not the same
// question as who named it. A policy that picks Other and sees the throw fail
// gets whatever the draft hands him, and saying "chosen by the policy" over
// that service credits the policy with a choice it never made - in exactly
// the case this exists to expose.
func whoseService(r record) (string, error) {
	attempted, byPolicy, err := serviceAttempted(r)
	if err != nil {
		return "", err
	}

	switch {
	case attempted == "":
		return "", nil
	case !strings.EqualFold(attempted, r.Service):
		return fmt.Sprintf(" (%s after the %s refused him)", r.Enlistment.How, attempted), nil
	case byPolicy:
		return " (service chosen by the policy)", nil
	default:
		// He asked for this service and is in it. He knows.
		return "", nil
	}
}

// serviceAttempted is the service the enlistment throw was made against, and
// whether the policy is what named it.
//
// The record holds it in two places for two reasons, and neither is
// redundant. A forced run carries it in `inputs` and logs no choice at all -
// there was nothing to ask. An unforced run logs the choice and leaves
// `inputs.service` empty, and that answer is the one the draft can override.
func serviceAttempted(r record) (string, bool, error) {
	if r.Inputs.Service != "" {
		return r.Inputs.Service, false, nil
	}

	choice, asked, err := choiceAt(r, traveller.ChoiceService.String())
	if err != nil || !asked {
		return "", false, err
	}

	return choice.Chosen, choice.By == traveller.ByPolicy.String(), nil
}

func writeSection(out *strings.Builder, heading string, lines []string) {
	if len(lines) == 0 {
		return
	}

	fmt.Fprintf(out, "## %s\n\n", heading)

	for _, line := range lines {
		fmt.Fprintf(out, "- %s\n", line)
	}

	out.WriteString("\n")
}

func skillLines(r record) []string {
	lines := make([]string, 0, len(r.Skills))
	for _, skill := range r.Skills {
		lines = append(lines, fmt.Sprintf("%s-%d", skill.Name, skill.Level))
	}

	return lines
}

func possessionLines(r record) []string {
	var lines []string

	if r.Benefits.Cash > 0 {
		lines = append(lines, credits(r.Benefits.Cash))
	}

	lines = append(lines, r.Benefits.Passages...)

	if r.Benefits.TravellersAid {
		lines = append(lines, "Travellers' Aid Society membership")
	}

	lines = append(lines, r.Benefits.Weapons...)

	for _, ship := range r.Benefits.Ships {
		lines = append(lines, shipLine(ship))
	}

	if r.Pension > 0 {
		lines = append(lines, credits(r.Pension)+" a year in retirement pay")
	}

	return lines
}

func credits(amount int64) string { return fmt.Sprintf("CR %d", amount) }

// shipLine reads a ship's terms off its kind, which is what the terms turn
// on: a scout ship is held without title (p. 23) whatever its age, and a
// Free Trader is owned whatever is left to pay. Inferring the one from the
// other's numbers would call a new, unencumbered Free Trader a scout ship.
func shipLine(ship shipRecord) string {
	if ship.Kind == traveller.ScoutShip.String() {
		return fmt.Sprintf("%s, %d tons, held in constructive possession", ship.Kind, ship.Tons)
	}

	if ship.PaymentYears == 0 {
		return fmt.Sprintf("%s, %d tons, %d years old, owned free and clear",
			ship.Kind, ship.Tons, ship.Years)
	}

	return fmt.Sprintf("%s, %d tons, %d years old, with %d years of payments left",
		ship.Kind, ship.Tons, ship.Years, ship.PaymentYears)
}

func serviceLines(r record) []string {
	if r.Service == "" {
		return []string{r.Enlistment.How}
	}

	lines := []string{fmt.Sprintf("%s, %s", r.Enlistment.Service, r.Enlistment.How)}

	if r.Departure == nil {
		return lines
	}

	switch {
	case r.Departure.Characteristic != "":
		return append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			r.Departure.How, r.Terms, r.Departure.Characteristic))
	case r.Departure.Fatal:
		return append(lines, fmt.Sprintf("%s in term %d", r.Departure.How, r.Terms))
	default:
		return append(lines, fmt.Sprintf("%s after term %d", r.Departure.How, r.Terms))
	}
}

// choices reads the record's choice events, which are the only place it says
// who decided anything. A record whose events will not read fails here.
func choices(r record) ([]eventJSON, error) {
	var found []eventJSON

	for _, raw := range r.Events {
		var event eventJSON

		err := unmarshalEvent(raw, &event)
		if err != nil {
			return nil, err
		}

		if event.Kind == kindChoice {
			found = append(found, event)
		}
	}

	return found, nil
}

// choiceAt returns a named choice point's event, and whether it was put at
// all. A point nobody was asked is not the same as one the policy took.
//
// It hands back the whole event rather than who answered, because both halves
// are wanted together: who chose, and what they chose. Asking only who leads
// to crediting a decider with an outcome he did not pick.
func choiceAt(r record, point string) (eventJSON, bool, error) {
	made, err := choices(r)
	if err != nil {
		return eventJSON{}, false, err
	}

	for _, choice := range made {
		if choice.Point == point {
			return choice, true, nil
		}
	}

	return eventJSON{}, false, nil
}
