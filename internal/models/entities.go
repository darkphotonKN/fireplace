package models

import (
	"time"

	"github.com/google/uuid"
)

/**
* Shared entities that are imported by more than one package.
**/
type User struct {
	BaseDBDateModel
	Email    string `db:"email" json:"email"`
	Name     string `db:"name" json:"name"`
	Password string `db:"password" json:"password,omitempty"`
}

type Booking struct {
	BaseDBUserDateModel
	// FK to User, One to Many Relation
	UserID    uuid.UUID `db:"user_id" json:"userId"`
	Status    string    `db:"status" json:"status"`
	StartDate time.Time `db:"start_date" json:"startDate"`
	EndDate   time.Time `db:"end_date" json:"endDate"`
}

type Plan struct {
	BaseDBDateModel
	UserID      uuid.UUID `db:"user_id" json:"userId"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	PlanType    string    `db:"plan_type" json:"planType"`
}

type ChecklistItem struct {
	BaseDBDateModel
	Description string    `db:"description" json:"description"`
	Done        string    `db:"done" json:"done"`
	Sequence    string    `db:"sequence" json:"sequence"`
	PlanID      uuid.UUID `db:"plan_id" json:"planId"`
}

/**
* Base models for default table columns.
**/
type BaseDBUserModel struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UpdatedUser uuid.UUID `db:"updated_user" json:"updatedUser"`
	CreatedUser uuid.UUID `db:"created_user" json:"createdUser"`
}

type BaseDBUserDateModel struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UpdatedUser uuid.UUID `db:"updated_user" json:"updatedUser"`
	CreatedUser uuid.UUID `db:"created_user" json:"createdUser"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type BaseDBDateModel struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
