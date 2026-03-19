module github.com/nathfavour/vibeauracle/daemon

go 1.24.0

require (
	github.com/nathfavour/vibeauracle/brain v0.0.0
	google.golang.org/grpc v1.79.3
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cli/go-gh/v2 v2.11.2 // indirect
	github.com/cli/safeexec v1.0.0 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.7.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/github/copilot-sdk/go v0.0.0 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/nathfavour/vibeauracle/auth v0.0.0-00010101000000-000000000000 // indirect
	github.com/nathfavour/vibeauracle/context v0.0.0-00010101000000-000000000000 // indirect
	github.com/nathfavour/vibeauracle/copilot v0.0.0 // indirect
	github.com/nathfavour/vibeauracle/internal/doctor v0.0.0-20260213160425-489ed677ea63 // indirect
	github.com/nathfavour/vibeauracle/internal/vibe v0.0.0-20260213160425-489ed677ea63 // indirect
	github.com/nathfavour/vibeauracle/prompt v0.0.0 // indirect
	github.com/nathfavour/vibeauracle/sys v0.0.0 // indirect
	github.com/nathfavour/vibeauracle/tooling v0.0.0-00010101000000-000000000000 // indirect
	github.com/nathfavour/vibeauracle/vault v0.0.0-00010101000000-000000000000 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/philippgille/chromem-go v0.7.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.3.4 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.lsp.dev/jsonrpc2 v0.10.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/protocol v0.12.0 // indirect
	go.lsp.dev/uri v0.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.37.6 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/sqlite v1.28.0 // indirect
	mvdan.cc/sh/v3 v3.12.0 // indirect
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
