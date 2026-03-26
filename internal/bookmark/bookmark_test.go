package bookmark

import (
	"math"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// テスト用の浮動小数点比較の許容誤差。
const tolerance = 1e-9

func TestNewProcessor(t *testing.T) {
	p := NewProcessor()
	if p == nil {
		t.Fatal("NewProcessor() は nil を返すべきではない")
	}
}

func TestAdjust(t *testing.T) {
	proc := NewProcessor()

	tests := []struct {
		name        string
		bookmarks   []model.Bookmark
		pageOffsets []float64
		want        []model.Bookmark
	}{
		{
			name:        "空のしおりリスト",
			bookmarks:   nil,
			pageOffsets: []float64{0, 100, 200},
			want:        nil,
		},
		{
			name: "単一しおり",
			bookmarks: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 10.0},
			},
			pageOffsets: []float64{0, 100, 200},
			want: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 10.0},
			},
		},
		{
			name: "複数しおり（異なるページ）",
			bookmarks: []model.Bookmark{
				{Title: "Chapter 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Chapter 2", Level: 0, PageIdx: 1, Y: 20.0},
				{Title: "Chapter 3", Level: 0, PageIdx: 2, Y: 30.0},
			},
			pageOffsets: []float64{0, 100, 250},
			want: []model.Bookmark{
				{Title: "Chapter 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Chapter 2", Level: 0, PageIdx: 1, Y: 120.0},
				{Title: "Chapter 3", Level: 0, PageIdx: 2, Y: 280.0},
			},
		},
		{
			name: "2階層のしおり",
			bookmarks: []model.Bookmark{
				{
					Title: "Chapter 1", Level: 0, PageIdx: 0, Y: 0,
					Children: []model.Bookmark{
						{Title: "Section 1.1", Level: 1, PageIdx: 0, Y: 50.0},
						{Title: "Section 1.2", Level: 1, PageIdx: 1, Y: 10.0},
					},
				},
			},
			pageOffsets: []float64{0, 200},
			want: []model.Bookmark{
				{
					Title: "Chapter 1", Level: 0, PageIdx: 0, Y: 0,
					Children: []model.Bookmark{
						{Title: "Section 1.1", Level: 1, PageIdx: 0, Y: 50.0},
						{Title: "Section 1.2", Level: 1, PageIdx: 1, Y: 210.0},
					},
				},
			},
		},
		{
			name: "3段以上の深い階層",
			bookmarks: []model.Bookmark{
				{
					Title: "L0", Level: 0, PageIdx: 0, Y: 0,
					Children: []model.Bookmark{
						{
							Title: "L1", Level: 1, PageIdx: 0, Y: 10.0,
							Children: []model.Bookmark{
								{
									Title: "L2", Level: 2, PageIdx: 1, Y: 5.0,
									Children: []model.Bookmark{
										{Title: "L3", Level: 3, PageIdx: 2, Y: 15.0},
									},
								},
							},
						},
					},
				},
			},
			pageOffsets: []float64{0, 100, 300},
			want: []model.Bookmark{
				{
					Title: "L0", Level: 0, PageIdx: 0, Y: 0,
					Children: []model.Bookmark{
						{
							Title: "L1", Level: 1, PageIdx: 0, Y: 10.0,
							Children: []model.Bookmark{
								{
									Title: "L2", Level: 2, PageIdx: 1, Y: 105.0,
									Children: []model.Bookmark{
										{Title: "L3", Level: 3, PageIdx: 2, Y: 315.0},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "同一ページの複数しおり",
			bookmarks: []model.Bookmark{
				{Title: "Top", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Middle", Level: 0, PageIdx: 0, Y: 100.0},
				{Title: "Bottom", Level: 0, PageIdx: 0, Y: 200.0},
			},
			pageOffsets: []float64{50.0},
			want: []model.Bookmark{
				{Title: "Top", Level: 0, PageIdx: 0, Y: 50.0},
				{Title: "Middle", Level: 0, PageIdx: 0, Y: 150.0},
				{Title: "Bottom", Level: 0, PageIdx: 0, Y: 250.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proc.Adjust(tt.bookmarks, tt.pageOffsets)
			assertBookmarksEqual(t, got, tt.want)
		})
	}
}

func TestAdjustImmutability(t *testing.T) {
	// 入力のしおりスライスが変更されないことを検証する
	proc := NewProcessor()

	original := []model.Bookmark{
		{
			Title: "Chapter 1", Level: 0, PageIdx: 0, Y: 10.0,
			Children: []model.Bookmark{
				{Title: "Section 1.1", Level: 1, PageIdx: 1, Y: 20.0},
			},
		},
	}

	pageOffsets := []float64{0, 100}

	// 元のY座標を記録する
	originalY := original[0].Y
	originalChildY := original[0].Children[0].Y

	_ = proc.Adjust(original, pageOffsets)

	// 入力が変更されていないことを確認する
	if original[0].Y != originalY {
		t.Errorf("入力の Y が変更された: %f → %f", originalY, original[0].Y)
	}
	if original[0].Children[0].Y != originalChildY {
		t.Errorf("入力の Children[0].Y が変更された: %f → %f",
			originalChildY, original[0].Children[0].Y)
	}
}

func TestGenerate(t *testing.T) {
	proc := NewProcessor()

	tests := []struct {
		name        string
		pages       []model.Page
		pageOffsets []float64
		pattern     string
		want        []model.Bookmark
	}{
		{
			name:        "空のページリスト",
			pages:       nil,
			pageOffsets: nil,
			pattern:     "Page {n}",
			want:        nil,
		},
		{
			name: "1ページ",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0},
			pattern:     "Page {n}",
			want: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 0},
			},
		},
		{
			name: "3ページ",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
				{Index: 2, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0, 842, 1684},
			pattern:     "Page {n}",
			want: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Page 2", Level: 0, PageIdx: 1, Y: 842},
				{Title: "Page 3", Level: 0, PageIdx: 2, Y: 1684},
			},
		},
		{
			name: "カスタムパターン",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0, 842},
			pattern:     "Section {n}",
			want: []model.Bookmark{
				{Title: "Section 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Section 2", Level: 0, PageIdx: 1, Y: 842},
			},
		},
		{
			name: "パターンに{n}が複数回含まれる",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0},
			pattern:     "{n}-{n}",
			want: []model.Bookmark{
				{Title: "1-1", Level: 0, PageIdx: 0, Y: 0},
			},
		},
		{
			name: "パターンに{n}が含まれない",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0, 842},
			pattern:     "Bookmark",
			want: []model.Bookmark{
				{Title: "Bookmark", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Bookmark", Level: 0, PageIdx: 1, Y: 842},
			},
		},
		{
			name: "ガター付きのオフセット",
			pages: []model.Page{
				{Index: 0, Width: 595, Height: 842},
				{Index: 1, Width: 595, Height: 842},
				{Index: 2, Width: 595, Height: 842},
			},
			pageOffsets: []float64{0, 852, 1704},
			pattern:     "Page {n}",
			want: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Page 2", Level: 0, PageIdx: 1, Y: 852},
				{Title: "Page 3", Level: 0, PageIdx: 2, Y: 1704},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proc.Generate(tt.pages, tt.pageOffsets, tt.pattern)
			assertBookmarksEqual(t, got, tt.want)
		})
	}
}

