package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const manualWechatSettingsRequestMaxBytes int64 = 1024 * 1024

const (
	manualWechatEnabledOptionKey      = "payment_setting.manual_wechat_enabled"
	manualWechatQRCodeOptionKey       = "payment_setting.manual_wechat_qr_code_image"
	manualWechatExpiresAtOptionKey    = "payment_setting.manual_wechat_expires_at"
	manualWechatInstructionsOptionKey = "payment_setting.manual_wechat_instructions"
)

type manualWechatTopUpRequest struct {
	Amount int64 `json:"amount"`
}

type manualWechatSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

type manualWechatAdminOrderRequest struct {
	TradeNo string `json:"trade_no"`
}

type manualWechatOrderResponse struct {
	Kind            string         `json:"kind"`
	TradeNo         string         `json:"trade_no"`
	UserId          int            `json:"user_id"`
	AmountCNY       float64        `json:"amount_cny"`
	Currency        string         `json:"currency"`
	Status          string         `json:"status"`
	CreatedAt       int64          `json:"created_at"`
	Reused          bool           `json:"reused"`
	Purpose         map[string]any `json:"purpose"`
	QRCodeImage     string         `json:"qr_code_image"`
	QRCodeExpiresAt int64          `json:"qr_code_expires_at"`
	Instructions    string         `json:"instructions"`
}

func getAdminPaymentPageInfo(c *gin.Context) (*common.PageInfo, error) {
	pageInfo := common.GetPageQuery(c)
	if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
		page, err := strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			return nil, errors.New("分页参数错误")
		}
		pageInfo.Page = page
	}
	if rawPageSize := strings.TrimSpace(c.Query("page_size")); rawPageSize != "" {
		pageSize, err := strconv.Atoi(rawPageSize)
		if err != nil || pageSize < 1 {
			return nil, errors.New("分页参数错误")
		}
		if pageSize > 100 {
			pageSize = 100
		}
		pageInfo.PageSize = pageSize
	}
	if pageInfo.Page < 1 || pageInfo.PageSize < 1 {
		return nil, errors.New("分页参数错误")
	}
	return pageInfo, nil
}

func GetManualWechatPaymentSettings(c *gin.Context) {
	common.ApiSuccess(c, service.GetManualWechatPaymentSettings())
}

func UpdateManualWechatPaymentSettings(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, manualWechatSettingsRequestMaxBytes)
	var settings service.ManualWechatPaymentSettings
	if err := common.DecodeJson(c.Request.Body, &settings); err != nil {
		common.ApiErrorMsg(c, "参数错误或二维码图片过大")
		return
	}
	settings.QRCodeImage = strings.TrimSpace(settings.QRCodeImage)
	settings.Instructions = strings.TrimSpace(settings.Instructions)
	if settings.QRCodeImage == "" {
		settings.Enabled = false
	}
	if settings.Instructions == "" {
		settings.Instructions = service.DefaultManualWechatInstructions
	}
	if err := service.ValidateManualWechatPaymentSettings(settings, common.GetTimestamp()); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.UpdateOptionsBulk(map[string]string{
		manualWechatEnabledOptionKey:      strconv.FormatBool(settings.Enabled),
		manualWechatQRCodeOptionKey:       settings.QRCodeImage,
		manualWechatExpiresAtOptionKey:    strconv.FormatInt(settings.ExpiresAt, 10),
		manualWechatInstructionsOptionKey: settings.Instructions,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service.GetManualWechatPaymentSettings())
}

func GetManualWechatPaymentInfo(c *gin.Context) {
	settings := service.GetManualWechatPaymentSettings()
	available := operation_setting.IsPaymentComplianceConfirmed() &&
		service.IsManualWechatPaymentAvailable(settings, common.GetTimestamp())
	common.ApiSuccess(c, gin.H{
		"enabled":      available,
		"display_name": service.ManualWechatPaymentDisplayName,
		"expires_at":   settings.ExpiresAt,
	})
}

func requireManualWechatPayment(c *gin.Context) (service.ManualWechatPaymentSettings, bool) {
	if !requirePaymentCompliance(c) {
		return service.ManualWechatPaymentSettings{}, false
	}
	settings := service.GetManualWechatPaymentSettings()
	if !service.IsManualWechatPaymentAvailable(settings, common.GetTimestamp()) {
		common.ApiErrorMsg(c, "微信人工收款当前不可用")
		return service.ManualWechatPaymentSettings{}, false
	}
	return settings, true
}

func RequestManualWechatTopUp(c *gin.Context) {
	settings, ok := requireManualWechatPayment(c)
	if !ok {
		return
	}
	var req manualWechatTopUpRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, "充值数量不合法")
		return
	}
	userId := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userId, req.Amount) {
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney := getPayMoney(req.Amount, user.Group)
	if payMoney <= 0.01 {
		common.ApiErrorMsg(c, "充值金额过低")
		return
	}
	storedAmount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		if common.QuotaPerUnit <= 0 {
			common.ApiErrorMsg(c, "充值配置无效")
			return
		}
		storedAmount = decimal.NewFromInt(req.Amount).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).
			IntPart()
		if storedAmount <= 0 {
			common.ApiErrorMsg(c, "充值数量不合法")
			return
		}
	}
	now := common.GetTimestamp()
	order, reused, err := model.CreateOrReuseManualWechatTopUp(userId, storedAmount, payMoney, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, manualWechatOrderResponse{
		Kind:            "topup",
		TradeNo:         order.TradeNo,
		UserId:          order.UserId,
		AmountCNY:       order.Money,
		Currency:        "CNY",
		Status:          order.Status,
		CreatedAt:       order.CreateTime,
		Reused:          reused,
		Purpose:         map[string]any{"topup_amount": req.Amount},
		QRCodeImage:     settings.QRCodeImage,
		QRCodeExpiresAt: settings.ExpiresAt,
		Instructions:    settings.Instructions,
	})
}

