package apperror

// ExitError はプロセス終了コードを伴うアプリケーションエラーである。
// 内部パッケージ（validate, pdf, config等）で生成し、cmd/ で os.Exit(code) に変換する。
//
// ExitError はラップの終端である。生成後は fmt.Errorf で再ラップせず、
// 上位層はそのまま return すること。
//
// 使用例:
//
//	// モジュール内での ExitError 生成:
//	return apperror.NewInputFileError(
//	    fmt.Sprintf("input file not found: %s", path), err)
//
//	// cmd/ での ExitError 判定:
//	var exitErr *apperror.ExitError
//	if errors.As(err, &exitErr) {
//	    slog.Error(exitErr.Message)
//	    os.Exit(exitErr.Code)
//	}
//	slog.Error(err.Error())
//	os.Exit(ExitGeneral)
type ExitError struct {
	Code    int    // 終了コード（0〜4）。codes.go の定数を参照
	Message string // ユーザー向けエラーメッセージ
	Err     error  // 原因エラー（nilの場合あり）
}

// Error はユーザー向けエラーメッセージを返す。
// error インターフェースを満たす。
func (e *ExitError) Error() string {
	return e.Message
}

// Unwrap は原因エラーを返す。
// errors.Is / errors.As によるチェーン探索を可能にする。
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewGeneralError は ExitGeneral（コード1）の ExitError を生成する。
// 引数不正、オプション競合、内部処理失敗等に使用する。
//
// 使用例:
//
//	return apperror.NewGeneralError(
//	    "--verbose and --quiet cannot be used together", nil)
func NewGeneralError(msg string, err error) *ExitError {
	return &ExitError{Code: ExitGeneral, Message: msg, Err: err}
}

// NewInputFileError は ExitInputFile（コード2）の ExitError を生成する。
// ファイル未存在、権限不足、不正PDF、マジックバイト不一致等に使用する。
//
// 使用例:
//
//	return apperror.NewInputFileError(
//	    fmt.Sprintf("input file not found: %s", path), err)
func NewInputFileError(msg string, err error) *ExitError {
	return &ExitError{Code: ExitInputFile, Message: msg, Err: err}
}

// NewOutputFileError は ExitOutputFile（コード3）の ExitError を生成する。
// 書き込み失敗、ディレクトリ未存在、リネーム失敗等に使用する。
//
// 使用例:
//
//	return apperror.NewOutputFileError(
//	    fmt.Sprintf("failed to write output PDF: %s", path), err)
func NewOutputFileError(msg string, err error) *ExitError {
	return &ExitError{Code: ExitOutputFile, Message: msg, Err: err}
}

// NewPasswordError は ExitPassword（コード4）の ExitError を生成する。
// パスワード不正、パスワード入力失敗、空パスワード等に使用する。
//
// 使用例:
//
//	return apperror.NewPasswordError(
//	    fmt.Sprintf("incorrect password for: %s", path), err)
func NewPasswordError(msg string, err error) *ExitError {
	return &ExitError{Code: ExitPassword, Message: msg, Err: err}
}
