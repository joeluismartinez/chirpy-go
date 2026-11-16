package auth

import (
	"fmt"
	"time"
	"uuid"

	"github.com/alexedwards/argon2id"
)

func HashPassword(pwd string) (string, error) {
	hash, err := argon2id.CreateHash(pwd, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("Hashing password: %w", err)
	}
	return hash, nil
}

func CheckPasswordHash(pwd, hash string) (bool, error) {
	check, err := argon2id.ComparePasswordAndHash(pwd, hash)
	if err != nil {
		return false, fmt.Errorf("Hash out of sync: %w", err)
	}
	return check, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {

}
