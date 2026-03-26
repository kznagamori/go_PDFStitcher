package output

import (
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name       string
		inputPath  string
		outputFlag string
		pattern    string
		want       string
	}{
		// -o フラグ指定あり
		{
			name:       "-oフラグ指定時はそのパスを返す",
			inputPath:  filepath.Join("dir", "input.pdf"),
			outputFlag: "custom_output.pdf",
			pattern:    "{input}_stitched.pdf",
			want:       "custom_output.pdf",
		},
		{
			name:       "-oフラグにディレクトリ付きパス",
			inputPath:  filepath.Join("dir", "input.pdf"),
			outputFlag: filepath.Join("other", "out.pdf"),
			pattern:    "{input}_stitched.pdf",
			want:       filepath.Join("other", "out.pdf"),
		},
		// パターン展開
		{
			name:       "デフォルトパターンでの展開",
			inputPath:  filepath.Join("dir", "document.pdf"),
			outputFlag: "",
			pattern:    "{input}_stitched.pdf",
			want:       filepath.Join("dir", "document_stitched.pdf"),
		},
		{
			name:       "入力ファイルと同じディレクトリに出力",
			inputPath:  filepath.Join("path", "to", "report.pdf"),
			outputFlag: "",
			pattern:    "{input}_stitched.pdf",
			want:       filepath.Join("path", "to", "report_stitched.pdf"),
		},
		{
			name:       "カスタムパターン",
			inputPath:  "input.pdf",
			outputFlag: "",
			pattern:    "{input}_merged.pdf",
			want:       "input_merged.pdf",
		},
		{
			name:       "パターンに{input}が複数回含まれる",
			inputPath:  "doc.pdf",
			outputFlag: "",
			pattern:    "{input}_{input}.pdf",
			want:       "doc_doc.pdf",
		},
		{
			name:       "パターンに{input}が含まれない",
			inputPath:  "input.pdf",
			outputFlag: "",
			pattern:    "output.pdf",
			want:       "output.pdf",
		},
		// 境界値
		{
			name:       "日本語ファイル名",
			inputPath:  filepath.Join("dir", "文書.pdf"),
			outputFlag: "",
			pattern:    "{input}_stitched.pdf",
			want:       filepath.Join("dir", "文書_stitched.pdf"),
		},
		{
			name:       "スペースを含むファイル名",
			inputPath:  filepath.Join("my dir", "my file.pdf"),
			outputFlag: "",
			pattern:    "{input}_stitched.pdf",
			want:       filepath.Join("my dir", "my file_stitched.pdf"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.inputPath, tt.outputFlag, tt.pattern)
			if got != tt.want {
				t.Errorf("Resolve() = %q, 期待値 %q", got, tt.want)
			}
		})
	}
}
