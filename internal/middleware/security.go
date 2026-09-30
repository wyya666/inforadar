package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"inforadar/internal/common"
	"inforadar/internal/config"
	"inforadar/internal/constants"
)

func SecurityHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Cache-Control", "no-store")
	c.Next()
}

// CheckOrigin protects state-changing requests and requires Origin for refresh cookies.
func CheckOrigin(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete {
			origin := c.GetHeader("Origin")
			if (origin != "" && !trustedOrigin(cfg, origin)) || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
				common.WriteError(c, common.ErrOriginNotAllowed)
				c.Abort()
				return
			}
			if (c.FullPath() == constants.APIPrefix+constants.AuthRoute+constants.RefreshRoute || c.FullPath() == constants.APIPrefix+constants.AuthRoute+constants.LogoutRoute) && origin == "" {
				common.WriteError(c, common.ErrOriginRequired)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func trustedOrigin(cfg config.Config, origin string) bool {
	if origin == cfg.FrontendOrigin() {
		return true
	}
	if cfg.Server.Environment != constants.EnvironmentDevelopment {
		return false
	}
	configured, configuredErr := url.Parse(cfg.FrontendOrigin())
	requested, requestedErr := url.Parse(origin)
	return configuredErr == nil && requestedErr == nil &&
		configured.Scheme == requested.Scheme && configured.Port() == requested.Port() &&
		loopbackHost(configured.Hostname()) && loopbackHost(requested.Hostname()) &&
		requested.Path == "" && requested.RawQuery == "" && requested.Fragment == "" && requested.User == nil
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
