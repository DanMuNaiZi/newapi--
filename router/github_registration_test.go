package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGitHubRegistrationAdminRoutesRequireOAuthPermissions(t *testing.T) {
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
		&model.CasbinRule{},
		&model.AuthzRole{},
		&model.Log{},
	))
	require.NoError(t, authz.Init(db))

	rootToken := "root-github-registration-token"
	adminToken := "admin-github-registration-token"
	require.NoError(t, db.Create(&model.User{Id: 101, Username: "root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, AccessToken: &rootToken, AffCode: "root-code"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 102, Username: "admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AccessToken: &adminToken, AffCode: "admin-code"}).Error)

	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("github-registration-router-test"))))
	SetApiRouter(r)

	request := func(method, path, token string, userID int) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", token)
		req.Header.Set("New-Api-User", fmt.Sprintf("%d", userID))
		r.ServeHTTP(recorder, req)
		return recorder
	}

	adminRead := request(http.MethodGet, "/api/github-registration/whitelist", adminToken, 102)
	assert.Equal(t, http.StatusForbidden, adminRead.Code)
	rootRead := request(http.MethodGet, "/api/github-registration/whitelist", rootToken, 101)
	assert.Equal(t, http.StatusOK, rootRead.Code)

	adminWrite := request(http.MethodDelete, "/api/github-registration/whitelist/999", adminToken, 102)
	assert.Equal(t, http.StatusForbidden, adminWrite.Code)
	rootWrite := request(http.MethodDelete, "/api/github-registration/whitelist/999", rootToken, 101)
	assert.NotEqual(t, http.StatusForbidden, rootWrite.Code)
}
