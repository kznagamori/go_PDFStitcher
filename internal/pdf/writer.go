package pdf

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfModel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
	"github.com/kznagamori/go_PDFStitcher/internal/model"
)

// Writer は pdfcpu を使った PDF 書き込みの実装である。
// stitcher.PageWriter インターフェースを満たす。
// 一時ファイル→リネーム方式で安全に出力する。
type Writer struct {
	tmpPath string // 現在の一時ファイルパス（Cleanup用）
}

// NewWriter は Writer を生成する。
func NewWriter() *Writer {
	return &Writer{}
}

// Write は連結済みの1ページPDFを出力ファイルに書き込む。
// 一時ファイルに書き込み後、os.Rename で最終パスに移動する。
// 失敗時は一時ファイルを自動的にクリーンアップする。
//
// エラーケース:
//   - 一時ファイル作成失敗: ExitError（コード3）
//   - 書き込み失敗: ExitError（コード3）
//   - リネーム失敗: ExitError（コード3）
func (w *Writer) Write(outputPath string, result *model.StitchResult) error {
	tmpPath := outputPath + ".tmp"
	w.tmpPath = tmpPath

	slog.Debug("writing output PDF", "tmp_path", tmpPath, "output_path", outputPath)

	if err := w.buildStitchedPDF(tmpPath, result); err != nil {
		w.Cleanup()
		return err
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		w.Cleanup()
		return apperror.NewOutputFileError(
			fmt.Sprintf("failed to rename output file: %s", err), err)
	}

	w.tmpPath = ""
	slog.Info("output PDF written", "path", outputPath)
	return nil
}

// Cleanup は一時ファイルを削除する。
// シグナルハンドリングから呼ばれる。一時ファイルが存在しなくてもエラーにしない。
func (w *Writer) Cleanup() {
	if w.tmpPath == "" {
		return
	}
	if err := os.Remove(w.tmpPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to cleanup temp file", "error", err, "path", w.tmpPath)
	}
	w.tmpPath = ""
}

// buildStitchedPDF は Form XObject 方式で連結済みPDFを生成し、指定パスに書き込む。
// 各ソースページを Form XObject として登録し、1ページの出力PDFに正確な座標で配置する。
func (w *Writer) buildStitchedPDF(tmpPath string, result *model.StitchResult) error {
	if len(result.AlignedPages) == 0 {
		return apperror.NewGeneralError("no pages to stitch", nil)
	}

	pc, ok := result.AlignedPages[0].Page.Content.(*pageContent)
	if !ok {
		return apperror.NewGeneralError("internal error: invalid page content type", nil)
	}
	srcCtx := pc.ctx
	xrt := srcCtx.XRefTable

	// 各ソースページから Form XObject を作成し、配置コマンドを構築する
	xobjectDict := types.Dict{}
	var contentBuf bytes.Buffer

	yAccum := 0.0 // モデル座標系でのY累積（上端基準）
	for i, ap := range result.AlignedPages {
		pgContent, ok := ap.Page.Content.(*pageContent)
		if !ok {
			return apperror.NewGeneralError("internal error: invalid page content type", nil)
		}

		xobjRef, bbox, err := createFormXObject(xrt, pgContent.pageNum)
		if err != nil {
			return apperror.NewOutputFileError(
				fmt.Sprintf("failed to create form XObject for page %d: %s", i+1, err), err)
		}

		xobjName := fmt.Sprintf("Fm%d", i)
		xobjectDict[xobjName] = *xobjRef

		// PDF座標系（左下原点）への変換
		// PDF Y = totalHeight - modelY - pageHeight
		pdfY := result.TotalHeight - yAccum - bbox.Height()
		fmt.Fprintf(&contentBuf, "q 1 0 0 1 %f %f cm /%s Do Q\n",
			ap.XOffset, pdfY, xobjName)

		yAccum += ap.Page.Height + result.GapPt
	}

	// 出力ページを作成し、ページツリーを差し替える
	outMediaBox := types.RectForDim(result.MaxWidth, result.TotalHeight)

	contentBytes := contentBuf.Bytes()
	contentSD, err := xrt.NewStreamDictForBuf(contentBytes)
	if err != nil {
		return apperror.NewOutputFileError("failed to create content stream", err)
	}
	if err := contentSD.Encode(); err != nil {
		return apperror.NewOutputFileError("failed to encode content stream", err)
	}
	contentRef, err := xrt.IndRefForNewObject(*contentSD)
	if err != nil {
		return apperror.NewOutputFileError("failed to register content stream", err)
	}

	resDict := types.Dict{
		"XObject": xobjectDict,
	}

	// Pages辞書の IndirectRef を取得する
	catalogDict, err := xrt.Catalog()
	if err != nil {
		return apperror.NewOutputFileError("failed to get catalog", err)
	}
	pagesRef, ok := catalogDict.Find("Pages")
	if !ok {
		return apperror.NewOutputFileError("catalog missing Pages entry", nil)
	}
	pagesIndRef, ok := pagesRef.(types.IndirectRef)
	if !ok {
		return apperror.NewOutputFileError("invalid Pages reference in catalog", nil)
	}

	pageDict := types.Dict{
		"Type":      types.Name("Page"),
		"Parent":    pagesIndRef,
		"MediaBox":  outMediaBox.Array(),
		"Resources": resDict,
		"Contents":  *contentRef,
	}
	pageRef, err := xrt.IndRefForNewObject(pageDict)
	if err != nil {
		return apperror.NewOutputFileError("failed to register output page", err)
	}

	// Pages辞書を更新: Kids を出力ページ1件のみに差し替える
	pagesDict, err := xrt.DereferenceDict(pagesIndRef)
	if err != nil {
		return apperror.NewOutputFileError("failed to dereference Pages dict", err)
	}
	pagesDict["Kids"] = types.Array{*pageRef}
	pagesDict["Count"] = types.Integer(1)
	xrt.PageCount = 1

	// しおりを設定する
	if err := w.setBookmarks(srcCtx, result.Bookmarks, result.TotalHeight); err != nil {
		slog.Warn("failed to set bookmarks", "error", err)
	}

	// 一時ファイルに書き出す
	if err := api.WriteContextFile(srcCtx, tmpPath); err != nil {
		return apperror.NewOutputFileError(
			fmt.Sprintf("failed to write output PDF: %s", err), err)
	}

	return nil
}

