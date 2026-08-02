package classify

import (
	"strings"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const defaultTransferDateWindowDays = 3

// DetectTransferCandidates finds opposite-amount pairs across different accounts.
// existingLegs marks transaction IDs already in any transfer pair (skip those txs).
func DetectTransferCandidates(
	txs []domain.TransferScanTx,
	existingLegs map[uuid.UUID]struct{},
) []domain.TransferPair {
	if existingLegs == nil {
		existingLegs = map[uuid.UUID]struct{}{}
	}

	var suggestions []domain.TransferPair
	used := make(map[uuid.UUID]struct{}, len(existingLegs))
	for id := range existingLegs {
		used[id] = struct{}{}
	}

	for i := 0; i < len(txs); i++ {
		a := txs[i]
		if _, taken := used[a.ID]; taken {
			continue
		}
		if a.Amount.IsZero() {
			continue
		}

		bestJ := -1
		var best domain.TransferPair

		for j := i + 1; j < len(txs); j++ {
			b := txs[j]
			if _, taken := used[b.ID]; taken {
				continue
			}
			if a.AccountID == b.AccountID {
				continue
			}
			if !amountsOppositeEqual(a.Amount, b.Amount) {
				continue
			}

			dayDelta := absDays(a.BookingDate, b.BookingDate)
			if dayDelta > defaultTransferDateWindowDays {
				continue
			}

			confidence := domain.TransferConfidenceProbable
			if dayDelta == 0 {
				confidence = domain.TransferConfidenceExact
			}
			if looksLikeTransferText(a) || looksLikeTransferText(b) {
				if confidence == domain.TransferConfidenceProbable && dayDelta <= 1 {
					confidence = domain.TransferConfidenceExact
				}
			}

			out, in := orderLegs(a, b)
			candidate := domain.TransferPair{
				TxOutID:    out.ID,
				TxInID:     in.ID,
				Status:     domain.TransferStatusSuggested,
				Confidence: confidence,
				Rationale: map[string]any{
					"amount":           out.Amount.Abs().StringFixed(2),
					"day_delta":        dayDelta,
					"out_account_id":   out.AccountID.String(),
					"in_account_id":    in.AccountID.String(),
					"out_booking_date": out.BookingDate.Format("2006-01-02"),
					"in_booking_date":  in.BookingDate.Format("2006-01-02"),
				},
			}

			if bestJ < 0 || confidenceRank(candidate.Confidence) > confidenceRank(best.Confidence) {
				bestJ = j
				best = candidate
			}
		}

		if bestJ >= 0 {
			used[a.ID] = struct{}{}
			used[txs[bestJ].ID] = struct{}{}
			suggestions = append(suggestions, best)
		}
	}

	return suggestions
}

func amountsOppositeEqual(a, b decimal.Decimal) bool {
	if a.Sign() == 0 || b.Sign() == 0 {
		return false
	}
	if a.Sign() == b.Sign() {
		return false
	}
	return a.Abs().Equal(b.Abs())
}

func orderLegs(a, b domain.TransferScanTx) (out, in domain.TransferScanTx) {
	if a.Amount.IsNegative() {
		return a, b
	}
	return b, a
}

func absDays(a, b time.Time) int {
	a = time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	b = time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	d := int(a.Sub(b).Hours() / 24)
	if d < 0 {
		return -d
	}
	return d
}

func looksLikeTransferText(tx domain.TransferScanTx) bool {
	hay := strings.ToLower(tx.BookingText + " " + tx.Purpose + " " + tx.Counterparty)
	needles := []string{
		"umbuchung",
		"übertrag",
		"uebertrag",
		"überweisung",
		"ueberweisung",
		"eigenübertrag",
		"eigenuebertrag",
		"transfer",
	}
	for _, n := range needles {
		if strings.Contains(hay, n) {
			return true
		}
	}
	return false
}

func confidenceRank(c string) int {
	if c == domain.TransferConfidenceExact {
		return 2
	}
	return 1
}
