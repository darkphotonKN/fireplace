package dayplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
)

// hashInput is everything the day plan's rule gate compares (R17): DF4ZB's
// plan-state hash of the tree, plus the plan's type and focus, the plan date,
// the items ticked today and the set of due revisions for the date, each
// sorted. Equal inputs mean the model would be asked the same question, so a
// stored ai plan can be served instead.
//
// Ticked and Revisions are omitted when empty, so a plan with neither hashes
// as it did before ticks and revisions existed. DailyMinutes, the user's
// daily time on the plan, is omitted when unset for the same reason: plans
// drawn before the setting existed keep their gate (FS-02JBZ R99).
type hashInput struct {
	Tree         string   `json:"tree"`
	PlanType     string   `json:"plan_type"`
	Focus        string   `json:"focus"`
	Date         string   `json:"date"`
	Ticked       []string `json:"ticked,omitempty"`
	Revisions    []string `json:"revisions,omitempty"`
	DailyMinutes *int     `json:"daily_minutes,omitempty"`
}

// planState is what the day plan's plan-state hash covers, as the draw reads
// it: the tree, the plan's profile, the plan date, the items ticked today and
// the revisions due on the date, each in any order, and the user's daily time
// on the plan, nil when unset.
type planState struct {
	Tree         guided.PlanTree
	Profile      PlanProfile
	Date         time.Time
	Ticked       []uuid.UUID
	Due          []dueRevision
	DailyMinutes *int
}

// hash is the day plan's plan-state hash of s: hashInput, canonicalised and
// digested.
func (s planState) hash() string {
	ids := make([]string, 0, len(s.Ticked))
	for _, id := range s.Ticked {
		ids = append(ids, id.String())
	}
	slices.Sort(ids)
	revs := make([]string, 0, len(s.Due))
	for _, r := range s.Due {
		revs = append(revs, fmt.Sprintf("%s:%d", r.ItemID, r.Rung))
	}
	slices.Sort(revs)
	// Marshalling a struct of strings and an int cannot fail.
	b, _ := json.Marshal(hashInput{
		Tree:         s.Tree.StateHash(),
		PlanType:     s.Profile.Type,
		Focus:        s.Profile.Focus,
		Date:         s.Date.Format(guided.DateLayout),
		Ticked:       ids,
		Revisions:    revs,
		DailyMinutes: s.DailyMinutes,
	})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
