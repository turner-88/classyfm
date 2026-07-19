package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type classiersListData struct {
	Base      baseData
	Classiers []sqlc.Classier
}

// ClassiersList renders every classier (active and inactive).
func (h *Handler) ClassiersList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	classiers, err := h.q.ListClassiers(r.Context())
	if err != nil {
		http.Error(w, "gagal memuat classiers", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/classiers_list", classiersListData{
		Base:      h.base(r, "Classiers", "classiers"),
		Classiers: classiers,
	})
}

type classierDetailData struct {
	Base     baseData
	Classier sqlc.Classier
}

// ClassierDetail renders a read-only view of a single classier.
func (h *Handler) ClassierDetail(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	classier, err := h.q.GetClassier(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/classiers_detail", classierDetailData{
		Base:     h.base(r, classier.Name, "classiers"),
		Classier: classier,
	})
}

type classierFormData struct {
	Base     baseData
	IsNew    bool
	Classier sqlc.Classier
	Error    string
}

// ClassierNew renders the create-classier form.
func (h *Handler) ClassierNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/classiers_form", classierFormData{
		Base:     h.base(r, "Classier Baru", "classiers"),
		IsNew:    true,
		Classier: sqlc.Classier{IsActive: true},
	})
}

// ClassierCreate inserts a new classier.
func (h *Handler) ClassierCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	c, sortOrder, isActive, uploadErr := h.classierFromForm(w, r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/classiers_form", classierFormData{
			Base:     h.base(r, "Classier Baru", "classiers"),
			IsNew:    true,
			Classier: c,
			Error:    msg,
		})
	}

	if uploadErr != nil {
		renderErr("Gagal mengunggah foto: " + uploadErr.Error())
		return
	}

	if c.Name == "" || c.Slug == "" {
		renderErr("Nama dan slug wajib diisi.")
		return
	}

	res, err := h.q.CreateClassier(r.Context(), sqlc.CreateClassierParams{
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
		renderErr(friendlyDBError(err, "Slug sudah digunakan classier lain."))
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "classier", &uid, "Membuat classier "+c.Name)
	http.Redirect(w, r, "/admin/classiers/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// ClassierEdit renders the edit form for an existing classier.
func (h *Handler) ClassierEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	classier, err := h.q.GetClassier(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/classiers_form", classierFormData{
		Base:     h.base(r, "Ubah Classier", "classiers"),
		IsNew:    false,
		Classier: classier,
	})
}

// ClassierUpdate saves edits to an existing classier.
func (h *Handler) ClassierUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	c, sortOrder, isActive, uploadErr := h.classierFromForm(w, r)
	c.ID = id

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/classiers_form", classierFormData{
			Base:     h.base(r, "Ubah Classier", "classiers"),
			IsNew:    false,
			Classier: c,
			Error:    msg,
		})
	}

	if uploadErr != nil {
		renderErr("Gagal mengunggah foto: " + uploadErr.Error())
		return
	}

	if c.Name == "" || c.Slug == "" {
		renderErr("Nama dan slug wajib diisi.")
		return
	}

	err := h.q.UpdateClassier(r.Context(), sqlc.UpdateClassierParams{
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
		renderErr(friendlyDBError(err, "Slug sudah digunakan classier lain."))
		return
	}
	h.audit(r, "update", "classier", &id, "Mengubah classier "+c.Name)
	http.Redirect(w, r, "/admin/classiers/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// ClassierDelete removes a classier.
func (h *Handler) ClassierDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteClassier(r.Context(), id); err != nil {
		http.Error(w, "gagal menghapus classier", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "classier", &id, "Menghapus classier")
	http.Redirect(w, r, "/admin/classiers", http.StatusSeeOther)
}

// classierFromForm reads classier fields common to create/update from the request
// body (a multipart form, since the photo is a real file upload). The photo_url
// column defaults to whatever the hidden current_image_url field carries forward
// (i.e. "no change"), and is overwritten if a new file was submitted. uploadErr is
// non-nil if a file was submitted but rejected (wrong type or over the size limit) -
// callers should surface it and not save.
func (h *Handler) classierFromForm(w http.ResponseWriter, r *http.Request) (c sqlc.Classier, sortOrder int32, isActive bool, uploadErr error) {
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
	if url, err := h.saveUploadedImage(r, "photo", uploadSubdirClassiers); err != nil {
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
