package encdec

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

type TinkConfiguration struct {
	KekUri       string
	KekUriPrefix string
	KeySetData   string
	KekAd        []byte
}

type TinkCryptoHandler struct {
	ksh  *keyset.Handle
	aead tink.AEAD
}

func NewTinkCryptoHandler(c *TinkConfiguration) (*TinkCryptoHandler, error) {
	client, err := NewCaasKmsClient(c.KekUriPrefix)
	if err != nil {
		return nil, err
	}
	kekAEAD, err := client.GetAEAD(c.KekUri)
	if err != nil {
		return nil, err
	}

	keysetAssociatedData := c.KekAd

	encryptedKeyset, err := base64.RawURLEncoding.DecodeString(c.KeySetData)
	if err != nil {
		return nil, err
	}
	// To use the primitive, we first need to decrypt the keyset. We use the same
	// KEK AEAD and the same associated data that we used to encrypt it.
	reader := keyset.NewBinaryReader(bytes.NewReader(encryptedKeyset))
	handle, err := keyset.ReadWithAssociatedData(reader, kekAEAD, keysetAssociatedData)
	if err != nil {
		return nil, err
	}

	aead, err := aead.New(handle)

	if err != nil {
		return nil, err
	}

	return &TinkCryptoHandler{ksh: handle, aead: aead}, nil
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
