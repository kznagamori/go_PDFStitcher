package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// mockPasswordPrompter は PasswordPrompter のテスト用モック。
type mockPasswordPrompter struct {
	password string
	err      error
}

// Password はモックのパスワードを返す。
func (m *mockPasswordPrompter) Password(_ string) (string, error) {
	return m.password, m.err
}

// buildMinimalPDF はN ページの最小限PDFをバイト列として生成する。
// 各ページの MediaBox は [0 0 width height] で設定する。
func buildMinimalPDF(pages []types.Dim) []byte {
	var buf bytes.Buffer

	buf.WriteString("%PDF-1.4\n")

	// オブジェクト番号:
	// 1: Catalog
	// 2: Pages
	// 3..N+2: 各ページ

	// Kids 配列を構築する
	var kids bytes.Buffer
	kids.WriteString("[")
	for i := range pages {
		if i > 0 {
			kids.WriteString(" ")
		}
		fmt.Fprintf(&kids, "%d 0 R", i+3)
	}
	kids.WriteString("]")

	offsets := make([]int, len(pages)+2)

	// Object 1: Catalog
	offsets[0] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets[1] = buf.Len()
	fmt.Fprintf(&buf, "2 0 obj\n<< /Type /Pages /Kids %s /Count %d >>\nendobj\n",
		kids.String(), len(pages))

	// 各ページオブジェクト
	for i, dim := range pages {
		offsets[i+2] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] >>\nendobj\n",
			i+3, dim.Width, dim.Height)
	}

	// xref テーブル
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(pages)+3)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	// Trailer
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(pages)+3)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes()
}

// createTestPDFFile はテスト用のPDFファイルを生成する。
func createTestPDFFile(t *testing.T, name string, pageCount int) string {
	t.Helper()
	dim := types.Dim{Width: 595.28, Height: 841.89} // A4サイズ
	dims := make([]types.Dim, pageCount)
	for i := range dims {
		dims[i] = dim
	}
	return createPDFFromDims(t, name, dims)
}

// createMultiSizeTestPDF は異なるページサイズのPDFを生成する。
func createMultiSizeTestPDF(t *testing.T, name string, dims []types.Dim) string {
	t.Helper()
	return createPDFFromDims(t, name, dims)
}

// createPDFFromDims はページサイズ指定でテスト用PDFファイルを生成する。
func createPDFFromDims(t *testing.T, name string, dims []types.Dim) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	data := buildMinimalPDF(dims)
	if err := writeBytes(path, data); err != nil {
		t.Fatalf("テスト用PDF書き込み失敗: %v", err)
	}
	return path
}

// writeBytes はバイト列をファイルに書き込む。
func writeBytes(path string, data []byte) error {
	return writeFile(path, data)
}

// writeFile はバイト列をファイルに書き込む（os.WriteFile のラッパー）。
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// _は未使用import回避（pdfcpu はテスト内で間接的に使用）
var _ = pdfcpu.Bookmarks

func TestReadPages(t *testing.T) {
	prompter := &mockPasswordPrompter{}
	reader := NewReader(prompter)

	t.Run("3ページPDFの読み込み", func(t *testing.T) {
		path := createTestPDFFile(t, "test.pdf", 3)

		pages, err := reader.ReadPages(path)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		if len(pages) != 3 {
			t.Fatalf("ページ数 = %d, 期待値 3", len(pages))
		}

		for i, page := range pages {
			if page.Index != i {
				t.Errorf("pages[%d].Index = %d, 期待値 %d", i, page.Index, i)
			}
			if page.Width <= 0 {
				t.Errorf("pages[%d].Width = %f, 正の値であるべき", i, page.Width)
			}
			if page.Height <= 0 {
				t.Errorf("pages[%d].Height = %f, 正の値であるべき", i, page.Height)
			}
			if page.Content == nil {
				t.Errorf("pages[%d].Content は nil であるべきではない", i)
			}
		}
	})

	t.Run("1ページPDFの読み込み", func(t *testing.T) {
		path := createTestPDFFile(t, "single.pdf", 1)

		pages, err := reader.ReadPages(path)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		if len(pages) != 1 {
			t.Fatalf("ページ数 = %d, 期待値 1", len(pages))
		}
	})

	t.Run("異なるページサイズのPDF", func(t *testing.T) {
		dims := []types.Dim{
			{Width: 595.28, Height: 841.89}, // A4
			{Width: 841.89, Height: 595.28}, // A4横
			{Width: 419.53, Height: 595.28}, // A5
		}
		path := createMultiSizeTestPDF(t, "mixed.pdf", dims)

		pages, err := reader.ReadPages(path)
		if err != nil {
			t.Fatalf("ReadPages() で予期しないエラー: %v", err)
		}

		if len(pages) != 3 {
			t.Fatalf("ページ数 = %d, 期待値 3", len(pages))
		}
	})

	t.Run("存在しないファイルでExitError(2)", func(t *testing.T) {
		_, err := reader.ReadPages(filepath.Join(t.TempDir(), "nonexistent.pdf"))
		if err == nil {
			t.Fatal("エラーが返されるべき")
		}
		var exitErr *apperror.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("エラーが *apperror.ExitError ではない: %T", err)
		}
		if exitErr.Code != apperror.ExitInputFile {
			t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, apperror.ExitInputFile)
		}
	})
}

