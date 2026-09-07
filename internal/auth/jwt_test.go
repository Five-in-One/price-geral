package auth

import (
	"testing"
	"time"

	"priceauth/internal/models"
)

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	token, err := GenerateToken("test-secret", "user-123", "alice", models.RoleCompany)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ParseToken("test-secret", token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("expected Username alice, got %s", claims.Username)
	}
	if claims.Role != models.RoleCompany {
		t.Errorf("expected Role company, got %s", claims.Role)
	}
}

func TestParseToken_WrongSecret_Fails(t *testing.T) {
	token, err := GenerateToken("test-secret", "user-123", "alice", models.RoleCompany)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	if _, err := ParseToken("wrong-secret", token); err == nil {
		t.Fatal("expected an error for a token signed with a different secret")
	}
}

func TestParseToken_Expired_Fails(t *testing.T) {
	original := AccessTokenTTL
	AccessTokenTTL = -1 * time.Hour // force an already-expired token
	defer func() { AccessTokenTTL = original }()

	token, err := GenerateToken("test-secret", "user-123", "alice", models.RoleCompany)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	if _, err := ParseToken("test-secret", token); err == nil {
		t.Fatal("expected an error for an expired token")
	}
}

func TestParseToken_Garbage_Fails(t *testing.T) {
	if _, err := ParseToken("test-secret", "not-a-jwt"); err == nil {
		t.Fatal("expected an error for a malformed token")
	}
}

func TestHashPassword_VerifiesAndRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !CheckPassword(hash, "correct-horse-battery-staple") {
		t.Error("expected the correct password to verify")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("expected an incorrect password to be rejected")
	}
}
