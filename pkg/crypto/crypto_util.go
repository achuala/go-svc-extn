package crypto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"

	"github.com/achuala/go-svc-extn/pkg/crypto/encdec"
	"github.com/achuala/go-svc-extn/pkg/crypto/hash"

	"github.com/pkg/errors"
)

type CryptoUtil struct {
	hashProvider   hash.Hasher
	cryptoProvider encdec.CryptoHandler
	ad             []byte
}

type CryptoConfig struct {
	KmsUri     string
	KeysetData string
	HmacKey    string
}

func NewCryptoUtil(cfg *CryptoConfig) *CryptoUtil {
	conf := hash.SipHashConfiguration{Key: cfg.HmacKey}
	hasher := hash.NewHasherSipHash24(&conf)
	tinkCfg := &encdec.TinkConfiguration{KekUri: cfg.KmsUri, KeySetData: cfg.KeysetData}
	cryptoProvider := encdec.NewTinkCryptoHandler(tinkCfg)
	return &CryptoUtil{hasher, cryptoProvider, []byte("8af14fe29dc1af27646dc61f")}

}

// CreateAlias creates an alias for the given plain text.
// It returns the hashed value of the plain text.
func (u *CryptoUtil) CreateAlias(ctx context.Context, plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return make([]byte, 0), nil
	}
	return u.hashProvider.Generate(ctx, plain)
}

// CompareHash compares the plain text with the stored hash.
// It returns true if the plain text is the same as the stored hash.
func (h *CryptoUtil) CompareHash(ctx context.Context, plainName, storedHash []byte) (bool, error) {
	newHash, err := h.CreateAlias(ctx, plainName)
	if err != nil {
		return false, err
	}

	// Compare the stored hash with the newly generated hash
	isEqual := bytes.Equal(storedHash, newHash)

	return isEqual, nil
}

// Encrypt encrypts the given plain text.
// It returns the encrypted value of the plain text.
func (u *CryptoUtil) Encrypt(ctx context.Context, plainText []byte) (string, error) {
	if cipherText, err := u.cryptoProvider.Encrypt(ctx, plainText, u.ad); err != nil {
		return "", err
	} else {
		return base64.RawStdEncoding.EncodeToString(cipherText), nil
	}
}

// Decrypt decrypts the given cipher text.
// It returns the decrypted value of the cipher text.
func (u *CryptoUtil) Decrypt(ctx context.Context, cipeherText string) ([]byte, error) {
	if cipher, err := base64.RawStdEncoding.DecodeString(cipeherText); err != nil {
		return nil, errors.Wrap(err, "unable to decode")
	} else {
		if plainText, err := u.cryptoProvider.Decrypt(ctx, cipher, u.ad); err != nil {
			return nil, err
		} else {
			return plainText, nil
		}
	}
}

// GenerateAesKey generates an AES key.
// It returns the AES key.
func (u *CryptoUtil) GenerateAesKey(ctx context.Context, key string) (string, error) {
	sessionKey, err := u.generateKey()
	return sessionKey, err
}

func (h *CryptoUtil) generateKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(key), nil
}

// EncryptWithKey encrypts the given plain text with the given key.
// It returns the encrypted value of the plain text.
func (u *CryptoUtil) EncryptWithKey(ctx context.Context, key, plainText string) (string, error) {
	keyBytes, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	cipherBytes := aesgcm.Seal(nil, nonce, []byte(plainText), nil)

	ciperText := base64.RawStdEncoding.EncodeToString(nonce) + "$$" + base64.RawStdEncoding.EncodeToString(cipherBytes)

	return ciperText, nil
}

// DecryptWithKey decrypts the given cipher text with the given key.
// It returns the decrypted value of the cipher text.
func (u *CryptoUtil) DecryptWithKey(ctx context.Context, key, cipeherText string) ([]byte, error) {
	keyBytes, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	// We need to split the data using $
	splitCipherText := strings.Split(cipeherText, "$$")
	if len(splitCipherText) != 2 {
		return nil, errors.New("invalid format for the cipher data")
	}

	nonce, err := base64.RawStdEncoding.DecodeString(splitCipherText[0])
	if err != nil {
		return nil, err
	}

	cipherBytes, err := base64.RawStdEncoding.DecodeString(splitCipherText[1])
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, nonce, cipherBytes, nil)
}
