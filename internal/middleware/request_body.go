package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"inforadar/internal/common"
)

// LimitRequestBody rejects known oversized bodies before handlers run and caps
// streamed bodies while JSON binding reads them.
func LimitRequestBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			common.WriteError(c, common.ErrRequestTooLarge)
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
