package admin

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/classyfm/classyfm/internal/imgsave"
)

const (
	// maxUploadBytes is a safety net, not the everyday path: the admin forms run
	// web/static/js/admin-image-upload.js, which downscales and re-encodes to
	// WebP in the browser, so a real upload lands in the low hundreds of KB. The
	// ceiling only ever binds on animated GIFs (deliberately never re-encoded)
	// and on posts made with JavaScript disabled.
	maxUploadBytes = 12 << 20 // 12 MiB

	// maxRequestBytes bounds the whole multipart body. Most forms carry a single
	// image plus the text fields around it, but the Hot Release editor can post a
	// whole mid-article gallery at once (each browser-compressed to a few hundred
	// KB, but the JS-disabled fallback posts originals), so the ceiling is sized
	// for several full-size images. deploy/classyfm.co.id must keep
	// client_max_body_size above this, or nginx rejects the request first.
	maxRequestBytes = 8*maxUploadBytes + 1<<20

	uploadSubdirPrograms     = "programs"
	uploadSubdirBroadcasters = "broadcasters"
	uploadSubdirHotRelease   = "hot-release"
	uploadSubdirAbout        = "about"
	uploadSubdirAds          = "ads"
	uploadSubdirHero         = "hero"
	uploadSubdirSeo          = "seo"
	uploadSubdirEvent        = "event"
)

// saveUploadedImage reads the multipart field named `field` from the request (if
// present) and, on success, writes it under h.uploadDir/subdir/<random-name>.<ext>,
// returning the public URL path to store (e.g. "/uploads/programs/xxxx.jpg").
//
// If the field was left empty (no file chosen), it returns ("", nil) so the caller
// can keep whatever image the record already has. Files over maxUploadBytes or
// whose sniffed content isn't a recognized image type are rejected with an error.
//
// Known limitation: replacing/deleting a record's image does not delete the old
// file from disk - acceptable for v1, a cleanup pass can be added later.
func (h *Handler) saveUploadedImage(r *http.Request, field, subdir string) (string, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()
	return h.writeUploadedImage(file, subdir)
}

// saveUploadedImages is the multi-file counterpart of saveUploadedImage: it saves
// every file submitted under `field` (in the order the browser posted them) and
// returns their public URLs. Returns (nil, nil) when the field carried no files.
// The request must already be parsed (parseUploadForm) so r.MultipartForm is set.
//
// Order matters to the caller (it drives the gallery/slideshow sequence), so the
// URLs come back in r.MultipartForm.File[field] order and are never sorted.
func (h *Handler) saveUploadedImages(r *http.Request, field, subdir string) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	headers := r.MultipartForm.File[field]
	if len(headers) == 0 {
		return nil, nil
	}
	urls := make([]string, 0, len(headers))
	for _, fh := range headers {
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open upload: %w", err)
		}
		url, err := h.writeUploadedImage(file, subdir)
		file.Close()
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// writeUploadedImage sniffs, names, and writes an already-opened upload under
// h.uploadDir/subdir/<random-name>.<ext>, returning the public URL path to store.
// Shared by the single- and multi-file save paths above.
func (h *Handler) writeUploadedImage(file multipart.File, subdir string) (string, error) {
	ext, err := sniffImageExt(file)
	if err != nil {
		return "", err
	}

	name, err := imgsave.RandomName(ext)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(h.uploadDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	path := filepath.Join(dir, name)
	dst, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create upload file: %w", err)
	}
	defer dst.Close()

	// Read one byte past the limit so an oversize file is detected rather than
	// silently truncated into a corrupt image, and drop the partial file on any
	// failure so nothing unreferenced is left behind.
	written, err := io.Copy(dst, io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		os.Remove(path)
		return "", fmt.Errorf("write upload file: %w", err)
	}
	if written > maxUploadBytes {
		os.Remove(path)
		return "", errUploadTooLarge
	}

	return "/uploads/" + subdir + "/" + name, nil
}

// errUploadTooLarge is what both size checks report. The message is Indonesian
// to match imgsave.ExtForHeader, which surfaces in the same form error banner.
var errUploadTooLarge = fmt.Errorf("berkas terlalu besar; maksimal %d MB", maxUploadBytes>>20)

// parseUploadForm caps the request body and parses the multipart form for the
// handlers that accept an image.
//
// It exists because every one of those handlers used to discard the parse error
// (`_ = r.ParseMultipartForm(...)`). Past the ceiling that left every FormValue
// empty, so an oversize upload surfaced as "Title and slug are required" - the
// admin had no way to tell that the image was the problem. Callers should render
// their form back with this error's message.
func parseUploadForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	if err := r.ParseMultipartForm(maxRequestBytes); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return errUploadTooLarge
		}
		return fmt.Errorf("gagal membaca formulir: %w", err)
	}
	return nil
}

// sniffImageExt reads the first 512 bytes of f to determine its real content type
// (never trusting the client-supplied filename), rewinds f so the caller can still
// read the full contents, and returns the extension to save it under.
func sniffImageExt(f multipart.File) (string, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read upload: %w", err)
	}
	ext, err := imgsave.ExtForHeader(head[:n])
	if err != nil {
		return "", err
	}
	if seeker, ok := f.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("seek upload: %w", err)
		}
	}
	return ext, nil
}
