package handlerinterface

import "github.com/gin-gonic/gin"

// UserHandler exposes the account HTTP endpoints registered by the router.
type UserHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
	Refresh(*gin.Context)
	Logout(*gin.Context)
	ForgotPassword(*gin.Context)
	ResetPassword(*gin.Context)
	Me(*gin.Context)
	UpdateProfile(*gin.Context)
	ChangePassword(*gin.Context)
}
