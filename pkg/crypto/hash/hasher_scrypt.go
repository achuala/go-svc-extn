package hash

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/crypto/scrypt"
)

type Scrypt struct {
	c *ScryptConfiguration
}

type ScryptConfiguration struct {
	Cost            uint32
	Block           uint32
	Parallelization uint32
	SaltLength      uint32
	KeyLength       uint32
}

func NewHasherScrypt(c *ScryptConfiguration) *Scrypt {
	return &Scrypt{c: c}
}

func (h *Scrypt) Generate(ctx context.Context, password []byte) ([]byte, error) {
	salt := make([]byte, h.c.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	// Derive the key from the other password using the same parameters.
	hash, err := scrypt.Key(password, salt, int(h.c.Cost), int(h.c.Block), int(h.c.Parallelization), int(h.c.KeyLength))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// format: $scrypt$ln=<cost>,r=<block>,p=<parrrelization>$<salt>$<hash>
	var b bytes.Buffer
	if _, err := fmt.Fprintf(
		&b,
		"$scrypt$ln=%d,r=%d,p=%d$%s$%s",
		h.c.Cost, h.c.Block, h.c.Parallelization,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash),
	); err != nil {
		return nil, errors.WithStack(err)
	}

	return b.Bytes(), nil
}

func (h *Scrypt) Understands(hash []byte) bool {
	return IsScryptHash(hash)
}
