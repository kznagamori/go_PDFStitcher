// Package bookmark はPDFしおり（アウトライン）の処理を提供する。
// 既存しおりのY座標調整（階層構造保持）、またはページ番号ベースの
// しおり自動生成を担当する。
//
// Adjust / Generate は純粋な計算処理でありエラーを返さない。
// stitcher パッケージから BookmarkProcessor インターフェース経由で呼び出される。
package bookmark

import (
	"strconv"
	"strings"

	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// pageNumberPlaceholder はしおりテキストパターン中のページ番号プレースホルダー。
const pageNumberPlaceholder = "{n}"

// Processor は BookmarkProcessor インターフェースの実装である。
// 状態を持たない純粋な計算処理を提供する。
type Processor struct{}

// NewProcessor は Processor を生成する。
func NewProcessor() *Processor {
	return &Processor{}
}

// Adjust は既存しおりのY座標を連結後の位置に調整する。
//
// 各しおりの PageIdx から pageOffsets[PageIdx] を取得し、
// Y座標を pageOffsets[PageIdx] + 元のページ内Y座標 に調整する。
// Children を再帰的に処理し、階層構造をそのまま保持する。
//
// 入力のしおりスライスは変更せず、座標調整済みの新しいスライスを返す（イミュータブル）。
// pageOffsets の長さは元PDFのページ数と一致すること（呼び出し元の責務）。
func (p *Processor) Adjust(bookmarks []model.Bookmark, pageOffsets []float64) []model.Bookmark {
	if len(bookmarks) == 0 {
		return nil
	}

	result := make([]model.Bookmark, len(bookmarks))
	for i, bm := range bookmarks {
		result[i] = adjustBookmark(bm, pageOffsets)
	}
	return result
}

// Generate はページ番号ベースのしおりを自動生成する。
//
// 各ページに対して1つのしおりを生成する。
// pattern 中の {n} を1始まりのページ番号に置換する。
// Y座標は pageOffsets[i]（各ページの上端）、Level は 0（フラット構造）。
//
// pageOffsets の長さは pages の長さと一致すること（呼び出し元の責務）。
func (p *Processor) Generate(pages []model.Page, pageOffsets []float64, pattern string) []model.Bookmark {
	if len(pages) == 0 {
		return nil
	}

	result := make([]model.Bookmark, len(pages))
	for i, page := range pages {
		result[i] = model.Bookmark{
			Title:   replacePageNumber(pattern, i+1),
			Level:   0,
			PageIdx: page.Index,
			Y:       pageOffsets[i],
		}
	}
	return result
}

// adjustBookmark は単一しおりのY座標を調整する。
// Children がある場合は再帰的に処理する。
func adjustBookmark(bm model.Bookmark, pageOffsets []float64) model.Bookmark {
	adjusted := model.Bookmark{
		Title:   bm.Title,
		Level:   bm.Level,
		PageIdx: bm.PageIdx,
		Y:       pageOffsets[bm.PageIdx] + bm.Y,
	}

	if len(bm.Children) > 0 {
		adjusted.Children = make([]model.Bookmark, len(bm.Children))
		for i, child := range bm.Children {
			adjusted.Children[i] = adjustBookmark(child, pageOffsets)
		}
	}

	return adjusted
}

// replacePageNumber はパターン文字列中の {n} を指定ページ番号に置換する。
func replacePageNumber(pattern string, pageNum int) string {
	return strings.ReplaceAll(pattern, pageNumberPlaceholder, strconv.Itoa(pageNum))
}
