// Package stitcher はPDF縦連結処理のコアロジックを提供する。
//
// Stitcher が全体フローをオーケストレーションし、
// PageReader / PageWriter / BookmarkProcessor の各インターフェース経由で
// PDF読み書きとしおり処理を実行する。
//
// レイアウト計算（CalculateLayout）とページ幅揃え（AlignPages）は
// PDF / しおりに依存しない純粋な計算処理である。
package stitcher

import (
	"log/slog"

	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// PageReader はPDFからページ情報を読み取るインターフェース。
// pdf.Reader がこのインターフェースを実装する。
type PageReader interface {
	// ReadPages はPDFファイルを読み込み、全ページ情報を返す。
	ReadPages(path string) ([]model.Page, error)

	// ReadBookmarks はPDFから既存しおり情報を読み取る。
	// しおりが存在しない場合は nil を返す（エラーではない）。
	ReadBookmarks(path string) ([]model.Bookmark, error)
}

// PageWriter は連結結果のPDFを書き出すインターフェース。
// pdf.Writer がこのインターフェースを実装する。
type PageWriter interface {
	// Write は連結済みの1ページPDFを出力ファイルに書き込む。
	Write(outputPath string, result *model.StitchResult) error

	// Cleanup は一時ファイルを削除する（シグナルハンドリング用）。
	Cleanup()
}

// BookmarkProcessor はしおりの処理を行うインターフェース。
// bookmark.Processor がこのインターフェースを実装する。
type BookmarkProcessor interface {
	// Adjust は既存しおりのY座標を連結後の位置に調整する。
	Adjust(bookmarks []model.Bookmark, pageOffsets []float64) []model.Bookmark

	// Generate はページ番号ベースのしおりを自動生成する。
	Generate(pages []model.Page, pageOffsets []float64, pattern string) []model.Bookmark
}

// StitchOptions は連結処理のオプション。
type StitchOptions struct {
	GapPt           float64 // ガター幅（pt）
	Align           string  // ページ配置方針（"left", "center", "right"）
	BookmarkPattern string  // しおりテキストパターン（例: "Page {n}"）
}

// Stitcher はPDF縦連結処理のオーケストレーターである。
// 依存注入を受け取り、Run で全体フローを実行する。
type Stitcher struct {
	reader       PageReader
	writer       PageWriter
	bookmarkProc BookmarkProcessor
}

// NewStitcher は依存を注入して Stitcher を生成する。
func NewStitcher(reader PageReader, writer PageWriter, bookmarkProc BookmarkProcessor) *Stitcher {
	return &Stitcher{
		reader:       reader,
		writer:       writer,
		bookmarkProc: bookmarkProc,
	}
}

// Run は入力PDFを読み込み、全ページを縦連結して出力する。
//
// 処理フロー:
//  1. reader.ReadPages でページ情報を取得
//  2. reader.ReadBookmarks でしおり情報を取得
//  3. CalculateLayout でレイアウト計算（上限チェック含む）
//  4. AlignPages でページ幅揃え
//  5. しおり処理（既存しおりの座標調整 or 自動生成）
//  6. StitchResult を組み立て
//  7. writer.Write で出力
//
// 下位モジュールのエラーはそのまま伝播する（再ラップしない）。
func (s *Stitcher) Run(inputPath string, outputPath string, opts StitchOptions) error {
	// 1. ページ情報を読み取る
	pages, err := s.reader.ReadPages(inputPath)
	if err != nil {
		return err
	}

	// 2. しおり情報を読み取る
	bookmarks, err := s.reader.ReadBookmarks(inputPath)
	if err != nil {
		return err
	}

	// 3. レイアウト計算
	layout := CalculateLayout(pages, opts.GapPt)

	if layout.ExceedsLimit {
		slog.Warn("output height exceeds PDF spec limit",
			"height_pt", layout.TotalHeight, "limit_pt", maxPDFUserUnits)
	}

	slog.Debug("layout calculated",
		"total_height_pt", layout.TotalHeight,
		"max_width_pt", layout.MaxWidth,
		"pages", len(pages))

	// 4. ページ幅揃え
	alignedPages := AlignPages(pages, layout.MaxWidth, opts.Align)

	// 5. しおり処理
	var resultBookmarks []model.Bookmark
	if len(bookmarks) > 0 {
		resultBookmarks = s.bookmarkProc.Adjust(bookmarks, layout.PageOffsets)
	} else {
		resultBookmarks = s.bookmarkProc.Generate(pages, layout.PageOffsets, opts.BookmarkPattern)
	}

	// 6. StitchResult を組み立てる
	result := &model.StitchResult{
		AlignedPages: alignedPages,
		Bookmarks:    resultBookmarks,
		TotalHeight:  layout.TotalHeight,
		MaxWidth:     layout.MaxWidth,
		GapPt:        opts.GapPt,
	}

	// 7. 出力
	return s.writer.Write(outputPath, result)
}
