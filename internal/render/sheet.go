package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// writeSheet writes the sheet the way Book 1 summarises a character: the
// profile line, then skills and benefits, as p. 25 does for Jamison, and the
// service a term at a time. The summary follows the profile line, and is
// left out where it has nothing to say.
func writeSheet(out *strings.Builder, r view) {
	fmt.Fprintf(out, "# %s\n\n%s\n\n", heading(r), profileLine(r))

	if told := summary(r); told != "" {
		fmt.Fprintf(out, "%s\n\n", told)
	}

	writeSection(out, "Skills", skillLines(r))
	writeSection(out, "Benefits", benefitLines(r))
	writeSection(out, "Service record", serviceLines(r))
}

// age reads as whole years, or years and months where a medical crisis
// recovery added some (pp. 7-8).
func age(a ageView) string {
	if a.Months == 0 {
		return strconv.Itoa(a.Years)
	}

	return fmt.Sprintf("%d years %d months", a.Years, a.Months)
}

// profileLine is the line the book itself leads with: the UPP, the age, and
// what the character is.
func profileLine(r view) string {
	parts := []string{"UPP " + r.UPP, "age " + age(r.Age)}

	if r.Service == "" {
		return strings.Join(append(parts, "civilian, no prior service"), ", ")
	}

	parts = append(parts, serving(r)+entered(r)+", "+terms(r.Terms))

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

// entered says on the profile line how the character entered the service:
// enlisted, or drafted after the service attempted refused enlistment (p. 5),
// which may be the service the draft gave.
//
// The service record four sections down has always said so, but the profile
// line is the line a referee reads first, and a character who typed --service
// marines and got a Navy man had nothing there to explain it. Who named the
// service attempted, the policy or the player, is not this line's to say: the
// record's choice line says it (#583).
//
// Implements: The profile line says how the character entered the service
// [1:5].
func entered(r view) string {
	if r.Enlistment.Drafted {
		return fmt.Sprintf(" (drafted after the %s refused enlistment)", r.Attempted)
	}

	return " (" + r.Enlistment.How + ")"
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

// benefitLines is what mustering out gave, one list under the sheet's
// Benefits heading: the possessions property's items with the cash and the
// retirement pay beside them.
func benefitLines(r view) []string {
	var lines []string

	if r.Benefits.Cash > 0 {
		lines = append(lines, credits(r.Benefits.Cash))
	}

	lines = append(lines, passageLines(r.Benefits.Passages)...)

	if r.Benefits.TravellersAid {
		lines = append(lines, "Travellers' Aid Society membership")
	}

	lines = append(lines, r.Benefits.Weapons...)

	for _, ship := range r.Benefits.Ships {
		lines = append(lines, shipLine(ship))
	}

	if r.Pension > 0 {
		lines = append(lines, retirementPay(r.Pension))
	}

	return lines
}

// passageLines is each class of passage once, in the order first received,
// with a count where there is more than one (#560). The possessions property
// keeps one item for each passage, as #399 holds it.
//
// Implements: Passages of one class are one line with a count [Sheet].
func passageLines(passages []string) []string {
	var order []string

	count := map[string]int{}

	for _, passage := range passages {
		if count[passage] == 0 {
			order = append(order, passage)
		}

		count[passage]++
	}

	lines := make([]string, 0, len(order))

	for _, passage := range order {
		if count[passage] == 1 {
			lines = append(lines, passage)

			continue
		}

		lines = append(lines, fmt.Sprintf("%s ×%d", passage, count[passage]))
	}

	return lines
}

func credits(amount int64) string { return fmt.Sprintf("CR %d", amount) }

// retirementPay is the pension in the one wording the summary and the sheet's
// benefits both write (#557).
//
// Implements: Retirement pay reads the same in the summary and the benefits
// [Sheet].
func retirementPay(pension int64) string { return credits(pension) + " a year retirement pay" }

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
// tell, how the service ended, and what mustering out changed (#490).
//
// Implements: The service record narrates the career a term at a time
// [Sheet].
//
// Implements: A characteristic raised at mustering out is on the service
// record [Sheet].
//
// Implements: Expertise taken at mustering out is on the service record
// [Sheet].
//
// Implements: A retirement the reenlistment throw refused says so on the
// service record [Sheet].
//
// Implements: A death after the service ended ends the service record [Sheet].
func serviceLines(r view) []string {
	if r.Service == "" {
		return []string{r.Enlistment.How}
	}

	entered := fmt.Sprintf("%s, %s", r.Enlistment.Service, r.Enlistment.How)

	entered = joinNonEmpty("; ", entered, strings.Join(r.History.OnEntering, ", "))

	terms := r.History.Terms

	lines := []string{entered}

	for _, term := range terms {
		told := termText(term)
		if told == "" {
			continue
		}

		lines = append(lines, fmt.Sprintf("Term %d: %s", term.Term, told))
	}

	// The departure is the last that ended the service, so a crisis that
	// kills a character the reenlistment throw has retired is told here, and
	// the retirement it overtook is not.
	switch {
	case r.Departure.Characteristic != "":
		lines = append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			r.Departure.How, r.Terms, r.Departure.Characteristic))
	case r.Departure.Fatal:
		lines = append(lines, fmt.Sprintf("%s in term %d", r.Departure.How, r.Terms))
	case r.History.RetiredDenied:
		lines = append(lines, fmt.Sprintf("%s after term %d, reenlistment denied", r.Departure.How, r.Terms))
	default:
		lines = append(lines, fmt.Sprintf("%s after term %d", r.Departure.How, r.Terms))
	}

	if len(r.History.MusteredOut) > 0 {
		lines = append(lines, "mustered out with "+strings.Join(r.History.MusteredOut, ", "))
	}

	return lines
}

// termText is one term's rank, gains, a reenlistment throw that required
// another term, aging and recovery, in the order the term ran, each part left
// out where the term has none of it, so a term with nothing to tell is empty.
// A rank's grants and a recovery's months are in parentheses, so "; " only
// ever separates the parts.
//
// Implements: A term a 12 on the reenlistment throw required says so on the
// service record [Sheet].
func termText(term termView) string {
	raised := make([]string, 0, len(term.Raised))
	for _, raise := range term.Raised {
		raised = append(raised, raise.How+" "+raise.Rank+labelled(" (", strings.Join(raise.Grants, ", "), ")"))
	}

	retained := ""
	if term.Retained {
		retained = "the service required another term"
	}

	return joinNonEmpty("; ",
		strings.Join(raised, " and "),
		strings.Join(term.Gained, ", "),
		retained,
		labelled("aging ", strings.Join(term.Aged, ", "), ""),
		strings.Join(term.Recovered, ", "))
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

// serving is the service and the rank held in it, as the profile line says
// them and a batch member's note is named by them.
func serving(r view) string {
	if r.RankName == "" {
		return r.Service
	}

	return r.Service + " " + r.RankName
}

// skillText is a skill in the book's notation, Skill-1 (p. 12): the one form
// the sheet's skills and the summary write. The service record writes a
// skill's level before and after (#562), and the properties write the name
// alone (#474).
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
