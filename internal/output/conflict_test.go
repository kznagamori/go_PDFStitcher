package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// mockConfirmer は Confirmer インターフェースのテスト用モック。
type mockConfirmer struct {
	response bool
	err      error
}

// Confirm はモックの応答を返す。
func (m *mockConfirmer) Confirm(_ string) (bool, error) {
	return m.response, m.err
}

// createExistingFile はテスト用のファイルを作成する。
func createExistingFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatalf("テスト用ファイル作成失敗: %v", err)
	}
}

func TestHandleConflict(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // 出力パスを返す
		policy    ConflictPolicy
		force     bool
		confirmer Confirmer
		wantPath  string // 期待する最終パス（相対）。空文字の場合は setup の返値を期待
		wantErr   error  // ErrUserCancelled 等
		wantCode  int    // ExitError 期待時のコード（-1 で未チェック）
	}{
		// 衝突なし
		{
			name: "ファイルが存在しない場合はそのまま返す",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "output.pdf")
			},
			policy: PolicyNumbering,
		},
		// force=true
		{
			name: "force=trueで既存ファイルを上書き",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy: PolicyNumbering,
			force:  true,
		},
		// confirmer=nil + PolicyNumbering
		{
			name: "D&D+numbering: 連番(1)付与",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy:   PolicyNumbering,
			wantPath: "output(1).pdf",
		},
		{
			name: "D&D+numbering: (1)が使用済みなら(2)付与",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				createExistingFile(t, filepath.Join(dir, "output(1).pdf"))
				return path
			},
			policy:   PolicyNumbering,
			wantPath: "output(2).pdf",
		},
		// confirmer=nil + PolicyOverwrite
		{
			name: "D&D+overwrite: そのまま返す",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy: PolicyOverwrite,
		},
		// confirmer 承認
		{
			name: "CLI+承認: 上書きパスを返す",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy:    PolicyNumbering,
			confirmer: &mockConfirmer{response: true},
		},
		// confirmer 拒否
		{
			name: "CLI+拒否: ErrUserCancelled",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy:    PolicyNumbering,
			confirmer: &mockConfirmer{response: false},
			wantErr:   ErrUserCancelled,
		},
		// confirmer エラー
		{
			name: "CLI+入力エラー: ExitError(1)",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "output.pdf")
				createExistingFile(t, path)
				return path
			},
			policy:    PolicyNumbering,
			confirmer: &mockConfirmer{err: fmt.Errorf("failed to read user input: EOF")},
			wantCode:  apperror.ExitGeneral,
		},
		// 出力ディレクトリ未存在
		{
			name: "出力ディレクトリ未存在でExitError(3)",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent_dir", "output.pdf")
			},
			policy:   PolicyNumbering,
			wantCode: apperror.ExitOutputFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := tt.setup(t)

			got, err := HandleConflict(outputPath, tt.policy, tt.force, tt.confirmer)

			// エラー期待（ExitError コードチェック）
			if tt.wantCode > 0 {
				if err == nil {
					t.Fatalf("HandleConflict() はエラーを返すべき")
				}
				var exitErr *apperror.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("エラーが *apperror.ExitError ではない: %T: %v", err, err)
				}
				if exitErr.Code != tt.wantCode {
					t.Errorf("終了コード = %d, 期待値 %d", exitErr.Code, tt.wantCode)
				}
				return
			}

			// センチネルエラー期待
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("HandleConflict() error = %v, 期待 %v", err, tt.wantErr)
				}
				return
			}

			// 成功期待
			if err != nil {
				t.Fatalf("HandleConflict() で予期しないエラー: %v", err)
			}

			// パス検証
			if tt.wantPath != "" {
				wantFull := filepath.Join(filepath.Dir(outputPath), tt.wantPath)
				if got != wantFull {
					t.Errorf("HandleConflict() = %q, 期待値 %q", got, wantFull)
				}
			} else {
				if got != outputPath {
					t.Errorf("HandleConflict() = %q, 期待値 %q", got, outputPath)
				}
			}
		})
	}
}

func TestHandleConflictNumberingBoundary(t *testing.T) {
	// (1)〜(99)まで使用済みの場合に (100) が付与されることを検証する
	dir := t.TempDir()
	basePath := filepath.Join(dir, "output.pdf")
	createExistingFile(t, basePath)

	// (1)〜(99) を作成する
	for i := 1; i <= 99; i++ {
		createExistingFile(t, filepath.Join(dir, fmt.Sprintf("output(%d).pdf", i)))
	}

	got, err := HandleConflict(basePath, PolicyNumbering, false, nil)
	if err != nil {
		t.Fatalf("HandleConflict() で予期しないエラー: %v", err)
	}

	want := filepath.Join(dir, "output(100).pdf")
	if got != want {
		t.Errorf("HandleConflict() = %q, 期待値 %q", got, want)
	}
}

func TestHandleConflictNoConflict(t *testing.T) {
	// 衝突がない場合に outputPath がそのまま返されることを検証する
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "new_output.pdf")

	got, err := HandleConflict(outputPath, PolicyNumbering, false, nil)
	if err != nil {
		t.Fatalf("HandleConflict() で予期しないエラー: %v", err)
	}
	if got != outputPath {
		t.Errorf("HandleConflict() = %q, 期待値 %q", got, outputPath)
	}
}

func TestErrUserCancelledIsDetectable(t *testing.T) {
	// errors.Is で ErrUserCancelled を判定できることを検証する
	dir := t.TempDir()
	path := filepath.Join(dir, "output.pdf")
	createExistingFile(t, path)

	_, err := HandleConflict(path, PolicyNumbering, false, &mockConfirmer{response: false})
	if !errors.Is(err, ErrUserCancelled) {
		t.Errorf("errors.Is(err, ErrUserCancelled) = false, 期待 true")
	}
}
