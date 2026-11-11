package render

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// bestSkills is how many skills a summary names at most: enough to say what
// the character can do, few enough to read aloud.
const bestSkills = 3

// summary is the paragraph under the profile line, which a referee can read
// aloud (#425, #557). The profile line has just given the service, rank,
// terms, any title assumed, departure and age, so this repeats them only for
// the dead, who are told first how and when they died (#418); the title is
// the profile line's alone. Then it names the highest-level skills, then
// what was mustered out with that matters: cash, retirement pay, Travellers'
// Aid and a ship. A civilian is told he declined the draft. It says nothing
// the sheet doesn't, and a character with nothing to tell gets no summary at
// all.
//
// Implements: The profile line opens the sheet, and a summary a referee can
// read aloud follows it [Sheet].
func summary(r view) string {
	if r.Service == "" {
		return fmt.Sprintf("A civilian who declined the draft at %d.", r.Age.Years)
	}

	var sentences []string

	if r.Departure.Fatal {
		sentences = append(sentences, fmt.Sprintf("%s in term %d, aged %d.",
			capitalised(r.Departure.How), r.Terms, r.Age.Years))
	}

	if known := best(r.Skills); len(known) > 0 {
		sentences = append(sentences, "Skills include "+listed(known)+".")
	}

	if musters := notable(r.Benefits, r.Pension); len(musters) > 0 {
		sentences = append(sentences, "Musters out with "+listed(musters)+".")
	}

	return strings.Join(sentences, " ")
}

// capitalised is a departure's words opening a sentence. Every departure is
// worded in lower case ASCII, so the first byte is the first letter.
func capitalised(words string) string {
	if words == "" {
		return words
	}

	return strings.ToUpper(words[:1]) + words[1:]
}

// best is the highest-level skills, at any level, in the book's notation, at
// most bestSkills of them; skills of one level keep the sheet's order (#558).
// A character with no skills has none named.
func best(skills []skillView) []string {
	ranked := slices.Clone(skills)
	slices.SortStableFunc(ranked, func(a, b skillView) int { return cmp.Compare(b.Level, a.Level) })

	named := make([]string, 0, bestSkills)
	for _, skill := range ranked[:min(bestSkills, len(ranked))] {
		named = append(named, skillText(skill.Name, skill.Level))
	}

	return named
}

// notable is what a mustered-out character is known to carry: credits,
// retirement pay, Travellers' Aid and any ship. Passages and weapons are on
// the sheet. The pay follows the cash, so a ship's ", Type A" comes last and
// never splits the list.
func notable(benefits benefitsView, pension int64) []string {
	var named []string

	if benefits.Cash > 0 {
		named = append(named, credits(benefits.Cash))
	}

	if pension > 0 {
		named = append(named, retirementPay(pension))
	}

	if benefits.TravellersAid {
		named = append(named, "Travellers' Aid Society membership")
	}

	for _, ship := range benefits.Ships {
		named = append(named, shipWords(ship))
	}

	return named
}

// listed joins words as prose does: "a", "a and b", "a, b and c".
func listed(words []string) string {
	if len(words) <= 1 {
		return strings.Join(words, "")
	}

	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// shipWords is a ship as a summary names it. A scout ship is held, not owned
// (p. 23), and a summary read aloud says so, as the sheet does.
func shipWords(ship shipView) string {
	if ship.Kind == traveller.ScoutShip.String() {
		return "a " + ship.Kind + ", " + heldNotOwned
	}

	return "a " + ship.Kind
}
