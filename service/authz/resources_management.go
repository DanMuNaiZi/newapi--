package authz

const (
	ResourceUser           = "user"
	ResourceUsageLog       = "usage_log"
	ResourceModel          = "model"
	ResourceDeployment     = "deployment"
	ResourceSubscription   = "subscription"
	ResourceRedemption     = "redemption"
	ResourceLottery        = "lottery"
	ResourceVendor         = "vendor"
	ResourceGroup          = "group"
	ResourcePrefillGroup   = "prefill_group"
	ResourceReport         = "report"
	ResourceTaskLog        = "task_log"
	ResourceSystemSettings = "system_settings"
	ResourcePayment        = "payment"
	ResourceOAuth          = "oauth"
	ResourcePerformance    = "performance"
	ResourceSystemInfo     = "system_info"
	ResourceLogMaintenance = "log_maintenance"

	ActionQuota           = "quota"
	ActionSecurity        = "security"
	ActionSearch          = "search"
	ActionExport          = "export"
	ActionActualModelView = "actual_model_view"
	ActionAffinityView    = "affinity_view"
	ActionSensitiveView   = "sensitive_view"
)

var (
	UserRead     = Permission{Resource: ResourceUser, Action: ActionRead}
	UserWrite    = Permission{Resource: ResourceUser, Action: ActionWrite}
	UserQuota    = Permission{Resource: ResourceUser, Action: ActionQuota}
	UserSecurity = Permission{Resource: ResourceUser, Action: ActionSecurity}

	UsageLogRead            = Permission{Resource: ResourceUsageLog, Action: ActionRead}
	UsageLogSearch          = Permission{Resource: ResourceUsageLog, Action: ActionSearch}
	UsageLogExport          = Permission{Resource: ResourceUsageLog, Action: ActionExport}
	UsageLogActualModelView = Permission{Resource: ResourceUsageLog, Action: ActionActualModelView}
	UsageLogAffinityView    = Permission{Resource: ResourceUsageLog, Action: ActionAffinityView}

	ModelRead         = Permission{Resource: ResourceModel, Action: ActionRead}
	ModelWrite        = Permission{Resource: ResourceModel, Action: ActionWrite}
	ModelOperate      = Permission{Resource: ResourceModel, Action: ActionOperate}
	DeploymentRead    = Permission{Resource: ResourceDeployment, Action: ActionRead}
	DeploymentWrite   = Permission{Resource: ResourceDeployment, Action: ActionWrite}
	DeploymentOperate = Permission{Resource: ResourceDeployment, Action: ActionOperate}

	SubscriptionRead    = Permission{Resource: ResourceSubscription, Action: ActionRead}
	SubscriptionWrite   = Permission{Resource: ResourceSubscription, Action: ActionWrite}
	SubscriptionOperate = Permission{Resource: ResourceSubscription, Action: ActionOperate}
	RedemptionRead      = Permission{Resource: ResourceRedemption, Action: ActionRead}
	RedemptionWrite     = Permission{Resource: ResourceRedemption, Action: ActionWrite}
	RedemptionOperate   = Permission{Resource: ResourceRedemption, Action: ActionOperate}
	LotteryRead         = Permission{Resource: ResourceLottery, Action: ActionRead}
	LotteryWrite        = Permission{Resource: ResourceLottery, Action: ActionWrite}
	LotteryOperate      = Permission{Resource: ResourceLottery, Action: ActionOperate}

	VendorRead        = Permission{Resource: ResourceVendor, Action: ActionRead}
	VendorWrite       = Permission{Resource: ResourceVendor, Action: ActionWrite}
	GroupRead         = Permission{Resource: ResourceGroup, Action: ActionRead}
	PrefillGroupRead  = Permission{Resource: ResourcePrefillGroup, Action: ActionRead}
	PrefillGroupWrite = Permission{Resource: ResourcePrefillGroup, Action: ActionWrite}
	ReportRead        = Permission{Resource: ResourceReport, Action: ActionRead}
	TaskLogRead       = Permission{Resource: ResourceTaskLog, Action: ActionRead}

	SystemSettingsRead          = Permission{Resource: ResourceSystemSettings, Action: ActionRead}
	SystemSettingsWrite         = Permission{Resource: ResourceSystemSettings, Action: ActionWrite}
	SystemSettingsSensitiveView = Permission{Resource: ResourceSystemSettings, Action: ActionSensitiveView}
	PaymentRead                 = Permission{Resource: ResourcePayment, Action: ActionRead}
	PaymentWrite                = Permission{Resource: ResourcePayment, Action: ActionWrite}
	PaymentSensitiveView        = Permission{Resource: ResourcePayment, Action: ActionSensitiveView}
	OAuthRead                   = Permission{Resource: ResourceOAuth, Action: ActionRead}
	OAuthWrite                  = Permission{Resource: ResourceOAuth, Action: ActionWrite}
	OAuthSensitiveView          = Permission{Resource: ResourceOAuth, Action: ActionSensitiveView}
	PerformanceRead             = Permission{Resource: ResourcePerformance, Action: ActionRead}
	PerformanceOperate          = Permission{Resource: ResourcePerformance, Action: ActionOperate}
	SystemInfoRead              = Permission{Resource: ResourceSystemInfo, Action: ActionRead}
	SystemInfoOperate           = Permission{Resource: ResourceSystemInfo, Action: ActionOperate}
	LogMaintenanceOperate       = Permission{Resource: ResourceLogMaintenance, Action: ActionOperate}
)

