package model

import (
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type LotteryPlanStatus string

const (
	LotteryPlanStatusDraft     LotteryPlanStatus = "draft"
	LotteryPlanStatusScheduled LotteryPlanStatus = "scheduled"
	LotteryPlanStatusOpen      LotteryPlanStatus = "open"
	LotteryPlanStatusDrawing   LotteryPlanStatus = "drawing"
	LotteryPlanStatusFinished  LotteryPlanStatus = "finished"
	LotteryPlanStatusCancelled LotteryPlanStatus = "cancelled"
)

type LotteryEligibilityMode string

const (
	LotteryEligibilityAll    LotteryEligibilityMode = "all"
	LotteryEligibilityGroups LotteryEligibilityMode = "groups"
	LotteryEligibilityUsers  LotteryEligibilityMode = "users"
)

type LotteryParticipantStatus string

const (
	LotteryParticipantStatusJoined LotteryParticipantStatus = "joined"
	LotteryParticipantStatusLeft   LotteryParticipantStatus = "left"
)

type LotteryRewardType string

const (
	LotteryRewardQuota        LotteryRewardType = "quota"
	LotteryRewardSubscription LotteryRewardType = "subscription"
)

type LotteryFulfillmentMode string

const (
	LotteryFulfillmentAuto           LotteryFulfillmentMode = "auto"
	LotteryFulfillmentSelfClaim      LotteryFulfillmentMode = "self_claim"
	LotteryFulfillmentRedemptionCode LotteryFulfillmentMode = "redemption_code"
)

type LotteryDrawTrigger string

const (
	LotteryDrawTriggerScheduled LotteryDrawTrigger = "scheduled"
	LotteryDrawTriggerFull      LotteryDrawTrigger = "full"
	LotteryDrawTriggerManual    LotteryDrawTrigger = "manual"
)

type LotteryDrawRunStatus string

const (
	LotteryDrawRunStatusFinished LotteryDrawRunStatus = "finished"
	LotteryDrawRunStatusEmpty    LotteryDrawRunStatus = "empty"
)

type LotteryPlan struct {
	Id                    int                    `json:"id"`
	Title                 string                 `json:"title" gorm:"type:varchar(128);not null"`
	Description           string                 `json:"description" gorm:"type:text"`
	Status                LotteryPlanStatus      `json:"status" gorm:"type:varchar(32);index"`
	EligibilityMode       LotteryEligibilityMode `json:"eligibility_mode" gorm:"type:varchar(32);not null"`
	MaxParticipants       int                    `json:"max_participants" gorm:"type:int;not null"`
	RegistrationStartTime int64                  `json:"registration_start_time" gorm:"type:bigint;index"`
	DrawTime              int64                  `json:"draw_time" gorm:"type:bigint;index"`
	DrawAlgorithm         string                 `json:"draw_algorithm" gorm:"type:varchar(32);not null"`
	CreatedBy             int                    `json:"created_by" gorm:"index"`
	CreatedAt             int64                  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt             int64                  `json:"updated_at" gorm:"type:bigint"`
}

func (plan *LotteryPlan) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if plan.Status == "" {
		plan.Status = LotteryPlanStatusDraft
	}
	if plan.DrawAlgorithm == "" {
		plan.DrawAlgorithm = "weighted_random_without_replacement"
	}
	if plan.CreatedAt == 0 {
		plan.CreatedAt = now
	}
	if plan.UpdatedAt == 0 {
		plan.UpdatedAt = now
	}
	return nil
}

func (plan *LotteryPlan) BeforeUpdate(_ *gorm.DB) error {
	plan.UpdatedAt = common.GetTimestamp()
	return nil
}

type LotteryPlanGroup struct {
	Id     int    `json:"id"`
	PlanId int    `json:"plan_id" gorm:"uniqueIndex:idx_lottery_plan_group,priority:1"`
	Group  string `json:"group" gorm:"type:varchar(64);uniqueIndex:idx_lottery_plan_group,priority:2"`
}

type LotteryPlanUser struct {
	Id     int `json:"id"`
	PlanId int `json:"plan_id" gorm:"uniqueIndex:idx_lottery_plan_user,priority:1"`
	UserId int `json:"user_id" gorm:"uniqueIndex:idx_lottery_plan_user,priority:2;index"`
}

