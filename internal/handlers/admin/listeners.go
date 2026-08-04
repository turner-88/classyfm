package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/classyfm/classyfm/internal/schedule"
)

// listenersJSON is the /admin/api/listeners response shape. This endpoint exists
// only because the listener count is deliberately kept off the public
// /api/nowplaying payload (see radio.NowPlaying.Listeners) - everything else the
// admin dashboard polls, it polls from the public API.
type listenersJSON struct {
	Listeners int  `json:"listeners"`
	Live      bool `json:"live"`
	PeakToday int  `json:"peak_today"`
}

// ListenersJSON serves the current audience and today's peak so far, polled by
// admin-dashboard.js. The live number comes from the shared now-playing cache
// rather than a fresh fetch: an admin watching the page must not put the Shoutcast
// server under more load than a visitor does.
func (h *Handler) ListenersJSON(w http.ResponseWriter, r *http.Request) {
	np := h.radio.Current(r.Context())
	resp := listenersJSON{Listeners: np.Listeners, Live: np.Live}

	if h.q != nil {
		now := time.Now().In(schedule.Loc)
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, schedule.Loc)
		// Any failure leaves the peak at zero rather than failing the poll: no row
		// yet is the normal state just after midnight, and the live count is the
		// part the admin is actually watching.
		if row, err := h.q.GetListenerDay(r.Context(), day); err == nil {
			resp.PeakToday = int(row.PeakListeners)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}
