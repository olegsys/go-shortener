// Package middleware содержит HTTP-middleware для обработки запросов
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

// UserIDKey используется как ключ в контексте запроса для хранения идентификатора пользователя
const UserIDKey ctxKey = "userID"

// SignUserID подписывает идентификатор пользователя с помощью HMAC-SHA256
func SignUserID(secretKey, userID string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	return userID + "|" + hex.EncodeToString(h.Sum(nil))
}

// ValidateSignedUserID проверяет подписанный токен пользователя
func ValidateSignedUserID(secretKey, signed string) (string, bool) {
	parts := strings.SplitN(signed, "|", 2)
	if len(parts) != 2 {
		return "", false
	}

	id, sigHex := parts[0], parts[1]

	signature, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", false
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(id))

	if !hmac.Equal(signature, h.Sum(nil)) {
		return "", false
	}

	return id, true
}

// AuthMiddleware возвращает middleware, который извлекает или создает идентификатор пользователя
// Идентификатор хранится в cookie и подписывается HMAC-SHA256 для предотвращения компрометации
func AuthMiddleware(secretKey string, enableHTTPS bool) func(http.Handler) http.Handler {
	key := []byte(secretKey)
	_ = key

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID string
			var isNewUser bool

			cookie, err := r.Cookie("user_id")
			if err == nil {
				if id, ok := ValidateSignedUserID(secretKey, cookie.Value); ok {
					userID = id
				}
			}
			// New ID if new user or invalid cookie
			if userID == "" {
				userID = uuid.New().String()
				isNewUser = true
			}
			// Set cookie for new user
			if isNewUser {
				http.SetCookie(w, &http.Cookie{
					Name:     "user_id",
					Value:    SignUserID(secretKey, userID),
					Path:     "/",
					HttpOnly: true,
					Secure:   enableHTTPS,
				})
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
