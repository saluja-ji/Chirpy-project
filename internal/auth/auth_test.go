package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashing(t *testing.T) {
	password := "secret123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("passwords should match")
	}
}

func TestJWT(t *testing.T) {
	userID := uuid.New()
	secret := "super-secret"

	token, err := MakeJWT(
		userID,
		secret,
		time.Hour,
	)

	if err != nil {
		t.Fatal(err)
	}

	parsedID, err := ValidateJWT(
		token,
		secret,
	)

	if err != nil {
		t.Fatal(err)
	}

	if parsedID != userID {
		t.Fatalf(
			"expected %v got %v",
			userID,
			parsedID,
		)
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(
		userID,
		"correct-secret",
		time.Hour,
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = ValidateJWT(
		token,
		"wrong-secret",
	)

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestJWTExpired(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(
		userID,
		"secret",
		-time.Hour,
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = ValidateJWT(
		token,
		"secret",
	)

	if err == nil {
		t.Fatal("expected expired token error")
	}
}
