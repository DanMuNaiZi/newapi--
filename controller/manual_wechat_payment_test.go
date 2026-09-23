package controller

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupManualWechatControllerTest(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldSettings := *operation_setting.GetPaymentSetting()
	oldPrice := operation_setting.Price
	oldMinTopUp := operation_setting.MinTopUp
	oldQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldQuotaPerUnit := common.QuotaPerUnit
	oldRedisEnabled := common.RedisEnabled
	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "manual-wechat.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.TopUp{},
		&model.SubscriptionPlan{},
		&model.SubscriptionOrder{},
		&model.UserSubscription{},
		&model.Option{},
		&model.Log{},
		&model.AuditLog{},
	))
	model.DB = db
	model.LOG_DB = db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	operation_setting.Price = 7.3
	operation_setting.MinTopUp = 1
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	common.QuotaPerUnit = 500_000
	common.RedisEnabled = false
	*operation_setting.GetPaymentSetting() = operation_setting.PaymentSetting{
		AmountOptions:            []int{5, 10},
		AmountDiscount:           map[int]float64{},
		ComplianceConfirmed:      true,
		ComplianceTermsVersion:   operation_setting.CurrentComplianceTermsVersion,
		ManualWechatEnabled:      true,
		ManualWechatQRCodeImage:  manualWechatControllerPNGDataURL(t, 256, 256),
		ManualWechatInstructions: service.DefaultManualWechatInstructions,
	}
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		*operation_setting.GetPaymentSetting() = oldSettings
		operation_setting.Price = oldPrice
		operation_setting.MinTopUp = oldMinTopUp
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldQuotaDisplayType
		common.QuotaPerUnit = oldQuotaPerUnit
		common.RedisEnabled = oldRedisEnabled
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	})
	return db
}

