// Package auth verifies the HS256 session tokens minted by the C# Supabase auth
// service. The browser receives a token on login/register/OAuth and presents it
// as `Authorization: Bearer <token>` on every REST call; the gateway derives the
// authenticated user id from the verified token, never from a client header.
//
// The token format is a standard compact JWT (HS256). Both services share a
// secret via the ORSA_SESSION_SECRET environment variable so the gateway can
// verify locally without a round-trip to the auth service.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// DevSecret is the insecure fallback used by both services when
// ORSA_SESSION_SECRET is unset, so local development works out of the box. It
// must never be used in production. The identical literal lives in the C#
// service (SessionTokens / Program.cs); keep them in sync.
const DevSecret = "orsa-dev-insecure-session-secret-change-me"

// Claims is the minimal session-token payload.
type Claims struct {
	Subject       string `json:"sub"`            // user UUID
	Email         string `json:"email"`          // convenience only; not an identity source
	EmailVerified bool   `json:"email_verified"` // gates chat/upload until the address is confirmed
	Issued        int64  `json:"iat"`
	Expiry        int64  `json:"exp"`
}

var (
	ErrMalformed = errors.New("malformed token")
	ErrSignature = errors.New("invalid token signature")
	ErrExpired   = errors.New("token expired")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Sign produces an HS256 JWT for the given subject. Primarily used by tests; the
// C# service is the production issuer.
func Sign(secret, subject, email string, emailVerified bool, ttl time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(subject) == "" {
		return "", ErrMalformed
	}
	now := time.Now()
	h, _ := json.Marshal(header{Alg: "HS256", Typ: "JWT"})
	p, _ := json.Marshal(Claims{
		Subject:       subject,
		Email:         email,
		EmailVerified: emailVerified,
		Issued:        now.Unix(),
		Expiry:        now.Add(ttl).Unix(),
	})
	signingInput := encodeSegment(h) + "." + encodeSegment(p)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sign(secret, signingInput)), nil
}

// Verify checks an HS256 JWT against the secret and returns its claims. It
// rejects any other algorithm (no `alg: none`), bad signatures, and expired
// tokens.
func Verify(secret, token string) (Claims, error) {
	var claims Claims
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(token) == "" {
		return claims, ErrMalformed
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, ErrMalformed
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, ErrMalformed
	}
	var h header
	if err := json.Unmarshal(headerBytes, &h); err != nil || h.Alg != "HS256" {
		return claims, ErrMalformed
	}

	signingInput := parts[0] + "." + parts[1]
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, ErrMalformed
	}
	if !hmac.Equal(sign(secret, signingInput), gotSig) {
		return claims, ErrSignature
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrMalformed
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, ErrMalformed
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return claims, ErrMalformed
	}
	if claims.Expiry > 0 && time.Now().Unix() > claims.Expiry {
		return claims, ErrExpired
	}
	return claims, nil
}

func encodeSegment(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sign(secret, input string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return mac.Sum(nil)
}
