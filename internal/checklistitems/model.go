package checklistitems

type CreateReq struct {
	Description string  `json:"description"`
	Scope       *string `json:"scope,omitempty"`
}

type UpdateReq struct {
	Description *string `json:"description,omitempty"`
	Done        *bool   `json:"done,omitempty"`
	Sequence    *bool   `json:"sequence,omitempty"`
	Scope       *string `json:"scope,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
}

type BatchUpdateReq struct {
	list []UpdateReq
}