type LotteryPrize struct {
	Id                   int                    `json:"id"`
	PlanId               int                    `json:"plan_id" gorm:"index"`
	Name                 string                 `json:"name" gorm:"type:varchar(128);not null"`
	Quantity             int                    `json:"quantity" gorm:"type:int;not null"`
	RewardType           LotteryRewardType      `json:"reward_type" gorm:"type:varchar(32);not null"`
	Quota                int                    `json:"quota" gorm:"type:int;not null"`
	SubscriptionPlanId   int                    `json:"subscription_plan_id" gorm:"index"`
	SubscriptionSnapshot string                 `json:"subscription_snapshot" gorm:"type:text"`
	FulfillmentMode      LotteryFulfillmentMode `json:"fulfillment_mode" gorm:"type:varchar(32);not null"`
	ClaimExpireSeconds   int64                  `json:"claim_expire_seconds" gorm:"type:bigint"`
	SortOrder            int                    `json:"sort_order" gorm:"type:int"`
}

type LotteryParticipant struct {
	Id            int                      `json:"id"`
	PlanId        int                      `json:"plan_id" gorm:"uniqueIndex:idx_lottery_participant,priority:1;index"`
	UserId        int                      `json:"user_id" gorm:"uniqueIndex:idx_lottery_participant,priority:2;index"`
	UserGroup     string                   `json:"user_group" gorm:"type:varchar(64)"`
	Weight        int                      `json:"weight" gorm:"type:int;not null"`
	PresetPrizeId int                      `json:"preset_prize_id" gorm:"index"`
	Status        LotteryParticipantStatus `json:"status" gorm:"type:varchar(32);index"`
	JoinedAt      int64                    `json:"joined_at" gorm:"type:bigint"`
	LeftAt        int64                    `json:"left_at" gorm:"type:bigint"`
}

type LotteryDrawRun struct {
	Id               int                  `json:"id"`
	PlanId           int                  `json:"plan_id" gorm:"uniqueIndex"`
	Status           LotteryDrawRunStatus `json:"status" gorm:"type:varchar(32);index"`
	Trigger          LotteryDrawTrigger   `json:"trigger" gorm:"type:varchar(32)"`
	Reason           string               `json:"reason" gorm:"type:text"`
	ParticipantCount int                  `json:"participant_count" gorm:"type:int"`
	CreatedAt        int64                `json:"created_at" gorm:"type:bigint"`
	FinishedAt       int64                `json:"finished_at" gorm:"type:bigint"`
}

type LotteryResult struct {
	Id                   int                    `json:"id"`
	PlanId               int                    `json:"plan_id" gorm:"uniqueIndex:idx_lottery_result_user,priority:1;index"`
	DrawRunId            int                    `json:"draw_run_id" gorm:"index"`
	UserId               int                    `json:"user_id" gorm:"uniqueIndex:idx_lottery_result_user,priority:2;index"`
	PrizeId              int                    `json:"prize_id" gorm:"index"`
	PrizeSnapshot        string                 `json:"prize_snapshot" gorm:"type:text"`
	RewardType           LotteryRewardType      `json:"reward_type" gorm:"type:varchar(32)"`
	Quota                int                    `json:"quota" gorm:"type:int"`
	SubscriptionPlanId   int                    `json:"subscription_plan_id" gorm:"index"`
	SubscriptionSnapshot string                 `json:"subscription_snapshot" gorm:"type:text"`
	FulfillmentMode      LotteryFulfillmentMode `json:"fulfillment_mode" gorm:"type:varchar(32)"`
	FulfillmentStatus    string                 `json:"fulfillment_status" gorm:"type:varchar(32);index"`
	ClaimExpiresAt       int64                  `json:"claim_expires_at" gorm:"type:bigint;index"`
	ClaimedAt            int64                  `json:"claimed_at" gorm:"type:bigint"`
	CreatedAt            int64                  `json:"created_at" gorm:"type:bigint"`
}

type LotteryNotification struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index"`
	PlanId    int    `json:"plan_id" gorm:"index"`
	Type      string `json:"type" gorm:"type:varchar(64);index"`
	Content   string `json:"content" gorm:"type:text"`
	ReadAt    int64  `json:"read_at" gorm:"type:bigint"`
	CreatedAt int64  `json:"created_at" gorm:"type:bigint;index"`
}

