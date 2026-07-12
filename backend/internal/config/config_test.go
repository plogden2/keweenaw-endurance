package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"GO_ENV", "PORT", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_ACCESS_TOKEN_TTL", "JWT_REFRESH_TOKEN_TTL",
		"RATE_LIMIT_REQUESTS", "RATE_LIMIT_WINDOW", "CORS_ORIGINS",
		"AUTH_USERS", "ORGANIZER_PIN", "RFID_INJECT", "PROXMARK3_ENABLED", "HOSTED_API_URL", "DATA_DIR",
	}
	saved := make(map[string]string, len(keys))
	for _, key := range keys {
		saved[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if val, ok := saved[key]; ok && val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	})
}

func TestLoadConfig(t *testing.T) {
	// Test default configuration
	t.Run("DefaultConfig", func(t *testing.T) {
		clearConfigEnv(t)
		config, err := Load()
		
		require.NoError(t, err)
		assert.NotNil(t, config)
		
		// Test default values
		assert.Equal(t, "development", config.Environment)
		assert.Equal(t, "8080", config.Port)
		assert.Equal(t, "data", config.DataDir)
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, "5432", config.Database.Port)
		assert.Equal(t, "keweenaw_timing", config.Database.Name)
		assert.Equal(t, "timing_user", config.Database.User)
		assert.Equal(t, "timing_pass", config.Database.Password)
		assert.Equal(t, 25, config.Database.MaxOpenConns)
		assert.Equal(t, 5, config.Database.MaxIdleConns)
		assert.Equal(t, time.Hour, config.Database.ConnMaxLifetime)
		
		assert.Equal(t, "localhost", config.Redis.Host)
		assert.Equal(t, "6379", config.Redis.Port)
		assert.Equal(t, "", config.Redis.Password)
		assert.Equal(t, 0, config.Redis.DB)
		
		assert.Equal(t, time.Hour*24, config.JWT.AccessTokenTTL)
		assert.Equal(t, time.Hour*24*7, config.JWT.RefreshTokenTTL)
		
		assert.Equal(t, 100, config.Security.RateLimitRequests)
		assert.Equal(t, time.Minute, config.Security.RateLimitWindow)
		assert.Equal(t, []string{"http://localhost:3000"}, config.Security.CORSOrigins)

		assert.Equal(t, "1738", config.Auth.OrganizerPIN)
		assert.False(t, config.RFID.InjectEnabled)
		assert.False(t, config.RFID.Proxmark3Enabled)
		assert.Equal(t, "", config.RFID.HostedAPIURL)
	})

	t.Run("RFIDAndPINEnv", func(t *testing.T) {
		clearConfigEnv(t)
		t.Setenv("ORGANIZER_PIN", "9999")
		t.Setenv("RFID_INJECT", "true")
		t.Setenv("PROXMARK3_ENABLED", "true")
		t.Setenv("HOSTED_API_URL", "https://api.example.com")
		config, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "9999", config.Auth.OrganizerPIN)
		assert.True(t, config.RFID.InjectEnabled)
		assert.True(t, config.RFID.Proxmark3Enabled)
		assert.Equal(t, "https://api.example.com", config.RFID.HostedAPIURL)
	})
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "Environment variable not set",
			key:          "NON_EXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "Empty environment variable",
			key:          "EMPTY_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "Valid integer",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "25",
			expected:     25,
		},
		{
			name:         "Invalid integer",
			key:          "TEST_INT_INVALID",
			defaultValue: 10,
			envValue:     "invalid",
			expected:     10,
		},
		{
			name:         "Negative integer",
			key:          "TEST_INT_NEGATIVE",
			defaultValue: 10,
			envValue:     "-5",
			expected:     -5,
		},
		{
			name:         "Zero value",
			key:          "TEST_INT_ZERO",
			defaultValue: 10,
			envValue:     "0",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvAsInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "Valid duration",
			key:          "TEST_DURATION",
			defaultValue: time.Hour,
			envValue:     "2h30m",
			expected:     time.Hour*2 + time.Minute*30,
		},
		{
			name:         "Invalid duration",
			key:          "TEST_DURATION_INVALID",
			defaultValue: time.Hour,
			envValue:     "invalid",
			expected:     time.Hour,
		},
		{
			name:         "Seconds duration",
			key:          "TEST_DURATION_SECONDS",
			defaultValue: time.Hour,
			envValue:     "30s",
			expected:     time.Second * 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvAsDuration(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsSlice(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue []string
		envValue     string
		separator    string
		expected     []string
	}{
		{
			name:         "Single value",
			key:          "TEST_SLICE_SINGLE",
			defaultValue: []string{"default"},
			envValue:     "value1",
			separator:    ",",
			expected:     []string{"value1"},
		},
		{
			name:         "Multiple values",
			key:          "TEST_SLICE_MULTIPLE",
			defaultValue: []string{"default"},
			envValue:     "value1,value2,value3",
			separator:    ",",
			expected:     []string{"value1", "value2", "value3"},
		},
		{
			name:         "Values with spaces",
			key:          "TEST_SLICE_SPACES",
			defaultValue: []string{"default"},
			envValue:     " value1 , value2 , value3 ",
			separator:    ",",
			expected:     []string{"value1", "value2", "value3"},
		},
		{
			name:         "Empty values",
			key:          "TEST_SLICE_EMPTY",
			defaultValue: []string{"default"},
			envValue:     "value1,,value3",
			separator:    ",",
			expected:     []string{"value1", "value3"},
		},
		{
			name:         "Different separator",
			key:          "TEST_SLICE_SEMICOLON",
			defaultValue: []string{"default"},
			envValue:     "value1;value2;value3",
			separator:    ";",
			expected:     []string{"value1", "value2", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvAsSlice(tt.key, tt.defaultValue, tt.separator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringManipulationFunctions(t *testing.T) {
	t.Run("SplitString", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			separator string
			expected  []string
		}{
			{
				name:      "Empty string",
				input:     "",
				separator: ",",
				expected:  []string{},
			},
			{
				name:      "Single item",
				input:     "item1",
				separator: ",",
				expected:  []string{"item1"},
			},
			{
				name:      "Multiple items",
				input:     "item1,item2,item3",
				separator: ",",
				expected:  []string{"item1", "item2", "item3"},
			},
			{
				name:      "Empty separator",
				input:     "item1,item2",
				separator: "",
				expected:  []string{"item1,item2"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := splitString(tt.input, tt.separator)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Split", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			separator string
			expected  []string
		}{
			{
				name:      "Basic split",
				input:     "a,b,c",
				separator: ",",
				expected:  []string{"a", "b", "c"},
			},
			{
				name:      "Consecutive separators",
				input:     "a,,c",
				separator: ",",
				expected:  []string{"a", "", "c"},
			},
			{
				name:      "Separator at start",
				input:     ",a,b",
				separator: ",",
				expected:  []string{"", "a", "b"},
			},
			{
				name:      "Separator at end",
				input:     "a,b,",
				separator: ",",
				expected:  []string{"a", "b", ""},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := split(tt.input, tt.separator)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Trim", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "Leading spaces",
				input:    "   hello",
				expected: "hello",
			},
			{
				name:     "Trailing spaces",
				input:    "hello   ",
				expected: "hello",
			},
			{
				name:     "Leading and trailing spaces",
				input:    "   hello   ",
				expected: "hello",
			},
			{
				name:     "Leading and trailing tabs",
				input:    "\thello\t",
				expected: "hello",
			},
			{
				name:     "Leading and trailing newlines",
				input:    "\nhello\n",
				expected: "hello",
			},
			{
				name:     "Mixed whitespace",
				input:    " \t\nhello\n\t ",
				expected: "hello",
			},
			{
				name:     "Empty string",
				input:    "",
				expected: "",
			},
			{
				name:     "Only whitespace",
				input:    "   \t\n  ",
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := trim(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}