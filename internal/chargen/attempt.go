package chargen

import "github.com/philoserf/ctchargen/internal/traveller"

// Attempted is the service enlistment was attempted in, read from the
// enlistment throw's outcome in the record (#397). The draft can put him
// somewhere else, so it is not the service he served in. A character whose
// record holds no enlistment throw attempted none.
func (c *Character) Attempted() (traveller.ServiceName, bool) {
	for _, event := range c.Events {
		outcome, isOutcome := event.(traveller.OutcomeEvent)
		if !isOutcome {
			continue
		}

		if enlisted, ok := outcome.Fact.(traveller.EnlistedFact); ok {
			return enlisted.Service, true
		}
	}

	return 0, false
}
