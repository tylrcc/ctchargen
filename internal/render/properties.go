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
// (ADR 0006), so the meanings are stated here, decided on #474:
//
//   - Every property is written on every note. Text with no value is an
//     empty key (name:, service:), and a list with none is [] (#480, #477).
//   - name and aliases: the name given, and that name as the one alias;
//     unnamed, name is empty and aliases [].
//   - upp, and the six characteristics as numbers.
//   - age is completed years, without the months a crisis recovery adds.
//   - service and rank are single values, since a character serves one
//     service (p. 5); empty for a civilian, and rank empty below a commission.
//   - terms counts every term begun, the one a death ended included, as the
//     sheet counts them.
//   - status is alive or dead (#418), and departure how the service ended,
//     empty for a civilian.
//   - skills is the skill names, one exact item each, with no level: the
//     sheet has the levels (#474).
//   - credits and pension are numbers.
//   - nobility is the title assumed, as Book 3 p. 22 pairs it (knight/dame),
//     or empty.
//   - possessions is what mustering out gave besides cash, one item each:
//     two passages are two items, and a Free Trader received again is still
//     one (#475).
//   - ctchargen is the build that wrote the note, or empty where none is
//     embedded (#476).
//
// Every text value is a double-quoted string, so no word a later build
// writes can change a property's type, and nothing a referee types is
// trusted; numbers are bare.
//
// Implements: A note opens with properties a vault can query [Sheet].
func writeProperties(out *strings.Builder, r view) {
	out.WriteString("---\n")

	property(out, "name", r.Name)

	if r.Name == "" {
		list(out, "aliases", nil)
	} else {
		list(out, "aliases", []string{r.Name})
	}

	property(out, "upp", r.UPP)

	for _, c := range traveller.Characteristics {
		fmt.Fprintf(out, "%s: %d\n", characteristicKeys[c], r.Profile[c])
	}

	fmt.Fprintf(out, "age: %d\n", r.Age.Years)
	property(out, "service", r.Service)
	property(out, "rank", r.RankName)
	fmt.Fprintf(out, "terms: %d\n", r.Terms)
	property(out, "status", status(r))
	property(out, "departure", r.Departure.How)

	names := make([]string, 0, len(r.Skills))
	for _, skill := range r.Skills {
		names = append(names, skill.Name)
	}

	list(out, "skills", names)
	fmt.Fprintf(out, "credits: %d\npension: %d\n", r.Benefits.Cash, r.Pension)

	nobility := ""
	if r.Nobility != nil && r.Nobility.Assumed {
		nobility = r.Nobility.Title
	}

	property(out, "nobility", nobility)
	list(out, "possessions", possessions(r.Benefits))
	property(out, "ctchargen", r.Build)

	out.WriteString("---\n\n")
}

// property writes a key with its value quoted, or the key alone where there
// is none, so every note has every key (#480).
func property(out *strings.Builder, key, value string) {
	if value == "" {
		fmt.Fprintf(out, "%s:\n", key)

		return
	}

	fmt.Fprintf(out, "%s: %s\n", key, strconv.Quote(value))
}

// list writes a key with its items quoted, one each, or [] where there are
// none.
func list(out *strings.Builder, key string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(out, "%s: []\n", key)

		return
	}

	fmt.Fprintf(out, "%s:\n", key)

	for _, item := range items {
		fmt.Fprintf(out, "  - %s\n", strconv.Quote(item))
	}
}

// possessions is what mustering out gave besides cash, one item each, in the
// order the sheet lists them: passages, Travellers' Aid, weapons, ships
// (#475).
//
// Implements: Possessions are one item each [Sheet].
func possessions(b benefitsView) []string {
	items := append([]string(nil), b.Passages...)

	if b.TravellersAid {
		items = append(items, "Travellers' Aid Society")
	}

	items = append(items, b.Weapons...)

	for _, ship := range b.Ships {
		items = append(items, ship.Item)
	}

	return items
}

// status is whether the character lived, said outright (#418).
func status(r view) string {
	if r.Departure.Fatal {
		return "dead"
	}

	return "alive"
}
