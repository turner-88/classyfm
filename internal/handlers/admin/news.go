package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
)

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
	PublishedAtForm string // formatted for <input type="datetime-local">
	Error           string
}

func newHotReleaseForm(r *http.Request, item sqlc.NewsItem) hotReleaseForm {
	return hotReleaseForm{Item: item, PublishedAtForm: item.PublishedAt.Format(publishedAtLayout)}
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
		Title:       item.Title,
		Slug:        item.Slug,
		Excerpt:     item.Excerpt,
		Content:     item.Content,
		ImageUrl:    item.ImageUrl,
		PublishedAt: publishedAt,
		IsPublished: item.IsPublished,
		IsFeatured:  item.IsFeatured,
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
		Title:       item.Title,
		Slug:        item.Slug,
		Excerpt:     item.Excerpt,
		Content:     item.Content,
		ImageUrl:    item.ImageUrl,
		PublishedAt: publishedAt,
		IsPublished: item.IsPublished,
		IsFeatured:  item.IsFeatured,
		ID:          id,
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
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	_ = r.ParseMultipartForm(maxUploadBytes + 1<<20)
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
