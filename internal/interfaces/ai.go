package interfaces

type ContentGenerator interface {
	ChatCompletion(message string) (string, error)
}
