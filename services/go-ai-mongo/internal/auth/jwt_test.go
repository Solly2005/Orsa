package auth

import (
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	token, err := Sign("secret", "user-123", "a@b.com", true, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := Verify("secret", token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "user-123" || claims.Email != "a@b.com" || !claims.EmailVerified {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	token, _ := Sign("secret", "user-123", "", false, time.Hour)
	if _, err := Verify("other-secret", token); err != ErrSignature {
		t.Fatalf("expected signature error, got %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	token, _ := Sign("secret", "user-123", "", false, -time.Minute)
	if _, err := Verify("secret", token); err != ErrExpired {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	token, _ := Sign("secret", "user-123", "", false, time.Hour)
	// Flip the last character of the payload/signature region.
	tampered := token[:len(token)-1] + "X"
	if _, err := Verify("secret", tampered); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	if _, err := Verify("secret", "not-a-jwt"); err != ErrMalformed {
		t.Fatalf("expected malformed error, got %v", err)
	}
}
