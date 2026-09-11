package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerGitHubRegistrationRoutes(apiRouter *gin.RouterGroup) {
	route := apiRouter.Group("/github-registration")
	route.Use(middleware.ManagedAdminAuth())
	{
		route.GET("/whitelist", middleware.RequirePermission(authz.OAuthRead), controller.ListGitHubRegistrationWhitelist)
		route.POST("/resolve", middleware.RequirePermission(authz.OAuthWrite), controller.ResolveGitHubRegistrationIdentity)
		route.POST("/whitelist", middleware.RequirePermission(authz.OAuthWrite), controller.CreateGitHubRegistrationWhitelist)
		route.PATCH("/whitelist/:id", middleware.RequirePermission(authz.OAuthWrite), controller.UpdateGitHubRegistrationWhitelist)
		route.DELETE("/whitelist/:id", middleware.RequirePermission(authz.OAuthWrite), controller.DeleteGitHubRegistrationWhitelist)
	}
}
