package decisions

import "fmt"

const (
	StatusPlanned    = "planned"
	StatusDone       = "done"
	StatusSkipped    = "skipped"
	StatusIrrelevant = "irrelevant"
)

// ValidStatuses are the allowed action lifecycle values.
var ValidStatuses = map[string]struct{}{
	StatusPlanned:    {},
	StatusDone:       {},
	StatusSkipped:    {},
	StatusIrrelevant: {},
}

// CanTransition reports whether from → to is allowed.
// planned may move to done/skipped/irrelevant; terminal statuses are fixed.
func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	if from != StatusPlanned {
		return false
	}
	_, ok := ValidStatuses[to]
	return ok && to != StatusPlanned
}

// ValidateStatus returns an error if status is unknown.
func ValidateStatus(status string) error {
	if _, ok := ValidStatuses[status]; !ok {
		return fmt.Errorf("invalid action status %q", status)
	}
	return nil
}
