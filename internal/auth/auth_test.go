package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHash(t *testing.T) {
	passTest := "password"
	hash, err := HashPassword(passTest)

	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == passTest {
		t.Fatalf("hash must not equal plaintext password")
	}
}

func TestCheckHashwordPass(t *testing.T) {
	passwd := "password"
	hash, err := HashPassword(passwd)

	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	checkedHash, err := CheckPasswordHash(passwd, hash)
	if err != nil {
		t.Fatalf("hashed and passwd do not match: %v", err)
	}

	if checkedHash != true {
		t.Fatalf("hashed and passwd do not match: %v", err)
	}
}

func TestJWTRoundTrip(t *testing.T) {
	userID := uuid.New()
	token, err := MakeJWT(userID, "secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	got, err := ValidateJWT(token, "secret")
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}
	if got != userID {
		t.Fatalf("ValidateJWT() = %v, want %v", got, userID)
	}
}

func TestValidateJWTExpired(t *testing.T) {
	token, err := MakeJWT(uuid.New(), "secret", -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	got, err := ValidateJWT(token, "secret")
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("ValidateJWT() error = %v, want %v", err, jwt.ErrTokenExpired)
	}
	if got != uuid.Nil {
		t.Fatalf("ValidateJWT() = %v, want uuid.Nil", got)
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	token, err := MakeJWT(uuid.New(), "secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	got, err := ValidateJWT(token, "wrong-secret")
	if !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		t.Fatalf("ValidateJWT() error = %v, want %v", err, jwt.ErrTokenSignatureInvalid)
	}
	if got != uuid.Nil {
		t.Fatalf("ValidateJWT() = %v, want uuid.Nil", got)
	}
}

func TestValidateJWTMalformed(t *testing.T) {
	if _, err := ValidateJWT("not.a.token", "secret"); err == nil {
		t.Fatal("ValidateJWT() error = nil, want error for malformed token")
	}
}

func TestValidateJWTWrongIssuer(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "someone-else",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   uuid.New().String(),
	})
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	if _, err := ValidateJWT(tokenString, "secret"); !errors.Is(err, jwt.ErrTokenInvalidIssuer) {
		t.Fatalf("ValidateJWT() error = %v, want %v", err, jwt.ErrTokenInvalidIssuer)
	}
}
