package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
)

// maxMiddleImages caps the mid-article gallery. Generous for an editorial photo
// set but bounded so a single post can't balloon the row or the request body.
const maxMiddleImages = 12

const publishedAtLayout = "2006-01-02T15:04"

type hotReleaseListData struct {
	Base       baseData
	Items      []sqlc.NewsItem
	Pagination pagination
}

// HotReleaseList renders every Hot Release item (published and unpublished).
func (h *Handler) HotReleaseList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "published_at", "desc", "title", "published_at")
	total, err := h.q.CountAllHotRelease(r.Context(), pattern)
	if err != nil {
		http.Error(w, "failed to load Hot Release", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/hot-release", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	items, err := h.q.ListAllHotRelease(r.Context(), sqlc.ListAllHotReleaseParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load Hot Release", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/news_list", hotReleaseListData{
		Base:       h.base(r, "Hot Release", "hot-release"),
		Items:      items,
		Pagination: pg,
	})
}

// hotReleaseForm is the view-model shared by the create and edit forms.
type hotReleaseForm struct {
	Base            baseData
	IsNew           bool
	Item            sqlc.NewsItem
	MiddleImages    []string // decoded from Item.MiddleImages, in display order
	PublishedAtForm string   // formatted for <input type="datetime-local">
	Error           string
}

func newHotReleaseForm(r *http.Request, item sqlc.NewsItem) hotReleaseForm {
	return hotReleaseForm{
		Item:            item,
		MiddleImages:    decodeImageList(item.MiddleImages),
		PublishedAtForm: item.PublishedAt.Format(publishedAtLayout),
	}
}

// decodeImageList unpacks the JSON array stored in news_items.middle_images.
// A NULL/blank column or malformed JSON yields nil - a broken value must not
// take the form or the article page down.
func decodeImageList(s sql.NullString) []string {
	if !s.Valid || strings.TrimSpace(s.String) == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(s.String), &urls); err != nil {
		return nil
	}
	return urls
}

