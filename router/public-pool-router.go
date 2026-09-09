package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerPublicPoolRoutes(apiRouter *gin.RouterGroup) {
	userRoute := apiRouter.Group("/public-pool")
	userRoute.Use(middleware.UserAuth())
	{
		userRoute.GET("/sites", controller.ListPublicPoolSitesForSelf)
		userRoute.GET("/contributions/self", controller.ListPublicPoolContributionsForSelf)
		userRoute.POST("/contributions", controller.CreatePublicPoolContributionForSelf)
		userRoute.GET("/status", controller.GetPublicPoolStatus)
	}

	adminRoute := apiRouter.Group("/public-pool/admin")
	adminRoute.Use(middleware.ManagedAdminAuth())
	{
		adminRoute.GET("/reward-plans", middleware.RequirePermission(authz.PublicPoolWrite), controller.AdminListRewardSubscriptionPlans)
		adminRoute.GET("/sites", middleware.RequirePermission(authz.PublicPoolRead), controller.AdminListPublicPoolSites)
		adminRoute.POST("/sites", middleware.RequirePermission(authz.PublicPoolWrite), controller.AdminCreatePublicPoolSite)
		adminRoute.PUT("/sites/:id", middleware.RequirePermission(authz.PublicPoolWrite), controller.AdminUpdatePublicPoolSite)
		adminRoute.DELETE("/sites/:id", middleware.RequirePermission(authz.PublicPoolWrite), controller.AdminDeletePublicPoolSite)
		adminRoute.GET("/contributions", middleware.RequirePermission(authz.PublicPoolRead), controller.AdminListPublicPoolContributions)
		adminRoute.POST("/contributions/:id/review", middleware.RequirePermission(authz.PublicPoolOperate), controller.AdminReviewPublicPoolContribution)
		adminRoute.POST("/contributions/:id/retry-reward", middleware.RequirePermission(authz.PublicPoolOperate), controller.AdminRetryPublicPoolContributionReward)
		adminRoute.GET("/status", middleware.RequirePermission(authz.PublicPoolRead), controller.GetPublicPoolStatus)
	}
}
