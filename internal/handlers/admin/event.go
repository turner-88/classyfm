package admin

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
)

// eventDateLayout is the wire format of the form's <input type="datetime-local">, the
// same layout the Hot Release editor uses for published_at.
const eventDateLayout = "2006-01-02T15:04"

type eventsListData struct {
	Base       baseData
	Events     []sqlc.Event
	Pagination pagination
}

// EventsList renders every event (published and unpublished).
func (h *Handler) EventsList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "created_at", "desc", "title", "created_at")
	total, err := h.q.CountEvents(r.Context(), pattern)
	if err != nil {
		http.Error(w, "failed to load events", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/events", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	events, err := h.q.ListEvents(r.Context(), sqlc.ListEventsParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load events", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/events_list", eventsListData{
		Base:       h.base(r, "Event", "event"),
		Events:     events,
		Pagination: pg,
	})
}

// eventFormData is the view-model shared by the create and edit forms. EventDateForm is
// the event date formatted for <input type="datetime-local"> (blank when unset), carried
// separately so a validation-error re-render shows what was just posted.
type eventFormData struct {
	Base          baseData
	IsNew         bool
	Event         sqlc.Event
	EventDateForm string
	Error         string
}

// EventNew renders the create form.
func (h *Handler) EventNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/events_form", eventFormData{
		Base:  h.base(r, "New Event", "event"),
		IsNew: true,
		Event: sqlc.Event{Category: sqlc.EventsCategoryEvent, IsPublished: true},
	})
}

// EventCreate inserts a new event.
func (h *Handler) EventCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	e, formErr, uploadErr := h.eventFromForm(r, w)

	renderErr := func(status int, msg string) {
		h.r.Page(w, status, "admin/events_form", eventFormData{
			Base:          h.base(r, "New Event", "event"),
			IsNew:         true,
			Event:         e,
			EventDateForm: r.FormValue("event_date"),
			Error:         msg,
		})
	}

	if uploadErr != nil {
		renderErr(http.StatusBadRequest, uploadErr.Error())
		return
	}
	if formErr != "" {
		renderErr(http.StatusBadRequest, formErr)
		return
	}

	e.Slug = h.uniqueEventSlug(r.Context(), slugify(e.Title), 0)

	res, err := h.q.CreateEvent(r.Context(), sqlc.CreateEventParams{
		Title:       e.Title,
		Slug:        e.Slug,
		Category:    e.Category,
		Description: e.Description,
		ImageUrl:    e.ImageUrl,
		EventDate:   e.EventDate,
		Location:    e.Location,
		LinkUrl:     e.LinkUrl,
		IsPublished: e.IsPublished,
	})
	if err != nil {
		renderErr(http.StatusInternalServerError, "Failed to save event: "+err.Error())
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "event", &uid, "Created event "+e.Title)
	h.flash(w, "Event created.")
	http.Redirect(w, r, "/admin/events/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// EventEdit renders the edit form for an existing event.
func (h *Handler) EventEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	event, err := h.q.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/events_form", eventFormData{
		Base:          h.base(r, "Edit Event", "event"),
		IsNew:         false,
		Event:         event,
		EventDateForm: eventDateForInput(event.EventDate),
	})
}

