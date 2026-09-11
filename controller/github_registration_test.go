package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupGitHubRegistrationControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.GitHubRegistrationWhitelist{},
		&model.Log{},
	))

	originalRegisterEnabled := common.RegisterEnabled
	originalMinAgeDays := common.GitHubRegistrationMinAgeDays
	originalQuotaForNewUser := common.QuotaForNewUser
	common.RegisterEnabled = true
	common.GitHubRegistrationMinAgeDays = 180
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.RegisterEnabled = originalRegisterEnabled
		common.GitHubRegistrationMinAgeDays = originalMinAgeDays
		common.QuotaForNewUser = originalQuotaForNewUser
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestEvaluateGitHubRegistrationAgePolicy(t *testing.T) {
	setupGitHubRegistrationControllerTestDB(t)
	now := time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC)

	t.Run("default 180 days accepts exact boundary", func(t *testing.T) {
		decision, err := evaluateGitHubRegistration(&oauth.OAuthUser{
			ProviderUserID: "1001",
			Username:       "boundary-user",
			CreatedAt:      now.Add(-180 * 24 * time.Hour).Unix(),
		}, now)

		require.NoError(t, err)
		assert.False(t, decision.AgeExempt)
		assert.Equal(t, now.Add(-180*24*time.Hour).Unix(), decision.CreatedAt)
	})

	t.Run("default 180 days rejects younger account", func(t *testing.T) {
		_, err := evaluateGitHubRegistration(&oauth.OAuthUser{
			ProviderUserID: "1002",
			Username:       "young-user",
			CreatedAt:      now.Add(-180*24*time.Hour + time.Second).Unix(),
		}, now)

		require.Error(t, err)
		assert.ErrorContains(t, err, "180")
	})

	t.Run("zero disables age limit but still requires verified date", func(t *testing.T) {
		common.GitHubRegistrationMinAgeDays = 0
		t.Cleanup(func() { common.GitHubRegistrationMinAgeDays = 180 })

		_, err := evaluateGitHubRegistration(&oauth.OAuthUser{
			ProviderUserID: "1003",
			Username:       "no-limit-user",
			CreatedAt:      now.Unix(),
		}, now)
		require.NoError(t, err)

		_, err = evaluateGitHubRegistration(&oauth.OAuthUser{
			ProviderUserID: "1004",
			Username:       "missing-date-user",
		}, now)
		require.Error(t, err)
		assert.ErrorContains(t, err, "created_at")
	})
}

func TestGitHubRegistrationAgeOptionUsesOAuthPermissions(t *testing.T) {
	assert.Equal(t, authz.OAuthRead, optionPermission("GitHubRegistrationMinAgeDays", authz.ActionRead))
	assert.Equal(t, authz.OAuthWrite, optionPermission("GitHubRegistrationMinAgeDays", authz.ActionWrite))
}

func TestEvaluateGitHubRegistrationWhitelistMatchesNumericID(t *testing.T) {
	db := setupGitHubRegistrationControllerTestDB(t)
	now := time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&model.GitHubRegistrationWhitelist{
		GitHubId:        "2001",
		GitHubLogin:     "original-login",
		GitHubCreatedAt: now.Add(-30 * 24 * time.Hour).Unix(),
		Remark:          "approved exception",
		CreatedBy:       9,
		UpdatedBy:       9,
	}).Error)

	_, err := evaluateGitHubRegistration(&oauth.OAuthUser{
		ProviderUserID: "9999",
		Username:       "original-login",
		CreatedAt:      now.Add(-30 * 24 * time.Hour).Unix(),
	}, now)
	require.Error(t, err, "matching a whitelisted username must not bypass the age gate")

	decision, err := evaluateGitHubRegistration(&oauth.OAuthUser{
		ProviderUserID: "2001",
		Username:       "renamed-login",
	}, now)
	require.NoError(t, err)
	assert.True(t, decision.AgeExempt)
	assert.Zero(t, decision.CreatedAt, "a stable-ID whitelist may bypass missing creation date")
}

