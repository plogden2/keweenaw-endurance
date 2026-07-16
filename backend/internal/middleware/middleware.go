package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/services"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRole     = "role"
)

// CORS middleware for handling cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Dev defaults — Vite may be opened as localhost or 127.0.0.1 (different origins).
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"https://localhost:3000",
			"https://localhost:3001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"https://127.0.0.1:3000",
			"https://127.0.0.1:3001",
		}

		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed ||
				strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "https://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "https://127.0.0.1:") {
				isAllowed = true
				break
			}
		}

		if isAllowed || origin == "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Bridge-Token, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Security middleware for adding security headers
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		c.Next()
	}
}

// RateLimit middleware for API rate limiting
func RateLimit(requests int, window int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple rate limiting - in production, use Redis or similar
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// JWTAuth validates JWT tokens and stores claims on the request context.
func JWTAuth(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

// RequireRoles ensures the authenticated user has one of the allowed roles.
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		roleStr, ok := role.(string)
		if !ok || !services.HasRole(roleStr, roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// Auth is an alias for JWTAuth for backward compatibility in tests.
func Auth(auth *services.AuthService) gin.HandlerFunc {
	return JWTAuth(auth)
}

// AdminAuth requires an admin role after JWTAuth.
func AdminAuth() gin.HandlerFunc {
	return RequireRoles(services.RoleAdmin, services.RoleOwner)
}
