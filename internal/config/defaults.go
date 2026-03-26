// Package config は TOML 設定ファイルの読み込み・自動生成・CLIフラグとのマージを提供する。
//
// config.toml は実行ファイルと同ディレクトリに配置する。
// ファイルが存在しない場合はデフォルト値で自動生成し、不正な値は警告を出力して
// デフォルト値にフォールバックする。
//
// CLIオプションが明示的に指定された場合は設定ファイルの値より優先される。
package config

// 設定ファイルの全フィールドのデフォルト値。
const (
	// DefaultOutputPattern は出力ファイル名パターンのデフォルト値。
	// {input} は入力ファイル名（拡張子なし）に置換される。
	DefaultOutputPattern = "{input}_stitched.pdf"

	// DefaultGap はページ間余白（ガター）のデフォルト値（単位は GapUnit に従う）。
	DefaultGap = 0.0

	// DefaultGapUnit はガター単位のデフォルト値。
	DefaultGapUnit = "pt"

	// DefaultAlign はページ配置方針のデフォルト値。
	DefaultAlign = "left"

	// DefaultOnConflict はファイル衝突時の挙動のデフォルト値。
	DefaultOnConflict = "numbering"

	// DefaultVerbose は詳細ログ出力のデフォルト値。
	DefaultVerbose = false

	// DefaultQuiet はログ抑制のデフォルト値。
	DefaultQuiet = false

	// DefaultBookmarkPattern はしおり自動生成時のテキストパターンのデフォルト値。
	// {n} はページ番号（1始まり）に置換される。
	DefaultBookmarkPattern = "Page {n}"
)

// configFileName は設定ファイル名。
const configFileName = "config.toml"

// defaultConfigTemplate は自動生成する config.toml のテンプレート。
const defaultConfigTemplate = `# go_PDFStitcher 設定ファイル
# CLIオプションのデフォルト値を管理します。
# CLIオプションが明示的に指定された場合はこの設定より優先されます。

# 出力ファイル名パターン
# {input} は入力ファイル名（拡張子なし）に置換されます
output_pattern = "{input}_stitched.pdf"

# ページ間余白（ガター）の値
gap = 0

# ガター単位 ("pt" or "mm")
gap_unit = "pt"

# ページ配置方針 ("left", "center", "right")
align = "left"

# ファイル衝突時の挙動 ("numbering" or "overwrite")
on_conflict = "numbering"

# 詳細ログ出力
verbose = false

# ログ抑制（エラーのみ出力）
quiet = false

# しおり自動生成時のテキストパターン
# {n} はページ番号（1始まり）に置換されます
bookmark_pattern = "Page {n}"
`
