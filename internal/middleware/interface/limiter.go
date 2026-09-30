package middlewareinterface

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Limiter supports route-level IP limits and account-level request limits.
type Limiter interface {
	Allow(key string, count int, period time.Duration) bool
	IPLimit(name string, count int, period time.Duration) gin.HandlerFunc
}
