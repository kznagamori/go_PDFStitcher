package prompt

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewPrompter(t *testing.T) {
	p := NewPrompter()

	if p == nil {
		t.Fatal("NewPrompter() は nil を返すべきではない")
	}
	if p.In != os.Stdin {
		t.Error("In は os.Stdin であるべき")
	}
	if p.Out != os.Stderr {
		t.Error("Out は os.Stderr であるべき")
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		want    string
		wantErr bool
	}{
		// 正常系
		{
			name:    "有効なパスワード",
			input:   "mypassword\n",
			message: "Enter PDF password: ",
			want:    "mypassword",
		},
		{
			name:    "特殊文字を含むパスワード",
			input:   "p@$$w0rd!#%\n",
			message: "Enter PDF password: ",
			want:    "p@$$w0rd!#%",
		},
		{
			name:    "スペースを含むパスワード",
			input:   "my pass word\n",
			message: "Enter PDF password: ",
			want:    "my pass word",
		},
		{
			name:    "長いパスワード",
			input:   strings.Repeat("a", 256) + "\n",
			message: "Enter PDF password: ",
			want:    strings.Repeat("a", 256),
		},
		{
			name:    "日本語パスワード",
			input:   "パスワード123\n",
			message: "Enter PDF password: ",
			want:    "パスワード123",
		},
		// 異常系
		{
			name:    "空パスワード（改行のみ）",
			input:   "\n",
			message: "Enter PDF password: ",
			wantErr: true,
		},
		{
			name:    "EOF（空リーダー）",
			input:   "",
			message: "Enter PDF password: ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			p := &Prompter{
				In:  bytes.NewBufferString(tt.input),
				Out: out,
			}

			got, err := p.Password(tt.message)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Password() はエラーを返すべきだが、値 %q が返された", got)
				}
				return
			}

			if err != nil {
				t.Errorf("Password() で予期しないエラー: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("Password() = %q, 期待値 %q", got, tt.want)
			}

			// プロンプトメッセージが Out に出力されていることを検証する
			if !strings.Contains(out.String(), tt.message) {
				t.Errorf("Out にメッセージ %q が含まれていない。実際の出力: %q",
					tt.message, out.String())
			}
		})
	}
}

func TestPasswordErrorMessages(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{
			name:    "空パスワードのエラーメッセージ",
			input:   "\n",
			wantMsg: "password cannot be empty",
		},
		{
			name:    "EOF時のエラーメッセージ",
			input:   "",
			wantMsg: "failed to read password: EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prompter{
				In:  bytes.NewBufferString(tt.input),
				Out: &bytes.Buffer{},
			}

			_, err := p.Password("Enter PDF password: ")
			if err == nil {
				t.Fatal("Password() はエラーを返すべきだが、nil が返された")
			}

			if err.Error() != tt.wantMsg {
				t.Errorf("エラーメッセージ = %q, 期待値 %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		want    bool
		wantErr bool
	}{
		// 正常系: true を返すケース
		{
			name:    "小文字y",
			input:   "y\n",
			message: "Overwrite? [y/N]: ",
			want:    true,
		},
		{
			name:    "大文字Y",
			input:   "Y\n",
			message: "Overwrite? [y/N]: ",
			want:    true,
		},
		// 正常系: false を返すケース
		{
			name:    "小文字n",
			input:   "n\n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		{
			name:    "大文字N",
			input:   "N\n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		{
			name:    "空入力（改行のみ）",
			input:   "\n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		{
			name:    "yesはfalse（yのみがtrue）",
			input:   "yes\n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		{
			name:    "任意の文字列はfalse",
			input:   "anything\n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		{
			name:    "スペース付きyはfalse",
			input:   " y \n",
			message: "Overwrite? [y/N]: ",
			want:    false,
		},
		// 異常系
		{
			name:    "EOF（空リーダー）",
			input:   "",
			message: "Overwrite? [y/N]: ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			p := &Prompter{
				In:  bytes.NewBufferString(tt.input),
				Out: out,
			}

			got, err := p.Confirm(tt.message)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Confirm() はエラーを返すべきだが、値 %v が返された", got)
				}
				return
			}

			if err != nil {
				t.Errorf("Confirm() で予期しないエラー: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("Confirm() = %v, 期待値 %v", got, tt.want)
			}

			// プロンプトメッセージが Out に出力されていることを検証する
			if !strings.Contains(out.String(), tt.message) {
				t.Errorf("Out にメッセージ %q が含まれていない。実際の出力: %q",
					tt.message, out.String())
			}
		})
	}
}

func TestConfirmErrorMessage(t *testing.T) {
	p := &Prompter{
		In:  bytes.NewBufferString(""),
		Out: &bytes.Buffer{},
	}

	_, err := p.Confirm("Overwrite? [y/N]: ")
	if err == nil {
		t.Fatal("Confirm() はエラーを返すべきだが、nil が返された")
	}

	wantMsg := "failed to read user input: EOF"
	if err.Error() != wantMsg {
		t.Errorf("エラーメッセージ = %q, 期待値 %q", err.Error(), wantMsg)
	}
}
