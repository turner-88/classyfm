package admin

import (
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
)

type programsListData struct {
	Base       baseData
	Programs   []sqlc.Program
	Pagination pagination
}

// ProgramsList renders every program (active and inactive).
func (h *Handler) ProgramsList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "sort_order", "asc", "title", "slug", "host", "sort_order")
	total, err := h.q.CountPrograms(r.Context(), sqlc.CountProgramsParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load programs", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/programs", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	programs, err := h.q.ListPrograms(r.Context(), sqlc.ListProgramsParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load programs", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/programs_list", programsListData{
		Base:       h.base(r, "Program", "programs"),
		Programs:   programs,
		Pagination: pg,
	})
}

type programDetailData struct {
	Base      baseData
	Program   sqlc.Program
	Schedules []sqlc.ProgramSchedule
}

// ProgramDetail renders a read-only view of a single program, including its full
// weekly schedule.
func (h *Handler) ProgramDetail(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	program, err := h.q.GetProgram(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	schedules, err := h.q.ListSchedulesForProgram(r.Context(), id)
	if err != nil {
		slog.Error("list schedules failed", "err", err, "program_id", id)
		http.Error(w, "failed to load schedule", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/programs_detail", programDetailData{
		Base:      h.base(r, program.Title, "programs"),
		Program:   program,
		Schedules: schedules,
	})
}

type programFormData struct {
	Base      baseData
	IsNew     bool
	Program   sqlc.Program
	Schedules []sqlc.ProgramSchedule
	Weekdays  [7]string
	Error     string
}

// ProgramNew renders the create-program form.
func (h *Handler) ProgramNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/programs_form", programFormData{
		Base:     h.base(r, "New Program", "programs"),
		IsNew:    true,
		Program:  sqlc.Program{IsActive: true},
		Weekdays: models.Weekdays(),
	})
}

