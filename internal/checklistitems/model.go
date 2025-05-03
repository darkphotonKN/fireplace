package checklistitems

type CreateReq struct {
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

type UpdateItem struct {
	Description string `db:"description" json:"description"`
	Done        bool   `db:"done" json:"done"`
	Sequence    bool   `db:"sequence"`
}

type UpdateReq struct {
	list []UpdateItem
}
