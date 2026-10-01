package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	radarhandler "inforadar/internal/handler/radar"
	userhandler "inforadar/internal/handler/user"
	"inforadar/internal/middleware"
	serviceinterface "inforadar/internal/service/interface"
)

type emptyRadarService struct{ serviceinterface.RadarService }

type httpSessionService struct {
	serviceinterface.UserService
	refresh string
}

func (s *httpSessionService) Login(context.Context, string, string) (usermodel.SessionResult, error) {
	s.refresh = strings.Repeat("a", constants.ActionTokenLength)
	return usermodel.SessionResult{AccessToken: "access-token", Refresh: s.refresh}, nil
}

func (s *httpSessionService) Refresh(_ context.Context, raw string) (usermodel.SessionResult, error) {
	if s.refresh == "" || raw != s.refresh {
		return usermodel.SessionResult{}, common.ErrNotLogin
	}
	s.refresh = strings.Repeat("b", constants.ActionTokenLength)
	return usermodel.SessionResult{AccessToken: "renewed-access-token", Refresh: s.refresh}, nil
}

func (s *httpSessionService) Logout(_ context.Context, raw string) error {
	if raw != s.refresh {
		return common.ErrNotLogin
	}
	s.refresh = ""
	return nil
}

func TestProductionHTTPCookieSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewUnstartedServer(nil)
	cfg := config.Config{
		Server: config.Server{FrontendURL: "http://" + server.Listener.Addr().String(), Environment: constants.EnvironmentProduction},
		Auth:   config.Auth{CookieSecure: false},
	}
	service := &httpSessionService{}
	limits := middleware.NewLimiter()
	server.Config.Handler = newEngine(cfg, userhandler.New(service, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits)
	server.Start()
	t.Cleanup(server.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	client.Jar = jar
	cookieURL, err := url.Parse(server.URL + constants.RefreshCookiePath)
	if err != nil {
		t.Fatal(err)
	}
	post := func(path, body string, wantStatus int) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, server.URL+constants.APIPrefix+constants.AuthRoute+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", cfg.FrontendOrigin())
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != wantStatus {
			t.Fatalf("%s status = %d, want %d", path, response.StatusCode, wantStatus)
		}
	}

	post(constants.LoginRoute, `{"email":"user@example.test","password":"valid password"}`, http.StatusOK)
	cookies := jar.Cookies(cookieURL)
	if len(cookies) != 1 || cookies[0].Value != strings.Repeat("a", constants.ActionTokenLength) {
		t.Fatalf("HTTP login did not store the refresh cookie: %v", cookies)
	}
	post(constants.RefreshRoute, "", http.StatusOK)
	cookies = jar.Cookies(cookieURL)
	if len(cookies) != 1 || cookies[0].Value != strings.Repeat("b", constants.ActionTokenLength) {
		t.Fatalf("HTTP refresh did not rotate the cookie: %v", cookies)
	}
	post(constants.LogoutRoute, "", http.StatusOK)
	if cookies = jar.Cookies(cookieURL); len(cookies) != 0 || service.refresh != "" {
		t.Fatalf("HTTP logout left an active session: cookies=%v", cookies)
	}
	post(constants.RefreshRoute, "", http.StatusUnauthorized)
}

func TestRadarRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://127.0.0.1:5173", Environment: "development"}}
	limits := middleware.NewLimiter()
	routes := newEngine(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits, radarhandler.New(&emptyRadarService{}))
	for _, path := range []string{"/api/v1/radar/settings", "/api/v1/radar/topics", "/api/v1/radar/feed"} {
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
}

type testAuthenticator struct {
	profile usermodel.Profile
	err     error
}

func (a testAuthenticator) Authenticate(_ context.Context, _ string) (usermodel.Profile, uint64, error) {
	if a.err != nil {
		return usermodel.Profile{}, 0, a.err
	}
	return a.profile, 42, nil
}

func TestUserRoutesAndMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://127.0.0.1:5173", Environment: "development"}}
	limits := middleware.NewLimiter()
	routes := newEngine(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits)
	testCases := []struct {
		name, method, path, origin, authorization, body string
		want, wantCode                                  int
	}{
		{"untrusted origin", http.MethodPost, "/api/v1/auth/register", "https://attacker.example", "", "{}", http.StatusForbidden, common.ErrOriginNotAllowed.Code},
		{"dev loopback alias", http.MethodPost, "/api/v1/auth/register", "http://localhost:5173", "", "{}", http.StatusBadRequest, common.ErrParams.Code},
		{"wrong development port", http.MethodPost, "/api/v1/auth/register", "http://127.0.0.1:5174", "", "{}", http.StatusForbidden, common.ErrOriginNotAllowed.Code},
		{"missing refresh origin", http.MethodPost, "/api/v1/auth/refresh", "", "", "{}", http.StatusForbidden, common.ErrOriginRequired.Code},
		{"invalid request", http.MethodPost, "/api/v1/auth/register", cfg.FrontendOrigin(), "", "{}", http.StatusBadRequest, common.ErrParams.Code},
		{"invalid registration email", http.MethodPost, "/api/v1/auth/register", cfg.FrontendOrigin(), "", `{"email":"invalid","password":"valid password"}`, http.StatusBadRequest, common.ErrParams.Code},
		{"short registration password", http.MethodPost, "/api/v1/auth/register", cfg.FrontendOrigin(), "", `{"email":"user@example.test","password":"short"}`, http.StatusBadRequest, common.ErrParams.Code},
		{"missing bearer", http.MethodGet, "/api/v1/users/me", "", "", "", http.StatusUnauthorized, common.ErrNotLogin.Code},
		{"revoked session", http.MethodGet, "/api/v1/users/me", "", "Bearer expired", "", http.StatusUnauthorized, common.ErrNotLogin.Code},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.want, response.Body.String())
			}
			var payload common.BaseResponse
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.Code != tc.wantCode || payload.Message == "" {
				t.Fatalf("response = %s, want business code %d: %v", response.Body.String(), tc.wantCode, err)
			}
		})
	}
}

func TestAPIRequestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://127.0.0.1:5173", Environment: constants.EnvironmentDevelopment}}
	limits := middleware.NewLimiter()
	routes := newEngine(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits)
	body := `{"email":"` + strings.Repeat("a", constants.MaxAPIRequestBodyBytes) + `","password":"valid-password"}`
	for _, tc := range []struct {
		name    string
		chunked bool
	}{
		{name: "known length"},
		{name: "streamed body", chunked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
			if tc.chunked {
				req.ContentLength = -1
			}
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, req)
			var payload common.BaseResponse
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || response.Code != http.StatusRequestEntityTooLarge || payload.Code != common.ErrRequestTooLarge.Code {
				t.Fatalf("response = %d %s: %v", response.Code, response.Body.String(), err)
			}
		})
	}
}

func TestAuthenticatedProfileRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://127.0.0.1:5173", Environment: "development"}}
	limits := middleware.NewLimiter()
	profile := usermodel.Profile{ID: "42", Email: "user@example.test", DisplayName: "Radar", CreatedAt: time.Now().UTC()}
	routes := New(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{profile: profile}, limits)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, req)
	var payload struct {
		Code int `json:"code"`
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || response.Code != http.StatusOK || payload.Code != 0 || payload.Data.Email != profile.Email {
		t.Fatalf("authenticated profile response = %s, error = %v", response.Body.String(), err)
	}
}

func TestProductionRequiresConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://203.0.113.10:8081", Environment: constants.EnvironmentProduction}}
	limits := middleware.NewLimiter()
	routes := newEngine(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits)
	for _, tc := range []struct {
		origin string
		want   int
	}{
		{cfg.FrontendOrigin(), http.StatusBadRequest},
		{"http://203.0.113.10", http.StatusForbidden},
		{"http://203.0.113.11:8081", http.StatusForbidden},
		{"https://203.0.113.10:8081", http.StatusForbidden},
		{"http://localhost:8081", http.StatusForbidden},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{}"))
			req.Header.Set("Origin", tc.origin)
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("origin %s: status=%d, want=%d, body=%s", tc.origin, response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestAccountRoutesRemainRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Server: config.Server{FrontendURL: "http://127.0.0.1:5173", Environment: "development"}}
	limits := middleware.NewLimiter()
	routes := newEngine(cfg, userhandler.New(nil, cfg, limits), testAuthenticator{err: common.ErrNotLogin}, limits)
	want := map[string]bool{
		"GET /healthz":                      true,
		"POST /api/v1/auth/register":        true,
		"POST /api/v1/auth/login":           true,
		"POST /api/v1/auth/refresh":         true,
		"POST /api/v1/auth/logout":          true,
		"POST /api/v1/auth/forgot-password": true,
		"POST /api/v1/auth/reset-password":  true,
		"GET /api/v1/users/me":              true,
		"PATCH /api/v1/users/me":            true,
		"POST /api/v1/users/me/password":    true,
	}
	for _, route := range routes.Routes() {
		key := route.Method + " " + route.Path
		if !want[key] {
			t.Errorf("unexpected route: %s", key)
		}
		delete(want, key)
	}
	for key := range want {
		t.Errorf("missing route: %s", key)
	}
}
