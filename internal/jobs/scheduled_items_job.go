package jobs

import (
	"context"

	"github.com/darkphotonKN/fireplace/internal/logger"
	"github.com/robfig/cron/v3"
)

type ScheduledItemsJob struct {
	checklistService ChecklistScheduledItemsService
	cron             *cron.Cron
	jobID            cron.EntryID
}

type ChecklistScheduledItemsService interface {
	TriggerScheduledReminder(ctx context.Context) error
	CheckAllScheduledItems(ctx context.Context) error
}

func NewScheduledItemsJob(checklistService ChecklistScheduledItemsService) *ScheduledItemsJob {
	c := cron.New(cron.WithSeconds())
	return &ScheduledItemsJob{
		checklistService: checklistService,
		cron:             c,
	}
}

func (j *ScheduledItemsJob) Start() {
	logger.Info("Starting scheduled reminder checker job")
	// Run every minute (second minute hour day month weekday)
	jobID, err := j.cron.AddFunc("0 * * * * *", func() {
		ctx := context.Background()
		err := j.checklistService.CheckAllScheduledItems(ctx)
		if err != nil {
			logger.Error("Error checking scheduled checklist items", "error", err)
		}
	})

	if err != nil {
		logger.Error("Error scheduling checklist items job", "error", err)
		return
	}

	j.jobID = jobID
	j.cron.Start()
}

func (j *ScheduledItemsJob) Stop() {
	logger.Info("Stopping scheduled items job")

	ctx := j.cron.Stop()
	// Wait for jobs to finish
	<-ctx.Done()
}
