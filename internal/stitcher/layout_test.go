package stitcher

import (
	"math"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// テスト用の浮動小数点比較の許容誤差。
const tolerance = 1e-9

// almostEqual は2つの float64 値が許容誤差内で等しいかを判定する。
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCalculateLayout(t *testing.T) {
	tests := []struct {
		name         string
		pages        []model.Page
		gapPt        float64
		wantHeight   float64
		wantWidth    float64
		wantOffsets  []float64
		wantExceeds  bool
	}{
		{
			name: "1ページ・ガターなし",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
			},
			gapPt:       0,
			wantHeight:  842,
			wantWidth:   595,
			wantOffsets: []float64{0},
			wantExceeds: false,
		},
		{
			name: "3ページ・ガターなし",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
				{Index: 2, Width: 595, Height: 842},
			},
			gapPt:       0,
			wantHeight:  2526,
			wantWidth:   595,
			wantOffsets: []float64{0, 842, 1684},
			wantExceeds: false,
		},
		{
			name: "3ページ・ガターあり（10pt）",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
				{Index: 2, Width: 595, Height: 842},
			},
			gapPt:       10,
			wantHeight:  2546, // 842*3 + 10*2
			wantWidth:   595,
			wantOffsets: []float64{0, 852, 1704},
			wantExceeds: false,
		},
		{
			name: "異なるページサイズ",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},  // A4
				{Index: 1, Width: 842, Height: 595},  // A4横
				{Index: 2, Width: 420, Height: 595},   // A5
			},
			gapPt:       0,
			wantHeight:  2032,
			wantWidth:   842, // 最大幅
			wantOffsets: []float64{0, 842, 1437},
			wantExceeds: false,
		},
		{
			name: "PDF仕様上限超過",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 7200},
				{Index: 1, Width: 595, Height: 7201},
			},
			gapPt:       0,
			wantHeight:  14401,
			wantWidth:   595,
			wantOffsets: []float64{0, 7200},
			wantExceeds: true,
		},
		{
			name: "PDF仕様上限ちょうど",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 7200},
				{Index: 1, Width: 595, Height: 7200},
			},
			gapPt:       0,
			wantHeight:  14400,
			wantWidth:   595,
			wantOffsets: []float64{0, 7200},
			wantExceeds: false,
		},
		{
			name: "全ページ同一幅",
			pages: []model.Page{
				{Index: 0, Width: 100, Height: 200},
				{Index: 1, Width: 100, Height: 300},
			},
			gapPt:       5,
			wantHeight:  505, // 200 + 5 + 300
			wantWidth:   100,
			wantOffsets: []float64{0, 205},
			wantExceeds: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculateLayout(tt.pages, tt.gapPt)

			if !almostEqual(layout.TotalHeight, tt.wantHeight) {
				t.Errorf("TotalHeight = %f, 期待値 %f", layout.TotalHeight, tt.wantHeight)
			}
			if !almostEqual(layout.MaxWidth, tt.wantWidth) {
				t.Errorf("MaxWidth = %f, 期待値 %f", layout.MaxWidth, tt.wantWidth)
			}
			if layout.ExceedsLimit != tt.wantExceeds {
				t.Errorf("ExceedsLimit = %v, 期待値 %v", layout.ExceedsLimit, tt.wantExceeds)
			}
			if len(layout.PageOffsets) != len(tt.wantOffsets) {
				t.Fatalf("PageOffsets 長さ = %d, 期待値 %d",
					len(layout.PageOffsets), len(tt.wantOffsets))
			}
			for i, offset := range layout.PageOffsets {
				if !almostEqual(offset, tt.wantOffsets[i]) {
					t.Errorf("PageOffsets[%d] = %f, 期待値 %f", i, offset, tt.wantOffsets[i])
				}
			}
		})
	}
}
