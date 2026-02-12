package encdec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAEAD_RoundTrip(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	localAead, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	plaintext := []byte("sensitive data")
	ad := []byte("context")

	ciphertext, err := localAead.Encrypt(plaintext, ad)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := localAead.Decrypt(ciphertext, ad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestLocalAEAD_WrongAD(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	localAead, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	ciphertext, err := localAead.Encrypt([]byte("data"), []byte("correct-ad"))
	require.NoError(t, err)

	_, err = localAead.Decrypt(ciphertext, []byte("wrong-ad"))
	assert.Error(t, err)
}

func TestLocalAEAD_InvalidKeyLength(t *testing.T) {
	// 16 bytes = 128 bits, too short
	_, err := NewLocalAEAD("AAAAAAAAAAAAAAAAAAAAAA==")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestLocalAEAD_InvalidBase64(t *testing.T) {
	_, err := NewLocalAEAD("not!valid!base64!!!")
	assert.Error(t, err)
}

func TestGenerateMasterKey(t *testing.T) {
	key1, err := GenerateMasterKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	key2, err := GenerateMasterKey()
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2, "keys should be random")
}

func TestGenerateEncryptedKeyset(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	ad := []byte("test keyset ad")

	keysetData, err := GenerateEncryptedKeyset(masterAEAD, ad)
	require.NoError(t, err)
	assert.NotEmpty(t, keysetData)
}

func TestMigrateToLocalKEK(t *testing.T) {
	// --- Simulate existing caas-kms deployment ---
	// Generate a caas-kms KEK URI (cleartext keyset in URI)
	oldKmsUri, err := NewKeyURI()
	require.NoError(t, err)

	oldCfg := &TinkConfiguration{
		KekUri:       oldKmsUri,
		KekUriPrefix: "caas-kms://",
		KekAd:        []byte("old ad"),
	}

	// Create old KEK AEAD to generate an encrypted DEK
	oldClient, err := NewCaasKmsClient(oldCfg.KekUriPrefix)
	require.NoError(t, err)
	oldKekAEAD, err := oldClient.GetAEAD(oldCfg.KekUri)
	require.NoError(t, err)

	oldKeysetData, err := GenerateEncryptedKeyset(oldKekAEAD, oldCfg.KekAd)
	require.NoError(t, err)
	oldCfg.KeySetData = oldKeysetData

	// Encrypt some data with the old handler
	oldHandler, err := NewTinkCryptoHandler(oldCfg)
	require.NoError(t, err)

	plain := []byte("data encrypted before migration")
	appAD := []byte("app context")
	ciphertext, err := oldHandler.Encrypt(nil, plain, appAD)
	require.NoError(t, err)

	// --- Migration ---
	newMasterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	newAd := []byte("new local ad")
	newKeysetData, err := MigrateToLocalKEK(oldCfg, newMasterKey, newAd)
	require.NoError(t, err)

	// --- Use new local KEK config to decrypt old data ---
	newCfg := &TinkConfiguration{
		MasterKey:  newMasterKey,
		KeySetData: newKeysetData,
		KekAd:      newAd,
	}
	newHandler, err := NewTinkCryptoHandler(newCfg)
	require.NoError(t, err)

	decrypted, err := newHandler.Decrypt(nil, ciphertext, appAD)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted, "old ciphertext must decrypt with migrated keyset")

	// New handler can also encrypt new data
	newCipher, err := newHandler.Encrypt(nil, []byte("new data"), appAD)
	require.NoError(t, err)

	newPlain, err := newHandler.Decrypt(nil, newCipher, appAD)
	require.NoError(t, err)
	assert.Equal(t, []byte("new data"), newPlain)
}

func TestMigrateToKeysetFile(t *testing.T) {
	// --- Simulate existing caas-kms deployment ---
	oldKmsUri, err := NewKeyURI()
	require.NoError(t, err)

	oldCfg := &TinkConfiguration{
		KekUri:       oldKmsUri,
		KekUriPrefix: "caas-kms://",
		KekAd:        []byte("old ad"),
	}

	oldClient, err := NewCaasKmsClient(oldCfg.KekUriPrefix)
	require.NoError(t, err)
	oldKekAEAD, err := oldClient.GetAEAD(oldCfg.KekUri)
	require.NoError(t, err)

	oldKeysetData, err := GenerateEncryptedKeyset(oldKekAEAD, oldCfg.KekAd)
	require.NoError(t, err)
	oldCfg.KeySetData = oldKeysetData

	// Encrypt data with old config
	oldHandler, err := NewTinkCryptoHandler(oldCfg)
	require.NoError(t, err)

	plain := []byte("data encrypted with old caas-kms config")
	appAD := []byte("app context")
	ciphertext, err := oldHandler.Encrypt(nil, plain, appAD)
	require.NoError(t, err)

	// --- One-step migration to JSON file ---
	newMasterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	newAd := []byte("new keyset ad")
	keysetPath := filepath.Join(t.TempDir(), "keyset.json")

	err = MigrateToKeysetFile(oldCfg, newMasterKey, newAd, keysetPath)
	require.NoError(t, err)

	// Verify the JSON file was created
	data, err := os.ReadFile(keysetPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "encryptedKeyset")
	t.Logf("Migrated keyset JSON:\n%s", string(data))

	// --- New config uses master key + JSON file ---
	newCfg := &TinkConfiguration{
		MasterKey:  newMasterKey,
		KeySetFile: keysetPath,
		KekAd:      newAd,
	}
	newHandler, err := NewTinkCryptoHandler(newCfg)
	require.NoError(t, err)

	// Old ciphertext decrypts with the migrated keyset
	decrypted, err := newHandler.Decrypt(nil, ciphertext, appAD)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted)

	// New encryptions work too
	newCipher, err := newHandler.Encrypt(nil, []byte("post-migration data"), appAD)
	require.NoError(t, err)
	newPlain, err := newHandler.Decrypt(nil, newCipher, appAD)
	require.NoError(t, err)
	assert.Equal(t, []byte("post-migration data"), newPlain)
}

