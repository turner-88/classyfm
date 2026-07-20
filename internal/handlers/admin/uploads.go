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
	maxUploadBytes           = 5 << 20 // 5 MiB
	uploadSubdirPrograms     = "programs"
	uploadSubdirBroadcasters = "broadcasters"
	uploadSubdirHotRelease   = "hot-release"
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

	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("create upload file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, io.LimitReader(file, maxUploadBytes)); err != nil {
		return "", fmt.Errorf("write upload file: %w", err)
	}

	return "/uploads/" + subdir + "/" + name, nil
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
