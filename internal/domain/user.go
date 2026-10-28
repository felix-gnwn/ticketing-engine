package domain

import (
	"context"
	"time"
)

type User struct {
	ID			string	`json:"id"`
	Name		string	`json:"name"`
	Email		string	`json:"email"`
	Password	string	`json:"-"`
	CreatedAt	string	`json:"created_at"`
	UpdatedAt	string	`json:"updated_at"`
}

type UserRepository interface {
	Create(ctx.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserUseCase interface {
	Register(ctx context.Context, user *User) error
	Login(ctx context.Context, email, password string) (string, error)
}