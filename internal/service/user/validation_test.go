package user

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	if validatePassword("short") == nil || validatePassword(strings.Repeat("x", 129)) == nil || validatePassword("valid password") != nil {
		t.Fatal("password length policy failed")
	}
}
