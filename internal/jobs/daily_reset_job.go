package jobs

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/robfig/cron/v3"
)

type DailyResetJob struct {
	checklistService ChecklistDailyResetService
	cron             *cron.Cron
	jobID            cron.EntryID
}

type ChecklistDailyResetService interface {
	ResetDailyItems(ctx context.Context) error
}

func NewDailyResetJob(checklistService ChecklistDailyResetService) *DailyResetJob {
	c := cron.New(cron.WithSeconds())

	return &DailyResetJob{
		checklistService: checklistService,
		cron:             c,
	}
}

func (j *DailyResetJob) Start() {
	logger.Info("Starting daily reset job")

	jobID, err := j.cron.AddFunc("0 0 14 * * *", func() {
		logger.Info("Running daily reset job")
		ctx := context.Background()
		err := j.checklistService.ResetDailyItems(ctx)
		if err != nil {
			logger.Error("Error resetting daily items", "error", err)
		}
	})

	if err != nil {
		logger.Error("Error scheduling daily reset job", "error", err)
		return
	}

	j.jobID = jobID
	j.cron.Start()
}

func (j *DailyResetJob) Stop() {
	logger.Info("Stopping daily reset job")
	ctx := j.cron.Stop()
	// Wait for jobs to finish
	<-ctx.Done()
}
