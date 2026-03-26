package stitcher

import (
	"errors"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// --- モック定義 ---

// mockPageReader は PageReader のテスト用モック。
type mockPageReader struct {
	pages     []model.Page
	pagesErr  error
	bookmarks []model.Bookmark
	bmErr     error
}

func (m *mockPageReader) ReadPages(_ string) ([]model.Page, error) {
	return m.pages, m.pagesErr
}

func (m *mockPageReader) ReadBookmarks(_ string) ([]model.Bookmark, error) {
	return m.bookmarks, m.bmErr
}

// mockPageWriter は PageWriter のテスト用モック。
type mockPageWriter struct {
	writtenResult *model.StitchResult
	writeErr      error
	cleanupCalled bool
}

func (m *mockPageWriter) Write(_ string, result *model.StitchResult) error {
	m.writtenResult = result
	return m.writeErr
}

func (m *mockPageWriter) Cleanup() {
	m.cleanupCalled = true
}

// mockBookmarkProcessor は BookmarkProcessor のテスト用モック。
type mockBookmarkProcessor struct {
	adjustCalled   bool
	generateCalled bool
	adjustResult   []model.Bookmark
	generateResult []model.Bookmark
}

func (m *mockBookmarkProcessor) Adjust(bookmarks []model.Bookmark, _ []float64) []model.Bookmark {
	m.adjustCalled = true
	if m.adjustResult != nil {
		return m.adjustResult
	}
	return bookmarks
}

func (m *mockBookmarkProcessor) Generate(_ []model.Page, _ []float64, _ string) []model.Bookmark {
	m.generateCalled = true
	return m.generateResult
}

// --- AlignPages テスト ---

func TestAlignPages(t *testing.T) {
	pages := []model.Page{
		{Index: 0, Width: 400, Height: 600},
		{Index: 1, Width: 600, Height: 800},
		{Index: 2, Width: 500, Height: 700},
	}
	maxWidth := 600.0

	tests := []struct {
		name        string
		align       string
		wantOffsets []float64
	}{
		{
			name:        "左揃え",
			align:       "left",
			wantOffsets: []float64{0, 0, 0},
		},
		{
			name:        "中央揃え",
			align:       "center",
			wantOffsets: []float64{100, 0, 50},
		},
		{
			name:        "右揃え",
			align:       "right",
			wantOffsets: []float64{200, 0, 100},
		},
		{
			name:        "不正値はleft扱い",
			align:       "invalid",
			wantOffsets: []float64{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AlignPages(pages, maxWidth, tt.align)

			if len(result) != len(pages) {
				t.Fatalf("結果の長さ = %d, 期待値 %d", len(result), len(pages))
			}

			for i, ap := range result {
				if !almostEqual(ap.XOffset, tt.wantOffsets[i]) {
					t.Errorf("result[%d].XOffset = %f, 期待値 %f",
						i, ap.XOffset, tt.wantOffsets[i])
				}
				if ap.Page.Index != pages[i].Index {
					t.Errorf("result[%d].Page.Index = %d, 期待値 %d",
						i, ap.Page.Index, pages[i].Index)
				}
			}
		})
	}
}

func TestAlignPagesSameWidth(t *testing.T) {
	// 全ページ同一幅の場合、全方向で XOffset = 0 になることを検証する
	pages := []model.Page{
		{Index: 0, Width: 595, Height: 842},
		{Index: 1, Width: 595, Height: 842},
	}

	for _, align := range []string{"left", "center", "right"} {
		result := AlignPages(pages, 595, align)
		for i, ap := range result {
			if ap.XOffset != 0 {
				t.Errorf("align=%q: result[%d].XOffset = %f, 期待値 0",
					align, i, ap.XOffset)
			}
		}
	}
}

// --- Stitcher.Run テスト ---

func TestRun(t *testing.T) {
	testPages := []model.Page{
		{Index: 0, Width: 595, Height: 842},
		{Index: 1, Width: 595, Height: 842},
		{Index: 2, Width: 595, Height: 842},
	}

	t.Run("正常系: しおりなし（自動生成）", func(t *testing.T) {
		reader := &mockPageReader{pages: testPages}
		writer := &mockPageWriter{}
		bmProc := &mockBookmarkProcessor{
			generateResult: []model.Bookmark{
				{Title: "Page 1", Y: 0},
				{Title: "Page 2", Y: 842},
				{Title: "Page 3", Y: 1684},
			},
		}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("input.pdf", "output.pdf", StitchOptions{
			GapPt:           0,
			Align:           "left",
			BookmarkPattern: "Page {n}",
		})

		if err != nil {
			t.Fatalf("Run() で予期しないエラー: %v", err)
		}
		if !bmProc.generateCalled {
			t.Error("しおりなし時に Generate が呼ばれるべき")
		}
		if bmProc.adjustCalled {
			t.Error("しおりなし時に Adjust は呼ばれるべきではない")
		}
		if writer.writtenResult == nil {
			t.Fatal("Write が呼ばれていない")
		}
		if len(writer.writtenResult.AlignedPages) != 3 {
			t.Errorf("AlignedPages 数 = %d, 期待値 3",
				len(writer.writtenResult.AlignedPages))
		}
		if !almostEqual(writer.writtenResult.TotalHeight, 2526) {
			t.Errorf("TotalHeight = %f, 期待値 2526", writer.writtenResult.TotalHeight)
		}
	})

	t.Run("正常系: しおりあり（座標調整）", func(t *testing.T) {
		existingBMs := []model.Bookmark{
			{Title: "Chapter 1", PageIdx: 0},
		}
		reader := &mockPageReader{
			pages:     testPages,
			bookmarks: existingBMs,
		}
		writer := &mockPageWriter{}
		bmProc := &mockBookmarkProcessor{}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("input.pdf", "output.pdf", StitchOptions{
			Align: "center",
		})

		if err != nil {
			t.Fatalf("Run() で予期しないエラー: %v", err)
		}
		if !bmProc.adjustCalled {
			t.Error("しおりあり時に Adjust が呼ばれるべき")
		}
		if bmProc.generateCalled {
			t.Error("しおりあり時に Generate は呼ばれるべきではない")
		}
	})

	t.Run("正常系: ガター付き", func(t *testing.T) {
		reader := &mockPageReader{pages: testPages}
		writer := &mockPageWriter{}
		bmProc := &mockBookmarkProcessor{}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("input.pdf", "output.pdf", StitchOptions{
			GapPt: 10,
			Align: "left",
		})

		if err != nil {
			t.Fatalf("Run() で予期しないエラー: %v", err)
		}

		// 842*3 + 10*2 = 2546
		if !almostEqual(writer.writtenResult.TotalHeight, 2546) {
			t.Errorf("TotalHeight = %f, 期待値 2546", writer.writtenResult.TotalHeight)
		}
		if !almostEqual(writer.writtenResult.GapPt, 10) {
			t.Errorf("GapPt = %f, 期待値 10", writer.writtenResult.GapPt)
		}
	})

	t.Run("異常系: ReadPages エラー伝播", func(t *testing.T) {
		readErr := apperror.NewInputFileError("file not found", nil)
		reader := &mockPageReader{pagesErr: readErr}
		writer := &mockPageWriter{}
		bmProc := &mockBookmarkProcessor{}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("missing.pdf", "output.pdf", StitchOptions{})

		if err == nil {
			t.Fatal("エラーが返されるべき")
		}

		var exitErr *apperror.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("ExitError であるべき: %T", err)
		}
		if exitErr.Code != apperror.ExitInputFile {
			t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, apperror.ExitInputFile)
		}
	})

	t.Run("異常系: Write エラー伝播", func(t *testing.T) {
		reader := &mockPageReader{pages: testPages}
		writeErr := apperror.NewOutputFileError("write failed", nil)
		writer := &mockPageWriter{writeErr: writeErr}
		bmProc := &mockBookmarkProcessor{}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("input.pdf", "output.pdf", StitchOptions{})

		if err == nil {
			t.Fatal("エラーが返されるべき")
		}

		var exitErr *apperror.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("ExitError であるべき: %T", err)
		}
		if exitErr.Code != apperror.ExitOutputFile {
			t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, apperror.ExitOutputFile)
		}
	})

	t.Run("正常系: 1ページPDF", func(t *testing.T) {
		singlePage := []model.Page{
			{Index: 0, Width: 595, Height: 842},
		}
		reader := &mockPageReader{pages: singlePage}
		writer := &mockPageWriter{}
		bmProc := &mockBookmarkProcessor{}

		s := NewStitcher(reader, writer, bmProc)
		err := s.Run("input.pdf", "output.pdf", StitchOptions{GapPt: 10})

		if err != nil {
			t.Fatalf("Run() で予期しないエラー: %v", err)
		}

		// 1ページのみ→ガターは加算されない
		if !almostEqual(writer.writtenResult.TotalHeight, 842) {
			t.Errorf("TotalHeight = %f, 期待値 842", writer.writtenResult.TotalHeight)
		}
	})
}

func TestNewStitcher(t *testing.T) {
	s := NewStitcher(&mockPageReader{}, &mockPageWriter{}, &mockBookmarkProcessor{})
	if s == nil {
		t.Fatal("NewStitcher() は nil を返すべきではない")
	}
}
