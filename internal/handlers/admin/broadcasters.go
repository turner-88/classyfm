package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type broadcastersListData struct {
	Base         baseData
	Broadcasters []sqlc.Broadcaster
	Pagination   pagination
}

// BroadcastersList renders every broadcaster (active and inactive).
func (h *Handler) BroadcastersList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "sort_order", "asc", "name", "slug", "role", "sort_order")
	total, err := h.q.CountBroadcasters(r.Context(), sqlc.CountBroadcastersParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load broadcasters", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/broadcasters", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	broadcasters, err := h.q.ListBroadcasters(r.Context(), sqlc.ListBroadcastersParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load broadcasters", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/broadcasters_list", broadcastersListData{
		Base:         h.base(r, "Broadcasters", "broadcasters"),
		Broadcasters: broadcasters,
		Pagination:   pg,
	})
}

type broadcasterDetailData struct {
	Base        baseData
	Broadcaster sqlc.Broadcaster
}

// BroadcasterDetail renders a read-only view of a single broadcaster.
func (h *Handler) BroadcasterDetail(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	broadcaster, err := h.q.GetBroadcaster(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/broadcasters_detail", broadcasterDetailData{
		Base:        h.base(r, broadcaster.Name, "broadcasters"),
		Broadcaster: broadcaster,
	})
}

type broadcasterFormData struct {
	Base        baseData
	IsNew       bool
	Broadcaster sqlc.Broadcaster
	Error       string
}

// BroadcasterNew renders the create-broadcaster form.
func (h *Handler) BroadcasterNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/broadcasters_form", broadcasterFormData{
		Base:        h.base(r, "New Broadcaster", "broadcasters"),
		IsNew:       true,
		Broadcaster: sqlc.Broadcaster{IsActive: true},
	})
}

// BroadcasterCreate inserts a new broadcaster.
func (h *Handler) BroadcasterCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	c, sortOrder, isActive, uploadErr := h.broadcasterFromForm(w, r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/broadcasters_form", broadcasterFormData{
			Base:        h.base(r, "New Broadcaster", "broadcasters"),
			IsNew:       true,
			Broadcaster: c,
			Error:       msg,
		})
	}

	if uploadErr != nil {
		renderErr("Failed to upload photo: " + uploadErr.Error())
		return
	}

	if c.Name == "" || c.Slug == "" {
		renderErr("Name and slug are required.")
		return
	}

	res, err := h.q.CreateBroadcaster(r.Context(), sqlc.CreateBroadcasterParams{
		Name:       c.Name,
		Slug:       c.Slug,
		Role:       c.Role,
		PhotoUrl:   c.PhotoUrl,
		Bio:        c.Bio,
		BirthPlace: c.BirthPlace,
		BirthDate:  c.BirthDate,
		Instagram:  c.Instagram,
		Twitter:    c.Twitter,
		Facebook:   c.Facebook,
		SortOrder:  sortOrder,
		IsActive:   isActive,
	})
	if err != nil {
		renderErr(friendlyDBError(err, "Slug is already used by another broadcaster."))
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "broadcaster", &uid, "Created broadcaster "+c.Name)
	http.Redirect(w, r, "/admin/broadcasters/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// BroadcasterEdit renders the edit form for an existing broadcaster.
func (h *Handler) BroadcasterEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	broadcaster, err := h.q.GetBroadcaster(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/broadcasters_form", broadcasterFormData{
		Base:        h.base(r, "Edit Broadcaster", "broadcasters"),
		IsNew:       false,
		Broadcaster: broadcaster,
	})
}

// BroadcasterUpdate saves edits to an existing broadcaster.
func (h *Handler) BroadcasterUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	c, sortOrder, isActive, uploadErr := h.broadcasterFromForm(w, r)
	c.ID = id

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/broadcasters_form", broadcasterFormData{
			Base:        h.base(r, "Edit Broadcaster", "broadcasters"),
			IsNew:       false,
			Broadcaster: c,
			Error:       msg,
		})
	}

	if uploadErr != nil {
		renderErr("Failed to upload photo: " + uploadErr.Error())
		return
	}

	if c.Name == "" || c.Slug == "" {
		renderErr("Name and slug are required.")
		return
	}

	err := h.q.UpdateBroadcaster(r.Context(), sqlc.UpdateBroadcasterParams{
		Name:       c.Name,
		Slug:       c.Slug,
		Role:       c.Role,
		PhotoUrl:   c.PhotoUrl,
		Bio:        c.Bio,
		BirthPlace: c.BirthPlace,
		BirthDate:  c.BirthDate,
		Instagram:  c.Instagram,
		Twitter:    c.Twitter,
		Facebook:   c.Facebook,
		SortOrder:  sortOrder,
		IsActive:   isActive,
		ID:         id,
	})
	if err != nil {
		renderErr(friendlyDBError(err, "Slug is already used by another broadcaster."))
		return
	}
	h.audit(r, "update", "broadcaster", &id, "Updated broadcaster "+c.Name)
	http.Redirect(w, r, "/admin/broadcasters/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// BroadcasterDelete removes a broadcaster.
func (h *Handler) BroadcasterDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteBroadcaster(r.Context(), id); err != nil {
		http.Error(w, "failed to delete broadcaster", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "broadcaster", &id, "Deleted broadcaster")
	http.Redirect(w, r, "/admin/broadcasters", http.StatusSeeOther)
}

// broadcasterFromForm reads broadcaster fields common to create/update from the
// request body (a multipart form, since the photo is a real file upload). The
// photo_url column defaults to whatever the hidden current_image_url field carries
// forward (i.e. "no change"), and is overwritten if a new file was submitted.
// uploadErr is non-nil if a file was submitted but rejected (wrong type or over the
// size limit) - callers should surface it and not save.
func (h *Handler) broadcasterFromForm(w http.ResponseWriter, r *http.Request) (c sqlc.Broadcaster, sortOrder int32, isActive bool, uploadErr error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	_ = r.ParseMultipartForm(maxUploadBytes + 1<<20)
	c.Name = strings.TrimSpace(r.FormValue("name"))
	c.Slug = strings.TrimSpace(r.FormValue("slug"))
	c.Role = toNullString(r.FormValue("role"))
	c.Bio = toNullString(r.FormValue("bio"))
	c.BirthPlace = toNullString(r.FormValue("birth_place"))
	c.BirthDate = toNullString(r.FormValue("birth_date"))
	c.Instagram = toNullString(r.FormValue("instagram"))
	c.Twitter = toNullString(r.FormValue("twitter"))
	c.Facebook = toNullString(r.FormValue("facebook"))
	c.PhotoUrl = toNullString(r.FormValue("current_image_url"))
	if url, err := h.saveUploadedImage(r, "photo", uploadSubdirBroadcasters); err != nil {
		uploadErr = err
	} else if url != "" {
		c.PhotoUrl = toNullString(url)
	}
	if n, err := strconv.Atoi(r.FormValue("sort_order")); err == nil {
		sortOrder = int32(n)
	}
	isActive = r.FormValue("is_active") == "on"
	return c, sortOrder, isActive, uploadErr
}
