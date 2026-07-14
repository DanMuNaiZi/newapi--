package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// NotifyPendingLotteryWinners sends external notifications only for winners.
// The durable in-app notification is created in the draw transaction; this
// dispatcher records successful external delivery separately so retries do
// not send a second notification.
func NotifyPendingLotteryWinners() error {
	notifications, err := model.ListPendingLotteryWinnerNotifications(100)
	if err != nil {
		return err
	}
	for _, notification := range notifications {
		user, err := model.GetUserById(notification.UserId, true)
		if err != nil {
			common.SysError(fmt.Sprintf("failed to load lottery winner %d: %s", notification.UserId, err.Error()))
			continue
		}
		if err := NotifyUser(
			user.Id,
			user.Email,
			user.GetSetting(),
			dto.NewNotify(dto.NotifyTypeLotteryResult, "Lottery result", "You won a lottery reward. Sign in to view or claim it.", nil),
		); err != nil {
			common.SysError(fmt.Sprintf("failed to notify lottery winner %d: %s", user.Id, err.Error()))
			continue
		}
		if _, err := model.MarkLotteryNotificationExternallyNotified(notification.Id, common.GetTimestamp()); err != nil {
			return err
		}
	}
	return nil
}
