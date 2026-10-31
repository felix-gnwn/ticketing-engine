package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// custom type to avoid key collision in context
type contextKey string

const (
	UserIDKey	contextKey = "userID"
	RoleKey		contextKey = "role"
)

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}
		tokenString := parts[1]

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			writeError(w, http.StatusInternalServerError, "server configuration error")
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, RoleKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			writeError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}
	}
}