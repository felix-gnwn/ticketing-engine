package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

// Initiate global validator (Thread-safe, best practice is to put in package level)
var validate = validator.New()

// Request & Response Payloads
type RegisterRequest struct {
	Name		string	`json:"name" validate:"required"`
	Email		string	`json:"email" validate:"required,email"`
	Password	string	`json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email		string	`json:"email" validate:"required,email"`
	Password	string	`json:"password" validate:"required"`
}

type UserHandler struct {
	userUC domain.UserUseCase
}

func NewUserHandler(userUC domain.UserUseCase) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

// HTTP Handlers
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}

	if err := validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err) 
		return
	}

	user := &domain.User{
		Name:		req.Name,
		Email:		req.Email,
		Password:	req.Password,
	}

	if err := h.userUC.Register(r.Context(), user); err != nil {
		slog.Error("Failed to register user", "error", err, "email", req.Email)
		writeError(w, http.StatusInternalServerError, errors.New("failed to register user"))
		return
	}

	writeJSON(w, http.StatusCreated, jsonResponse{
		Status:  "success",
		Message: "User registered successfully",
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}

	if err := validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	token, err := h.userUC.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		slog.Warn("Failed login attempt", "email", req.Email, "error", err)
		writeError(w, http.StatusUnauthorized, errors.New("invalid email or password"))
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{
		Status:  "success",
		Message: "Login successful",
		Data: map[string]string{
			"token": token,
		},
	})
}