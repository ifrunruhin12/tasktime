package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	BcryptCost        = 10
	MinPasswordLength = 8
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrMissingJWTSecret   = errors.New("JWT_SECRET environment variable not set")
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Manager struct {
	jwtSecret    []byte
	expiryDays   int
}

func NewManager(jwtSecret string, expiryDays int) (*Manager, error) {
	if jwtSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	if len(jwtSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters")
	}
	if expiryDays <= 0 {
		return nil, errors.New("JWT expiry days must be positive")
	}

	return &Manager{
		jwtSecret:  []byte(jwtSecret),
		expiryDays: expiryDays,
	}, nil
}

func (m *Manager) HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func (m *Manager) ComparePassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}
	return nil
}

func (m *Manager) GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(m.expiryDays) * 24 * time.Hour)

	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (m *Manager) ValidateToken(tokenString string) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrTokenExpired
		}
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	return claims.Username, nil
}