func RequestManualWechatSubscription(c *gin.Context) {
	settings, ok := requireManualWechatPayment(c)
	if !ok {
		return
	}
	var req manualWechatSubscriptionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}
	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}
	order, reused, err := model.CreateOrReuseManualWechatSubscriptionOrder(
		userId,
		plan.Id,
		plan.PriceAmount,
		common.GetTimestamp(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, manualWechatOrderResponse{
		Kind:      "subscription",
		TradeNo:   order.TradeNo,
		UserId:    order.UserId,
		AmountCNY: order.Money,
		Currency:  "CNY",
		Status:    order.Status,
		CreatedAt: order.CreateTime,
		Reused:    reused,
		Purpose: map[string]any{
			"plan_id":    plan.Id,
			"plan_title": plan.Title,
		},
		QRCodeImage:     settings.QRCodeImage,
		QRCodeExpiresAt: settings.ExpiresAt,
		Instructions:    settings.Instructions,
	})
}

func AdminListSubscriptionOrders(c *gin.Context) {
	pageInfo, err := getAdminPaymentPageInfo(c)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	userId := 0
	if rawUserId := strings.TrimSpace(c.Query("user_id")); rawUserId != "" {
		parsed, err := strconv.Atoi(rawUserId)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "用户 ID 不合法")
			return
		}
		userId = parsed
	}
	planId := 0
	if rawPlanId := strings.TrimSpace(c.Query("plan_id")); rawPlanId != "" {
		parsed, err := strconv.Atoi(rawPlanId)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "套餐 ID 不合法")
			return
		}
		planId = parsed
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != common.TopUpStatusPending && status != common.TopUpStatusSuccess && status != common.TopUpStatusExpired {
		common.ApiErrorMsg(c, "订单状态不合法")
		return
	}

	orders, total, err := model.GetAdminSubscriptionOrders(pageInfo, model.AdminSubscriptionOrderFilter{
		PaymentProvider: strings.TrimSpace(c.Query("payment_provider")),
		Status:          status,
		UserId:          userId,
		PlanId:          planId,
		Keyword:         c.Query("keyword"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

func recordManualPaymentOrderAudit(c *gin.Context, action string, kind string, tradeNo string, userId int, amount float64, planId int, operationErr error) {
	params := map[string]any{
		"trade_no":   tradeNo,
		"order_type": kind,
		"amount_cny": amount,
		"success":    operationErr == nil,
	}
	if userId > 0 {
		params["target_user_id"] = userId
	}
	if planId > 0 {
		params["plan_id"] = planId
	}
	if operationErr != nil {
		params["error"] = operationErr.Error()
	}
	recordManageAuditFor(c, userId, action, params)
}

func AdminCompleteManualWechatTopUp(c *gin.Context) {
	var req manualWechatAdminOrderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.TradeNo = strings.TrimSpace(req.TradeNo)
	order := model.GetTopUpByTradeNo(req.TradeNo)

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)
	err := model.CompleteManualWechatTopUp(req.TradeNo, c.ClientIP())
	if order == nil {
		order = &model.TopUp{TradeNo: req.TradeNo}
	}
	recordManualPaymentOrderAudit(c, "manual_payment.topup_complete", "topup", req.TradeNo, order.UserId, order.Money, 0, err)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminExpireManualWechatTopUp(c *gin.Context) {
	var req manualWechatAdminOrderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.TradeNo = strings.TrimSpace(req.TradeNo)
	order := model.GetTopUpByTradeNo(req.TradeNo)

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)
	err := model.UpdatePendingTopUpStatus(req.TradeNo, model.PaymentProviderManualWechat, common.TopUpStatusExpired)
	if order == nil {
		order = &model.TopUp{TradeNo: req.TradeNo}
	}
	recordManualPaymentOrderAudit(c, "manual_payment.topup_expire", "topup", req.TradeNo, order.UserId, order.Money, 0, err)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminCompleteManualWechatSubscription(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	err := model.CompleteSubscriptionOrder(tradeNo, "", model.PaymentProviderManualWechat, model.PaymentMethodManualWechat)
	if order == nil {
		order = &model.SubscriptionOrder{TradeNo: tradeNo}
	}
	recordManualPaymentOrderAudit(c, "manual_payment.subscription_complete", "subscription", tradeNo, order.UserId, order.Money, order.PlanId, err)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminExpireManualWechatSubscription(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	err := model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderManualWechat)
	if order == nil {
		order = &model.SubscriptionOrder{TradeNo: tradeNo}
	}
	recordManualPaymentOrderAudit(c, "manual_payment.subscription_expire", "subscription", tradeNo, order.UserId, order.Money, order.PlanId, err)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
