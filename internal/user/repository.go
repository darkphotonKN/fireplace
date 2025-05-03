package user

import (
	"fmt"

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

func (r *repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE users.email = $1`

	fmt.Println("Querying user with email:", email)

	err := r.DB.Get(&user, query, email)
	fmt.Println("Error:", err)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
