package models

import (
	"encoding/json"
	"testing"
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
