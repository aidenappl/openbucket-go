package env

import (
	"fmt"
	"os"
)

var (
	Port   = getEnv("PORT", "8080")
	Region = getEnv("REGION", "openbucket")
	Database          = getEnvOrPanic("DATABASE")
	PresignSecretKey  = getEnvOrPanic("PRESIGN_SECRET_KEY")
	PresignBaseURL    = getEnv("PRESIGN_BASE_URL", "http://localhost:8080")
	AdminToken        = getEnvOrPanic("OB_ADMIN_TOKEN")
)

func getEnv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

func getEnvOrPanic(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("❌ missing required environment variable: '%v'\n", key))
	}
	return value
}
