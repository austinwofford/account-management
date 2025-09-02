package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddress            string `env:"HTTP_ADDRESS" envDefault:":8080"`
	CORSEnabled            bool   `env:"CORS_ENABLED" envDefault:"true"`
	DebugEnabled           bool   `env:"DEBUG_ENABLED"`
	PostgresURL            string `env:"PSQL_URL,required"`
	AccessTokenTTLMinutes  int    `env:"REFRESH_TOKEN_TTL_MINUTES" envDefault:"15"`
	RefreshTokenTTLMinutes int    `env:"REFRESH_TOKEN_TTL_MINUTES" envDefault:"1440"`
	JWTSecretKey           string `env:"JWT_SECRET_KEY,required"`
}

func Load() (*Config, error) {
	var cfg Config

	// for local dev, default config with .env if enabled
	if _, ok := os.LookupEnv("USE_DOTENV"); ok {
		err := godotenv.Load()
		if err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}

		// reparse after godotenv sets the environment up
		if err := env.Parse(&cfg); err != nil {
			return nil, fmt.Errorf("error parsing config: %w", err)
		}
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}
