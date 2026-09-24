package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const UserPreviewTTL = 15 * time.Minute

var (
	ErrUserPreviewInvalid          = errors.New("user preview token is invalid")
	ErrUserPreviewExpired          = errors.New("user preview token is expired")
	ErrUserPreviewActorMismatch    = errors.New("user preview actor mismatch")
	ErrUserPreviewTargetInvalid    = errors.New("user preview target is invalid")
	ErrUserPreviewRequiresSession  = errors.New("user preview requires a browser session")
	ErrUserPreviewReadOnly         = errors.New("user preview is read-only")
	ErrUserPreviewPathNotAllowlist = errors.New("user preview path is not allowed")
)

type UserPreviewClaims struct {
	ActorID   int   `json:"actor_id"`
	TargetID  int   `json:"target_id"`
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

func userPreviewSigningKey() []byte {
	return []byte("user-preview-v1:" + common.SessionSecret)
}

func IssueUserPreviewToken(actorID, targetID int, now time.Time) (string, UserPreviewClaims, error) {
	if actorID <= 0 || targetID <= 0 {
		return "", UserPreviewClaims{}, ErrUserPreviewTargetInvalid
	}
	claims := UserPreviewClaims{
		ActorID: actorID, TargetID: targetID,
		IssuedAt: now.Unix(), ExpiresAt: now.Add(UserPreviewTTL).Unix(),
	}
	payload, err := common.Marshal(claims)
	if err != nil {
		return "", UserPreviewClaims{}, fmt.Errorf("marshal user preview claims: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, userPreviewSigningKey())
	_, _ = mac.Write([]byte(encodedPayload))
	encodedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + encodedSignature, claims, nil
}

func VerifyUserPreviewToken(raw string, now time.Time) (UserPreviewClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return UserPreviewClaims{}, ErrUserPreviewInvalid
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil {
		return UserPreviewClaims{}, ErrUserPreviewInvalid
	}
	signature, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(signature) != sha256.Size {
		return UserPreviewClaims{}, ErrUserPreviewInvalid
	}
	mac := hmac.New(sha256.New, userPreviewSigningKey())
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return UserPreviewClaims{}, ErrUserPreviewInvalid
	}
	var claims UserPreviewClaims
	if err := common.Unmarshal(payload, &claims); err != nil || claims.ActorID <= 0 || claims.TargetID <= 0 || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return UserPreviewClaims{}, ErrUserPreviewInvalid
	}
	nowUnix := now.Unix()
	if claims.IssuedAt > nowUnix+5 || claims.ExpiresAt <= nowUnix {
		return claims, ErrUserPreviewExpired
	}
	return claims, nil
}

func UserPreviewTokenFingerprint(raw string) string {
	return common.GenerateHMACWithKey([]byte("user-preview-ref:"+common.SessionSecret), raw)
}

func UserPreviewExpiry(claims UserPreviewClaims) string {
	return strconv.FormatInt(claims.ExpiresAt, 10)
}
