package auth

import (
	"testing"
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
