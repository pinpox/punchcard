package main

import (
	"fmt"
	"os"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	OIDC        OIDCConfig
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	config := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "punchcard.db"),
		OIDC: OIDCConfig{
			IssuerURL:    getEnv("OIDC_ISSUER_URL", ""),
			ClientID:     getEnv("OIDC_CLIENT_ID", ""),
			ClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OIDC_REDIRECT_URL", "http://localhost:8080/callback"),
		},
	}

	return config
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// ValidateOIDCConfig validates OIDC configuration
func (c *Config) ValidateOIDCConfig() error {
	if c.OIDC.IssuerURL == "" {
		return fmt.Errorf("OIDC_ISSUER_URL is required")
	}
	if c.OIDC.ClientID == "" {
		return fmt.Errorf("OIDC_CLIENT_ID is required")
	}
	if c.OIDC.ClientSecret == "" {
		return fmt.Errorf("OIDC_CLIENT_SECRET is required")
	}
	if c.OIDC.RedirectURL == "" {
		return fmt.Errorf("OIDC_REDIRECT_URL is required")
	}
	return nil
}