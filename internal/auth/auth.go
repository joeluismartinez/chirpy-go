package auth

import (
	"fmt"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("Issue signing string: %w", err)
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("chirpy-access"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("Invalid token: %w", err)
	}
	return uuid.UUID(token.Signature), nil
}
