package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	"inforadar/internal/middleware"
	middlewareinterface "inforadar/internal/middleware/interface"
)

// New groups API routes and attaches cross-cutting middleware.
func New(cfg config.Config, user handlerinterface.UserHandler, authenticator middlewareinterface.Authenticator, limits middlewareinterface.Limiter, radar ...handlerinterface.RadarHandler) http.Handler {
	return newEngine(cfg, user, authenticator, limits, radar...)
}

func newEngine(cfg config.Config, user handlerinterface.UserHandler, authenticator middlewareinterface.Authenticator, limits middlewareinterface.Limiter, radar ...handlerinterface.RadarHandler) *gin.Engine {
	if cfg.Server.Environment == constants.EnvironmentProduction {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		panic(err)
	}
	r.Use(middleware.SecurityHeaders, middleware.CheckOrigin(cfg))
	r.GET(constants.HealthRoute, func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": constants.MessageOK}) })
	api := r.Group(constants.APIPrefix)
	api.Use(middleware.LimitRequestBody(constants.MaxAPIRequestBodyBytes))
	registerUserRoutes(api, user, authenticator, limits)
	if len(radar) > 0 && radar[0] != nil {
		registerRadarRoutes(api, radar[0], authenticator)
	}
	return r
}
