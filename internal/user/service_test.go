package user

import (
	"errors"
	"testing"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

// --- Mock Repository ---

type mockRepository struct {
	users map[uuid.UUID]*models.User
}

func newMockRepo() *mockRepository {
	return &mockRepository{users: make(map[uuid.UUID]*models.User)}
}

func (m *mockRepository) Create(user models.User) error { return nil }
func (m *mockRepository) GetAll() ([]*Response, error)  { return nil, nil }
func (m *mockRepository) GetUserByEmail(email string) (*models.User, error) {
	return nil, nil
}

func (m *mockRepository) GetById(id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockRepository) UpdateProfile(id uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.DisplayName != nil {
		u.DisplayName = req.DisplayName
	}
	if req.Bio != nil {
		u.Bio = req.Bio
	}
	u.Password = ""
	return u, nil
}

// --- Tests ---

func TestGetProfile_ReturnsUserData(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	id := uuid.New()
	dn := "Display"
	bio := "A bio"
	repo.users[id] = &models.User{
		BaseDBDateModel: models.BaseDBDateModel{ID: id},
		Name:            "Test User",
		Email:           "test@example.com",
		Password:        "hashedpw",
		DisplayName:     &dn,
		Bio:             &bio,
	}

	profile, err := svc.GetProfile(id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", profile.Name)
	}
	if profile.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", profile.Email)
	}
	if profile.Password != "" {
		t.Error("expected password to be stripped")
	}
	if *profile.DisplayName != "Display" {
		t.Errorf("expected display name 'Display', got '%s'", *profile.DisplayName)
	}
	if *profile.Bio != "A bio" {
		t.Errorf("expected bio 'A bio', got '%s'", *profile.Bio)
	}
}

func TestUpdateProfile_PartialUpdate_DisplayNameOnly(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	id := uuid.New()
	repo.users[id] = &models.User{
		BaseDBDateModel: models.BaseDBDateModel{ID: id},
		Name:            "Original",
		Email:           "test@example.com",
	}

	dn := "New Display Name"
	updated, err := svc.UpdateProfile(id, UpdateProfileRequest{DisplayName: &dn})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *updated.DisplayName != "New Display Name" {
		t.Errorf("expected display name 'New Display Name', got '%s'", *updated.DisplayName)
	}
	if updated.Name != "Original" {
		t.Errorf("expected name to remain 'Original', got '%s'", updated.Name)
	}
}

func TestUpdateProfile_RejectsEmptyName(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	id := uuid.New()
	repo.users[id] = &models.User{
		BaseDBDateModel: models.BaseDBDateModel{ID: id},
		Name:            "Original",
		Email:           "test@example.com",
	}

	empty := ""
	_, err := svc.UpdateProfile(id, UpdateProfileRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if err.Error() != "name cannot be empty" {
		t.Errorf("expected 'name cannot be empty', got '%s'", err.Error())
	}
}

func TestUpdateProfile_UpdatesMultipleFields(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	id := uuid.New()
	repo.users[id] = &models.User{
		BaseDBDateModel: models.BaseDBDateModel{ID: id},
		Name:            "Original",
		Email:           "test@example.com",
	}

	name := "New Name"
	dn := "New DN"
	bio := "New Bio"
	updated, err := svc.UpdateProfile(id, UpdateProfileRequest{
		Name:        &name,
		DisplayName: &dn,
		Bio:         &bio,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", updated.Name)
	}
	if *updated.DisplayName != "New DN" {
		t.Errorf("expected display name 'New DN', got '%s'", *updated.DisplayName)
	}
	if *updated.Bio != "New Bio" {
		t.Errorf("expected bio 'New Bio', got '%s'", *updated.Bio)
	}
}
