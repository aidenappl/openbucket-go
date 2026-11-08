package env

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aidenappl/go-keyring"
)

var (
	Port   = getEnv("PORT", "8080")
	Region = getEnv("REGION", "openbucket")
	// KeyringPrefix names this instance's secrets in Keyring (e.g. OB_APLB_,
	// OB_TRAILBLAZE_): several OpenBucket instances run the same image with
	// different secrets.
	KeyringPrefix    = getEnv("KEYRING_SECRET_PREFIX", "OB_")
	Database         = getSecretOrPanic("DATABASE")
	PresignSecretKey = getSecretOrPanic("PRESIGN_SECRET_KEY")
	PresignBaseURL   = getEnv("PRESIGN_BASE_URL", "http://localhost:8080")
	AdminToken       = getSecretOrPanic("OB_ADMIN_TOKEN")
)

const keyringTimeout = 10 * time.Second

func getEnv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

// getSecretOrPanic reads a secret from the environment (local development,
// tests) or, when unset, from Keyring as KeyringPrefix+key using the
// container's KEYRING_* service token.
func getSecretOrPanic(key string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()
	if v, ok := keyring.Lookup(ctx, KeyringPrefix+key); ok {
		return v
	}
	panic(fmt.Sprintf("❌ missing required secret: '%v' (env) or '%v' (Keyring)\n", key, KeyringPrefix+key))
}
