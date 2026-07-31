package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// heroOrderModes are the ways the home hero mixes news items with the image
// slides below, in the order the settings panel offers them. An allowlist, not a
// lookup: order_mode is an ENUM and a hand-crafted value must never reach it.
var heroOrderModes = []heroOrderMode{
	{Key: "updated", Label: "Newest first", Help: "News by publish date, image slides by last edit."},
	{Key: "random", Label: "Random", Help: "Reshuffled on every page load."},
	{Key: "news_first", Label: "News first", Help: "All news items, then image slides by order."},
	{Key: "images_first", Label: "Image slides first", Help: "Image slides by order, then all news items."},
}

// heroMaxCount bounds both slide counts. The public handler clamps to the same
// number independently, so a row written before this bound existed still renders.
const heroMaxCount = 10

// heroOrderMode is one ordering choice on the settings panel.
type heroOrderMode struct {
	Key     string
	Label   string
	Help    string
	Checked bool
}

type heroListData struct {
	Base     baseData
	Settings sqlc.HeroSetting
	Modes    []heroOrderMode
	// OrderLabel is the current mode's label, for the collapsed settings summary.
	OrderLabel string
	Slides     []sqlc.HeroSlide
	Pagination pagination
	Error      string
}

// HeroList renders the hero settings panel plus every image slide (active and
// inactive).
func (h *Handler) HeroList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	data, err := h.heroListData(r)
	if err != nil {
		http.Error(w, "failed to load hero slides", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/hero_list", data)
}

// heroListData loads everything the list page renders. The settings read is a
// hard error here, unlike on the public side where it falls back to defaults: an
// admin must not edit a phantom row and then save it over the real one.
func (h *Handler) heroListData(r *http.Request) (heroListData, error) {
	settings, err := h.q.GetHeroSettings(r.Context())
	if err != nil {
		return heroListData{}, err
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "sort_order", "asc", "title", "badge_label", "sort_order", "updated_at")
	total, err := h.q.CountHeroSlides(r.Context(), sqlc.CountHeroSlidesParams{Search: pattern})
	if err != nil {
		return heroListData{}, err
	}
	pg := paginate(r, total, "/admin/hero", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	slides, err := h.q.ListHeroSlides(r.Context(), sqlc.ListHeroSlidesParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		return heroListData{}, err
	}
	return heroListData{
		Base:       h.base(r, "Hero", "hero"),
		Settings:   settings,
		Modes:      heroOrderModeChoices(string(settings.OrderMode)),
		OrderLabel: heroOrderModeLabel(string(settings.OrderMode)),
		Slides:     slides,
		Pagination: pg,
	}, nil
}

// heroOrderModeChoices returns the radio options with the stored mode ticked.
func heroOrderModeChoices(current string) []heroOrderMode {
	modes := make([]heroOrderMode, len(heroOrderModes))
	copy(modes, heroOrderModes)
	for i := range modes {
		modes[i].Checked = modes[i].Key == current
	}
	return modes
}

// heroOrderModeLabel returns the display label for a stored mode, for the
// collapsed settings summary.
func heroOrderModeLabel(key string) string {
	for _, m := range heroOrderModes {
		if m.Key == key {
			return m.Label
		}
	}
	return key
}

// HeroSettingsUpdate saves how the home hero mixes its two sources: the ordering
// mode, how many news items to pull, and how many image slides.
func (h *Handler) HeroSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	// No file field on this form, so the plain parser is enough.
	_ = r.ParseForm()

	renderErr := func(msg string) {
		data, err := h.heroListData(r)
		if err != nil {
			http.Error(w, "failed to load hero slides", http.StatusInternalServerError)
			return
		}
		data.Error = msg
		h.r.Page(w, http.StatusBadRequest, "admin/hero_list", data)
	}

	mode := strings.TrimSpace(r.FormValue("order_mode"))
	if !validHeroOrderMode(mode) {
		renderErr("Choose a valid slide order.")
		return
	}

	if err := h.q.UpdateHeroSettings(r.Context(), sqlc.UpdateHeroSettingsParams{
		OrderMode: sqlc.HeroSettingsOrderMode(mode),
		// Clamped rather than rejected: like the ad rotation interval, these are
		// presentation dials, and a hero of 200 slides is a slip, not an intent.
		NewsCount: heroCountFromForm(r.FormValue("news_count")),
		MaxImages: heroCountFromForm(r.FormValue("max_images")),
	}); err != nil {
		http.Error(w, "failed to save hero settings", http.StatusInternalServerError)
		return
	}

	h.audit(r, "update", "hero_settings", nil, "Updated hero settings")
	h.flash(w, "Hero settings saved.")
	http.Redirect(w, r, "/admin/hero", http.StatusSeeOther)
}

// heroCountFromForm parses a slide count, clamped to 0..heroMaxCount. A blank or
// unparseable value means 0, i.e. "none from this source".
func heroCountFromForm(s string) int32 {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	if n > heroMaxCount {
		return heroMaxCount
	}
	return int32(n)
}

// validHeroOrderMode reports whether s is one of the known ordering modes.
func validHeroOrderMode(s string) bool {
	for _, m := range heroOrderModes {
		if m.Key == s {
			return true
		}
	}
	return false
}

type heroSlideFormData struct {
	Base  baseData
	IsNew bool
	Slide sqlc.HeroSlide
	Error string
}

// HeroSlideNew renders the create-slide form.
func (h *Handler) HeroSlideNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/hero_form", heroSlideFormData{
		Base:  h.base(r, "New Hero Slide", "hero"),
		IsNew: true,
		Slide: sqlc.HeroSlide{IsActive: true},
	})
}

