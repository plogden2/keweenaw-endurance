package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS middleware for handling cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Allow all origins in development, configure for production
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"https://localhost:3000",
			"https://localhost:3001",
		}
		
		// Check if origin is allowed
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed || strings.HasPrefix(origin, "http://localhost:") {
				isAllowed = true
				break
			}
		}
		
		if isAllowed || origin == "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
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
		// Security headers
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
		// For now, this is a placeholder that allows all requests
		c.Next()
	}
}

// Auth middleware for JWT authentication
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder for JWT authentication
		// In production, validate JWT token here
		c.Next()
	}
}

// AdminAuth middleware for admin-only endpoints
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder for admin authentication
		// In production, check user role here
		c.Next()
	}
}