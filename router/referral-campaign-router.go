package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerReferralCampaignRoutes(apiRouter *gin.RouterGroup) {
	userRoute := apiRouter.Group("/referral-campaign")
	userRoute.Use(middleware.UserAuth())
	{
		userRoute.GET("/self", controller.GetReferralCampaignForSelf)
	}

	adminRoute := apiRouter.Group("/referral-campaign/admin")
	adminRoute.Use(middleware.ManagedAdminAuth())
	{
		adminRoute.GET("/reward-plans", middleware.RequirePermission(authz.ReferralCampaignWrite), controller.AdminListRewardSubscriptionPlans)
		adminRoute.GET("/campaigns", middleware.RequirePermission(authz.ReferralCampaignRead), controller.AdminListReferralCampaigns)
		adminRoute.POST("/campaigns", middleware.RequirePermission(authz.ReferralCampaignWrite), controller.AdminCreateReferralCampaign)
		adminRoute.PUT("/campaigns/:id", middleware.RequirePermission(authz.ReferralCampaignWrite), controller.AdminUpdateReferralCampaign)
		adminRoute.GET("/campaigns/:id/events", middleware.RequirePermission(authz.ReferralCampaignRead), controller.AdminListReferralCampaignEvents)
		adminRoute.POST("/events/:id/retry-reward", middleware.RequirePermission(authz.ReferralCampaignOperate), controller.AdminRetryReferralCampaignReward)
		adminRoute.GET("/events/:id/review", middleware.RequirePermission(authz.ReferralCampaignRead), controller.AdminGetReferralEventReview)
		adminRoute.POST("/events/:id/review", middleware.RequirePermission(authz.ReferralCampaignOperate), controller.AdminReviewReferralEvent)
	}
}
