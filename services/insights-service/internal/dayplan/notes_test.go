package dayplan

import (
	"strings"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
)

// FS-PGG2N R45: the day plan reads notes through the plan tree, which leaves
// them out, so a long note changes neither the day plan's prompt nor its
// plan-state hash, and so never triggers a regeneration.
func TestDayPlan_ANotesBodyChangesNeitherPromptNorHash(t *testing.T) {
	tf := newTreeFixture()
	date := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	profile := PlanProfile{Type: "learning", Focus: "Graphs"}
	longBody := strings.Repeat("a very long note body ", 400)

	withLongNote := tf.items()
	for i := range withLongNote {
		if withLongNote[i].ID == tf.note {
			withLongNote[i].Description = longBody
		}
	}
	before, after := guided.NewPlanTree(tf.items()), guided.NewPlanTree(withLongNote)

	if (planState{Tree: before, Profile: profile, Date: date}).hash() != (planState{Tree: after, Profile: profile, Date: date}).hash() {
		t.Fatal("a note's body changed the day plan's state hash")
	}
	a := buildPrompt(promptInput{Tree: before, Profile: profile, Date: date})
	b := buildPrompt(promptInput{Tree: after, Profile: profile, Date: date})
	if a != b || strings.Contains(b.User, "very long note body") || strings.Contains(a.User, "remember the visited set") {
		t.Fatal("a note's text reached the day plan's prompt")
	}
}
