package handlerinterface

import "github.com/gin-gonic/gin"

type RadarHandler interface {
	GetSettings(*gin.Context)
	PutSettings(*gin.Context)
	PutKey(*gin.Context)
	DeleteKey(*gin.Context)
	ListTopics(*gin.Context)
	CreateTopic(*gin.Context)
	UpdateTopic(*gin.Context)
	DeleteTopic(*gin.Context)
	ScanTopic(*gin.Context)
	ListRuns(*gin.Context)
	ListFeed(*gin.Context)
}
