package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Encryption     EncryptionConfig
	Storage        StorageConfig
	Logging        LoggingConfig
	GeminiAPIKey string

type ServerConfig struct {
	Host         string
	Port         int
	Env          string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret     string
	RefreshSecret    string
	AccessExpiryMin  int
	RefreshExpiryDay int
}

type VKConfig struct {
	SecretKey string
	AppID     int
}

type EncryptionConfig struct {
	AESKey string
}

type StorageConfig struct {
	Type string
	Path string
}

type LoggingConfig struct {
	Level string
}

// Load loads configuration from environment variables or .env file
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Try to read from .env file, but don't fail if it doesn't exist
	_ = viper.ReadInConfig()

	config := &Config{
		Server: ServerConfig{
			Host:         viper.GetString("SERVER_HOST"),
			Port:         viper.GetInt("SERVER_PORT"),
			Env:          viper.GetString("ENV"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSL_MODE"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			AccessSecret:     viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret:    viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpiryMin:  viper.GetInt("JWT_ACCESS_EXPIRY_MIN"),
			RefreshExpiryDay: viper.GetInt("JWT_REFRESH_EXPIRY_DAY"),
		},
		VK: VKConfig{
			SecretKey: viper.GetString("VK_SECRET_KEY"),
			AppID:     viper.GetInt("VK_APP_ID"),
		},
		Encryption: EncryptionConfig{
			AESKey: viper.GetString("AES_ENCRYPTION_KEY"),
		},
		Storage: StorageConfig{
			Type: viper.GetString("STORAGE_TYPE"),
			Path: viper.GetString("STORAGE_PATH"),
		},
		Logging: LoggingConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		GeminiAPIKey: viper.GetString("GEMINI_API_KEY"),
	}

	// Validate critical configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates critical configuration values
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT access secret is required")
	}
	if len(c.JWT.AccessSecret) < 32 {
		return fmt.Errorf("JWT access secret must be at least 32 characters")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT refresh secret is required")
	}
	if len(c.Encryption.AESKey) != 32 {
		return fmt.Errorf("AES encryption key must be exactly 32 characters for AES-256")
	}
	if c.VK.SecretKey == "" {
		return fmt.Errorf("VK secret key is required")
	}
	if len(c.VK.SecretKey) < 16 {
		return fmt.Errorf("VK secret key must be at least 16 characters")
	}
	return nil
}

// GetDSN returns PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// GetRedisAddr returns Redis address
func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
