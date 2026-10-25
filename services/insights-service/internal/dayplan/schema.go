package dayplan

import (
	"encoding/json"
	"fmt"
)

// schemaName names the structured-output schema in the request.
const schemaName = "day_plan"

// schema is the strict JSON schema the model answers in (R6). Strict mode
// requires every property to be required and no others allowed, so the two
// ids are required-but-nullable: item_id for an existing_item, parent_item_id
// for a new_step. It bounds groups and entries per group; the total of six
// across groups cannot be said in a schema and is validation's job (R9).
// There is no revision field: revisions are the server's (R10).
var schema = json.RawMessage(fmt.Sprintf(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["groups"],
  "properties": {
    "groups": {
      "type": "array",
      "maxItems": %[1]d,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["label", "entries"],
        "properties": {
          "label": {"type": "string", "description": "short topic label for the group, at most %[3]d characters"},
          "entries": {
            "type": "array",
            "minItems": 1,
            "maxItems": %[2]d,
            "items": {
              "type": "object",
              "additionalProperties": false,
              "required": ["kind", "item_id", "parent_item_id", "text", "reason", "minutes"],
              "properties": {
                "kind": {"type": "string", "enum": ["existing_item", "new_step"]},
                "item_id": {"type": ["string", "null"], "description": "existing_item: id of the open item to do today; null for new_step"},
                "parent_item_id": {"type": ["string", "null"], "description": "new_step: id of the open TOP-LEVEL item the new step goes under; null for existing_item"},
                "text": {"type": "string", "description": "existing_item: the item's text; new_step: the new step"},
                "reason": {"type": "string", "description": "one short line: why today"},
                "minutes": {"type": "integer", "description": "rough minutes this takes today, %[4]d to %[5]d"}
              }
            }
          }
        }
      }
    }
  }
}`, MaxGroups, MaxEntries, MaxLabelLength, MinMinutes, MaxMinutes))

// generatedEntry is one entry exactly as the model returned it, before
// validation.
type generatedEntry struct {
	Kind         string  `json:"kind"`
	ItemID       *string `json:"item_id"`
	ParentItemID *string `json:"parent_item_id"`
	Text         string  `json:"text"`
	Reason       string  `json:"reason"`
	Minutes      int     `json:"minutes"`
}

type generatedGroup struct {
	Label   string           `json:"label"`
	Entries []generatedEntry `json:"entries"`
}

type generatedPlan struct {
	Groups []generatedGroup `json:"groups"`
}

// parseGenerated decodes the model's output. Output that is not JSON of the
// schema's shape is an error, never repaired. Counts are not checked here:
// a group or entry beyond the bounds is dropped by validation (R9), not a
// failed generation.
func parseGenerated(content string) ([]generatedGroup, error) {
	var out generatedPlan
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, fmt.Errorf("output is not valid JSON for the schema: %w", err)
	}
	if out.Groups == nil {
		return nil, fmt.Errorf("output has no groups array")
	}
	return out.Groups, nil
}