func init() {
	admin := []string{BuiltInRoleAdmin}
	RegisterResource(ResourceDefinition{Resource: ResourceUser, LabelKey: "User Management", Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View users", DescriptionKey: "View user lists and profiles.", DefaultRoles: admin},
		{Action: ActionWrite, LabelKey: "Edit users", DescriptionKey: "Create and edit regular users.", DefaultRoles: admin},
		{Action: ActionQuota, LabelKey: "Manage user quota", DescriptionKey: "Adjust quota for regular users.", DefaultRoles: admin},
		{Action: ActionSecurity, LabelKey: "Manage user security", DescriptionKey: "Reset 2FA and remove user bindings.", DefaultRoles: admin},
	}})
	RegisterResource(ResourceDefinition{Resource: ResourceUsageLog, LabelKey: "Usage Logs", Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View usage logs", DescriptionKey: "View global usage logs.", DefaultRoles: admin},
		{Action: ActionSearch, LabelKey: "Search usage logs", DescriptionKey: "Search and view usage statistics.", DefaultRoles: admin},
		{Action: ActionExport, LabelKey: "Export usage logs", DescriptionKey: "Export usage log data."},
		{Action: ActionActualModelView, LabelKey: "View actual models", DescriptionKey: "View the upstream model used for a request.", DefaultRoles: admin},
		{Action: ActionAffinityView, LabelKey: "View channel affinity cache", DescriptionKey: "View channel affinity cache statistics.", DefaultRoles: admin},
	}})
	registerManagementResource(ResourceModel, "Model Management", admin)
	registerManagementResource(ResourceDeployment, "Deployment Management", admin)
	registerManagementResource(ResourceSubscription, "Subscription Management", admin)
	registerManagementResource(ResourceRedemption, "Redemption Code Management", admin)
	registerManagementResource(ResourceLottery, "Lottery Management", admin)
	registerReadWriteResource(ResourceVendor, "Vendor Management", admin)
	RegisterResource(ResourceDefinition{Resource: ResourceGroup, LabelKey: "Group Management", Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View groups", DescriptionKey: "View available user groups.", DefaultRoles: admin},
	}})
	registerReadWriteResource(ResourcePrefillGroup, "Prefill Group Management", admin)
	RegisterResource(ResourceDefinition{Resource: ResourceReport, LabelKey: "Data Reports", Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View data reports", DescriptionKey: "View global quota and flow reports.", DefaultRoles: admin},
	}})
	RegisterResource(ResourceDefinition{Resource: ResourceTaskLog, LabelKey: "Task Logs", Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View task logs", DescriptionKey: "View global task and drawing records.", DefaultRoles: admin},
	}})
	registerSensitiveManagementResource(ResourceSystemSettings, "System Settings")
	registerSensitiveManagementResource(ResourcePayment, "Payment Configuration")
	registerSensitiveManagementResource(ResourceOAuth, "OAuth Configuration")
	registerManagementResource(ResourcePerformance, "Performance Operations", nil)
	registerManagementResource(ResourceSystemInfo, "System Instance Management", nil)
	RegisterResource(ResourceDefinition{Resource: ResourceLogMaintenance, LabelKey: "Log Maintenance", Actions: []ActionDefinition{
		{Action: ActionOperate, LabelKey: "Clean usage logs", DescriptionKey: "Create and manage usage log cleanup tasks."},
	}})
}

func registerManagementResource(resource string, label string, defaultRoles []string) {
	RegisterResource(ResourceDefinition{Resource: resource, LabelKey: label, Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View", DescriptionKey: "View this management module.", DefaultRoles: defaultRoles},
		{Action: ActionWrite, LabelKey: "Edit", DescriptionKey: "Edit this management module.", DefaultRoles: defaultRoles},
		{Action: ActionOperate, LabelKey: "Operate", DescriptionKey: "Run operational actions for this module.", DefaultRoles: defaultRoles},
	}})
}

func registerReadWriteResource(resource string, label string, defaultRoles []string) {
	RegisterResource(ResourceDefinition{Resource: resource, LabelKey: label, Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View", DescriptionKey: "View this management module.", DefaultRoles: defaultRoles},
		{Action: ActionWrite, LabelKey: "Edit", DescriptionKey: "Edit this management module.", DefaultRoles: defaultRoles},
	}})
}

func registerSensitiveManagementResource(resource string, label string) {
	RegisterResource(ResourceDefinition{Resource: resource, LabelKey: label, Actions: []ActionDefinition{
		{Action: ActionRead, LabelKey: "View", DescriptionKey: "View this configuration module."},
		{Action: ActionWrite, LabelKey: "Edit", DescriptionKey: "Edit this configuration module."},
		{Action: ActionSensitiveView, LabelKey: "View secrets", DescriptionKey: "View sensitive configuration values for this module."},
	}})
}
