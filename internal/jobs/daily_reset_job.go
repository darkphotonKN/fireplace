package jobs

import (
	"context"
	"fmt"

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
	fmt.Println("Starting daily reset jobs.")

	jobID, err := j.cron.AddFunc("0 0 15 * * *", func() {
		fmt.Println("Running daily job...")
		ctx := context.Background()
		err := j.checklistService.ResetDailyItems(ctx)
		if err != nil {
			fmt.Printf("Error resetting daily items: %s\n", err.Error())
		}
	})

	if err != nil {
		fmt.Printf("Error scheduling daily reset job: %s\n", err.Error())
		return
	}

	j.jobID = jobID
	j.cron.Start()
}

func (j *DailyResetJob) Stop() {
	fmt.Println("Stopping daily reset job.")
	ctx := j.cron.Stop()
	// Wait for jobs to finish
	<-ctx.Done()
}
