package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func Fingerprint(tx Transaction) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(tx.OrderAccount)),
		tx.BookingDate.Format("2006-01-02"),
		tx.ValueDate.Format("2006-01-02"),
		tx.Amount.StringFixed(2),
		strings.ToUpper(strings.TrimSpace(tx.Currency)),
		strings.TrimSpace(tx.BookingText),
		strings.TrimSpace(tx.Counterparty),
		strings.TrimSpace(tx.EndToEndReference),
		strings.TrimSpace(tx.Purpose),
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
