package decisions

import "testing"

func TestCanTransition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		from, to string
		want     bool
	}{
		{StatusPlanned, StatusDone, true},
		{StatusPlanned, StatusSkipped, true},
		{StatusPlanned, StatusIrrelevant, true},
		{StatusPlanned, StatusPlanned, true},
		{StatusDone, StatusSkipped, false},
		{StatusSkipped, StatusPlanned, false},
		{StatusIrrelevant, StatusDone, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
