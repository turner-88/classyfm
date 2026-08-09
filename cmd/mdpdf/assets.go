package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"
)

//go:embed assets/template.html.tmpl
var templateFS embed.FS

//go:embed assets/fonts/*.woff2
var fontFS embed.FS

// fontFace describes one vendored woff2 to emit as an @font-face rule. The three
// families are wired to the CSS in the template: Outfit (headings), Plus Jakarta
// Sans (body), Cascadia Mono (monospace, a variable font so it spans a weight range).
type fontFace struct {
	family string
	weight string // CSS font-weight ("400" or a variable range "200 700")
	style  string
	file   string
}

var vendoredFonts = []fontFace{
	{"Outfit", "400", "normal", "Outfit-400.woff2"},
	{"Outfit", "600", "normal", "Outfit-600.woff2"},
	{"Outfit", "700", "normal", "Outfit-700.woff2"},
	{"Plus Jakarta Sans", "400", "normal", "PlusJakartaSans-400.woff2"},
	{"Plus Jakarta Sans", "500", "normal", "PlusJakartaSans-500.woff2"},
	{"Plus Jakarta Sans", "700", "normal", "PlusJakartaSans-700.woff2"},
	{"Cascadia Mono", "200 700", "normal", "CascadiaMono.woff2"},
}

// buildFontFaceCSS base64-embeds each vendored woff2 into a self-contained
// @font-face block so the rendered PDF carries its own fonts with no network or
// system-font dependency.
func buildFontFaceCSS() (string, error) {
	var b strings.Builder
	for _, f := range vendoredFonts {
		data, err := fontFS.ReadFile("assets/fonts/" + f.file)
		if err != nil {
			return "", fmt.Errorf("read embedded font %s: %w", f.file, err)
		}
		enc := base64.StdEncoding.EncodeToString(data)
		fmt.Fprintf(&b, "@font-face{font-family:'%s';font-style:%s;font-weight:%s;font-display:swap;"+
			"src:url(data:font/woff2;base64,%s) format('woff2');}\n",
			f.family, f.style, f.weight, enc)
	}
	return b.String(), nil
}
