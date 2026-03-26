package prompt

import "fmt"

// Confirm はユーザーに Y/N 確認を求める。
//
// message をプロンプトとして Out に出力し、In から1行読み取る。
// "y" または "Y" の場合に true を返す。それ以外は false を返す。
//
// output.Confirmer インターフェースを暗黙的に満たす。
//
// エラーケース:
//   - 入力失敗: "failed to read user input: <os error>"
func (p *Prompter) Confirm(message string) (bool, error) {
	fmt.Fprint(p.Out, message)

	line, err := readLine(p.In)
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	return line == "y" || line == "Y", nil
}
