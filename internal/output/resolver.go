// Package output は出力ファイルパスの解決とファイル衝突処理を提供する。
//
// Resolve で出力パスを決定し、HandleConflict で既存ファイルとの衝突を処理する。
// 上書き確認は Confirmer インターフェース経由で対話的に行う。
package output

import (
	"path/filepath"
	"strings"
)

// inputPlaceholder は出力パターン中の入力ファイル名プレースホルダー。
const inputPlaceholder = "{input}"

// Resolve は出力ファイルパスを決定する。
//
// outputFlag が非空の場合はそのまま filepath.Clean して返す。
// outputFlag が空の場合は pattern 中の {input} を入力ファイルのベース名（拡張子なし）に
// 置換し、入力ファイルと同じディレクトリに配置する。
//
// 常に非空の filepath.Clean 済み文字列を返す。
//
// 使用例:
//
//	output.Resolve("doc.pdf", "", "{input}_stitched.pdf")
//	// → "doc_stitched.pdf"
//
//	output.Resolve("doc.pdf", "out.pdf", "{input}_stitched.pdf")
//	// → "out.pdf"
func Resolve(inputPath string, outputFlag string, pattern string) string {
	if outputFlag != "" {
		return filepath.Clean(outputFlag)
	}

	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	outputName := strings.ReplaceAll(pattern, inputPlaceholder, name)

	return filepath.Clean(filepath.Join(dir, outputName))
}
