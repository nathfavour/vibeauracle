module github.com/nathfavour/vibeauracle/model

go 1.24.1

require (
	github.com/nathfavour/vibeauracle/copilot v0.0.0
	github.com/ollama/ollama v0.13.5
	github.com/tmc/langchaingo v0.1.12
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/cli/go-gh/v2 v2.11.2 // indirect
	github.com/cli/safeexec v1.0.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/github/copilot-sdk/go v0.0.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/nathfavour/vibeauracle/auth v0.0.0
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
)

replace github.com/nathfavour/vibeauracle/copilot => ../copilot

replace github.com/github/copilot-sdk/go => ../copilot-sdk-go

replace github.com/nathfavour/vibeauracle/auth => ../auth
