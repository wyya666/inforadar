package crypto

import (
	"strings"
	"testing"
	"time"

	"inforadar/internal/constants"
)

func TestPasswordHashAndVerify(t *testing.T) {
	first, err := HashPassword("a long passphrase 123")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("a long passphrase 123")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !VerifyPassword(first, "a long passphrase 123") || VerifyPassword(first, "wrong password") {
		t.Fatal("password hashing or verification failed")
	}
	if VerifyPassword("$argon2id$v=19$m=999999999,t=2,p=1$bad$bad", "password") {
		t.Fatal("unsafe encoded parameters accepted")
	}
}

func TestAccessTokenValidation(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	raw, err := SignAccess(secret, 42, "abc123", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	userID, sessionID, err := ParseAccess(secret, raw)
	if err != nil || userID != 42 || sessionID != "abc123" {
		t.Fatalf("valid access token rejected: %v", err)
	}
	if _, _, err := ParseAccess([]byte(strings.Repeat("x", 32)), raw); err == nil {
		t.Fatal("token signed with another key accepted")
	}
	expired, err := SignAccess(secret, 42, "abc123", time.Now().Add(-constants.AccessTTL-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseAccess(secret, expired); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestRandomTokens(t *testing.T) {
	a, err := NewRandomToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRandomToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 || a == b || TokenHash(a) == a || TokenHash(a) == TokenHash(b) {
		t.Fatal("random token generation failed")
	}
}
