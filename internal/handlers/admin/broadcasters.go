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
	Programs    []programChoice
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
	// Reuses the form's picker rather than ListProgramsForBroadcaster, which
	// filters to active programs: an admin can tick an inactive program, and the
	// read-only view should show what is actually linked.
	linked, _ := h.q.GetBroadcasterProgramIDs(r.Context(), id)
	var programs []programChoice
	for _, p := range h.programChoices(r, linked) {
		if p.Checked {
			programs = append(programs, p)
		}
	}
	h.r.Page(w, http.StatusOK, "admin/broadcasters_detail", broadcasterDetailData{
		Base:        h.base(r, broadcaster.Name, "broadcasters"),
		Broadcaster: broadcaster,
		Programs:    programs,
	})
}

// programChoice is one checkbox in the form's program picker. Inactive programs
// are still listed - one may already be linked, and hiding it would silently
// drop the link on the next save.
type programChoice struct {
	ID       uint64
	Title    string
	IsActive bool
	Checked  bool
}

type broadcasterFormData struct {
	Base        baseData
	IsNew       bool
	Broadcaster sqlc.Broadcaster
	Programs    []programChoice
	Error       string
}

// programChoices lists every program, ticking the ones in selected.
func (h *Handler) programChoices(r *http.Request, selected []uint64) []programChoice {
	programs, err := h.q.ListAllPrograms(r.Context())
	if err != nil {
		return nil
	}
	checked := make(map[uint64]bool, len(selected))
	for _, id := range selected {
		checked[id] = true
	}
	choices := make([]programChoice, 0, len(programs))
	for _, p := range programs {
		choices = append(choices, programChoice{
			ID: p.ID, Title: p.Title, IsActive: p.IsActive, Checked: checked[p.ID],
		})
	}
	return choices
}

// saveBroadcasterPrograms replaces a broadcaster's program links with ids.
//
// Not transactional, for the same reason as saveAdBannerPages: neither handler
// package holds a *sql.DB. A partial write leaves the broadcaster linked to
// fewer programs rather than corrupting anything, and the caller surfaces the
// error so the admin can re-save.
func (h *Handler) saveBroadcasterPrograms(r *http.Request, broadcasterID uint64, ids []uint64) error {
	if err := h.q.DeleteBroadcasterPrograms(r.Context(), broadcasterID); err != nil {
		return err
	}
	for _, id := range ids {
		err := h.q.CreateBroadcasterProgram(r.Context(), sqlc.CreateBroadcasterProgramParams{
			BroadcasterID: broadcasterID,
			ProgramID:     id,
		})
		if err != nil {
			return err
		}
	}
	return nil
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
		Programs:    h.programChoices(r, nil),
	})
}

// BroadcasterCreate inserts a new broadcaster.
func (h *Handler) BroadcasterCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	c, sortOrder, isActive, programIDs, uploadErr := h.broadcasterFromForm(w, r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/broadcasters_form", broadcasterFormData{
			Base:        h.base(r, "New Broadcaster", "broadcasters"),
			IsNew:       true,
			Broadcaster: c,
			// Re-tick what was submitted, so a validation error doesn't quietly
			// clear the picker.
			Programs: h.programChoices(r, programIDs),
			Error:    msg,
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
	if err := h.saveBroadcasterPrograms(r, uid, programIDs); err != nil {
		renderErr("Broadcaster saved, but its programs could not be linked. Try saving again.")
		return
	}
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
	linked, _ := h.q.GetBroadcasterProgramIDs(r.Context(), id)
	h.r.Page(w, http.StatusOK, "admin/broadcasters_form", broadcasterFormData{
		Base:        h.base(r, "Edit Broadcaster", "broadcasters"),
		IsNew:       false,
		Broadcaster: broadcaster,
		Programs:    h.programChoices(r, linked),
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
	c, sortOrder, isActive, programIDs, uploadErr := h.broadcasterFromForm(w, r)
	c.ID = id

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/broadcasters_form", broadcasterFormData{
			Base:        h.base(r, "Edit Broadcaster", "broadcasters"),
			IsNew:       false,
			Broadcaster: c,
			Programs:    h.programChoices(r, programIDs),
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
	if err := h.saveBroadcasterPrograms(r, id, programIDs); err != nil {
		renderErr("Broadcaster saved, but its programs could not be linked. Try saving again.")
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
// size limit) - callers should surface it and not save. programIDs carries the
// program picker's selection, saved separately since it lives in its own table.
func (h *Handler) broadcasterFromForm(w http.ResponseWriter, r *http.Request) (c sqlc.Broadcaster, sortOrder int32, isActive bool, programIDs []uint64, uploadErr error) {
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
	// ParseMultipartForm above has already populated r.Form, so the repeated
	// checkbox values are readable here. Unparseable ids are skipped rather than
	// failing the save - they can only come from a tampered form.
	for _, v := range r.Form["program_ids"] {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			programIDs = append(programIDs, id)
		}
	}
	return c, sortOrder, isActive, programIDs, uploadErr
}
