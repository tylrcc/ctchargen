package render

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// bestSkills is how many skills a summary names at most: enough to say what he
// is known for, few enough to read aloud.
const bestSkills = 3

// standsOut is the level a skill must reach for a summary to name it: a
// skill held at level 1 is one most characters have, and says little.
const standsOut = 2

// summary is the paragraph under a note's heading, which a referee can read
// aloud (#425): the service and rank, the terms, how he left and at what age,
// his best skills, and what he mustered out with that matters. The dead are
// told how and when they died (#418); a civilian, that he declined the
// draft. It says nothing the sheet below it doesn't.
//
// Implements: A note opens with a summary a referee can read aloud [Sheet].
func summary(r view) string {
	if r.Service == "" {
		return fmt.Sprintf("A civilian who declined the draft at %d.", r.Age.Years)
	}

	var sentences []string

	switch {
	case r.Departure == nil:
		// Generation gives every served character a departure; the sheet
		// reads one without it all the same, and so does this.
		sentences = append(sentences, fmt.Sprintf("%s, %s, aged %d.",
			serving(r), terms(r.Terms), r.Age.Years))
	case r.Departure.Fatal:
		sentences = append(sentences, fmt.Sprintf("%s, %s in term %d, aged %d.",
			serving(r), r.Departure.How, r.Terms, r.Age.Years))
	default:
		sentences = append(sentences, fmt.Sprintf("%s, %s, %s at %d.",
			serving(r), terms(r.Terms), r.Departure.How, r.Age.Years))
	}

	if known := best(r.Skills); len(known) > 0 {
		sentences = append(sentences, "Known for "+listed(known)+".")
	}

	if musters := notable(r.Benefits); len(musters) > 0 {
		sentences = append(sentences, "Musters out with "+listed(musters)+".")
	}

	return strings.Join(sentences, " ")
}

// best is the highest-level skills at standsOut or above, in the book's
// notation, at most bestSkills of them; skills of one level keep the sheet's
// order. A character with none above level 1 is known for nothing in
// particular, and the summary says nothing of his skills.
func best(skills []skillView) []string {
	ranked := slices.DeleteFunc(slices.Clone(skills), func(s skillView) bool { return s.Level < standsOut })
	slices.SortStableFunc(ranked, func(a, b skillView) int { return cmp.Compare(b.Level, a.Level) })

	named := make([]string, 0, bestSkills)
	for _, skill := range ranked[:min(bestSkills, len(ranked))] {
		named = append(named, skillText(skill.Name, skill.Level))
	}

	return named
}

// notable is what a mustered-out character is known to carry: his credits,
// Travellers' Aid, and any ship. Passages and weapons are on the sheet.
func notable(b benefitsView) []string {
	var named []string

	if b.Cash > 0 {
		named = append(named, credits(b.Cash))
	}

	if b.TravellersAid {
		named = append(named, "Travellers' Aid Society membership")
	}

	for _, ship := range b.Ships {
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
