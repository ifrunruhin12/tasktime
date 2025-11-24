package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/ifrunruhin12/tasktime/internal/auth"
)

type contextKey string

const (
	UsernameContextKey contextKey = "username"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.logger.Warn("Token validation failed: missing authorization header", map[string]interface{}{
				"ip":   r.RemoteAddr,
				"path": r.URL.Path,
			})
			sendErrorResponse(w, "UNAUTHORIZED", "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.logger.Warn("Token validation failed: invalid header format", map[string]interface{}{
				"ip":   r.RemoteAddr,
				"path": r.URL.Path,
			})
			sendErrorResponse(w, "UNAUTHORIZED", "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		username, err := s.authManager.ValidateToken(token)
		if err != nil {
			if err == auth.ErrTokenExpired {
				s.logger.Warn("Token validation failed: token expired", map[string]interface{}{
					"ip":   r.RemoteAddr,
					"path": r.URL.Path,
				})
				sendErrorResponse(w, "TOKEN_EXPIRED", "Token has expired", http.StatusUnauthorized)
				return
			}
			s.logger.Warn("Token validation failed: invalid token", map[string]interface{}{
				"ip":    r.RemoteAddr,
				"path":  r.URL.Path,
				"error": err.Error(),
			})
			sendErrorResponse(w, "UNAUTHORIZED", "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UsernameContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUsernameFromContext(r *http.Request) string {
	username, ok := r.Context().Value(UsernameContextKey).(string)
	if !ok {
		return ""
	}
	return username
}