func CreateLotteryPlan(plan *LotteryPlan, userIds []int, groups []string, prizes []*LotteryPrize) error {
	if plan == nil || strings.TrimSpace(plan.Title) == "" {
		return errors.New("lottery plan title is required")
	}
	if plan.MaxParticipants <= 0 {
		return errors.New("max participants must be positive")
	}
	if plan.RegistrationStartTime <= 0 || plan.DrawTime <= plan.RegistrationStartTime {
		return errors.New("invalid lottery schedule")
	}
	if plan.EligibilityMode != LotteryEligibilityAll && plan.EligibilityMode != LotteryEligibilityGroups && plan.EligibilityMode != LotteryEligibilityUsers {
		return errors.New("invalid eligibility mode")
	}
	if plan.EligibilityMode == LotteryEligibilityUsers && len(userIds) == 0 {
		return errors.New("user allow list is required")
	}
	if plan.EligibilityMode == LotteryEligibilityGroups && len(groups) == 0 {
		return errors.New("group allow list is required")
	}
	if len(prizes) == 0 {
		return errors.New("at least one prize is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		for _, userId := range userIds {
			if userId <= 0 {
				return errors.New("invalid allowed user")
			}
			entry := &LotteryPlanUser{PlanId: plan.Id, UserId: userId}
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if group == "" {
				return errors.New("invalid allowed group")
			}
			entry := &LotteryPlanGroup{PlanId: plan.Id, Group: group}
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		for index, prize := range prizes {
			if prize == nil || strings.TrimSpace(prize.Name) == "" || prize.Quantity <= 0 {
				return errors.New("invalid lottery prize")
			}
			if prize.RewardType != LotteryRewardQuota && prize.RewardType != LotteryRewardSubscription {
				return errors.New("invalid prize reward type")
			}
			if prize.FulfillmentMode != LotteryFulfillmentAuto && prize.FulfillmentMode != LotteryFulfillmentSelfClaim && prize.FulfillmentMode != LotteryFulfillmentRedemptionCode {
				return errors.New("invalid prize fulfillment mode")
			}
			if prize.RewardType == LotteryRewardQuota && prize.Quota <= 0 {
				return errors.New("quota prize must be positive")
			}
			if prize.RewardType == LotteryRewardSubscription && prize.SubscriptionPlanId <= 0 && prize.SubscriptionSnapshot == "" {
				return errors.New("subscription prize is required")
			}
			if prize.RewardType == LotteryRewardSubscription && prize.SubscriptionSnapshot == "" {
				var subscriptionPlan SubscriptionPlan
				if err := tx.First(&subscriptionPlan, prize.SubscriptionPlanId).Error; err != nil {
					return err
				}
				snapshot, err := common.Marshal(subscriptionPlan)
				if err != nil {
					return err
				}
				prize.SubscriptionSnapshot = string(snapshot)
			}
			prize.PlanId = plan.Id
			prize.SortOrder = index
			if err := tx.Create(prize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func JoinLotteryPlan(planId int, userId int) error {
	if planId <= 0 || userId <= 0 {
		return errors.New("invalid lottery participant")
	}
	shouldDraw := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var plan LotteryPlan
		if err := lockForUpdate(tx).First(&plan, planId).Error; err != nil {
			return err
		}
		if plan.Status != LotteryPlanStatusOpen {
			return errors.New("lottery plan is not open")
		}
		now := common.GetTimestamp()
		if now < plan.RegistrationStartTime || now >= plan.DrawTime {
			return errors.New("lottery registration is not available")
		}
		var user User
		if err := lockForUpdate(tx).First(&user, userId).Error; err != nil {
			return err
		}
		if user.Role == common.RoleAdminUser || user.Role == common.RoleRootUser {
			return errors.New("administrators cannot join lotteries")
		}
		if user.Status != common.UserStatusEnabled {
			return errors.New("user is not enabled")
		}
		switch plan.EligibilityMode {
		case LotteryEligibilityGroups:
			var count int64
			if err := tx.Model(&LotteryPlanGroup{}).Where("plan_id = ? AND "+commonGroupCol+" = ?", plan.Id, user.Group).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return errors.New("user is not eligible for this lottery")
			}
		case LotteryEligibilityUsers:
			var count int64
			if err := tx.Model(&LotteryPlanUser{}).Where("plan_id = ? AND user_id = ?", plan.Id, userId).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return errors.New("user is not eligible for this lottery")
			}
		}
		var joinedCount int64
		if err := tx.Model(&LotteryParticipant{}).Where("plan_id = ? AND status = ?", plan.Id, LotteryParticipantStatusJoined).Count(&joinedCount).Error; err != nil {
			return err
		}
		var participant LotteryParticipant
		err := lockForUpdate(tx).Where("plan_id = ? AND user_id = ?", plan.Id, userId).First(&participant).Error
		if err == nil && participant.Status == LotteryParticipantStatusJoined {
			return errors.New("user already joined this lottery")
		}
		if joinedCount >= int64(plan.MaxParticipants) {
			return errors.New("lottery participant limit reached")
		}
		shouldDraw = joinedCount+1 >= int64(plan.MaxParticipants)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&LotteryParticipant{
				PlanId:    plan.Id,
				UserId:    userId,
				UserGroup: user.Group,
				Weight:    100,
				Status:    LotteryParticipantStatusJoined,
				JoinedAt:  now,
			}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&LotteryParticipant{}).Where("id = ?", participant.Id).Updates(map[string]interface{}{
			"user_group": user.Group,
			"weight":     100,
			"status":     LotteryParticipantStatusJoined,
			"joined_at":  now,
			"left_at":    0,
		}).Error
	})
	if err != nil {
		return err
	}
	if shouldDraw {
		_, err = DrawLotteryPlan(planId, LotteryDrawTriggerFull, "")
	}
	return err
}

