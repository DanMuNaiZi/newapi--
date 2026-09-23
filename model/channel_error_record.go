package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxChannelErrorSampleBytes = 4 * 1024

type ChannelErrorRecord struct {
	ID                 uint                        `json:"id" gorm:"primaryKey"`
	ChannelID          int                         `json:"channel_id" gorm:"not null;index"`
	Signature          string                      `json:"signature" gorm:"type:varchar(64);not null;uniqueIndex"`
	RequestPath        string                      `json:"request_path" gorm:"type:varchar(512);not null"`
	RequestModel       string                      `json:"request_model" gorm:"type:varchar(256);not null;index"`
	UpstreamStatusCode int                         `json:"upstream_status_code" gorm:"not null;index"`
	FinalStatusCode    int                         `json:"final_status_code" gorm:"not null"`
	ErrorType          string                      `json:"error_type" gorm:"type:varchar(128);not null"`
	ErrorCode          string                      `json:"error_code" gorm:"type:varchar(256);not null"`
	SampleMessage      string                      `json:"sample_message" gorm:"type:text;not null"`
	OccurrenceCount    int64                       `json:"occurrence_count" gorm:"not null"`
	FirstSeenTime      int64                       `json:"first_seen_time" gorm:"not null;index"`
	LastSeenTime       int64                       `json:"last_seen_time" gorm:"not null;index"`
	LastRequestID      string                      `json:"last_request_id" gorm:"type:varchar(128)"`
	LastMultiKeyIndex  int                         `json:"last_multi_key_index"`
	DisplayMode        dto.ChannelErrorDisplayMode `json:"display_mode" gorm:"type:varchar(16);not null"`
	DisplayStatusCode  int                         `json:"display_status_code"`
	DisplayMessage     string                      `json:"display_message" gorm:"type:varchar(500)"`
}

type ChannelErrorRecordInput struct {
	ChannelID          int
	RequestPath        string
	RequestModel       string
	UpstreamStatusCode int
	FinalStatusCode    int
	ErrorType          string
	ErrorCode          string
	SampleMessage      string
	RequestID          string
	MultiKeyIndex      int
}

type ChannelErrorRecordQuery struct {
	ChannelID    int
	StatusCode   int
	RequestModel string
	Keyword      string
	Offset       int
	Limit        int
}

func (record *ChannelErrorRecord) TableName() string {
	return "channel_error_records"
}

func UpsertChannelErrorRecord(input ChannelErrorRecordInput) (*ChannelErrorRecord, error) {
	normalizedMessage := common.NormalizeChannelErrorMessage(input.SampleMessage)
	signatureSource := strings.Join([]string{
		strconv.Itoa(input.ChannelID),
		strings.TrimSpace(input.RequestPath),
		strings.TrimSpace(input.RequestModel),
		strconv.Itoa(input.UpstreamStatusCode),
		strings.TrimSpace(input.ErrorType),
		strings.TrimSpace(input.ErrorCode),
		normalizedMessage,
	}, "\x00")
	signatureBytes := sha256.Sum256([]byte(signatureSource))
	signature := hex.EncodeToString(signatureBytes[:])
	now := common.GetTimestamp()
	record := &ChannelErrorRecord{
		ChannelID:          input.ChannelID,
		Signature:          signature,
		RequestPath:        strings.TrimSpace(input.RequestPath),
		RequestModel:       strings.TrimSpace(input.RequestModel),
		UpstreamStatusCode: input.UpstreamStatusCode,
		FinalStatusCode:    input.FinalStatusCode,
		ErrorType:          strings.TrimSpace(input.ErrorType),
		ErrorCode:          strings.TrimSpace(input.ErrorCode),
		SampleMessage:      truncateChannelErrorSample(input.SampleMessage),
		OccurrenceCount:    1,
		FirstSeenTime:      now,
		LastSeenTime:       now,
		LastRequestID:      strings.TrimSpace(input.RequestID),
		LastMultiKeyIndex:  input.MultiKeyIndex,
		DisplayMode:        dto.ChannelErrorDisplayModeInherit,
	}
	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "signature"}},
		DoUpdates: clause.Assignments(map[string]any{
			"final_status_code":    input.FinalStatusCode,
			"sample_message":       record.SampleMessage,
			"occurrence_count":     gorm.Expr("occurrence_count + ?", 1),
			"last_seen_time":       now,
			"last_request_id":      record.LastRequestID,
			"last_multi_key_index": input.MultiKeyIndex,
		}),
	}).Create(record).Error
	if err != nil {
		return nil, err
	}
	if err := DB.Where("signature = ?", signature).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func GetChannelErrorRecords(query ChannelErrorRecordQuery) ([]ChannelErrorRecord, int64, error) {
	db := DB.Model(&ChannelErrorRecord{}).Where("channel_id = ?", query.ChannelID)
	if query.StatusCode != 0 {
		db = db.Where("upstream_status_code = ?", query.StatusCode)
	}
	if requestModel := strings.TrimSpace(query.RequestModel); requestModel != "" {
		db = db.Where("request_model = ?", requestModel)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("sample_message LIKE ? OR error_code LIKE ? OR error_type LIKE ? OR request_path LIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := query.Limit
	if limit <= 0 {
		limit = common.ItemsPerPage
	}
	if limit > 100 {
		limit = 100
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	items := make([]ChannelErrorRecord, 0)
	if err := db.Order("last_seen_time DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func UpdateChannelErrorDisplayRule(channelID int, recordID uint, rule dto.ChannelErrorDisplayRule) (*ChannelErrorRecord, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	updates := map[string]any{
		"display_mode":        rule.Mode,
		"display_status_code": 0,
		"display_message":     "",
	}
	if rule.Mode == dto.ChannelErrorDisplayModeCustom {
		updates["display_status_code"] = rule.StatusCode
		updates["display_message"] = strings.TrimSpace(rule.Message)
	}
	result := DB.Model(&ChannelErrorRecord{}).
		Where("id = ? AND channel_id = ?", recordID, channelID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	record := &ChannelErrorRecord{}
	if err := DB.Where("id = ? AND channel_id = ?", recordID, channelID).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func truncateChannelErrorSample(message string) string {
	if len(message) <= MaxChannelErrorSampleBytes {
		return message
	}
	message = message[:MaxChannelErrorSampleBytes]
	for !utf8.ValidString(message) {
		message = message[:len(message)-1]
	}
	return message
}
