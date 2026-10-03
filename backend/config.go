package backend

import (
	"errors"
	"os"
)

// Config is read from environment variables only. Secrets never reach the frontend.
type Config struct {
	DatabaseURL       string
	AdminEmail        string
	AdminPasswordHash string
	SessionSecret     string
	BrevoAPIKey       string
	EmailFrom         string
	// Geo enrichment (optional): without a key, leads are simply not geocoded.
	GeoapifyAPIKey string
	// Comma-separated federal states, e.g. "Sachsen,Hamburg".
	ServiceAreaStates string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AdminEmail:        os.Getenv("ADMIN_EMAIL"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
		BrevoAPIKey:       os.Getenv("BREVO_API_KEY"),
		EmailFrom:         os.Getenv("EMAIL_FROM"),
		GeoapifyAPIKey:    os.Getenv("GEOAPIFY_API_KEY"),
		ServiceAreaStates: os.Getenv("SERVICE_AREA_STATES"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is not set")
	}
	return cfg, nil
}
