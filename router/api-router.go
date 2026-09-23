package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"

	// Import oauth package to register providers via init()
	_ "github.com/QuantumNous/new-api/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.AccessTokenAudit())
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	anonymousRequestBodyLimit := middleware.AnonymousRequestBodyLimit()
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", anonymousRequestBodyLimit, controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.HeaderNavModuleAuth("pricing"), controller.GetPricing)
		perfMetricsRoute := apiRouter.Group("/perf-metrics")
		perfMetricsRoute.Use(middleware.HeaderNavModulePublicOrUserAuth("pricing"))
		{
			perfMetricsRoute.GET("/summary", controller.GetPerfMetricsSummary)
			perfMetricsRoute.GET("", controller.GetPerfMetrics)
		}
		apiRouter.GET("/rankings", middleware.HeaderNavModuleAuth("rankings"), controller.GetRankings)
		apiRouter.GET("/usage-rankings", middleware.UserAuth(), controller.GetUsageRankings)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.POST("/oauth/state", middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.TryUserAuth(), anonymousRequestBodyLimit, controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind/start", middleware.UserAuth(), middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("account-security"), middleware.EmailVerificationRateLimit(), middleware.DisableCache(), controller.EmailBindStart)
		apiRouter.POST("/oauth/email/bind/resend", middleware.UserAuth(), middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("account-security"), middleware.EmailVerificationRateLimit(), middleware.DisableCache(), controller.EmailBindResend)
		apiRouter.POST("/oauth/email/bind", middleware.UserAuth(), middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("account-security"), middleware.DisableCache(), controller.EmailBind)
		// WeChat uses its existing authorization-code service.
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.TelegramLegacyAuth)
		apiRouter.POST("/oauth/telegram/bind/start", middleware.UserAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), controller.TelegramLegacyAuth)
		apiRouter.GET("/oauth/telegram/bind/:flow_token", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.TelegramLegacyAuth)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO, Telegram) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.TryUserAuth(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)

		apiRouter.POST("/stripe/webhook", anonymousRequestBodyLimit, controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", anonymousRequestBodyLimit, controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", anonymousRequestBodyLimit, controller.WaffoWebhook)
		// :env separates test vs prod URLs so the operator can register each
		// in Pancake's matching webhook slot; handler enforces env match.
		apiRouter.POST("/waffo-pancake/webhook/:env", anonymousRequestBodyLimit, controller.WaffoPancakeWebhook)

		// Universal secure verification routes
		apiRouter.GET("/verify/methods", middleware.UserAuth(), middleware.DisableCache(), controller.GetVerificationMethods)
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/auth/refresh", middleware.SessionCookieOriginGuard(), middleware.CriticalRateLimit(), middleware.DisableCache(), controller.RefreshAuth)
			userRoute.POST("/auth/logout", middleware.SessionCookieOriginGuard(), middleware.CriticalRateLimit(), middleware.DisableCache(), controller.AuthLogout)
			userRoute.POST("/register", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Register)
			userRoute.GET("/login/encryption-key", middleware.DisableCache(), controller.GetPasswordEncryptionKey)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.Verify2FALogin)
			userRoute.POST("/login/verify", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.VerifyLogin)
			userRoute.POST("/login/passkey/begin", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.LoginPasskeyBegin)
			userRoute.POST("/login/passkey/finish", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.LoginPasskeyFinish)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), middleware.DisableCache(), anonymousRequestBodyLimit, controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.POST("/epay/notify", anonymousRequestBodyLimit, controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.DisableCache(), middleware.UserAuth())
			{
				selfRoute.GET("/sessions", middleware.DisableCache(), controller.GetLoginSessions)
				selfRoute.DELETE("/sessions/:sid", middleware.DisableCache(), controller.DeleteLoginSession)
				selfRoute.POST("/sessions/revoke-others", middleware.DisableCache(), controller.RevokeOtherLoginSessions)
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.UpdateSelf)
				selfRoute.DELETE("/self", middleware.DisableCache(), controller.DeleteSelf)
				selfRoute.GET("/token", middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("access-token"), middleware.DisableCache(), controller.GenerateAccessToken)
				selfRoute.GET("/token/status", middleware.DisableCache(), controller.GetAccessTokenStatus)
				selfRoute.POST("/token", middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("access-token"), middleware.DisableCache(), controller.GenerateAccessToken)
				selfRoute.DELETE("/token", middleware.CriticalRateLimit(), middleware.UserCriticalRateLimit("access-token"), middleware.DisableCache(), controller.RevokeAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", middleware.DisableCache(), controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.GET("/manual-wechat/info", controller.GetManualWechatPaymentInfo)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/manual-wechat/pay", middleware.CriticalRateLimit(), controller.RequestManualWechatTopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/amount", controller.RequestWaffoAmount)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				selfRoute.POST("/waffo-pancake/amount", controller.RequestWaffoPancakeAmount)
				selfRoute.POST("/waffo-pancake/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPancakePay)
				selfRoute.POST("/aff_transfer", middleware.UserCriticalRateLimit("aff-transfer"), controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.Setup2FA)
				selfRoute.POST("/2fa/enable", middleware.UserCriticalRateLimit("security-verification"), middleware.DisableCache(), controller.Enable2FA)
				selfRoute.POST("/2fa/disable", middleware.DisableCache(), controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", middleware.DisableCache(), controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.ManagedAdminAuth())
			{
				adminRoute.GET("/", middleware.RequirePermission(authz.UserRead), controller.GetAllUsers)
				adminRoute.GET("/topup", middleware.RequirePermission(authz.UserRead), controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", middleware.RequirePermission(authz.UserQuota), controller.AdminCompleteTopUp)
				adminRoute.POST("/topup/manual-wechat/complete", middleware.RequirePermission(authz.UserQuota), controller.AdminCompleteManualWechatTopUp)
				adminRoute.POST("/topup/manual-wechat/expire", middleware.RequirePermission(authz.UserQuota), controller.AdminExpireManualWechatTopUp)
				adminRoute.GET("/search", middleware.RequirePermission(authz.UserRead), controller.SearchUsers)
				adminRoute.GET("/:id/oauth/bindings", middleware.RequirePermission(authz.UserSecurity), controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", middleware.RequirePermission(authz.UserSecurity), controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", middleware.RequirePermission(authz.UserSecurity), controller.AdminClearUserBinding)
				adminRoute.GET("/:id", middleware.RequirePermission(authz.UserRead), controller.GetUser)
				adminRoute.POST("/", middleware.RequirePermission(authz.UserWrite), controller.CreateUser)
				adminRoute.POST("/manage", middleware.RequireAnyPermission(authz.UserWrite, authz.UserQuota, authz.UserSecurity), controller.ManageUser)
				adminRoute.PUT("/", middleware.RequirePermission(authz.UserWrite), controller.UpdateUser)
				adminRoute.DELETE("/:id", middleware.RequirePermission(authz.UserWrite), controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", middleware.RequirePermission(authz.UserSecurity), controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", middleware.RequirePermission(authz.UserSecurity), controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", middleware.RequirePermission(authz.UserSecurity), controller.AdminDisable2FA)
			}
		}

		// Subscription billing (plans, purchase, admin management)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.UserAuth())
		{
			subscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.PUT("/self/consume-priority", controller.UpdateSubscriptionConsumePriority)
			subscriptionRoute.DELETE("/self/consume-priority", controller.ResetSubscriptionConsumePriority)
			subscriptionRoute.POST("/balance/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestBalancePay)
			subscriptionRoute.POST("/manual-wechat/pay", middleware.CriticalRateLimit(), controller.RequestManualWechatSubscription)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
			subscriptionRoute.POST("/waffo-pancake/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestWaffoPancakePay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		subscriptionAdminRoute.Use(middleware.ManagedAdminAuth())
		{
			subscriptionAdminRoute.GET("/plans", middleware.RequirePermission(authz.SubscriptionRead), controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", middleware.RequirePermission(authz.SubscriptionWrite), controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", middleware.RequirePermission(authz.SubscriptionWrite), controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", middleware.RequirePermission(authz.SubscriptionWrite), controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminBindSubscription)
			subscriptionAdminRoute.POST("/plans/:id/subscriptions/reset", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminResetPlanSubscriptions)
			subscriptionAdminRoute.GET("/orders", middleware.RequirePermission(authz.SubscriptionRead), controller.AdminListSubscriptionOrders)
			subscriptionAdminRoute.POST("/orders/:trade_no/complete", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminCompleteManualWechatSubscription)
			subscriptionAdminRoute.POST("/orders/:trade_no/expire", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminExpireManualWechatSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", middleware.RequirePermission(authz.SubscriptionRead), controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.POST("/users/:id/subscriptions/reset", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminResetUserSubscriptionsByPlan)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", middleware.RequirePermission(authz.SubscriptionOperate), controller.AdminDeleteUserSubscription)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", anonymousRequestBodyLimit, controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", anonymousRequestBodyLimit, controller.SubscriptionEpayReturn)
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.ManagedAdminAuth())
		{
			optionRoute.GET("/", middleware.RequireAnyPermission(authz.SystemSettingsRead, authz.PaymentRead, authz.OAuthRead), controller.GetOptions)
			optionRoute.GET("/request_policy", middleware.RequirePermission(authz.SystemSettingsRead), controller.GetRequestPolicy)
			optionRoute.PATCH("/request_policy", middleware.RequirePermission(authz.SystemSettingsWrite), controller.UpdateRequestPolicy)
			optionRoute.PUT("/", middleware.RequireAnyPermission(authz.SystemSettingsWrite, authz.PaymentWrite, authz.OAuthWrite), controller.UpdateOption)
			optionRoute.PUT("/passkey/domains", middleware.RequirePermission(authz.SystemSettingsWrite), controller.UpdatePasskeyDomains)
			optionRoute.GET("/model_pricing", middleware.RequirePermission(authz.SystemSettingsRead), controller.GetModelPricingConfig)
			optionRoute.PATCH("/model_pricing", middleware.RequirePermission(authz.SystemSettingsWrite), controller.UpdateModelPricingConfig)
			optionRoute.POST("/model_pricing/convert", middleware.RequirePermission(authz.SystemSettingsWrite), controller.PreviewModelPricingConversion)
			optionRoute.POST("/model_pricing/preview", middleware.RequirePermission(authz.SystemSettingsRead), controller.PreviewModelPricing)
			optionRoute.POST("/payment_compliance", middleware.RequirePermission(authz.PaymentWrite), controller.ConfirmPaymentCompliance)
			optionRoute.GET("/manual-wechat-payment", middleware.RequirePermission(authz.PaymentRead), controller.GetManualWechatPaymentSettings)
			optionRoute.PUT("/manual-wechat-payment", middleware.RequirePermission(authz.PaymentWrite), controller.UpdateManualWechatPaymentSettings)
			optionRoute.GET("/channel_affinity_cache", middleware.RequirePermission(authz.UsageLogAffinityView), controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", middleware.RequirePermission(authz.SystemSettingsWrite), controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", middleware.RequirePermission(authz.SystemSettingsWrite), controller.ResetModelRatio)
			optionRoute.GET("/waffo-pancake/catalog", middleware.RequirePermission(authz.PaymentRead), controller.ListWaffoPancakeCatalog)
			optionRoute.POST("/waffo-pancake/pair", middleware.RequirePermission(authz.PaymentWrite), controller.CreateWaffoPancakePair)
			optionRoute.POST("/waffo-pancake/save", middleware.RequirePermission(authz.PaymentWrite), controller.SaveWaffoPancake)
			optionRoute.POST("/waffo-pancake/subscription-product", middleware.RequirePermission(authz.PaymentWrite), controller.CreateWaffoPancakeSubscriptionProduct)
			optionRoute.GET("/waffo-pancake/subscription-product-options", middleware.RequirePermission(authz.PaymentRead), controller.ListWaffoPancakeSubscriptionProductOptions)
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.ManagedAdminAuth())
		{
			customOAuthRoute.POST("/discovery", middleware.RequirePermission(authz.OAuthWrite), controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", middleware.RequirePermission(authz.OAuthRead), controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", middleware.RequirePermission(authz.OAuthRead), controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", middleware.RequirePermission(authz.OAuthWrite), controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", middleware.RequirePermission(authz.OAuthWrite), controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", middleware.RequirePermission(authz.OAuthWrite), controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.ManagedAdminAuth())
		{
			performanceRoute.GET("/stats", middleware.RequirePermission(authz.PerformanceRead), controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", middleware.RequirePermission(authz.PerformanceOperate), controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", middleware.RequirePermission(authz.PerformanceOperate), controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", middleware.RequirePermission(authz.PerformanceOperate), controller.ForceGC)
			performanceRoute.GET("/logs", middleware.RequirePermission(authz.PerformanceRead), controller.GetLogFiles)
			performanceRoute.DELETE("/logs", middleware.RequirePermission(authz.PerformanceOperate), controller.CleanupLogFiles)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.DeploymentOperate))
		{
			ratioSyncRoute.GET("/channels", controller.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", controller.FetchUpstreamRatios)
		}
		taskPluginRoute := apiRouter.Group("/plugin/task")
		taskPluginRoute.Use(middleware.RootAuth())
		{
			taskPluginRoute.GET("", controller.ListTaskPlugins)
			taskPluginRoute.POST("", controller.UploadTaskPlugin)
			taskPluginRoute.PUT("", controller.UploadTaskPlugin)
			taskPluginRoute.GET("/runtime/status", controller.GetTaskPluginRuntime)
			taskPluginRoute.GET("/marketplace/sources", controller.GetTaskPluginMarketplaceSources)
			taskPluginRoute.PUT("/marketplace/sources", controller.UpdateTaskPluginMarketplaceSources)
			taskPluginRoute.GET("/:key", controller.GetTaskPlugin)
			taskPluginRoute.GET("/:key/icon", controller.GetTaskPluginIcon)
			taskPluginRoute.GET("/:key/versions", controller.GetTaskPluginVersions)
			taskPluginRoute.POST("/:key/activate", controller.ActivateTaskPlugin)
			taskPluginRoute.POST("/:key/status", controller.SetTaskPluginStatus)
			taskPluginRoute.POST("/:key/dryrun", controller.DryRunTaskPlugin)
			taskPluginRoute.DELETE("/:key/versions/:version", controller.DeleteTaskPluginVersion)
		}
		apiRouter.GET("/task_plugin_options", middleware.AdminAuth(), middleware.RequirePermission(authz.TaskPluginBind), controller.GetTaskPluginOptions)
		registerChannelRoutes(apiRouter)
		registerAuthzRoutes(apiRouter)
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		tokenRoute.Use(middleware.TokenOperationAudit())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/auto-groups", controller.GetTokenAutoGroups)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
			tokenRoute.POST("/batch/keys", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKeysBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		lotteryRoute := apiRouter.Group("/lottery")
		lotteryRoute.Use(middleware.UserAuth())
		{
			lotteryRoute.GET("/self", controller.GetLotteryPlansForSelf)
			lotteryRoute.GET("/plans/:id/participants", controller.GetLotteryParticipantsForSelf)
			lotteryRoute.GET("/plans/:id/results", controller.GetLotteryPlanResultsForSelf)
			lotteryRoute.GET("/results/self", controller.GetLotteryResultsForSelf)
			lotteryRoute.GET("/results/self/page", controller.GetLotteryResultsPageForSelf)
			lotteryRoute.GET("/results/self/pending", controller.GetClaimableLotteryResultsForSelf)
			lotteryRoute.GET("/notifications/self", controller.GetLotteryNotificationsForSelf)
			lotteryRoute.GET("/notifications/self/page", controller.GetLotteryNotificationsPageForSelf)
			lotteryRoute.POST("/notifications/self/read", controller.MarkLotteryNotificationsReadForSelf)
			lotteryRoute.POST("/:id/join", controller.JoinLotteryPlanForSelf)
			lotteryRoute.POST("/:id/leave", controller.LeaveLotteryPlanForSelf)
			lotteryRoute.POST("/results/:id/claim", controller.ClaimLotteryResultForSelf)
		}
		lotteryAdminRoute := apiRouter.Group("/lottery/admin")
		lotteryAdminRoute.Use(middleware.ManagedAdminAuth())
		{
			lotteryAdminRoute.GET("/plans", middleware.RequirePermission(authz.LotteryRead), controller.AdminListLotteryPlans)
			lotteryAdminRoute.POST("/plans", middleware.RequirePermission(authz.LotteryWrite), controller.AdminCreateLotteryPlan)
			lotteryAdminRoute.PATCH("/plans/:id", middleware.RequirePermission(authz.LotteryWrite), controller.AdminUpdateLotteryPlan)
			lotteryAdminRoute.POST("/plans/:id/cancel", middleware.RequirePermission(authz.LotteryOperate), controller.AdminCancelLotteryPlan)
			lotteryAdminRoute.GET("/plans/:id/prizes", middleware.RequirePermission(authz.LotteryRead), controller.AdminListLotteryPrizes)
			lotteryAdminRoute.GET("/plans/:id/results", middleware.RequirePermission(authz.LotteryRead), controller.AdminListLotteryResults)
			lotteryAdminRoute.POST("/plans/:id/draw", middleware.RequirePermission(authz.LotteryOperate), controller.AdminDrawLotteryPlan)
			lotteryAdminRoute.GET("/plans/:id/participants", middleware.RequirePermission(authz.LotteryRead), controller.AdminListLotteryParticipants)
			lotteryAdminRoute.PUT("/plans/:id/participants", middleware.RequirePermission(authz.LotteryOperate), controller.AdminUpdateLotteryParticipant)
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.ManagedAdminAuth())
		{
			redemptionRoute.GET("/", middleware.RequirePermission(authz.RedemptionRead), controller.GetAllRedemptions)
			redemptionRoute.GET("/search", middleware.RequirePermission(authz.RedemptionRead), controller.SearchRedemptions)
			redemptionRoute.GET("/:id", middleware.RequirePermission(authz.RedemptionRead), controller.GetRedemption)
			redemptionRoute.POST("/", middleware.RequirePermission(authz.RedemptionWrite), controller.AddRedemption)
			redemptionRoute.POST("/batch", middleware.RequirePermission(authz.RedemptionOperate), controller.DeleteRedemptionBatch)
			redemptionRoute.PUT("/", middleware.RequirePermission(authz.RedemptionWrite), controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", middleware.RequirePermission(authz.RedemptionOperate), controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", middleware.RequirePermission(authz.RedemptionOperate), controller.DeleteRedemption)
		}
		apiRouter.GET("/audit", middleware.DisableCache(), middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.AuditRead), controller.GetAuditLogs)
		apiRouter.GET("/audit/self", middleware.DisableCache(), middleware.UserAuth(), controller.GetAuditLogs)
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.UsageLogRead), controller.GetAllLogs)
		logRoute.GET("/stat", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.UsageLogSearch), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.UsageLogAffinityView), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.UsageLogSearch), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		systemTaskRoute := apiRouter.Group("/system-task")
		systemTaskRoute.Use(middleware.ManagedAdminAuth())
		{
			systemTaskRoute.POST("/log-cleanup", middleware.RequirePermission(authz.LogMaintenanceOperate), controller.CreateLogCleanupSystemTask)
			systemTaskRoute.GET("/list", middleware.RequirePermission(authz.LogMaintenanceOperate), controller.ListSystemTasks)
			systemTaskRoute.DELETE("/history", middleware.RequirePermission(authz.LogMaintenanceOperate), controller.DeleteSystemTaskHistory)
			systemTaskRoute.GET("/current", middleware.RequirePermission(authz.LogMaintenanceOperate), controller.GetCurrentSystemTask)
			systemTaskRoute.GET("/:task_id", middleware.RequirePermission(authz.LogMaintenanceOperate), controller.GetSystemTask)
		}
		systemInfoRoute := apiRouter.Group("/system-info")
		systemInfoRoute.Use(middleware.ManagedAdminAuth())
		{
			systemInfoRoute.GET("/instances", middleware.RequirePermission(authz.SystemInfoRead), controller.ListSystemInstances)
			systemInfoRoute.DELETE("/stale-instances", middleware.RequirePermission(authz.SystemInfoOperate), controller.DeleteStaleSystemInstances)
			systemInfoRoute.DELETE("/instances/:node_name", middleware.RequirePermission(authz.SystemInfoOperate), controller.DeleteStaleSystemInstance)
		}

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.ReportRead), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.ReportRead), controller.GetQuotaDatesByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)
		dataRoute.GET("/flow", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.ReportRead), controller.GetAllFlowQuotaDates)
		dataRoute.GET("/flow/self", middleware.UserAuth(), controller.GetUserFlowQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.ManagedAdminAuth())
		{
			groupRoute.GET("/", middleware.RequirePermission(authz.GroupRead), controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.ManagedAdminAuth())
		{
			prefillGroupRoute.GET("/", middleware.RequirePermission(authz.PrefillGroupRead), controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", middleware.RequirePermission(authz.PrefillGroupWrite), controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", middleware.RequirePermission(authz.PrefillGroupWrite), controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", middleware.RequirePermission(authz.PrefillGroupWrite), controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.TaskLogRead), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("", middleware.ManagedAdminAuth(), middleware.RequirePermission(authz.TaskLogRead), controller.GetAllTask)
			taskRoute.GET("/:task_id/artifacts", middleware.UserAuth(), controller.GetDashboardTaskArtifacts)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.ManagedAdminAuth())
		{
			vendorRoute.POST("/operations/preview", middleware.RequirePermission(authz.VendorWrite), controller.PreviewVendorOperation)
			vendorRoute.POST("/operations", middleware.RequirePermission(authz.VendorWrite), controller.ApplyVendorOperation)
			vendorRoute.GET("/", middleware.RequirePermission(authz.VendorRead), controller.GetAllVendors)
			vendorRoute.GET("/search", middleware.RequirePermission(authz.VendorRead), controller.SearchVendors)
			vendorRoute.GET("/:id", middleware.RequirePermission(authz.VendorRead), controller.GetVendorMeta)
			vendorRoute.POST("/", middleware.RequirePermission(authz.VendorWrite), controller.CreateVendorMeta)
			vendorRoute.PUT("/", middleware.RequirePermission(authz.VendorWrite), controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", middleware.RequirePermission(authz.VendorWrite), controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.ManagedAdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", middleware.RequirePermission(authz.ModelOperate), controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", middleware.RequirePermission(authz.ModelOperate), controller.SyncUpstreamModels)
			modelsRoute.POST("/delete", middleware.RequirePermission(authz.ModelWrite), controller.BatchDeleteModelMeta)
			modelsRoute.GET("/missing", middleware.RequirePermission(authz.ModelRead), controller.GetMissingModels)
			modelsRoute.GET("/", middleware.RequirePermission(authz.ModelRead), controller.GetAllModelsMeta)
			modelsRoute.GET("/search", middleware.RequirePermission(authz.ModelRead), controller.SearchModelsMeta)
			modelsRoute.GET("/:id", middleware.RequirePermission(authz.ModelRead), controller.GetModelMeta)
			modelsRoute.POST("/", middleware.RequirePermission(authz.ModelWrite), controller.CreateModelMeta)
			modelsRoute.PUT("/", middleware.RequirePermission(authz.ModelWrite), controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", middleware.RequirePermission(authz.ModelWrite), controller.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.ManagedAdminAuth())
		{
			deploymentsRoute.GET("/settings", middleware.RequirePermission(authz.DeploymentRead), controller.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", middleware.RequirePermission(authz.DeploymentOperate), controller.TestIoNetConnection)
			deploymentsRoute.GET("/", middleware.RequirePermission(authz.DeploymentRead), controller.GetAllDeployments)
			deploymentsRoute.GET("/search", middleware.RequirePermission(authz.DeploymentRead), controller.SearchDeployments)
			deploymentsRoute.POST("/test-connection", middleware.RequirePermission(authz.DeploymentOperate), controller.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", middleware.RequirePermission(authz.DeploymentRead), controller.GetHardwareTypes)
			deploymentsRoute.GET("/locations", middleware.RequirePermission(authz.DeploymentRead), controller.GetLocations)
			deploymentsRoute.GET("/available-replicas", middleware.RequirePermission(authz.DeploymentRead), controller.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", middleware.RequirePermission(authz.DeploymentRead), controller.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", middleware.RequirePermission(authz.DeploymentRead), controller.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", middleware.RequirePermission(authz.DeploymentWrite), controller.CreateDeployment)

			deploymentsRoute.GET("/:id", middleware.RequirePermission(authz.DeploymentRead), controller.GetDeployment)
			deploymentsRoute.GET("/:id/logs", middleware.RequirePermission(authz.DeploymentRead), controller.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", middleware.RequirePermission(authz.DeploymentRead), controller.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", middleware.RequirePermission(authz.DeploymentRead), controller.GetContainerDetails)
			deploymentsRoute.PUT("/:id", middleware.RequirePermission(authz.DeploymentWrite), controller.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", middleware.RequirePermission(authz.DeploymentWrite), controller.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", middleware.RequirePermission(authz.DeploymentOperate), controller.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", middleware.RequirePermission(authz.DeploymentWrite), controller.DeleteDeployment)
		}
	}
}
