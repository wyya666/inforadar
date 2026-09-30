package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"inforadar/internal/constants"
	"golang.org/x/crypto/argon2"
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, constants.ArgonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, constants.ArgonIterations, constants.ArgonMemoryKiB, constants.ArgonParallelism, constants.ArgonKeyLength)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$%s$m=%d,t=%d,p=%d$%s$%s", constants.ArgonVersion, constants.ArgonMemoryKiB, constants.ArgonIterations, constants.ArgonParallelism, b64.EncodeToString(salt), b64.EncodeToString(hash)), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != constants.ArgonVersion {
		return false
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	// Parameters are fixed in v1. Reject malformed or excessive values before allocating memory.
	if memory != constants.ArgonMemoryKiB || iterations != constants.ArgonIterations || parallelism != constants.ArgonParallelism {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != constants.ArgonSaltLength {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) != constants.ArgonKeyLength {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
