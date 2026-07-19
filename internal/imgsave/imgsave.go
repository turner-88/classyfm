// Package imgsave holds the small pieces of image-saving logic shared between
// the admin file-upload handler and one-off tools that write images to the
// same uploads directory (e.g. cmd/importhotrelease).
package imgsave

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

// AllowedExt maps a sniffed content-type to the file extension it's saved
// under. Only these types are accepted; a client- or remote-supplied filename
// is never trusted.
var AllowedExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// ExtForHeader sniffs the real content type from header (the leading bytes of
// a file, as read by the caller) and returns the extension to save it under.
func ExtForHeader(header []byte) (string, error) {
	contentType := http.DetectContentType(header)
	ext, ok := AllowedExt[contentType]
	if !ok {
		return "", fmt.Errorf("tipe berkas tidak didukung (%s); gunakan JPG, PNG, WEBP, atau GIF", contentType)
	}
	return ext, nil
}

// RandomName generates a random filename with the given extension.
func RandomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate filename: %w", err)
	}
	return hex.EncodeToString(b) + ext, nil
}
