package unit

import (
	"math"
	"testing"
)

// テスト用の浮動小数点比較の許容誤差。
const tolerance = 1e-9

// almostEqual は2つの float64 値が許容誤差内で等しいかを判定する。
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestParseGap(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		defaultUnit string
		want        float64
		wantErr     bool
	}{
		// 正常系: pt サフィックス
		{
			name:        "ptサフィックス付き整数",
			value:       "10pt",
			defaultUnit: "pt",
			want:        10.0,
		},
		{
			name:        "ptサフィックス付き小数",
			value:       "5.5pt",
			defaultUnit: "pt",
			want:        5.5,
		},
		{
			name:        "ptサフィックス大文字",
			value:       "10PT",
			defaultUnit: "pt",
			want:        10.0,
		},
		{
			name:        "ptサフィックス混合ケース",
			value:       "10Pt",
			defaultUnit: "pt",
			want:        10.0,
		},
		// 正常系: mm サフィックス
		{
			name:        "mmサフィックス付き整数",
			value:       "5mm",
			defaultUnit: "pt",
			want:        MmToPoints(5.0),
		},
		{
			name:        "mmサフィックス付き小数",
			value:       "2.5mm",
			defaultUnit: "pt",
			want:        MmToPoints(2.5),
		},
		{
			name:        "mmサフィックス大文字",
			value:       "5MM",
			defaultUnit: "pt",
			want:        MmToPoints(5.0),
		},
		{
			name:        "25.4mmは72ptに変換される",
			value:       "25.4mm",
			defaultUnit: "pt",
			want:        72.0,
		},
		// 正常系: サフィックスなし（defaultUnit適用）
		{
			name:        "サフィックスなしデフォルトpt",
			value:       "20",
			defaultUnit: "pt",
			want:        20.0,
		},
		{
			name:        "サフィックスなしデフォルトmm",
			value:       "10",
			defaultUnit: "mm",
			want:        MmToPoints(10.0),
		},
		{
			name:        "サフィックスなし小数デフォルトpt",
			value:       "3.14",
			defaultUnit: "pt",
			want:        3.14,
		},
		// 正常系: 境界値
		{
			name:        "ゼロ値pt",
			value:       "0",
			defaultUnit: "pt",
			want:        0.0,
		},
		{
			name:        "ゼロ値mm",
			value:       "0mm",
			defaultUnit: "pt",
			want:        0.0,
		},
		{
			name:        "ゼロ小数pt",
			value:       "0.0pt",
			defaultUnit: "pt",
			want:        0.0,
		},
		{
			name:        "非常に大きな値",
			value:       "99999.99pt",
			defaultUnit: "pt",
			want:        99999.99,
		},
		{
			name:        "非常に小さな正の値",
			value:       "0.001pt",
			defaultUnit: "pt",
			want:        0.001,
		},
		// 正常系: defaultUnit の正規化
		{
			name:        "defaultUnit大文字PT",
			value:       "10",
			defaultUnit: "PT",
			want:        10.0,
		},
		{
			name:        "defaultUnit大文字MM",
			value:       "10",
			defaultUnit: "MM",
			want:        MmToPoints(10.0),
		},
		{
			name:        "defaultUnit不正値はptにフォールバック",
			value:       "10",
			defaultUnit: "invalid",
			want:        10.0,
		},
		{
			name:        "defaultUnit空文字列はptにフォールバック",
			value:       "10",
			defaultUnit: "",
			want:        10.0,
		},
		// 異常系: パース失敗
		{
			name:        "文字列のみ",
			value:       "abc",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "空文字列",
			value:       "",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "単位のみ",
			value:       "pt",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "単位のみmm",
			value:       "mm",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "不正な単位サフィックス",
			value:       "10xyz",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "スペース含む",
			value:       "10 pt",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "小数点のみ",
			value:       ".",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "複数の小数点",
			value:       "1.2.3pt",
			defaultUnit: "pt",
			wantErr:     true,
		},
		// 異常系: 負の値
		{
			name:        "負の値pt",
			value:       "-5pt",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "負の値mm",
			value:       "-3mm",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "負の値サフィックスなし",
			value:       "-10",
			defaultUnit: "pt",
			wantErr:     true,
		},
		{
			name:        "負の小数値",
			value:       "-0.5pt",
			defaultUnit: "pt",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGap(tt.value, tt.defaultUnit)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGap(%q, %q) はエラーを返すべきだが、値 %f が返された",
						tt.value, tt.defaultUnit, got)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseGap(%q, %q) で予期しないエラー: %v",
					tt.value, tt.defaultUnit, err)
				return
			}

			if !almostEqual(got, tt.want) {
				t.Errorf("ParseGap(%q, %q) = %f, 期待値 %f",
					tt.value, tt.defaultUnit, got, tt.want)
			}
		})
	}
}

func TestParseGapErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		defaultUnit string
		wantMsg     string
	}{
		{
			name:        "パース失敗のエラーメッセージ",
			value:       "abc",
			defaultUnit: "pt",
			wantMsg:     `invalid gap value "abc": expected number with optional unit (e.g., 10pt, 5mm)`,
		},
		{
			name:        "負の値のエラーメッセージ",
			value:       "-5",
			defaultUnit: "pt",
			wantMsg:     "gap value must be non-negative: -5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGap(tt.value, tt.defaultUnit)
			if err == nil {
				t.Fatal("ParseGap はエラーを返すべきだが、nil が返された")
			}

			if err.Error() != tt.wantMsg {
				t.Errorf("エラーメッセージ = %q, 期待値 %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestMmToPoints(t *testing.T) {
	tests := []struct {
		name string
		mm   float64
		want float64
	}{
		{
			name: "1インチ(25.4mm)は72ptに変換される",
			mm:   25.4,
			want: 72.0,
		},
		{
			name: "0mmは0ptに変換される",
			mm:   0.0,
			want: 0.0,
		},
		{
			name: "1mmの変換",
			mm:   1.0,
			want: PointsPerInch / MmPerInch,
		},
		{
			name: "10mmの変換",
			mm:   10.0,
			want: 10.0 * PointsPerInch / MmPerInch,
		},
		{
			name: "A4幅(210mm)の変換",
			mm:   210.0,
			want: 210.0 * PointsPerInch / MmPerInch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MmToPoints(tt.mm)
			if !almostEqual(got, tt.want) {
				t.Errorf("MmToPoints(%f) = %f, 期待値 %f", tt.mm, got, tt.want)
			}
		})
	}
}

func TestPointsToMm(t *testing.T) {
	tests := []struct {
		name string
		pt   float64
		want float64
	}{
		{
			name: "72ptは25.4mm(1インチ)に変換される",
			pt:   72.0,
			want: 25.4,
		},
		{
			name: "0ptは0mmに変換される",
			pt:   0.0,
			want: 0.0,
		},
		{
			name: "1ptの変換",
			pt:   1.0,
			want: MmPerInch / PointsPerInch,
		},
		{
			name: "A4幅(595.28pt)の変換",
			pt:   595.28,
			want: 595.28 * MmPerInch / PointsPerInch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PointsToMm(tt.pt)
			if !almostEqual(got, tt.want) {
				t.Errorf("PointsToMm(%f) = %f, 期待値 %f", tt.pt, got, tt.want)
			}
		})
	}
}

func TestMmToPointsRoundTrip(t *testing.T) {
	// mm → pt → mm の往復変換で値が保持されることを検証する
	tests := []float64{0.0, 1.0, 10.0, 25.4, 100.0, 210.0, 297.0}

	for _, mm := range tests {
		pt := MmToPoints(mm)
		got := PointsToMm(pt)
		if !almostEqual(got, mm) {
			t.Errorf("往復変換 MmToPoints → PointsToMm で値が変化: 入力 %f, 結果 %f", mm, got)
		}
	}
}