// ProgramCreate inserts a new program.
func (h *Handler) ProgramCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	p, sortOrder, isActive, uploadErr := h.programFromForm(w, r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/programs_form", programFormData{
			Base:     h.base(r, "New Program", "programs"),
			IsNew:    true,
			Program:  p,
			Weekdays: models.Weekdays(),
			Error:    msg,
		})
	}

	if uploadErr != nil {
		renderErr("Failed to upload image: " + uploadErr.Error())
		return
	}

	if p.Title == "" || p.Slug == "" {
		renderErr("Title and slug are required.")
		return
	}

	res, err := h.q.CreateProgram(r.Context(), sqlc.CreateProgramParams{
		Title:       p.Title,
		Slug:        p.Slug,
		Description: p.Description,
		Host:        p.Host,
		ImageUrl:    p.ImageUrl,
		SortOrder:   sortOrder,
		IsActive:    isActive,
	})
	if err != nil {
		renderErr(friendlyDBError(err, "Slug is already used by another program."))
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "program", &uid, "Created program "+p.Title)
	http.Redirect(w, r, "/admin/programs/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// ProgramEdit renders the edit form for an existing program, including its schedule.
func (h *Handler) ProgramEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	program, err := h.q.GetProgram(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	schedules, err := h.q.ListSchedulesForProgram(r.Context(), id)
	if err != nil {
		slog.Error("list schedules failed", "err", err, "program_id", id)
		http.Error(w, "failed to load schedule", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/programs_form", programFormData{
		Base:      h.base(r, "Edit Program", "programs"),
		IsNew:     false,
		Program:   program,
		Schedules: schedules,
		Weekdays:  models.Weekdays(),
	})
}

// ProgramUpdate saves edits to an existing program.
func (h *Handler) ProgramUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, sortOrder, isActive, uploadErr := h.programFromForm(w, r)
	p.ID = id

	renderErr := func(msg string) {
		schedules, _ := h.q.ListSchedulesForProgram(r.Context(), id)
		h.r.Page(w, http.StatusBadRequest, "admin/programs_form", programFormData{
			Base:      h.base(r, "Edit Program", "programs"),
			IsNew:     false,
			Program:   p,
			Schedules: schedules,
			Weekdays:  models.Weekdays(),
			Error:     msg,
		})
	}

	if uploadErr != nil {
		renderErr("Failed to upload image: " + uploadErr.Error())
		return
	}

	if p.Title == "" || p.Slug == "" {
		renderErr("Title and slug are required.")
		return
	}

	err := h.q.UpdateProgram(r.Context(), sqlc.UpdateProgramParams{
		Title:       p.Title,
		Slug:        p.Slug,
		Description: p.Description,
		Host:        p.Host,
		ImageUrl:    p.ImageUrl,
		SortOrder:   sortOrder,
		IsActive:    isActive,
		ID:          id,
	})
	if err != nil {
		renderErr(friendlyDBError(err, "Slug is already used by another program."))
		return
	}
	h.audit(r, "update", "program", &id, "Updated program "+p.Title)
	http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// ProgramDelete removes a program (and its schedule, via ON DELETE CASCADE).
func (h *Handler) ProgramDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteProgram(r.Context(), id); err != nil {
		http.Error(w, "failed to delete program", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "program", &id, "Deleted program")
	http.Redirect(w, r, "/admin/programs", http.StatusSeeOther)
}

// ScheduleCreate adds one weekly schedule slot to a program.
func (h *Handler) ScheduleCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	day, err := strconv.Atoi(r.FormValue("day_of_week"))
	if err != nil || day < 0 || day > 6 {
		http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
		return
	}
	start, errS := parseClock(r.FormValue("start_time"))
	end, errE := parseClock(r.FormValue("end_time"))
	if errS != nil || errE != nil || start == end {
		http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
		return
	}

	if err := h.q.CreateSchedule(r.Context(), sqlc.CreateScheduleParams{
		ProgramID: id,
		DayOfWeek: int8(day),
		StartTime: start,
		EndTime:   end,
		Host:      toNullString(r.FormValue("host")),
	}); err != nil {
		slog.Error("create schedule failed", "err", err, "program_id", id, "day", day)
	}
	http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// ScheduleDelete removes one schedule slot from a program.
func (h *Handler) ScheduleDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	sid, err := strconv.ParseUint(chi.URLParam(r, "scheduleID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = h.q.DeleteSchedule(r.Context(), sqlc.DeleteScheduleParams{ID: sid, ProgramID: id})
	http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// parseClock validates an "HH:MM" input (from an <input type=time>) and formats it
// as "HH:MM:00" for a MySQL TIME column.
func parseClock(s string) (string, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return "", err
	}
	return t.Format("15:04:05"), nil
}

func parseIDParam(r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil
}

// programFromForm reads program fields common to create/update from the request
// body (a multipart form, since the banner image is a real file upload). The
// image_url column defaults to whatever the hidden current_image_url field
// carries forward (i.e. "no change"), and is overwritten if a new file was
// submitted. uploadErr is non-nil if a file was submitted but rejected (wrong
// type or over the size limit) - callers should surface it and not save.
func (h *Handler) programFromForm(w http.ResponseWriter, r *http.Request) (p sqlc.Program, sortOrder int32, isActive bool, uploadErr error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	_ = r.ParseMultipartForm(maxUploadBytes + 1<<20)
	p.Title = strings.TrimSpace(r.FormValue("title"))
	p.Slug = strings.TrimSpace(r.FormValue("slug"))
	p.Description = toNullString(r.FormValue("description"))
	p.Host = toNullString(r.FormValue("host"))
	p.ImageUrl = toNullString(r.FormValue("current_image_url"))
	if url, err := h.saveUploadedImage(r, "image", uploadSubdirPrograms); err != nil {
		uploadErr = err
	} else if url != "" {
		p.ImageUrl = toNullString(url)
	}
	if n, err := strconv.Atoi(r.FormValue("sort_order")); err == nil {
		sortOrder = int32(n)
	}
	isActive = r.FormValue("is_active") == "on"
	return p, sortOrder, isActive, uploadErr
}

func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// friendlyDBError swaps a raw driver error for a human message when it looks like a
// duplicate-key violation, otherwise falls back to a generic message.
func friendlyDBError(err error, dupMsg string) string {
	msg := err.Error()
	if strings.Contains(msg, "Duplicate entry") {
		return dupMsg
	}
	return "Failed to save program: " + msg
}
