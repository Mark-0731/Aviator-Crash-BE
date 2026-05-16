package config

import (
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Server
	Port           string
	Env            string
	AllowedOrigins []string

	// MongoDB
	MongoURI     string
	MongoDB      string
	MongoMaxPool uint64
	MongoMinPool uint64

	// JWT
	JWTSecret              string
	JWTAccessExpiryMinutes int
	JWTRefreshExpiryDays   int

	// Game
	WaitingDuration              time.Duration
	CrashDuration                time.Duration
	MultiplierTickMinMS          int
	MultiplierTickMaxMS          int
	MinBetCents                  int64
	MaxBetCents                  int64
	MaxWinCents                  int64
	DefaultBalanceCents          int64
	MaxConsecutiveInstantCrashes int

	// Provably Fair
	ServerSeedSecret string

	// NOWPayments (Crypto Deposits)
	NOWPaymentsAPIKey     string
	NOWPaymentsIPNSecret  string
	NOWPaymentsCallbackURL string
	NOWPaymentsPayCurrency string // e.g. "usdttrc20"

	// Security
	BcryptCost            int
	WSMaxMessageBytes     int64
	RateLimitREST         int
	RateLimitWS           int
	RateLimitAuth         int
	WSTicketExpirySeconds int
}

var AppConfig *Config

// Load reads environment variables and initializes the global config
func Load() error {
	_ = godotenv.Load() // Ignore error - .env is optional

	AppConfig = &Config{
		// Server
		Port:           getEnv("PORT", "8080"),
		Env:            getEnv("ENV", "development"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),

		// MongoDB
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:      getEnv("MONGO_DB", "aviator"),
		MongoMaxPool: getEnvAsUint64("MONGO_MAX_POOL", 50),
		MongoMinPool: getEnvAsUint64("MONGO_MIN_POOL", 10),

		// JWT
		JWTSecret:              getEnv("JWT_SECRET", "change-this-to-a-secure-32-char-minimum-secret-in-production"),
		JWTAccessExpiryMinutes: getEnvAsInt("JWT_ACCESS_EXPIRY_MINUTES", 15),
		JWTRefreshExpiryDays:   getEnvAsInt("JWT_REFRESH_EXPIRY_DAYS", 7),

		// Game
		WaitingDuration:              time.Duration(getEnvAsInt("WAITING_DURATION_SECONDS", 5)) * time.Second,
		CrashDuration:                time.Duration(getEnvAsInt("CRASH_DURATION_SECONDS", 2)) * time.Second,
		MultiplierTickMinMS:          getEnvAsInt("MULTIPLIER_TICK_MIN_MS", 90),
		MultiplierTickMaxMS:          getEnvAsInt("MULTIPLIER_TICK_MAX_MS", 110),
		MinBetCents:                  getEnvAsInt64("MIN_BET_CENTS", 100),
		MaxBetCents:                  getEnvAsInt64("MAX_BET_CENTS", 1000000),
		MaxWinCents:                  getEnvAsInt64("MAX_WIN_CENTS", 50000000),
		DefaultBalanceCents:          getEnvAsInt64("DEFAULT_BALANCE_CENTS", 100000),
		MaxConsecutiveInstantCrashes: getEnvAsInt("MAX_CONSECUTIVE_INSTANT_CRASHES", 3),

		// Provably Fair
		ServerSeedSecret: getEnv("SERVER_SEED_SECRET", "change-this-separate-32-char-minimum-hmac-secret-production"),

		// NOWPayments
		NOWPaymentsAPIKey:      getEnv("NOWPAYMENTS_API_KEY", ""),
		NOWPaymentsIPNSecret:   getEnv("NOWPAYMENTS_IPN_SECRET", ""),
		NOWPaymentsCallbackURL: getEnv("NOWPAYMENTS_CALLBACK_URL", "http://localhost:8080/api/wallet/webhook/nowpayments"),
		NOWPaymentsPayCurrency: getEnv("NOWPAYMENTS_PAY_CURRENCY", "usdttrc20"),

		// Security
		BcryptCost:            getEnvAsInt("BCRYPT_COST", 12),
		WSMaxMessageBytes:     getEnvAsInt64("WS_MAX_MESSAGE_BYTES", 512),
		RateLimitREST:         getEnvAsInt("RATE_LIMIT_REST", 60),
		RateLimitWS:           getEnvAsInt("RATE_LIMIT_WS", 10),
		RateLimitAuth:         getEnvAsInt("RATE_LIMIT_AUTH", 5),
		WSTicketExpirySeconds: getEnvAsInt("WS_TICKET_EXPIRY_SECONDS", 10),
	}

	validateProduction(AppConfig)

	log.Info().
		Str("env", AppConfig.Env).
		Str("port", AppConfig.Port).
		Msg("config_loaded")

	return nil
}
