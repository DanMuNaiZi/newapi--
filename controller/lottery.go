package controller

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type lotteryRewardUnit string

const (
	lotteryRewardUnitUSD   lotteryRewardUnit = "usd"
	lotteryRewardUnitCNY   lotteryRewardUnit = "cny"
	lotteryRewardUnitQuota lotteryRewardUnit = "quota"
)

type lotteryPrizeRequest struct {
	Name               string                       `json:"name"`
	Quantity           int                          `json:"quantity"`
	RewardType         model.LotteryRewardType      `json:"reward_type"`
	Quota              *int                         `json:"quota"`
	RewardAmount       *decimal.Decimal             `json:"reward_amount"`
	RewardUnit         lotteryRewardUnit            `json:"reward_unit"`
	SubscriptionPlanId int                          `json:"subscription_plan_id"`
	FulfillmentMode    model.LotteryFulfillmentMode `json:"fulfillment_mode"`
	ClaimExpireSeconds int64                        `json:"claim_expire_seconds"`
}

type lotteryPrizeConversionAudit struct {
	InputAmount     string `json:"input_amount"`
	InputUnit       string `json:"input_unit"`
	Quota           int    `json:"quota"`
	QuotaPerUnit    string `json:"quota_per_unit"`
	USDExchangeRate string `json:"usd_exchange_rate"`
}

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
	Prizes                []lotteryPrizeRequest        `json:"prizes"`
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

func GetLotteryPlanForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	plan, err := model.GetLotteryPlanForUser(planId, c.GetInt("id"))
	if err != nil {
		lotteryUserAPIError(c, 0, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func GetLotteryParticipantsForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	participants, err := model.ListLotteryParticipantsForUser(planId, c.GetInt("id"))
	if err != nil {
		lotteryUserAPIError(c, 0, err)
		return
	}
	common.ApiSuccess(c, participants)
}

func GetLotteryParticipantsPageForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	limit, err := lotterySelfHistoryLimit(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	beforeCreatedAt, beforeId, err := lotteryHistoryCursor(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	page, err := model.ListLotteryParticipantsForUserPage(planId, c.GetInt("id"), limit, beforeCreatedAt, beforeId)
	if err != nil {
		lotteryUserAPIError(c, 0, err)
		return
	}
	if page.HasMore && len(page.Items) > 0 {
		lastItem := page.Items[len(page.Items)-1]
		page.NextCursor = lotteryHistoryNextCursor(lastItem.JoinedAt, lastItem.Id)
	}
	common.ApiSuccess(c, page)
}

func GetLotteryPlanResultsForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	results, err := model.ListLotteryResultsForUserPlan(planId, c.GetInt("id"))
	if err != nil {
		lotteryUserAPIError(c, 0, err)
		return
	}
	common.ApiSuccess(c, results)
}

func GetLotteryPlanResultsPageForSelf(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	limit, err := lotterySelfHistoryLimit(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	beforeCreatedAt, beforeId, err := lotteryHistoryCursor(c)
	if err != nil {
		lotteryUserAPIError(c, http.StatusBadRequest, err)
		return
	}
	page, err := model.ListLotteryResultsForUserPlanPage(planId, c.GetInt("id"), limit, beforeCreatedAt, beforeId)
	if err != nil {
		lotteryUserAPIError(c, 0, err)
		return
	}
	if page.HasMore && len(page.Items) > 0 {
		lastItem := page.Items[len(page.Items)-1]
		page.NextCursor = lotteryHistoryNextCursor(lastItem.CreatedAt, lastItem.Id)
	}
	common.ApiSuccess(c, page)
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
	redactLotteryPreviewCodes(c, results)
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
	redactLotteryPreviewCodes(c, page.Items)
	common.ApiSuccess(c, page)
}

func GetClaimableLotteryResultsForSelf(c *gin.Context) {
	results, err := model.ListClaimableLotterySelfResultsForUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redactLotteryPreviewCodes(c, results)
	common.ApiSuccess(c, results)
}

func redactLotteryPreviewCodes(c *gin.Context, results []model.LotterySelfResultView) {
	if !c.GetBool("preview_mode") {
		return
	}
	for index := range results {
		results[index].RedemptionCode = ""
	}
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
	prizes := make([]*model.LotteryPrize, 0, len(req.Prizes))
	prizeAudits := make([]lotteryPrizeConversionAudit, 0, len(req.Prizes))
	for _, prizeRequest := range req.Prizes {
		prize, audit, err := normalizeLotteryPrizeRequest(prizeRequest)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		prizes = append(prizes, prize)
		if prize.RewardType == model.LotteryRewardQuota {
			prizeAudits = append(prizeAudits, audit)
		}
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
	if err := model.CreateLotteryPlan(plan, req.UserIds, req.Groups, prizes); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "lottery.plan_create", map[string]interface{}{
		"plan_id":          plan.Id,
		"eligibility_mode": plan.EligibilityMode,
		"max_participants": plan.MaxParticipants,
		"quota_prizes":     prizeAudits,
	})
	common.ApiSuccess(c, plan)
}

func normalizeLotteryPrizeRequest(request lotteryPrizeRequest) (*model.LotteryPrize, lotteryPrizeConversionAudit, error) {
	prize := &model.LotteryPrize{
		Name:               request.Name,
		Quantity:           request.Quantity,
		RewardType:         request.RewardType,
		SubscriptionPlanId: request.SubscriptionPlanId,
		FulfillmentMode:    request.FulfillmentMode,
		ClaimExpireSeconds: request.ClaimExpireSeconds,
	}
	if request.RewardType != model.LotteryRewardQuota {
		return prize, lotteryPrizeConversionAudit{}, nil
	}

	hasAmount := request.RewardAmount != nil
	hasUnit := request.RewardUnit != ""
	if hasAmount != hasUnit {
		return nil, lotteryPrizeConversionAudit{}, errors.New("lottery reward amount and unit must be provided together")
	}
	if hasAmount && request.Quota != nil {
		return nil, lotteryPrizeConversionAudit{}, errors.New("lottery reward amount conflicts with legacy quota")
	}

	audit := lotteryPrizeConversionAudit{
		QuotaPerUnit:    strconv.FormatFloat(common.QuotaPerUnit, 'f', -1, 64),
		USDExchangeRate: strconv.FormatFloat(operation_setting.USDExchangeRate, 'f', -1, 64),
	}
	if !hasAmount {
		if request.Quota == nil || *request.Quota <= 0 || *request.Quota > common.MaxQuota {
			return nil, lotteryPrizeConversionAudit{}, errors.New("quota prize must be positive and within the database limit")
		}
		prize.Quota = *request.Quota
		audit.InputAmount = strconv.Itoa(*request.Quota)
		audit.InputUnit = string(lotteryRewardUnitQuota)
		audit.Quota = *request.Quota
		return prize, audit, nil
	}

	if request.RewardAmount.LessThanOrEqual(decimal.Zero) {
		return nil, lotteryPrizeConversionAudit{}, errors.New("lottery reward amount must be positive")
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return nil, lotteryPrizeConversionAudit{}, errors.New("invalid quota per unit configuration")
	}
	amount := *request.RewardAmount
	quotaDecimal := decimal.Zero
	switch request.RewardUnit {
	case lotteryRewardUnitUSD:
		quotaDecimal = amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	case lotteryRewardUnitCNY:
		if operation_setting.USDExchangeRate <= 0 || math.IsNaN(operation_setting.USDExchangeRate) || math.IsInf(operation_setting.USDExchangeRate, 0) {
			return nil, lotteryPrizeConversionAudit{}, errors.New("invalid USD exchange rate configuration")
		}
		quotaDecimal = amount.Div(decimal.NewFromFloat(operation_setting.USDExchangeRate)).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	case lotteryRewardUnitQuota:
		if !amount.Equal(amount.Truncate(0)) {
			return nil, lotteryPrizeConversionAudit{}, errors.New("raw lottery quota must be an integer")
		}
		quotaDecimal = amount
	default:
		return nil, lotteryPrizeConversionAudit{}, errors.New("invalid lottery reward unit")
	}

	roundedQuota := quotaDecimal.Round(0)
	if roundedQuota.LessThan(decimal.NewFromInt(1)) || roundedQuota.GreaterThan(decimal.NewFromInt(common.MaxQuota)) {
		return nil, lotteryPrizeConversionAudit{}, errors.New("lottery reward converts outside the supported quota range")
	}
	prize.Quota = common.QuotaFromDecimal(roundedQuota)
	audit.InputAmount = amount.String()
	audit.InputUnit = string(request.RewardUnit)
	audit.Quota = prize.Quota
	return prize, audit, nil
}

func AdminListLotteryPlans(c *gin.Context) {
	plans, err := model.ListLotteryPlansForAdmin()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

func AdminGetLotteryPlan(c *gin.Context) {
	planId, err := lotteryPathID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	plan, err := model.GetLotteryPlanForAdmin(planId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success":    false,
				"message":    "lottery plan not found",
				"request_id": c.GetString(common.RequestIdKey),
			})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
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

func lotteryUserAPIError(c *gin.Context, status int, err error) {
	if status == 0 {
		switch {
		case errors.Is(err, model.ErrLotteryPlanForbidden):
			status = http.StatusForbidden
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		default:
			status = http.StatusInternalServerError
		}
	}
	message := err.Error()
	if status == http.StatusInternalServerError {
		logger.LogError(c, "lottery user API error: "+err.Error())
		message = "internal server error"
	}
	c.JSON(status, gin.H{
		"success":    false,
		"message":    message,
		"request_id": c.GetString(common.RequestIdKey),
	})
}
