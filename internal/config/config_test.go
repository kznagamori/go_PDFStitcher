package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.OutputPattern != DefaultOutputPattern {
		t.Errorf("OutputPattern = %q, 期待値 %q", cfg.OutputPattern, DefaultOutputPattern)
	}
	if cfg.Gap != DefaultGap {
		t.Errorf("Gap = %f, 期待値 %f", cfg.Gap, DefaultGap)
	}
	if cfg.GapUnit != DefaultGapUnit {
		t.Errorf("GapUnit = %q, 期待値 %q", cfg.GapUnit, DefaultGapUnit)
	}
	if cfg.Align != DefaultAlign {
		t.Errorf("Align = %q, 期待値 %q", cfg.Align, DefaultAlign)
	}
	if cfg.OnConflict != DefaultOnConflict {
		t.Errorf("OnConflict = %q, 期待値 %q", cfg.OnConflict, DefaultOnConflict)
	}
	if cfg.Verbose != DefaultVerbose {
		t.Errorf("Verbose = %v, 期待値 %v", cfg.Verbose, DefaultVerbose)
	}
	if cfg.Quiet != DefaultQuiet {
		t.Errorf("Quiet = %v, 期待値 %v", cfg.Quiet, DefaultQuiet)
	}
	if cfg.BookmarkPattern != DefaultBookmarkPattern {
		t.Errorf("BookmarkPattern = %q, 期待値 %q", cfg.BookmarkPattern, DefaultBookmarkPattern)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		toml    string // 設定ファイルの内容（空文字ならファイルを作成しない）
		create  bool   // true: toml の内容でファイルを作成する
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name:   "有効な設定ファイル（全フィールド指定）",
			create: true,
			toml: `output_pattern = "{input}_merged.pdf"
gap = 10.5
gap_unit = "mm"
align = "center"
on_conflict = "overwrite"
verbose = true
quiet = false
bookmark_pattern = "Section {n}"
`,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OutputPattern != "{input}_merged.pdf" {
					t.Errorf("OutputPattern = %q", cfg.OutputPattern)
				}
				if cfg.Gap != 10.5 {
					t.Errorf("Gap = %f", cfg.Gap)
				}
				if cfg.GapUnit != "mm" {
					t.Errorf("GapUnit = %q", cfg.GapUnit)
				}
				if cfg.Align != "center" {
					t.Errorf("Align = %q", cfg.Align)
				}
				if cfg.OnConflict != "overwrite" {
					t.Errorf("OnConflict = %q", cfg.OnConflict)
				}
				if !cfg.Verbose {
					t.Error("Verbose should be true")
				}
				if cfg.Quiet {
					t.Error("Quiet should be false")
				}
				if cfg.BookmarkPattern != "Section {n}" {
					t.Errorf("BookmarkPattern = %q", cfg.BookmarkPattern)
				}
			},
		},
		{
			name:   "部分的な設定ファイル（未指定フィールドはデフォルト値）",
			create: true,
			toml:   `align = "right"` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != "right" {
					t.Errorf("Align = %q, 期待値 %q", cfg.Align, "right")
				}
				// 未指定フィールドはデフォルト値
				if cfg.OutputPattern != DefaultOutputPattern {
					t.Errorf("OutputPattern = %q, 期待値 %q", cfg.OutputPattern, DefaultOutputPattern)
				}
				if cfg.Gap != DefaultGap {
					t.Errorf("Gap = %f, 期待値 %f", cfg.Gap, DefaultGap)
				}
				if cfg.BookmarkPattern != DefaultBookmarkPattern {
					t.Errorf("BookmarkPattern = %q, 期待値 %q", cfg.BookmarkPattern, DefaultBookmarkPattern)
				}
			},
		},
		{
			name:   "ファイル未存在（自動生成＋デフォルト値返却）",
			create: false,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				assertDefaultConfig(t, cfg)
			},
		},
		{
			name:   "空ファイル（全フィールドがデフォルト値）",
			create: true,
			toml:   "",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				assertDefaultConfig(t, cfg)
			},
		},
		{
			name:   "TOML構文エラー（警告＋デフォルト値返却）",
			create: true,
			toml:   "this is not valid TOML [[[",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				assertDefaultConfig(t, cfg)
			},
		},
		{
			name:   "不正な align 値（フォールバック）",
			create: true,
			toml:   `align = "xxx"` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != DefaultAlign {
					t.Errorf("Align = %q, フォールバック後は %q であるべき", cfg.Align, DefaultAlign)
				}
			},
		},
		{
			name:   "不正な gap_unit 値（フォールバック）",
			create: true,
			toml:   `gap_unit = "inch"` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.GapUnit != DefaultGapUnit {
					t.Errorf("GapUnit = %q, フォールバック後は %q であるべき", cfg.GapUnit, DefaultGapUnit)
				}
			},
		},
		{
			name:   "不正な on_conflict 値（フォールバック）",
			create: true,
			toml:   `on_conflict = "delete"` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OnConflict != DefaultOnConflict {
					t.Errorf("OnConflict = %q, フォールバック後は %q であるべき", cfg.OnConflict, DefaultOnConflict)
				}
			},
		},
		{
			name:   "負の gap 値（フォールバック）",
			create: true,
			toml:   `gap = -5.0` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Gap != DefaultGap {
					t.Errorf("Gap = %f, フォールバック後は %f であるべき", cfg.Gap, DefaultGap)
				}
			},
		},
		{
			name:   "空の output_pattern（フォールバック）",
			create: true,
			toml:   `output_pattern = ""` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OutputPattern != DefaultOutputPattern {
					t.Errorf("OutputPattern = %q, フォールバック後は %q であるべき", cfg.OutputPattern, DefaultOutputPattern)
				}
			},
		},
		{
			name:   "空の bookmark_pattern（フォールバック）",
			create: true,
			toml:   `bookmark_pattern = ""` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.BookmarkPattern != DefaultBookmarkPattern {
					t.Errorf("BookmarkPattern = %q, フォールバック後は %q であるべき", cfg.BookmarkPattern, DefaultBookmarkPattern)
				}
			},
		},
		{
			name:   "ゼロの gap 値（有効値として保持）",
			create: true,
			toml:   `gap = 0` + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Gap != 0.0 {
					t.Errorf("Gap = %f, 0.0 が有効値として保持されるべき", cfg.Gap)
				}
			},
		},
		{
			name:   "未知のキーは無視される",
			create: true,
			toml: `align = "center"
unknown_key = "value"
`,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != "center" {
					t.Errorf("Align = %q, 期待値 %q", cfg.Align, "center")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.create {
				configPath := filepath.Join(dir, configFileName)
				if err := os.WriteFile(configPath, []byte(tt.toml), 0644); err != nil {
					t.Fatalf("テスト用設定ファイル作成失敗: %v", err)
				}
			}

			cfg, err := Load(dir)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Load() はエラーを返すべき")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() で予期しないエラー: %v", err)
			}

			if cfg == nil {
				t.Fatal("Load() は非nil の *Config を返すべき")
			}

			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadAutoGeneration(t *testing.T) {
	// ファイル未存在時に config.toml が自動生成されることを検証する
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() で予期しないエラー: %v", err)
	}

	assertDefaultConfig(t, cfg)

	// config.toml が生成されていることを確認する
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.toml が自動生成されていない")
	}
}

