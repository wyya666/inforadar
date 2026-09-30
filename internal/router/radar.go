package router

import (
	"github.com/gin-gonic/gin"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	"inforadar/internal/middleware"
	middlewareinterface "inforadar/internal/middleware/interface"
)

func registerRadarRoutes(api *gin.RouterGroup, radar handlerinterface.RadarHandler, auth middlewareinterface.Authenticator) {
	group := api.Group(constants.RadarRoute)
	group.Use(middleware.RequireAuth(auth))
	group.GET("/settings", radar.GetSettings)
	group.PUT("/settings", radar.PutSettings)
	group.PUT("/keys/:provider", radar.PutKey)
	group.DELETE("/keys/:provider", radar.DeleteKey)
	group.GET("/topics", radar.ListTopics)
	group.POST("/topics", radar.CreateTopic)
	group.PATCH("/topics/:id", radar.UpdateTopic)
	group.DELETE("/topics/:id", radar.DeleteTopic)
	group.POST("/topics/:id/scan", radar.ScanTopic)
	group.GET("/topics/:id/runs", radar.ListRuns)
	group.GET("/feed", radar.ListFeed)
}
