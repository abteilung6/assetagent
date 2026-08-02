package classify

import (
	"fmt"
	"sort"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type recurringIntervalSpec struct {
	Name       string
	TargetDays int
	Tolerance  int
	MinMembers int
}

var recurringIntervalSpecs = []recurringIntervalSpec{
	{Name: domain.RecurringIntervalMonthly, TargetDays: 30, Tolerance: 5, MinMembers: 3},
	{Name: domain.RecurringIntervalQuarterly, TargetDays: 91, Tolerance: 10, MinMembers: 3},
	{Name: domain.RecurringIntervalYearly, TargetDays: 365, Tolerance: 20, MinMembers: 2},
}

// DetectRecurringSeries finds monthly/quarterly/yearly payment series.
// existingMembers marks transaction IDs already assigned to a series.
// now is used to decide active vs ended status.
func DetectRecurringSeries(
	txs []domain.RecurringScanTx,
	existingMembers map[uuid.UUID]struct{},
	now time.Time,
) []domain.RecurringSeries {
	if existingMembers == nil {
		existingMembers = map[uuid.UUID]struct{}{}
	}
	now = dateOnly(now)

	type groupKey struct {
		pattern string
		sign    int
	}
	groups := map[groupKey][]domain.RecurringScanTx{}
	displayNames := map[string]string{}

	for _, tx := range txs {
		if _, taken := existingMembers[tx.ID]; taken {
			continue
		}
		if tx.Amount.IsZero() {
			continue
		}
		label, ok := NormalizeMerchantLabel(tx.Counterparty, tx.Purpose)
		if !ok {
			label, ok = NormalizeMerchantLabel(tx.BookingText, "")
		}
		if !ok {
			continue
		}
		key := groupKey{pattern: label.Pattern, sign: tx.Amount.Sign()}
		groups[key] = append(groups[key], tx)
		if _, seen := displayNames[label.Pattern]; !seen {
			displayNames[label.Pattern] = label.DisplayName
		}
	}

	var out []domain.RecurringSeries
	for key, members := range groups {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool {
			if members[i].BookingDate.Equal(members[j].BookingDate) {
				return members[i].ID.String() < members[j].ID.String()
			}
			return members[i].BookingDate.Before(members[j].BookingDate)
		})

		best := pickBestRecurringSeries(members, key.pattern, displayNames[key.pattern], key.sign, now)
		if best != nil {
			out = append(out, *best)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayName == out[j].DisplayName {
			return out[i].Fingerprint < out[j].Fingerprint
		}
		return out[i].DisplayName < out[j].DisplayName
	})
	return out
}

func pickBestRecurringSeries(
	members []domain.RecurringScanTx,
	pattern string,
	displayName string,
	sign int,
	now time.Time,
) *domain.RecurringSeries {
	var best *domain.RecurringSeries
	for _, spec := range recurringIntervalSpecs {
		candidate := buildRecurringSeries(members, spec, pattern, displayName, sign, now)
		if candidate == nil {
			continue
		}
		if best == nil ||
			candidate.MemberCount > best.MemberCount ||
			(candidate.MemberCount == best.MemberCount &&
				uncertaintyRank(candidate.Uncertainty) < uncertaintyRank(best.Uncertainty)) {
			best = candidate
		}
	}
	return best
}

func buildRecurringSeries(
	members []domain.RecurringScanTx,
	spec recurringIntervalSpec,
	pattern string,
	displayName string,
	sign int,
	now time.Time,
) *domain.RecurringSeries {
	chain := extractIntervalChain(members, spec)
	if len(chain) < spec.MinMembers {
		return nil
	}

	amounts := make([]decimal.Decimal, len(chain))
	ids := make([]uuid.UUID, len(chain))
	for i, tx := range chain {
		amounts[i] = tx.Amount.Abs()
		ids[i] = tx.ID
	}
	typical := medianDecimal(amounts)
	last := amounts[len(amounts)-1]
	changed := amountChanged(typical, last)

	kind := domain.RecurringKindFixed
	if sign > 0 {
		kind = domain.RecurringKindIncome
	} else if !amountsNearlyEqual(amounts) {
		kind = domain.RecurringKindVariableRegular
	}

	lastDate := dateOnly(chain[len(chain)-1].BookingDate)
	next := lastDate.AddDate(0, 0, spec.TargetDays)
	uncertainty := chainUncertainty(chain, spec)
	status := seriesStatus(lastDate, next, spec, uncertainty, now)

	signLabel := "expense"
	if sign > 0 {
		signLabel = "income"
	}
	fingerprint := fmt.Sprintf("%s|%s|%s", pattern, signLabel, spec.Name)

	return &domain.RecurringSeries{
		Fingerprint:   fingerprint,
		DisplayName:   displayName,
		Interval:      spec.Name,
		Kind:          kind,
		Status:        status,
		AmountTypical: typical,
		AmountLast:    last,
		AmountChanged: changed,
		NextExpected:  &next,
		Uncertainty:   uncertainty,
		MemberCount:   len(chain),
		MemberIDs:     ids,
	}
}

