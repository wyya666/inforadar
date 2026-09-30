package router

import (
	"github.com/gin-gonic/gin"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	"inforadar/internal/middleware"
	middlewareinterface "inforadar/internal/middleware/interface"
)

func registerUserRoutes(api *gin.RouterGroup, user handlerinterface.UserHandler, authenticator middlewareinterface.Authenticator, limits middlewareinterface.Limiter) {
	auth := api.Group(constants.AuthRoute)
	auth.POST(constants.RegisterRoute, limits.IPLimit("register", constants.RegisterIPLimit, constants.RegisterIPPeriod), user.Register)
	auth.POST(constants.LoginRoute, limits.IPLimit("login", constants.LoginIPLimit, constants.LoginIPPeriod), user.Login)
	auth.POST(constants.RefreshRoute, user.Refresh)
	auth.POST(constants.LogoutRoute, user.Logout)
	auth.POST(constants.ForgotPasswordRoute, limits.IPLimit("forgot", constants.ForgotIPLimit, constants.ForgotIPPeriod), user.ForgotPassword)
	auth.POST(constants.ResetPasswordRoute, limits.IPLimit("reset", constants.ResetIPLimit, constants.ResetIPPeriod), user.ResetPassword)

	users := api.Group(constants.UsersRoute)
	users.Use(middleware.RequireAuth(authenticator))
	users.GET(constants.UserSelfRoute, user.Me)
	users.PATCH(constants.UserSelfRoute, user.UpdateProfile)
	users.POST(constants.UserPasswordRoute, user.ChangePassword)
}
