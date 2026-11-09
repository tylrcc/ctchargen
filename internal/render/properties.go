package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// characteristicKeys are the six characteristics' property names, in the
// order p. 4 rolls them. They are the sheet's words, lowercased, and Social
// Standing is "social" so a key is one word a query can type.
//
//nolint:gochecknoglobals // a fixed table, and Go has no const map.
var characteristicKeys = map[traveller.Characteristic]string{
	traveller.Strength:       "strength",
	traveller.Dexterity:      "dexterity",
	traveller.Endurance:      "endurance",
	traveller.Intelligence:   "intelligence",
	traveller.Education:      "education",
	traveller.SocialStanding: "social",
}

// writeProperties opens the note with its properties: YAML frontmatter a
// vault's queries read (#399). Their names and meanings freeze at v1.0.0
// (ADR 0006), so the meanings are stated here:
//
//   - service, rank and terms are single values, since a character serves
//     one service (p. 5); skills and aliases are lists.
//   - age is completed years, without the months a crisis recovery adds.
//   - terms counts every term begun, the one a death ended included, as the
//     sheet counts them.
//   - status is alive or dead (#418).
//   - service, rank, departure and nobility are absent where there is none,
//     and aliases where no name was given; name, skills, credits and pension
//     are always written.
//
// Every text value is a double-quoted string, so no word a later build
// writes can change a property's type, and nothing a referee types is
// trusted; numbers are bare.
//
// Implements: A note opens with properties a vault can query [Sheet].
func writeProperties(out *strings.Builder, r view) {
	out.WriteString("---\n")

	if r.Name == "" {
		out.WriteString("name:\n")
	} else {
		fmt.Fprintf(out, "name: %s\naliases:\n  - %s\n", strconv.Quote(r.Name), strconv.Quote(r.Name))
	}

	fmt.Fprintf(out, "upp: %s\n", strconv.Quote(r.UPP))

	for _, c := range traveller.Characteristics {
		fmt.Fprintf(out, "%s: %d\n", characteristicKeys[c], r.Profile[c])
	}

	fmt.Fprintf(out, "age: %d\n", r.Age.Years)
	property(out, "service", r.Service)
	property(out, "rank", r.RankName)
	fmt.Fprintf(out, "terms: %d\nstatus: %s\n", r.Terms, strconv.Quote(status(r)))

	if r.Departure != nil {
		property(out, "departure", r.Departure.How)
	}

	if len(r.Skills) == 0 {
		out.WriteString("skills: []\n")
	} else {
		out.WriteString("skills:\n")

		for _, skill := range r.Skills {
			fmt.Fprintf(out, "  - %s\n", strconv.Quote(skillText(skill.Name, skill.Level)))
		}
	}

	fmt.Fprintf(out, "credits: %d\npension: %d\n", r.Benefits.Cash, r.Pension)

	if r.Nobility != nil && r.Nobility.Assumed {
		property(out, "nobility", r.Nobility.Title)
	}

	out.WriteString("---\n\n")
}

// property writes a key with its value quoted, or nothing where there is
// none.
func property(out *strings.Builder, key, value string) {
	if value != "" {
		fmt.Fprintf(out, "%s: %s\n", key, strconv.Quote(value))
	}
}

// status is whether the character lived, said outright (#418).
func status(r view) string {
	if r.Departure != nil && r.Departure.Fatal {
		return "dead"
	}

	return "alive"
}
