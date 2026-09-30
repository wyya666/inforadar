package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	"inforadar/internal/config"
	serviceinterface "inforadar/internal/service/interface"
)

type staleRefreshService struct{ serviceinterface.UserService }

func (staleRefreshService) Refresh(context.Context, string) (usermodel.SessionResult, error) {
	return usermodel.SessionResult{}, common.ErrNotLogin
}

func TestHandleErrorUsesCommonEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}
	for _, tc := range []struct {
		err      error
		wantCode int
	}{
		{fmt.Errorf("wrapped: %w", common.ErrInvalidInput), common.ErrInvalidInput.Code},
		{errors.New("database details must stay private"), common.ErrSystem.Code},
	} {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		h.handleError(c, tc.err)
		var payload common.BaseResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.Code != tc.wantCode || strings.Contains(payload.Message, "database details") {
			t.Fatalf("response = %s, want code %d: %v", response.Body.String(), tc.wantCode, err)
		}
	}
}

func TestRefreshCookieFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Auth: config.Auth{CookieSecure: true}}
	h := &Handler{config: cfg}
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	h.setRefreshCookie(c, strings.Repeat("a", 64))
	cookie := response.Header().Get("Set-Cookie")
	for _, flag := range []string{"HttpOnly", "Secure", "SameSite=Lax", "Path=/api/v1/auth"} {
		if !strings.Contains(cookie, flag) {
			t.Fatalf("cookie missing %s: %s", flag, cookie)
		}
	}
}

func TestStaleRefreshDoesNotClearNewerCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{service: staleRefreshService{}}
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: "inforadar_refresh", Value: strings.Repeat("a", 64)})
	h.Refresh(c)
	if response.Code != http.StatusUnauthorized || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("stale refresh changed browser cookie: status=%d cookie=%q", response.Code, response.Header().Get("Set-Cookie"))
	}
}
