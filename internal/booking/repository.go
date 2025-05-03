package booking

import (
	"fmt"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) Create(userId uuid.UUID, req CreateRequest) error {
	query := `INSERT INTO bookings(user_id, start_date, end_date, status) VALUES (:userId, :startDate, :endDate, :status)`

	// create fields required for DB insert
	fields := map[string]interface{}{
		"userId":    userId,
		"startDate": req.StartDate,
		"endDate":   req.EndDate,
		"status":    req.Status,
	}

	fmt.Printf("fields: %+v", fields)

	_, err := r.DB.NamedExec(query, fields)

	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetById(userId uuid.UUID, id uuid.UUID) (*models.Booking, error) {
	// One to Many example
	query := `
	SELECT 
		bookings.id,
		bookings.start_date,
		bookings.end_date,
		bookings.status
	FROM bookings
	JOIN users ON users.id = bookings.user_id
	WHERE bookings.id = $1 AND bookings.user_id = $2
	`

	var booking models.Booking

	err := r.DB.Get(&booking, query, id, userId)

	if err != nil {
		return nil, err
	}

	return &booking, nil
}
