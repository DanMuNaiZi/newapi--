package service

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type lotteryScheduleTaskHandler struct{}

func (lotteryScheduleTaskHandler) Type() string {
	return model.SystemTaskTypeLotterySchedule
}

func (lotteryScheduleTaskHandler) Enabled() bool {
	return true
}

func (lotteryScheduleTaskHandler) Interval() time.Duration {
	return 15 * time.Second
}

func (lotteryScheduleTaskHandler) NewPayload() any {
	return map[string]string{"source": "scheduler"}
}

func (lotteryScheduleTaskHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	select {
	case <-ctx.Done():
		failSystemTask(task, runnerID, ctx.Err())
		return
	default:
	}
	if err := model.ProcessLotterySchedule(common.GetTimestamp()); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if err := NotifyPendingLotteryWinners(); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, map[string]string{"status": "processed"}, "")
}

func init() {
	RegisterSystemTaskHandler(lotteryScheduleTaskHandler{})
}
