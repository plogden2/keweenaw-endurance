package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/stretchr/testify/assert"
)

func testAuthService() *services.AuthService {
	return services.NewAuthService(&config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-secret",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24,
		},
		Auth: config.AuthConfig{
			Users: "admin:admin123:admin,timer:timer123:timer,viewer:viewer123:viewer",
		},
	})
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		expectedOrigin string
	}{
		{
			name:           "Allowed origin - localhost:3000",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:           "Allowed origin - localhost:3001",
			origin:         "http://localhost:3001",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedOrigin: "http://localhost:3001",
		},
		{
			name:           "Disallowed origin",
			origin:         "http://malicious-site.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedOrigin: "",
		},
		{
			name:           "OPTIONS request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedStatus: http.StatusNoContent,
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:           "No origin header",
			origin:         "",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORS())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedOrigin != "" {
				assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
				assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
			}

			if tt.method == "OPTIONS" {
				assert.Equal(t, http.StatusNoContent, w.Code)
			}
		})
	}
}

func TestSecurity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Security())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(100, 60))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuthMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := testAuthService()

	router := gin.New()
	router.Use(JWTAuth(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := testAuthService()
	login, err := auth.Login("viewer", "viewer123")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(JWTAuth(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": c.GetString(ContextRole)})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRolesForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := testAuthService()
	login, err := auth.Login("viewer", "viewer123")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(JWTAuth(auth), RequireRoles(services.RoleAdmin))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRolesAdminAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := testAuthService()
	login, err := auth.Login("admin", "admin123")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(JWTAuth(auth), AdminAuth())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddlewareChaining(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.Use(Security())
	router.Use(RateLimit(100, 60))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}
