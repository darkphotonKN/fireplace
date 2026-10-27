package guided

import (
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

func TestNewDayKey_DateMustBeWithinOneDayOfServerUTC(t *testing.T) {
	// Already the 26th in a zone ahead of UTC: the bound is the UTC date, 2026-10-25.
	now := time.Date(2026, 10, 26, 3, 30, 0, 0, time.FixedZone("UTC+5", 5*60*60))
	tests := []struct {
		date    string
		wantErr error
	}{
		{"2026-10-24", nil},
		{"2026-10-25", nil},
		{"2026-10-26", nil},
		{"2026-10-23", ErrDateOutOfRange},
		{"2026-10-27", ErrDateOutOfRange},
		{"2025-10-25", ErrDateOutOfRange},
		{"25-10-2026", commonconstants.ErrInvalidInput},
		{"", commonconstants.ErrInvalidInput},
	}
	planID, userID := uuid.New(), uuid.New()
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			key, err := NewDayKey(planID, userID, tt.date, now)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if key.PlanID != planID || key.UserID != userID || key.Date.Format(DateLayout) != tt.date || key.Date.Location() != time.UTC {
				t.Fatalf("key = %+v, want (%s, %s, %s at midnight UTC)", key, planID, userID, tt.date)
			}
		})
	}
}
