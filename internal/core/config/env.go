package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// ApplyEnvOverrides applies environment variable overrides to the configuration.
// Pattern: CIRCULAR_[SECTION]_[KEY] (e.g., CIRCULAR_OBSERVABILITY_PORT).
func ApplyEnvOverrides(cfg *Config) {
	// Paths
	setEnvString(&cfg.Paths.ProjectRoot, "CIRCULAR_PATHS_PROJECT_ROOT")
	setEnvString(&cfg.Paths.ConfigDir, "CIRCULAR_PATHS_CONFIG_DIR")
	setEnvString(&cfg.Paths.StateDir, "CIRCULAR_PATHS_STATE_DIR")
	setEnvString(&cfg.Paths.CacheDir, "CIRCULAR_PATHS_CACHE_DIR")
	setEnvString(&cfg.Paths.DatabaseDir, "CIRCULAR_PATHS_DATABASE_DIR")

	// Database
	setEnvBool(&cfg.DB.Enabled, "CIRCULAR_DB_ENABLED")
	setEnvString(&cfg.DB.Driver, "CIRCULAR_DB_DRIVER")
	setEnvString(&cfg.DB.Path, "CIRCULAR_DB_PATH")
	setEnvDuration(&cfg.DB.BusyTimeout, "CIRCULAR_DB_BUSY_TIMEOUT")

	// MCP
	setEnvBool(&cfg.MCP.Enabled, "CIRCULAR_MCP_ENABLED")
	setEnvString(&cfg.MCP.Mode, "CIRCULAR_MCP_MODE")
	setEnvString(&cfg.MCP.Transport, "CIRCULAR_MCP_TRANSPORT")
	setEnvString(&cfg.MCP.Address, "CIRCULAR_MCP_ADDRESS")
	setEnvString(&cfg.MCP.ServerName, "CIRCULAR_MCP_SERVER_NAME")
	setEnvInt(&cfg.MCP.MaxResponseItems, "CIRCULAR_MCP_MAX_RESPONSE_ITEMS")
	setEnvDuration(&cfg.MCP.RequestTimeout, "CIRCULAR_MCP_REQUEST_TIMEOUT")

	// Watch
	setEnvDuration(&cfg.Watch.Debounce, "CIRCULAR_WATCH_DEBOUNCE")

	// Secrets
	setEnvBool(&cfg.Secrets.Enabled, "CIRCULAR_SECRETS_ENABLED")
	setEnvFloat64(&cfg.Secrets.EntropyThreshold, "CIRCULAR_SECRETS_ENTROPY_THRESHOLD")
	setEnvInt(&cfg.Secrets.MinTokenLength, "CIRCULAR_SECRETS_MIN_TOKEN_LENGTH")

	// Caches
	setEnvInt(&cfg.Caches.Files, "CIRCULAR_CACHES_FILES")
	setEnvInt(&cfg.Caches.FileContents, "CIRCULAR_CACHES_FILE_CONTENTS")

	// Observability
	setEnvBool(&cfg.Observability.Enabled, "CIRCULAR_OBSERVABILITY_ENABLED")
	setEnvInt(&cfg.Observability.Port, "CIRCULAR_OBSERVABILITY_PORT")
	setEnvString(&cfg.Observability.OTLPEndpoint, "CIRCULAR_OBSERVABILITY_OTLP_ENDPOINT")
	setEnvBool(&cfg.Observability.EnableTracing, "CIRCULAR_OBSERVABILITY_ENABLE_TRACING")
	setEnvBool(&cfg.Observability.EnableMetrics, "CIRCULAR_OBSERVABILITY_ENABLE_METRICS")
}

func setEnvString(target *string, key string) {
	if val, ok := os.LookupEnv(key); ok {
		log.Printf("Applying env override: %s=%s", key, val)
		*target = val
	}
}

func setEnvInt(target *int, key string) {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			log.Printf("Applying env override: %s=%s", key, val)
			*target = i
		}
	}
}

func setEnvBool(target *bool, key string) {
	if val, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(strings.ToLower(val))
		if err == nil {
			log.Printf("Applying env override: %s=%s", key, val)
			*target = b
		}
	}
}

func setEnvFloat64(target *float64, key string) {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			log.Printf("Applying env override: %s=%s", key, val)
			*target = f
		}
	}
}

func setEnvDuration(target *time.Duration, key string) {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			log.Printf("Applying env override: %s=%s", key, val)
			*target = d
		}
	}
}
