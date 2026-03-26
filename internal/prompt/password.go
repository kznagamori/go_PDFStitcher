// Package prompt は対話入力処理を提供する。
// パスワード入力（エコーバック無効）と上書き確認（Y/N入力）を担当する。
//
// テスト時は io.Reader/io.Writer を Prompter の In/Out フィールドに注入して
// モック化できる。
//
// apperror に依存しない。標準 error のみを返す。
// 呼び出し元（pdf.Reader, cmd/）が適切な ExitError にラップする。
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Prompter は対話入力の実装である。
// In と Out を差し替えることでテスト時にモック化できる。
//
// 使用例（本番）:
//
//	p := prompt.NewPrompter()
//	password, err := p.Password("Enter PDF password: ")
//
// 使用例（テスト）:
//
//	p := &prompt.Prompter{
//	    In:  bytes.NewBufferString("mypassword\n"),
//	    Out: &bytes.Buffer{},
//	}
//	password, err := p.Password("Enter PDF password: ")
type Prompter struct {
	In  io.Reader // 入力元（本番: os.Stdin、テスト: bytes.Buffer 等）
	Out io.Writer // 出力先（本番: os.Stderr、テスト: bytes.Buffer 等）
}

// NewPrompter は標準入出力を使った Prompter を生成する。
// In = os.Stdin、Out = os.Stderr で初期化する。
func NewPrompter() *Prompter {
	return &Prompter{
		In:  os.Stdin,
		Out: os.Stderr,
	}
}

// Password はパスワード入力を求める。
//
// message をプロンプトとして Out に出力し、入力を待つ。
// In が *os.File の場合は term.ReadPassword でエコーバック無効の入力を取得する。
// それ以外（テスト時）は In から1行読み取る。
//
// エラーケース:
//   - 入力失敗: "failed to read password: <os error>"
//   - 空パスワード: "password cannot be empty"
func (p *Prompter) Password(message string) (string, error) {
	fmt.Fprint(p.Out, message)

	var password string

	if f, ok := p.In.(*os.File); ok {
		// 本番環境: エコーバック無効で読み取り
		raw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(p.Out) // term.ReadPassword は改行を出力しないため補完する
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		password = string(raw)
	} else {
		// テスト環境: io.Reader から直接読み取り
		line, err := readLine(p.In)
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		password = line
	}

	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	return password, nil
}

// readLine は io.Reader から1行読み取り、改行を除去して返す。
// EOF に達しデータがない場合は io.EOF を返す。
func readLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}
