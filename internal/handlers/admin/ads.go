package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/models"
)

// defaultPlaceholderText is used when an admin enables the empty-slot placeholder
// but leaves its label blank.
const defaultPlaceholderText = "Ad space available"

// adBannerRow is one banner plus the labels of the pages it targets, so the list
// can show targeting without a lookup per row. Empty PageLabels means every page.
type adBannerRow struct {
	sqlc.AdBanner
	PageLabels []string
}

// adSlotGroup pairs one placement slot with the banners assigned to it, so the
// admin page can render per-slot settings plus its banner list in one pass.
type adSlotGroup struct {
	Slot    sqlc.AdSlot
	Banners []adBannerRow
}

type adsListData struct {
	Base   baseData
	Groups []adSlotGroup
	Error  string
	// OpenSlot expands that slot's settings panel on load. The panels are
	// collapsed by default, so without this a rejected save would show an error
	// message with no visible form to correct. Empty on the normal render.
	OpenSlot string
}

// AdsList renders every ad slot with its settings and the banners inside it.
// Unlike the other admin lists this one isn't paginated or searchable - ads are
// a handful of rows, and grouping by placement is the only useful ordering.
func (h *Handler) AdsList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	groups, err := h.adSlotGroups(r)
	if err != nil {
		http.Error(w, "failed to load ads", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/ads_list", adsListData{
		Base:   h.base(r, "Ads", "ads"),
		Groups: groups,
	})
}

