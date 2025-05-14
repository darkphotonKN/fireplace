package interfaces

type ContentGenerator interface {
	Generate(message string) (string, error)
}
