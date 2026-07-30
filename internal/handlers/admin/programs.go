package admin

import (
	"context"
	"database/sql"
	"errors"
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

// scheduleRow is one row of the schedule editor, rendered identically whether it
// came from the database or straight back off a rejected submission. It is a plain
// view struct rather than an embedded sqlc row because those two sources have to
// produce the same shape, and a rejected save must show what the admin typed.
type scheduleRow struct {
	// Key is stable per row and disambiguates that row's fields in the POST: the
	// broadcaster picker posts a variable number of values, so it cannot ride in a
	// parallel array and is named slot_bc_<Key> instead. Stored slots use their id,
	// rows added client-side use "new-0", "new-1", ...
	Key            string
	ID             uint64 // 0 for a row that has never been saved
	DayOfWeek      int8
	StartTime      string // "HH:MM", the format <input type=time> reads and writes
	EndTime        string
	BroadcasterIDs []uint64
	// Inherited is the program's default broadcaster line, shown only on a slot with
	// no set of its own. Display hint only - the fallback itself lives in SQL.
	Inherited string
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
	h.flash(w, "Program created. Add its weekly schedule below.")
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
	// The whole page is one form now, so the schedule comes back with the program.
	// The rows are echoed straight back on any error rather than re-read from the
	// database, which would throw away what the admin just typed.
	slots, slotErr := slotsFromForm(r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/programs_form", programFormData{
			Base:                   h.base(r, "Edit Program", "programs"),
			IsNew:                  false,
			Program:                p,
			Schedules:              slots,
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

	// Validate the schedule before writing anything, so a bad slot doesn't leave the
	// program row saved and the page reporting an error.
	if slotErr != nil {
		renderErr(slotErr.Error())
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
	h.applySchedule(r.Context(), id, slots)
	h.audit(r, "update", "program", &id, "Updated program "+p.Title)
	h.flash(w, "Program saved.")
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
	h.flash(w, "Program deleted.")
	http.Redirect(w, r, "/admin/programs", http.StatusSeeOther)
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
	if err := parseUploadForm(w, r); err != nil {
		return p, sortOrder, isActive, err
	}
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
		row := scheduleRow{
			Key:            strconv.FormatUint(s.ID, 10),
			ID:             s.ID,
			DayOfWeek:      s.DayOfWeek,
			StartTime:      models.ClockLabel(s.StartTime),
			EndTime:        models.ClockLabel(s.EndTime),
			BroadcasterIDs: bySlot[s.ID],
		}
		// broadcaster_name is the *effective* line; it is the program's default set
		// only when the slot has no set of its own.
		if !s.HasOwnBroadcasters && s.BroadcasterName.Valid {
			row.Inherited = s.BroadcasterName.String
		}
		rows[i] = row
	}
	return rows, nil
}

// slotsFromForm reads the schedule editor's parallel arrays back into rows. It
// returns every row it was given even when one is invalid, so a rejected save can
// re-render exactly what the admin typed; err names the first problem found.
//
// Only a hand-crafted request can normally fail this - the controls are a <select>
// and two <input type=time> - but the previous per-row forms discarded such failures
// silently, which left no way to explain a slot that quietly refused to save.
func slotsFromForm(r *http.Request) ([]scheduleRow, error) {
	keys := r.Form["slot_key"]
	ids := r.Form["slot_id"]
	days := r.Form["slot_day"]
	starts := r.Form["slot_start"]
	ends := r.Form["slot_end"]

	rows := make([]scheduleRow, 0, len(keys))
	var firstErr error
	fail := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	for i, key := range keys {
		if i >= len(ids) || i >= len(days) || i >= len(starts) || i >= len(ends) {
			fail(errors.New("Schedule rows were submitted incompletely; please try again."))
			break
		}
		row := scheduleRow{Key: key, StartTime: starts[i], EndTime: ends[i]}
		if ids[i] != "" {
			id, err := strconv.ParseUint(ids[i], 10, 64)
			if err != nil {
				fail(errors.New("Invalid schedule row."))
				continue
			}
			row.ID = id
		}
		row.BroadcasterIDs = parseBroadcasterIDs(r.Form["slot_bc_"+key])

		day, err := strconv.Atoi(days[i])
		if err != nil || day < 0 || day > 6 {
			fail(errors.New("Every schedule slot needs a valid day."))
			rows = append(rows, row)
			continue
		}
		row.DayOfWeek = int8(day)

		start, errS := parseClock(starts[i])
		end, errE := parseClock(ends[i])
		switch {
		case errS != nil || errE != nil:
			fail(errors.New("Every schedule slot needs a valid start and end time."))
		case start == end:
			fail(errors.New("A schedule slot's start and end time must differ (" + weekdayName(row.DayOfWeek) + " " + starts[i] + ")."))
		}
		rows = append(rows, row)
	}
	return rows, firstErr
}

// applySchedule saves the submitted slots against what is stored: rows carrying an
// id are updated, rows without one are inserted, and any stored slot whose id is
// absent from the submission is deleted - the rows are server-rendered, so an id can
// only go missing because the admin removed that row.
//
// Not transactional, matching setProgramBroadcasters below: a failure part-way leaves
// the schedule partly saved, which the admin can see and fix on the page they are
// already on.
func (h *Handler) applySchedule(ctx context.Context, programID uint64, rows []scheduleRow) {
	stored, err := h.q.ListSchedulesForProgram(ctx, programID)
	if err != nil {
		slog.Error("list schedules failed", "err", err, "program_id", programID)
		return
	}
	kept := make(map[uint64]bool, len(rows))
	for _, row := range rows {
		if row.ID != 0 {
			kept[row.ID] = true
		}
	}
	for _, s := range stored {
		if kept[s.ID] {
			continue
		}
		if err := h.q.DeleteSchedule(ctx, sqlc.DeleteScheduleParams{ID: s.ID, ProgramID: programID}); err != nil {
			slog.Error("delete schedule failed", "err", err, "program_id", programID, "schedule_id", s.ID)
		}
	}

	for _, row := range rows {
		start, end := clockForDB(row.StartTime), clockForDB(row.EndTime)
		if row.ID == 0 {
			res, err := h.q.CreateSchedule(ctx, sqlc.CreateScheduleParams{
				ProgramID: programID, DayOfWeek: row.DayOfWeek, StartTime: start, EndTime: end,
			})
			if err != nil {
				slog.Error("create schedule failed", "err", err, "program_id", programID, "day", row.DayOfWeek)
				continue
			}
			sid, err := res.LastInsertId()
			if err != nil {
				slog.Error("create schedule id failed", "err", err, "program_id", programID)
				continue
			}
			h.setScheduleBroadcasters(ctx, uint64(sid), row.BroadcasterIDs)
			continue
		}
		// program_id in the query is what stops this touching another program's slot.
		if err := h.q.UpdateSchedule(ctx, sqlc.UpdateScheduleParams{
			DayOfWeek: row.DayOfWeek, StartTime: start, EndTime: end,
			ID: row.ID, ProgramID: programID,
		}); err != nil {
			slog.Error("update schedule failed", "err", err, "program_id", programID, "schedule_id", row.ID)
			continue
		}
		h.setScheduleBroadcasters(ctx, row.ID, row.BroadcasterIDs)
	}
}

// clockForDB widens a validated "HH:MM" to the "HH:MM:00" a MySQL TIME column wants.
func clockForDB(s string) string {
	v, err := parseClock(s)
	if err != nil {
		return "00:00:00"
	}
	return v
}

func weekdayName(d int8) string {
	names := models.Weekdays()
	if d < 0 || int(d) >= len(names) {
		return ""
	}
	return names[d]
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
