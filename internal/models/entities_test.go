package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUser_NullableProfileFields(t *testing.T) {
	user := User{
		Name:  "Test User",
		Email: "test@example.com",
	}

	if user.DisplayName != nil {
		t.Error("expected DisplayName to be nil by default")
	}
	if user.Bio != nil {
		t.Error("expected Bio to be nil by default")
	}

	dn := "Test DN"
	bio := "A short bio"
	user.DisplayName = &dn
	user.Bio = &bio

	if *user.DisplayName != "Test DN" {
		t.Errorf("expected DisplayName 'Test DN', got '%s'", *user.DisplayName)
	}
	if *user.Bio != "A short bio" {
		t.Errorf("expected Bio 'A short bio', got '%s'", *user.Bio)
	}
}

func TestUser_JSONSerialization_ProfileFields(t *testing.T) {
	user := User{Name: "Test", Email: "test@example.com"}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, exists := m["displayName"]; !exists {
		t.Error("expected displayName key in JSON output")
	}

	dn := "Display"
	bio := "Bio text"
	user.DisplayName = &dn
	user.Bio = &bio

	data, _ = json.Marshal(user)
	json.Unmarshal(data, &m)

	if m["displayName"] != "Display" {
		t.Errorf("expected displayName 'Display', got '%v'", m["displayName"])
	}
	if m["bio"] != "Bio text" {
		t.Errorf("expected bio 'Bio text', got '%v'", m["bio"])
	}
}

func TestChecklistItem_DateFields_NilByDefault(t *testing.T) {
	item := ChecklistItem{Description: "task"}

	if item.StartDate != nil {
		t.Error("expected StartDate to be nil by default")
	}
	if item.DueDate != nil {
		t.Error("expected DueDate to be nil by default")
	}
}

func TestChecklistItem_DateFields_JSONKeys(t *testing.T) {
	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	due := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	item := ChecklistItem{
		Description: "task",
		StartDate:   &start,
		DueDate:     &due,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := m["startDate"]; !ok {
		t.Error("expected startDate key in JSON output")
	}
	if _, ok := m["dueDate"]; !ok {
		t.Error("expected dueDate key in JSON output")
	}
	if _, ok := m["scheduledTime"]; ok {
		t.Error("scheduledTime key should not exist after migration")
	}
}

func TestChecklistItem_DateFields_OmittedWhenNil(t *testing.T) {
	item := ChecklistItem{Description: "task"}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := m["startDate"]; ok {
		t.Error("expected startDate to be omitted when nil")
	}
	if _, ok := m["dueDate"]; ok {
		t.Error("expected dueDate to be omitted when nil")
	}
}

func TestChecklistItem_TypeField_DefaultsAndJSONKey(t *testing.T) {
	// Type is a string with DB default 'task' — Go zero value is "" but the
	// shape must support both 'task' and 'note'.
	item := ChecklistItem{Description: "task", Type: "task"}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if m["type"] != "task" {
		t.Errorf("expected type 'task', got %v", m["type"])
	}

	noteItem := ChecklistItem{Description: "note", Type: "note"}
	data, _ = json.Marshal(noteItem)
	json.Unmarshal(data, &m)
	if m["type"] != "note" {
		t.Errorf("expected type 'note', got %v", m["type"])
	}
}

func TestChecklistItem_ParentID_NilByDefault(t *testing.T) {
	item := ChecklistItem{Description: "task"}
	if item.ParentID != nil {
		t.Error("expected ParentID to be nil by default")
	}
}

func TestChecklistItem_ParentID_JSONKeyAndOmitempty(t *testing.T) {
	// nil → omitted
	item := ChecklistItem{Description: "task"}
	data, _ := json.Marshal(item)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if _, ok := m["parentId"]; ok {
		t.Error("expected parentId to be omitted when nil")
	}

	// set → present
	parentID := uuid.New()
	item.ParentID = &parentID
	data, _ = json.Marshal(item)
	json.Unmarshal(data, &m)
	if m["parentId"] != parentID.String() {
		t.Errorf("expected parentId %s, got %v", parentID, m["parentId"])
	}
}