func LeaveLotteryPlan(planId int, userId int) error {
	if planId <= 0 || userId <= 0 {
		return errors.New("invalid lottery participant")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var plan LotteryPlan
		if err := lockForUpdate(tx).First(&plan, planId).Error; err != nil {
			return err
		}
		if plan.Status != LotteryPlanStatusOpen {
			return errors.New("lottery plan does not allow withdrawal")
		}
		result := tx.Model(&LotteryParticipant{}).
			Where("plan_id = ? AND user_id = ? AND status = ?", plan.Id, userId, LotteryParticipantStatusJoined).
			Updates(map[string]interface{}{
				"status":          LotteryParticipantStatusLeft,
				"left_at":         common.GetTimestamp(),
				"preset_prize_id": 0,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("user has not joined this lottery")
		}
		return nil
	})
}

func SetLotteryParticipantPreset(planId int, userId int, prizeId int) error {
	if planId <= 0 || userId <= 0 || prizeId <= 0 {
		return errors.New("invalid lottery preset")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var plan LotteryPlan
		if err := lockForUpdate(tx).First(&plan, planId).Error; err != nil {
			return err
		}
		if plan.Status != LotteryPlanStatusOpen {
			return errors.New("lottery plan is not open")
		}
		var prize LotteryPrize
		if err := tx.Where("id = ? AND plan_id = ?", prizeId, plan.Id).First(&prize).Error; err != nil {
			return err
		}
		return tx.Model(&LotteryParticipant{}).Where("plan_id = ? AND user_id = ? AND status = ?", plan.Id, userId, LotteryParticipantStatusJoined).Update("preset_prize_id", prize.Id).Error
	})
}

