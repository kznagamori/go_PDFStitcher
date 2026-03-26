# CLAUDE.md

## プロジェクト概要

go_PDFStitcher は、単一PDFの全ページを縦方向に連結し、1枚の縦長PDFとして出力するCLIツール。
Go + cobra + pdfcpu で実装。個人開発、AIエージェントが実装の主力（人がレビュー）。

## 技術スタック

- **言語**: Go（最新安定版）
- **CLI**: github.com/spf13/cobra
- **PDF操作**: github.com/pdfcpu/pdfcpu
- **設定ファイル**: github.com/BurntSushi/toml（TOML形式）
- **パスワード入力**: golang.org/x/term
- **ログ**: log/slog（標準ライブラリ）
- **対応OS**: Windows / macOS / Linux
- **ライセンス**: MIT

## コーディング必須ルール

以下は必ず遵守すること。詳細・コード例は各参照先を確認。

**型安全性:**
- [1.1] 暗黙の型変換を避け、明示的な変換を使用する
- [1.3] interface{} / any の使用を制限する（型アサーションまたは型スイッチでガード必須）

**制御フロー:**
- [2.1] goto の使用を禁止する
- [2.2] ネストの深さを3段階以内に制限する
- [2.4] ループ制御変数をループ内で再代入しない

**エラーハンドリング:**
- [3.1] エラーを握りつぶさない（`_` でエラーを捨てない）
- [3.2] エラーはラップして文脈を付加する（`fmt.Errorf("...: %w", err)`）
- [3.3] panic を使用しない（初期化エラーのみ例外）

**メモリ・リソース管理:**
- [4.1] リソースは取得直後に defer で解放する
- [4.2] 一時ファイルは異常終了時にクリーンアップする
- [4.3] ループ内でリソースを取得する場合は即座に解放する（ループ内 defer 禁止）

**命名規則:**
- [5.1] Go標準の命名規則に従う（エクスポート=PascalCase、非エクスポート=camelCase）
- [5.3] ファイル名は小文字のスネークケースを使用する
- [5.4] パッケージ名は短く単数形にする（util/common/helper 禁止）

**関数の戻り値型:**
- [7.1] エラーを返す関数は (結果, error) パターンを使用する

**セキュリティ:**
- [9.1] ユーザー入力を検証する（パスサニタイズ、拡張子、マジックバイト）
- [9.2] ハードコードされた秘密情報を禁止する
- [9.3] ファイルパスをサニタイズする（filepath.Clean）

**並行処理:**
- [10.2] goroutine使用時はリークを防止する（context.Cancel / チャネルクローズ必須）

**禁止パターン:**
- [11.1] init() 関数の使用を禁止する
- [11.2] グローバル変数の使用を禁止する（const は許可）
- [11.3] マジックナンバーを禁止する（名前付き定数を使用）
- [11.4] 型アサーションは必ず2値形式（val, ok := v.(T)）を使用する

**ログ管理:**
- [13.1] log/slog を唯一のログ出力手段とする（fmt.Println / log.Printf 禁止）
- [13.2] ログレベルを適切に使い分ける（ERROR/WARN/INFO/DEBUG）
- [13.4] ログに秘密情報を含めない（パスワード、認証情報）

**テスタビリティ・DI:**
- [14.1] 外部依存はインターフェースで抽象化する
- [14.2] コンストラクタ関数で依存を注入する（構造体内部での生成禁止）
- [14.3] テスト困難なパターンを禁止する（os.Exit はコアロジック内で呼ばない、グローバル状態依存禁止）

**コメント・ドキュメンテーション:**
- [15.1] エクスポートされる識別子にはGoDocコメントを付ける
- [15.5] コメントは日本語で記述する（ドキュメントコメント、インラインコメント、パッケージコメントすべて）
- [15.6] コメントアウトされたコードを残さない（git で復元可能）

> 全ルール詳細: ./.ai/docs/specification/coding_standards.md

## ディレクトリ構成

