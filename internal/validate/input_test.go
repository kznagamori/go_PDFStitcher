package validate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// createValidPDF はテスト用の有効なPDFファイルを作成する。
func createValidPDF(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("%PDF-1.4 test content"), 0644); err != nil {
		t.Fatalf("テストPDF作成失敗: %v", err)
	}
	return path
}

// createFile はテスト用のファイルを指定内容で作成する。
func createFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("テストファイル作成失敗: %v", err)
	}
	return path
}

// assertExitError はエラーが指定された終了コードの ExitError であることを検証する。
func assertExitError(t *testing.T, err error, wantCode int) {
	t.Helper()
	var exitErr *apperror.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("エラーが *apperror.ExitError ではない: %T: %v", err, err)
	}
	if exitErr.Code != wantCode {
		t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, wantCode)
	}
}

func TestInputFile(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string // テスト用ファイルパスを返す
		wantErr    bool
		wantCode   int    // 期待する終了コード（wantErr=true 時のみ）
		wantMsgSub string // エラーメッセージに含まれるべき文字列（wantErr=true 時のみ）
	}{
		// 正常系
		{
			name: "有効なPDFファイル",
			setup: func(t *testing.T) string {
				return createValidPDF(t, "test.pdf")
			},
		},
		{
			name: "大文字拡張子PDF",
			setup: func(t *testing.T) string {
				return createValidPDF(t, "test.PDF")
			},
		},
		{
			name: "混合ケース拡張子Pdf",
			setup: func(t *testing.T) string {
				return createValidPDF(t, "test.Pdf")
			},
		},
		{
			name: "スペースを含むパス",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subDir := filepath.Join(dir, "sub dir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					t.Fatalf("ディレクトリ作成失敗: %v", err)
				}
				path := filepath.Join(subDir, "my file.pdf")
				if err := os.WriteFile(path, []byte("%PDF-1.4 content"), 0644); err != nil {
					t.Fatalf("テストPDF作成失敗: %v", err)
				}
				return path
			},
		},
		{
			name: "日本語ファイル名",
			setup: func(t *testing.T) string {
				return createValidPDF(t, "テスト文書.pdf")
			},
		},
		// 異常系: ファイル未存在
		{
			name: "存在しないファイル",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.pdf")
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "input file not found:",
		},
		// 異常系: ディレクトリ指定
		{
			name: "ディレクトリを指定",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "input is a directory, not a file:",
		},
		// 異常系: 拡張子不正
		{
			name: "txt拡張子",
			setup: func(t *testing.T) string {
				return createFile(t, "test.txt", []byte("%PDF-1.4 content"))
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "input must be a PDF file:",
		},
		{
			name: "拡張子なし",
			setup: func(t *testing.T) string {
				return createFile(t, "testfile", []byte("%PDF-1.4 content"))
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "input must be a PDF file:",
		},
		// 異常系: マジックバイト不一致
		{
			name: "PDF拡張子だがマジックバイト不一致",
			setup: func(t *testing.T) string {
				return createFile(t, "fake.pdf", []byte("This is not a PDF file"))
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "file is not a valid PDF (invalid header):",
		},
		{
			name: "空ファイル（0バイト）",
			setup: func(t *testing.T) string {
				return createFile(t, "empty.pdf", []byte{})
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "file is not a valid PDF (invalid header):",
		},
		{
			name: "5バイト未満のファイル",
			setup: func(t *testing.T) string {
				return createFile(t, "short.pdf", []byte("%PD"))
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "file is not a valid PDF (invalid header):",
		},
		// 境界値: 空文字パス
		{
			name: "空文字パス（カレントディレクトリに解決される）",
			setup: func(t *testing.T) string {
				return ""
			},
			wantErr:    true,
			wantCode:   apperror.ExitInputFile,
			wantMsgSub: "input is a directory, not a file:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := InputFile(path)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("InputFile() はエラーを返すべきだが、パス %q が返された", got)
				}
				assertExitError(t, err, tt.wantCode)
				if !strings.Contains(err.Error(), tt.wantMsgSub) {
					t.Errorf("エラーメッセージに %q が含まれていない: %q",
						tt.wantMsgSub, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("InputFile() で予期しないエラー: %v", err)
			}

			// 返却パスが filepath.Clean 済みであることを検証する
			if got != filepath.Clean(path) {
				t.Errorf("InputFile() = %q, 期待値 %q", got, filepath.Clean(path))
			}
		})
	}
}

func TestInputFileReturnsCleanPath(t *testing.T) {
	// 冗長なパス区切りを含むパスが Clean 済みで返されることを検証する
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 content"), 0644); err != nil {
		t.Fatalf("テストPDF作成失敗: %v", err)
	}

	// 冗長なパスを作成（"dir/./test.pdf"）
	redundantPath := filepath.Join(dir, ".", "test.pdf")
	got, err := InputFile(redundantPath)
	if err != nil {
		t.Fatalf("InputFile() で予期しないエラー: %v", err)
	}

	expected := filepath.Clean(redundantPath)
	if got != expected {
		t.Errorf("InputFile() = %q, 期待値 %q (Clean済み)", got, expected)
	}
}

func TestInputFileErrorTypes(t *testing.T) {
	// 全エラーが *apperror.ExitError であり、errors.As で判定可能なことを検証する
	_, err := InputFile(filepath.Join(t.TempDir(), "nonexistent.pdf"))
	if err == nil {
		t.Fatal("エラーが返されるべき")
	}

	var exitErr *apperror.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("errors.As で ExitError を取得できない: %T", err)
	}

	if exitErr.Code != apperror.ExitInputFile {
		t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, apperror.ExitInputFile)
	}
}
