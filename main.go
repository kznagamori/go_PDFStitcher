// Package main は go_PDFStitcher のエントリポイントである。
// 全てのロジックは cmd パッケージに委譲する。
package main

import "github.com/kznagamori/go_PDFStitcher/cmd"

// main は cmd.Execute() を呼び出す。
// このファイルにはそれ以外のロジックを含めない。
func main() {
	cmd.Execute()
}
