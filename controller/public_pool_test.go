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

func TestCreatePublicPoolContributionUsesAuthenticatedUserIdentity(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PublicPoolSite{}, &model.PublicPoolContribution{}))
	owner := &model.User{Username: "pool-controller-owner", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-controller-owner"}
	other := &model.User{Username: "pool-controller-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-controller-other"}
	require.NoError(t, db.Create([]*model.User{owner, other}).Error)
	site := &model.PublicPoolSite{Name: "Example", URL: "https://example.com", Status: model.PublicPoolSiteStatusEnabled}
	require.NoError(t, model.CreatePublicPoolSite(site))

	payload := []byte(fmt.Sprintf(`{"site_id":%d,"description":"registered","proof":"invitation completed","user_id":999,"api_key":"must-not-be-persisted"}`, site.Id))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/public-pool/contributions", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", owner.Id)

	CreatePublicPoolContributionForSelf(ctx)

	var response struct {
		Success bool                         `json:"success"`
		Data    model.PublicPoolContribution `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, owner.Id, response.Data.UserId)
	assert.NotEqual(t, other.Id, response.Data.UserId)
	assert.NotContains(t, recorder.Body.String(), "must-not-be-persisted")
}

func TestAdminReviewPublicPoolContributionCannotOverrideReviewer(t *testing.T) {
	db := setupLotteryControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PublicPoolSite{}, &model.PublicPoolContribution{}))
	site := &model.PublicPoolSite{Name: "Example", URL: "https://example.com", Status: model.PublicPoolSiteStatusEnabled}
	require.NoError(t, model.CreatePublicPoolSite(site))
	contribution := &model.PublicPoolContribution{UserId: 10, SiteId: site.Id, Description: "registered"}
	require.NoError(t, model.CreatePublicPoolContribution(contribution))

	payload := []byte(`{"status":"approved","review_note":"verified","reviewer_id":999}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/public-pool/admin/contributions/1/review", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(contribution.Id)}}
	ctx.Set("id", 88)

	AdminReviewPublicPoolContribution(ctx)

	var stored model.PublicPoolContribution
	require.NoError(t, db.First(&stored, contribution.Id).Error)
	assert.Equal(t, 88, stored.ReviewerId)
	assert.Equal(t, model.PublicPoolContributionStatusApproved, stored.Status)
}
