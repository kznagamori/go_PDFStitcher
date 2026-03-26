package stitcher_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/kznagamori/go_PDFStitcher/internal/bookmark"
	"github.com/kznagamori/go_PDFStitcher/internal/pdf"
	"github.com/kznagamori/go_PDFStitcher/internal/stitcher"
)

// mockPrompter は PasswordPrompter のテスト用モック。
type mockPrompter struct{}

func (m *mockPrompter) Password(_ string) (string, error) {
	return "", fmt.Errorf("no password")
}

// buildMinimalPDF は N ページの最小限PDFをバイト列として生成する。
func buildMinimalPDF(pages []types.Dim) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

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

	offsets[0] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	offsets[1] = buf.Len()
	fmt.Fprintf(&buf, "2 0 obj\n<< /Type /Pages /Kids %s /Count %d >>\nendobj\n",
		kids.String(), len(pages))

	for i, dim := range pages {
		offsets[i+2] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] >>\nendobj\n",
			i+3, dim.Width, dim.Height)
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(pages)+3)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(pages)+3)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return buf.Bytes()
}

// createTestPDF はテスト用PDFファイルを指定ディレクトリに作成する。
func createTestPDF(t *testing.T, dir string, name string, dims []types.Dim) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := buildMinimalPDF(dims)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("テスト用PDF作成失敗: %v", err)
	}
	return path
}

// a4 は A4 サイズのページ寸法。
func a4() types.Dim {
	return types.Dim{Width: 595.28, Height: 841.89}
}

// verifyOutputPDF は出力PDFが有効であることを検証する。
func verifyOutputPDF(t *testing.T, path string, wantPages int) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("出力ファイルが存在しない: %s", path)
	}

	ctx, err := api.ReadContextFile(path)
	if err != nil {
		t.Fatalf("出力PDFの読み込みに失敗: %v", err)
	}

	if ctx.PageCount != wantPages {
		t.Errorf("出力PDFのページ数 = %d, 期待値 %d", ctx.PageCount, wantPages)
	}
}

// --- 結合テスト ---

func TestStitchWithRealPDF(t *testing.T) {
	// 実PDFの読み込み→連結→出力の一気通貫テスト（UC-01）
	dir := t.TempDir()
	inputPath := createTestPDF(t, dir, "input.pdf", []types.Dim{a4(), a4(), a4()})
	outputPath := filepath.Join(dir, "output.pdf")

	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(inputPath, outputPath, stitcher.StitchOptions{
		GapPt:           0,
		Align:           "left",
		BookmarkPattern: "Page {n}",
	})

	if err != nil {
		t.Fatalf("Run() で予期しないエラー: %v", err)
	}

	verifyOutputPDF(t, outputPath, 1)
}

func TestStitchWithGap(t *testing.T) {
	// ガター付き連結（UC-01 + UC-02）
	dir := t.TempDir()
	inputPath := createTestPDF(t, dir, "input.pdf", []types.Dim{a4(), a4()})
	outputPath := filepath.Join(dir, "output.pdf")

	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	gapPt := 20.0
	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(inputPath, outputPath, stitcher.StitchOptions{
		GapPt:           gapPt,
		Align:           "left",
		BookmarkPattern: "Page {n}",
	})

	if err != nil {
		t.Fatalf("Run() で予期しないエラー: %v", err)
	}

	verifyOutputPDF(t, outputPath, 1)

	// 出力PDFの高さがガター分加算されていることを検証する
	ctx, err := api.ReadContextFile(outputPath)
	if err != nil {
		t.Fatalf("出力PDF読み込み失敗: %v", err)
	}

	dims, err := ctx.PageDims()
	if err != nil {
		t.Fatalf("ページ寸法取得失敗: %v", err)
	}

	expectedHeight := a4().Height*2 + gapPt
	if dims[0].Height < expectedHeight-1 || dims[0].Height > expectedHeight+1 {
		t.Errorf("出力PDFの高さ = %.2f, 期待値 ≈ %.2f（ガター %.2f 含む）",
			dims[0].Height, expectedHeight, gapPt)
	}
}

func TestStitchWithAlignment(t *testing.T) {
	// 異なるページ幅の幅揃え（UC-01 + UC-03）
	dir := t.TempDir()
	dims := []types.Dim{
		{Width: 595.28, Height: 841.89}, // A4
		{Width: 419.53, Height: 595.28}, // A5
	}
	inputPath := createTestPDF(t, dir, "mixed.pdf", dims)
	outputPath := filepath.Join(dir, "output.pdf")

	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(inputPath, outputPath, stitcher.StitchOptions{
		GapPt:           0,
		Align:           "center",
		BookmarkPattern: "Page {n}",
	})

	if err != nil {
		t.Fatalf("Run() で予期しないエラー: %v", err)
	}

	// 出力PDFの幅が最大幅（A4幅）であることを検証する
	ctx, err := api.ReadContextFile(outputPath)
	if err != nil {
		t.Fatalf("出力PDF読み込み失敗: %v", err)
	}

	outDims, err := ctx.PageDims()
	if err != nil {
		t.Fatalf("ページ寸法取得失敗: %v", err)
	}

	maxWidth := 595.28
	if outDims[0].Width < maxWidth-1 || outDims[0].Width > maxWidth+1 {
		t.Errorf("出力PDFの幅 = %.2f, 期待値 ≈ %.2f（最大幅）",
			outDims[0].Width, maxWidth)
	}
}

func TestStitchWithoutBookmarks(t *testing.T) {
	// しおり自動生成（UC-01 + UC-10）
	dir := t.TempDir()
	inputPath := createTestPDF(t, dir, "input.pdf", []types.Dim{a4(), a4()})
	outputPath := filepath.Join(dir, "output.pdf")

	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(inputPath, outputPath, stitcher.StitchOptions{
		GapPt:           0,
		Align:           "left",
		BookmarkPattern: "Page {n}",
	})

	if err != nil {
		t.Fatalf("Run() で予期しないエラー: %v", err)
	}

	verifyOutputPDF(t, outputPath, 1)
}

func TestStitchSinglePage(t *testing.T) {
	// 1ページPDFの連結（エッジケース）
	dir := t.TempDir()
	inputPath := createTestPDF(t, dir, "single.pdf", []types.Dim{a4()})
	outputPath := filepath.Join(dir, "output.pdf")

	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(inputPath, outputPath, stitcher.StitchOptions{
		BookmarkPattern: "Page {n}",
	})

	if err != nil {
		t.Fatalf("Run() で予期しないエラー: %v", err)
	}

	verifyOutputPDF(t, outputPath, 1)
}

func TestStitchErrorPropagation(t *testing.T) {
	// 存在しないファイルのエラーが正しく伝播することを検証する
	reader := pdf.NewReader(&mockPrompter{})
	writer := pdf.NewWriter()
	bmProc := bookmark.NewProcessor()

	s := stitcher.NewStitcher(reader, writer, bmProc)
	err := s.Run(
		filepath.Join(t.TempDir(), "nonexistent.pdf"),
		filepath.Join(t.TempDir(), "output.pdf"),
		stitcher.StitchOptions{},
	)

	if err == nil {
		t.Fatal("存在しないファイルでエラーが返されるべき")
	}
}
