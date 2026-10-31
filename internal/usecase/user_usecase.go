package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type userUseCase struct {
	userRepo	domain.UserRepository
	jwtSecret	string
}

func NewUserUseCase(repo domain.UserRepository, secret string) domain.UserUseCase {
	return &userUseCase{
		userRepo:	repo,
		jwtSecret:	secret,
	}
}

func (u *userUseCase) Register(ctx context.Context, user *domain.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.ID = uuid.New().String()
	user.Password = string(hashedPassword)
	if user.Role == "" {
		user.Role = "user"
	}
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	return u.userRepo.Create(ctx, user)
}

func (u *userUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	token, err := u.generateJWT(user)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (u *userUseCase) generateJWT(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":		user.ID,
		"role":		user.Role,
		"email":	user.Email,
		"exp":		time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(u.jwtSecret))
}