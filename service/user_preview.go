package service

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const UserPreviewTokenTTL = 15 * time.Minute

var (
	ErrUserPreviewInvalid = errors.New("invalid user preview token")
	ErrUserPreviewExpired = errors.New("user preview token has expired")
)

type UserPreviewClaims struct {
	ActorUserId  int   `json:"actor_user_id"`
	TargetUserId int   `json:"target_user_id"`
	IssuedAt     int64 `json:"issued_at"`
	ExpiresAt    int64 `json:"expires_at"`
}

func CreateUserPreviewToken(actorUserId int, targetUserId int) (string, *model.User, *UserPreviewClaims, error) {
	if actorUserId <= 0 || targetUserId <= 0 || actorUserId == targetUserId {
		return "", nil, nil, ErrUserPreviewInvalid
	}
	target, err := loadUserPreviewTarget(targetUserId)
	if err != nil {
		return "", nil, nil, err
	}
	now := common.GetTimestamp()
	claims := &UserPreviewClaims{
		ActorUserId:  actorUserId,
		TargetUserId: targetUserId,
		IssuedAt:     now,
		ExpiresAt:    now + int64(UserPreviewTokenTTL/time.Second),
	}
	token, err := signUserPreviewClaims(claims)
	if err != nil {
		return "", nil, nil, err
	}
	return token, target, claims, nil
}

func ValidateUserPreviewToken(token string, actorUserId int) (*UserPreviewClaims, *model.User, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, nil, ErrUserPreviewInvalid
	}
	expectedSignature := common.GenerateHMAC(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expectedSignature)) != 1 {
		return nil, nil, ErrUserPreviewInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, ErrUserPreviewInvalid
	}
	claims := &UserPreviewClaims{}
	if err := common.Unmarshal(payload, claims); err != nil || claims.ActorUserId <= 0 || claims.TargetUserId <= 0 {
		return nil, nil, ErrUserPreviewInvalid
	}
	if claims.ActorUserId != actorUserId {
		return claims, nil, ErrUserPreviewInvalid
	}
	if claims.ExpiresAt <= common.GetTimestamp() || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return claims, nil, ErrUserPreviewExpired
	}
	target, err := loadUserPreviewTarget(claims.TargetUserId)
	if err != nil {
		return claims, nil, err
	}
	return claims, target, nil
}

func signUserPreviewClaims(claims *UserPreviewClaims) (string, error) {
	payload, err := common.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + common.GenerateHMAC(encoded), nil
}

func loadUserPreviewTarget(targetUserId int) (*model.User, error) {
	target, err := model.GetUserById(targetUserId, false)
	if err != nil {
		return nil, err
	}
	if target.Status != common.UserStatusEnabled || target.Role != common.RoleCommonUser {
		return nil, errors.New("only enabled regular users can be previewed")
	}
	return target, nil
}
