package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// writeSheet writes the sheet the way Book 1 summarises a character: the
// UPP line, then skills and possessions, as p. 25 does for Jamison, and the
// service a term at a time.
//
// Implements: The errata are on the generation record and not the sheet
// [Generation record].
func writeSheet(out *strings.Builder, r view) {
	fmt.Fprintf(out, "# %s\n\n%s\n\n", heading(r), summary(r))
	fmt.Fprintf(out, "%s\n\n", headline(r))

	writeSection(out, "Skills", skillLines(r))
	writeSection(out, "Possessions", possessionLines(r))
	writeSection(out, "Service record", serviceLines(r))

	// The errata are deliberately not here. They are on the generation
	// record below, where an id beside the outcome it governed is how a
	// reading is checked against the page. On the sheet a player reads they
	// are codes he cannot expand and does not need.
}

// age reads as whole years, or years and months where a medical crisis
// recovery added some (pp. 7-8).
func age(a ageView) string {
	if a.Months == 0 {
		return strconv.Itoa(a.Years)
	}

	return fmt.Sprintf("%d years %d months", a.Years, a.Months)
}

// headline is the line the book itself leads with: the UPP, the age, and
// what the character is.
func headline(r view) string {
	parts := []string{"UPP " + r.UPP, "age " + age(r.Age)}

	if r.Service == "" {
		return strings.Join(append(parts, "civilian, no prior service"), ", ")
	}

	parts = append(parts, serving(r)+whoseService(r)+", "+terms(r.Terms))

	if r.Nobility != nil && r.Nobility.Assumed {
		parts = append(parts, r.Nobility.Title)
	}

	return strings.Join(append(parts, r.Departure.How), ", ")
}

// terms is the count and the word, so a one-term character does not read
// "1 terms".
//
// Implements: A one-term character reads as one term [Sheet].
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
//
// Implements: The headline names the service attempted, not the one the draft
// gave [1:5].
func whoseService(r view) string {
	attempted, byPolicy := r.Attempt.Service, r.Attempt.ByPolicy

	switch {
	case attempted == "":
		return ""
	case attempted != r.Service:
		return fmt.Sprintf(" (%s after the %s refused him)", r.Enlistment.How, attempted)
	case byPolicy:
		return " (service chosen by the policy)"
	default:
		// He asked for this service and is in it. He knows.
		return ""
	}
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

func skillLines(r view) []string {
	lines := make([]string, 0, len(r.Skills))
	for _, skill := range r.Skills {
		lines = append(lines, skillText(skill.Name, skill.Level))
	}

	return lines
}

func possessionLines(r view) []string {
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
//
// Implements: A ship's terms follow its kind, not its numbers [1:22–23].
func shipLine(ship shipView) string {
	if ship.Kind == traveller.ScoutShip.String() {
		return fmt.Sprintf("%s, %d tons, %s", ship.Kind, ship.Tons, heldNotOwned)
	}

	if ship.PaymentYears == 0 {
		return fmt.Sprintf("%s, %d tons, %d years old, owned free and clear",
			ship.Kind, ship.Tons, ship.Years)
	}

	return fmt.Sprintf("%s, %d tons, %d years old, with %d years of payments left",
		ship.Kind, ship.Tons, ship.Years, ship.PaymentYears)
}

// serviceLines is how the character entered, each term that has something to
// tell, and how the service ended.
//
// Implements: The service record narrates the career a term at a time
// [Sheet].
func serviceLines(r view) []string {
	if r.Service == "" {
		return []string{r.Enlistment.How}
	}

	entered := fmt.Sprintf("%s, %s", r.Enlistment.Service, r.Enlistment.How)

	var terms []termView

	entered = joinNonEmpty("; ", entered, strings.Join(r.History.OnEntering, ", "))
	terms = r.History.Terms

	lines := []string{entered}

	for _, term := range terms {
		told := termText(term)
		if told == "" {
			continue
		}

		lines = append(lines, fmt.Sprintf("Term %d: %s", term.Term, told))
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

// termText is one term's rank, gains and aging, each part left out where the
// term has none of it, so a term with nothing to tell is empty. A rank's
// grants are in parentheses, so "; " only ever separates the parts.
func termText(term termView) string {
	raised := make([]string, 0, len(term.Raised))
	for _, raise := range term.Raised {
		raised = append(raised, raise.How+" "+raise.Rank+labelled(" (", strings.Join(raise.Grants, ", "), ")"))
	}

	recovered := make([]string, 0, len(term.Recovered))
	for _, characteristic := range term.Recovered {
		recovered = append(recovered, characteristic+" recovered from a medical crisis")
	}

	return joinNonEmpty("; ",
		strings.Join(raised, " and "),
		strings.Join(term.Gained, ", "),
		labelled("aging ", strings.Join(term.Aged, ", "), ""),
		strings.Join(recovered, ", "))
}

// labelled is text between its label and its close, or nothing where there
// is no text.
func labelled(label, text, closing string) string {
	if text == "" {
		return ""
	}

	return label + text + closing
}

// joinNonEmpty joins the parts that say something.
func joinNonEmpty(separator string, parts ...string) string {
	kept := make([]string, 0, len(parts))

	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, separator)
}

// serving is the service and the rank held in it, as the headline says them
// and a batch member's note is named by them.
func serving(r view) string {
	if r.RankName == "" {
		return r.Service
	}

	return r.Service + " " + r.RankName
}

// skillText is a skill in the book's notation, Skill-1 (p. 12): the one form
// the sheet, the summary, the service history and the properties all write.
func skillText(name string, level int) string {
	return fmt.Sprintf("%s-%d", name, level)
}

// heldNotOwned is how a scout ship is held: in constructive possession, not
// owned (p. 23).
const heldNotOwned = "held in constructive possession"

// heading is what a note is headed by: the character's name, or, where he
// has none, his title, so an unnamed note reads sensibly and a referee names
// him by setting the name and aliases properties (#406).
//
// Implements: An unnamed note is headed by its title [Sheet].
func heading(r view) string {
	if r.Name == "" {
		return titleOf(r)
	}

	return r.Name
}
