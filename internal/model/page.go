// Package model は複数パッケージ間で共有するデータ型を定義する。
// このパッケージは依存を持たず、ロジックを含まない純粋なデータ構造のみを提供する。
package model

// Page はPDFの1ページ分の情報を表す。
// Width と Height はPDFユーザー単位（ポイント、1pt = 1/72インチ）で表現する。
// Content は pdfcpu 固有のページコンテンツ参照であり、pdf パッケージのみが解釈する。
type Page struct {
	Index   int         // 0始まりのページインデックス
	Width   float64     // ページ幅（pt）
	Height  float64     // ページ高さ（pt）
	Content interface{} // pdfcpu固有のページコンテンツ参照（pdfパッケージのみが解釈）
}
