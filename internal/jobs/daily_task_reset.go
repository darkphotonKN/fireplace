package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron"
)

type DailyResetJob struct {
	checklistService ChecklistDailyResetService
	scheduler        *gocron.Scheduler
}

type ChecklistDailyResetService interface {
	ResetDailyItems(ctx context.Context) error
}

func NewDailyResetJob(checklistService ChecklistDailyResetService) *DailyResetJob {
	scheduler := gocron.NewScheduler(time.UTC)
	return &DailyResetJob{
		checklistService: checklistService,
		scheduler:        scheduler,
	}
}

func (j *DailyResetJob) Start() {
	j.scheduler.Every(1).Day().At("02:44").Do(func() {
		fmt.Println("Running daily job...")
		ctx := context.Background()
		err := j.checklistService.ResetDailyItems(ctx)
		if err != nil {
			fmt.Printf("Error resetting daily items: %v\n", err)
		}
	})

	j.scheduler.StartAsync()
}

func (j *DailyResetJob) Stop() {
	j.scheduler.Stop()
}
