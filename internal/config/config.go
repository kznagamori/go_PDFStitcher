package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// Config はアプリケーション設定を表す。
// TOML設定ファイルから読み込み、CLIフラグでオーバーライドする。
type Config struct {
	OutputPattern   string  `toml:"output_pattern"`   // 出力ファイル名パターン
	Gap             float64 `toml:"gap"`              // ガター幅（GapUnit の単位）
	GapUnit         string  `toml:"gap_unit"`         // ガター単位（"pt" or "mm"）
	Align           string  `toml:"align"`            // ページ配置方針（"left", "center", "right"）
	OnConflict      string  `toml:"on_conflict"`      // ファイル衝突時の挙動（"numbering", "overwrite"）
	Verbose         bool    `toml:"verbose"`          // 詳細ログ出力
	Quiet           bool    `toml:"quiet"`            // ログ抑制（エラーのみ出力）
	BookmarkPattern string  `toml:"bookmark_pattern"` // しおりテキストパターン
}

// CLIFlags はCLIフラグの値を保持する構造体。
// Merge で Config に上書きマージする際に使用する。
// 空文字列/false は「未指定」を意味し、Config の値を上書きしない。
type CLIFlags struct {
	Output  string // -o フラグの値（空文字 = 未指定）
	Gap     string // --gap フラグの値（空文字 = 未指定）
	Align   string // --align フラグの値（空文字 = 未指定）
	Force   bool   // --force フラグ
	Verbose bool   // --verbose フラグ
	Quiet   bool   // --quiet フラグ
}

// Load は実行ファイルと同ディレクトリの config.toml を読み込む。
//
// ファイル未存在時は初期値で config.toml の自動生成を試みる（書き込み不可時は警告のみ）。
// TOML構文エラー時は警告ログを出力しデフォルト値を返す。
// 不正な設定値は個別に警告ログを出力しデフォルト値にフォールバックする。
//
// 常に非nil の *Config を返す（エラー時もデフォルト値で初期化済み）。
// ファイルが存在するのに読み取れない場合のみ ExitError（コード1）を返す。
func Load(execDir string) (*Config, error) {
	cfg := Default()
	configPath := filepath.Join(execDir, configFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			generateConfig(configPath)
			return cfg, nil
		}
		return nil, apperror.NewGeneralError(
			fmt.Sprintf("failed to read config file: %s", configPath), err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		slog.Warn("invalid config file, using defaults", "error", err, "path", configPath)
		return Default(), nil
	}

	validateAndFallback(cfg)

	return cfg, nil
}

// Default はデフォルト設定を返す。
// 全フィールドが defaults.go で定義された初期値で設定される。
func Default() *Config {
	return &Config{
		OutputPattern:   DefaultOutputPattern,
		Gap:             DefaultGap,
		GapUnit:         DefaultGapUnit,
		Align:           DefaultAlign,
		OnConflict:      DefaultOnConflict,
		Verbose:         DefaultVerbose,
		Quiet:           DefaultQuiet,
		BookmarkPattern: DefaultBookmarkPattern,
	}
}

// Merge はCLIフラグで指定された値を Config に上書きマージする。
//
// フラグの空文字列/false は「未指定」と解釈し、Config の値を上書きしない。
// Output, Gap, Force は cmd/ で別途処理するため、Merge では取り扱わない。
func (c *Config) Merge(flags CLIFlags) {
	if flags.Align != "" {
		c.Align = flags.Align
	}
	if flags.Verbose {
		c.Verbose = true
	}
	if flags.Quiet {
		c.Quiet = true
	}
}

// generateConfig はデフォルト設定で config.toml を自動生成する。
// 書き込みに失敗した場合は警告ログを出力する（処理は続行）。
func generateConfig(path string) {
	err := os.WriteFile(path, []byte(defaultConfigTemplate), 0644)
	if err != nil {
		slog.Warn("failed to generate config file", "error", err, "path", path)
	}
}

// validateAndFallback は Config の各フィールドを検証し、不正値をデフォルト値にフォールバックする。
// フォールバック時は警告ログを出力する。
func validateAndFallback(cfg *Config) {
	if cfg.OutputPattern == "" {
		logFallback("output_pattern", cfg.OutputPattern, DefaultOutputPattern)
		cfg.OutputPattern = DefaultOutputPattern
	}

	if cfg.Gap < 0 {
		logFallback("gap", cfg.Gap, DefaultGap)
		cfg.Gap = DefaultGap
	}

	if !isValidGapUnit(cfg.GapUnit) {
		logFallback("gap_unit", cfg.GapUnit, DefaultGapUnit)
		cfg.GapUnit = DefaultGapUnit
	}

	if !isValidAlign(cfg.Align) {
		logFallback("align", cfg.Align, DefaultAlign)
		cfg.Align = DefaultAlign
	}

	if !isValidOnConflict(cfg.OnConflict) {
		logFallback("on_conflict", cfg.OnConflict, DefaultOnConflict)
		cfg.OnConflict = DefaultOnConflict
	}

	if cfg.BookmarkPattern == "" {
		logFallback("bookmark_pattern", cfg.BookmarkPattern, DefaultBookmarkPattern)
		cfg.BookmarkPattern = DefaultBookmarkPattern
	}
}

// logFallback は不正な設定値のフォールバックを警告ログに記録する。
func logFallback(key string, value any, defaultValue any) {
	slog.Warn("invalid config value, using default",
		"key", key, "value", value, "default", defaultValue)
}

// isValidGapUnit はガター単位が有効な値であるかを判定する。
func isValidGapUnit(u string) bool {
	return u == "pt" || u == "mm"
}

// isValidAlign はページ配置方針が有効な値であるかを判定する。
func isValidAlign(a string) bool {
	return a == "left" || a == "center" || a == "right"
}

// isValidOnConflict はファイル衝突ポリシーが有効な値であるかを判定する。
func isValidOnConflict(c string) bool {
	return c == "numbering" || c == "overwrite"
}