func manualWechatControllerPNGDataURL(t *testing.T, width int, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func manualWechatTestContext(method string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	return context, recorder
}

func TestUpdateManualWechatPaymentSettingsPersistsValidatedConfiguration(t *testing.T) {
	setupManualWechatControllerTest(t)
	imageData := manualWechatControllerPNGDataURL(t, 300, 300)
	expiresAt := common.GetTimestamp() + 3_600
	body := fmt.Sprintf(`{"enabled":true,"qr_code_image":%q,"expires_at":%d,"instructions":"填写订单号"}`, imageData, expiresAt)
	context, recorder := manualWechatTestContext(http.MethodPut, body)

	UpdateManualWechatPaymentSettings(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	settings := service.GetManualWechatPaymentSettings()
	assert.True(t, settings.Enabled)
	assert.Equal(t, imageData, settings.QRCodeImage)
	assert.Equal(t, expiresAt, settings.ExpiresAt)
	assert.Equal(t, "填写订单号", settings.Instructions)
}

func TestUpdateManualWechatPaymentSettingsClearingImageDisablesPayment(t *testing.T) {
	setupManualWechatControllerTest(t)
	context, recorder := manualWechatTestContext(
		http.MethodPut,
		`{"enabled":true,"qr_code_image":"","expires_at":0,"instructions":""}`,
	)

	UpdateManualWechatPaymentSettings(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	settings := service.GetManualWechatPaymentSettings()
	assert.False(t, settings.Enabled)
	assert.Empty(t, settings.QRCodeImage)
	assert.Equal(t, service.DefaultManualWechatInstructions, settings.Instructions)
}

func TestUpdateManualWechatPaymentSettingsDoesNotPartiallySaveInvalidImage(t *testing.T) {
	setupManualWechatControllerTest(t)
	before := service.GetManualWechatPaymentSettings()
	context, recorder := manualWechatTestContext(
		http.MethodPut,
		`{"enabled":true,"qr_code_image":"data:image/png;base64,bm90LWltYWdl","expires_at":0,"instructions":"changed"}`,
	)

	UpdateManualWechatPaymentSettings(context)

	assert.Contains(t, recorder.Body.String(), `"success":false`)
	after := service.GetManualWechatPaymentSettings()
	assert.Equal(t, before, after)
}

func TestGetManualWechatPaymentInfoDoesNotExposeQRCode(t *testing.T) {
	setupManualWechatControllerTest(t)
	context, recorder := manualWechatTestContext(http.MethodGet, "")

	GetManualWechatPaymentInfo(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "qr_code_image")
	assert.NotContains(t, recorder.Body.String(), "base64")
	assert.Contains(t, recorder.Body.String(), service.ManualWechatPaymentDisplayName)
}

func TestRequestManualWechatTopUpCreatesPendingOrderWithServerAmount(t *testing.T) {
	db := setupManualWechatControllerTest(t)
	user := &model.User{Id: 901, Username: "manual_topup", Status: common.UserStatusEnabled, Quota: 0, Group: "default"}
	require.NoError(t, db.Create(user).Error)
	context, recorder := manualWechatTestContext(http.MethodPost, `{"amount":5,"amount_cny":0.01}`)
	context.Set("id", user.Id)

	RequestManualWechatTopUp(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Kind      string  `json:"kind"`
			TradeNo   string  `json:"trade_no"`
			AmountCNY float64 `json:"amount_cny"`
			Currency  string  `json:"currency"`
			Status    string  `json:"status"`
			Purpose   struct {
				TopUpAmount int64 `json:"topup_amount"`
			} `json:"purpose"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, "topup", response.Data.Kind)
	assert.Equal(t, 36.5, response.Data.AmountCNY)
	assert.Equal(t, "CNY", response.Data.Currency)
	assert.Equal(t, common.TopUpStatusPending, response.Data.Status)
	assert.Equal(t, int64(5), response.Data.Purpose.TopUpAmount)
	assert.True(t, strings.HasPrefix(response.Data.TradeNo, "MWX-T-"))
	assert.Equal(t, 0, user.Quota)
}

func TestRequestManualWechatTopUpNormalizesTokenDisplayAmount(t *testing.T) {
	db := setupManualWechatControllerTest(t)
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeTokens
	user := &model.User{Id: 934, Username: "manual_topup_tokens", Status: common.UserStatusEnabled, Quota: 0, Group: "default"}
	require.NoError(t, db.Create(user).Error)
	context, recorder := manualWechatTestContext(http.MethodPost, `{"amount":1000000}`)
	context.Set("id", user.Id)

	RequestManualWechatTopUp(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TradeNo string `json:"trade_no"`
			Purpose struct {
				TopUpAmount int64 `json:"topup_amount"`
			} `json:"purpose"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, int64(1_000_000), response.Data.Purpose.TopUpAmount)

	order := model.GetTopUpByTradeNo(response.Data.TradeNo)
	require.NotNil(t, order)
	assert.Equal(t, int64(2), order.Amount)
	require.NoError(t, model.CompleteManualWechatTopUp(order.TradeNo, "127.0.0.1"))
	var refreshed model.User
	require.NoError(t, db.First(&refreshed, user.Id).Error)
	assert.Equal(t, 1_000_000, refreshed.Quota)
}

func TestRequestManualWechatTopUpRejectsUnavailablePaymentConfiguration(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*operation_setting.PaymentSetting)
	}{
		{
			name: "payment compliance not confirmed",
			mutate: func(settings *operation_setting.PaymentSetting) {
				settings.ComplianceConfirmed = false
			},
		},
		{
			name: "manual payment disabled",
			mutate: func(settings *operation_setting.PaymentSetting) {
				settings.ManualWechatEnabled = false
			},
		},
		{
			name: "missing image",
			mutate: func(settings *operation_setting.PaymentSetting) {
				settings.ManualWechatQRCodeImage = ""
			},
		},
		{
			name: "expired image",
			mutate: func(settings *operation_setting.PaymentSetting) {
				settings.ManualWechatExpiresAt = common.GetTimestamp() - 1
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			db := setupManualWechatControllerTest(t)
			user := &model.User{Id: 930, Username: "manual_unavailable", Status: common.UserStatusEnabled, Group: "default"}
			require.NoError(t, db.Create(user).Error)
			testCase.mutate(operation_setting.GetPaymentSetting())
			context, recorder := manualWechatTestContext(http.MethodPost, `{"amount":5}`)
			context.Set("id", user.Id)

			RequestManualWechatTopUp(context)

			assert.Contains(t, recorder.Body.String(), `"success":false`)
			var orderCount int64
			require.NoError(t, db.Model(&model.TopUp{}).Count(&orderCount).Error)
			assert.Zero(t, orderCount)
		})
	}
}

func TestRequestManualWechatSubscriptionCreatesPendingOrder(t *testing.T) {
	db := setupManualWechatControllerTest(t)
	user := &model.User{Id: 902, Username: "manual_subscription", Status: common.UserStatusEnabled, Quota: 0, Group: "default"}
	plan := &model.SubscriptionPlan{
		Id:            903,
		Title:         "Pro",
		PriceAmount:   29.9,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1_000,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(plan).Error)
	context, recorder := manualWechatTestContext(http.MethodPost, `{"plan_id":903}`)
	context.Set("id", user.Id)

	RequestManualWechatSubscription(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"amount_cny":29.9`)
	assert.Contains(t, recorder.Body.String(), `"currency":"CNY"`)
	assert.Contains(t, recorder.Body.String(), `"plan_title":"Pro"`)
	var orderCount int64
	require.NoError(t, db.Model(&model.SubscriptionOrder{}).Count(&orderCount).Error)
	assert.Equal(t, int64(1), orderCount)
	var subscriptionCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&subscriptionCount).Error)
	assert.Zero(t, subscriptionCount)
}

func setManualWechatAdminContext(context *gin.Context, adminId int) {
	context.Set("id", adminId)
	context.Set("role", common.RoleRootUser)
	context.Set("username", "manual_admin")
}

func TestAdminManualWechatTopUpCompleteAndExpire(t *testing.T) {
	db := setupManualWechatControllerTest(t)
	admin := &model.User{Id: 910, Username: "manual_admin", AffCode: "manual-admin-910", Role: common.RoleRootUser, Status: common.UserStatusEnabled}
	user := &model.User{Id: 911, Username: "manual_topup_target", AffCode: "manual-user-911", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(user).Error)
	completeOrder, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 2, 14.6, 10_000)
	require.NoError(t, err)
	operation_setting.GetPaymentSetting().ManualWechatEnabled = false

	context, recorder := manualWechatTestContext(http.MethodPost, fmt.Sprintf(`{"trade_no":%q}`, completeOrder.TradeNo))
	setManualWechatAdminContext(context, admin.Id)
	AdminCompleteManualWechatTopUp(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var refreshed model.User
	require.NoError(t, db.First(&refreshed, user.Id).Error)
	assert.Equal(t, 1_000_000, refreshed.Quota)

	expireOrder, _, err := model.CreateOrReuseManualWechatTopUp(user.Id, 3, 21.9, 10_061)
	require.NoError(t, err)
	context, recorder = manualWechatTestContext(http.MethodPost, fmt.Sprintf(`{"trade_no":%q}`, expireOrder.TradeNo))
	setManualWechatAdminContext(context, admin.Id)
	AdminExpireManualWechatTopUp(context)

	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Equal(t, common.TopUpStatusExpired, model.GetTopUpByTradeNo(expireOrder.TradeNo).Status)

	context, recorder = manualWechatTestContext(http.MethodPost, fmt.Sprintf(`{"trade_no":%q}`, expireOrder.TradeNo))
	setManualWechatAdminContext(context, admin.Id)
	AdminCompleteManualWechatTopUp(context)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
}

func TestAdminManualWechatSubscriptionCompleteAndExpire(t *testing.T) {
	db := setupManualWechatControllerTest(t)
	admin := &model.User{Id: 920, Username: "manual_admin", AffCode: "manual-admin-920", Role: common.RoleRootUser, Status: common.UserStatusEnabled}
	user := &model.User{Id: 921, Username: "manual_subscription_target", AffCode: "manual-user-921", Status: common.UserStatusEnabled}
	plan := &model.SubscriptionPlan{
		Id: 922, Title: "Manual Admin Pro", PriceAmount: 29.9, Currency: "CNY",
		DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, Enabled: true, TotalAmount: 1_000,
	}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(plan).Error)
	completeOrder, _, err := model.CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 20_000)
	require.NoError(t, err)
	operation_setting.GetPaymentSetting().ManualWechatExpiresAt = common.GetTimestamp() - 1

	context, recorder := manualWechatTestContext(http.MethodPost, "")
	context.Params = gin.Params{{Key: "trade_no", Value: completeOrder.TradeNo}}
	setManualWechatAdminContext(context, admin.Id)
	AdminCompleteManualWechatSubscription(context)

	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var subscriptionCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", user.Id).Count(&subscriptionCount).Error)
	assert.Equal(t, int64(1), subscriptionCount)

	expireOrder, _, err := model.CreateOrReuseManualWechatSubscriptionOrder(user.Id, plan.Id, plan.PriceAmount, 20_061)
	require.NoError(t, err)
	context, recorder = manualWechatTestContext(http.MethodPost, "")
	context.Params = gin.Params{{Key: "trade_no", Value: expireOrder.TradeNo}}
	setManualWechatAdminContext(context, admin.Id)
	AdminExpireManualWechatSubscription(context)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	context, recorder = manualWechatTestContext(http.MethodPost, "")
	context.Params = gin.Params{{Key: "trade_no", Value: expireOrder.TradeNo}}
	setManualWechatAdminContext(context, admin.Id)
	AdminCompleteManualWechatSubscription(context)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", user.Id).Count(&subscriptionCount).Error)
	assert.Equal(t, int64(1), subscriptionCount)
}
