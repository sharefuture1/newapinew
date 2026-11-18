package router

// Wending OAuth routes (fork-local customization; see WENDING_PATCHES.md).

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetWendingOAuthRouter(router *gin.Engine) {
	// Public OAuth endpoints used by client back-ends.
	publicRoute := router.Group("/api/oauth")
	publicRoute.Use(middleware.RouteTag("api"), middleware.BodyStorageCleanup(), middleware.GlobalAPIRateLimit(), middleware.DisableCache())
	{
		publicRoute.POST("/token", middleware.CriticalRateLimit(), middleware.AnonymousRequestBodyLimit(), controller.PostWendingOAuthToken)
		publicRoute.GET("/userinfo", middleware.CORS(), controller.GetWendingOAuthUserInfo)
		publicRoute.OPTIONS("/userinfo", middleware.CORS())
		publicRoute.GET("/clients/:client_id", controller.GetWendingOAuthPublicClient)
	}

	// Console endpoints (dashboard session auth via Authorization header).
	consoleRoute := router.Group("/api/wending/oauth")
	consoleRoute.Use(middleware.RouteTag("api"), middleware.AccessTokenAudit(), middleware.BodyStorageCleanup(), middleware.GlobalAPIRateLimit(), middleware.DisableCache())
	{
		userRoute := consoleRoute.Group("")
		userRoute.Use(middleware.UserAuth())
		{
			userRoute.GET("/authorize", controller.GetWendingOAuthAuthorize)
			userRoute.POST("/authorize", middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("wending-oauth"), controller.PostWendingOAuthAuthorize)
			userRoute.GET("/sidebar-apps", controller.ListWendingOAuthSidebarApps)
			userRoute.GET("/grants", controller.ListSelfWendingOAuthGrants)
			userRoute.DELETE("/grants/:id", middleware.CriticalRateLimit(), controller.RevokeSelfWendingOAuthGrant)
		}
		adminRoute := consoleRoute.Group("/admin")
		adminRoute.Use(middleware.AdminAuth())
		{
			adminRoute.GET("/grants", controller.ListAdminWendingOAuthGrants)
			adminRoute.DELETE("/grants/:id", controller.RevokeAdminWendingOAuthGrant)
		}
		rootRoute := consoleRoute.Group("/clients")
		rootRoute.Use(middleware.RootAuth())
		{
			rootRoute.GET("", controller.ListWendingOAuthClients)
			rootRoute.POST("", controller.CreateWendingOAuthClient)
			rootRoute.PUT("/:id", controller.UpdateWendingOAuthClient)
			rootRoute.POST("/:id/secret", controller.RotateWendingOAuthClientSecret)
			rootRoute.DELETE("/:id", controller.DeleteWendingOAuthClient)
		}
	}
}
