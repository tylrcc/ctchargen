package chargen

import "github.com/philoserf/ctchargen/internal/traveller"

// Attempt is the service the enlistment throw was made against, and whether
// the policy named it (#397). The draft can put him somewhere else, so it is
// not the service he served in.
type Attempt struct {
	Service  traveller.ServiceName
	ByPolicy bool
}

// Attempted is the service enlistment was attempted in, read from the
// enlistment throw's outcome in the record. Whether the policy named it is read
// from the run instead: an unforced auto run is the policy's choice, and a
// --service or a guided answer is not. A character whose record holds no
// enlistment throw attempted none.
func (c *Character) Attempted() (Attempt, bool) {
	for _, event := range c.Events {
		outcome, isOutcome := event.(traveller.OutcomeEvent)
		if !isOutcome {
			continue
		}

		if enlisted, ok := outcome.Fact.(traveller.EnlistedFact); ok {
			_, forced := c.Inputs.Forced.Service()

			return Attempt{Service: enlisted.Service, ByPolicy: !forced && c.Policy != nil}, true
		}
	}

	return Attempt{}, false
}
