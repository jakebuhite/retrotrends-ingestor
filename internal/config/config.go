package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	EbayClientID       string
	EbayClientSecret   string
	IGDBClientID       string
	IGDBClientSecret   string
	EbayDailyCallLimit int
	IngestMaxPages     int
}

func Load() *Config {
	return &Config{
		DatabaseURL:        mustEnv("DATABASE_URL"),
		EbayClientID:       mustEnv("EBAY_CLIENT_ID"),
		EbayClientSecret:   mustEnv("EBAY_CLIENT_SECRET"),
		IGDBClientID:       mustEnv("IGDB_CLIENT_ID"),
		IGDBClientSecret:   mustEnv("IGDB_CLIENT_SECRET"),
		EbayDailyCallLimit: envInt("EBAY_DAILY_CALL_LIMIT", 5000),
		IngestMaxPages:     envInt("INGEST_MAX_PAGES", 5),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func envInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("env var %s must be an integer, got %q", key, v)
	}
	return n
}
