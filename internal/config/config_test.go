package config

import (
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/retrotrends")
	t.Setenv("EBAY_CLIENT_ID", "test-client-id")
	t.Setenv("EBAY_CLIENT_SECRET", "test-client-secret")
	t.Setenv("IGDB_CLIENT_ID", "test-igdb-id")
	t.Setenv("IGDB_CLIENT_SECRET", "test-igdb-secret")

	cfg := Load()

	if cfg.EbayDailyCallLimit != 5000 {
		t.Errorf("EbayDailyCallLimit default = %d, want 5000", cfg.EbayDailyCallLimit)
	}
	if cfg.IngestMaxPages != 5 {
		t.Errorf("IngestMaxPages default = %d, want 5", cfg.IngestMaxPages)
	}
}

func TestLoad_overrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/retrotrends")
	t.Setenv("EBAY_CLIENT_ID", "test-client-id")
	t.Setenv("EBAY_CLIENT_SECRET", "test-client-secret")
	t.Setenv("IGDB_CLIENT_ID", "test-igdb-id")
	t.Setenv("IGDB_CLIENT_SECRET", "test-igdb-secret")
	t.Setenv("EBAY_DAILY_CALL_LIMIT", "1000")
	t.Setenv("INGEST_MAX_PAGES", "3")

	cfg := Load()

	if cfg.EbayDailyCallLimit != 1000 {
		t.Errorf("EbayDailyCallLimit = %d, want 1000", cfg.EbayDailyCallLimit)
	}
	if cfg.IngestMaxPages != 3 {
		t.Errorf("IngestMaxPages = %d, want 3", cfg.IngestMaxPages)
	}
}

func TestLoad_requiredFields(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/retrotrends")
	t.Setenv("EBAY_CLIENT_ID", "my-client-id")
	t.Setenv("EBAY_CLIENT_SECRET", "my-secret")
	t.Setenv("IGDB_CLIENT_ID", "my-igdb-id")
	t.Setenv("IGDB_CLIENT_SECRET", "my-igdb-secret")

	cfg := Load()

	if cfg.DatabaseURL != "postgres://localhost/retrotrends" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://localhost/retrotrends")
	}
	if cfg.EbayClientID != "my-client-id" {
		t.Errorf("EbayClientID = %q, want %q", cfg.EbayClientID, "my-client-id")
	}
	if cfg.IGDBClientSecret != "my-igdb-secret" {
		t.Errorf("IGDBClientSecret = %q, want %q", cfg.IGDBClientSecret, "my-igdb-secret")
	}
}
