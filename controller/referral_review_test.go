package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferralCampaignDualUSDRewardContract(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ReferralCampaign{}, &model.ReferralCampaignScheduleLock{}))
	previous := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = previous })
	now := common.GetTimestamp()
	for _, tc := range []struct {
		inviter, invitee           string
		valid                      bool
		inviterQuota, inviteeQuota int
	}{
		{"10", "1.000001", true, 5_000_000, 500_001},
		{"0", "1", true, 0, 500_000},
		{"1", "0", true, 500_000, 0},
		{"-1", "1", false, 0, 0},
		{"1", "0.00000001", false, 0, 0},
		{"9999999999", "1", false, 0, 0},
		{"1e2147483647", "1", false, 0, 0},
	} {
		t.Run(tc.inviter+"/"+tc.invitee, func(t *testing.T) {
			body := fmt.Sprintf(`{"title":"Dual USD", "start_time":%d,"end_time":%d,"inviter_reward_usd":%q,"invitee_reward_usd":%q}`, now-60, now+3600, tc.inviter, tc.invitee)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set("id", 1)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/referral-campaign/admin/campaigns", bytes.NewBufferString(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			AdminCreateReferralCampaign(ctx)
			var response struct {
				Success bool
				Data    model.ReferralCampaign
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, tc.valid, response.Success, recorder.Body.String())
			if !tc.valid {
				return
			}
			if tc.inviterQuota == 0 {
				assert.Nil(t, response.Data.Reward)
			} else {
				require.NotNil(t, response.Data.Reward)
				assert.Equal(t, tc.inviterQuota, response.Data.Reward.Quota)
			}
			if tc.inviteeQuota == 0 {
				assert.Nil(t, response.Data.InviteeReward)
			} else {
				require.NotNil(t, response.Data.InviteeReward)
				assert.Equal(t, tc.inviteeQuota, response.Data.InviteeReward.Quota)
			}
		})
	}
}
