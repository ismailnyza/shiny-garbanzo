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
