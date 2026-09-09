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
	require.NoError(t, DB.AutoMigrate(&PublicPoolSite{}, &PublicPoolContribution{}, &RewardGrant{}, &Channel{}, &Ability{}))
	for _, table := range []interface{}{&RewardGrant{}, &PublicPoolContribution{}, &PublicPoolSite{}, &Ability{}, &Channel{}} {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("username LIKE ?", "pool-%").Delete(&User{}).Error)
	t.Cleanup(func() {
		for _, table := range []interface{}{&RewardGrant{}, &PublicPoolContribution{}, &PublicPoolSite{}, &Ability{}, &Channel{}} {
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
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	user := &User{Username: "pool-review-user", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-review-user"}
	require.NoError(t, DB.Create(user).Error)
	site := &PublicPoolSite{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled, RewardSnapshotJSON: rawReward}
	require.NoError(t, CreatePublicPoolSite(site))
	contribution := &PublicPoolContribution{UserId: user.Id, SiteId: site.Id, Description: "Registered", Proof: "Proof"}
	require.NoError(t, CreatePublicPoolContribution(contribution))

	reviewed, err := ReviewPublicPoolContribution(contribution.Id, 99, PublicPoolContributionStatusApproved, "Verified")
	require.NoError(t, err)
	assert.Equal(t, 99, reviewed.ReviewerId)
	assert.Equal(t, PublicPoolContributionStatusApproved, reviewed.Status)
	assert.Equal(t, "Verified", reviewed.ReviewNote)
	assert.NotZero(t, reviewed.ReviewedAt)
	assert.Equal(t, RewardGrantStatusSucceeded, reviewed.RewardStatus)
	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, 100, stored.Quota)
}

func TestPublicPoolApprovalKeepsFailedRewardVisibleAndRetryable(t *testing.T) {
	setupPublicPoolFixture(t)
	rawReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	user := &User{
		Username: "pool-full-balance", Password: "password", Status: common.UserStatusEnabled,
		AffCode: "pool-full-balance", Quota: common.MaxQuota,
	}
	require.NoError(t, DB.Create(user).Error)
	site := &PublicPoolSite{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled, RewardSnapshotJSON: rawReward}
	require.NoError(t, CreatePublicPoolSite(site))
	contribution := &PublicPoolContribution{UserId: user.Id, SiteId: site.Id, Description: "Registered", Proof: "Proof"}
	require.NoError(t, CreatePublicPoolContribution(contribution))

	reviewed, err := ReviewPublicPoolContribution(contribution.Id, 99, PublicPoolContributionStatusApproved, "Verified")
	require.NoError(t, err)
	assert.Equal(t, PublicPoolContributionStatusApproved, reviewed.Status)
	assert.Equal(t, RewardGrantStatusFailed, reviewed.RewardStatus)
	assert.NotEmpty(t, reviewed.RewardFailureReason)

	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, common.MaxQuota, stored.Quota)
}

func TestPublicPoolAdminContributionPageFiltersAndOrdersStably(t *testing.T) {
	setupPublicPoolFixture(t)
	users := []*User{
		{Username: "pool-page-alice", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-page-a"},
		{Username: "pool-page-bob", Password: "password", Status: common.UserStatusEnabled, AffCode: "pool-page-b"},
	}
	require.NoError(t, DB.Create(&users).Error)
	site := &PublicPoolSite{Name: "Example Search Site", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled}
	require.NoError(t, CreatePublicPoolSite(site))
	first := &PublicPoolContribution{UserId: users[0].Id, SiteId: site.Id, Description: "one", Status: PublicPoolContributionStatusPending, CreatedAt: 100, UpdatedAt: 100}
	second := &PublicPoolContribution{UserId: users[1].Id, SiteId: site.Id, Description: "two", Status: PublicPoolContributionStatusApproved, CreatedAt: 100, UpdatedAt: 100}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, DB.Create(second).Error)

	items, total, err := ListPublicPoolContributionsForAdminPage("BOB", PublicPoolContributionStatusApproved, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, second.Id, items[0].Id)

	items, total, err = ListPublicPoolContributionsForAdminPage("Example Search", "", 1, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, items, 1)
	assert.Equal(t, second.Id, items[0].Id)
}

func TestRejectedPublicPoolContributionCanBeResubmittedWithLatestFrozenReward(t *testing.T) {
	setupPublicPoolFixture(t)
	firstReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 100})
	require.NoError(t, err)
	site := &PublicPoolSite{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled, RewardSnapshotJSON: firstReward}
	require.NoError(t, CreatePublicPoolSite(site))
	contribution := &PublicPoolContribution{UserId: 15, SiteId: site.Id, Description: "Registered", Proof: "First proof"}
	require.NoError(t, CreatePublicPoolContribution(contribution))
	_, err = ReviewPublicPoolContribution(contribution.Id, 99, PublicPoolContributionStatusRejected, "Try again")
	require.NoError(t, err)

	secondReward, err := EncodeRewardSnapshot(RewardSnapshot{Type: RewardTypeQuota, Quota: 200})
	require.NoError(t, err)
	site.RewardSnapshotJSON = secondReward
	require.NoError(t, UpdatePublicPoolSite(site))
	resubmitted := &PublicPoolContribution{UserId: 15, SiteId: site.Id, Description: "Registered", Proof: "Updated proof"}
	require.NoError(t, CreatePublicPoolContribution(resubmitted))

	assert.Equal(t, contribution.Id, resubmitted.Id)
	snapshot, err := DecodeRewardSnapshot(resubmitted.RewardSnapshotJSON)
	require.NoError(t, err)
	assert.Equal(t, 200, snapshot.Quota)
}

func TestPublicPoolContributionCannotResubmitWhenAnOlderActiveRecordExists(t *testing.T) {
	setupPublicPoolFixture(t)
	site := &PublicPoolSite{Name: "Example", URL: "https://example.com", Status: PublicPoolSiteStatusEnabled}
	require.NoError(t, CreatePublicPoolSite(site))
	require.NoError(t, DB.Create(&PublicPoolContribution{
		UserId: 41, SiteId: site.Id, Description: "Approved", Proof: "Proof",
		Status: PublicPoolContributionStatusApproved, CreatedAt: 100, UpdatedAt: 100,
	}).Error)
	require.NoError(t, DB.Create(&PublicPoolContribution{
		UserId: 41, SiteId: site.Id, Description: "Rejected", Proof: "Proof",
		Status: PublicPoolContributionStatusRejected, CreatedAt: 200, UpdatedAt: 200,
	}).Error)

	err := CreatePublicPoolContribution(&PublicPoolContribution{
		UserId: 41, SiteId: site.Id, Description: "Retry", Proof: "Updated proof",
	})
	require.ErrorContains(t, err, "already exists")
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
