package constants

const (
	APIPrefix           = "/api/v1"
	AuthRoute           = "/auth"
	UsersRoute          = "/users"
	HealthRoute         = "/healthz"
	RegisterRoute       = "/register"
	LoginRoute          = "/login"
	RefreshRoute        = "/refresh"
	LogoutRoute         = "/logout"
	ForgotPasswordRoute = "/forgot-password"
	ResetPasswordRoute  = "/reset-password"
	UserSelfRoute       = "/me"
	UserPasswordRoute   = "/me/password"
	UserContextKey      = "user"
	UserIDContextKey    = "userID"
	TableUser           = "user"
	TableSessions       = "auth_sessions"
	TableActionTokens   = "action_tokens"
)