func extractIntervalChain(members []domain.RecurringScanTx, spec recurringIntervalSpec) []domain.RecurringScanTx {
	if len(members) == 0 {
		return nil
	}

	best := []domain.RecurringScanTx{members[0]}
	for start := 0; start < len(members); start++ {
		chain := []domain.RecurringScanTx{members[start]}
		expected := dateOnly(members[start].BookingDate).AddDate(0, 0, spec.TargetDays)
		for i := start + 1; i < len(members); i++ {
			day := dateOnly(members[i].BookingDate)
			delta := absDays(day, expected)
			if delta <= spec.Tolerance {
				chain = append(chain, members[i])
				expected = day.AddDate(0, 0, spec.TargetDays)
				continue
			}
			if day.Before(expected.AddDate(0, 0, -spec.Tolerance)) {
				continue
			}
			// Too far past expected — stop this chain.
			if day.After(expected.AddDate(0, 0, spec.Tolerance)) {
				break
			}
		}
		if len(chain) > len(best) {
			best = chain
		}
	}
	return best
}

func chainUncertainty(chain []domain.RecurringScanTx, spec recurringIntervalSpec) string {
	if len(chain) < 2 {
		return domain.RecurringUncertaintyHigh
	}
	var maxDelta int
	for i := 1; i < len(chain); i++ {
		delta := absDays(chain[i].BookingDate, chain[i-1].BookingDate)
		off := absInt(delta - spec.TargetDays)
		if off > maxDelta {
			maxDelta = off
		}
	}
	switch {
	case maxDelta <= spec.Tolerance/2:
		return domain.RecurringUncertaintyLow
	case maxDelta <= spec.Tolerance:
		return domain.RecurringUncertaintyMedium
	default:
		return domain.RecurringUncertaintyHigh
	}
}

func seriesStatus(
	lastDate time.Time,
	next time.Time,
	spec recurringIntervalSpec,
	uncertainty string,
	now time.Time,
) string {
	grace := spec.TargetDays + spec.Tolerance
	if lastDate.Before(now.AddDate(0, 0, -2*grace)) {
		return domain.RecurringStatusEnded
	}
	if uncertainty == domain.RecurringUncertaintyHigh {
		return domain.RecurringStatusUncertain
	}
	if next.Before(now.AddDate(0, 0, -grace)) {
		return domain.RecurringStatusUncertain
	}
	return domain.RecurringStatusActive
}

func amountChanged(typical, last decimal.Decimal) bool {
	if typical.IsZero() {
		return !last.IsZero()
	}
	diff := last.Sub(typical).Abs()
	if diff.LessThanOrEqual(decimal.NewFromFloat(1)) {
		return false
	}
	// >5% step
	return diff.GreaterThan(typical.Mul(decimal.NewFromFloat(0.05)))
}

func amountsNearlyEqual(amounts []decimal.Decimal) bool {
	if len(amounts) == 0 {
		return true
	}
	base := amounts[0]
	for _, a := range amounts[1:] {
		if a.Sub(base).Abs().GreaterThan(decimal.NewFromFloat(0.05)) {
			return false
		}
	}
	return true
}

func medianDecimal(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	sorted := append([]decimal.Decimal(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LessThan(sorted[j])
	})
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return sorted[mid-1].Add(sorted[mid]).Div(decimal.NewFromInt(2))
}

func uncertaintyRank(u string) int {
	switch u {
	case domain.RecurringUncertaintyLow:
		return 0
	case domain.RecurringUncertaintyMedium:
		return 1
	default:
		return 2
	}
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
