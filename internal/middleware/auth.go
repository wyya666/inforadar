package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"inforadar/internal/common"
	"inforadar/internal/constants"
	middlewareinterface "inforadar/internal/middleware/interface"
)

// RequireAuth validates the bearer JWT and the server-side session before a protected handler runs.
func RequireAuth(authenticator middlewareinterface.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			common.WriteError(c, common.ErrNotLogin)
			c.Abort()
			return
		}
		profile, userID, err := authenticator.Authenticate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			common.WriteErrorFrom(c, err)
			c.Abort()
			return
		}
		c.Set(constants.UserContextKey, profile)
		c.Set(constants.UserIDContextKey, userID)
		c.Next()
	}
}
