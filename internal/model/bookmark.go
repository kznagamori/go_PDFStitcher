package model

// Bookmark はPDFしおり（アウトライン）ツリーの1エントリを表す。
// 階層構造は Children フィールドで表現する。
// Y は連結後PDFにおける縦方向の座標（ポイント単位、上端基準）。
type Bookmark struct {
	Title    string     // しおりの表示テキスト
	Level    int        // 階層レベル（0が最上位）
	PageIdx  int        // 参照先の元ページインデックス（0始まり）
	Y        float64    // 連結後PDFにおけるY座標（pt、上端基準）
	Children []Bookmark // 子しおり（階層構造を保持）
}
