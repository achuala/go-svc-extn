package crypto

import (
	"context"
	"fmt"
	"testing"

	"github.com/achuala/go-svc-extn/pkg/crypto/encdec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var kmsUri = "caas-kms://CILTuPkNElQKSAowdHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuQWVzR2NtS2V5EhIaECT2tUAgXpiynVn2MMgFlUgYARABGILTuPkNIAE"
var keySetData = "En0B3y4pgv8ANT2U89bAY9PkrR7Tz6Ww-hyiuLKsiUlTbWSZqBy5xO0HOt9BnlWviqnxYFt3jJbJKUsnkBp1m3C4WzQP702nSYOFCL7yjw556v9YSIuIMSLo-6vcFD0CTv1q-RnRMOycGOus-FjnWtK9mswGqHeacLBcXjf6tBpECLmZ5-8CEjwKMHR5cGUuZ29vZ2xlYXBpcy5jb20vZ29vZ2xlLmNyeXB0by50aW5rLkFlc0djbUtleRABGLmZ5-8CIAE"
var hmacKey = "QWVzR2NtS2V5EhIaECT2tUhyiuLKsiUlTbWSZq"
var cfg = &CryptoConfig{KmsUri: kmsUri, KmsUriPrefix: "caas-kms://", KeysetData: keySetData, HmacKey: hmacKey, KekAd: []byte("caas kek"), HashAlgorithm: "siphash24"}

func TestSipHash24(t *testing.T) {
	cu, err := NewCryptoUtil(cfg)
	require.NoError(t, err)
	plain := []byte("James Bond")
	hash, err := cu.CreateAlias(context.Background(), plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, hash)
	fmt.Println(string(hash))
}

func TestAesEncDec(t *testing.T) {
	cu, err := NewCryptoUtil(cfg)
	require.NoError(t, err)
	plain := []byte("James Bond")
	cipher, err := cu.Encrypt(context.Background(), plain, []byte("caas ad"))
	require.NoError(t, err)
	plainText, err := cu.Decrypt(context.Background(), cipher, []byte("caas ad"))
	require.NoError(t, err)
	assert.Equal(t, plain, plainText)
}

func TestLocalKEK_EncryptDecrypt(t *testing.T) {
	// Bootstrap: generate a master key and an encrypted keyset
	masterKey, err := encdec.GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := encdec.NewLocalAEAD(masterKey)
	require.NoError(t, err)

	kekAd := []byte("test kek ad")
	keysetData, err := encdec.GenerateEncryptedKeyset(masterAEAD, kekAd)
	require.NoError(t, err)

	localCfg := &CryptoConfig{
		MasterKey:     masterKey,
		KeysetData:    keysetData,
		HmacKey:       hmacKey,
		KekAd:         kekAd,
		HashAlgorithm: "siphash24",
	}
	cu, err := NewCryptoUtil(localCfg)
	require.NoError(t, err)

	plain := []byte("James Bond - Local KEK")
	ad := []byte("app ad")

	cipher, err := cu.Encrypt(context.Background(), plain, ad)
	require.NoError(t, err)

	plainText, err := cu.Decrypt(context.Background(), cipher, ad)
	require.NoError(t, err)
	assert.Equal(t, plain, plainText)
}

func TestLocalKEK_CompareHash(t *testing.T) {
	masterKey, err := encdec.GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := encdec.NewLocalAEAD(masterKey)
	require.NoError(t, err)

	kekAd := []byte("test kek ad")
	keysetData, err := encdec.GenerateEncryptedKeyset(masterAEAD, kekAd)
	require.NoError(t, err)

	localCfg := &CryptoConfig{
		MasterKey:     masterKey,
		KeysetData:    keysetData,
		HmacKey:       hmacKey,
		KekAd:         kekAd,
		HashAlgorithm: "siphash24",
	}
	cu, err := NewCryptoUtil(localCfg)
	require.NoError(t, err)

	ctx := context.Background()
	plain := []byte("James Bond")

	hash, err := cu.CreateAlias(ctx, plain)
	require.NoError(t, err)

	match, err := cu.CompareHash(ctx, plain, hash)
	require.NoError(t, err)
	assert.True(t, match)

	noMatch, err := cu.CompareHash(ctx, []byte("wrong"), hash)
	require.NoError(t, err)
	assert.False(t, noMatch)
}
