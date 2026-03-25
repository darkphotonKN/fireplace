package interfaces

type ContentGenerator interface {
	Generate(message string) (string, error)
}

// AIGeneratorInterface for generating AI-powered content
type AIGeneratorInterface interface {
	GenerateContent(context string) (string, error)
}
