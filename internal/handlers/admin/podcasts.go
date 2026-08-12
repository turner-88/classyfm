package admin

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/sanitize"
	"github.com/classyfm/classyfm/internal/spotify"
)

// spotifyClient resolves podcast artwork from Spotify's oEmbed endpoint. It is
// package-level with a short timeout because the fetch is best-effort - a slow or
// failing Spotify must not hold up (or block) saving the podcast.
var spotifyClient = &http.Client{Timeout: 6 * time.Second}

type podcastsListData struct {
	Base       baseData
	Podcasts   []sqlc.ListPodcastsRow
	Pagination pagination
}

// PodcastsList renders every podcast (published and unpublished).
func (h *Handler) PodcastsList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "created_at", "desc", "title", "created_at")
	total, err := h.q.CountPodcasts(r.Context(), pattern)
	if err != nil {
		http.Error(w, "failed to load podcasts", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/podcasts", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	podcasts, err := h.q.ListPodcasts(r.Context(), sqlc.ListPodcastsParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load podcasts", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/podcasts_list", podcastsListData{
		Base:       h.base(r, "Podcast", "podcast"),
		Podcasts:   podcasts,
		Pagination: pg,
	})
}

// podcastFormData is the view-model shared by the create and edit forms. Broadcasters
// is the full roster (the multiselect options); SelectedBroadcasterIDs is this
// podcast's set, carried separately so a validation-error re-render shows what was
// just posted rather than what is stored.
type podcastFormData struct {
	Base                   baseData
	IsNew                  bool
	Podcast                sqlc.Podcast
	Series                 []sqlc.PodcastSeries
	Broadcasters           []sqlc.Broadcaster
	SelectedBroadcasterIDs []uint64
	Error                  string
}

// PodcastNew renders the create form.
func (h *Handler) PodcastNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	series, _ := h.q.ListPodcastSeries(r.Context())
	h.r.Page(w, http.StatusOK, "admin/podcasts_form", podcastFormData{
		Base:         h.base(r, "New Podcast", "podcast"),
		IsNew:        true,
		Podcast:      sqlc.Podcast{IsPublished: true},
		Series:       series,
		Broadcasters: broadcasters,
	})
}

// PodcastCreate inserts a new podcast.
func (h *Handler) PodcastCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	p, formErr := h.podcastFromForm(r)
	selected := parseBroadcasterIDs(r.Form["broadcaster_ids"])
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	series, _ := h.q.ListPodcastSeries(r.Context())

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/podcasts_form", podcastFormData{
			Base:                   h.base(r, "New Podcast", "podcast"),
			IsNew:                  true,
			Podcast:                p,
			Series:                 series,
			Broadcasters:           broadcasters,
			SelectedBroadcasterIDs: selected,
			Error:                  msg,
		})
	}

	if formErr != "" {
		renderErr(formErr)
		return
	}

	// Title, thumbnail, and description are all derived from the Spotify link. The
	// title is required (NOT NULL, drives the slug and all display), so a blank result -
	// an invalid link or a transient fetch failure - is a hard error rather than a saved
	// row with no title.
	p.Title, p.ThumbUrl = fetchPodcastOEmbed(r.Context(), p.SpotifyUrl)
	if p.Title == "" {
		renderErr("Couldn't read the title from Spotify - check the URL and try again.")
		return
	}
	p.Description = fetchPodcastDescription(r.Context(), p.SpotifyUrl)
	p.Slug = h.uniquePodcastSlug(r.Context(), slugify(p.Title), 0)

	res, err := h.q.CreatePodcast(r.Context(), sqlc.CreatePodcastParams{
		Title:       p.Title,
		Slug:        p.Slug,
		SeriesID:    p.SeriesID,
		Description: p.Description,
		SpotifyUrl:  p.SpotifyUrl,
		ThumbUrl:    p.ThumbUrl,
		IsPublished: p.IsPublished,
	})
	if err != nil {
		renderErr("Failed to save podcast: " + err.Error())
		return
	}
	id, _ := res.LastInsertId()
	uid := uint64(id)
	h.setPodcastBroadcasters(r.Context(), uid, selected)
	h.audit(r, "create", "podcast", &uid, "Created podcast "+p.Title)
	h.flash(w, "Podcast created.")
	http.Redirect(w, r, "/admin/podcasts/"+strconv.FormatInt(id, 10)+"/edit", http.StatusSeeOther)
}

