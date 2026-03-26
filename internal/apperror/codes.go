// Package apperror はアプリケーション共通のエラー型と終了コード定数を定義する。
// ExitError が主要なエラー型であり、プロセスの終了コードを保持する。
// コアロジックパッケージで生成し、cmd/ で os.Exit に変換する。
package apperror

// アプリケーションの終了コード定数（UC-09準拠）。
// ExitError の Code フィールドおよびプロセス終了コードとして使用する。
const (
	// ExitOK は正常終了を示す。
	ExitOK = 0

	// ExitGeneral は一般エラーを示す。
	// 引数不正、オプション競合、設定ファイル読み込み失敗等。
	ExitGeneral = 1

	// ExitInputFile は入力ファイルエラーを示す。
	// ファイル未存在、権限不足、不正PDF、マジックバイト不一致等。
	ExitInputFile = 2

	// ExitOutputFile は出力ファイルエラーを示す。
	// 書き込み不可、ディレクトリ未存在、リネーム失敗等。
	ExitOutputFile = 3

	// ExitPassword はパスワード関連エラーを示す。
	// パスワード不正、パスワード入力失敗等。
	ExitPassword = 4
)
