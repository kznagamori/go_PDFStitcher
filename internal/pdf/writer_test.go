package pdf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

func TestNewWriter(t *testing.T) {
	w := NewWriter()
	if w == nil {
		t.Fatal("NewWriter() は nil を返すべきではない")
	}
}

func TestWriteAndCleanup(t *testing.T) {
	prompter := &mockPasswordPrompter{}
	reader := NewReader(prompter)

	t.Run("3ページPDFの連結出力", func(t *testing.T) {
		// テスト用PDFを作成する
		inputPath := createTestPDFFile(t, "input.pdf", 3)
		outputPath := filepath.Join(t.TempDir(), "output.pdf")

		// ページ情報を読み取る
		pages, err := reader.ReadPages(inputPath)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		// StitchResult を構築する
		totalHeight := 0.0
		maxWidth := 0.0
		alignedPages := make([]model.AlignedPage, len(pages))
		for i, p := range pages {
			alignedPages[i] = model.AlignedPage{Page: p, XOffset: 0}
			totalHeight += p.Height
			if p.Width > maxWidth {
				maxWidth = p.Width
			}
		}

		result := &model.StitchResult{
			AlignedPages: alignedPages,
			TotalHeight:  totalHeight,
			MaxWidth:     maxWidth,
			GapPt:        0,
		}

		// 書き込む
		writer := NewWriter()
		err = writer.Write(outputPath, result)
		if err != nil {
			t.Fatalf("Write() で予期しないエラー: %v", err)
		}

		// 出力ファイルが存在することを確認する
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("出力ファイルが生成されていない")
		}

		// 一時ファイルが残っていないことを確認する
		tmpPath := outputPath + ".tmp"
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Error("一時ファイルが残っている")
		}
	})

	t.Run("1ページPDFの連結出力", func(t *testing.T) {
		inputPath := createTestPDFFile(t, "single.pdf", 1)
		outputPath := filepath.Join(t.TempDir(), "output.pdf")

		pages, err := reader.ReadPages(inputPath)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		result := &model.StitchResult{
			AlignedPages: []model.AlignedPage{
				{Page: pages[0], XOffset: 0},
			},
			TotalHeight: pages[0].Height,
			MaxWidth:    pages[0].Width,
		}

		writer := NewWriter()
		err = writer.Write(outputPath, result)
		if err != nil {
			t.Fatalf("Write() で予期しないエラー: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("出力ファイルが生成されていない")
		}
	})

	t.Run("しおり付き出力", func(t *testing.T) {
		inputPath := createTestPDFFile(t, "with_bm.pdf", 2)
		outputPath := filepath.Join(t.TempDir(), "output.pdf")

		pages, err := reader.ReadPages(inputPath)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		totalHeight := pages[0].Height + pages[1].Height
		maxWidth := pages[0].Width

		result := &model.StitchResult{
			AlignedPages: []model.AlignedPage{
				{Page: pages[0], XOffset: 0},
				{Page: pages[1], XOffset: 0},
			},
			Bookmarks: []model.Bookmark{
				{Title: "Page 1", Level: 0, PageIdx: 0, Y: 0},
				{Title: "Page 2", Level: 0, PageIdx: 1, Y: pages[0].Height},
			},
			TotalHeight: totalHeight,
			MaxWidth:    maxWidth,
		}

		writer := NewWriter()
		err = writer.Write(outputPath, result)
		if err != nil {
			t.Fatalf("Write() で予期しないエラー: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("出力ファイルが生成されていない")
		}
	})

	t.Run("異なるページサイズの連結出力", func(t *testing.T) {
		dims := []types.Dim{
			{Width: 595.28, Height: 841.89},
			{Width: 419.53, Height: 595.28},
		}
		inputPath := createMultiSizeTestPDF(t, "mixed.pdf", dims)
		outputPath := filepath.Join(t.TempDir(), "output.pdf")

		pages, err := reader.ReadPages(inputPath)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		totalHeight := 0.0
		maxWidth := 0.0
		alignedPages := make([]model.AlignedPage, len(pages))
		for i, p := range pages {
			alignedPages[i] = model.AlignedPage{Page: p, XOffset: 0}
			totalHeight += p.Height
			if p.Width > maxWidth {
				maxWidth = p.Width
			}
		}

		result := &model.StitchResult{
			AlignedPages: alignedPages,
			TotalHeight:  totalHeight,
			MaxWidth:     maxWidth,
		}

		writer := NewWriter()
		err = writer.Write(outputPath, result)
		if err != nil {
			t.Fatalf("Write() で予期しないエラー: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("出力ファイルが生成されていない")
		}
	})
}

func TestCleanup(t *testing.T) {
	t.Run("一時ファイルの削除", func(t *testing.T) {
		dir := t.TempDir()
		tmpPath := filepath.Join(dir, "output.pdf.tmp")

		// 一時ファイルを作成する
		if err := os.WriteFile(tmpPath, []byte("temp"), 0644); err != nil {
			t.Fatalf("一時ファイル作成失敗: %v", err)
		}

		w := &Writer{tmpPath: tmpPath}
		w.Cleanup()

		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Error("Cleanup 後に一時ファイルが残っている")
		}

		if w.tmpPath != "" {
			t.Error("Cleanup 後に tmpPath がクリアされていない")
		}
	})

	t.Run("一時ファイルが存在しない場合もエラーにしない", func(t *testing.T) {
		w := &Writer{tmpPath: filepath.Join(t.TempDir(), "nonexistent.tmp")}
		w.Cleanup() // パニックしないことを確認する
	})

	t.Run("tmpPathが空の場合は何もしない", func(t *testing.T) {
		w := &Writer{}
		w.Cleanup() // パニックしないことを確認する
	})
}

func TestCountOutlineItems(t *testing.T) {
	tests := []struct {
		name      string
		bookmarks []model.Bookmark
		want      int
	}{
		{
			name:      "空リスト",
			bookmarks: nil,
			want:      0,
		},
		{
			name: "フラット2件",
			bookmarks: []model.Bookmark{
				{Title: "A"},
				{Title: "B"},
			},
			want: 2,
		},
		{
			name: "階層構造",
			bookmarks: []model.Bookmark{
				{
					Title: "Chapter 1",
					Children: []model.Bookmark{
						{Title: "Section 1.1"},
					},
				},
				{Title: "Chapter 2"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countOutlineItems(tt.bookmarks)
			if got != tt.want {
				t.Errorf("countOutlineItems() = %d, 期待値 %d", got, tt.want)
			}
		})
	}
}
