package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	FFmpeg   FFmpegConfig
	Storage  StorageConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port    string
	Mode    string
	Version string
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

type RabbitMQConfig struct {
	URL string
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

type FFmpegConfig struct {
	FFmpegPath  string
	FFprobePath string
}

type StorageConfig struct {
	UploadPath    string
	MaxUploadSize string
}

type LogConfig struct {
	Level  string
	Format string
}

var AppConfig *Config

func LoadConfig() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, continue without it
	}

	jwtExpiresIn, err := time.ParseDuration(getEnvOrDefault("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		return fmt.Errorf("invalid JWT_EXPIRES_IN duration: %w", err)
	}

	dbPort, err := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	if err != nil {
		return fmt.Errorf("invalid DB_PORT: %w", err)
	}

	redisPort, err := strconv.Atoi(getEnvOrDefault("REDIS_PORT", "6379"))
	if err != nil {
		return fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))
	if err != nil {
		return fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	AppConfig = &Config{
		Server: ServerConfig{
			Port:    getEnvOrDefault("SERVER_PORT", "8080"),
			Mode:    getEnvOrDefault("GIN_MODE", "debug"),
			Version: "1.0.0",
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "password"),
			DBName:   getEnvOrDefault("DB_NAME", "creative_studio"),
			SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:     redisPort,
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnvOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		JWT: JWTConfig{
			Secret:    getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpiresIn: jwtExpiresIn,
		},
		FFmpeg: FFmpegConfig{
			FFmpegPath:  getEnvOrDefault("FFMPEG_PATH", "/usr/local/bin/ffmpeg"),
			FFprobePath: getEnvOrDefault("FFPROBE_PATH", "/usr/local/bin/ffprobe"),
		},
		Storage: StorageConfig{
			UploadPath:    getEnvOrDefault("UPLOAD_PATH", "./uploads"),
			MaxUploadSize: getEnvOrDefault("MAX_UPLOAD_SIZE", "100MB"),
		},
		Log: LogConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", "info"),
			Format: getEnvOrDefault("LOG_FORMAT", "json"),
		},
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}