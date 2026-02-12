package encdec

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

type TinkConfiguration struct {
	KekUri       string // caas-kms URI (used when MasterKey is empty)
	KekUriPrefix string // caas-kms URI prefix (used when MasterKey is empty)
	MasterKey    string // Base64-encoded AES-256 key for local KEK mode
	KeySetData   string // Base64url-encoded encrypted keyset (inline)
	KeySetFile   string // Path to JSON file containing encrypted keyset (takes precedence over KeySetData)
	KekAd        []byte // Associated data for keyset encryption
}

type TinkCryptoHandler struct {
	ksh  *keyset.Handle
	aead tink.AEAD
}

func NewTinkCryptoHandler(c *TinkConfiguration) (*TinkCryptoHandler, error) {
	var kekAEAD tink.AEAD
	var err error

	if c.MasterKey != "" {
		// Local KEK mode: build AEAD from a raw AES-256 key (e.g. from env var).
		kekAEAD, err = NewLocalAEAD(c.MasterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create local KEK AEAD: %w", err)
		}
	} else {
		// caas-kms mode: delegate to the custom KMS client.
		client, err := NewCaasKmsClient(c.KekUriPrefix)
		if err != nil {
			return nil, err
		}
		kekAEAD, err = client.GetAEAD(c.KekUri)
		if err != nil {
			return nil, err
		}
	}

	// Read the encrypted keyset from JSON file or inline base64url data.
	var handle *keyset.Handle
	if c.KeySetFile != "" {
		handle, err = ReadKeysetFile(kekAEAD, c.KekAd, c.KeySetFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read keyset file: %w", err)
		}
	} else {
		encryptedKeyset, err := base64.RawURLEncoding.DecodeString(c.KeySetData)
		if err != nil {
			return nil, err
		}
		reader := keyset.NewBinaryReader(bytes.NewReader(encryptedKeyset))
		handle, err = keyset.ReadWithAssociatedData(reader, kekAEAD, c.KekAd)
		if err != nil {
			return nil, err
		}
	}

	a, err := aead.New(handle)
	if err != nil {
		return nil, err
	}

	return &TinkCryptoHandler{ksh: handle, aead: a}, nil
}

func (h *TinkCryptoHandler) Encrypt(ctx context.Context, plain, associatedData []byte) ([]byte, error) {
	cipher, err := h.aead.Encrypt(plain, associatedData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to encrypt")
	}
	return cipher, nil
}

func (h *TinkCryptoHandler) Decrypt(ctx context.Context, cipher, associatedData []byte) ([]byte, error) {
	decrypted, err := h.aead.Decrypt(cipher, associatedData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decrypt")
	}
	return decrypted, nil
}
