package tools

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomKey(length int) string {
	// Define allowed characters in AWS keys
	const allowedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	// Create a slice to hold the result
	result := make([]byte, length)

	// Generate random bytes and map them to allowed characters
	for i := 0; i < length; i++ {
		// Get a random byte, then map it to a character in the allowed characters list
		randomByte, err := rand.Int(rand.Reader, big.NewInt(int64(len(allowedChars))))
		if err != nil {
			panic(err)
		}

		result[i] = allowedChars[randomByte.Int64()]
	}

	// Return the generated key
	return string(result)
}