// createFormXObject はソースページから Form XObject を作成し、XRefTable に登録する。
// ページのコンテンツストリームとリソースを Form XObject にコピーする。
func createFormXObject(xrt *pdfModel.XRefTable, pageNum int) (*types.IndirectRef, *types.Rectangle, error) {
	pageDict, _, inhPAttrs, err := xrt.PageDict(pageNum, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get page dict: %w", err)
	}

	// MediaBox を取得する（ページ直接 or 継承）
	var mediaBox *types.Rectangle
	if mb := pageDict.ArrayEntry("MediaBox"); mb != nil {
		mediaBox = types.RectForArray(mb)
	} else if inhPAttrs != nil && inhPAttrs.MediaBox != nil {
		mediaBox = inhPAttrs.MediaBox
	}
	if mediaBox == nil {
		return nil, nil, fmt.Errorf("page %d has no MediaBox", pageNum)
	}

	// コンテンツストリームのバイト列を取得する
	// コンテンツが存在しないページ（空白ページ）は空のストリームで処理する
	contentBytes, err := xrt.PageContent(pageDict, pageNum)
	if err != nil {
		contentBytes = []byte{}
	}

	// リソース辞書を取得する（ページ直接 or 継承）
	var resources types.Dict
	if r := pageDict.DictEntry("Resources"); r != nil {
		resources = r
	} else if inhPAttrs != nil && inhPAttrs.Resources != nil {
		resources = inhPAttrs.Resources
	}

	// Form XObject StreamDict を構築する
	sd, err := xrt.NewStreamDictForBuf(contentBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stream dict: %w", err)
	}

	sd.Dict["Type"] = types.Name("XObject")
	sd.Dict["Subtype"] = types.Name("Form")
	sd.Dict["BBox"] = mediaBox.Array()
	if resources != nil {
		sd.Dict["Resources"] = resources
	}

	if err := sd.Encode(); err != nil {
		return nil, nil, fmt.Errorf("failed to encode form XObject: %w", err)
	}

	ref, err := xrt.IndRefForNewObject(*sd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register form XObject: %w", err)
	}

	return ref, mediaBox, nil
}

