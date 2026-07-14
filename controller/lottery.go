package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type lotteryPlanRequest struct {
	Title                 string                       `json:"title"`
	Description           string                       `json:"description"`
	Status                model.LotteryPlanStatus      `json:"status"`
	EligibilityMode       model.LotteryEligibilityMode `json:"eligibility_mode"`
	MaxParticipants       int                          `json:"max_participants"`
	RegistrationStartTime int64                        `json:"registration_start_time"`
	DrawTime              int64                        `json:"draw_time"`
	UserIds               []int                        `json:"user_ids"`
	Groups                []string                     `json:"groups"`
	Prizes                []*model.LotteryPrize        `json:"prizes"`
}

type lotteryManualDrawRequest struct {
	Reason string `json:"reason"`
}

type lotteryParticipantUpdateRequest struct {
	UserId        int  `json:"user_id"`
	Weight        *int `json:"weight"`
	PresetPrizeId *int `json:"preset_prize_id"`
}

// GetLotteryPlansForSelf returns only plans the authenticated user may see.
// Eligibility remains enforced in model.ListLotteryPlansForUser so clients
// cannot reveal private plans by changing frontend filters.
func GetLotteryPlansForSelf(c *gin.Context) {
	plans, err := model.ListLotteryPlansForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

func JoinLotteryPlanForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.JoinLotteryPlan(planId, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func LeaveLotteryPlanForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.LeaveLotteryPlan(planId, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ClaimLotteryResultForSelf(c *gin.Context) {
	resultId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ClaimLotteryResult(resultId, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminCreateLotteryPlan(c *gin.Context) {
	req := lotteryPlanRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Status == "" {
		req.Status = model.LotteryPlanStatusDraft
	}
	if req.Status != model.LotteryPlanStatusDraft && req.Status != model.LotteryPlanStatusScheduled && req.Status != model.LotteryPlanStatusOpen {
		common.ApiErrorMsg(c, "invalid lottery plan status")
		return
	}
	plan := &model.LotteryPlan{
		Title:                 req.Title,
		Description:           req.Description,
		Status:                req.Status,
		EligibilityMode:       req.EligibilityMode,
		MaxParticipants:       req.MaxParticipants,
		RegistrationStartTime: req.RegistrationStartTime,
		DrawTime:              req.DrawTime,
		CreatedBy:             c.GetInt("id"),
	}
	if err := model.CreateLotteryPlan(plan, req.UserIds, req.Groups, req.Prizes); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "lottery.plan_create", map[string]interface{}{
		"plan_id":          plan.Id,
		"eligibility_mode": plan.EligibilityMode,
		"max_participants": plan.MaxParticipants,
	})
	common.ApiSuccess(c, plan)
}

func AdminDrawLotteryPlan(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	req := lotteryManualDrawRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	run, err := model.DrawLotteryPlan(planId, model.LotteryDrawTriggerManual, req.Reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "lottery.plan_draw", map[string]interface{}{
		"plan_id": planId,
		"reason":  req.Reason,
		"run_id":  run.Id,
	})
	common.ApiSuccess(c, run)
}

func AdminListLotteryParticipants(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	participants, err := model.ListLotteryParticipants(planId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, participants)
}

func AdminUpdateLotteryParticipant(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	req := lotteryParticipantUpdateRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId <= 0 || (req.Weight == nil && req.PresetPrizeId == nil) {
		common.ApiErrorMsg(c, "invalid lottery participant update")
		return
	}
	if req.Weight != nil {
		if err := model.SetLotteryParticipantWeight(planId, req.UserId, *req.Weight); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if req.PresetPrizeId != nil {
		if err := model.SetLotteryParticipantPreset(planId, req.UserId, *req.PresetPrizeId); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	recordManageAudit(c, "lottery.participant_update", map[string]interface{}{
		"plan_id":          planId,
		"participant_user": req.UserId,
		"weight":           req.Weight,
		"preset_prize_id":  req.PresetPrizeId,
	})
	common.ApiSuccess(c, nil)
}

func lotteryPathID(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid lottery id")
	}
	return id, nil
}
