package checklistitems

import "time"

type CreateReq struct {
	Description string `json:"description"`
}

type UpdateReq struct {
	Description   *string `json:"description,omitempty"`
	Done          *bool   `json:"done,omitempty"`
	Sequence      *bool   `json:"sequence,omitempty"`
	ScheduledTime *time.Time
}

type BatchUpdateReq struct {
	list []UpdateReq
}

type SetScheduleReq struct {
	ScheduledTime *string `json:"scheduledTime,omitempty" binding:"omitempty,datetime" time_format:"2006-01-02T15:04:05Z07:00"`
}
