package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type lotteryPlanRequest struct {
	Title                 string                       `json:"title"`
	Icon                  string                       `json:"icon"`
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

type lotteryPlanUpdateRequest struct {
	Title       *string `json:"title"`
	Icon        *string `json:"icon"`
	Description *string `json:"description"`
	DrawTime    *int64  `json:"draw_time"`
}

type lotteryParticipantUpdateRequest struct {
	UserId        int  `json:"user_id"`
	Weight        *int `json:"weight"`
	PresetPrizeId *int `json:"preset_prize_id"`
}

type lotteryNotificationReadRequest struct {
	Ids []int `json:"ids"`
}

const lotteryNotificationReadMaxIds = 50

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

func GetLotteryParticipantsForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	participants, err := model.ListLotteryParticipantsForUser(planId, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, participants)
}

func GetLotteryPlanResultsForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	results, err := model.ListLotteryResultsForUserPlan(planId, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, results)
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

func GetLotteryResultsForSelf(c *gin.Context) {
	results, err := model.ListLotterySelfResultsForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, results)
}

func GetLotteryResultsPageForSelf(c *gin.Context) {
	limit, err := lotterySelfHistoryLimit(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	beforeCreatedAt, beforeId, err := lotteryHistoryCursor(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page, err := model.ListLotterySelfResultsForUserPage(c.GetInt("id"), limit, beforeCreatedAt, beforeId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if page.HasMore && len(page.Items) > 0 {
		lastItem := page.Items[len(page.Items)-1]
		page.NextCursor = lotteryHistoryNextCursor(lastItem.CreatedAt, lastItem.Id)
	}
	common.ApiSuccess(c, page)
}

func GetClaimableLotteryResultsForSelf(c *gin.Context) {
	results, err := model.ListClaimableLotterySelfResultsForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, results)
}

func GetLotteryNotificationsForSelf(c *gin.Context) {
	notifications, err := model.ListLotteryNotificationsForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, notifications)
}

func GetLotteryNotificationsPageForSelf(c *gin.Context) {
	limit, err := lotterySelfHistoryLimit(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	beforeCreatedAt, beforeId, err := lotteryHistoryCursor(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page, err := model.ListLotteryNotificationsForUserPage(
		c.GetInt("id"),
		limit,
		c.Query("unread_only") == "true",
		beforeCreatedAt,
		beforeId,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if page.HasMore && len(page.Items) > 0 {
		lastItem := page.Items[len(page.Items)-1]
		page.NextCursor = lotteryHistoryNextCursor(lastItem.CreatedAt, lastItem.Id)
	}
	common.ApiSuccess(c, page)
}

func MarkLotteryNotificationsReadForSelf(c *gin.Context) {
	req := lotteryNotificationReadRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 || len(req.Ids) > lotteryNotificationReadMaxIds {
		common.ApiErrorMsg(c, "invalid lottery notification ids")
		return
	}
	seen := make(map[int]struct{}, len(req.Ids))
	notificationIds := make([]int, 0, len(req.Ids))
	for _, notificationId := range req.Ids {
		if notificationId <= 0 {
			common.ApiErrorMsg(c, "invalid lottery notification ids")
			return
		}
		if _, duplicate := seen[notificationId]; duplicate {
			continue
		}
		seen[notificationId] = struct{}{}
		notificationIds = append(notificationIds, notificationId)
	}
	if err := model.MarkLotteryNotificationsReadForUser(c.GetInt("id"), notificationIds); err != nil {
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
		Icon:                  req.Icon,
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

func AdminListLotteryPlans(c *gin.Context) {
	plans, err := model.ListLotteryPlansForAdmin()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

func AdminListLotteryPrizes(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prizes, err := model.ListLotteryPrizes(planId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, prizes)
}

func AdminListLotteryResults(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	results, err := model.ListLotteryResultsForPlan(planId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, results)
}

func AdminUpdateLotteryPlan(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	req := lotteryPlanUpdateRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	plan, err := model.UpdatePublishedLotteryPlan(planId, model.LotteryPlanPublishedUpdate{
		Title:       req.Title,
		Icon:        req.Icon,
		Description: req.Description,
		DrawTime:    req.DrawTime,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "lottery.plan_update", map[string]interface{}{
		"plan_id":   plan.Id,
		"draw_time": req.DrawTime,
	})
	common.ApiSuccess(c, plan)
}

func AdminCancelLotteryPlan(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CancelLotteryPlan(planId); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "lottery.plan_cancel", map[string]interface{}{
		"plan_id": planId,
	})
	common.ApiSuccess(c, nil)
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

func lotterySelfHistoryLimit(c *gin.Context) (int, error) {
	rawLimit := c.Query("limit")
	if rawLimit == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return 0, errors.New("invalid lottery history limit")
	}
	return limit, nil
}

func lotteryHistoryCursor(c *gin.Context) (int64, int, error) {
	rawCursor := strings.TrimSpace(c.Query("cursor"))
	if rawCursor == "" {
		return 0, 0, nil
	}
	parts := strings.Split(rawCursor, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid lottery history cursor")
	}
	createdAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || createdAt < 0 {
		return 0, 0, errors.New("invalid lottery history cursor")
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, 0, errors.New("invalid lottery history cursor")
	}
	return createdAt, id, nil
}

func lotteryHistoryNextCursor(createdAt int64, id int) string {
	return strconv.FormatInt(createdAt, 10) + ":" + strconv.Itoa(id)
}
