module github.com/nathfavour/vibeauracle/daemon

go 1.21
toolchain go1.21.0

require (
	github.com/nathfavour/vibeauracle/brain v0.0.0
	github.com/nathfavour/vibeauracle/sys v0.0.0
	google.golang.org/grpc v1.78.0
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/nathfavour/vibeauracle/brain => ../brain
replace github.com/nathfavour/vibeauracle/sys => ../sys
replace github.com/nathfavour/vibeauracle/copilot => ../copilot
replace github.com/nathfavour/vibeauracle/prompt => ../prompt
replace github.com/nathfavour/vibeauracle/vault => ../vault
replace github.com/nathfavour/vibeauracle/context => ../context
replace github.com/nathfavour/vibeauracle/tooling => ../tooling
replace github.com/github/copilot-sdk/go => ../copilot-sdk-go
replace github.com/nathfavour/vibeauracle/auth => ../auth
