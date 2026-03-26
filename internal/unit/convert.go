// Package unit は pt/mm 単位変換と --gap 値のパース（サフィックス解析）を提供する。
// 外部依存を持たない純粋な計算パッケージである。
//
// cmd/ から ParseGap を呼び出し、stitcher に渡す pt 単位のガター値を取得する。
// apperror に依存せず、標準 error のみを返す。
package unit

import (
	"fmt"
	"strconv"
	"strings"
)

// 単位変換に使用する定数。
const (
	// PointsPerInch は 1インチあたりのポイント数（72.0）。
	PointsPerInch = 72.0

	// MmPerInch は 1インチあたりのミリメートル数（25.4）。
	MmPerInch = 25.4
)

// 単位サフィックスの定数。マジックナンバーの使用を回避する。
const (
	unitPt = "pt"
	unitMm = "mm"
)

// ParseGap はガター値文字列をパースし、pt 単位の値を返す。
//
// value のフォーマットは "10pt", "5mm", "20"（サフィックスなし）。
// サフィックスが省略された場合は defaultUnit に従って単位を決定する。
// defaultUnit が不正な値（"pt" / "mm" 以外）の場合は "pt" として扱う。
//
// エラーケース:
//   - 数値パース失敗: 標準 error を返す
//   - 負の値: 標準 error を返す
//
// 使用例:
//
//	gapPt, err := unit.ParseGap("10mm", "pt")  // gapPt ≈ 28.35
//	gapPt, err := unit.ParseGap("20", "pt")    // gapPt = 20.0
func ParseGap(value string, defaultUnit string) (float64, error) {
	numStr, u := splitSuffix(value)

	// サフィックスなしの場合は defaultUnit を適用
	if u == "" {
		u = normalizeUnit(defaultUnit)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid gap value %q: expected number with optional unit (e.g., 10pt, 5mm)", value)
	}

	if num < 0 {
		return 0, fmt.Errorf("gap value must be non-negative: %s", value)
	}

	if u == unitMm {
		return MmToPoints(num), nil
	}

	return num, nil
}

// MmToPoints は mm 値を pt 値に変換する。
//
// 変換式: mm * PointsPerInch / MmPerInch
//
// 使用例:
//
//	pts := unit.MmToPoints(25.4) // = 72.0
func MmToPoints(mm float64) float64 {
	return mm * PointsPerInch / MmPerInch
}

// PointsToMm は pt 値を mm 値に変換する。
//
// 変換式: pt * MmPerInch / PointsPerInch
//
// 使用例:
//
//	mm := unit.PointsToMm(72.0) // = 25.4
func PointsToMm(pt float64) float64 {
	return pt * MmPerInch / PointsPerInch
}

// splitSuffix は値文字列から単位サフィックス（"pt" / "mm"）を分離する。
// サフィックスが見つからない場合は空文字列を返す。
func splitSuffix(value string) (numPart string, unit string) {
	lower := strings.ToLower(value)

	if strings.HasSuffix(lower, unitMm) {
		return value[:len(value)-len(unitMm)], unitMm
	}

	if strings.HasSuffix(lower, unitPt) {
		return value[:len(value)-len(unitPt)], unitPt
	}

	return value, ""
}

// normalizeUnit は単位文字列を正規化する。
// "pt" / "mm"（大文字小文字不問）以外の値は "pt" にフォールバックする。
func normalizeUnit(u string) string {
	switch strings.ToLower(u) {
	case unitMm:
		return unitMm
	case unitPt:
		return unitPt
	default:
		return unitPt
	}
}
