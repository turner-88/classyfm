package public

import "net/http"

// Connect renders the dedicated /connect chatroom page. The feed and composer are entirely
// client-driven: connect.js (with the vendored Firebase SDK, see firebase-config.js) hydrates
// the feed from the shared Firebase Realtime Database and drives sign-in/out via Firebase
// Authentication. The server only renders the shell - it holds no chat state and no chat
// identity. The site-wide floating widget uses the same client path, so message markup has a
// single source.
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, http.StatusOK, "public/connect", struct {
		Base baseData
	}{h.base(r, "Connect", "connect", "Join the "+h.station+" public chatroom.")})
}
