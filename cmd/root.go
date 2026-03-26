// Package cmd はCLI層を担当する。
// cobra によるコマンド定義、フラグ解析、設定マージ、内部パッケージの組み立てと実行を行う。
// ExitError を終了コードに変換し os.Exit を呼ぶのはこのパッケージの責務である。
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kznagamori/go_PDFStitcher/internal/apperror"
	"github.com/kznagamori/go_PDFStitcher/internal/bookmark"
	"github.com/kznagamori/go_PDFStitcher/internal/config"
	"github.com/kznagamori/go_PDFStitcher/internal/output"
	"github.com/kznagamori/go_PDFStitcher/internal/pdf"
	"github.com/kznagamori/go_PDFStitcher/internal/prompt"
	"github.com/kznagamori/go_PDFStitcher/internal/stitcher"
	"github.com/kznagamori/go_PDFStitcher/internal/unit"
	"github.com/kznagamori/go_PDFStitcher/internal/validate"
)

// フラグ変数。cobra のフラグバインディングに使用する。
var (
	outputFlag  string
	gapFlag     string
	alignFlag   string
	forceFlag   bool
	verboseFlag bool
	quietFlag   bool
)

// newRootCmd はルートコマンドを構築する。
// テストから呼び出せるよう Execute() と分離している。
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "go_PDFStitcher [flags] <input.pdf> [input2.pdf ...]",
		Short:   "PDFの全ページを縦方向に連結し、1枚の縦長PDFとして出力する",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Args:    cobra.MinimumNArgs(1),
		RunE:    runStitch,
	}

	// ランタイムエラー時にusageを表示しない（引数不足等のcobraエラーでは表示される）
	rootCmd.SilenceUsage = true

	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "出力ファイルパス")
	rootCmd.Flags().StringVar(&gapFlag, "gap", "", "ページ間余白（例: 10pt, 5mm）")
	rootCmd.Flags().StringVar(&alignFlag, "align", "", "ページ配置方針（left, center, right）")
	rootCmd.Flags().BoolVar(&forceFlag, "force", false, "既存ファイルを確認なしで上書き")
	rootCmd.Flags().BoolVar(&verboseFlag, "verbose", false, "詳細ログ出力（DEBUGレベル）")
	rootCmd.Flags().BoolVar(&quietFlag, "quiet", false, "ログ抑制（ERRORのみ出力）")

	return rootCmd
}

// Execute はルートコマンドを実行する。main.go から呼び出される唯一のエントリポイント。
// エラー発生時は適切な終了コードで os.Exit する。
func Execute() {
	rootCmd := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		// cobra が自動的にエラーメッセージを表示するため、
		// ExitError の判定と終了コード変換のみ行う
		var exitErr *apperror.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(apperror.ExitGeneral)
	}
}

// runStitch は cobra RunE のメイン処理。設定読み込みからPDF連結までの全体フローを実行する。
func runStitch(cmd *cobra.Command, args []string) error {
	// verbose/quiet 競合チェック + ログレベル設定
	if err := configureLogging(); err != nil {
		return err
	}

	// align バリデーション
	if alignFlag != "" && !isValidAlign(alignFlag) {
		return apperror.NewGeneralError(
			fmt.Sprintf("invalid align value %q: must be left, center, or right", alignFlag), nil)
	}

	// 実行ファイルディレクトリの取得
	execDir := executableDir()

	// 設定ファイル読み込み
	cfg, err := config.Load(execDir)
	if err != nil {
		return err
	}

	// CLIフラグのマージ
	cfg.Merge(config.CLIFlags{
		Output:  outputFlag,
		Gap:     gapFlag,
		Align:   alignFlag,
		Force:   forceFlag,
		Verbose: verboseFlag,
		Quiet:   quietFlag,
	})

	slog.Debug("configuration loaded",
		"align", cfg.Align,
		"gap", cfg.Gap,
		"gap_unit", cfg.GapUnit,
		"on_conflict", cfg.OnConflict)

	// ガター値の決定
	gapPt, err := resolveGap(cfg)
	if err != nil {
		return apperror.NewGeneralError(err.Error(), err)
	}

	// DI組み立て
	prompter := prompt.NewPrompter()
	reader := pdf.NewReader(prompter)
	writer := pdf.NewWriter()
	bookmarkProc := bookmark.NewProcessor()

	// シグナルハンドリング: SIGINT/SIGTERM で一時ファイルをクリーンアップする
	ctx, cancel := setupSignalHandler(writer)
	defer cancel()
	_ = ctx // コンテキストは将来の拡張用

	// D&D判定: CLIフラグが指定されていない場合
	isDragAndDrop := isDragAndDropExecution(cmd)

	// 各ファイルを処理する
	var firstErr error
	successCount := 0
	failCount := 0

	for _, inputPath := range args {
		err := processFile(inputPath, cfg, gapPt, reader, writer, bookmarkProc, prompter, isDragAndDrop)
		if err != nil {
			if errors.Is(err, output.ErrUserCancelled) {
				slog.Info("operation cancelled by user")
				return nil
			}
			failCount++
			if firstErr == nil {
				firstErr = err
			}
			slog.Error(err.Error())
			// 複数ファイル処理時はエラーでも続行する
			if len(args) == 1 {
				return err
			}
			continue
		}
		successCount++
	}

	// 複数ファイル処理の結果サマリー
	if len(args) > 1 {
		slog.Info("processing complete",
			"succeeded", successCount, "failed", failCount)
	}

	// D&D時はウィンドウを保持する
	if isDragAndDrop {
		waitForEnter(prompter)
	}

	return firstErr
}

