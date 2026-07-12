package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleViewer = "viewer"
	RoleTimer  = "timer"
	RoleAdmin  = "admin"
	RoleOwner  = "owner"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type LoginResponse struct {
	Token     string `json:"token"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}

type authUserRecord struct {
	id           string
	passwordHash []byte
	role         string
}

type AuthService struct {
	jwtCfg       *config.JWTConfig
	users        map[string]authUserRecord
	organizerPIN string
}

func NewAuthService(cfg *config.Config) *AuthService {
	users := parseAuthUsers(cfg.Auth.Users)
	pin := cfg.Auth.OrganizerPIN
	if pin == "" {
		pin = "1738"
	}
	return &AuthService{
		jwtCfg:       &cfg.JWT,
		users:        users,
		organizerPIN: pin,
	}
}

func parseAuthUsers(spec string) map[string]authUserRecord {
	if spec == "" {
		spec = "admin:admin:admin,timer:timer:timer,viewer:viewer:viewer"
	}

	users := make(map[string]authUserRecord)
	for _, entry := range strings.Split(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		if len(parts) != 3 {
			continue
		}
		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])
		role := strings.TrimSpace(parts[2])
		if username == "" || password == "" || role == "" {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			continue
		}
		users[username] = authUserRecord{
			id:           uuid.New().String(),
			passwordHash: hash,
			role:         role,
		}
	}
	return users
}

func (s *AuthService) Login(username, password string) (*LoginResponse, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword(user.passwordHash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.issueToken(user.id, username, user.role)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResponse{
		Token:     token,
		Role:      user.role,
		ExpiresAt: expiresAt,
	}, nil
}

// ExchangePIN validates the organizer PIN and issues an admin-role JWT so
// existing adminOnly / timerWrite middleware protects management routes.
func (s *AuthService) ExchangePIN(pin string) (*LoginResponse, error) {
	if pin != s.organizerPIN {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.issueToken(uuid.New().String(), "organizer", RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResponse{
		Token:     token,
		Role:      RoleAdmin,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) issueToken(userID, username, role string) (string, int64, error) {
	expiresAt := time.Now().Add(s.jwtCfg.AccessTokenTTL)
	claims := &AuthClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtCfg.Secret))
	if err != nil {
		return "", 0, err
	}
	return signed, expiresAt.Unix(), nil
}

func HasRole(actual string, allowed ...string) bool {
	for _, role := range allowed {
		if actual == role {
			return true
		}
	}
	return false
}