// setBookmarks は Outline ツリーを直接構築し、各しおりに Y 座標付き Dest を設定する。
// PDF の Destination 形式: [pageRef /XYZ left top zoom]
func (w *Writer) setBookmarks(ctx *pdfModel.Context, bookmarks []model.Bookmark, totalHeight float64) error {
	if len(bookmarks) == 0 {
		return nil
	}

	xrt := ctx.XRefTable

	// 出力ページ(1ページ目)の IndirectRef を取得する
	pageRef, err := xrt.PageDictIndRef(1)
	if err != nil {
		return fmt.Errorf("failed to get page reference: %w", err)
	}

	// Outline ルートを作成する
	outlineDict := types.Dict{
		"Type": types.Name("Outlines"),
	}
	outlineRef, err := xrt.IndRefForNewObject(outlineDict)
	if err != nil {
		return fmt.Errorf("failed to create outline root: %w", err)
	}

	// しおりアイテムを再帰的に構築する
	itemRefs, err := buildOutlineItems(xrt, bookmarks, *outlineRef, *pageRef, totalHeight)
	if err != nil {
		return err
	}

	if len(itemRefs) > 0 {
		// ルートに First/Last/Count を設定する
		outlineDict["First"] = itemRefs[0]
		outlineDict["Last"] = itemRefs[len(itemRefs)-1]
		outlineDict["Count"] = types.Integer(countOutlineItems(bookmarks))
	}

	// カタログに Outlines を追加する
	catalogDict, err := xrt.Catalog()
	if err != nil {
		return fmt.Errorf("failed to get catalog: %w", err)
	}
	catalogDict["Outlines"] = *outlineRef

	return nil
}

// buildOutlineItems はしおりリストから Outline アイテムの連鎖を構築する。
// 各アイテムは /Dest [pageRef /XYZ 0 pdfY null] で Y 座標付きジャンプを実現する。
func buildOutlineItems(
	xrt *pdfModel.XRefTable,
	bookmarks []model.Bookmark,
	parentRef types.IndirectRef,
	pageRef types.IndirectRef,
	totalHeight float64,
) ([]types.IndirectRef, error) {
	refs := make([]types.IndirectRef, len(bookmarks))

	for i, bm := range bookmarks {
		// モデルY（上端基準）→ PDF Y（下端基準）に変換する
		pdfY := totalHeight - bm.Y

		dest := types.Array{
			pageRef,
			types.Name("XYZ"),
			types.Float(0),
			types.Float(pdfY),
			nil, // zoom = null（現在のズーム倍率を維持）
		}

		itemDict := types.Dict{
			"Title":  types.StringLiteral(bm.Title),
			"Parent": parentRef,
			"Dest":   dest,
		}

		ref, err := xrt.IndRefForNewObject(itemDict)
		if err != nil {
			return nil, fmt.Errorf("failed to create outline item: %w", err)
		}
		refs[i] = *ref

		// 子しおりを処理する
		if len(bm.Children) > 0 {
			childRefs, err := buildOutlineItems(xrt, bm.Children, *ref, pageRef, totalHeight)
			if err != nil {
				return nil, err
			}
			if len(childRefs) > 0 {
				itemDict["First"] = childRefs[0]
				itemDict["Last"] = childRefs[len(childRefs)-1]
				itemDict["Count"] = types.Integer(countOutlineItems(bm.Children))
			}
		}
	}

	// 兄弟アイテム間を Next/Prev で連結する
	for i := range refs {
		entry, found := xrt.FindTableEntryForIndRef(&refs[i])
		if !found {
			continue
		}
		d, ok := entry.Object.(types.Dict)
		if !ok {
			continue
		}
		if i > 0 {
			d["Prev"] = refs[i-1]
		}
		if i < len(refs)-1 {
			d["Next"] = refs[i+1]
		}
	}

	return refs, nil
}

// countOutlineItems はしおりの総数を再帰的にカウントする。
func countOutlineItems(bookmarks []model.Bookmark) int {
	count := len(bookmarks)
	for _, bm := range bookmarks {
		count += countOutlineItems(bm.Children)
	}
	return count
}
