package hash

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/pkg/errors"
)

type Sha256 struct {
	c *Sha256Configuration
}

type Sha256Configuration struct {
	Key []byte
}

func NewHasherSha256(c *Sha256Configuration) *Sha256 {
	return &Sha256{c: c}
}

func (h *Sha256) Generate(ctx context.Context, data []byte) ([]byte, error) {
	hasher := hmac.New(sha256.New, h.c.Key)
	if _, err := hasher.Write(data); err != nil {
		return nil, errors.WithStack(err)
	}
	hash := hasher.Sum(nil)
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(hash)))
	base64.StdEncoding.Encode(dst, hash)
	return dst, nil
}

func (h *Sha256) Understands(hash []byte) bool {
	return IsSHAHash(hash)
}
