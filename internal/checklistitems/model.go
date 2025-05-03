package checklistitems

type CreateReq struct {
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Position    int    `json:"position"`
}
