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

const tokenIssuer = "chirpy-access"

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		Subject:   userID.String(),
	})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("issue signing string: %w", err)
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(tokenIssuer))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token: %w", err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("issue parsing the UUID: %w", err)
	}
	return userID, nil
}
