package clickup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

type CredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	KeyVersion() string
	Enabled() bool
}

type aesCipher struct {
	aead       cipher.AEAD
	keyVersion string
	enabled    bool
}

func NewCredentialCipher(base64Key string, keyVersion string) (CredentialCipher, error) {
	base64Key = strings.TrimSpace(base64Key)
	if base64Key == "" {
		return &aesCipher{enabled: false, keyVersion: keyVersion}, nil
	}

	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("decode clickup encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("clickup encryption key must be 32 bytes in base64")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create clickup cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create clickup gcm: %w", err)
	}

	return &aesCipher{
		aead:       aead,
		keyVersion: keyVersion,
		enabled:    true,
	}, nil
}

func (c *aesCipher) Encrypt(plaintext string) (string, error) {
	if !c.enabled {
		return "", ErrIntegrationDisabled
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *aesCipher) Decrypt(ciphertext string) (string, error) {
	if !c.enabled {
		return "", ErrIntegrationDisabled
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, payload := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt clickup credential: %w", err)
	}

	return string(plaintext), nil
}

func (c *aesCipher) KeyVersion() string {
	return c.keyVersion
}

func (c *aesCipher) Enabled() bool {
	return c.enabled
}