func DrawLotteryPlan(planId int, trigger LotteryDrawTrigger, reason string) (*LotteryDrawRun, error) {
	if planId <= 0 {
		return nil, errors.New("invalid lottery plan")
	}
	if trigger != LotteryDrawTriggerScheduled && trigger != LotteryDrawTriggerFull && trigger != LotteryDrawTriggerManual {
		return nil, errors.New("invalid lottery draw trigger")
	}
	if trigger == LotteryDrawTriggerManual && strings.TrimSpace(reason) == "" {
		return nil, errors.New("manual lottery draw reason is required")
	}
	run := &LotteryDrawRun{}
	autoResultIds := make([]int, 0)
	err := DB.Transaction(func(tx *gorm.DB) error {
		var plan LotteryPlan
		if err := lockForUpdate(tx).First(&plan, planId).Error; err != nil {
			return err
		}
		if plan.Status != LotteryPlanStatusOpen {
			return errors.New("lottery plan cannot be drawn")
		}
		var participants []LotteryParticipant
		if err := lockForUpdate(tx).Where("plan_id = ? AND status = ?", plan.Id, LotteryParticipantStatusJoined).Order("id asc").Find(&participants).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		run.PlanId = plan.Id
		run.Trigger = trigger
		run.Reason = strings.TrimSpace(reason)
		run.ParticipantCount = len(participants)
		run.CreatedAt = now
		run.FinishedAt = now
		if len(participants) == 0 {
			run.Status = LotteryDrawRunStatusEmpty
			if err := tx.Create(run).Error; err != nil {
				return err
			}
			return tx.Model(&LotteryPlan{}).Where("id = ? AND status = ?", plan.Id, LotteryPlanStatusOpen).Update("status", LotteryPlanStatusFinished).Error
		}
		var prizes []LotteryPrize
		if err := tx.Where("plan_id = ?", plan.Id).Order("sort_order asc, id asc").Find(&prizes).Error; err != nil {
			return err
		}
		if len(prizes) == 0 {
			return errors.New("lottery plan has no prizes")
		}
		remaining := make(map[int]int, len(prizes))
		prizeByID := make(map[int]LotteryPrize, len(prizes))
		for _, prize := range prizes {
			remaining[prize.Id] = prize.Quantity
			prizeByID[prize.Id] = prize
		}
		selected := make(map[int]struct{}, len(participants))
		results := make([]LotteryResult, 0)
		for _, participant := range participants {
			if participant.PresetPrizeId == 0 {
				continue
			}
			prize, ok := prizeByID[participant.PresetPrizeId]
			if !ok || remaining[prize.Id] <= 0 {
				return errors.New("preset prize exceeds available slots")
			}
			remaining[prize.Id]--
			selected[participant.UserId] = struct{}{}
			results = append(results, lotteryResultFromPrize(plan.Id, participant.UserId, prize, now))
		}
		for _, prize := range prizes {
			for remaining[prize.Id] > 0 {
				candidates := make([]LotteryParticipant, 0, len(participants))
				for _, participant := range participants {
					if _, used := selected[participant.UserId]; !used {
						candidates = append(candidates, participant)
					}
				}
				if len(candidates) == 0 {
					break
				}
				winner, err := selectWeightedLotteryParticipant(candidates)
				if err != nil {
					return err
				}
				selected[winner.UserId] = struct{}{}
				remaining[prize.Id]--
				results = append(results, lotteryResultFromPrize(plan.Id, winner.UserId, prize, now))
			}
		}
		run.Status = LotteryDrawRunStatusFinished
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		for index := range results {
			results[index].DrawRunId = run.Id
			if err := tx.Create(&results[index]).Error; err != nil {
				return err
			}
			if results[index].FulfillmentMode == LotteryFulfillmentAuto {
				autoResultIds = append(autoResultIds, results[index].Id)
			}
		}
		return tx.Model(&LotteryPlan{}).Where("id = ? AND status = ?", plan.Id, LotteryPlanStatusOpen).Update("status", LotteryPlanStatusFinished).Error
	})
	if err != nil {
		return nil, err
	}
	for _, resultId := range autoResultIds {
		if err := FulfillLotteryResult(resultId); err != nil {
			common.SysError("failed to auto fulfill lottery result: " + err.Error())
		}
	}
	return run, nil
}

// ProcessLotterySchedule advances plans that have reached registration or draw
// time. Each draw re-checks the plan state transactionally, so concurrent
// scheduler nodes can safely scan the same rows.
func ProcessLotterySchedule(now int64) error {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	if err := DB.Model(&LotteryPlan{}).
		Where("status = ? AND registration_start_time <= ?", LotteryPlanStatusScheduled, now).
		Update("status", LotteryPlanStatusOpen).Error; err != nil {
		return err
	}
	var plans []LotteryPlan
	if err := DB.Where("status = ? AND draw_time <= ?", LotteryPlanStatusOpen, now).Order("id asc").Find(&plans).Error; err != nil {
		return err
	}
	for _, plan := range plans {
		if _, err := DrawLotteryPlan(plan.Id, LotteryDrawTriggerScheduled, ""); err != nil {
			return err
		}
	}
	return nil
}

