package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Environment  string
	Version      string
	API          APIConfig
	Admin        AdminConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Solana       SolanaConfig
	BSC          BSCConfig
	TRON         TRONConfig
	ExchangeRate ExchangeRateConfig
	Security     SecurityConfig
	Email        EmailConfig
	Storage      StorageConfig
	AWS          AWSConfig
	TRM           TRMConfig
	TRMLabs       TRMConfig  // Alias for TRM
	JWT           JWTConfig
	OpsTeamEmails []string   // Email addresses for ops team alerts
}

// APIConfig contains API server configuration
type APIConfig struct {
	Port           int
	Host           string
	RateLimit      int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	AllowedOrigins []string
}

// AdminConfig contains admin server configuration
type AdminConfig struct {
	Port     int
	Host     string
	Email    string // Admin email for login
	Password string // Admin password for login
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
	WSURL              string // WebSocket URL for real-time transaction monitoring
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

// TRONConfig contains TRON blockchain configuration
type TRONConfig struct {
	RPCURL              string
	WalletPrivateKey    string
	WalletAddress       string
	ColdWalletAddress   string
	Network             string // mainnet, testnet (shasta, nile)
	USDTContract        string // TRC20 USDT contract
	USDCContract        string // TRC20 USDC contract
	MinConfirmations    int64  // Minimum confirmations (default: 19 for solid block)
	SweepThresholdUSD   int64  // Hot wallet sweep threshold in USD (default: 10000)
	SweepIntervalHours  int    // Auto-sweep interval in hours (default: 6)
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

// AWSConfig contains AWS configuration for S3 and other services
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
}

// TRMConfig contains TRM Labs AML screening configuration
type TRMConfig struct {
	APIKey  string
	BaseURL string
	Timeout int // seconds
}

// JWTConfig contains JWT authentication configuration
type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	trmConfig := TRMConfig{
		APIKey:  getEnv("TRM_LABS_API_KEY", ""),
		BaseURL: getEnv("TRM_LABS_BASE_URL", "https://api.trmlabs.com/public/v1"),
		Timeout: getEnvAsInt("TRM_LABS_TIMEOUT", 30),
	}

	config := &Config{
		Environment: getEnv("ENV", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		API: APIConfig{
			Port:           getEnvAsInt("API_PORT", 8080),
			Host:           getEnv("API_HOST", "0.0.0.0"),
			RateLimit:      getEnvAsInt("API_RATE_LIMIT", 100),
			ReadTimeout:    time.Duration(getEnvAsInt("API_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout:   time.Duration(getEnvAsInt("API_WRITE_TIMEOUT", 30)) * time.Second,
			AllowedOrigins: getEnvAsSlice("API_ALLOW_ORIGINS", []string{"http://localhost:3000"}),
		},
		Admin: AdminConfig{
			Port:     getEnvAsInt("ADMIN_PORT", 8081),
			Host:     getEnv("ADMIN_HOST", "0.0.0.0"),
			Email:    getEnv("ADMIN_EMAIL", "admin@payment-gateway.vn"),
			Password: getEnv("ADMIN_PASSWORD", ""),
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
			WSURL:             getEnv("SOLANA_WS_URL", ""), // Empty string triggers auto-derivation from RPC URL
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
		TRON: TRONConfig{
			RPCURL:             getEnv("TRON_RPC_URL", "grpc.shasta.trongrid.io:50051"),
			WalletPrivateKey:   getEnv("TRON_WALLET_PRIVATE_KEY", ""),
			WalletAddress:      getEnv("TRON_WALLET_ADDRESS", ""),
			ColdWalletAddress:  getEnv("TRON_COLD_WALLET_ADDRESS", ""),
			Network:            getEnv("TRON_NETWORK", "testnet"),
			USDTContract:       getEnv("TRON_USDT_CONTRACT", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"),
			USDCContract:       getEnv("TRON_USDC_CONTRACT", "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8"),
			MinConfirmations:   getEnvAsInt64("TRON_MIN_CONFIRMATIONS", 19),
			SweepThresholdUSD:  getEnvAsInt64("TRON_SWEEP_THRESHOLD_USD", 10000),
			SweepIntervalHours: getEnvAsInt("TRON_SWEEP_INTERVAL_HOURS", 6),
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
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			S3Bucket:        getEnv("AWS_S3_BUCKET", ""),
		},
		TRM:     trmConfig,
		TRMLabs: trmConfig, // Set alias for backwards compatibility
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		},
		OpsTeamEmails: getEnvAsSlice("OPS_TEAM_EMAILS", []string{}),
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

	// Validate TRON config
	if c.TRON.RPCURL == "" {
		errors = append(errors, "TRON_RPC_URL is required")
	}
	if c.TRON.WalletAddress == "" {
		errors = append(errors, "TRON_WALLET_ADDRESS is required")
	}
	if c.Environment == "production" && c.TRON.WalletPrivateKey == "" {
		errors = append(errors, "TRON_WALLET_PRIVATE_KEY is required in production")
	}
	if c.Environment == "production" && c.TRON.ColdWalletAddress == "" {
		errors = append(errors, "TRON_COLD_WALLET_ADDRESS is required in production")
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
