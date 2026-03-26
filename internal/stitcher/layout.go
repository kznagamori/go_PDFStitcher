package stitcher

import (
	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// maxPDFUserUnits はPDFページサイズの仕様上限（ユーザー単位）。
// 約200インチ（14,400 / 72 ≈ 200）。
const maxPDFUserUnits = 14400.0

// Layout はページ配置の計算結果を保持する。
type Layout struct {
	TotalHeight  float64   // 連結後の総高さ（pt）
	MaxWidth     float64   // 最大ページ幅（pt）
	PageOffsets  []float64 // 各ページのY座標オフセット（pt、上端基準）
	ExceedsLimit bool      // PDF仕様上限（14,400pt）を超過しているか
}

// CalculateLayout は全ページのレイアウトを計算する。
//
// 各ページの高さを累積し、ページ間に gapPt を加算して総高さを算出する。
// 総高さが PDF 仕様上限（14,400pt）を超える場合、ExceedsLimit を true に設定する。
//
// PageOffsets[i] はページ i の上端Y座標（モデル座標系、上端基準）。
//
//	PageOffsets[0] = 0
//	PageOffsets[1] = pages[0].Height + gapPt
//	PageOffsets[2] = pages[0].Height + gapPt + pages[1].Height + gapPt
//	...
func CalculateLayout(pages []model.Page, gapPt float64) Layout {
	pageCount := len(pages)
	offsets := make([]float64, pageCount)

	var totalHeight float64
	var maxWidth float64

	for i, page := range pages {
		offsets[i] = totalHeight
		totalHeight += page.Height

		// 最終ページ以外はガターを加算する
		if i < pageCount-1 {
			totalHeight += gapPt
		}

		if page.Width > maxWidth {
			maxWidth = page.Width
		}
	}

	return Layout{
		TotalHeight:  totalHeight,
		MaxWidth:     maxWidth,
		PageOffsets:  offsets,
		ExceedsLimit: totalHeight > maxPDFUserUnits,
	}
}
