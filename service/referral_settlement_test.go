package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failedReferralSettlement struct{ relaycommon.BillingSettler }

func (failedReferralSettlement) GetPreConsumedQuota() int { return 0 }
func (failedReferralSettlement) Settle(int) error         { return errors.New("funding settlement failed") }

func TestFailedBillingSettlementCannotQualifyReferral(t *testing.T) {
	truncate(t)
	require.NoError(t, model.DB.AutoMigrate(&model.ReferralCampaign{}, &model.ReferralCampaignEvent{}, &model.ReferralCampaignScheduleLock{}))
	t.Cleanup(func() {
		for _, table := range []interface{}{&model.ReferralCampaignEvent{}, &model.ReferralCampaign{}, &model.ReferralCampaignScheduleLock{}} {
			require.NoError(t, model.DB.Where("1 = 1").Delete(table).Error)
		}
	})
	seedUser(t, 83, 100000)
	seedChannel(t, 83)
	require.NoError(t, model.DB.Create(&model.User{Id: 84, Username: "inviter", AffCode: "settlement-inviter", Status: common.UserStatusEnabled}).Error)
	now := common.GetTimestamp()
	raw, err := model.EncodeRewardSnapshot(model.RewardSnapshot{Type: model.RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	require.NoError(t, model.CreateReferralCampaign(&model.ReferralCampaign{Title: "manual review", Enabled: true, StartTime: now - 60, EndTime: now + 3600, RewardSnapshotJSON: raw}))
	_, err = model.HandleInvitationRegistration(84, 83)
	require.NoError(t, err)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{
		UserId: 83, UserQuota: 100000, RequestModelName: "client-model", OriginModelName: "client-model", UsingGroup: "default", IsPlayground: true,
		StartTime: time.Now().Add(-time.Second), FirstResponseTime: time.Now().Add(-time.Second),
		PriceData:   types.PriceData{ModelRatio: 1, CompletionRatio: 1, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}},
		ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 83}, Billing: failedReferralSettlement{},
	}
	PostTextConsumeQuota(ctx, info, &dto.Usage{PromptTokens: 10, CompletionTokens: 1}, nil)
	var event model.ReferralCampaignEvent
	require.NoError(t, model.DB.Where("invitee_user_id = ?", 83).First(&event).Error)
	assert.Zero(t, event.QualifiedQuota, "an estimated consume log is not proof of a successful debit")
	log := getLastLog(t)
	assert.Positive(t, log.Quota)
	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	assert.Equal(t, "client-model", other["request_model_name"], "unmapped requests also need an explicit client model for the safe review DTO")
}