// adSlotGroups loads the slots, banners and page targets, then buckets each
// banner into its slot with its target labels resolved.
func (h *Handler) adSlotGroups(r *http.Request) ([]adSlotGroup, error) {
	slots, err := h.q.ListAdSlots(r.Context())
	if err != nil {
		return nil, err
	}
	banners, err := h.q.ListAdBanners(r.Context())
	if err != nil {
		return nil, err
	}
	targets, err := h.q.ListAdBannerPages(r.Context())
	if err != nil {
		return nil, err
	}
	// One pass over every target row instead of a query per banner.
	labels := make(map[uint64][]string, len(banners))
	for _, t := range targets {
		labels[t.BannerID] = append(labels[t.BannerID], models.AdPageLabel(string(t.Page)))
	}
	groups := make([]adSlotGroup, 0, len(slots))
	for _, s := range slots {
		g := adSlotGroup{Slot: s}
		for _, b := range banners {
			if string(b.Slot) == string(s.Slot) {
				g.Banners = append(g.Banners, adBannerRow{AdBanner: b, PageLabels: labels[b.ID]})
			}
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// AdSlotUpdate saves one placement's presentation settings: stacked vs slideshow,
// the rotation interval, what an empty slot does, and whether the slot renders at
// all. Slots are fixed rows, so the {slot} param is validated against an allowlist
// rather than looked up.
func (h *Handler) AdSlotUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	slot := chi.URLParam(r, "slot")
	if !validAdSlot(slot) {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()

	renderErr := func(msg string) {
		groups, _ := h.adSlotGroups(r)
		h.r.Page(w, http.StatusBadRequest, "admin/ads_list", adsListData{
			Base:     h.base(r, "Ads", "ads"),
			Groups:   groups,
			Error:    msg,
			OpenSlot: slot,
		})
	}

	mode := strings.TrimSpace(r.FormValue("display_mode"))
	switch mode {
	case "stacked", "slideshow":
	default:
		renderErr("Invalid display mode.")
		return
	}

	// Clamped rather than rejected: the interval is a presentation detail, and a
	// sub-second rotation would be unreadable.
	rotate := int32(6)
	if n, err := strconv.Atoi(r.FormValue("rotate_secs")); err == nil {
		switch {
		case n < 2:
			rotate = 2
		case n > 60:
			rotate = 60
		default:
			rotate = int32(n)
		}
	}

	placeholderText := strings.TrimSpace(r.FormValue("placeholder_text"))
	if placeholderText == "" {
		placeholderText = defaultPlaceholderText
	}

	err := h.q.UpdateAdSlot(r.Context(), sqlc.UpdateAdSlotParams{
		DisplayMode:     sqlc.AdSlotsDisplayMode(mode),
		RotateSecs:      rotate,
		ShowPlaceholder: r.FormValue("show_placeholder") == "on",
		PlaceholderText: placeholderText,
		IsActive:        r.FormValue("is_active") == "on",
		Slot:            sqlc.AdSlotsSlot(slot),
	})
	if err != nil {
		http.Error(w, "failed to save ad slot", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "ad_slot", nil, "Updated ad slot "+slot)
	http.Redirect(w, r, "/admin/ads", http.StatusSeeOther)
}

// adPageChoice is one target-page checkbox on the banner form.
type adPageChoice struct {
	Key     string
	Label   string
	Checked bool
}

type adBannerFormData struct {
	Base     baseData
	IsNew    bool
	Banner   sqlc.AdBanner
	Slots    []sqlc.AdSlot
	Pages    []adPageChoice
	AllPages bool // no specific targets: the banner shows on every page
	Error    string
}

// adPageChoices builds the checkbox list from the page catalog, ticking the keys
// in selected.
func adPageChoices(selected []string) []adPageChoice {
	set := make(map[string]bool, len(selected))
	for _, k := range selected {
		set[k] = true
	}
	choices := make([]adPageChoice, 0, len(models.AdPages))
	for _, p := range models.AdPages {
		choices = append(choices, adPageChoice{Key: p.Key, Label: p.Label, Checked: set[p.Key]})
	}
	return choices
}

// adPagesFromForm reads the checked target pages. An explicit "all" checkbox wins
// over any individual ticks, and both it and an empty selection mean "every page",
// which is stored as zero rows. ok is false if a value isn't a known page key.
func adPagesFromForm(r *http.Request) (pages []string, ok bool) {
	for _, v := range r.Form["pages"] {
		if v == "all" {
			return nil, true
		}
		if !models.ValidAdPageKey(v) {
			return nil, false
		}
		pages = append(pages, v)
	}
	return pages, true
}

// saveAdBannerPages replaces a banner's target rows with pages.
//
// Not transactional: neither handler package holds a *sql.DB and Queries.WithTx
// is unused across the repo, so wrapping this would mean changing New's
// signature. A partial write leaves the banner targeting fewer pages (at worst
// zero rows, i.e. every page) rather than corrupting it, and the caller surfaces
// the error so the admin can re-save.
func (h *Handler) saveAdBannerPages(r *http.Request, bannerID uint64, pages []string) error {
	if err := h.q.DeleteAdBannerPages(r.Context(), bannerID); err != nil {
		return err
	}
	for _, p := range pages {
		err := h.q.CreateAdBannerPage(r.Context(), sqlc.CreateAdBannerPageParams{
			BannerID: bannerID,
			Page:     sqlc.AdBannerPagesPage(p),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// AdBannerNew renders the create-banner form. ?slot=top pre-selects a placement so
// the per-slot "Add banner" links land on the right slot.
func (h *Handler) AdBannerNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	banner := sqlc.AdBanner{IsActive: true, Slot: sqlc.AdBannersSlotTop}
	if slot := r.URL.Query().Get("slot"); validAdSlot(slot) {
		banner.Slot = sqlc.AdBannersSlot(slot)
	}
	slots, err := h.q.ListAdSlots(r.Context())
	if err != nil {
		http.Error(w, "failed to load ad slots", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/ads_form", adBannerFormData{
		Base:     h.base(r, "New Ad Banner", "ads"),
		IsNew:    true,
		Banner:   banner,
		Slots:    slots,
		Pages:    adPageChoices(nil),
		AllPages: true,
	})
}

// AdBannerCreate inserts a new ad banner.
func (h *Handler) AdBannerCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	b, uploadErr := h.adBannerFromForm(w, r)
	pages, pagesOK := adPagesFromForm(r)

	renderErr := func(msg string) {
		slots, _ := h.q.ListAdSlots(r.Context())
		h.r.Page(w, http.StatusBadRequest, "admin/ads_form", adBannerFormData{
			Base:     h.base(r, "New Ad Banner", "ads"),
			IsNew:    true,
			Banner:   b,
			Slots:    slots,
			Pages:    adPageChoices(pages),
			AllPages: len(pages) == 0,
			Error:    msg,
		})
	}

	if !pagesOK {
		renderErr("Choose valid target pages.")
		return
	}
	if msg := validateAdBanner(b, uploadErr); msg != "" {
		renderErr(msg)
		return
	}

	res, err := h.q.CreateAdBanner(r.Context(), sqlc.CreateAdBannerParams{
		Slot:      b.Slot,
		Title:     b.Title,
		AltText:   b.AltText,
		ImageUrl:  b.ImageUrl,
		LinkUrl:   b.LinkUrl,
		SortOrder: b.SortOrder,
		IsActive:  b.IsActive,
	})
	if err != nil {
		renderErr("Failed to save the banner.")
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	if err := h.saveAdBannerPages(r, uid, pages); err != nil {
		renderErr("Banner saved, but its target pages could not be saved. Edit it and try again.")
		return
	}
	h.audit(r, "create", "ad_banner", &uid, "Created ad banner "+b.Title)
	http.Redirect(w, r, "/admin/ads/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// AdBannerEdit renders the edit form for an existing banner.
func (h *Handler) AdBannerEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	banner, err := h.q.GetAdBanner(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	slots, err := h.q.ListAdSlots(r.Context())
	if err != nil {
		http.Error(w, "failed to load ad slots", http.StatusInternalServerError)
		return
	}
	targets, err := h.q.GetAdBannerPages(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to load target pages", http.StatusInternalServerError)
		return
	}
	pages := make([]string, 0, len(targets))
	for _, t := range targets {
		pages = append(pages, string(t))
	}
	h.r.Page(w, http.StatusOK, "admin/ads_form", adBannerFormData{
		Base:     h.base(r, "Edit Ad Banner", "ads"),
		IsNew:    false,
		Banner:   banner,
		Slots:    slots,
		Pages:    adPageChoices(pages),
		AllPages: len(pages) == 0,
	})
}

// AdBannerUpdate saves edits to an existing banner.
func (h *Handler) AdBannerUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	b, uploadErr := h.adBannerFromForm(w, r)
	b.ID = id
	pages, pagesOK := adPagesFromForm(r)

	renderErr := func(msg string) {
		slots, _ := h.q.ListAdSlots(r.Context())
		h.r.Page(w, http.StatusBadRequest, "admin/ads_form", adBannerFormData{
			Base:     h.base(r, "Edit Ad Banner", "ads"),
			IsNew:    false,
			Banner:   b,
			Slots:    slots,
			Pages:    adPageChoices(pages),
			AllPages: len(pages) == 0,
			Error:    msg,
		})
	}

	if !pagesOK {
		renderErr("Choose valid target pages.")
		return
	}
	if msg := validateAdBanner(b, uploadErr); msg != "" {
		renderErr(msg)
		return
	}

	err := h.q.UpdateAdBanner(r.Context(), sqlc.UpdateAdBannerParams{
		Slot:      b.Slot,
		Title:     b.Title,
		AltText:   b.AltText,
		ImageUrl:  b.ImageUrl,
		LinkUrl:   b.LinkUrl,
		SortOrder: b.SortOrder,
		IsActive:  b.IsActive,
		ID:        id,
	})
	if err != nil {
		renderErr("Failed to save the banner.")
		return
	}
	if err := h.saveAdBannerPages(r, id, pages); err != nil {
		renderErr("Failed to save the target pages.")
		return
	}
	h.audit(r, "update", "ad_banner", &id, "Updated ad banner "+b.Title)
	http.Redirect(w, r, "/admin/ads/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// AdBannerDelete removes a banner. Like every other entity here, the uploaded
// creative is left on disk (see saveUploadedImage).
func (h *Handler) AdBannerDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteAdBanner(r.Context(), id); err != nil {
		http.Error(w, "failed to delete ad banner", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "ad_banner", &id, "Deleted ad banner")
	http.Redirect(w, r, "/admin/ads", http.StatusSeeOther)
}

// adBannerFromForm reads banner fields common to create/update from the request
// body (a multipart form, since the creative is a real file upload). image_url
// defaults to whatever the hidden current_image_url field carries forward (i.e.
// "no change"), and is overwritten if a new file was submitted. uploadErr is
// non-nil if a file was submitted but rejected - callers should surface it and
// not save.
func (h *Handler) adBannerFromForm(w http.ResponseWriter, r *http.Request) (b sqlc.AdBanner, uploadErr error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	_ = r.ParseMultipartForm(maxUploadBytes + 1<<20)
	b.Slot = sqlc.AdBannersSlot(strings.TrimSpace(r.FormValue("slot")))
	b.Title = strings.TrimSpace(r.FormValue("title"))
	b.AltText = strings.TrimSpace(r.FormValue("alt_text"))
	b.LinkUrl = toNullString(r.FormValue("link_url"))
	b.ImageUrl = strings.TrimSpace(r.FormValue("current_image_url"))
	if u, err := h.saveUploadedImage(r, "image", uploadSubdirAds); err != nil {
		uploadErr = err
	} else if u != "" {
		b.ImageUrl = u
	}
	if n, err := strconv.Atoi(r.FormValue("sort_order")); err == nil {
		b.SortOrder = int32(n)
	}
	b.IsActive = r.FormValue("is_active") == "on"
	return b, uploadErr
}

// validateAdBanner returns a user-facing message for the first problem found, or
// "" if the banner is savable.
func validateAdBanner(b sqlc.AdBanner, uploadErr error) string {
	if uploadErr != nil {
		return "Failed to upload image: " + uploadErr.Error()
	}
	if !validAdSlot(string(b.Slot)) {
		return "Choose a valid placement slot."
	}
	if b.ImageUrl == "" {
		return "A banner image is required."
	}
	if !validAdLink(b.LinkUrl.String) {
		return "Link must start with http:// or https://, or be a path starting with /."
	}
	return ""
}

// validAdSlot reports whether s is one of the fixed placement slots. Kept as an
// allowlist so a hand-crafted request can't write a value the layout can't render.
func validAdSlot(s string) bool {
	switch s {
	case "top", "bottom":
		return true
	}
	return false
}

// validAdLink accepts an empty link (banner is not clickable), a site-relative
// path, or an absolute http(s) URL. html/template would neuter a javascript: URL
// into #ZgotmplZ anyway, but rejecting it here tells the admin why.
func validAdLink(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "//") {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return true
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
