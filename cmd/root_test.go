package cmd

import (
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/config"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	if cmd == nil {
		t.Fatal("newRootCmd() は nil を返すべきではない")
	}
	if cmd.Use == "" {
		t.Error("Use が空であるべきではない")
	}
}

func TestNewRootCmdVersion(t *testing.T) {
	cmd := newRootCmd()
	if cmd.Version == "" {
		t.Error("Version が設定されているべき")
	}
}

func TestNewRootCmdFlags(t *testing.T) {
	cmd := newRootCmd()
	flags := cmd.Flags()

	tests := []struct {
		name     string
		flagName string
	}{
		{"outputフラグ", "output"},
		{"gapフラグ", "gap"},
		{"alignフラグ", "align"},
		{"forceフラグ", "force"},
		{"verboseフラグ", "verbose"},
		{"quietフラグ", "quiet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := flags.Lookup(tt.flagName)
			if f == nil {
				t.Errorf("フラグ %q が定義されていない", tt.flagName)
			}
		})
	}
}

func TestNewRootCmdOutputShortFlag(t *testing.T) {
	cmd := newRootCmd()
	f := cmd.Flags().ShorthandLookup("o")
	if f == nil {
		t.Error("-o ショートフラグが定義されていない")
	}
}

func TestNewRootCmdRequiresArgs(t *testing.T) {
	// 引数なしで実行するとエラーになることを検証する
	cmd := newRootCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("引数なしでエラーが返されるべき")
	}
}

func TestConfigureLogging(t *testing.T) {
	t.Run("verbose と quiet の同時指定はエラー", func(t *testing.T) {
		// グローバルフラグを設定する
		origVerbose := verboseFlag
		origQuiet := quietFlag
		defer func() {
			verboseFlag = origVerbose
			quietFlag = origQuiet
		}()

		verboseFlag = true
		quietFlag = true

		err := configureLogging()
		if err == nil {
			t.Error("--verbose と --quiet の同時指定でエラーが返されるべき")
		}
	})

	t.Run("verbose のみは正常", func(t *testing.T) {
		origVerbose := verboseFlag
		origQuiet := quietFlag
		defer func() {
			verboseFlag = origVerbose
			quietFlag = origQuiet
		}()

		verboseFlag = true
		quietFlag = false

		if err := configureLogging(); err != nil {
			t.Errorf("verbose のみで予期しないエラー: %v", err)
		}
	})

	t.Run("quiet のみは正常", func(t *testing.T) {
		origVerbose := verboseFlag
		origQuiet := quietFlag
		defer func() {
			verboseFlag = origVerbose
			quietFlag = origQuiet
		}()

		verboseFlag = false
		quietFlag = true

		if err := configureLogging(); err != nil {
			t.Errorf("quiet のみで予期しないエラー: %v", err)
		}
	})

	t.Run("両方未指定は正常（INFOレベル）", func(t *testing.T) {
		origVerbose := verboseFlag
		origQuiet := quietFlag
		defer func() {
			verboseFlag = origVerbose
			quietFlag = origQuiet
		}()

		verboseFlag = false
		quietFlag = false

		if err := configureLogging(); err != nil {
			t.Errorf("両方未指定で予期しないエラー: %v", err)
		}
	})
}

func TestIsValidAlign(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"left は有効", "left", true},
		{"center は有効", "center", true},
		{"right は有効", "right", true},
		{"空文字は無効", "", false},
		{"不正値は無効", "xxx", false},
		{"大文字は無効", "Left", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidAlign(tt.value)
			if got != tt.want {
				t.Errorf("isValidAlign(%q) = %v, 期待値 %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveGap(t *testing.T) {
	t.Run("CLIフラグ指定時はフラグ値を使用", func(t *testing.T) {
		origGap := gapFlag
		defer func() { gapFlag = origGap }()
		gapFlag = "10pt"

		cfg := &config.Config{Gap: 5, GapUnit: "mm"}
		got, err := resolveGap(cfg)
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if got != 10.0 {
			t.Errorf("resolveGap() = %f, 期待値 10.0", got)
		}
	})

	t.Run("CLIフラグ未指定時は設定ファイル値を使用", func(t *testing.T) {
		origGap := gapFlag
		defer func() { gapFlag = origGap }()
		gapFlag = ""

		cfg := &config.Config{Gap: 0, GapUnit: "pt"}
		got, err := resolveGap(cfg)
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if got != 0.0 {
			t.Errorf("resolveGap() = %f, 期待値 0.0", got)
		}
	})
}

func TestIsDragAndDropExecution(t *testing.T) {
	t.Run("フラグなしはD&D判定", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"input.pdf"})
		// フラグをパースせずに判定する
		if !isDragAndDropExecution(cmd) {
			t.Error("フラグなしはD&D判定されるべき")
		}
	})
}
