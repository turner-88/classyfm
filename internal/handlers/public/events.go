package public

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// eventCategory is one filter chip on the event list. The two values are fixed (a column,
// not a lookup table), so the list is built here rather than queried.
type eventCategory struct {
	Slug  string
	Label string
}

var eventCategories = []eventCategory{
	{Slug: "event", Label: "Event"},
	{Slug: "promo", Label: "Promo"},
}

// Event renders the public event list at /event, newest first, optionally filtered to one
// category via ?category=event|promo.
func (h *Handler) Event(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := int32((page - 1) * newsPageSize)

	var items []sqlc.Event
	var total int64
	selectedCategory := ""
	if h.q != nil {
		// An unknown ?category= value falls back to the unfiltered list rather than 404,
		// mirroring the podcast series-filter behavior.
		if sel := r.URL.Query().Get("category"); sel == "event" || sel == "promo" {
			selectedCategory = sel
		}
		if selectedCategory != "" {
			items, _ = h.q.ListPublishedEventsByCategory(r.Context(), sqlc.ListPublishedEventsByCategoryParams{
				Category: sqlc.EventsCategory(selectedCategory), Limit: newsPageSize, Offset: offset,
			})
			total, _ = h.q.CountPublishedEventsByCategory(r.Context(), sqlc.EventsCategory(selectedCategory))
		} else {
			items, _ = h.q.ListPublishedEvents(r.Context(), sqlc.ListPublishedEventsParams{
				Limit: newsPageSize, Offset: offset,
			})
			total, _ = h.q.CountPublishedEvents(r.Context())
		}
	}
	totalPages := int((total + newsPageSize - 1) / newsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	h.r.Page(w, http.StatusOK, "public/event", struct {
		Base             baseData
		Categories       []eventCategory
		SelectedCategory string
		Events           []sqlc.Event
		Page             int
		TotalPages       int
	}{h.base(r, "Event", "event", "Events and promos from "+h.station+"."), eventCategories, selectedCategory, items, page, totalPages})
}

// EventDetail renders a single event at /event/{slug}.
func (h *Handler) EventDetail(w http.ResponseWriter, r *http.Request) {
	if h.q == nil {
		h.NotFound(w, r)
		return
	}
	slug := chi.URLParam(r, "slug")
	item, err := h.q.GetPublishedEventBySlug(r.Context(), slug)
	if err != nil {
		h.NotFound(w, r)
		return
	}

	base := h.base(r, item.Title, "event", item.Description)
	base.OGImage = item.ImageUrl.String
	h.r.Page(w, http.StatusOK, "public/event_detail", struct {
		Base baseData
		Item sqlc.Event
	}{base, item})
}
