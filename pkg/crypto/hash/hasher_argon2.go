package hash

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/inhies/go-bytesize"

	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash                 = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion         = errors.New("incompatible version of argon2")
	ErrMismatchedHashAndCredential = errors.New("credential does not match")
)

type Argon2 struct {
	c *Argon2Configuration
}

type Argon2Configuration struct {
	Parallelism uint8
	Memory      bytesize.ByteSize
	Iterations  uint32
	SaltLength  uint8
	KeyLength   uint32
}

func NewHasherArgon2(c *Argon2Configuration) *Argon2 {
	return &Argon2{c: c}
}

func toKB(mem bytesize.ByteSize) uint32 {
	return uint32(mem / bytesize.KB)
}

func (h *Argon2) Generate(ctx context.Context, password []byte) ([]byte, error) {
	salt := make([]byte, h.c.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Pass the plaintext password, salt and parameters to the argon2.IDKey
	// function. This will generate a hash of the password using the Argon2id
	// variant.
	hash := argon2.IDKey(password, salt, h.c.Iterations, toKB(h.c.Memory), h.c.Parallelism, h.c.KeyLength)

	var b bytes.Buffer
	if _, err := fmt.Fprintf(
		&b,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, toKB(h.c.Memory), h.c.Iterations, h.c.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	); err != nil {
		return nil, errors.WithStack(err)
	}

	return b.Bytes(), nil
}

func (h *Argon2) Understands(hash []byte) bool {
	return IsArgon2idHash(hash)
}
