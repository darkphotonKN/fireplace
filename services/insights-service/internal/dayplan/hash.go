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
// as it did before ticks and revisions existed.
type hashInput struct {
	Tree      string   `json:"tree"`
	PlanType  string   `json:"plan_type"`
	Focus     string   `json:"focus"`
	Date      string   `json:"date"`
	Ticked    []string `json:"ticked,omitempty"`
	Revisions []string `json:"revisions,omitempty"`
}

// stateHash is the day plan's plan-state hash.
func stateHash(tree guided.PlanTree, profile PlanProfile, date time.Time, ticked []uuid.UUID, due []dueRevision) string {
	ids := make([]string, 0, len(ticked))
	for _, id := range ticked {
		ids = append(ids, id.String())
	}
	slices.Sort(ids)
	revs := make([]string, 0, len(due))
	for _, r := range due {
		revs = append(revs, fmt.Sprintf("%s:%d", r.ItemID, r.Rung))
	}
	slices.Sort(revs)
	// Marshalling a struct of strings cannot fail.
	b, _ := json.Marshal(hashInput{
		Tree:      tree.StateHash(),
		PlanType:  profile.Type,
		Focus:     profile.Focus,
		Date:      date.Format(guided.DateLayout),
		Ticked:    ids,
		Revisions: revs,
	})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
