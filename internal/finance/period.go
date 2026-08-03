package finance

import "time"

// DefaultPeriod chooses the window for baseline recompute.
// Prefer the last complete calendar month when booking data reaches it;
// otherwise fall back to the last 90 days ending at the latest booking (or now).
func DefaultPeriod(latestBooking, now time.Time) (from, to time.Time, assumption string) {
	now = dateOnly(now)
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthEnd := firstOfThisMonth.AddDate(0, 0, -1)
	lastMonthStart := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC)

	if !latestBooking.IsZero() {
		latest := dateOnly(latestBooking)
		if !latest.Before(lastMonthStart) {
			return lastMonthStart, lastMonthEnd, "period=last_complete_calendar_month"
		}
		to = latest
		from = to.AddDate(0, 0, -89)
		return from, to, "period=last_90_days"
	}

	to = lastMonthEnd
	from = lastMonthStart
	return from, to, "period=last_complete_calendar_month"
}