// EventUpdate saves edits to an existing event.
func (h *Handler) EventUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := h.q.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	e, formErr, uploadErr := h.eventFromForm(r, w)
	e.ID = id

	renderErr := func(status int, msg string) {
		h.r.Page(w, status, "admin/events_form", eventFormData{
			Base:          h.base(r, "Edit Event", "event"),
			IsNew:         false,
			Event:         e,
			EventDateForm: r.FormValue("event_date"),
			Error:         msg,
		})
	}

	if uploadErr != nil {
		renderErr(http.StatusBadRequest, uploadErr.Error())
		return
	}
	if formErr != "" {
		renderErr(http.StatusBadRequest, formErr)
		return
	}

	// The slug follows the title but only changes when the title does, so an existing
	// public URL stays stable across unrelated edits.
	e.Slug = existing.Slug
	if slugify(e.Title) != slugify(existing.Title) {
		e.Slug = h.uniqueEventSlug(r.Context(), slugify(e.Title), id)
	}

	if err := h.q.UpdateEvent(r.Context(), sqlc.UpdateEventParams{
		Title:       e.Title,
		Slug:        e.Slug,
		Category:    e.Category,
		Description: e.Description,
		ImageUrl:    e.ImageUrl,
		EventDate:   e.EventDate,
		Location:    e.Location,
		LinkUrl:     e.LinkUrl,
		IsPublished: e.IsPublished,
		ID:          id,
	}); err != nil {
		renderErr(http.StatusInternalServerError, "Failed to save event: "+err.Error())
		return
	}
	h.audit(r, "update", "event", &id, "Updated event "+e.Title)
	h.flash(w, "Event saved.")
	http.Redirect(w, r, "/admin/events/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// EventDelete removes an event.
func (h *Handler) EventDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteEvent(r.Context(), id); err != nil {
		http.Error(w, "failed to delete event", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "event", &id, "Deleted event")
	h.flash(w, "Event deleted.")
	http.Redirect(w, r, "/admin/events", http.StatusSeeOther)
}

// eventFromForm reads and validates the event fields from the request. formErr is
// non-empty when a required field is missing; uploadErr carries an image-upload failure
// (checked first by the caller). The returned event carries whatever was typed so an
// error re-render shows it back. The slug is derived from the title by the caller.
func (h *Handler) eventFromForm(r *http.Request, w http.ResponseWriter) (e sqlc.Event, formErr string, uploadErr error) {
	if err := parseUploadForm(w, r); err != nil {
		return e, "", err
	}
	e.Title = strings.TrimSpace(r.FormValue("title"))
	e.Category = sqlc.EventsCategoryEvent
	if r.FormValue("category") == "promo" {
		e.Category = sqlc.EventsCategoryPromo
	}
	e.Description = sanitize.PlainText(r.FormValue("description"))
	e.Location = toNullString(r.FormValue("location"))
	e.LinkUrl = toNullString(r.FormValue("link_url"))

	// Keep the existing banner unless a new file was chosen; saveUploadedImage returns
	// "" for an empty field, matching the Hot Release / Broadcaster upload flow.
	e.ImageUrl = toNullString(r.FormValue("current_image_url"))
	if urlStr, err := h.saveUploadedImage(r, "image", uploadSubdirEvent); err != nil {
		uploadErr = err
	} else if urlStr != "" {
		e.ImageUrl = toNullString(urlStr)
	}

	e.IsPublished = r.FormValue("is_published") == "on"

	if e.Title == "" {
		return e, "Title is required.", uploadErr
	}
	if strings.TrimSpace(e.Description) == "" {
		return e, "Description is required.", uploadErr
	}

	// The date is optional (promos may run without one); an empty field stays NULL, a
	// malformed one is a hard error so it is not silently dropped.
	if raw := strings.TrimSpace(r.FormValue("event_date")); raw != "" {
		parsed, err := time.ParseInLocation(eventDateLayout, raw, time.Local)
		if err != nil {
			return e, "Invalid event date.", uploadErr
		}
		e.EventDate = sql.NullTime{Time: parsed, Valid: true}
	}
	return e, "", uploadErr
}

// uniqueEventSlug returns base, or base-2, base-3, ... if a different event already holds
// it. excludeID is the event being updated (0 on create), so re-saving without a title
// change keeps its own slug.
func (h *Handler) uniqueEventSlug(ctx context.Context, base string, excludeID uint64) string {
	if base == "" {
		base = "event"
	}
	candidate := base
	for n := 2; ; n++ {
		existing, err := h.q.GetEventBySlug(ctx, candidate)
		if err != nil || existing.ID == excludeID {
			return candidate
		}
		candidate = base + "-" + strconv.Itoa(n)
	}
}

// eventDateForInput formats a stored event date for the form's datetime-local input,
// returning "" when unset.
func eventDateForInput(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(eventDateLayout)
}
