package services

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAuthConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
		Auth: config.AuthConfig{
			Users: "admin:admin123:admin,timer:timer123:timer,viewer:viewer123:viewer",
		},
	}
}

func TestAuthService_LoginSuccess(t *testing.T) {
	svc := NewAuthService(testAuthConfig())

	resp, err := svc.Login("admin", "admin123")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, RoleAdmin, resp.Role)
	assert.True(t, resp.ExpiresAt > time.Now().Unix())
}

func TestAuthService_LoginInvalidCredentials(t *testing.T) {
	svc := NewAuthService(testAuthConfig())

	_, err := svc.Login("admin", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Login("nobody", "admin123")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_ValidateTokenRoundTrip(t *testing.T) {
	svc := NewAuthService(testAuthConfig())

	resp, err := svc.Login("timer", "timer123")
	require.NoError(t, err)

	claims, err := svc.ValidateToken(resp.Token)
	require.NoError(t, err)
	assert.Equal(t, "timer", claims.Username)
	assert.Equal(t, RoleTimer, claims.Role)
	assert.NotEmpty(t, claims.UserID)
}

func TestAuthService_ValidateTokenInvalid(t *testing.T) {
	svc := NewAuthService(testAuthConfig())

	_, err := svc.ValidateToken("not-a-jwt")
	assert.ErrorIs(t, err, ErrInvalidToken)

	other := NewAuthService(&config.Config{
		JWT: config.JWTConfig{
			Secret:          "different-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24,
		},
		Auth: config.AuthConfig{Users: "admin:admin123:admin"},
	})
	resp, err := other.Login("admin", "admin123")
	require.NoError(t, err)

	_, err = svc.ValidateToken(resp.Token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_HasRole(t *testing.T) {
	assert.True(t, HasRole(RoleAdmin, RoleAdmin))
	assert.True(t, HasRole(RoleAdmin, RoleTimer, RoleAdmin))
	assert.False(t, HasRole(RoleViewer, RoleAdmin))
	assert.False(t, HasRole(RoleTimer, RoleAdmin))
}
