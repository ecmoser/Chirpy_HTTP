package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error hashing password: %v", err)
	}
	err = CheckPasswordHash(hash, password)
	if err != nil {
		t.Fatalf("Error checking password hash: %v", err)
	}
}

func TestWrongHash(t *testing.T) {
	password := "password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error hashing password: %v", err)
	}
	wrongPassword := "wrong_password"
	err = CheckPasswordHash(hash, wrongPassword)
	if err == nil {
		t.Fatalf("Expected error for wrong password, got nil")
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := 1 * time.Hour
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}
	parsedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Error validating JWT: %v", err)
	}
	if parsedUserID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, parsedUserID)
	}
}

func TestWrongToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := 1 * time.Hour
	_, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}
	wrongToken := "wrong_token"
	parsedUserID, err := ValidateJWT(wrongToken, tokenSecret)
	if err == nil {
		t.Fatalf("Expected error for wrong token, got nil")
	}
	if parsedUserID != uuid.Nil {
		t.Fatalf("Expected user ID %v, got %v", uuid.Nil, parsedUserID)
	}
}

func TestExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret"
	expiresIn := -1 * time.Hour
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}
	parsedUserID, err := ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatalf("Expected error for expired token, got nil")
	}
	if parsedUserID != uuid.Nil {
		t.Fatalf("Expected user ID %v, got %v", uuid.Nil, parsedUserID)
	}
}
