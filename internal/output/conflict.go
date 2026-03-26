package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// ErrUserCancelled はユーザーが上書きを拒否したことを示すセンチネルエラー。
// cmd/ で errors.Is(err, output.ErrUserCancelled) で判定し、終了コード0で終了する。
var ErrUserCancelled = errors.New("operation cancelled by user")

// Confirmer は上書き確認の対話入力インターフェース。
// prompt.Prompter がこのインターフェースを暗黙的に満たす。
type Confirmer interface {
	// Confirm はユーザーに Y/N 確認を求める。
	Confirm(message string) (bool, error)
}

// ConflictPolicy はファイル衝突時の挙動を定義する型。
type ConflictPolicy string

// ファイル衝突ポリシー定数。
const (
	// PolicyNumbering は連番付与で衝突を回避する。
	PolicyNumbering ConflictPolicy = "numbering"

	// PolicyOverwrite は既存ファイルを上書きする。
	PolicyOverwrite ConflictPolicy = "overwrite"
)

// HandleConflict は出力先ファイルの衝突を処理する。
//
// 出力ディレクトリの存在を検証した後、ファイルの衝突を以下の優先順位で判定する:
//  1. ファイルが存在しない → outputPath をそのまま返す
//  2. force == true → outputPath を返す（上書き）
//  3. confirmer が nil（D&D時）→ policy に従う
//     - PolicyNumbering: (1), (2), ... と連番で空きパスを探す
//     - PolicyOverwrite: outputPath を返す
//  4. confirmer が非nil（CLI時）→ 対話確認
//     - Y → outputPath を返す
//     - N → ErrUserCancelled を返す
//
// エラーケース:
//   - 出力ディレクトリ未存在: ExitError（コード3）
//   - 書き込み権限なし: ExitError（コード3）
//   - 確認プロンプト失敗: ExitError（コード1）
func HandleConflict(outputPath string, policy ConflictPolicy, force bool, confirmer Confirmer) (string, error) {
	if err := validateOutputDir(outputPath); err != nil {
		return "", err
	}

	_, err := os.Stat(outputPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return outputPath, nil
		}
		return "", apperror.NewOutputFileError(
			fmt.Sprintf("cannot access output path: %s", outputPath), err)
	}

	// ファイルが存在する場合の衝突処理
	if force {
		return outputPath, nil
	}

	if confirmer == nil {
		return handlePolicyConflict(outputPath, policy)
	}

	return handleInteractiveConflict(outputPath, confirmer)
}

// validateOutputDir は出力先ディレクトリの存在と書き込み可能性を検証する。
func validateOutputDir(outputPath string) error {
	dir := filepath.Dir(outputPath)

	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apperror.NewOutputFileError(
				fmt.Sprintf("output directory does not exist: %s", dir), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return apperror.NewOutputFileError(
				"cannot write to output path: permission denied", err)
		}
		return apperror.NewOutputFileError(
			fmt.Sprintf("cannot access output directory: %s", dir), err)
	}

	if !info.IsDir() {
		return apperror.NewOutputFileError(
			fmt.Sprintf("output directory does not exist: %s", dir), nil)
	}

	return nil
}

// handlePolicyConflict はD&D時のファイル衝突をポリシーに従って処理する。
func handlePolicyConflict(outputPath string, policy ConflictPolicy) (string, error) {
	if policy == PolicyOverwrite {
		return outputPath, nil
	}

	// PolicyNumbering: 連番で空きパスを探す
	return findAvailablePath(outputPath), nil
}

// handleInteractiveConflict はCLI時のファイル衝突を対話確認で処理する。
func handleInteractiveConflict(outputPath string, confirmer Confirmer) (string, error) {
	message := fmt.Sprintf("overwrite %s? [y/N]: ", outputPath)

	approved, err := confirmer.Confirm(message)
	if err != nil {
		return "", apperror.NewGeneralError(err.Error(), err)
	}

	if approved {
		return outputPath, nil
	}

	return "", ErrUserCancelled
}

// findAvailablePath は連番を付与して衝突しないパスを探す。
// 例: "output_stitched.pdf" → "output_stitched(1).pdf", "output_stitched(2).pdf", ...
func findAvailablePath(basePath string) string {
	ext := filepath.Ext(basePath)
	stem := strings.TrimSuffix(basePath, ext)

	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s(%d)%s", stem, n, ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}
