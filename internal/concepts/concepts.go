package concepts

// represents a learning or development resource concept
type Concept struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Relevance   float32 `json:"relevance"`
}
