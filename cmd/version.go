package cmd

// ビルド時に ldflags で注入されるバージョン情報。
// 例: go build -ldflags "-X github.com/kznagamori/go_PDFStitcher/cmd.version=1.0.0"
//
//nolint:gochecknoglobals // ldflags 注入のため var が必須
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)
