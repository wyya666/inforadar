package user

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	userdto "inforadar/internal/api/dto/user"
	usermodel "inforadar/internal/api/model/user"
	uservo "inforadar/internal/api/vo/user"
	"inforadar/internal/common"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	middlewareinterface "inforadar/internal/middleware/interface"
	serviceinterface "inforadar/internal/service/interface"
	cryptoutil "inforadar/internal/utils/crypto"
)

type Handler struct {
	service serviceinterface.UserService
	config  config.Config
	limits  middlewareinterface.Limiter
}

func New(service serviceinterface.UserService, cfg config.Config, limits middlewareinterface.Limiter) handlerinterface.UserHandler {
	return &Handler{service: service, config: cfg, limits: limits}
}

func profileVO(profile usermodel.Profile) uservo.Profile {
	return uservo.Profile{
		ID: profile.ID, Email: profile.Email, DisplayName: profile.DisplayName,
		CreatedAt: profile.CreatedAt,
	}
}

func loginVO(session usermodel.SessionResult) uservo.Login {
	return uservo.Login{
		AccessToken: session.AccessToken, ExpiresIn: session.ExpiresIn, User: profileVO(session.User),
	}
}

func (h *Handler) emailLimit(c *gin.Context, name, email string, count int, period time.Duration) bool {
	if !h.limits.Allow(name+":email:"+cryptoutil.TokenHash(email), count, period) {
		common.WriteError(c, common.ErrTooManyRequests)
		return false
	}
	return true
}

func (h *Handler) handleError(c *gin.Context, err error) {
	common.WriteErrorFrom(c, err)
}

func bind(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		common.WriteBindError(c, err)
		return false
	}
	return true
}

func (h *Handler) Register(c *gin.Context) {
	var req userdto.RegisterRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.Register(c.Request.Context(), req.Email, req.Password); err != nil {
		h.handleError(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusCreated, uservo.Message{Message: constants.MessageRegistered})
}

func (h *Handler) Login(c *gin.Context) {
	var req userdto.LoginRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.emailLimit(c, "login", email, constants.LoginEmailLimit, constants.LoginEmailPeriod) {
		return
	}
	result, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}
	h.setRefreshCookie(c, result.Refresh)
	common.WriteSuccess(c, http.StatusOK, loginVO(result))
}

func (h *Handler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(constants.RefreshCookieName)
	if err != nil {
		h.handleError(c, common.ErrNotLogin)
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), raw)
	if err != nil {
		// A stale request from another tab must not erase a newer rotated cookie.
		h.handleError(c, err)
		return
	}
	h.setRefreshCookie(c, result.Refresh)
	common.WriteSuccess(c, http.StatusOK, loginVO(result))
}

func (h *Handler) Logout(c *gin.Context) {
	raw, _ := c.Cookie(constants.RefreshCookieName)
	if err := h.service.Logout(c.Request.Context(), raw); err != nil {
		h.handleError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	common.WriteSuccess(c, http.StatusOK, nil)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req userdto.EmailRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.emailLimit(c, "forgot", email, constants.ForgotEmailLimit, constants.ForgotEmailPeriod) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		if errors.Is(err, common.ErrMailUnavailable) {
			log.Printf("password reset email failed: %v", err)
		} else {
			h.handleError(c, err)
			return
		}
	}
	common.WriteSuccess(c, http.StatusAccepted, uservo.Message{Message: constants.MessageResetAccepted})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req userdto.ResetPasswordRequest
	if !bind(c, &req) {
		return
	}
	if len(req.Token) != constants.ActionTokenLength {
		h.handleError(c, common.ErrInvalidToken)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.handleError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	common.WriteSuccess(c, http.StatusOK, nil)
}

func (h *Handler) Me(c *gin.Context) {
	common.WriteSuccess(c, http.StatusOK, profileVO(c.MustGet(constants.UserContextKey).(usermodel.Profile)))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req userdto.UpdateProfileRequest
	if !bind(c, &req) {
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		h.handleError(c, common.ErrInvalidInput)
		return
	}
	user, err := h.service.UpdateProfile(c.Request.Context(), c.MustGet(constants.UserIDContextKey).(uint64), req.DisplayName)
	if err != nil {
		h.handleError(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, profileVO(user))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req userdto.ChangePasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), c.MustGet(constants.UserIDContextKey).(uint64), req.CurrentPassword, req.NewPassword); err != nil {
		h.handleError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	common.WriteSuccess(c, http.StatusOK, nil)
}

func (h *Handler) setRefreshCookie(c *gin.Context, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(constants.RefreshCookieName, value, int(constants.RefreshTTL.Seconds()), constants.RefreshCookiePath, "", h.config.Auth.CookieSecure, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(constants.RefreshCookieName, "", -1, constants.RefreshCookiePath, "", h.config.Auth.CookieSecure, true)
}
