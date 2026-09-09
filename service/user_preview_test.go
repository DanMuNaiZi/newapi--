package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserPreviewFixture(t *testing.T) *model.User {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	previousDB := model.DB
	model.DB = db
	originalSecret := common.CryptoSecret
	common.CryptoSecret = "user-preview-test-secret"
	t.Cleanup(func() {
		model.DB = previousDB
		common.CryptoSecret = originalSecret
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	target := &model.User{Username: "preview-target", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "vip", AffCode: "preview-target"}
	require.NoError(t, db.Create(target).Error)
	return target
}

func TestUserPreviewTokenIsBoundToActorAndTargetState(t *testing.T) {
	target := setupUserPreviewFixture(t)
	token, _, claims, err := CreateUserPreviewToken(100, target.Id)
	require.NoError(t, err)
	assert.Equal(t, target.Id, claims.TargetUserId)

	validatedClaims, validatedTarget, err := ValidateUserPreviewToken(token, 100)
	require.NoError(t, err)
	assert.Equal(t, claims.TargetUserId, validatedClaims.TargetUserId)
	assert.Equal(t, target.Username, validatedTarget.Username)

	_, _, err = ValidateUserPreviewToken(token, 101)
	assert.ErrorIs(t, err, ErrUserPreviewInvalid)

	tamperedSuffix := "0"
	if strings.HasSuffix(token, tamperedSuffix) {
		tamperedSuffix = "1"
	}
	tampered := token[:len(token)-1] + tamperedSuffix
	_, _, err = ValidateUserPreviewToken(tampered, 100)
	assert.ErrorIs(t, err, ErrUserPreviewInvalid)

	require.NoError(t, model.DB.Model(target).Update("status", common.UserStatusDisabled).Error)
	_, _, err = ValidateUserPreviewToken(token, 100)
	assert.ErrorContains(t, err, "enabled regular users")
}

func TestUserPreviewTokenRejectsExpiredClaims(t *testing.T) {
	setupUserPreviewFixture(t)
	now := common.GetTimestamp()
	token, err := signUserPreviewClaims(&UserPreviewClaims{ActorUserId: 100, TargetUserId: 1, IssuedAt: now - 30, ExpiresAt: now - 1})
	require.NoError(t, err)
	claims, _, err := ValidateUserPreviewToken(token, 100)
	assert.ErrorIs(t, err, ErrUserPreviewExpired)
	require.NotNil(t, claims)
	assert.Equal(t, 1, claims.TargetUserId)
}