func TestGenerateAllLevelZero(t *testing.T) {
	// 自動生成されたしおりが全て Level 0（フラット）であることを検証する
	proc := NewProcessor()
	pages := make([]model.Page, 5)
	offsets := make([]float64, 5)
	for i := range pages {
		pages[i] = model.Page{Index: i, Width: 595, Height: 842}
		offsets[i] = float64(i) * 842
	}

	bookmarks := proc.Generate(pages, offsets, "Page {n}")

	for i, bm := range bookmarks {
		if bm.Level != 0 {
			t.Errorf("bookmarks[%d].Level = %d, 期待値 0", i, bm.Level)
		}
		if bm.Children != nil {
			t.Errorf("bookmarks[%d].Children は nil であるべき", i)
		}
	}
}

// assertBookmarksEqual は2つのしおりスライスが等しいことを再帰的に検証する。
func assertBookmarksEqual(t *testing.T, got, want []model.Bookmark) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("しおり数 = %d, 期待値 %d", len(got), len(want))
	}

	// 両方 nil の場合
	if got == nil && want == nil {
		return
	}

	for i := range got {
		assertBookmarkEqual(t, got[i], want[i], i)
	}
}

// assertBookmarkEqual は2つのしおりが等しいことを検証する。
func assertBookmarkEqual(t *testing.T, got, want model.Bookmark, index int) {
	t.Helper()

	if got.Title != want.Title {
		t.Errorf("bookmarks[%d].Title = %q, 期待値 %q", index, got.Title, want.Title)
	}
	if got.Level != want.Level {
		t.Errorf("bookmarks[%d].Level = %d, 期待値 %d", index, got.Level, want.Level)
	}
	if got.PageIdx != want.PageIdx {
		t.Errorf("bookmarks[%d].PageIdx = %d, 期待値 %d", index, got.PageIdx, want.PageIdx)
	}
	if math.Abs(got.Y-want.Y) > tolerance {
		t.Errorf("bookmarks[%d].Y = %f, 期待値 %f", index, got.Y, want.Y)
	}

	if len(got.Children) != len(want.Children) {
		t.Fatalf("bookmarks[%d].Children 数 = %d, 期待値 %d",
			index, len(got.Children), len(want.Children))
	}
	for i := range got.Children {
		assertBookmarkEqual(t, got.Children[i], want.Children[i], i)
	}
}
