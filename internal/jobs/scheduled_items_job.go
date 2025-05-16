package jobs

import (
	"context"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

type ScheduledItemsJob struct {
	checklistService ChecklistScheduledItemsService
	cron             *cron.Cron
	jobID            cron.EntryID
}

type ChecklistScheduledItemsService interface {
	TriggerScheduledReminder(ctx context.Context) error
	CheckUpcomingItems(ctx context.Context) error
}

func NewScheduledItemsJob(checklistService ChecklistScheduledItemsService) *ScheduledItemsJob {
	c := cron.New(cron.WithSeconds())
	return &ScheduledItemsJob{
		checklistService: checklistService,
		cron:             c,
	}
}

func (j *ScheduledItemsJob) Start() {
	fmt.Println("Starting scheduled reminder checker job.")
	// Run every minute (second minute hour day month weekday)
	jobID, err := j.cron.AddFunc("0 * * * * *", func() {
		ctx := context.Background()
		err := j.checklistService.CheckUpcomingItems(ctx)
		if err != nil {
			log.Printf("error when checking scheduled checklist items in job.: %s\n", err.Error())
		}
	})

	if err != nil {
		log.Printf("Error scheduling checklist items job: %s\n", err.Error())
		return
	}

	j.jobID = jobID
	j.cron.Start()
}

func (j *ScheduledItemsJob) Stop() {
	fmt.Println("Stopping scheduled items job.")

	ctx := j.cron.Stop()
	// Wait for jobs to finish
	<-ctx.Done()
}
