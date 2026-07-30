package admin

import (
	"context"
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
	sort, dir := parseSort(r, "sort_order", "asc", "title", "slug", "sort_order")
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
	Schedules []sqlc.ListSchedulesForProgramRow
	// Broadcasters is everyone presenting this program (per-slot assignments plus
	// the defaults); DefaultBroadcasters is named separately so the page can say
	// which set slots fall back to. Empty when the program has no defaults.
	Broadcasters        []sqlc.Broadcaster
	DefaultBroadcasters []sqlc.Broadcaster
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
	broadcasters, _ := h.q.ListBroadcastersForProgram(r.Context(), sqlc.ListBroadcastersForProgramParams{ProgramID: id})
	defaults, _ := h.q.ListProgramBroadcasters(r.Context(), id)
	h.r.Page(w, http.StatusOK, "admin/programs_detail", programDetailData{
		Base:                h.base(r, program.Title, "programs"),
		Program:             program,
		Schedules:           schedules,
		Broadcasters:        broadcasters,
		DefaultBroadcasters: defaults,
	})
}

// scheduleRow pairs a slot with the broadcaster ids assigned to that slot itself, so
// the form's per-slot <select multiple> can mark them selected. Empty means the slot
// inherits the program's defaults.
type scheduleRow struct {
	sqlc.ListSchedulesForProgramRow
	BroadcasterIDs []uint64
}

type programFormData struct {
	Base      baseData
	IsNew     bool
	Program   sqlc.Program
	Schedules []scheduleRow
	Weekdays  [7]string
	// Broadcasters is the full roster (the <select> options); SelectedBroadcasterIDs
	// is the program's default set. It is carried separately because sqlc.Program no
	// longer holds the relation, and because a validation-error re-render has to show
	// back what was just posted rather than what is stored.
	Broadcasters           []sqlc.Broadcaster
	SelectedBroadcasterIDs []uint64
	Error                  string
}

// ProgramNew renders the create-program form.
func (h *Handler) ProgramNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	h.r.Page(w, http.StatusOK, "admin/programs_form", programFormData{
		Base:         h.base(r, "New Program", "programs"),
		IsNew:        true,
		Program:      sqlc.Program{IsActive: true},
		Weekdays:     models.Weekdays(),
		Broadcasters: broadcasters,
	})
}

