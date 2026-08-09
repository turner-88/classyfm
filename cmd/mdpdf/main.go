// Command mdpdf renders a Markdown file to a branded ClassyFM PDF (Outfit title,
// Plus Jakarta Sans headings and body, Cascadia Mono code; navy/red logo palette),
// via goldmark for HTML and headless Chrome for print-to-PDF. Fonts are vendored
// and embedded, so output is self-contained and reproducible.
//
// Usage:
//
//	mdpdf [flags] <input.md> [output.pdf]
//
// output.pdf defaults to the input path with a .pdf extension.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(0)
	title := flag.String("title", "", "document title (default: first H1, else file name)")
	subtitle := flag.String("subtitle", "Classy 103.4 FM · Padang", "cover eyebrow line")
	footer := flag.String("footer", "", "left-hand footer text (default: <title>)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdpdf [flags] <input.md> [output.pdf]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	inPath := flag.Arg(0)
	outPath := flag.Arg(1)
	if outPath == "" {
		outPath = strings.TrimSuffix(inPath, ".md") + ".pdf"
	}

	md, err := os.ReadFile(inPath)
	if err != nil {
		log.Fatalf("read %s: %v", inPath, err)
	}

	docTitle := *title
	if docTitle == "" {
		docTitle = firstH1(md)
	}
	if docTitle == "" {
		docTitle = strings.TrimSuffix(baseName(inPath), ".md")
	}
	footerText := *footer
	if footerText == "" {
		footerText = docTitle
	}

	htmlDoc, err := renderHTML(md, *subtitle)
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	pdf, err := renderPDF(htmlDoc, footerText)
	if err != nil {
		log.Fatalf("pdf: %v", err)
	}
	if err := os.WriteFile(outPath, pdf, 0o644); err != nil {
		log.Fatalf("write %s: %v", outPath, err)
	}
	log.Printf("wrote %s (%.0f KB)", outPath, float64(len(pdf))/1024)
}

// firstH1 returns the text of the first ATX "# " heading, or "" if none.
func firstH1(md []byte) string {
	sc := bufio.NewScanner(strings.NewReader(string(md)))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}
