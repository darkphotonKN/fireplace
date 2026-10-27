package guided

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
)

// StateHash is the plan-state hash (R19): a digest of the tree's model-visible
// fields (each item's id, parent id, description, done and due date), with the
// items ordered by id. Two trees the model would see
// identically hash identically, whatever order plan-service returned them in;
// any change the model would see changes the hash.
//
// It is a function of the tree alone so any flow that sends this tree to a
// model can gate on it.
func (t PlanTree) StateHash() string {
	items := slices.Clone(t.Items)
	slices.SortFunc(items, func(a, b TreeItem) int { return bytes.Compare(a.ID[:], b.ID[:]) })
	sum := sha256.Sum256([]byte(serializeTree(PlanTree{Items: items})))
	return hex.EncodeToString(sum[:])
}

// treeLine is one plan-tree item as the hash reads it. Field order is the
// serialization order.
type treeLine struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Done     bool   `json:"done"`
	Due      string `json:"due,omitempty"`
	Text     string `json:"text"`
}

// serializeTree writes the tree compactly, one JSON object per line.
func serializeTree(tree PlanTree) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, it := range tree.Items {
		line := treeLine{ID: it.ID.String(), Done: it.Done, Text: it.Description}
		if it.ParentID != nil {
			line.ParentID = it.ParentID.String()
		}
		if it.DueDate != nil {
			line.Due = it.DueDate.UTC().Format(DateLayout)
		}
		// Encoding a struct of strings and a bool cannot fail.
		_ = enc.Encode(line)
	}
	return buf.String()
}
