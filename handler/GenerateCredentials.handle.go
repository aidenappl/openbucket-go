package handler

import (
	"github.com/aidenappl/openbucket-go/tools"
	"github.com/aidenappl/openbucket-go/types"
)

func GenerateCredentials() *types.Authorization {
	accessKey := tools.GenerateRandomKey(32)
	secretKey := tools.GenerateRandomKey(64)

	creds := types.Authorization{
		KeyID:     accessKey,
		SecretKey: secretKey,
	}

	return &creds
}