func TestLoadReadError(t *testing.T) {
	// 読み取り不可時に ExitError（コード1）を返すことを検証する
	// ディレクトリを config.toml の名前で作成し、読み取りエラーを発生させる
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)
	if err := os.Mkdir(configPath, 0755); err != nil {
		t.Fatalf("テスト用ディレクトリ作成失敗: %v", err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() はエラーを返すべき")
	}

	var exitErr *apperror.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("エラーが *apperror.ExitError ではない: %T: %v", err, err)
	}
	if exitErr.Code != apperror.ExitGeneral {
		t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, apperror.ExitGeneral)
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name  string
		flags CLIFlags
		check func(t *testing.T, cfg *Config)
	}{
		{
			name: "Alignフラグが上書きされる",
			flags: CLIFlags{
				Align: "center",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != "center" {
					t.Errorf("Align = %q, 期待値 %q", cfg.Align, "center")
				}
			},
		},
		{
			name: "Verboseフラグがtrueで上書きされる",
			flags: CLIFlags{
				Verbose: true,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.Verbose {
					t.Error("Verbose は true であるべき")
				}
			},
		},
		{
			name: "Quietフラグがtrueで上書きされる",
			flags: CLIFlags{
				Quiet: true,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.Quiet {
					t.Error("Quiet は true であるべき")
				}
			},
		},
		{
			name:  "全フラグ未指定ではConfig値が維持される",
			flags: CLIFlags{},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				assertDefaultConfig(t, cfg)
			},
		},
		{
			name: "Align空文字は未指定扱いで上書きしない",
			flags: CLIFlags{
				Align: "",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != DefaultAlign {
					t.Errorf("Align = %q, 期待値 %q", cfg.Align, DefaultAlign)
				}
			},
		},
		{
			name: "Verbose=falseは未指定扱いで上書きしない",
			flags: CLIFlags{
				Verbose: false,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Verbose != DefaultVerbose {
					t.Errorf("Verbose = %v, 期待値 %v", cfg.Verbose, DefaultVerbose)
				}
			},
		},
		{
			name: "複数フラグの同時指定",
			flags: CLIFlags{
				Align:   "right",
				Verbose: true,
				Quiet:   true,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.Align != "right" {
					t.Errorf("Align = %q, 期待値 %q", cfg.Align, "right")
				}
				if !cfg.Verbose {
					t.Error("Verbose は true であるべき")
				}
				if !cfg.Quiet {
					t.Error("Quiet は true であるべき")
				}
				// 他のフィールドはデフォルト値を維持
				if cfg.OutputPattern != DefaultOutputPattern {
					t.Errorf("OutputPattern = %q, 期待値 %q", cfg.OutputPattern, DefaultOutputPattern)
				}
			},
		},
		{
			name: "Output/Gap/Forceフラグは無視される",
			flags: CLIFlags{
				Output: "custom.pdf",
				Gap:    "10mm",
				Force:  true,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				// Output/Gap/Force は Merge で処理しないため、Config は変更されない
				assertDefaultConfig(t, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Merge(tt.flags)
			tt.check(t, cfg)
		})
	}
}

