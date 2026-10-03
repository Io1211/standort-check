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
	// Production enables the Secure flag on cookies.
	Production bool
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AdminEmail:        os.Getenv("ADMIN_EMAIL"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
		BrevoAPIKey:       os.Getenv("BREVO_API_KEY"),
		EmailFrom:         os.Getenv("EMAIL_FROM"),
		Production:        os.Getenv("VERCEL_ENV") == "production" || os.Getenv("APP_ENV") == "production",
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is not set")
	}
	return cfg, nil
}
