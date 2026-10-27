package guided

import (
	"errors"
	"fmt"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// DateLayout is the wire and storage form of a per-day key's date (YYYY-MM-DD).
const DateLayout = "2006-01-02"

// ErrDateOutOfRange: the date is more than one day from the server's UTC date
// (FS-DF4ZB R21).
var ErrDateOutOfRange = errors.New("guided: date out of range")

// NewDayKey parses date (YYYY-MM-DD) into a DayKey and checks it lies within
// one day of now's UTC date, inclusive (R21). Out of range it fails with
// ErrDateOutOfRange; unparsable, with ErrInvalidInput. Every per-day surface
// (the day plan, its ticks) keys and bounds its date this way.
func NewDayKey(planID, userID uuid.UUID, date string, now time.Time) (DayKey, error) {
	d, err := time.Parse(DateLayout, date)
	if err != nil {
		return DayKey{}, fmt.Errorf("%w: date %q is not YYYY-MM-DD", commonconstants.ErrInvalidInput, date)
	}
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(today.AddDate(0, 0, -1)) || d.After(today.AddDate(0, 0, 1)) {
		return DayKey{}, fmt.Errorf("%w: %s is not within one day of %s", ErrDateOutOfRange, date, today.Format(DateLayout))
	}
	return DayKey{PlanID: planID, UserID: userID, Date: d}, nil
}