func TestKeysetFile_RoundTrip(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	ad := []byte("file test ad")
	keysetPath := filepath.Join(t.TempDir(), "keyset.json")

	// Generate keyset file
	err = GenerateKeysetFile(masterAEAD, ad, keysetPath)
	require.NoError(t, err)

	// File should exist and contain JSON
	data, err := os.ReadFile(keysetPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "encryptedKeyset")
	t.Logf("Keyset JSON:\n%s", string(data))

	// Use it via TinkCryptoHandler with KeySetFile
	cfg := &TinkConfiguration{
		MasterKey:  masterKey,
		KeySetFile: keysetPath,
		KekAd:      ad,
	}
	handler, err := NewTinkCryptoHandler(cfg)
	require.NoError(t, err)

	plain := []byte("file-based keyset test")
	appAD := []byte("app ad")

	cipher, err := handler.Encrypt(nil, plain, appAD)
	require.NoError(t, err)

	decrypted, err := handler.Decrypt(nil, cipher, appAD)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted)
}

func TestRotateKeysetFile(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	ad := []byte("rotate ad")
	keysetPath := filepath.Join(t.TempDir(), "keyset.json")

	// Generate initial keyset
	err = GenerateKeysetFile(masterAEAD, ad, keysetPath)
	require.NoError(t, err)

	// Encrypt data with original key
	cfg := &TinkConfiguration{MasterKey: masterKey, KeySetFile: keysetPath, KekAd: ad}
	handler1, err := NewTinkCryptoHandler(cfg)
	require.NoError(t, err)

	plain := []byte("data before rotation")
	appAD := []byte("app ad")
	cipherBeforeRotation, err := handler1.Encrypt(nil, plain, appAD)
	require.NoError(t, err)

	// Rotate: adds a new key and sets it as primary
	err = RotateKeysetFile(masterAEAD, ad, keysetPath)
	require.NoError(t, err)

	// Log the rotated keyset to see both keys
	data, err := os.ReadFile(keysetPath)
	require.NoError(t, err)
	t.Logf("Rotated keyset JSON:\n%s", string(data))

	// Load handler with rotated keyset
	handler2, err := NewTinkCryptoHandler(cfg)
	require.NoError(t, err)

	// Old ciphertext still decrypts (old key is still in the keyset)
	decrypted, err := handler2.Decrypt(nil, cipherBeforeRotation, appAD)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted, "old ciphertext must still decrypt after rotation")

	// New encrypts use the new primary key
	newCipher, err := handler2.Encrypt(nil, []byte("new data"), appAD)
	require.NoError(t, err)

	newPlain, err := handler2.Decrypt(nil, newCipher, appAD)
	require.NoError(t, err)
	assert.Equal(t, []byte("new data"), newPlain)
}

func TestLocalKEK_TinkHandlerRoundTrip(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	require.NoError(t, err)

	masterAEAD, err := NewLocalAEAD(masterKey)
	require.NoError(t, err)

	ad := []byte("kek ad")
	keysetData, err := GenerateEncryptedKeyset(masterAEAD, ad)
	require.NoError(t, err)

	// Create a TinkCryptoHandler using local KEK mode
	cfg := &TinkConfiguration{
		MasterKey:  masterKey,
		KeySetData: keysetData,
		KekAd:      ad,
	}
	handler, err := NewTinkCryptoHandler(cfg)
	require.NoError(t, err)

	plain := []byte("Hello, envelope encryption!")
	appAD := []byte("app context")

	cipher, err := handler.Encrypt(nil, plain, appAD)
	require.NoError(t, err)

	decrypted, err := handler.Decrypt(nil, cipher, appAD)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted)
}
