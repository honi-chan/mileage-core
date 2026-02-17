package config

import (
	"os"
	"strconv"
	"time"
)

// Config はアプリケーション設定
type Config struct {
	Port      string
	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
}

// DBConfig はMySQL接続設定
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN はMySQL接続文字列を返す
func (c DBConfig) DSN() string {
	return c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + c.Port + ")/" + c.Name + "?parseTime=true&loc=UTC&charset=utf8mb4"
}

// RedisConfig はRedis接続設定
type RedisConfig struct {
	Addr string
}

// JWTConfig はJWT設定
type JWTConfig struct {
	Secret     string
	ExpiryTime time.Duration
}

// RateLimitConfig はレート制限設定
type RateLimitConfig struct {
	RPS   float64
	Burst int
}

// Load は環境変数から設定を読み込む
func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "mileage"),
			Password: getEnv("DB_PASSWORD", "mileage"),
			Name:     getEnv("DB_NAME", "mileage"),
		},
		Redis: RedisConfig{
			Addr: getEnv("REDIS_ADDR", "localhost:6379"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "dev-secret-key"),
			ExpiryTime: time.Duration(getEnvInt("JWT_EXPIRY_HOURS", 24)) * time.Hour,
		},
		RateLimit: RateLimitConfig{
			RPS:   float64(getEnvInt("RATE_LIMIT_RPS", 10)),
			Burst: getEnvInt("RATE_LIMIT_BURST", 20),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}
