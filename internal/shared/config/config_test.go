package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsUnsafeProductionJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_SSL_MODE", "require")
	t.Setenv("JWT_SECRET", "change-me")

	_, err := Load()
	if err == nil {
		t.Fatal("expected production placeholder JWT secret to be rejected")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
}

func TestLoadRejectsDisabledProductionDBSSL(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("DB_SSL_MODE", "disable")

	_, err := Load()
	if err == nil {
		t.Fatal("expected disabled production DB SSL to be rejected")
	}
	if !strings.Contains(err.Error(), "DB_SSL_MODE") {
		t.Fatalf("expected DB_SSL_MODE error, got %v", err)
	}
}

func TestLoadUsesDatabaseURLInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("DATABASE_URL", "postgresql://app:p%40ss%20word@db.example.com:5432/qr_restaurant")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected DATABASE_URL to load, got %v", err)
	}

	if got := cfg.DatabaseDSN(); !strings.HasPrefix(got, "postgres://app:") {
		t.Fatalf("expected normalized postgres URL, got %q", got)
	}
	if !strings.Contains(cfg.DatabaseDSN(), "sslmode=require") {
		t.Fatalf("expected production DATABASE_URL to default to sslmode=require, got %q", cfg.DatabaseDSN())
	}
	if got := cfg.MigrationDatabaseURL(); got != cfg.DatabaseDSN() {
		t.Fatalf("expected migration URL to match database DSN, got %q and %q", got, cfg.DatabaseDSN())
	}
}

func TestLoadUsesDatabaseURLInDevelopmentWithoutForcingSSL(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgresql://app:p%40ss%20word@db.example.com:5432/qr_restaurant")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected DATABASE_URL to load, got %v", err)
	}

	if strings.Contains(cfg.DatabaseDSN(), "sslmode=require") {
		t.Fatalf("expected development DATABASE_URL to avoid forcing sslmode=require, got %q", cfg.DatabaseDSN())
	}
	if got := cfg.MigrationDatabaseURL(); got != cfg.DatabaseDSN() {
		t.Fatalf("expected migration URL to match database DSN, got %q and %q", got, cfg.DatabaseDSN())
	}
}

func TestMigrationDatabaseURLUsesPostgresURL(t *testing.T) {
	cfg := &Config{
		DBHost:     "db.example.com",
		DBPort:     "5432",
		DBUser:     "app",
		DBPassword: "p@ss word",
		DBName:     "qr_restaurant",
		DBSSLMode:  "require",
	}

	got := cfg.MigrationDatabaseURL()
	if !strings.HasPrefix(got, "postgres://app:") {
		t.Fatalf("expected postgres URL, got %q", got)
	}
	if !strings.Contains(got, "@db.example.com:5432/qr_restaurant") {
		t.Fatalf("expected host/db in URL, got %q", got)
	}
	if !strings.Contains(got, "sslmode=require") {
		t.Fatalf("expected sslmode query, got %q", got)
	}
}
