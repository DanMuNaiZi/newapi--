package dto

type RewardSpec struct {
	Type               string `json:"type"`
	Amount             string `json:"amount"`
	Unit               string `json:"unit"`
	SubscriptionPlanId int    `json:"subscription_plan_id"`
}