func TestFindOrCreateOAuthUserKeepsExistingNonGitHubLoginAndRejectsNewOne(t *testing.T) {
	db := setupGitHubRegistrationControllerTestDB(t)
	existing := &model.User{
		Username:  "existing-discord-user",
		DiscordId: "discord-100",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(existing).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("github-registration-test"))))
	router.GET("/existing", func(c *gin.Context) {
		user, err := findOrCreateOAuthUser(c, &oauth.DiscordProvider{}, &oauth.OAuthUser{
			ProviderUserID: "discord-100",
			Username:       "existing-discord-user",
		}, sessions.Default(c))
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.String(http.StatusOK, "%d", user.Id)
	})
	router.GET("/new", func(c *gin.Context) {
		_, err := findOrCreateOAuthUser(c, &oauth.DiscordProvider{}, &oauth.OAuthUser{
			ProviderUserID: "discord-200",
			Username:       "new-discord-user",
		}, sessions.Default(c))
		if err == nil {
			c.Status(http.StatusCreated)
			return
		}
		c.String(http.StatusForbidden, err.Error())
	})

	existingRecorder := httptest.NewRecorder()
	router.ServeHTTP(existingRecorder, httptest.NewRequest(http.MethodGet, "/existing", nil))
	require.Equal(t, http.StatusOK, existingRecorder.Code)
	assert.Equal(t, fmt.Sprintf("%d", existing.Id), existingRecorder.Body.String())

	newRecorder := httptest.NewRecorder()
	router.ServeHTTP(newRecorder, httptest.NewRequest(http.MethodGet, "/new", nil))
	require.Equal(t, http.StatusForbidden, newRecorder.Code)
	assert.Contains(t, newRecorder.Body.String(), "GitHub")
}

func TestPasswordRegistrationIsRejectedByGitHubOnlyGate(t *testing.T) {
	originalRegisterEnabled := common.RegisterEnabled
	common.RegisterEnabled = true
	t.Cleanup(func() { common.RegisterEnabled = originalRegisterEnabled })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"username":"new-user","password":"password123"}`))
	Register(c)

	assert.Contains(t, recorder.Body.String(), string(i18n.MsgOAuthGitHubRegistrationOnly))
	assert.Contains(t, recorder.Body.String(), `"success":false`)
}

func TestFindOrCreateOAuthUserPersistsGitHubRegistrationEvidence(t *testing.T) {
	db := setupGitHubRegistrationControllerTestDB(t)
	createdAt := time.Now().UTC().Add(-365 * 24 * time.Hour).Unix()

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("github-registration-evidence-test"))))
	router.GET("/create", func(c *gin.Context) {
		user, err := findOrCreateOAuthUser(c, &oauth.GitHubProvider{}, &oauth.OAuthUser{
			ProviderUserID: "4001",
			Username:       "github-evidence-user",
			DisplayName:    "GitHub Evidence User",
			CreatedAt:      createdAt,
		}, sessions.Default(c))
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.String(http.StatusOK, "%d", user.Id)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/create", nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var stored model.User
	require.NoError(t, db.Where("github_id = ?", "4001").First(&stored).Error)
	assert.Equal(t, createdAt, stored.GithubCreatedAt)
	assert.False(t, stored.GithubAgeExempt)
}

func TestCreateGitHubRegistrationWhitelistRevalidatesResolvedIdentity(t *testing.T) {
	db := setupGitHubRegistrationControllerTestDB(t)
	originalResolver := resolveGitHubIdentity
	t.Cleanup(func() { resolveGitHubIdentity = originalResolver })

	resolvedID := int64(3001)
	resolveGitHubIdentity = func(_ context.Context, username string) (*oauth.GitHubIdentity, error) {
		return &oauth.GitHubIdentity{
			ID:        resolvedID,
			Login:     username,
			CreatedAt: time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC).Unix(),
		}, nil
	}

	resolveRecorder := httptest.NewRecorder()
	resolveContext, _ := gin.CreateTestContext(resolveRecorder)
	resolveContext.Request = httptest.NewRequest(http.MethodPost, "/api/github-registration/resolve", bytes.NewBufferString(`{"username":"octocat"}`))
	ResolveGitHubRegistrationIdentity(resolveContext)
	require.Equal(t, http.StatusOK, resolveRecorder.Code)

	resolvedID = 3002
	createRecorder := httptest.NewRecorder()
	createContext, _ := gin.CreateTestContext(createRecorder)
	createContext.Set("id", 42)
	createContext.Set("username", "root")
	createContext.Set("role", common.RoleRootUser)
	createContext.Request = httptest.NewRequest(http.MethodPost, "/api/github-registration/whitelist", bytes.NewBufferString(`{"username":"octocat","github_id":"3001","remark":"manual exception"}`))
	CreateGitHubRegistrationWhitelist(createContext)
	require.Equal(t, http.StatusConflict, createRecorder.Code)

	var count int64
	require.NoError(t, db.Model(&model.GitHubRegistrationWhitelist{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestResolveGitHubRegistrationIdentityFailsClosedOnAPIError(t *testing.T) {
	setupGitHubRegistrationControllerTestDB(t)
	originalResolver := resolveGitHubIdentity
	resolveGitHubIdentity = func(context.Context, string) (*oauth.GitHubIdentity, error) {
		return nil, fmt.Errorf("upstream unavailable: secret-token-must-not-leak")
	}
	t.Cleanup(func() { resolveGitHubIdentity = originalResolver })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/github-registration/resolve", bytes.NewBufferString(`{"username":"octocat"}`))
	ResolveGitHubRegistrationIdentity(c)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "secret-token-must-not-leak")
}
