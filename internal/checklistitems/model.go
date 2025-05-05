package checklistitems

type CreateReq struct {
	Description string `json:"description"`
}

type UpdateReq struct {
	Description *string `json:"description,omitempty"`
	Done        *bool   `json:"done,omitempty"`
	Sequence    *bool   `json:"sequence,omitempty"`
}

type BatchUpdateReq struct {
	list []UpdateReq
}
