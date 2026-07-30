// Package tiktok reports whether the station's TikTok account is broadcasting
// right now, so the floating card's live action can reflect reality instead of
// always pointing at a room that is usually empty.
//
// The source is TikTok's own live-room endpoint, api-live/user/room. It needs no
// cookies, token or API key, and is the only viable option: the public
// /@handle/live HTML page is served behind a WAF challenge to anything that
// isn't a real browser, so scraping it returns a "Please wait..." shell rather
// than markup.
package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// liveStatus is the value liveRoom.status takes while a broadcast is running.
// Anything else - 4 is what an account that has finished a stream reports -
// means offline. Only this exact value may be trusted; see the Title caveat on
// Status.
const liveStatus = 2

// endpoint is TikTok's live-room lookup. aid=1988 is the web client's app id and
// sourceType=54 is the profile-page source; both are required, and the response
// is an error envelope without them.
const endpoint = "https://www.tiktok.com/api-live/user/room/"

// Status is one live-state snapshot for a single handle.
type Status struct {
	Live bool
	// Title is the live room's title, and is only ever populated when Live.
	// TikTok keeps serving the *previous* room's title, cover and viewer count
	// long after a stream ends (verified across several large accounts, all
	// reporting a populated title alongside status 4), so none of that metadata
	// means anything on its own.
	Title string
	// Known is false when the handle has never been probed or the last probe
	// failed. Callers treat unknown as live rather than offline - see the
	// fail-open note on Service.
	Known bool
}

// Service caches the live state of one TikTok handle.
//
// Unlike radio.Service this is deliberately stale-while-revalidate rather than a
// blocking read-through cache: the floating card sits in the shared layout, so
// every public page render asks for this value, and a blocking refresh would
// make one visitor per TTL wait out the HTTP timeout on a third-party host
// before their page rendered. Page rendering never blocks on TikTok.
//
// The other half of that policy lives at the call sites: a failed or cold probe
// is Known:false, which they render as live (fail open). A blocked production IP
// or a changed status code therefore degrades to the old always-on button rather
// than silently hiding it forever.
type Service struct {
	ttl    time.Duration
	client *http.Client

	mu         sync.Mutex
	handle     string // cache key; a different handle invalidates what's cached
	cached     Status
	fetched    time.Time
	hasValue   bool
	refreshing bool // one in-flight refresh at a time, however many readers there are
}

// NewService builds a live-state service. The handle is supplied per call rather
// than at construction because it is admin-managed (see /admin/media) and can
// change without a restart.
func NewService() *Service {
	return &Service{
		ttl:    60 * time.Second,
		client: &http.Client{Timeout: 6 * time.Second},
	}
}

// Status returns what is currently known about handle (the bare uniqueId, no
// leading "@"), kicking off a background refresh when that is stale. It never
// blocks on the network: a cold cache returns the zero Status, which is
// Known:false.
func (s *Service) Status(ctx context.Context, handle string) Status {
	if handle == "" {
		return Status{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle != handle {
		s.handle, s.cached, s.hasValue = handle, Status{}, false
	}
	if !s.hasValue || time.Since(s.fetched) >= s.ttl {
		if !s.refreshing {
			s.refreshing = true
			// Deliberately not the request context: it is cancelled the moment
			// this page finishes rendering, which is long before the refresh
			// this reader started could land.
			go s.refresh(handle)
		}
	}
	return s.cached
}

// Current returns the cached status without touching the network or needing a
// handle, for the polling endpoint. It is safe unprimed: any page carrying the
// card renders through Status first, so by the time a browser polls, the handle
// is known.
func (s *Service) Current() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cached
}

// refresh probes handle and stores the result, including failures - a caching
// probe error means an outage costs one request per TTL rather than one per page
// render.
func (s *Service) refresh(handle string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := s.fetch(ctx, handle)
	if err != nil {
		slog.Warn("tiktok live probe failed", "err", err, "handle", handle)
		st = Status{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshing = false
	if s.handle != handle {
		return // handle changed under us; this result is for the old account
	}
	s.cached, s.fetched, s.hasValue = st, time.Now(), true
}

// liveRoomResponse is the small part of the endpoint's payload that matters. The
// real body is ~7KB, most of it an embedded stream-manifest blob that is present
// whether or not the account is live. The read limit below is generous rather
// than snug for that reason: a truncated body fails to decode, and a decode
// failure reads as "unknown", which fails open.
type liveRoomResponse struct {
	Data struct {
		LiveRoom struct {
			Status int    `json:"status"`
			Title  string `json:"title"`
		} `json:"liveRoom"`
	} `json:"data"`
}

func (s *Service) fetch(ctx context.Context, handle string) (Status, error) {
	u := endpoint + "?aid=1988&sourceType=54&uniqueId=" + url.QueryEscape(handle)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Status{}, err
	}
	req.Header.Set("User-Agent", "ClassyFM-Web/1.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, endpoint)
	}

	var body liveRoomResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return Status{}, err
	}

	// An unknown handle answers 200 with data:null, which decodes to a zero
	// room - offline, which is the right answer for an account that isn't there.
	room := body.Data.LiveRoom
	if room.Status != liveStatus {
		return Status{Known: true}, nil
	}
	return Status{Live: true, Title: room.Title, Known: true}, nil
}
