package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv          string
	Port            string
	FrontendBaseURL string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret      string
	JWTExpiryHours int

	BCryptCost int

	RateLimitPublic int
	RateLimitAdmin  int
	SessionTTLHours int

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	MigratePath string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		Port:            getEnv("PORT", "8080"),
		FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "qr_restaurant"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		JWTSecret:      getEnv("JWT_SECRET", "change-me"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),

		RateLimitPublic: getEnvInt("RATE_LIMIT_PUBLIC", 60),
		RateLimitAdmin:  getEnvInt("RATE_LIMIT_ADMIN", 120),
		SessionTTLHours: getEnvInt("SESSION_TTL_HOURS", 8),

		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "qr-restaurant"),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		MigratePath: getEnv("MIGRATE_PATH", "file://migrations"),
	}

	cost := getEnvInt("BCRYPT_COST", 12)
	if cost < 4 || cost > 31 {
		return nil, fmt.Errorf("BCRYPT_COST must be between 4 and 31, got %d", cost)
	}
	cfg.BCryptCost = cost

	if cfg.AppEnv == "production" {
		if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me" || cfg.JWTSecret == "your-256-bit-secret-here-change-in-production" || len(cfg.JWTSecret) < 32 {
			return nil, fmt.Errorf("JWT_SECRET must be set to a non-placeholder value of at least 32 characters in production")
		}
		if cfg.DBSSLMode == "disable" {
			return nil, fmt.Errorf("DB_SSL_MODE must not be disabled in production")
		}
	}
	if cfg.SessionTTLHours < 1 || cfg.SessionTTLHours > 72 {
		return nil, fmt.Errorf("SESSION_TTL_HOURS must be between 1 and 72, got %d", cfg.SessionTTLHours)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		i, err := strconv.Atoi(val)
		if err == nil {
			return i
		}
	}
	return fallback
}

func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) MigrationDatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   fmt.Sprintf("%s:%s", c.DBHost, c.DBPort),
		Path:   c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", c.DBSSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
