// Package validate は入力ファイルのバリデーションを提供する。
// パスのサニタイズ、存在確認、PDF拡張子チェック、マジックバイト検証を行う。
//
// 全てのエラーは apperror.NewInputFileError()（終了コード2）で返す。
package validate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
)

// pdfMagicBytes はPDFファイルの先頭に存在するマジックバイト文字列。
// ファイルがPDFであるかの検証に使用する。
const pdfMagicBytes = "%PDF-"

// pdfExtension はPDFファイルの拡張子。
const pdfExtension = ".pdf"

// InputFile は入力ファイルのバリデーションを行う。
//
// 以下の順序で検証を実行する:
//  1. filepath.Clean でパスをサニタイズ
//  2. os.Stat でファイルの存在確認
//  3. ディレクトリでないことを確認
//  4. 拡張子が .pdf（大文字小文字不問）であることを確認
//  5. 先頭5バイトが %PDF- であることを確認（マジックバイト検証）
//
// 検証成功時はサニタイズ済みのパスを返す。
// 検証失敗時は *apperror.ExitError（終了コード2: ExitInputFile）を返す。
//
// 使用例:
//
//	cleanPath, err := validate.InputFile(userInput)
//	if err != nil {
//	    return err // *apperror.ExitError (code 2)
//	}
func InputFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", newStatError(cleanPath, err)
	}

	if info.IsDir() {
		return "", apperror.NewInputFileError(
			fmt.Sprintf("input is a directory, not a file: %s", cleanPath), nil)
	}

	ext := filepath.Ext(cleanPath)
	if !strings.EqualFold(ext, pdfExtension) {
		return "", apperror.NewInputFileError(
			fmt.Sprintf("input must be a PDF file: %s", cleanPath), nil)
	}

	if err := validateMagicBytes(cleanPath); err != nil {
		return "", err
	}

	return cleanPath, nil
}

// newStatError は os.Stat のエラーを適切な ExitError に変換する。
func newStatError(path string, err error) *apperror.ExitError {
	if errors.Is(err, os.ErrNotExist) {
		return apperror.NewInputFileError(
			fmt.Sprintf("input file not found: %s", path), err)
	}
	if errors.Is(err, os.ErrPermission) {
		return apperror.NewInputFileError(
			"cannot access file: permission denied", err)
	}
	return apperror.NewInputFileError(
		fmt.Sprintf("cannot access file: %s", path), err)
}

// validateMagicBytes はファイルの先頭バイトがPDFマジックバイトと一致するか検証する。
// 一致しない場合やファイルが短すぎる場合は ExitError を返す。
func validateMagicBytes(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return apperror.NewInputFileError(
				"cannot access file: permission denied", err)
		}
		return apperror.NewInputFileError(
			fmt.Sprintf("cannot access file: %s", path), err)
	}
	defer f.Close()

	header := make([]byte, len(pdfMagicBytes))
	_, err = io.ReadFull(f, header)
	if err != nil {
		return apperror.NewInputFileError(
			fmt.Sprintf("file is not a valid PDF (invalid header): %s", path), err)
	}

	if string(header) != pdfMagicBytes {
		return apperror.NewInputFileError(
			fmt.Sprintf("file is not a valid PDF (invalid header): %s", path), nil)
	}

	return nil
}
