package auth

import (
	"context"
	"fmt"
	"strings"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{DB: db}
}

// wrapDBErr is the repo boundary translation point: it converts infrastructure
// errors (sql.ErrNoRows, duplicate keys, constraint violations, transient
// failures) into domain sentinels via AnalyzeDBErr, and wraps anything else
// with the repo name + operation for context. It never logs and never decides
// transport status.
func wrapDBErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if mapped := commonhelpers.AnalyzeDBErr(err); mapped != err {
		return mapped
	}
	return fmt.Errorf("auth repo: %s: %w", op, err)
}

func (r *repository) Create(ctx context.Context, u *User) error {
	query := `INSERT INTO users (name, email, password)
	          VALUES ($1, $2, $3)
	          RETURNING id, created_at, updated_at`
	err := r.DB.QueryRowContext(ctx, query, u.Name, u.Email, u.HashedPassword).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	return wrapDBErr("create user "+u.Email, err)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, name, password, display_name, bio, created_at, updated_at
		 FROM users WHERE id = $1`, id)
	if err != nil {
		return nil, wrapDBErr("get user by id "+id.String(), err)
	}
	return &u, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, name, password, display_name, bio, created_at, updated_at
		 FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, wrapDBErr("get user by email", err)
	}
	return &u, nil
}

func (r *repository) ListAll(ctx context.Context) ([]*User, error) {
	var users []*User
	err := r.DB.SelectContext(ctx, &users,
		`SELECT id, email, name, '' AS password, display_name, bio, created_at, updated_at
		 FROM users
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, wrapDBErr("list users", err)
	}
	return users, nil
}

func (r *repository) UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error) {
	set := []string{}
	args := []any{in.ID}
	pos := 2

	if in.Name != nil {
		set = append(set, fmt.Sprintf("name = $%d", pos))
		args = append(args, *in.Name)
		pos++
	}
	if in.DisplayName != nil {
		set = append(set, fmt.Sprintf("display_name = $%d", pos))
		args = append(args, *in.DisplayName)
		pos++
	}
	if in.Bio != nil {
		set = append(set, fmt.Sprintf("bio = $%d", pos))
		args = append(args, *in.Bio)
		pos++
	}

	if len(set) == 0 {
		return r.GetByID(ctx, in.ID)
	}

	query := "UPDATE users SET " + strings.Join(set, ", ") + ", updated_at = NOW() WHERE id = $1"
	if _, err := r.DB.ExecContext(ctx, query, args...); err != nil {
		return nil, wrapDBErr("update profile "+in.ID.String(), err)
	}
	return r.GetByID(ctx, in.ID)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return wrapDBErr("delete user "+id.String(), err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}
