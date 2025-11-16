package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Environment string
	API         APIConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Solana      SolanaConfig
	BSC         BSCConfig
	ExchangeRate ExchangeRateConfig
	Security    SecurityConfig
	Email       EmailConfig
	Storage     StorageConfig
}

// APIConfig contains API server configuration
type APIConfig struct {
	Port         int
	Host         string
	RateLimit    int
	ReadTimeout  int
	WriteTimeout int
	AllowOrigins []string
}

// DatabaseConfig contains PostgreSQL configuration
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	MaxOpenConns int
	MaxIdleConns int
	SSLMode      string
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// SolanaConfig contains Solana blockchain configuration
type SolanaConfig struct {
	RPCURL             string
	WalletPrivateKey   string
	WalletAddress      string
	Network            string // mainnet, testnet, devnet
	ConfirmationLevel  string // finalized, confirmed
	USDTMint           string
	USDCMint           string
}

// BSCConfig contains Binance Smart Chain configuration
type BSCConfig struct {
	RPCURL           string
	WalletPrivateKey string
	WalletAddress    string
	Network          string // mainnet, testnet
	ChainID          int64
	USDTContract     string
	BUSDContract     string
}

// ExchangeRateConfig contains exchange rate API configuration
type ExchangeRateConfig struct {
	PrimaryAPI   string
	SecondaryAPI string
	CacheTTL     int // seconds
	Timeout      int // seconds
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	JWTSecret           string
	JWTExpirationHours  int
	APIKeyLength        int
	WebhookSigningKey   string
	PasswordMinLength   int
	MaxLoginAttempts    int
	RateLimitPerMinute  int
}

// EmailConfig contains email service configuration
type EmailConfig struct {
	Provider      string // sendgrid, ses, smtp
	APIKey        string
	FromEmail     string
	FromName      string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
}

// StorageConfig contains file storage configuration
type StorageConfig struct {
	Provider        string // s3, minio, local
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // for MinIO
	BaseURL         string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	config := &Config{
		Environment: getEnv("ENV", "development"),
		API: APIConfig{
			Port:         getEnvAsInt("API_PORT", 8080),
			Host:         getEnv("API_HOST", "0.0.0.0"),
			RateLimit:    getEnvAsInt("API_RATE_LIMIT", 100),
			ReadTimeout:  getEnvAsInt("API_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("API_WRITE_TIMEOUT", 30),
			AllowOrigins: getEnvAsSlice("API_ALLOW_ORIGINS", []string{"http://localhost:3000"}),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvAsInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", ""),
			Database:     getEnv("DB_NAME", "payment_gateway"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Solana: SolanaConfig{
			RPCURL:            getEnv("SOLANA_RPC_URL", "https://api.devnet.solana.com"),
			WalletPrivateKey:  getEnv("SOLANA_WALLET_PRIVATE_KEY", ""),
			WalletAddress:     getEnv("SOLANA_WALLET_ADDRESS", ""),
			Network:           getEnv("SOLANA_NETWORK", "devnet"),
			ConfirmationLevel: getEnv("SOLANA_CONFIRMATION_LEVEL", "finalized"),
			USDTMint:          getEnv("SOLANA_USDT_MINT", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"),
			USDCMint:          getEnv("SOLANA_USDC_MINT", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"),
		},
		BSC: BSCConfig{
			RPCURL:           getEnv("BSC_RPC_URL", "https://data-seed-prebsc-1-s1.binance.org:8545"),
			WalletPrivateKey: getEnv("BSC_WALLET_PRIVATE_KEY", ""),
			WalletAddress:    getEnv("BSC_WALLET_ADDRESS", ""),
			Network:          getEnv("BSC_NETWORK", "testnet"),
			ChainID:          getEnvAsInt64("BSC_CHAIN_ID", 97),
			USDTContract:     getEnv("BSC_USDT_CONTRACT", ""),
			BUSDContract:     getEnv("BSC_BUSD_CONTRACT", ""),
		},
		ExchangeRate: ExchangeRateConfig{
			PrimaryAPI:   getEnv("EXCHANGE_RATE_PRIMARY_API", "https://api.coingecko.com/api/v3"),
			SecondaryAPI: getEnv("EXCHANGE_RATE_SECONDARY_API", "https://api.binance.com/api/v3"),
			CacheTTL:     getEnvAsInt("EXCHANGE_RATE_CACHE_TTL", 300),
			Timeout:      getEnvAsInt("EXCHANGE_RATE_TIMEOUT", 10),
		},
		Security: SecurityConfig{
			JWTSecret:          getEnv("JWT_SECRET", ""),
			JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
			APIKeyLength:       getEnvAsInt("API_KEY_LENGTH", 32),
			WebhookSigningKey:  getEnv("WEBHOOK_SIGNING_KEY", ""),
			PasswordMinLength:  getEnvAsInt("PASSWORD_MIN_LENGTH", 8),
			MaxLoginAttempts:   getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			RateLimitPerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 100),
		},
		Email: EmailConfig{
			Provider:     getEnv("EMAIL_PROVIDER", "smtp"),
			APIKey:       getEnv("EMAIL_API_KEY", ""),
			FromEmail:    getEnv("EMAIL_FROM_EMAIL", "noreply@payment-gateway.vn"),
			FromName:     getEnv("EMAIL_FROM_NAME", "Payment Gateway"),
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		},
		Storage: StorageConfig{
			Provider:        getEnv("STORAGE_PROVIDER", "local"),
			Bucket:          getEnv("STORAGE_BUCKET", "kyc-documents"),
			Region:          getEnv("STORAGE_REGION", "us-east-1"),
			AccessKeyID:     getEnv("STORAGE_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("STORAGE_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("STORAGE_ENDPOINT", ""),
			BaseURL:         getEnv("STORAGE_BASE_URL", "/uploads"),
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if all required configuration values are set
func (c *Config) Validate() error {
	var errors []string

	// Validate production environment requirements
	if c.Environment == "production" {
		if c.Security.JWTSecret == "" {
			errors = append(errors, "JWT_SECRET is required in production")
		}
		if c.Security.WebhookSigningKey == "" {
			errors = append(errors, "WEBHOOK_SIGNING_KEY is required in production")
		}
		if c.Solana.WalletPrivateKey == "" {
			errors = append(errors, "SOLANA_WALLET_PRIVATE_KEY is required")
		}
		if c.Database.Password == "" {
			errors = append(errors, "DB_PASSWORD is required in production")
		}
		if c.Database.SSLMode == "disable" {
			errors = append(errors, "DB_SSL_MODE must be enabled in production")
		}
	}

	// Validate database config
	if c.Database.Host == "" {
		errors = append(errors, "DB_HOST is required")
	}
	if c.Database.Database == "" {
		errors = append(errors, "DB_NAME is required")
	}

	// Validate Redis config
	if c.Redis.Host == "" {
		errors = append(errors, "REDIS_HOST is required")
	}

	// Validate Solana config
	if c.Solana.RPCURL == "" {
		errors = append(errors, "SOLANA_RPC_URL is required")
	}
	if c.Solana.WalletAddress == "" {
		errors = append(errors, "SOLANA_WALLET_ADDRESS is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsStaging returns true if running in staging mode
func (c *Config) IsStaging() bool {
	return c.Environment == "staging"
}

// GetDatabaseDSN returns PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// Helper functions to read environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
