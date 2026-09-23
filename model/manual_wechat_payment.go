package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const manualWechatOrderReuseSeconds int64 = 60

type AdminTopUpFilter struct {
	PaymentProvider string
	Status          string
	UserId          int
	Keyword         string
}

type AdminTopUpRecord struct {
	TopUp
	Username string `json:"username"`
}

type AdminSubscriptionOrderFilter struct {
	PaymentProvider string
	Status          string
	UserId          int
	PlanId          int
	Keyword         string
}

type AdminSubscriptionOrderRecord struct {
	SubscriptionOrder
	Username  string `json:"username"`
	PlanTitle string `json:"plan_title"`
}

func generateManualWechatTradeNo(kind string) (string, error) {
	randomPart, err := common.GenerateRandomCharsKey(12)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MWX-%s-%s", kind, strings.ToUpper(randomPart)), nil
}

func CreateOrReuseManualWechatTopUp(userId int, amount int64, money float64, now int64) (*TopUp, bool, error) {
	if userId <= 0 || amount <= 0 || money <= 0 || now <= 0 {
		return nil, false, errors.New("invalid manual wechat topup")
	}

	var result *TopUp
	reused := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}

		var existing TopUp
		err := tx.Where(
			"user_id = ? AND amount = ? AND money = ? AND payment_provider = ? AND status = ? AND create_time >= ?",
			userId,
			amount,
			money,
			PaymentProviderManualWechat,
			common.TopUpStatusPending,
			now-manualWechatOrderReuseSeconds,
		).Order("id DESC").First(&existing).Error
		if err == nil {
			result = &existing
			reused = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			tradeNo, err := generateManualWechatTradeNo("T")
			if err != nil {
				return err
			}
			order := &TopUp{
				UserId:          userId,
				Amount:          amount,
				Money:           money,
				TradeNo:         tradeNo,
				PaymentMethod:   PaymentMethodManualWechat,
				PaymentProvider: PaymentProviderManualWechat,
				CreateTime:      now,
				Status:          common.TopUpStatusPending,
			}
			lastErr = tx.Transaction(func(insertTx *gorm.DB) error {
				return insertTx.Create(order).Error
			})
			if lastErr == nil {
				result = order
				return nil
			}
		}
		return lastErr
	})
	return result, reused, err
}

func CreateOrReuseManualWechatSubscriptionOrder(userId int, planId int, money float64, now int64) (*SubscriptionOrder, bool, error) {
	if userId <= 0 || planId <= 0 || money <= 0 || now <= 0 {
		return nil, false, errors.New("invalid manual wechat subscription order")
	}

	var result *SubscriptionOrder
	reused := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}

		var existing SubscriptionOrder
		err := tx.Where(
			"user_id = ? AND plan_id = ? AND payment_provider = ? AND status = ? AND create_time >= ?",
			userId,
			planId,
			PaymentProviderManualWechat,
			common.TopUpStatusPending,
			now-manualWechatOrderReuseSeconds,
		).Order("id DESC").First(&existing).Error
		if err == nil {
			result = &existing
			reused = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			tradeNo, err := generateManualWechatTradeNo("S")
			if err != nil {
				return err
			}
			order := &SubscriptionOrder{
				UserId:          userId,
				PlanId:          planId,
				Money:           money,
				TradeNo:         tradeNo,
				PaymentMethod:   PaymentMethodManualWechat,
				PaymentProvider: PaymentProviderManualWechat,
				Status:          common.TopUpStatusPending,
				CreateTime:      now,
			}
			lastErr = tx.Transaction(func(insertTx *gorm.DB) error {
				return insertTx.Create(order).Error
			})
			if lastErr == nil {
				result = order
				return nil
			}
		}
		return lastErr
	})
	return result, reused, err
}

func GetAdminTopUps(pageInfo *common.PageInfo, filter AdminTopUpFilter) ([]*AdminTopUpRecord, int64, error) {
	if pageInfo == nil {
		return nil, 0, errors.New("page info is required")
	}

	query := DB.Table("top_ups").
		Select("top_ups.*, users.username AS username").
		Joins("LEFT JOIN users ON users.id = top_ups.user_id")
	if filter.PaymentProvider != "" {
		query = query.Where("top_ups.payment_provider = ?", filter.PaymentProvider)
	}
	if filter.Status != "" {
		query = query.Where("top_ups.status = ?", filter.Status)
	}
	if filter.UserId > 0 {
		query = query.Where("top_ups.user_id = ?", filter.UserId)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern, err := sanitizeLikePattern("%" + keyword + "%")
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("top_ups.trade_no LIKE ? ESCAPE '!' OR users.username LIKE ? ESCAPE '!'", pattern, pattern)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []*AdminTopUpRecord
	if err := query.Order("top_ups.id DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Scan(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func GetAdminSubscriptionOrders(pageInfo *common.PageInfo, filter AdminSubscriptionOrderFilter) ([]*AdminSubscriptionOrderRecord, int64, error) {
	if pageInfo == nil {
		return nil, 0, errors.New("page info is required")
	}

	query := DB.Table("subscription_orders").
		Select("subscription_orders.*, users.username AS username, subscription_plans.title AS plan_title").
		Joins("LEFT JOIN users ON users.id = subscription_orders.user_id").
		Joins("LEFT JOIN subscription_plans ON subscription_plans.id = subscription_orders.plan_id")
	if filter.PaymentProvider != "" {
		query = query.Where("subscription_orders.payment_provider = ?", filter.PaymentProvider)
	}
	if filter.Status != "" {
		query = query.Where("subscription_orders.status = ?", filter.Status)
	}
	if filter.UserId > 0 {
		query = query.Where("subscription_orders.user_id = ?", filter.UserId)
	}
	if filter.PlanId > 0 {
		query = query.Where("subscription_orders.plan_id = ?", filter.PlanId)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern, err := sanitizeLikePattern("%" + keyword + "%")
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(
			"subscription_orders.trade_no LIKE ? ESCAPE '!' OR users.username LIKE ? ESCAPE '!' OR subscription_plans.title LIKE ? ESCAPE '!'",
			pattern,
			pattern,
			pattern,
		)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []*AdminSubscriptionOrderRecord
	if err := query.Order("subscription_orders.id DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Scan(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}