// HeroSlideCreate inserts a new hero slide.
func (h *Handler) HeroSlideCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	s, uploadErr := h.heroSlideFromForm(w, r)

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/hero_form", heroSlideFormData{
			Base:  h.base(r, "New Hero Slide", "hero"),
			IsNew: true,
			Slide: s,
			Error: msg,
		})
	}

	if msg := validateHeroSlide(s, uploadErr); msg != "" {
		renderErr(msg)
		return
	}

	res, err := h.q.CreateHeroSlide(r.Context(), sqlc.CreateHeroSlideParams{
		Title:        s.Title,
		BadgeLabel:   s.BadgeLabel,
		Excerpt:      s.Excerpt,
		ImageUrl:     s.ImageUrl,
		LinkUrl:      s.LinkUrl,
		OpenInNewTab: s.OpenInNewTab,
		SortOrder:    s.SortOrder,
		IsActive:     s.IsActive,
	})
	if err != nil {
		renderErr("Failed to save the slide.")
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.audit(r, "create", "hero_slide", &uid, "Created hero slide "+s.Title)
	h.flash(w, "Hero slide created.")
	http.Redirect(w, r, "/admin/hero/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// HeroSlideEdit renders the edit form for an existing slide.
func (h *Handler) HeroSlideEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	slide, err := h.q.GetHeroSlide(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/hero_form", heroSlideFormData{
		Base:  h.base(r, "Edit Hero Slide", "hero"),
		IsNew: false,
		Slide: slide,
	})
}

// HeroSlideUpdate saves edits to an existing slide.
func (h *Handler) HeroSlideUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s, uploadErr := h.heroSlideFromForm(w, r)
	s.ID = id

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/hero_form", heroSlideFormData{
			Base:  h.base(r, "Edit Hero Slide", "hero"),
			IsNew: false,
			Slide: s,
			Error: msg,
		})
	}

	if msg := validateHeroSlide(s, uploadErr); msg != "" {
		renderErr(msg)
		return
	}

	err := h.q.UpdateHeroSlide(r.Context(), sqlc.UpdateHeroSlideParams{
		Title:        s.Title,
		BadgeLabel:   s.BadgeLabel,
		Excerpt:      s.Excerpt,
		ImageUrl:     s.ImageUrl,
		LinkUrl:      s.LinkUrl,
		OpenInNewTab: s.OpenInNewTab,
		SortOrder:    s.SortOrder,
		IsActive:     s.IsActive,
		ID:           id,
	})
	if err != nil {
		renderErr("Failed to save the slide.")
		return
	}
	h.audit(r, "update", "hero_slide", &id, "Updated hero slide "+s.Title)
	h.flash(w, "Hero slide saved.")
	http.Redirect(w, r, "/admin/hero/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// HeroSlideDelete removes a slide. Like every other entity here, the uploaded
// image is left on disk (see saveUploadedImage).
func (h *Handler) HeroSlideDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeleteHeroSlide(r.Context(), id); err != nil {
		http.Error(w, "failed to delete hero slide", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "hero_slide", &id, "Deleted hero slide")
	h.flash(w, "Hero slide deleted.")
	http.Redirect(w, r, "/admin/hero", http.StatusSeeOther)
}

// heroSlideFromForm reads slide fields common to create/update from the request
// body (a multipart form, since the image is a real file upload). image_url
// defaults to whatever the hidden current_image_url field carries forward (i.e.
// "no change"), and is overwritten if a new file was submitted. uploadErr is
// non-nil if a file was submitted but rejected - callers should surface it and
// not save.
func (h *Handler) heroSlideFromForm(w http.ResponseWriter, r *http.Request) (s sqlc.HeroSlide, uploadErr error) {
	if err := parseUploadForm(w, r); err != nil {
		return s, err
	}
	s.Title = strings.TrimSpace(r.FormValue("title"))
	s.BadgeLabel = strings.TrimSpace(r.FormValue("badge_label"))
	s.Excerpt = toNullString(r.FormValue("excerpt"))
	s.LinkUrl = toNullString(r.FormValue("link_url"))
	s.ImageUrl = strings.TrimSpace(r.FormValue("current_image_url"))
	if u, err := h.saveUploadedImage(r, "image", uploadSubdirHero); err != nil {
		uploadErr = err
	} else if u != "" {
		s.ImageUrl = u
	}
	s.OpenInNewTab = r.FormValue("open_in_new_tab") == "on"
	if n, err := strconv.Atoi(r.FormValue("sort_order")); err == nil {
		s.SortOrder = int32(n)
	}
	s.IsActive = r.FormValue("is_active") == "on"
	return s, uploadErr
}

// validateHeroSlide returns a user-facing message for the first problem found, or
// "" if the slide is savable. The link rule is validAdLink's: an empty link (the
// slide is a plain banner), a site-relative path, or an absolute http(s) URL.
func validateHeroSlide(s sqlc.HeroSlide, uploadErr error) string {
	if uploadErr != nil {
		return "Failed to upload image: " + uploadErr.Error()
	}
	if s.ImageUrl == "" {
		return "A slide image is required."
	}
	if !validAdLink(s.LinkUrl.String) {
		return "Link must start with http:// or https://, or be a path starting with /."
	}
	return ""
}