func TestMergePreservesConfigFileValues(t *testing.T) {
	// 設定ファイルの値が Merge で未指定のフラグにより上書きされないことを検証する
	cfg := &Config{
		OutputPattern:   "custom_pattern.pdf",
		Gap:             5.0,
		GapUnit:         "mm",
		Align:           "center",
		OnConflict:      "overwrite",
		Verbose:         true,
		Quiet:           false,
		BookmarkPattern: "Chapter {n}",
	}

	// Verbose=false は「未指定」扱い。設定ファイルの true を維持すべき
	cfg.Merge(CLIFlags{})

	if cfg.Align != "center" {
		t.Errorf("Align = %q, 設定ファイルの値 %q が維持されるべき", cfg.Align, "center")
	}
	if !cfg.Verbose {
		t.Error("Verbose は設定ファイルの true が維持されるべき")
	}
	if cfg.Gap != 5.0 {
		t.Errorf("Gap = %f, 設定ファイルの値 5.0 が維持されるべき", cfg.Gap)
	}
}

// assertDefaultConfig は Config の全フィールドがデフォルト値であることを検証する。
func assertDefaultConfig(t *testing.T, cfg *Config) {
	t.Helper()
	d := Default()
	if cfg.OutputPattern != d.OutputPattern {
		t.Errorf("OutputPattern = %q, 期待値 %q", cfg.OutputPattern, d.OutputPattern)
	}
	if cfg.Gap != d.Gap {
		t.Errorf("Gap = %f, 期待値 %f", cfg.Gap, d.Gap)
	}
	if cfg.GapUnit != d.GapUnit {
		t.Errorf("GapUnit = %q, 期待値 %q", cfg.GapUnit, d.GapUnit)
	}
	if cfg.Align != d.Align {
		t.Errorf("Align = %q, 期待値 %q", cfg.Align, d.Align)
	}
	if cfg.OnConflict != d.OnConflict {
		t.Errorf("OnConflict = %q, 期待値 %q", cfg.OnConflict, d.OnConflict)
	}
	if cfg.Verbose != d.Verbose {
		t.Errorf("Verbose = %v, 期待値 %v", cfg.Verbose, d.Verbose)
	}
	if cfg.Quiet != d.Quiet {
		t.Errorf("Quiet = %v, 期待値 %v", cfg.Quiet, d.Quiet)
	}
	if cfg.BookmarkPattern != d.BookmarkPattern {
		t.Errorf("BookmarkPattern = %q, 期待値 %q", cfg.BookmarkPattern, d.BookmarkPattern)
	}
}
