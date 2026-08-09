package main

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// A4 in inches, and margins expressed in inches (~16mm sides/top, ~18mm bottom).
const (
	a4WidthIn    = 8.27
	a4HeightIn   = 11.69
	marginTopIn  = 0.63
	marginBotIn  = 0.71
	marginSideIn = 0.63
)

// renderPDF drives headless Chrome to print the HTML document to a PDF byte slice.
// footer is the left-hand footer label; page numbers are added on the right.
func renderPDF(htmlDoc, footer string) ([]byte, error) {
	// Chrome loads the page from a temp file (a data: URL would exceed the URL
	// length limit once the fonts are inlined).
	tmp, err := os.CreateTemp("", "mdpdf-*.html")
	if err != nil {
		return nil, fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(htmlDoc); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("font-render-hinting", "none"))...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	footerTemplate := fmt.Sprintf(
		`<div style="width:100%%;font-family:ui-monospace,'Cascadia Mono','Courier New',monospace;font-size:7.5pt;color:#a1a6b4;padding:0 14mm;display:flex;justify-content:space-between;">`+
			`<span>%s</span>`+
			`<span>Page <span class="pageNumber"></span> / <span class="totalPages"></span></span>`+
			`</div>`, html.EscapeString(footer))

	var buf []byte
	fileURL := "file://" + filepath.ToSlash(tmp.Name())
	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		// Wait for the embedded webfonts to finish loading before printing, so no
		// glyphs fall back to a system font.
		chromedp.ActionFunc(func(ctx context.Context) error {
			var status string
			return chromedp.Evaluate(`document.fonts.ready.then(() => document.fonts.status)`, &status,
				func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
					return p.WithAwaitPromise(true)
				}).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			b, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(a4WidthIn).
				WithPaperHeight(a4HeightIn).
				WithMarginTop(marginTopIn).
				WithMarginBottom(marginBotIn).
				WithMarginLeft(marginSideIn).
				WithMarginRight(marginSideIn).
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate("<span></span>").
				WithFooterTemplate(footerTemplate).
				Do(ctx)
			if err != nil {
				return err
			}
			buf = b
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("print to pdf: %w", err)
	}
	return buf, nil
}
