package user

import (
	"strings"

	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{
		DB: db,
	}
}

func (r *repository) Create(user models.User) error {
	query := `INSERT INTO users (name, email, password) VALUES (:name, :email, :password)`

	_, err := r.DB.NamedExec(query, user)

	if err != nil {
		return err
	}

	return nil
}

func (r *repository) GetById(id uuid.UUID) (*models.User, error) {
	query := `SELECT * FROM users WHERE users.id = $1`

	var user models.User

	err := r.DB.Get(&user, query, id)

	if err != nil {
		return nil, err
	}

	// Remove password from the struct
	user.Password = ""

	return &user, nil
}

func (r *repository) GetAll() ([]*Response, error) {
	query := `
	SELECT
		users.id,
		users.name,
		users.email,
		users.display_name,
		users.bio,
		users.created_at,
		users.updated_at
	FROM users
	`

	var users []*Response
	if err := r.DB.Select(&users, query); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *repository) UpdateProfile(id uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	setClauses := []string{}
	args := map[string]interface{}{"id": id}

	if req.Name != nil {
		setClauses = append(setClauses, "name = :name")
		args["name"] = *req.Name
	}
	if req.DisplayName != nil {
		setClauses = append(setClauses, "display_name = :display_name")
		args["display_name"] = *req.DisplayName
	}
	if req.Bio != nil {
		setClauses = append(setClauses, "bio = :bio")
		args["bio"] = *req.Bio
	}

	if len(setClauses) == 0 {
		return r.GetById(id)
	}

	query := "UPDATE users SET " + strings.Join(setClauses, ", ") + ", updated_at = NOW() WHERE id = :id"
	_, err := r.DB.NamedExec(query, args)
	if err != nil {
		return nil, err
	}

	return r.GetById(id)
}

func (r *repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE users.email = $1`

	logger.Debug("Querying user with email", "email", email)

	err := r.DB.Get(&user, query, email)
	if err != nil {
		logger.Error("Error querying user by email", "error", err, "email", email)
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}
