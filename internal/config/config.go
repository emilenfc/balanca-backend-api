package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port        string
	Host        string
	Environment string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	DBURL    string
}

type JWTConfig struct {
	Secret                 string
	Expiration             time.Duration
	RefreshTokenExpiration time.Duration
}

type LoggingConfig struct {
	Level string
}

func Load() (*Config, error) {
	port := getEnv("SERVER_PORT", "8080")
	host := getEnv("SERVER_HOST", "0.0.0.0")

	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	refreshExp, _ := time.ParseDuration(getEnv("REFRESH_TOKEN_EXPIRATION", "168h"))

	return &Config{
		Server: ServerConfig{
			Port:        port,
			Host:        host,
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "balanca_user"),
			Password: getEnv("DB_PASSWORD", "balanca_password"),
			Name:     getEnv("DB_NAME", "balanca_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			DBURL:    getEnv("DBURL", ""),
		},
		JWT: JWTConfig{
			Secret:                 getEnv("JWT_SECRET", "your-secret-key"),
			Expiration:             jwtExp,
			RefreshTokenExpiration: refreshExp,
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "debug"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}