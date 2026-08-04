package finance

import "time"

const defaultCompleteMonthCount = 12

// DefaultPeriod chooses the window for baseline recompute.
// Prefer the last N complete calendar months when booking data reaches the
// latest complete month; otherwise fall back to the last 90 days ending at
// the latest booking (or now).
func DefaultPeriod(latestBooking, now time.Time) (from, to time.Time, assumption string) {
	now = dateOnly(now)
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthEnd := firstOfThisMonth.AddDate(0, 0, -1)
	lastMonthStart := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC)
	windowStart := lastMonthStart.AddDate(0, -(defaultCompleteMonthCount - 1), 0)

	if !latestBooking.IsZero() {
		latest := dateOnly(latestBooking)
		if !latest.Before(lastMonthStart) {
			return windowStart, lastMonthEnd, "period=last_12_complete_calendar_months"
		}
		to = latest
		from = to.AddDate(0, 0, -89)
		return from, to, "period=last_90_days"
	}

	return windowStart, lastMonthEnd, "period=last_12_complete_calendar_months"
}
