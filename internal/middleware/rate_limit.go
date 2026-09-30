package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"inforadar/internal/common"
	"inforadar/internal/constants"
	middlewareinterface "inforadar/internal/middleware/interface"
)

type Limiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	clock func() time.Time
}

func NewLimiter() middlewareinterface.Limiter {
	return &Limiter{hits: make(map[string][]time.Time), clock: time.Now}
}

func (l *Limiter) Allow(key string, count int, period time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	if len(l.hits) > constants.RateLimitCacheMaxEntries {
		for k, times := range l.hits {
			if len(times) == 0 || now.Sub(times[len(times)-1]) > constants.RateLimitCacheTTL {
				delete(l.hits, k)
			}
		}
	}
	active := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < period {
			active = append(active, t)
		}
	}
	if len(active) >= count {
		l.hits[key] = active
		return false
	}
	l.hits[key] = append(active, now)
	return true
}

func (l *Limiter) IPLimit(name string, count int, period time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(name+":ip:"+c.ClientIP(), count, period) {
			common.WriteError(c, common.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
