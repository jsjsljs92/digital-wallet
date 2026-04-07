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
	Auth     AuthConfig
	Limits   LimitsConfig
	Fraud    FraudConfig
	RateLimit RateLimitConfig
	OTP      OTPConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port int
	Env  string
}

type DatabaseConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	Name           string
	MaxConnections int
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret       string
	MockAuthEnabled bool
}

type LimitsConfig struct {
	DailyLimit  float64
	WeeklyLimit float64
}

type FraudConfig struct {
	AmountThreshold    float64
	VelocityThreshold  int
	VelocityWindowMins int
}

type RateLimitConfig struct {
	Requests int
	Window   int // seconds
}

type OTPConfig struct {
	ValiditySeconds int
	MockOTPSecret   string
}

type LoggingConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8080),
			Env:  getEnvString("SERVER_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:           getEnvString("DB_HOST", "localhost"),
			Port:           getEnvInt("DB_PORT", 3306),
			User:           getEnvString("DB_USER", "wallet_user"),
			Password:       getEnvString("DB_PASSWORD", "wallet_pass"),
			Name:           getEnvString("DB_NAME", "digital_wallet"),
			MaxConnections: getEnvInt("DB_MAX_CONNECTIONS", 25),
		},
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret:       getEnvString("JWT_SECRET", "secret-key"),
			MockAuthEnabled: getEnvBool("MOCK_AUTH_ENABLED", true),
		},
		Limits: LimitsConfig{
			DailyLimit:  getEnvFloat("DAILY_LIMIT", 5000.00),
			WeeklyLimit: getEnvFloat("WEEKLY_LIMIT", 15000.00),
		},
		Fraud: FraudConfig{
			AmountThreshold:    getEnvFloat("FRAUD_AMOUNT_THRESHOLD", 10000.00),
			VelocityThreshold:  getEnvInt("FRAUD_VELOCITY_THRESHOLD", 5),
			VelocityWindowMins: getEnvInt("FRAUD_VELOCITY_WINDOW_MINUTES", 10),
		},
		RateLimit: RateLimitConfig{
			Requests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
			Window:   getEnvInt("RATE_LIMIT_WINDOW", 60),
		},
		OTP: OTPConfig{
			ValiditySeconds: getEnvInt("OTP_VALIDITY_SECONDS", 300),
			MockOTPSecret:   getEnvString("MOCK_OTP_SECRET", "123456"),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

func GetContextTimeout() time.Duration {
	return 30 * time.Second
}
