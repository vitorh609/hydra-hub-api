package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

func HashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func CheckPassword(password, stored string) bool {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return false
	}

	hashed := HashPassword(password)
	if subtle.ConstantTimeCompare([]byte(hashed), []byte(stored)) == 1 {
		return true
	}

	// Compatibilidade com dados legados salvos em texto puro.
	return subtle.ConstantTimeCompare([]byte(password), []byte(stored)) == 1
}
