package encdec

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

// decryptOldKeyset decrypts an existing DEK keyset from a caas-kms TinkConfiguration.
func decryptOldKeyset(oldCfg *TinkConfiguration) (*keyset.Handle, error) {
	oldClient, err := NewCaasKmsClient(oldCfg.KekUriPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create old KMS client: %w", err)
	}
	oldKekAEAD, err := oldClient.GetAEAD(oldCfg.KekUri)
	if err != nil {
		return nil, fmt.Errorf("failed to get old KEK AEAD: %w", err)
	}

	oldKeyset, err := base64.RawURLEncoding.DecodeString(oldCfg.KeySetData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode old keyset data: %w", err)
	}
	reader := keyset.NewBinaryReader(bytes.NewReader(oldKeyset))
	handle, err := keyset.ReadWithAssociatedData(reader, oldKekAEAD, oldCfg.KekAd)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt old keyset: %w", err)
	}
	return handle, nil
}

// MigrateToLocalKEK re-wraps an existing DEK keyset from caas-kms mode to local KEK mode.
// It decrypts the DEK using the old caas-kms config, then re-encrypts the same DEK
// under the new master AEAD. Existing encrypted data remains decryptable because
// the DEK itself is unchanged — only its wrapper changes.
//
// Returns the new base64url-encoded encrypted keyset for use with local KEK inline config.
func MigrateToLocalKEK(oldCfg *TinkConfiguration, newMasterKey string, newAd []byte) (newKeysetData string, err error) {
	handle, err := decryptOldKeyset(oldCfg)
	if err != nil {
		return "", err
	}

	newMasterAEAD, err := NewLocalAEAD(newMasterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create new master AEAD: %w", err)
	}

	buf := new(bytes.Buffer)
	writer := keyset.NewBinaryWriter(buf)
	if err := handle.WriteWithAssociatedData(writer, newMasterAEAD, newAd); err != nil {
		return "", fmt.Errorf("failed to re-encrypt keyset: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// MigrateToKeysetFile re-wraps an existing DEK from caas-kms mode and writes it
// as an encrypted JSON keyset file under a new local master key.
//
// This is the one-step migration path:
//
//	Old: kmsUri (caas-kms://...) + keysetData (base64 string) → config/env
//	New: masterKey (env var)     + keyset.json (file)         → Secret + ConfigMap
//
// The DEK is unchanged — all existing ciphertext remains decryptable.
func MigrateToKeysetFile(oldCfg *TinkConfiguration, newMasterKey string, newAd []byte, keysetFilePath string) error {
	handle, err := decryptOldKeyset(oldCfg)
	if err != nil {
		return err
	}

	newMasterAEAD, err := NewLocalAEAD(newMasterKey)
	if err != nil {
		return fmt.Errorf("failed to create new master AEAD: %w", err)
	}

	return WriteKeysetFile(handle, newMasterAEAD, newAd, keysetFilePath)
}

// localAEAD implements tink.AEAD using a raw AES-256-GCM key.
// Used as the master KEK for envelope encryption when no external KMS is available.
type localAEAD struct {
	gcm cipher.AEAD
}

var _ tink.AEAD = (*localAEAD)(nil)

// NewLocalAEAD creates a tink.AEAD from a base64-encoded AES-256 key.
// The key must be exactly 32 bytes (256 bits) after decoding.
// Accepts both padded (StdEncoding) and unpadded (RawStdEncoding) base64.
func NewLocalAEAD(base64Key string) (tink.AEAD, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		keyBytes, err = base64.RawStdEncoding.DecodeString(base64Key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode master key: %w", err)
		}
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("master key must be 256 bits (32 bytes), got %d bytes", len(keyBytes))
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return &localAEAD{gcm: gcm}, nil
}

func (a *localAEAD) Encrypt(plaintext, associatedData []byte) ([]byte, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return a.gcm.Seal(nonce, nonce, plaintext, associatedData), nil
}

func (a *localAEAD) Decrypt(ciphertext, associatedData []byte) ([]byte, error) {
	nonceSize := a.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return a.gcm.Open(nil, nonce, ct, associatedData)
}

// GenerateMasterKey generates a random AES-256 key and returns it as a base64 string.
// Use this to bootstrap a new master KEK for local envelope encryption.
func GenerateMasterKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate master key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// GenerateEncryptedKeyset creates a new AES-256-GCM Tink keyset (DEK),
// encrypts it with the given master AEAD (KEK), and returns the
// base64url-encoded encrypted keyset suitable for storage in config.
func GenerateEncryptedKeyset(masterAEAD tink.AEAD, associatedData []byte) (string, error) {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return "", fmt.Errorf("failed to generate keyset: %w", err)
	}
	buf := new(bytes.Buffer)
	writer := keyset.NewBinaryWriter(buf)
	if err := handle.WriteWithAssociatedData(writer, masterAEAD, associatedData); err != nil {
		return "", fmt.Errorf("failed to encrypt keyset: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// WriteKeysetFile writes an encrypted keyset to a JSON file.
// The JSON format includes readable metadata (key IDs, types, status, primary)
// while the key material stays encrypted by the master AEAD.
func WriteKeysetFile(handle *keyset.Handle, masterAEAD tink.AEAD, associatedData []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create keyset file: %w", err)
	}
	defer f.Close()
	writer := keyset.NewJSONWriter(f)
	if err := handle.WriteWithAssociatedData(writer, masterAEAD, associatedData); err != nil {
		return fmt.Errorf("failed to write encrypted keyset: %w", err)
	}
	return nil
}

// ReadKeysetFile reads an encrypted keyset from a JSON file and returns the handle.
func ReadKeysetFile(masterAEAD tink.AEAD, associatedData []byte, path string) (*keyset.Handle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyset file: %w", err)
	}
	defer f.Close()
	reader := keyset.NewJSONReader(f)
	handle, err := keyset.ReadWithAssociatedData(reader, masterAEAD, associatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to read keyset from file: %w", err)
	}
	return handle, nil
}

// GenerateKeysetFile creates a new AES-256-GCM keyset, encrypts it with
// the master AEAD, and writes it to the given path as JSON.
func GenerateKeysetFile(masterAEAD tink.AEAD, associatedData []byte, path string) error {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return fmt.Errorf("failed to generate keyset: %w", err)
	}
	return WriteKeysetFile(handle, masterAEAD, associatedData, path)
}

// RotateKeysetFile adds a new AES-256-GCM key to an existing keyset file,
// sets it as the new primary (used for future encrypts), and writes back.
// Old keys remain in the keyset so existing ciphertext can still be decrypted.
func RotateKeysetFile(masterAEAD tink.AEAD, associatedData []byte, path string) error {
	handle, err := ReadKeysetFile(masterAEAD, associatedData, path)
	if err != nil {
		return fmt.Errorf("failed to read keyset for rotation: %w", err)
	}

	manager := keyset.NewManagerFromHandle(handle)
	keyID, err := manager.Add(aead.AES256GCMKeyTemplate())
	if err != nil {
		return fmt.Errorf("failed to add new key: %w", err)
	}
	if err := manager.SetPrimary(keyID); err != nil {
		return fmt.Errorf("failed to set new primary key: %w", err)
	}

	rotatedHandle, err := manager.Handle()
	if err != nil {
		return fmt.Errorf("failed to get rotated handle: %w", err)
	}
	return WriteKeysetFile(rotatedHandle, masterAEAD, associatedData, path)
}
