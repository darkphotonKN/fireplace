package dayplan

import (
	"testing"

	"github.com/google/uuid"
)

// goldenStateHash is goldenInput's plan-state hash, with its ticked entry and
// due revision, before FS-02JBZ §Change: Hours a day.
const goldenStateHash = "774ddd05558e44c4af007712f00c702f2b3b4a1690f610e9c204145b51d7305e"

func goldenStateHashWith(dailyMinutes *int) string {
	in := goldenInput()
	return planState{
		Tree: in.Tree, Profile: in.Profile, Date: in.Date, Ticked: []uuid.UUID{goldenBFS}, Due: in.Due, DailyMinutes: dailyMinutes,
	}.hash()
}

// An unset daily time hashes exactly as before, so plans drawn before the
// setting existed stay gated (R99).
func TestStateHash_NoDailyMinutes_IsTheHashFromBeforeTheChange(t *testing.T) {
	if got := goldenStateHashWith(nil); got != goldenStateHash {
		t.Fatalf("hash = %s, want the pre-change %s", got, goldenStateHash)
	}
}

func TestStateHash_DailyMinutesIsPartOfTheHash(t *testing.T) {
	unset, two, three := goldenStateHashWith(nil), goldenStateHashWith(ptr(120)), goldenStateHashWith(ptr(180))

	if unset == two || two == three || unset == three {
		t.Fatalf("hashes unset %s, 120 %s, 180 %s, want each distinct", unset, two, three)
	}
	if again := goldenStateHashWith(ptr(120)); again != two {
		t.Fatal("an unchanged daily time must hash the same")
	}
}
