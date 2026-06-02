package utils

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

func VerifyPassword(password, encodedHash string) error {
	parts := strings.Split(encodedHash, ".")
	if len(parts) != 2 {
		return ErrorHandler(errors.New("invalid encoded hash format"), "Internal server error")
	}

	saltBase64 := parts[0]
	hashedPasswordBase64 := parts[1]

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return ErrorHandler(err, "Internal server error")
	}

	hashedPassword, err := base64.StdEncoding.DecodeString(hashedPasswordBase64)
	if err != nil {
		return ErrorHandler(err, "Internal server error")
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	if len(hash) != len(hashedPassword) {
		return ErrorHandler(errors.New("hash length mismatch"), "Incorrect password")
	}

	if subtle.ConstantTimeCompare(hash, hashedPassword) != 1 {
		return ErrorHandler(errors.New("Incorrect Password"), "Incorrect Password")
	}
	return nil
}
