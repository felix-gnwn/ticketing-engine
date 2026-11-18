package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase/mocks"
)

func TestUserUseCase_Register(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "supersecretjwtkey"

	tests := []struct {
		name         string
		inputUser    *domain.User
		setupMocks   func(userRepo *mocks.UserRepositoryMock)
		expectedRole string
		expectedErr  error
	}{
		{
			name: "Success - Register User with Default Role",
			inputUser: &domain.User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "password123",
				Role:     "",
			},
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("password123"))

					return err == nil && u.Role == "user" && u.ID != ""
				})).Return(nil).Once()
			},
			expectedRole: "user",
			expectedErr:  nil,
		},
		{
			name: "Success - Register User Preserving Existing Role",
			inputUser: &domain.User{
				Name:     "Admin User",
				Email:    "admin@example.com",
				Password: "adminpassword",
				Role:     "admin",
			},
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					return u.Role == "admin"
				})).Return(nil).Once()
			},
			expectedRole: "admin",
			expectedErr:  nil,
		},
		{
			name: "Error - Repository Failure",
			inputUser: &domain.User{
				Name:     "Fail User",
				Email:    "fail@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("duplicate email constraint")).Once()
			},
			expectedErr: errors.New("duplicate email constraint"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := new(mocks.UserRepositoryMock)
			tt.setupMocks(userRepo)

			uc := usecase.NewUserUseCase(userRepo, jwtSecret)
			err := uc.Register(ctx, tt.inputUser)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRole, tt.inputUser.Role)
			}

			userRepo.AssertExpectations(t)
		})
	}
}

func TestUserUseCase_Login(t *testing.T) {
	ctx := context.Background()
	jwtSecret := "supersecretjwtkey"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	validUser := &domain.User{
		ID:       "usr-777",
		Email:    "john@example.com",
		Password: string(hashedPassword),
		Role:     "user",
	}

	tests := []struct {
		name        string
		email       string
		password    string
		setupMocks  func(userRepo *mocks.UserRepositoryMock)
		expectToken bool
		expectedErr error
	}{
		{
			name:     "Success - Login Valid Credentials",
			email:    "john@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("GetByEmail", mock.Anything, "john@example.com").Return(validUser, nil).Once()
			},
			expectToken: true,
			expectedErr: nil,
		},
		{
			name:     "Error - User Email Not Found",
			email:    "notfound@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("GetByEmail", mock.Anything, "notfound@example.com").Return((*domain.User)(nil), errors.New("user not found")).Once()
			},
			expectToken: false,
			expectedErr: errors.New("invalid email or password"),
		},
		{
			name:     "Error - Wrong Password",
			email:    "john@example.com",
			password: "wrongpassword",
			setupMocks: func(userRepo *mocks.UserRepositoryMock) {
				userRepo.On("GetByEmail", mock.Anything, "john@example.com").Return(validUser, nil).Once()
			},
			expectToken: false,
			expectedErr: errors.New("invalid email or password"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := new(mocks.UserRepositoryMock)
			tt.setupMocks(userRepo)

			uc := usecase.NewUserUseCase(userRepo, jwtSecret)
			token, err := uc.Login(ctx, tt.email, tt.password)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			}

			userRepo.AssertExpectations(t)
		})
	}
}
