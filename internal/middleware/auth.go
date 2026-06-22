package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

type AuthConfig struct {
	SecretKey string
}

func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	key := []byte(secretKey)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID string
			var isNewUser bool

			cookie, err := r.Cookie("user_id")

			// Validate cookie
			if err == nil {
				parts := strings.Split(cookie.Value, "|")
				if len(parts) == 2 {
					id := parts[0]
					signature, _ := hex.DecodeString(parts[1])

					h := hmac.New(sha256.New, key)
					h.Write([]byte(id))

					if hmac.Equal(signature, h.Sum(nil)) {
						userID = id
					}
				}
			}
			// New ID if new user or invalid cookie
			if userID == "" {
				userID = uuid.New().String()
				isNewUser = true
			}
			// Set cookie for new user
			if isNewUser {
				h := hmac.New(sha256.New, key)
				h.Write([]byte(userID))
				signedValue := userID + "|" + hex.EncodeToString(h.Sum(nil))

				http.SetCookie(w, &http.Cookie{
					Name:     "user_id",
					Value:    signedValue,
					Path:     "/",
					HttpOnly: true,
					Secure:   true,
				})
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
