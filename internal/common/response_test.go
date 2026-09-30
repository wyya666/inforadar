package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestResponseEnvelopeAndErrorMessage(t *testing.T) {
	success, err := json.Marshal(Success(map[string]string{"id": "42"}))
	if err != nil || !strings.Contains(string(success), `"code":0`) || !strings.Contains(string(success), `"data":{"id":"42"}`) {
		t.Fatalf("success response = %s: %v", success, err)
	}
	custom := ErrMailUnavailable.WithMessage("请稍后重发密码邮件")
	if custom.Message == ErrMailUnavailable.Message || !errors.Is(fmt.Errorf("send: %w", custom), ErrMailUnavailable) {
		t.Fatal("custom message changed the shared error or lost its identity")
	}
	failure, err := json.Marshal(Error(custom))
	if err != nil || strings.Contains(string(failure), `"data"`) || !strings.Contains(string(failure), `"code":50301`) {
		t.Fatalf("error response = %s: %v", failure, err)
	}
}
