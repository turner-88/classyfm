// Package web embeds the HTML templates and static assets so the server ships as a
// single self-contained binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// Templates returns a filesystem rooted at the templates directory
// (so "layouts/base.html" resolves).
func Templates() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}

// Static returns a filesystem rooted at the static directory, for serving under /static.
func Static() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
