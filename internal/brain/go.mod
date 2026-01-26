module github.com/nathfavour/vibeauracle/brain

go 1.25.0

require (
	github.com/nathfavour/vibeauracle/copilot v0.0.0
	github.com/nathfavour/vibeauracle/prompt v0.0.0
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/github/copilot-sdk/go v0.0.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
)

replace github.com/nathfavour/vibeauracle/prompt => ../prompt

replace github.com/nathfavour/vibeauracle/copilot => ../copilot

replace github.com/github/copilot-sdk/go => ../copilot-sdk-go
