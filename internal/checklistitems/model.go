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
	// NOTE: no binding for validation as datetime binding had a known issue
	ScheduledTime *string `json:"scheduledTime,omitempty"`
}
