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
	"inforadar/internal/constants"
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
	for _, secure := range []bool{false, true} {
		for _, clear := range []bool{false, true} {
			t.Run(fmt.Sprintf("secure=%t/clear=%t", secure, clear), func(t *testing.T) {
				h := &Handler{config: config.Config{Auth: config.Auth{CookieSecure: secure}}}
				response := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(response)
				if clear {
					h.clearRefreshCookie(c)
				} else {
					h.setRefreshCookie(c, strings.Repeat("a", 64))
				}
				cookies := response.Result().Cookies()
				if len(cookies) != 1 {
					t.Fatalf("cookies = %v, want one refresh cookie", cookies)
				}
				cookie := cookies[0]
				if cookie.Name != constants.RefreshCookieName || cookie.Secure != secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != constants.RefreshCookiePath {
					t.Fatalf("unexpected refresh cookie flags: %s", cookie)
				}
				if clear && (cookie.Value != "" || cookie.MaxAge >= 0) {
					t.Fatalf("refresh cookie was not cleared: %s", cookie)
				}
				if !clear && (cookie.Value != strings.Repeat("a", 64) || cookie.MaxAge != int(constants.RefreshTTL.Seconds())) {
					t.Fatalf("unexpected refresh cookie value or lifetime: %s", cookie)
				}
			})
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
