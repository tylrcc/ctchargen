package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// EventLine renders one event for a reader watching a generation happen
// rather than reading it afterwards.
//
// Throws, rolls and outcomes go through the same writeEvent the generation
// record does, from the same entry, so those lines cannot come to differ. A
// step is a plain numbered line here, where the note makes it a heading.
//
// A choice is written here instead, and deliberately does differ. It is one
// the player has just answered; leaving it out would leave a gap a watcher
// reads as a lost event, so it is written with its number. The choice is
// echoed as that number and the answer alone rather than the record's full
// form, because the player has just been asked the question and can still
// see the point, the decider and the alternatives above him; repeating them
// pushes the useful lines off the screen.
//
// Implements: The procedure is shown between the questions [Generation record].
//
// Implements: The live view marks a step with no markdown [Generation record].
func EventLine(event traveller.Event) string {
	lined := entryOf(event)

	// Echoed after the answer, because a choice's number does not exist
	// until it is made - the prompt cannot carry it. The record's own line
	// names the point, the decider and the alternatives, all of which the
	// person who just typed the answer can see above.
	if lined.Kind == kindChoice {
		return fmt.Sprintf("%d. you chose %s\n", lined.Seq, lined.Chosen)
	}

	// A step is a plain numbered line here: the note makes it a heading, and a
	// terminal would print the marks rather than render them (#422).
	if lined.Kind == kindStep {
		return "\n" + stepLine(lined) + "\n\n"
	}

	var out strings.Builder

	writeEvent(&out, lined, "")

	return out.String()
}
