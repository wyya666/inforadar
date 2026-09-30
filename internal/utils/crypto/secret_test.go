package crypto

import (
	"strings"
	"testing"
)

func TestSecretBoxIsBoundToUserAndProvider(t *testing.T) {
	box, err := NewSecretBox(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	first, nonce, err := box.Encrypt(5, "zhipu", "test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := box.Encrypt(5, "zhipu", "test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) == "test-api-key" || string(first) == string(second) {
		t.Fatal("credential was exposed or nonce was reused")
	}
	decrypted, err := box.Decrypt(5, "zhipu", first, nonce)
	if err != nil || decrypted != "test-api-key" {
		t.Fatalf("decrypt = %q, %v", decrypted, err)
	}
	if _, err := box.Decrypt(6, "zhipu", first, nonce); err == nil {
		t.Fatal("credential decrypted for another user")
	}
	if _, err := box.Decrypt(5, "bailian", first, nonce); err == nil {
		t.Fatal("credential decrypted for another provider")
	}
}
