package util

import (
	"cas-ums/internal/conf"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var kmsUri = "caas-kms://CILTuPkNElQKSAowdHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuQWVzR2NtS2V5EhIaECT2tUAgXpiynVn2MMgFlUgYARABGILTuPkNIAE"
var keySetData = "En0B3y4pgv8ANT2U89bAY9PkrR7Tz6Ww-hyiuLKsiUlTbWSZqBy5xO0HOt9BnlWviqnxYFt3jJbJKUsnkBp1m3C4WzQP702nSYOFCL7yjw556v9YSIuIMSLo-6vcFD0CTv1q-RnRMOycGOus-FjnWtK9mswGqHeacLBcXjf6tBpECLmZ5-8CEjwKMHR5cGUuZ29vZ2xlYXBpcy5jb20vZ29vZ2xlLmNyeXB0by50aW5rLkFlc0djbUtleRABGLmZ5-8CIAE"
var hmacKey = "QWVzR2NtS2V5EhIaECT2tUhyiuLKsiUlTbWSZq"
var cfg = &conf.Server{CaasKmsUri: kmsUri, KeysetData: keySetData, HmacKey: hmacKey}

func TestSipHash24(t *testing.T) {
	cu := NewCryptoUtil(cfg)
	plain := []byte("James Bond")
	hash, err := cu.CreateAlias(context.Background(), plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, hash)
	fmt.Println(string(hash))
}

func TestAesEncDec(t *testing.T) {
	cfg := &conf.Server{CaasKmsUri: kmsUri, KeysetData: keySetData, HmacKey: hmacKey}
	cu := NewCryptoUtil(cfg)
	plain := []byte("James Bond")
	cipher, err := cu.Encrypt(context.Background(), plain)
	require.NoError(t, err)
	plainText, err := cu.Decrypt(context.Background(), cipher)
	require.NoError(t, err)
	assert.Equal(t, plain, plainText)
}
