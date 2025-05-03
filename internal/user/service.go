package user

import (
	"errors"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/internal/auth"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type service struct {
	Repo Repository
}

type Repository interface {
	Create(user models.User) error
	GetById(id uuid.UUID) (*models.User, error)
	GetAll() ([]*Response, error)
	GetUserByEmail(email string) (*models.User, error)
}

func NewService(repo Repository) Service {
	return &service{
		Repo: repo,
	}
}

func (s *service) GetById(id uuid.UUID) (*models.User, error) {
	return s.Repo.GetById(id)
}

func (s *service) Create(user models.User) error {
	hashedPw, err := s.HashPassword(user.Password)

	if err != nil {
		return fmt.Errorf("Error when attempting to hash password.")
	}

	// update user's password with hashed password.
	user.Password = hashedPw

	return s.Repo.Create(user)
}

// HashPassword hashes the given password using bcrypt.
func (s *service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *service) GetAll() ([]*Response, error) {
	return s.Repo.GetAll()
}

func (s *service) Login(loginReq LoginRequest) (*LoginResponse, error) {
	user, err := s.Repo.GetUserByEmail(loginReq.Email)

	if err != nil {
		return nil, errors.New("Could not get user with provided email.")
	}

	// extract password, and compare hashes
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		return nil, errors.New("The credentials provided was incorrect.")
	}

	// construct response with both user info and auth credentials
	accessExpiryTime := time.Minute * 60
	accessToken, err := auth.GenerateJWT(*user, auth.Access, accessExpiryTime)
	refreshExpiryTime := time.Hour * 24 * 7
	refreshToken, err := auth.GenerateJWT(*user, auth.Refresh, refreshExpiryTime)

	user.Password = ""

	res := &LoginResponse{
		AccessToken:      accessToken,
		AccessExpiresIn:  int(accessExpiryTime),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int(refreshExpiryTime),
		UserInfo:         user,
	}

	return res, nil
}
