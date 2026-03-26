// Package pdf は pdfcpu ライブラリのラッパーとして PDF 読み書きを提供する。
// pdfcpu への依存をこのパッケージに閉じ込め、他パッケージはインターフェース経由でアクセスする。
//
// Reader は PDF 読み込みとページ情報・しおり抽出を、
// Writer は連結結果の安全な出力（一時ファイル→リネーム方式）を担当する。
package pdf

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdfModel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// PasswordPrompter はパスワード入力を抽象化するインターフェース。
// prompt.Prompter がこのインターフェースを暗黙的に満たす。
type PasswordPrompter interface {
	// Password はパスワード入力を求める（エコーバック無効）。
	Password(message string) (string, error)
}

// pageContent は pdfcpu 固有のページコンテンツ参照。
// model.Page.Content フィールドに格納し、Writer が解釈する。
type pageContent struct {
	ctx     *pdfModel.Context // 元PDFの pdfcpu コンテキスト
	pageNum int               // pdfcpu 内のページ番号（1始まり）
}

// Reader は pdfcpu を使った PDF 読み込みの実装である。
// stitcher.PageReader インターフェースを満たす。
type Reader struct {
	prompter PasswordPrompter
}

// NewReader は Reader を生成する。
// prompter はパスワード保護PDF検出時の対話入力に使用する。
func NewReader(prompter PasswordPrompter) *Reader {
	return &Reader{prompter: prompter}
}

// ReadPages はPDFファイルを読み込み、全ページ情報を返す。
// パスワード保護PDFの場合、PasswordPrompter を使って対話入力する。
//
// エラーケース:
//   - PDF読み込み失敗: ExitError（コード2）
//   - パスワード不正: ExitError（コード4）
//   - ページ数ゼロ: ExitError（コード2）
func (r *Reader) ReadPages(path string) ([]model.Page, error) {
	ctx, err := r.openPDF(path)
	if err != nil {
		return nil, err
	}

	if ctx.PageCount == 0 {
		return nil, apperror.NewInputFileError(
			fmt.Sprintf("PDF has no pages: %s", path), nil)
	}

	dims, err := ctx.PageDims()
	if err != nil {
		return nil, apperror.NewInputFileError(
			fmt.Sprintf("failed to read PDF: %s", path), err)
	}

	pages := make([]model.Page, ctx.PageCount)
	for i := 0; i < ctx.PageCount; i++ {
		pages[i] = model.Page{
			Index:   i,
			Width:   dims[i].Width,
			Height:  dims[i].Height,
			Content: &pageContent{ctx: ctx, pageNum: i + 1},
		}
	}

	slog.Info("PDF loaded", "path", path, "pages", ctx.PageCount)
	return pages, nil
}

// ReadBookmarks はPDFから既存しおり情報を読み取る。
// しおりが存在しない場合は空スライスを返す（エラーではない）。
// 抽出に失敗した場合は警告ログを出力し空スライスを返す。
func (r *Reader) ReadBookmarks(path string) ([]model.Bookmark, error) {
	ctx, err := r.openPDF(path)
	if err != nil {
		return nil, err
	}

	pdfBookmarks, err := pdfcpu.Bookmarks(ctx)
	if err != nil {
		slog.Warn("failed to extract bookmarks, generating page-based bookmarks",
			"error", err, "path", path)
		return nil, nil
	}

	if len(pdfBookmarks) == 0 {
		return nil, nil
	}

	bookmarks := convertFromPDFBookmarks(pdfBookmarks, 0)
	slog.Info("bookmarks extracted", "path", path, "count", len(bookmarks))
	return bookmarks, nil
}

// openPDF はPDFファイルを読み込む。パスワード保護PDFの場合は対話入力で再試行する。
func (r *Reader) openPDF(path string) (*pdfModel.Context, error) {
	ctx, err := api.ReadContextFile(path)
	if err == nil {
		return ctx, nil
	}

	if !isPasswordRequired(err) {
		return nil, apperror.NewInputFileError(
			fmt.Sprintf("failed to read PDF: %s", path), err)
	}

	// パスワード保護PDF: 対話入力で再試行
	password, promptErr := r.prompter.Password("Enter PDF password: ")
	if promptErr != nil {
		return nil, apperror.NewPasswordError(
			fmt.Sprintf("failed to read password for: %s", path), promptErr)
	}

	ctx, err = readWithPassword(path, password)
	if err != nil {
		return nil, apperror.NewPasswordError(
			fmt.Sprintf("incorrect password for: %s", path), err)
	}

	return ctx, nil
}

// readWithPassword はパスワード付きでPDFを読み込む。
func readWithPassword(path string, password string) (*pdfModel.Context, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := pdfModel.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password

	ctx, err := pdfModel.NewContext(f, conf)
	if err != nil {
		return nil, err
	}

	if err := api.ValidateContext(ctx); err != nil {
		return nil, err
	}

	if err := api.OptimizeContext(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

// isPasswordRequired は pdfcpu のエラーがパスワード要求であるかを判定する。
func isPasswordRequired(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") || strings.Contains(msg, "encrypt")
}

// convertFromPDFBookmarks は pdfcpu のしおりを model.Bookmark に変換する。
func convertFromPDFBookmarks(pdfBMs []pdfcpu.Bookmark, level int) []model.Bookmark {
	result := make([]model.Bookmark, len(pdfBMs))
	for i, bm := range pdfBMs {
		result[i] = model.Bookmark{
			Title:   bm.Title,
			Level:   level,
			PageIdx: bm.PageFrom - 1, // pdfcpu は1始まり → 0始まりに変換
			Y:       0,               // 元PDFのページ内Y座標（pdfcpu では取得不可のため0）
		}
		if len(bm.Kids) > 0 {
			result[i].Children = convertFromPDFBookmarks(bm.Kids, level+1)
		}
	}
	return result
}