```
go_PDFStitcher/
├── main.go                 # エントリポイント（cmd.Execute() のみ）
├── cmd/                    # CLI層（cobra コマンド定義、フラグ解析、DI組み立て）
├── internal/
│   ├── model/              # 共有データ型（Page, Bookmark, StitchResult）
│   ├── apperror/           # ExitError型、終了コード定数（0〜4）
│   ├── config/             # TOML設定ファイル管理
│   ├── pdf/                # pdfcpuラッパー（PDF読み書き）— pdfcpu依存はここに閉じ込め
│   ├── stitcher/           # コアロジック（縦連結、レイアウト計算、幅揃え）
│   ├── bookmark/           # しおり処理（座標調整、自動生成）
│   ├── output/             # 出力パス解決、ファイル衝突処理
│   ├── validate/           # 入力バリデーション
│   ├── unit/               # pt/mm単位変換
│   └── prompt/             # 対話入力（パスワード、Y/N確認）
├── config.toml             # デフォルト設定ファイル
└── testdata/               # テストデータ（PDF、設定ファイル）
```

## モジュールと仕様書のマッピング

| 実装パッケージ | 仕様書 |
|---------------|--------|
| cmd/ | .ai/docs/modules/cmd.md |
| internal/model | .ai/docs/modules/model.md |
| internal/apperror | .ai/docs/modules/apperror.md |
| internal/config | .ai/docs/modules/config.md |
| internal/pdf | .ai/docs/modules/pdf.md |
| internal/stitcher | .ai/docs/modules/stitcher.md |
| internal/bookmark | .ai/docs/modules/bookmark.md |
| internal/output | .ai/docs/modules/output.md |
| internal/validate | .ai/docs/modules/validate.md |
| internal/unit | .ai/docs/modules/unit.md |
| internal/prompt | .ai/docs/modules/prompt.md |

> 全マッピング: .ai/docs/modules/module_mapping.md

## エラーハンドリング方針

- **ExitError型**: 終了コード（0〜4）を持つカスタムエラー。コアロジック内で生成、cmd/ で os.Exit に変換
- **終了コード**: 0=正常, 1=一般, 2=入力ファイル, 3=出力ファイル, 4=パスワード
- **ExitError はラップの終端**: 生成後は fmt.Errorf で再ラップしない。上位層はそのまま return
- **os.Exit は cmd/ のみ**: コアロジック内で os.Exit を呼ばない
- **警告で続行**: 設定ファイル不正値、しおり抽出失敗、PDF仕様上限超過の3ケース
- **上書き拒否は正常終了**: ErrUserCancelled → 終了コード0

> 詳細: .ai/docs/error_design.md

## ビルド・テスト・実行コマンド

```bash
# ビルド
go build -o go_PDFStitcher .

# バージョン付きビルド
go build -ldflags "-X cmd.version=1.0.0 -X cmd.commit=$(git rev-parse --short HEAD) -X cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o go_PDFStitcher .

# 全テスト実行
go test ./...

# カバレッジ付き
go test -cover ./...

# race detector 付き
go test -race ./...

# 短縮モード（E2Eスキップ）
go test -short ./...

# ベンチマーク
go test -bench=. ./internal/stitcher/...

# 実行
./go_PDFStitcher input.pdf
./go_PDFStitcher input.pdf -o output.pdf --gap 10mm --align center
```

## 開発時の注意事項・禁止事項

- **pdfcpu 依存の局所化**: internal/pdf のみが pdfcpu をimport。他パッケージはインターフェース経由
- **循環依存禁止**: 依存は常に上位→下位の一方向。共有型は model パッケージに集約
- **パスワードをログに出力しない**: NFR-02 セキュリティ要件
- **入力PDFは読み取り専用**: 変更・破損しない（NFR-03）
- **一時ファイルは必ずクリーンアップ**: 異常終了時にも残さない（NFR-03）
- **テストはテーブル駆動**: 同一関数の複数ケースはテーブル駆動テストで記述
- **コアロジックはCLI非依存**: 将来のWails GUI化に対応するため、internal/ はCLI固有のコードを含まない

## 実装時の参照ドキュメント

実装・コードレビュー時は以下を必ず確認すること：

- 仕様書インデックス: ./.ai/docs/specification.md
- コーディング標準（全ルール詳細）: ./.ai/docs/specification/coding_standards.md
- 機能要件・非機能要件: ./.ai/docs/specification/requirements.md
- 制約事項・前提条件: ./.ai/docs/specification/constraints.md
- アーキテクチャ設計: ./.ai/docs/architecture.md
- エラー設計: ./.ai/docs/error_design.md
- インターフェース定義: ./.ai/docs/interfaces.md
- テスト戦略: ./.ai/docs/test_strategy.md
- モジュール仕様: ./.ai/docs/modules/[対象モジュール名].md
- モジュールマッピング: ./.ai/docs/modules/module_mapping.md
