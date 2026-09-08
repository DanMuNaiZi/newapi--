package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPublicPoolFixture(t *testing.T) {
	t.Helper()
	setupLotteryFixture(t)
	require.NoError(t, DB.AutoMigrate(&PublicPoolSite{}, &PublicPoolContribution{}, &Channel{}, &Ability{}))
	for _, table := range []interface{}{&PublicPoolContribution{}, &PublicPoolSite{}, &Ability{}, &Channel{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "pool-%").Delete(&User{}).Error)
	t.Cleanup(func() {
		for _, table := range []interface{}{&PublicPoolContribution{}, &PublicPoolSite{}, &Ability{}, &Channel{}} {
			require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
		}
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "pool-%").Delete(&User{}).Error)
	})
}

func TestPublicPoolContributionsStayScopedToTheirOwner(t *testing.T) {
	setupPublicPoolFixture(t)
	owner := &User{Username: "pool-owner", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-owner"}
	other := &User{Username: "pool-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-other"}
	require.NoError(t, DB.Create([]*User{owner, other}).Error)
	site := &PublicPoolSite{Name: "Example公益站", URL: "https://example.com", Description: "Free quota", Status: PublicPoolSiteStatusEnabled, SortOrder: 10}
	require.NoError(t, CreatePublicPoolSite(site))

	ownerContribution := &PublicPoolContribution{UserId: owner.Id, SiteId: site.Id, Description: "Registered", Proof: "Invitation completed"}
	otherContribution := &PublicPoolContribution{UserId: other.Id, SiteId: site.Id, Description: "Also registered", Proof: "Invitation completed"}
	require.NoError(t, CreatePublicPoolContribution(ownerContribution))
	require.NoError(t, CreatePublicPoolContribution(otherContribution))

	items, err := ListPublicPoolContributionsForUser(owner.Id)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, ownerContribution.Id, items[0].Id)
	assert.Equal(t, owner.Username, items[0].Username)
	assert.Equal(t, site.Name, items[0].SiteName)
	assert.Equal(t, PublicPoolContributionStatusPending, items[0].Status)
	assert.NotEqual(t, otherContribution.Id, items[0].Id)
}

func TestReviewPublicPoolContributionRecordsReviewerAndDecision(t *testing.T) {
	setupPublicPoolFixture(t)
	site := &PublicPoolSite{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled}
	require.NoError(t, CreatePublicPoolSite(site))
	contribution := &PublicPoolContribution{UserId: 12, SiteId: site.Id, Description: "Registered", Proof: "Proof"}
	require.NoError(t, CreatePublicPoolContribution(contribution))

	reviewed, err := ReviewPublicPoolContribution(contribution.Id, 99, PublicPoolContributionStatusApproved, "Verified")
	require.NoError(t, err)
	assert.Equal(t, 99, reviewed.ReviewerId)
	assert.Equal(t, PublicPoolContributionStatusApproved, reviewed.Status)
	assert.Equal(t, "Verified", reviewed.ReviewNote)
	assert.NotZero(t, reviewed.ReviewedAt)
}

func TestCountAvailablePublicPoolChannelsIgnoresDisabledChannels(t *testing.T) {
	setupPublicPoolFixture(t)
	previousMemoryCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCache })
	enabled := &Channel{Name: "public-enabled", Key: "key", Status: common.ChannelStatusEnabled}
	disabled := &Channel{Name: "public-disabled", Key: "key", Status: common.ChannelStatusManuallyDisabled}
	require.NoError(t, DB.Create([]*Channel{enabled, disabled}).Error)
	require.NoError(t, DB.Create([]Ability{
		{Group: "public_pool", Model: "model-a", ChannelId: enabled.Id, Enabled: true},
		{Group: "public_pool", Model: "model-a", ChannelId: disabled.Id, Enabled: true},
	}).Error)

	count, err := CountAvailablePublicPoolChannels()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.True(t, IsChannelEnabledForGroupModel("public_pool", "model-a", enabled.Id))
}

func TestNormalizePublicPoolContributionRejectsCredentialMaterial(t *testing.T) {
	tests := []struct {
		name        string
		description string
		proof       string
	}{
		{name: "password assignment", description: "registration complete", proof: "password=hunter2"},
		{name: "JSON api key", description: `{"api_key":"secret-value"}`},
		{name: "bearer token", proof: "Authorization: Bearer secret-token"},
		{name: "cookie header", proof: "Cookie: session=secret"},
		{name: "unlabelled opaque token", proof: "proof Ab3dEf6hJk9mNp2qRs5vWx8zYt4u"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := normalizePublicPoolContribution(&PublicPoolContribution{
				UserId:      1,
				SiteId:      1,
				Description: testCase.description,
				Proof:       testCase.proof,
			})
			require.ErrorContains(t, err, "must not contain credentials")
		})
	}
}

func TestNormalizePublicPoolContributionAllowsRegistrationDescriptions(t *testing.T) {
	err := normalizePublicPoolContribution(&PublicPoolContribution{
		UserId:      1,
		SiteId:      1,
		Description: "Registered through the invitation page",
		Proof:       "The public account page now shows the bonus quota",
	})
	require.NoError(t, err)
}

func TestNormalizePublicPoolSiteRejectsOversizedFields(t *testing.T) {
	tests := []PublicPoolSite{
		{Name: "Example", URL: "https://example.com", Description: strings.Repeat("a", 4001), Status: PublicPoolSiteStatusEnabled},
		{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled, SortOrder: publicPoolSortOrderMax + 1},
	}

	for _, site := range tests {
		assert.Error(t, normalizePublicPoolSite(&site))
	}
}
