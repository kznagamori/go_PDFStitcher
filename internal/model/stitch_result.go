package model

// AlignedPage は幅揃え後のページ配置情報を保持する。
// XOffset は配置方針（left/center/right）に基づいて計算される。
//
// 計算例:
//
//	left:   XOffset = 0
//	center: XOffset = (maxWidth - page.Width) / 2
//	right:  XOffset = maxWidth - page.Width
type AlignedPage struct {
	Page    Page    // 元ページ情報
	XOffset float64 // 幅揃えによるX座標オフセット（pt）
}

// StitchResult はPDF連結処理の結果を保持する。
// stitcher パッケージで生成し、pdf.Writer に渡して出力する。
// model パッケージに配置することで pdf → stitcher の逆方向依存を回避する。
//
// Content は pdfcpu 固有の連結済みデータであり、pdf パッケージのみが解釈する。
type StitchResult struct {
	AlignedPages []AlignedPage // 幅揃え済みページ（水平オフセット付き）
	Bookmarks    []Bookmark    // 調整済みまたは自動生成されたしおり
	TotalHeight  float64       // 連結後の総高さ（pt）
	MaxWidth     float64       // 最大ページ幅（pt）
	GapPt        float64       // ページ間ガター幅（pt）
	Content      interface{}   // pdfcpu固有の連結済みコンテンツ（pdfパッケージのみが解釈）
}
