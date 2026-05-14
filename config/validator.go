package config

import (
	"strings"

	"github.com/rs/zerolog/log"
)

// validateProduction validates critical configuration in production environment
func validateProduction(cfg *Config) {
	if cfg.Env != "production" {
		return
	}

	validations := []struct {
		condition bool
		message   string
	}{
		{len(cfg.JWTSecret) < 32, "JWT_SECRET must be at least 32 characters in production"},
		{len(cfg.ServerSeedSecret) < 32, "SERVER_SEED_SECRET must be at least 32 characters in production"},
		{strings.Contains(cfg.JWTSecret, "change-this"), "JWT_SECRET must be changed from default value in production"},
		{strings.Contains(cfg.ServerSeedSecret, "change-this"), "SERVER_SEED_SECRET must be changed from default value in production"},
	}

	for _, v := range validations {
		if v.condition {
			log.Fatal().Msg(v.message)
		}
	}
}
