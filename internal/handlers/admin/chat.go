package admin

import "net/http"

// chatModerationData is the view-model for the Connect chat moderation page.
type chatModerationData struct{ Base baseData }

// ChatModeration renders the Connect chat moderation page. It is a thin shell: the
// live message list and the delete action run entirely client-side against the shared
// Firebase Realtime Database (see web/static/js/admin-chat.js), the same way the public
// Connect widget reads and writes. There is no server-side chat store, so this handler
// takes no DB dependency and there is no POST/delete route here (and, as a consequence,
// client-side deletes cannot be recorded in audit_logs).
func (h *Handler) ChatModeration(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, http.StatusOK, "admin/chat", chatModerationData{
		Base: h.base(r, "Chat", "chat"),
	})
}