func lotteryResultFromPrize(planId int, userId int, prize LotteryPrize, now int64) LotteryResult {
	fulfillmentStatus := "pending"
	claimExpiresAt := int64(0)
	if prize.FulfillmentMode == LotteryFulfillmentAuto {
		fulfillmentStatus = "pending_auto"
	} else if prize.ClaimExpireSeconds > 0 {
		claimExpiresAt = now + prize.ClaimExpireSeconds
	}
	return LotteryResult{
		PlanId:               planId,
		UserId:               userId,
		PrizeId:              prize.Id,
		RewardType:           prize.RewardType,
		Quota:                prize.Quota,
		SubscriptionPlanId:   prize.SubscriptionPlanId,
		SubscriptionSnapshot: prize.SubscriptionSnapshot,
		FulfillmentMode:      prize.FulfillmentMode,
		FulfillmentStatus:    fulfillmentStatus,
		ClaimExpiresAt:       claimExpiresAt,
		CreatedAt:            now,
	}
}

func ClaimLotteryResult(resultId int, userId int) error {
	if resultId <= 0 || userId <= 0 {
		return errors.New("invalid lottery claim")
	}
	return fulfillLotteryResult(resultId, userId, false)
}

func FulfillLotteryResult(resultId int) error {
	if resultId <= 0 {
		return errors.New("invalid lottery result")
	}
	return fulfillLotteryResult(resultId, 0, true)
}

func fulfillLotteryResult(resultId int, requestedUserId int, allowAuto bool) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var result LotteryResult
		if err := lockForUpdate(tx).First(&result, resultId).Error; err != nil {
			return err
		}
		if requestedUserId != 0 && result.UserId != requestedUserId {
			return errors.New("lottery result does not belong to user")
		}
		if result.FulfillmentStatus == "fulfilled" {
			return errors.New("lottery reward has already been fulfilled")
		}
		if result.ClaimExpiresAt > 0 && result.ClaimExpiresAt < common.GetTimestamp() {
			return errors.New("lottery reward claim has expired")
		}
		if !allowAuto && result.FulfillmentMode == LotteryFulfillmentAuto {
			return errors.New("lottery reward is fulfilled automatically")
		}
		switch result.RewardType {
		case LotteryRewardQuota:
			if result.Quota <= 0 {
				return errors.New("invalid lottery quota reward")
			}
			if err := tx.Model(&User{}).Where("id = ?", result.UserId).Update("quota", gorm.Expr("quota + ?", result.Quota)).Error; err != nil {
				return err
			}
		case LotteryRewardSubscription:
			if result.SubscriptionSnapshot == "" {
				return errors.New("lottery subscription snapshot is missing")
			}
			var subscriptionPlan SubscriptionPlan
			if err := common.Unmarshal([]byte(result.SubscriptionSnapshot), &subscriptionPlan); err != nil {
				return err
			}
			if subscriptionPlan.Id == 0 {
				subscriptionPlan.Id = result.SubscriptionPlanId
			}
			if _, err := CreateEarnedUserSubscriptionFromPlanTx(tx, result.UserId, &subscriptionPlan, "lottery"); err != nil {
				return err
			}
		default:
			return errors.New("invalid lottery reward type")
		}
		return tx.Model(&LotteryResult{}).Where("id = ? AND fulfillment_status != ?", result.Id, "fulfilled").Updates(map[string]interface{}{
			"fulfillment_status": "fulfilled",
			"claimed_at":         common.GetTimestamp(),
		}).Error
	})
}

func selectWeightedLotteryParticipant(candidates []LotteryParticipant) (LotteryParticipant, error) {
	total := int64(0)
	for _, candidate := range candidates {
		if candidate.Weight <= 0 || candidate.Weight > 1000000 {
			return LotteryParticipant{}, errors.New("invalid lottery participant weight")
		}
		total += int64(candidate.Weight)
	}
	if total <= 0 {
		return LotteryParticipant{}, errors.New("lottery participant weight is empty")
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(total))
	if err != nil {
		return LotteryParticipant{}, err
	}
	remaining := value.Int64()
	for _, candidate := range candidates {
		if remaining < int64(candidate.Weight) {
			return candidate, nil
		}
		remaining -= int64(candidate.Weight)
	}
	return LotteryParticipant{}, errors.New("failed to choose lottery participant")
}
