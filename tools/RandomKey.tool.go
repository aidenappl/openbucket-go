package tools

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomKey(length int) string {

	const allowedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	result := make([]byte, length)

	for i := 0; i < length; i++ {

		randomByte, err := rand.Int(rand.Reader, big.NewInt(int64(len(allowedChars))))
		if err != nil {
			panic(err)
		}

		result[i] = allowedChars[randomByte.Int64()]
	}

	return string(result)
}
