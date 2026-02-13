package model

func init() {
	Register("gemini-cli", func(config map[string]string) (Provider, error) {
		modelName := config["model"]
		// If no specific model is requested, NewProvider will try to use the CLI default
		return NewProvider(modelName)
	})
}
