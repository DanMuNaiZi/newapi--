package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestManualWechatPaymentPermissionsSeparateReadAndOperate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(db))

	const readOnlyAdminID = 901
	require.NoError(t, authz.SetUserPermissionsForRole(readOnlyAdminID, common.RoleAuthorizedAdmin, authz.PermissionsMap{
		authz.ResourcePayment: {
			authz.ActionRead:  true,
			authz.ActionWrite: false,
		},
		authz.ResourceUser: {
			authz.ActionRead:  true,
			authz.ActionQuota: false,
		},
		authz.ResourceSubscription: {
			authz.ActionRead:    true,
			authz.ActionOperate: false,
		},
	}))

	gin.SetMode(gin.TestMode)
	check := func(t *testing.T, userID int, role int, permission authz.Permission, wantAllowed bool) {
		t.Helper()
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/manual-wechat-permission-test", nil)
		context.Set("id", userID)
		context.Set("role", role)
		middleware.RequirePermission(permission)(context)
		if wantAllowed {
			assert.False(t, context.IsAborted())
			assert.Equal(t, http.StatusOK, recorder.Code)
			return
		}
		assert.True(t, context.IsAborted())
		assert.Equal(t, http.StatusForbidden, recorder.Code)
	}

	for _, permission := range []authz.Permission{authz.PaymentRead, authz.UserRead, authz.SubscriptionRead} {
		check(t, readOnlyAdminID, common.RoleAuthorizedAdmin, permission, true)
	}
	for _, permission := range []authz.Permission{authz.PaymentWrite, authz.UserQuota, authz.SubscriptionOperate} {
		check(t, readOnlyAdminID, common.RoleAuthorizedAdmin, permission, false)
	}
	for _, permission := range []authz.Permission{authz.PaymentRead, authz.UserRead, authz.SubscriptionRead} {
		check(t, 902, common.RoleCommonUser, permission, false)
	}
}