func TestReadBookmarks(t *testing.T) {
	prompter := &mockPasswordPrompter{}
	reader := NewReader(prompter)

	t.Run("しおりなしPDFは空スライスを返す", func(t *testing.T) {
		path := createTestPDFFile(t, "no_bookmarks.pdf", 3)

		bookmarks, err := reader.ReadBookmarks(path)
		if err != nil {
			t.Fatalf("ReadBookmarks() で予期しないエラー: %v", err)
		}

		if bookmarks != nil {
			t.Errorf("しおりなしPDFでは nil を返すべきだが、%d 件返された", len(bookmarks))
		}
	})
}

func TestIsPasswordRequired(t *testing.T) {
	t.Run("isPasswordRequired検出", func(t *testing.T) {
		if !isPasswordRequired(errors.New("please provide the correct password")) {
			t.Error("パスワードエラーが検出されるべき")
		}
		if !isPasswordRequired(errors.New("encrypted PDF")) {
			t.Error("暗号化エラーが検出されるべき")
		}
		if isPasswordRequired(errors.New("file not found")) {
			t.Error("通常のエラーはパスワードエラーとして検出されるべきではない")
		}
	})

}

func TestConvertFromPDFBookmarks(t *testing.T) {
	t.Run("フラット構造の変換", func(t *testing.T) {
		pdfBMs := []pdfcpu.Bookmark{
			{Title: "Chapter 1", PageFrom: 1},
			{Title: "Chapter 2", PageFrom: 3},
		}

		result := convertFromPDFBookmarks(pdfBMs, 0)

		if len(result) != 2 {
			t.Fatalf("しおり数 = %d, 期待値 2", len(result))
		}
		if result[0].Title != "Chapter 1" {
			t.Errorf("Title = %q, 期待値 %q", result[0].Title, "Chapter 1")
		}
		if result[0].PageIdx != 0 {
			t.Errorf("PageIdx = %d, 期待値 0（0始まり変換）", result[0].PageIdx)
		}
		if result[1].PageIdx != 2 {
			t.Errorf("PageIdx = %d, 期待値 2", result[1].PageIdx)
		}
	})

	t.Run("階層構造の変換", func(t *testing.T) {
		pdfBMs := []pdfcpu.Bookmark{
			{
				Title:    "Part 1",
				PageFrom: 1,
				Kids: []pdfcpu.Bookmark{
					{Title: "Chapter 1.1", PageFrom: 1},
					{Title: "Chapter 1.2", PageFrom: 2},
				},
			},
		}

		result := convertFromPDFBookmarks(pdfBMs, 0)

		if len(result) != 1 {
			t.Fatalf("しおり数 = %d, 期待値 1", len(result))
		}
		if result[0].Level != 0 {
			t.Errorf("Level = %d, 期待値 0", result[0].Level)
		}
		if len(result[0].Children) != 2 {
			t.Fatalf("子しおり数 = %d, 期待値 2", len(result[0].Children))
		}
		if result[0].Children[0].Level != 1 {
			t.Errorf("子のLevel = %d, 期待値 1", result[0].Children[0].Level)
		}
	})
}

func TestNewReader(t *testing.T) {
	prompter := &mockPasswordPrompter{}
	r := NewReader(prompter)
	if r == nil {
		t.Fatal("NewReader() は nil を返すべきではない")
	}
}