// processFile は単一ファイルの連結処理を実行する。
func processFile(
	inputPath string,
	cfg *config.Config,
	gapPt float64,
	reader *pdf.Reader,
	writer *pdf.Writer,
	bookmarkProc *bookmark.Processor,
	prompter *prompt.Prompter,
	isDragAndDrop bool,
) error {
	// 入力バリデーション
	cleanPath, err := validate.InputFile(inputPath)
	if err != nil {
		return err
	}

	// 出力パス解決
	outputPath := output.Resolve(cleanPath, outputFlag, cfg.OutputPattern)

	// 衝突処理
	var confirmer output.Confirmer
	if !isDragAndDrop {
		confirmer = prompter
	}
	finalPath, err := output.HandleConflict(
		outputPath,
		output.ConflictPolicy(cfg.OnConflict),
		forceFlag,
		confirmer,
	)
	if err != nil {
		return err
	}

	// 連結処理
	s := stitcher.NewStitcher(reader, writer, bookmarkProc)
	opts := stitcher.StitchOptions{
		GapPt:           gapPt,
		Align:           cfg.Align,
		BookmarkPattern: cfg.BookmarkPattern,
	}

	return s.Run(cleanPath, finalPath, opts)
}

// configureLogging は verbose/quiet フラグに基づいてログレベルを設定する。
func configureLogging() error {
	if verboseFlag && quietFlag {
		return apperror.NewGeneralError(
			"--verbose and --quiet cannot be used together", nil)
	}

	var level slog.Level
	switch {
	case verboseFlag:
		level = slog.LevelDebug
	case quietFlag:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))

	return nil
}

// resolveGap はガター値を pt 単位で解決する。
// CLIフラグ指定時はそれを優先し、未指定時は設定ファイルの値を使用する。
func resolveGap(cfg *config.Config) (float64, error) {
	if gapFlag != "" {
		return unit.ParseGap(gapFlag, cfg.GapUnit)
	}

	// 設定ファイルの gap + gap_unit から pt に変換する
	gapStr := fmt.Sprintf("%g%s", cfg.Gap, cfg.GapUnit)
	return unit.ParseGap(gapStr, cfg.GapUnit)
}

// setupSignalHandler はSIGINT/SIGTERMを監視し、受信時に writer.Cleanup を呼び出す。
// 返される cancel を defer で呼ぶこと。
func setupSignalHandler(writer *pdf.Writer) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			slog.Info("signal received, cleaning up...")
			writer.Cleanup()
			os.Exit(apperror.ExitGeneral)
		case <-ctx.Done():
			return
		}
	}()

	return ctx, func() {
		signal.Stop(sigCh)
		cancel()
	}
}

// isDragAndDropExecution はD&D実行かどうかを判定する。
// CLIフラグが一つも明示的に指定されていない場合にD&Dと判定する。
func isDragAndDropExecution(cmd *cobra.Command) bool {
	flagChanged := false
	cmd.Flags().Visit(func(_ *pflag.Flag) {
		flagChanged = true
	})
	return !flagChanged
}

// waitForEnter はD&D実行時にウィンドウを保持するためEnterキー入力を待つ。
func waitForEnter(prompter *prompt.Prompter) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Press Enter to close...")
	// 入力結果は無視する（ウィンドウ保持が目的）
	_, _ = prompter.Confirm("")
}

// executableDir は実行ファイルのディレクトリパスを返す。
// 取得に失敗した場合はカレントディレクトリを返す。
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// isValidAlign は align 値が有効かどうかを判定する。
func isValidAlign(align string) bool {
	return align == "left" || align == "center" || align == "right"
}