// encodeImageList packs an ordered URL list back into the JSON column, using an
// invalid (NULL) value for the empty list so an article with no gallery stays NULL.
func encodeImageList(urls []string) sql.NullString {
	if len(urls) == 0 {
		return sql.NullString{}
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// HotReleaseNew renders the create form.
func (h *Handler) HotReleaseNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	form := newHotReleaseForm(r, sqlc.NewsItem{IsPublished: true, PublishedAt: time.Now()})
	form.Base = h.base(r, "New Hot Release", "hot-release")
	form.IsNew = true
	h.r.Page(w, http.StatusOK, "admin/news_form", form)
}

// HotReleaseCreate inserts a new Hot Release item.
func (h *Handler) HotReleaseCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	item, publishedAt, formErr, uploadErr := h.hotReleaseFromForm(w, r)

	renderErr := func(status int, msg string) {
		form := hotReleaseForm{
			Base:            h.base(r, "New Hot Release", "hot-release"),
			IsNew:           true,
			Item:            item,
			PublishedAtForm: r.FormValue("published_at"),
			Error:           msg,
		}
		h.r.Page(w, status, "admin/news_form", form)
	}

	if uploadErr != nil {
		renderErr(http.StatusBadRequest, "Failed to upload image: "+uploadErr.Error())
		return
	}

	if formErr != "" {
		renderErr(http.StatusBadRequest, formErr)
		return
	}

	res, err := h.q.CreateHotRelease(r.Context(), sqlc.CreateHotReleaseParams{
		Title:        item.Title,
		Slug:         item.Slug,
		Excerpt:      item.Excerpt,
		Content:      item.Content,
		ImageUrl:     item.ImageUrl,
		MiddleImages: item.MiddleImages,
		PublishedAt:  publishedAt,
		IsPublished:  item.IsPublished,
		IsFeatured:   item.IsFeatured,
	})
	if err != nil {
		renderErr(http.StatusBadRequest, friendlyDBError(err, "Slug is already used by another news item."))
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "hot_release", &uid, "Created Hot Release "+item.Title)
	h.flash(w, "Hot Release created.")
	http.Redirect(w, r, "/admin/hot-release/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// HotReleaseEdit renders the edit form for an existing item.
func (h *Handler) HotReleaseEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	item, err := h.q.GetNewsItem(r.Context(), id)
	if err != nil || item.Source != sqlc.NewsItemsSourceHotRelease {
		http.NotFound(w, r)
		return
	}
	form := newHotReleaseForm(r, item)
	form.Base = h.base(r, "Edit Hot Release", "hot-release")
	h.r.Page(w, http.StatusOK, "admin/news_form", form)
}

// HotReleaseUpdate saves edits to an existing Hot Release item.
func (h *Handler) HotReleaseUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	item, publishedAt, formErr, uploadErr := h.hotReleaseFromForm(w, r)
	item.ID = id

	renderErr := func(status int, msg string) {
		form := hotReleaseForm{
			Base:            h.base(r, "Edit Hot Release", "hot-release"),
			IsNew:           false,
			Item:            item,
			PublishedAtForm: r.FormValue("published_at"),
			Error:           msg,
		}
		h.r.Page(w, status, "admin/news_form", form)
	}

	if uploadErr != nil {
		renderErr(http.StatusBadRequest, "Failed to upload image: "+uploadErr.Error())
		return
	}

	if formErr != "" {
		renderErr(http.StatusBadRequest, formErr)
		return
	}

	err := h.q.UpdateHotRelease(r.Context(), sqlc.UpdateHotReleaseParams{
		Title:        item.Title,
		Slug:         item.Slug,
		Excerpt:      item.Excerpt,
		Content:      item.Content,
		ImageUrl:     item.ImageUrl,
		MiddleImages: item.MiddleImages,
		PublishedAt:  publishedAt,
		IsPublished:  item.IsPublished,
		IsFeatured:   item.IsFeatured,
		ID:           id,
	})
	if err != nil {
		renderErr(http.StatusBadRequest, friendlyDBError(err, "Slug is already used by another news item."))
		return
	}
	h.audit(r, "update", "hot_release", &id, "Updated Hot Release "+item.Title)
	h.flash(w, "Hot Release saved.")
	http.Redirect(w, r, "/admin/hot-release/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// HotReleaseToggleFeature flips is_featured on one Hot Release item.
func (h *Handler) HotReleaseToggleFeature(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	item, err := h.q.GetNewsItem(r.Context(), id)
	if err != nil || item.Source != sqlc.NewsItemsSourceHotRelease {
		http.NotFound(w, r)
		return
	}
	if err := h.q.SetNewsItemFeatured(r.Context(), sqlc.SetNewsItemFeaturedParams{IsFeatured: !item.IsFeatured, ID: id}); err != nil {
		http.Error(w, "failed to save changes", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "hot_release", &id, "Updated Hot Release featured status")
	http.Redirect(w, r, "/admin/hot-release", http.StatusSeeOther)
}

// HotReleaseDelete removes a Hot Release item.
func (h *Handler) HotReleaseDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteNewsItem(r.Context(), id); err != nil {
		http.Error(w, "failed to delete Hot Release", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "hot_release", &id, "Deleted Hot Release")
	h.flash(w, "Hot Release deleted.")
	http.Redirect(w, r, "/admin/hot-release", http.StatusSeeOther)
}

// hotReleaseFromForm reads and validates Hot Release fields from the request
// body (a multipart form, since the image is a real file upload). The
// image_url column defaults to whatever the hidden current_image_url field
// carries forward (i.e. "no change"), and is overwritten if a new file was
// submitted. uploadErr is non-nil if a file was submitted but rejected (wrong
// type or over the size limit) - callers should surface it and not save.
// The returned publishedAt is only meaningful when formErr == "" && uploadErr == nil.
func (h *Handler) hotReleaseFromForm(w http.ResponseWriter, r *http.Request) (item sqlc.NewsItem, publishedAt time.Time, formErr string, uploadErr error) {
	if err := parseUploadForm(w, r); err != nil {
		return item, time.Time{}, "", err
	}
	item.Source = sqlc.NewsItemsSourceHotRelease
	item.Title = strings.TrimSpace(r.FormValue("title"))
	item.Slug = toNullString(r.FormValue("slug"))
	item.Excerpt = toNullString(r.FormValue("excerpt"))
	item.Content = toNullString(sanitize.PlainText(r.FormValue("content")))
	item.ImageUrl = toNullString(r.FormValue("current_image_url"))
	if url, err := h.saveUploadedImage(r, "image", uploadSubdirHotRelease); err != nil {
		uploadErr = err
	} else if url != "" {
		item.ImageUrl = toNullString(url)
	}
	if imgs, err := h.middleImagesFromForm(r); err != nil {
		uploadErr = err
	} else {
		item.MiddleImages = encodeImageList(imgs)
	}
	item.IsPublished = r.FormValue("is_published") == "on"
	item.IsFeatured = r.FormValue("is_featured") == "on"

	if item.Title == "" || !item.Slug.Valid {
		return item, time.Time{}, "Title and slug are required.", uploadErr
	}

	parsed, err := time.ParseInLocation(publishedAtLayout, r.FormValue("published_at"), time.Local)
	if err != nil {
		return item, time.Time{}, "Invalid publish date.", uploadErr
	}
	item.PublishedAt = parsed
	return item, parsed, "", uploadErr
}

// middleImagesFromForm assembles the ordered mid-article gallery for a Hot
// Release submission: the retained existing images first (in the DOM/field order
// the admin arranged them, minus any marked for removal), then newly uploaded
// files appended at the end. The order is load-bearing - it is the slideshow
// order - so the list is never sorted. The count is capped at maxMiddleImages.
func (h *Handler) middleImagesFromForm(r *http.Request) ([]string, error) {
	removed := make(map[string]bool)
	for _, u := range r.Form["remove_middle_image"] {
		removed[u] = true
	}

	var urls []string
	for _, u := range r.Form["existing_middle_image"] {
		// Only keep values that look like one of our own uploads: the hidden
		// fields are admin-editable, and a stray URL would otherwise be trusted
		// straight onto the public page.
		if removed[u] || !strings.HasPrefix(u, "/uploads/") {
			continue
		}
		urls = append(urls, u)
	}

	uploaded, err := h.saveUploadedImages(r, "middle_image", uploadSubdirHotRelease)
	if err != nil {
		return nil, err
	}
	urls = append(urls, uploaded...)

	if len(urls) > maxMiddleImages {
		urls = urls[:maxMiddleImages]
	}
	return urls, nil
}
