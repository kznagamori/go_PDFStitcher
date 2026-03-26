package stitcher

import (
	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// アライメント方向の定数。マジック文字列の使用を回避する。
const (
	alignLeft   = "left"
	alignCenter = "center"
	alignRight  = "right"
)

// AlignPages は各ページのX座標オフセットを計算する。
//
// 配置方針に基づいて、最大幅より狭いページの水平位置を決定する:
//   - "left":   XOffset = 0
//   - "center": XOffset = (maxWidth - page.Width) / 2
//   - "right":  XOffset = maxWidth - page.Width
//
// 不正な align 値は "left" として扱う。
func AlignPages(pages []model.Page, maxWidth float64, align string) []model.AlignedPage {
	result := make([]model.AlignedPage, len(pages))

	for i, page := range pages {
		result[i] = model.AlignedPage{
			Page:    page,
			XOffset: calculateXOffset(page.Width, maxWidth, align),
		}
	}

	return result
}

// calculateXOffset はアライメント方向に基づいてX座標オフセットを計算する。
func calculateXOffset(pageWidth, maxWidth float64, align string) float64 {
	switch align {
	case alignCenter:
		return (maxWidth - pageWidth) / 2
	case alignRight:
		return maxWidth - pageWidth
	default:
		return 0
	}
}