// ProgramCreate inserts a new program.
func (h *Handler) ProgramCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	p, sortOrder, isActive, uploadErr := h.programFromForm(w, r)
	selected := parseBroadcasterIDs(r.Form["broadcaster_ids"])
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/programs_form", programFormData{
			Base:                   h.base(r, "New Program", "programs"),
			IsNew:                  true,
			Program:                p,
			Weekdays:               models.Weekdays(),
			Broadcasters:           broadcasters,
			SelectedBroadcasterIDs: selected,
			Error:                  msg,
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
	h.setProgramBroadcasters(r.Context(), uid, selected)
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
	schedules, err := h.scheduleRows(r.Context(), id)
	if err != nil {
		slog.Error("list schedules failed", "err", err, "program_id", id)
		http.Error(w, "failed to load schedule", http.StatusInternalServerError)
		return
	}
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	defaults, _ := h.q.ListProgramBroadcasters(r.Context(), id)
	h.r.Page(w, http.StatusOK, "admin/programs_form", programFormData{
		Base:                   h.base(r, "Edit Program", "programs"),
		IsNew:                  false,
		Program:                program,
		Schedules:              schedules,
		Weekdays:               models.Weekdays(),
		Broadcasters:           broadcasters,
		SelectedBroadcasterIDs: broadcasterIDs(defaults),
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
	selected := parseBroadcasterIDs(r.Form["broadcaster_ids"])
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())

	renderErr := func(msg string) {
		schedules, _ := h.scheduleRows(r.Context(), id)
		h.r.Page(w, http.StatusBadRequest, "admin/programs_form", programFormData{
			Base:                   h.base(r, "Edit Program", "programs"),
			IsNew:                  false,
			Program:                p,
			Schedules:              schedules,
			Weekdays:               models.Weekdays(),
			Broadcasters:           broadcasters,
			SelectedBroadcasterIDs: selected,
			Error:                  msg,
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
		ImageUrl:    p.ImageUrl,
		SortOrder:   sortOrder,
		IsActive:    isActive,
		ID:          id,
	})
	if err != nil {
		renderErr(friendlyDBError(err, "Slug is already used by another program."))
		return
	}
	h.setProgramBroadcasters(r.Context(), id, selected)
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
	slot, ok := slotFromForm(r)
	if !ok {
		http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
		return
	}

	res, err := h.q.CreateSchedule(r.Context(), sqlc.CreateScheduleParams{
		ProgramID: id,
		DayOfWeek: slot.day,
		StartTime: slot.start,
		EndTime:   slot.end,
	})
	if err != nil {
		slog.Error("create schedule failed", "err", err, "program_id", id, "day", slot.day)
	} else if sid, err := res.LastInsertId(); err == nil {
		h.setScheduleBroadcasters(r.Context(), uint64(sid), slot.broadcasterIDs)
	}
	http.Redirect(w, r, "/admin/programs/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// ScheduleUpdate saves edits to one existing schedule slot. The program_id in the
// query is what stops this from touching another program's slot.
func (h *Handler) ScheduleUpdate(w http.ResponseWriter, r *http.Request) {
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
	redirect := "/admin/programs/" + strconv.FormatUint(id, 10) + "/edit"
	slot, ok := slotFromForm(r)
	if !ok {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	if err := h.q.UpdateSchedule(r.Context(), sqlc.UpdateScheduleParams{
		DayOfWeek: slot.day,
		StartTime: slot.start,
		EndTime:   slot.end,
		ID:        sid,
		ProgramID: id,
	}); err != nil {
		slog.Error("update schedule failed", "err", err, "program_id", id, "schedule_id", sid)
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	h.setScheduleBroadcasters(r.Context(), sid, slot.broadcasterIDs)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// slotFormValues is one submitted schedule slot, already validated.
type slotFormValues struct {
	day            int8
	start, end     string
	broadcasterIDs []uint64
}

// slotFromForm reads and validates the day/time/broadcaster fields shared by the
// add-slot form and each row's edit form. ok is false when the day is out of range or
// the times are unparseable or equal - callers redirect back to the form rather than
// surfacing a message, as the inputs are a <select> and two <input type=time> and only
// a hand-crafted request can fail this.
func slotFromForm(r *http.Request) (slotFormValues, bool) {
	day, err := strconv.Atoi(r.FormValue("day_of_week"))
	if err != nil || day < 0 || day > 6 {
		return slotFormValues{}, false
	}
	start, errS := parseClock(r.FormValue("start_time"))
	end, errE := parseClock(r.FormValue("end_time"))
	if errS != nil || errE != nil || start == end {
		return slotFormValues{}, false
	}
	return slotFormValues{
		day:            int8(day),
		start:          start,
		end:            end,
		broadcasterIDs: parseBroadcasterIDs(r.Form["broadcaster_ids"]),
	}, true
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

// parseBroadcasterIDs reads the values of a broadcaster <select multiple>. An empty
// selection means "none", which for a schedule slot defers to the program's default
// broadcasters and for a program means no defaults at all. Duplicates are dropped so a
// hand-crafted request can't trip the junction table's primary key.
func parseBroadcasterIDs(vs []string) []uint64 {
	var ids []uint64
	seen := map[uint64]bool{}
	for _, v := range vs {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// broadcasterIDs projects broadcaster rows down to their ids, for templates that only
// need to know which options are selected.
func broadcasterIDs(bs []sqlc.Broadcaster) []uint64 {
	ids := make([]uint64, len(bs))
	for i, b := range bs {
		ids[i] = b.ID
	}
	return ids
}

// setProgramBroadcasters replaces a program's default broadcaster set. Clear-then-insert
// without a transaction: a failure part-way leaves the program with fewer broadcasters
// than intended, which the admin can see and fix on the page they are already on.
func (h *Handler) setProgramBroadcasters(ctx context.Context, programID uint64, ids []uint64) {
	if err := h.q.ClearProgramBroadcasters(ctx, programID); err != nil {
		slog.Error("clear program broadcasters failed", "err", err, "program_id", programID)
		return
	}
	for _, bid := range ids {
		if err := h.q.AddProgramBroadcaster(ctx, sqlc.AddProgramBroadcasterParams{
			ProgramID: programID, BroadcasterID: bid,
		}); err != nil {
			slog.Error("add program broadcaster failed", "err", err, "program_id", programID, "broadcaster_id", bid)
		}
	}
}

// setScheduleBroadcasters replaces one slot's own broadcaster set. An empty set is
// meaningful: it puts the slot back on the program's defaults.
func (h *Handler) setScheduleBroadcasters(ctx context.Context, scheduleID uint64, ids []uint64) {
	if err := h.q.ClearScheduleBroadcasters(ctx, scheduleID); err != nil {
		slog.Error("clear schedule broadcasters failed", "err", err, "schedule_id", scheduleID)
		return
	}
	for _, bid := range ids {
		if err := h.q.AddScheduleBroadcaster(ctx, sqlc.AddScheduleBroadcasterParams{
			ScheduleID: scheduleID, BroadcasterID: bid,
		}); err != nil {
			slog.Error("add schedule broadcaster failed", "err", err, "schedule_id", scheduleID, "broadcaster_id", bid)
		}
	}
}

// scheduleRows loads a program's slots together with each slot's own broadcaster ids,
// in two queries rather than one per row.
func (h *Handler) scheduleRows(ctx context.Context, programID uint64) ([]scheduleRow, error) {
	slots, err := h.q.ListSchedulesForProgram(ctx, programID)
	if err != nil {
		return nil, err
	}
	assigned, err := h.q.ListScheduleBroadcasterIDsForProgram(ctx, programID)
	if err != nil {
		return nil, err
	}
	bySlot := map[uint64][]uint64{}
	for _, a := range assigned {
		bySlot[a.ScheduleID] = append(bySlot[a.ScheduleID], a.BroadcasterID)
	}
	rows := make([]scheduleRow, len(slots))
	for i, s := range slots {
		rows[i] = scheduleRow{ListSchedulesForProgramRow: s, BroadcasterIDs: bySlot[s.ID]}
	}
	return rows, nil
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