// PodcastEdit renders the edit form for an existing podcast.
func (h *Handler) PodcastEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	podcast, err := h.q.GetPodcast(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	series, _ := h.q.ListPodcastSeries(r.Context())
	current, _ := h.q.ListPodcastBroadcasters(r.Context(), id)
	h.r.Page(w, http.StatusOK, "admin/podcasts_form", podcastFormData{
		Base:                   h.base(r, "Edit Podcast", "podcast"),
		IsNew:                  false,
		Podcast:                podcast,
		Series:                 series,
		Broadcasters:           broadcasters,
		SelectedBroadcasterIDs: broadcasterIDs(current),
	})
}

// PodcastUpdate saves edits to an existing podcast.
func (h *Handler) PodcastUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := h.q.GetPodcast(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, formErr := h.podcastFromForm(r)
	p.ID = id
	selected := parseBroadcasterIDs(r.Form["broadcaster_ids"])
	broadcasters, _ := h.q.ListAllBroadcasters(r.Context())
	series, _ := h.q.ListPodcastSeries(r.Context())

	renderErr := func(msg string) {
		h.r.Page(w, http.StatusBadRequest, "admin/podcasts_form", podcastFormData{
			Base:                   h.base(r, "Edit Podcast", "podcast"),
			IsNew:                  false,
			Podcast:                p,
			Series:                 series,
			Broadcasters:           broadcasters,
			SelectedBroadcasterIDs: selected,
			Error:                  msg,
		})
	}

	if formErr != "" {
		renderErr(formErr)
		return
	}

	// Title, thumbnail, and description all come from Spotify, so keep the stored ones
	// unless the link changed; re-resolve only then. The title keeps a non-empty guard
	// (a transient failure must never blank a NOT NULL field), while thumb/description
	// may legitimately go empty.
	p.Title = existing.Title
	p.ThumbUrl = existing.ThumbUrl
	p.Description = existing.Description
	if p.SpotifyUrl != existing.SpotifyUrl {
		if freshTitle, freshThumb := fetchPodcastOEmbed(r.Context(), p.SpotifyUrl); freshTitle != "" {
			p.Title = freshTitle
			p.ThumbUrl = freshThumb
		}
		p.Description = fetchPodcastDescription(r.Context(), p.SpotifyUrl)
	}
	// The slug follows the title but only changes when the title does, so an existing
	// public URL stays stable across unrelated edits.
	p.Slug = existing.Slug
	if slugify(p.Title) != slugify(existing.Title) {
		p.Slug = h.uniquePodcastSlug(r.Context(), slugify(p.Title), id)
	}

	if err := h.q.UpdatePodcast(r.Context(), sqlc.UpdatePodcastParams{
		Title:       p.Title,
		Slug:        p.Slug,
		SeriesID:    p.SeriesID,
		Description: p.Description,
		SpotifyUrl:  p.SpotifyUrl,
		ThumbUrl:    p.ThumbUrl,
		IsPublished: p.IsPublished,
		ID:          id,
	}); err != nil {
		renderErr("Failed to save podcast: " + err.Error())
		return
	}
	h.setPodcastBroadcasters(r.Context(), id, selected)
	h.audit(r, "update", "podcast", &id, "Updated podcast "+p.Title)
	h.flash(w, "Podcast saved.")
	http.Redirect(w, r, "/admin/podcasts/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// PodcastRefresh re-pulls the title, thumbnail, and description from the podcast's stored
// Spotify link, without touching any other field. It ignores the posted form body and
// reads the saved spotify_url, so it works even when the admin has unsaved edits or the
// required fields are momentarily blank (the button posts with formnovalidate). The
// fetches are best-effort: a fresh value overwrites, but an empty result (a transient
// failure or a genuinely blank field) keeps the existing one rather than blanking good
// data - the same rule PodcastUpdate applies on a URL change. The slug is left untouched
// so a refresh never breaks the live public URL.
func (h *Handler) PodcastRefresh(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := h.q.GetPodcast(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	title := existing.Title
	thumb := existing.ThumbUrl
	if freshTitle, freshThumb := fetchPodcastOEmbed(r.Context(), existing.SpotifyUrl); freshTitle != "" {
		title = freshTitle
		thumb = freshThumb
	}
	desc := existing.Description
	if fresh := fetchPodcastDescription(r.Context(), existing.SpotifyUrl); fresh != "" {
		desc = fresh
	}

	// The slug is intentionally left untouched here: a manual refresh must not break the
	// live public URL even when the episode's title changed upstream.
	if err := h.q.UpdatePodcast(r.Context(), sqlc.UpdatePodcastParams{
		Title:       title,
		Slug:        existing.Slug,
		SeriesID:    existing.SeriesID,
		Description: desc,
		SpotifyUrl:  existing.SpotifyUrl,
		ThumbUrl:    thumb,
		IsPublished: existing.IsPublished,
		ID:          id,
	}); err != nil {
		http.Error(w, "failed to refresh podcast", http.StatusInternalServerError)
		return
	}
	h.audit(r, "update", "podcast", &id, "Refreshed podcast from Spotify")
	h.flash(w, "Title, thumbnail, and description refreshed from Spotify.")
	http.Redirect(w, r, "/admin/podcasts/"+strconv.FormatUint(id, 10)+"/edit", http.StatusSeeOther)
}

// PodcastDelete removes a podcast (its broadcaster rows go via ON DELETE CASCADE).
func (h *Handler) PodcastDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.q.DeletePodcast(r.Context(), id); err != nil {
		http.Error(w, "failed to delete podcast", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "podcast", &id, "Deleted podcast")
	h.flash(w, "Podcast deleted.")
	http.Redirect(w, r, "/admin/podcasts", http.StatusSeeOther)
}

// podcastFromForm reads and validates the podcast fields from the request. formErr is
// non-empty when a required field is missing; the returned podcast carries whatever was
// typed so an error re-render shows it back. Title, slug, thumb_url, and description are
// not user input - they are resolved from the Spotify link by the caller, not here.
func (h *Handler) podcastFromForm(r *http.Request) (p sqlc.Podcast, formErr string) {
	if err := r.ParseForm(); err != nil {
		return p, "Invalid form submission."
	}
	p.SeriesID, _ = strconv.ParseUint(r.FormValue("series_id"), 10, 64)
	p.SpotifyUrl = strings.TrimSpace(r.FormValue("spotify_url"))
	p.IsPublished = r.FormValue("is_published") == "on"

	switch {
	case p.SeriesID == 0:
		return p, "A series is required."
	case spotify.EmbedURL(p.SpotifyUrl) == "":
		return p, "A valid Spotify URL (open.spotify.com/...) is required."
	}
	return p, ""
}

// setPodcastBroadcasters replaces a podcast's broadcaster set. Clear-then-insert
// without a transaction, matching setProgramBroadcasters: a failure part-way leaves
// the podcast with fewer broadcasters than intended, which the admin can see and fix
// on the page they are already on.
func (h *Handler) setPodcastBroadcasters(ctx context.Context, podcastID uint64, ids []uint64) {
	if err := h.q.ClearPodcastBroadcasters(ctx, podcastID); err != nil {
		slog.Error("clear podcast broadcasters failed", "err", err, "podcast_id", podcastID)
		return
	}
	for _, bid := range ids {
		if err := h.q.AddPodcastBroadcaster(ctx, sqlc.AddPodcastBroadcasterParams{
			PodcastID: podcastID, BroadcasterID: bid,
		}); err != nil {
			slog.Error("add podcast broadcaster failed", "err", err, "podcast_id", podcastID, "broadcaster_id", bid)
		}
	}
}

// uniquePodcastSlug returns base, or base-2, base-3, ... if a different podcast already
// holds it. excludeID is the podcast being updated (0 on create), so re-saving a
// podcast without a title change keeps its own slug.
func (h *Handler) uniquePodcastSlug(ctx context.Context, base string, excludeID uint64) string {
	if base == "" {
		base = "podcast"
	}
	candidate := base
	for n := 2; ; n++ {
		existing, err := h.q.GetPodcastBySlug(ctx, candidate)
		if err != nil || existing.ID == excludeID {
			return candidate
		}
		candidate = base + "-" + strconv.Itoa(n)
	}
}

// fetchPodcastOEmbed resolves the title and artwork URL for a Spotify link from a single
// oEmbed call, returning an empty title / empty NullString when the link is invalid or
// the fetch fails - callers decide whether an empty result is acceptable (it is for the
// thumbnail; the title is required, so PodcastCreate refuses a blank one). The title is
// run through PlainText to match the sanitization the description receives.
func fetchPodcastOEmbed(ctx context.Context, spotifyURL string) (title string, thumb sql.NullString) {
	oe, err := spotify.FetchOEmbed(ctx, spotifyClient, spotifyURL)
	if err != nil {
		slog.Warn("spotify oembed fetch failed", "err", err, "url", spotifyURL)
	}
	return sanitize.PlainText(oe.Title), toNullString(oe.ThumbnailURL)
}

// fetchPodcastDescription resolves the episode description for a Spotify link, returning
// an empty string when the link is invalid or the fetch fails - the podcast still saves,
// just without a stored description. The result is run through PlainText to match the
// sanitization the field previously received from manual entry.
func fetchPodcastDescription(ctx context.Context, spotifyURL string) string {
	desc, err := spotify.FetchDescription(ctx, spotifyClient, spotifyURL)
	if err != nil {
		slog.Warn("spotify description fetch failed", "err", err, "url", spotifyURL)
	}
	return sanitize.PlainText(desc)
}

var slugNonWord = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases a title and reduces it to url-safe words joined by hyphens.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonWord.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
